package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
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

var superAdmin = Operator{UserID: 1, RoleCodes: []string{model.SuperAdminRoleCode}}

func TestSetMenusSynchronizesCasbinPolicies(t *testing.T) {
	setupRolePolicyTestEnvironment(t)
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
	if err := service.SetMenus(ctx, superAdmin, role.ID, []uint{readMenu.ID}); err != nil {
		t.Fatalf("SetMenus(read) error = %v", err)
	}
	assertEnforced(t, app.Enforcer, "editor", "/admin/users/42", "GET", true)
	assertEnforced(t, app.Enforcer, "editor", "/admin/configs", "PUT", false)

	if err := service.SetMenus(ctx, superAdmin, role.ID, []uint{writeMenu.ID}); err != nil {
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

	// Policies must be persisted by auto-save alone (no SavePolicy): a fresh
	// enforcer loading from the same DB sees exactly the latest rules.
	reloaded := newGormEnforcer(t, app.DB)
	assertEnforced(t, reloaded, "editor", "/admin/configs", "PUT", true)
	assertEnforced(t, reloaded, "editor", "/admin/users/42", "GET", false)
}

// Seed menus share APIs (用户详情 and 编辑用户 both grant GET /admin/users/:id).
// Assigning both must not trip casbin_rule's unique index and wipe the role.
func TestSetMenusWithSharedAPIs(t *testing.T) {
	setupRolePolicyTestEnvironment(t)
	ctx := context.Background()

	role := model.SysRole{Name: "Editor", Code: "editor", Status: 1}
	if err := app.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	detailAPI := model.SysAPI{Path: "/admin/users/:id", Method: "GET", Group: "user"}
	editAPI := model.SysAPI{Path: "/admin/users/:id", Method: "PUT", Group: "user"}
	if err := app.DB.Create(&detailAPI).Error; err != nil {
		t.Fatalf("create detail API: %v", err)
	}
	if err := app.DB.Create(&editAPI).Error; err != nil {
		t.Fatalf("create edit API: %v", err)
	}
	detailMenu := model.SysMenu{Name: "用户详情", Type: 2, Status: 1, Visible: 1, APIs: []model.SysAPI{detailAPI}}
	editMenu := model.SysMenu{Name: "编辑用户", Type: 2, Status: 1, Visible: 1, APIs: []model.SysAPI{detailAPI, editAPI}}
	if err := app.DB.Create(&detailMenu).Error; err != nil {
		t.Fatalf("create detail menu: %v", err)
	}
	if err := app.DB.Create(&editMenu).Error; err != nil {
		t.Fatalf("create edit menu: %v", err)
	}

	service := NewRoleService()
	if err := service.SetMenus(ctx, superAdmin, role.ID, []uint{detailMenu.ID}); err != nil {
		t.Fatalf("SetMenus(detail) error = %v", err)
	}
	if err := service.SetMenus(ctx, superAdmin, role.ID, []uint{detailMenu.ID, editMenu.ID}); err != nil {
		t.Fatalf("SetMenus(detail, edit) error = %v", err)
	}

	assertEnforced(t, app.Enforcer, "editor", "/admin/users/42", "GET", true)
	assertEnforced(t, app.Enforcer, "editor", "/admin/users/42", "PUT", true)
	if got := countStoredRules(t, "editor"); got != 2 {
		t.Fatalf("stored rules = %d, want 2", got)
	}
}

func TestSetAPIsDeduplicatesRules(t *testing.T) {
	setupRolePolicyTestEnvironment(t)

	role := model.SysRole{Name: "Editor", Code: "editor", Status: 1}
	if err := app.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}

	err := NewRoleService().SetAPIs(context.Background(), superAdmin, role.ID, []RoleAPIItem{
		{Path: "/admin/users", Method: "GET"},
		{Path: "/admin/users", Method: "GET"},
		{Path: "/admin/users/:id", Method: "DELETE"},
	})
	if err != nil {
		t.Fatalf("SetAPIs() error = %v", err)
	}

	assertEnforced(t, app.Enforcer, "editor", "/admin/users", "GET", true)
	assertEnforced(t, app.Enforcer, "editor", "/admin/users/42", "DELETE", true)
	if got := countStoredRules(t, "editor"); got != 2 {
		t.Fatalf("stored rules = %d, want 2", got)
	}
}

func setupRolePolicyTestEnvironment(t *testing.T) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysUser{}, &model.SysRole{}, &model.SysMenu{}, &model.SysAPI{}, &model.SysConfig{}); err != nil {
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

	// Use the production gorm adapter: its casbin_rule unique index and
	// auto-save behavior are what the role service has to get right.
	previousEnforcer := app.Enforcer
	app.Enforcer = newGormEnforcer(t, db)
	t.Cleanup(func() { app.Enforcer = previousEnforcer })
}

func newGormEnforcer(t *testing.T, db *gorm.DB) *casbin.SyncedEnforcer {
	t.Helper()

	modelConfig, err := casbinmodel.NewModelFromString(rolePolicyTestModel)
	if err != nil {
		t.Fatalf("NewModelFromString() error = %v", err)
	}
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		t.Fatalf("NewAdapterByDB() error = %v", err)
	}
	enforcer, err := casbin.NewSyncedEnforcer(modelConfig, adapter)
	if err != nil {
		t.Fatalf("NewSyncedEnforcer() error = %v", err)
	}
	return enforcer
}

func countStoredRules(t *testing.T, roleCode string) int64 {
	t.Helper()

	var count int64
	if err := app.DB.Table("casbin_rule").Where("ptype = ? AND v0 = ?", "p", roleCode).Count(&count).Error; err != nil {
		t.Fatalf("count casbin rules: %v", err)
	}
	return count
}

func assertEnforced(t *testing.T, enforcer *casbin.SyncedEnforcer, role, path, method string, want bool) {
	t.Helper()

	got, err := enforcer.Enforce(role, path, method)
	if err != nil {
		t.Fatalf("Enforce(%q, %q, %q) error = %v", role, path, method, err)
	}
	if got != want {
		t.Fatalf("Enforce(%q, %q, %q) = %v, want %v", role, path, method, got, want)
	}
}

// Requests call Enforce while admins rewrite policies; run with -race to catch
// unsynchronized access to the enforcer's policy maps.
func TestEnforceConcurrentWithPolicyUpdates(t *testing.T) {
	setupRolePolicyTestEnvironment(t)

	role := model.SysRole{Name: "Editor", Code: "editor", Status: 1}
	if err := app.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	service := NewRoleService()

	done := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					_, _ = app.Enforcer.Enforce("editor", "/admin/users/42", "GET")
				}
			}
		}()
	}

	for i := 0; i < 20; i++ {
		err := service.SetAPIs(context.Background(), superAdmin, role.ID, []RoleAPIItem{
			{Path: "/admin/users/:id", Method: "GET"},
			{Path: fmt.Sprintf("/admin/items/%d", i), Method: "PUT"},
		})
		if err != nil {
			close(done)
			wg.Wait()
			t.Fatalf("SetAPIs() error = %v", err)
		}
	}
	close(done)
	wg.Wait()
}
