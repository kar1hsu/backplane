package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/glebarez/sqlite"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const rolePolicyTestModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && r.act == p.act || r.sub == "admin"
`

func TestSetMenusSynchronizesCasbinPolicies(t *testing.T) {
	policyFile := setupRolePolicyTestEnvironment(t)
	ctx := context.Background()

	role := model.SysRole{Name: "Editor", Code: "editor", Status: 1}
	if err := app.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	readAPI := model.SysAPI{Path: "/admin/users/:id", Method: "GET", Group: "user"}
	writeAPI := model.SysAPI{Path: "/admin/configs", Method: "PUT", Group: "config"}
	if err := app.DB.Create(&readAPI).Error; err != nil {
		t.Fatalf("create read API: %v", err)
	}
	if err := app.DB.Create(&writeAPI).Error; err != nil {
		t.Fatalf("create write API: %v", err)
	}
	readMenu := model.SysMenu{Name: "用户查看", Type: 2, Status: 1, Visible: 1}
	writeMenu := model.SysMenu{Name: "配置保存", Type: 2, Status: 1, Visible: 1}
	if err := app.DB.Create(&readMenu).Error; err != nil {
		t.Fatalf("create read menu: %v", err)
	}
	if err := app.DB.Create(&writeMenu).Error; err != nil {
		t.Fatalf("create write menu: %v", err)
	}
	if err := app.DB.Model(&readMenu).Association("APIs").Replace(&readAPI); err != nil {
		t.Fatalf("associate read API: %v", err)
	}
	if err := app.DB.Model(&writeMenu).Association("APIs").Replace(&writeAPI); err != nil {
		t.Fatalf("associate write API: %v", err)
	}

	service := NewRoleService()
	if err := service.SetMenus(ctx, role.ID, []uint{readMenu.ID}); err != nil {
		t.Fatalf("SetMenus(read) error = %v", err)
	}
	assertEnforced(t, app.Enforcer, "editor", "/admin/users/42", "GET", true)
	assertEnforced(t, app.Enforcer, "editor", "/admin/configs", "PUT", false)

	if err := service.SetMenus(ctx, role.ID, []uint{writeMenu.ID}); err != nil {
		t.Fatalf("SetMenus(write) error = %v", err)
	}
	assertEnforced(t, app.Enforcer, "editor", "/admin/users/42", "GET", false)
	assertEnforced(t, app.Enforcer, "editor", "/admin/configs", "PUT", true)

	var associatedMenus []model.SysMenu
	if err := app.DB.Model(&role).Association("Menus").Find(&associatedMenus); err != nil {
		t.Fatalf("load role menus: %v", err)
	}
	if len(associatedMenus) != 1 || associatedMenus[0].ID != writeMenu.ID {
		t.Fatalf("associated menus = %+v", associatedMenus)
	}

	reloaded := newFileEnforcer(t, policyFile)
	assertEnforced(t, reloaded, "editor", "/admin/configs", "PUT", true)
}

func setupRolePolicyTestEnvironment(t *testing.T) string {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysRole{}, &model.SysMenu{}, &model.SysAPI{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	previousDB := app.DB
	app.DB = db
	t.Cleanup(func() { app.DB = previousDB })

	server := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: server.Addr()})
	previousStore := cache.GetStore()
	cache.InitStore(cache.NewRedisStore(redisClient, "role-test:"))
	t.Cleanup(func() {
		cache.InitStore(previousStore)
		_ = redisClient.Close()
	})

	policyFile := filepath.Join(t.TempDir(), "policy.csv")
	if err := os.WriteFile(policyFile, nil, 0o600); err != nil {
		t.Fatalf("create policy file: %v", err)
	}
	previousEnforcer := app.Enforcer
	app.Enforcer = newFileEnforcer(t, policyFile)
	t.Cleanup(func() { app.Enforcer = previousEnforcer })
	return policyFile
}

func newFileEnforcer(t *testing.T, policyFile string) *casbin.Enforcer {
	t.Helper()

	modelConfig, err := casbinmodel.NewModelFromString(rolePolicyTestModel)
	if err != nil {
		t.Fatalf("NewModelFromString() error = %v", err)
	}
	enforcer, err := casbin.NewEnforcer(modelConfig, fileadapter.NewAdapter(policyFile))
	if err != nil {
		t.Fatalf("NewEnforcer() error = %v", err)
	}
	return enforcer
}

func assertEnforced(t *testing.T, enforcer *casbin.Enforcer, role, path, method string, want bool) {
	t.Helper()

	got, err := enforcer.Enforce(role, path, method)
	if err != nil {
		t.Fatalf("Enforce(%q, %q, %q) error = %v", role, path, method, err)
	}
	if got != want {
		t.Fatalf("Enforce(%q, %q, %q) = %v, want %v", role, path, method, got, want)
	}
}
