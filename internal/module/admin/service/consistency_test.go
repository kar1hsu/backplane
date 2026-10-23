package service

import (
	"context"
	"testing"

	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/repository"
)

// Every change to what a menu grants must reach the Casbin rules of the roles
// holding it, without anyone re-saving the role.
func TestMenuChangesResyncRolePolicies(t *testing.T) {
	setupRolePolicyTestEnvironment(t)
	ctx := context.Background()
	role := model.SysRole{Name: "Editor", Code: "editor", Status: 1}
	readAPI := model.SysAPI{Path: "/admin/users", Method: "GET"}
	deleteAPI := model.SysAPI{Path: "/admin/users/:id", Method: "DELETE"}
	mustCreate(t, &role)
	mustCreate(t, &readAPI)
	mustCreate(t, &deleteAPI)
	menu := model.SysMenu{Name: "用户", Type: 2, Status: 1, Visible: 1, APIs: []model.SysAPI{readAPI}}
	mustCreate(t, &menu)
	if err := NewRoleService().SetMenus(ctx, superAdmin, role.ID, []uint{menu.ID}); err != nil {
		t.Fatalf("SetMenus() error = %v", err)
	}
	menus := NewMenuService()

	rebind := []uint{deleteAPI.ID}
	if err := menus.Update(ctx, superAdmin, menu.ID, &UpdateMenuRequest{APIIDs: &rebind}); err != nil {
		t.Fatalf("Update(api_ids) error = %v", err)
	}
	assertEnforced(t, app.Enforcer, "editor", "/admin/users", "GET", false)
	assertEnforced(t, app.Enforcer, "editor", "/admin/users/7", "DELETE", true)

	if err := menus.Update(ctx, superAdmin, menu.ID, &UpdateMenuRequest{Status: ptr(int8(0))}); err != nil {
		t.Fatalf("Update(disable) error = %v", err)
	}
	assertEnforced(t, app.Enforcer, "editor", "/admin/users/7", "DELETE", false)

	if err := menus.Update(ctx, superAdmin, menu.ID, &UpdateMenuRequest{Status: ptr(int8(1))}); err != nil {
		t.Fatalf("Update(enable) error = %v", err)
	}
	assertEnforced(t, app.Enforcer, "editor", "/admin/users/7", "DELETE", true)

	if err := menus.Delete(ctx, menu.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	assertEnforced(t, app.Enforcer, "editor", "/admin/users/7", "DELETE", false)
	if n := countRows(t, "sys_role_menu", "sys_menu_id = ?", menu.ID); n != 0 {
		t.Fatalf("sys_role_menu rows for the deleted menu = %d, want 0", n)
	}
}

func TestAPIChangesResyncRolePolicies(t *testing.T) {
	setupRolePolicyTestEnvironment(t)
	ctx := context.Background()
	role := model.SysRole{Name: "Editor", Code: "editor", Status: 1}
	api := model.SysAPI{Path: "/admin/reports", Method: "GET"}
	mustCreate(t, &role)
	mustCreate(t, &api)
	menu := model.SysMenu{Name: "报表", Type: 1, Status: 1, Visible: 1, APIs: []model.SysAPI{api}}
	mustCreate(t, &menu)
	if err := NewRoleService().SetMenus(ctx, superAdmin, role.ID, []uint{menu.ID}); err != nil {
		t.Fatalf("SetMenus() error = %v", err)
	}
	apis := NewAPIService()

	if err := apis.Update(ctx, superAdmin, api.ID, &UpdateAPIRequest{Path: "/admin/stats", Method: "post"}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	assertEnforced(t, app.Enforcer, "editor", "/admin/reports", "GET", false)
	assertEnforced(t, app.Enforcer, "editor", "/admin/stats", "POST", true)

	if err := apis.Delete(ctx, superAdmin, api.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	assertEnforced(t, app.Enforcer, "editor", "/admin/stats", "POST", false)
	if n := countRows(t, "sys_menu_api", "sys_api_id = ?", api.ID); n != 0 {
		t.Fatalf("sys_menu_api rows for the deleted API = %d, want 0", n)
	}
}

// Association.Replace upserts the associated rows, so unknown IDs used to be
// inserted as blank roles, menus or APIs.
func TestUnknownIDsAreRejected(t *testing.T) {
	setupRolePolicyTestEnvironment(t)
	ctx := context.Background()
	role := model.SysRole{Name: "Editor", Code: "editor", Status: 1}
	mustCreate(t, &role)

	if err := NewRoleService().SetMenus(ctx, superAdmin, role.ID, []uint{9000}); err == nil {
		t.Fatal("SetMenus() accepted an unknown menu ID")
	}
	if err := NewUserService().Create(ctx, superAdmin, &CreateUserRequest{Username: "bob", Password: "secret-pass", RoleIDs: []uint{8000}}); err == nil {
		t.Fatal("Create(user) accepted an unknown role ID")
	}
	if err := NewMenuService().Create(ctx, superAdmin, &CreateMenuRequest{Name: "M", APIIDs: []uint{7000}}); err == nil {
		t.Fatal("Create(menu) accepted an unknown API ID")
	}

	for table, want := range map[string]int64{"sys_menu": 0, "sys_role": 1, "sys_api": 0, "sys_user": 0} {
		if n := countRows(t, table, "1 = 1"); n != want {
			t.Fatalf("%s rows = %d, want %d", table, n, want)
		}
	}
}

func TestMenuParentValidation(t *testing.T) {
	setupRolePolicyTestEnvironment(t)
	ctx := context.Background()
	dir := model.SysMenu{Name: "目录", Type: 0, Status: 1, Visible: 1}
	mustCreate(t, &dir)
	page := model.SysMenu{Name: "页面", ParentID: dir.ID, Type: 1, Status: 1, Visible: 1}
	mustCreate(t, &page)
	button := model.SysMenu{Name: "按钮", ParentID: page.ID, Type: 2, Status: 1, Visible: 1}
	mustCreate(t, &button)
	svc := NewMenuService()

	tests := []struct {
		name     string
		id       uint
		parentID uint
		wantErr  bool
	}{
		{name: "self", id: dir.ID, parentID: dir.ID, wantErr: true},
		{name: "descendant", id: dir.ID, parentID: page.ID, wantErr: true},
		{name: "button", id: page.ID, parentID: button.ID, wantErr: true},
		{name: "missing", id: page.ID, parentID: 9999, wantErr: true},
		{name: "root", id: page.ID, parentID: 0},
		{name: "directory", id: page.ID, parentID: dir.ID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Update(ctx, superAdmin, tt.id, &UpdateMenuRequest{ParentID: ptr(tt.parentID)})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Update(parent %d) error = %v, wantErr %v", tt.parentID, err, tt.wantErr)
			}
		})
	}
}

func TestMenuUpdateOnlyTouchesSentFields(t *testing.T) {
	setupRolePolicyTestEnvironment(t)
	ctx := context.Background()
	menu := model.SysMenu{Name: "用户", Type: 1, Sort: 5, Icon: "User", Status: 1, Visible: 1}
	mustCreate(t, &menu)
	svc := NewMenuService()

	if err := svc.Update(ctx, superAdmin, menu.ID, &UpdateMenuRequest{Name: ptr("用户管理")}); err != nil {
		t.Fatalf("Update(name) error = %v", err)
	}
	got := loadMenu(t, menu.ID)
	if got.Name != "用户管理" || got.Type != 1 || got.Sort != 5 || got.Status != 1 || got.Visible != 1 || got.Icon != "User" {
		t.Fatalf("menu after a name-only update = %+v", got)
	}

	if err := svc.Update(ctx, superAdmin, menu.ID, &UpdateMenuRequest{Icon: ptr("")}); err != nil {
		t.Fatalf("Update(clear icon) error = %v", err)
	}
	if got := loadMenu(t, menu.ID); got.Icon != "" {
		t.Fatalf("icon = %q, want cleared", got.Icon)
	}
}

// Status and visible have a column default of 1, which GORM's Create used to
// apply over an explicit 0.
func TestCreateKeepsDisabledStatus(t *testing.T) {
	setupRolePolicyTestEnvironment(t)
	ctx := context.Background()
	disabled := ptr(int8(0))

	if err := NewRoleService().Create(ctx, &CreateRoleRequest{Name: "R", Code: "r", Status: disabled}); err != nil {
		t.Fatalf("Create(role) error = %v", err)
	}
	if err := NewUserService().Create(ctx, superAdmin, &CreateUserRequest{Username: "u", Password: "secret-pass", Status: disabled}); err != nil {
		t.Fatalf("Create(user) error = %v", err)
	}
	if err := NewMenuService().Create(ctx, superAdmin, &CreateMenuRequest{Name: "M", Type: 1, Visible: disabled, Status: disabled}); err != nil {
		t.Fatalf("Create(menu) error = %v", err)
	}

	var role model.SysRole
	var user model.SysUser
	var menu model.SysMenu
	app.DB.Where("code = ?", "r").First(&role)
	app.DB.Where("username = ?", "u").First(&user)
	app.DB.Where("name = ?", "M").First(&menu)
	if role.Status != 0 || user.Status != 0 || menu.Status != 0 || menu.Visible != 0 {
		t.Fatalf("statuses role=%d user=%d menu=%d/%d, want all 0", role.Status, user.Status, menu.Status, menu.Visible)
	}
}

func TestRoleDeleteRemovesUserAssignments(t *testing.T) {
	setupRolePolicyTestEnvironment(t)
	ctx := context.Background()
	role := model.SysRole{Name: "Editor", Code: "editor", Status: 1}
	mustCreate(t, &role)
	mustCreate(t, &model.SysUser{Username: "alice", Password: "x", Status: 1, Roles: []model.SysRole{role}})

	if err := NewRoleService().Delete(ctx, superAdmin, role.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if n := countRows(t, "sys_user_role", "sys_role_id = ?", role.ID); n != 0 {
		t.Fatalf("sys_user_role rows = %d, want 0", n)
	}
}

func TestUserUpdateWritesOnlyChangedColumns(t *testing.T) {
	f := setupPrivilegeFixture(t)
	ctx := context.Background()
	if err := app.DB.Model(&model.SysUser{}).Where("id = ?", f.bob.ID).Updates(map[string]interface{}{"nickname": "Bob", "email": "bob@example.com"}).Error; err != nil {
		t.Fatalf("seed bob: %v", err)
	}

	if err := NewUserService().Update(ctx, f.rootOp(), f.bob.ID, &UpdateUserRequest{Email: ptr("")}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	var bob model.SysUser
	app.DB.First(&bob, f.bob.ID)
	if bob.Email != "" || bob.Nickname != "Bob" || bob.Password != "x" || bob.TokenVersion != 0 {
		t.Fatalf("bob after clearing email = %+v", bob)
	}
}

func TestConfigValueValidation(t *testing.T) {
	setupRolePolicyTestEnvironment(t)
	ctx := context.Background()
	retain := model.SysConfig{Key: "log.operation_retain_days", Name: "保留天数", Type: "int", Value: "30", Editable: true, Builtin: true}
	locked := model.SysConfig{Key: "site.locked", Name: "锁定", Type: "string", Value: "a", Editable: true}
	mustCreate(t, &retain)
	mustCreate(t, &locked)
	if err := app.DB.Model(&locked).Update("editable", false).Error; err != nil {
		t.Fatalf("lock config: %v", err)
	}
	svc := NewConfigService()

	tests := []struct {
		name    string
		run     func() error
		wantErr bool
	}{
		{name: "int config with text", wantErr: true, run: func() error {
			return svc.BatchUpdate(ctx, []ConfigItem{{Key: retain.Key, Value: "30天"}})
		}},
		{name: "unknown key", wantErr: true, run: func() error {
			return svc.BatchUpdate(ctx, []ConfigItem{{Key: "no.such.key", Value: "1"}})
		}},
		{name: "non-editable config", wantErr: true, run: func() error {
			return svc.BatchUpdate(ctx, []ConfigItem{{Key: locked.Key, Value: "b"}})
		}},
		{name: "builtin type change", wantErr: true, run: func() error {
			return svc.Update(ctx, retain.ID, &UpdateConfigRequest{Name: "保留天数", Type: "string", Value: "30"})
		}},
		{name: "re-saving a non-editable value", run: func() error {
			return svc.BatchUpdate(ctx, []ConfigItem{{Key: locked.Key, Value: "a"}})
		}},
		{name: "valid int without type keeps type", run: func() error {
			return svc.Update(ctx, retain.ID, &UpdateConfigRequest{Name: "保留天数", Value: "7"})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	var got model.SysConfig
	app.DB.First(&got, retain.ID)
	if got.Type != "int" || got.Value != "7" {
		t.Fatalf("retain config = type %q value %q, want int 7", got.Type, got.Value)
	}
}

func loadMenu(t *testing.T, id uint) *model.SysMenu {
	t.Helper()
	menu, err := repository.NewMenuRepo().GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("load menu: %v", err)
	}
	return menu
}

func countRows(t *testing.T, table, where string, args ...interface{}) int64 {
	t.Helper()
	var n int64
	if err := app.DB.Table(table).Where(where, args...).Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}
