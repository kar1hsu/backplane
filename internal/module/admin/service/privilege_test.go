package service

import (
	"context"
	"testing"

	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/setting"
)

type privilegeFixture struct {
	adminRole, editorRole, auditorRole model.SysRole
	root, alice, bob                   model.SysUser
	menuA, menuB                       model.SysMenu
}

// root is a super admin, alice an "editor" (holds menuA), bob an "auditor".
func setupPrivilegeFixture(t *testing.T) *privilegeFixture {
	t.Helper()
	setupRolePolicyTestEnvironment(t)

	f := &privilegeFixture{
		adminRole:   model.SysRole{Name: "Admin", Code: model.SuperAdminRoleCode, Status: 1},
		editorRole:  model.SysRole{Name: "Editor", Code: "editor", Status: 1},
		auditorRole: model.SysRole{Name: "Auditor", Code: "auditor", Status: 1},
		menuA:       model.SysMenu{Name: "A", Type: 1, Status: 1, Visible: 1},
		menuB:       model.SysMenu{Name: "B", Type: 1, Status: 1, Visible: 1},
	}
	mustCreate(t, &f.adminRole)
	mustCreate(t, &f.editorRole)
	mustCreate(t, &f.auditorRole)
	mustCreate(t, &f.menuA)
	mustCreate(t, &f.menuB)
	if err := app.DB.Model(&f.editorRole).Association("Menus").Replace([]model.SysMenu{f.menuA}); err != nil {
		t.Fatalf("grant editor menu: %v", err)
	}

	f.root = model.SysUser{Username: "root", Password: "x", Status: 1, Roles: []model.SysRole{f.adminRole}}
	f.alice = model.SysUser{Username: "alice", Password: "x", Status: 1, Roles: []model.SysRole{f.editorRole}}
	f.bob = model.SysUser{Username: "bob", Password: "x", Status: 1, Roles: []model.SysRole{f.auditorRole}}
	mustCreate(t, &f.root)
	mustCreate(t, &f.alice)
	mustCreate(t, &f.bob)
	return f
}

func (f *privilegeFixture) rootOp() Operator {
	return Operator{UserID: f.root.ID, RoleCodes: []string{model.SuperAdminRoleCode}}
}

func (f *privilegeFixture) aliceOp() Operator {
	return Operator{UserID: f.alice.ID, RoleCodes: []string{"editor"}}
}

func TestUserUpdatePrivilegeGuards(t *testing.T) {
	f := setupPrivilegeFixture(t)
	svc := NewUserService()
	ctx := context.Background()
	disabled := int8(0)
	active := int8(1)

	tests := []struct {
		name          string
		op            Operator
		target        uint
		req           UpdateUserRequest
		wantForbidden bool
		wantErr       bool
	}{
		{name: "non-super grants super admin role", op: f.aliceOp(), target: f.bob.ID,
			req: UpdateUserRequest{RoleIDs: []uint{f.adminRole.ID}}, wantForbidden: true},
		{name: "non-super resets super admin password", op: f.aliceOp(), target: f.root.ID,
			req: UpdateUserRequest{Password: "new-password"}, wantForbidden: true},
		{name: "non-super disables super admin", op: f.aliceOp(), target: f.root.ID,
			req: UpdateUserRequest{Status: &disabled}, wantForbidden: true},
		{name: "non-super disables self", op: f.aliceOp(), target: f.alice.ID,
			req: UpdateUserRequest{Status: &disabled}, wantForbidden: true},
		{name: "non-super changes own roles", op: f.aliceOp(), target: f.alice.ID,
			req: UpdateUserRequest{RoleIDs: []uint{f.editorRole.ID, f.auditorRole.ID}}, wantForbidden: true},
		{name: "super changes own roles", op: f.rootOp(), target: f.root.ID,
			req: UpdateUserRequest{RoleIDs: []uint{f.editorRole.ID}}, wantForbidden: true},
		{name: "unknown role id", op: f.rootOp(), target: f.bob.ID,
			req: UpdateUserRequest{RoleIDs: []uint{9999}}, wantErr: true},
		{name: "self edit with unchanged status and roles", op: f.aliceOp(), target: f.alice.ID,
			req: UpdateUserRequest{Nickname: ptr("Alice"), Status: &active, RoleIDs: []uint{f.editorRole.ID}}},
		{name: "non-super assigns ordinary role", op: f.aliceOp(), target: f.bob.ID,
			req: UpdateUserRequest{RoleIDs: []uint{f.editorRole.ID}}},
		{name: "super grants super admin role", op: f.rootOp(), target: f.bob.ID,
			req: UpdateUserRequest{RoleIDs: []uint{f.adminRole.ID}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Update(ctx, tt.op, tt.target, &tt.req)
			switch {
			case tt.wantForbidden:
				if !IsForbidden(err) {
					t.Fatalf("Update() error = %v, want forbidden", err)
				}
			case tt.wantErr:
				if err == nil || IsForbidden(err) {
					t.Fatalf("Update() error = %v, want a validation error", err)
				}
			default:
				if err != nil {
					t.Fatalf("Update() error = %v", err)
				}
			}
		})
	}
}

// The edit form always sends status and role_ids; only real changes may
// revoke the user's sessions.
func TestUserUpdateRevokesSessionsOnlyOnRealChanges(t *testing.T) {
	f := setupPrivilegeFixture(t)
	svc := NewUserService()
	ctx := context.Background()
	active := int8(1)

	if err := svc.Update(ctx, f.rootOp(), f.bob.ID, &UpdateUserRequest{
		Nickname: ptr("Bob"), Status: &active, RoleIDs: []uint{f.auditorRole.ID},
	}); err != nil {
		t.Fatalf("Update(unchanged) error = %v", err)
	}
	if got := tokenVersion(t, f.bob.ID); got != 0 {
		t.Fatalf("token_version after a nickname edit = %d, want 0", got)
	}

	if err := svc.Update(ctx, f.rootOp(), f.bob.ID, &UpdateUserRequest{
		Status: &active, RoleIDs: []uint{f.editorRole.ID},
	}); err != nil {
		t.Fatalf("Update(roles) error = %v", err)
	}
	if got := tokenVersion(t, f.bob.ID); got != 1 {
		t.Fatalf("token_version after a role change = %d, want 1", got)
	}
}

func TestUserCreatePrivilegeAndPasswordPolicy(t *testing.T) {
	f := setupPrivilegeFixture(t)
	svc := NewUserService()
	ctx := context.Background()

	err := svc.Create(ctx, f.aliceOp(), &CreateUserRequest{
		Username: "mallory", Password: "secret-pass", RoleIDs: []uint{f.adminRole.ID},
	})
	if !IsForbidden(err) {
		t.Fatalf("Create(super admin role) error = %v, want forbidden", err)
	}

	mustCreate(t, &model.SysConfig{Key: setting.PasswordMinLengthKey, Value: "10", Type: "int"})
	if err := svc.Create(ctx, f.rootOp(), &CreateUserRequest{Username: "carol", Password: "short-pw"}); err == nil {
		t.Fatal("Create() accepted a password shorter than security.password_min_length")
	}
	if err := svc.Create(ctx, f.rootOp(), &CreateUserRequest{Username: "carol", Password: "long-enough-pw"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestRolePrivilegeGuards(t *testing.T) {
	f := setupPrivilegeFixture(t)
	svc := NewRoleService()
	ctx := context.Background()
	disabled := int8(0)

	tests := []struct {
		name          string
		run           func() error
		wantForbidden bool
	}{
		{name: "non-super edits own role menus", wantForbidden: true, run: func() error {
			return svc.SetMenus(ctx, f.aliceOp(), f.editorRole.ID, []uint{f.menuA.ID, f.menuB.ID})
		}},
		{name: "non-super grants a menu they lack", wantForbidden: true, run: func() error {
			return svc.SetMenus(ctx, f.aliceOp(), f.auditorRole.ID, []uint{f.menuB.ID})
		}},
		{name: "non-super edits super admin role", wantForbidden: true, run: func() error {
			return svc.Update(ctx, f.aliceOp(), f.adminRole.ID, &UpdateRoleRequest{Name: ptr("x")})
		}},
		{name: "super disables super admin role", wantForbidden: true, run: func() error {
			return svc.Update(ctx, f.rootOp(), f.adminRole.ID, &UpdateRoleRequest{Status: &disabled})
		}},
		{name: "non-super sets raw API rules", wantForbidden: true, run: func() error {
			return svc.SetAPIs(ctx, f.aliceOp(), f.auditorRole.ID, []RoleAPIItem{{Path: "/admin/users", Method: "GET"}})
		}},
		{name: "non-super grants a menu they hold", run: func() error {
			return svc.SetMenus(ctx, f.aliceOp(), f.auditorRole.ID, []uint{f.menuA.ID})
		}},
		{name: "super grants any menu", run: func() error {
			return svc.SetMenus(ctx, f.rootOp(), f.editorRole.ID, []uint{f.menuA.ID, f.menuB.ID})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if tt.wantForbidden && !IsForbidden(err) {
				t.Fatalf("error = %v, want forbidden", err)
			}
			if !tt.wantForbidden && err != nil {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestMenuAPIBindingIsSuperAdminOnly(t *testing.T) {
	f := setupPrivilegeFixture(t)
	svc := NewMenuService()
	ctx := context.Background()
	api := model.SysAPI{Path: "/admin/configs", Method: "PUT"}
	mustCreate(t, &api)

	unchanged := []uint{}
	if err := svc.Update(ctx, f.aliceOp(), f.menuA.ID, &UpdateMenuRequest{Name: ptr("A2"), APIIDs: &unchanged}); err != nil {
		t.Fatalf("Update(unchanged api_ids) error = %v", err)
	}
	changed := []uint{api.ID}
	if err := svc.Update(ctx, f.aliceOp(), f.menuA.ID, &UpdateMenuRequest{APIIDs: &changed}); !IsForbidden(err) {
		t.Fatalf("Update(api_ids) by non-super error = %v, want forbidden", err)
	}
	if err := svc.Create(ctx, f.aliceOp(), &CreateMenuRequest{Name: "C", APIIDs: []uint{api.ID}}); !IsForbidden(err) {
		t.Fatalf("Create(api_ids) by non-super error = %v, want forbidden", err)
	}
	if err := svc.Update(ctx, f.rootOp(), f.menuA.ID, &UpdateMenuRequest{Name: ptr("A3"), APIIDs: &changed}); err != nil {
		t.Fatalf("Update(api_ids) by super error = %v", err)
	}
}

func mustCreate(t *testing.T, value interface{}) {
	t.Helper()
	if err := app.DB.Create(value).Error; err != nil {
		t.Fatalf("create %T: %v", value, err)
	}
}

func tokenVersion(t *testing.T, userID uint) int {
	t.Helper()
	var user model.SysUser
	if err := app.DB.Select("token_version").First(&user, userID).Error; err != nil {
		t.Fatalf("load token_version: %v", err)
	}
	return user.TokenVersion
}

func ptr[T any](v T) *T { return &v }
