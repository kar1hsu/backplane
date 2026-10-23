package service

import (
	"context"

	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/cache"
	"github.com/kar1hsu/backplane/internal/repository"
)

// rebuildRolePolicies recomputes each role's Casbin rules from the APIs of its
// enabled menus. Menus are the source of truth, so rules granted directly with
// SetAPIs are replaced as well. Call it after any change that affects what a
// role's menus grant: menu assignment, a menu's APIs or status, or an API's
// path or method.
//
// The enforcer auto-saves each change to the DB. It cannot join the caller's
// DB transaction, so call this after committing; on error the in-memory policy
// is reloaded from the DB to stay consistent.
func rebuildRolePolicies(ctx context.Context, roles []model.SysRole) error {
	roleRepo := repository.NewRoleRepo()
	menuRepo := repository.NewMenuRepo()
	for _, role := range roles {
		menus, err := roleRepo.GetMenusByRoleID(ctx, role.ID)
		if err != nil {
			return err
		}
		enabledIDs := make([]uint, 0, len(menus))
		for _, m := range menus {
			if m.Status == 1 {
				enabledIDs = append(enabledIDs, m.ID)
			}
		}
		var policies rolePolicies
		if len(enabledIDs) > 0 {
			withAPIs, err := menuRepo.GetByIDs(ctx, enabledIDs)
			if err != nil {
				return err
			}
			for _, m := range withAPIs {
				for _, api := range m.APIs {
					policies.add(role.Code, api.Path, api.Method)
				}
			}
		}

		if _, err := app.Enforcer.RemoveFilteredPolicy(0, role.Code); err != nil {
			return reloadPolicyError("清除旧权限策略失败", err)
		}
		if len(policies.rules) > 0 {
			if _, err := app.Enforcer.AddPolicies(policies.rules); err != nil {
				return reloadPolicyError("写入权限策略失败", err)
			}
		}
	}
	cache.ClearAllPermissionCache()
	return nil
}

// rebuildPoliciesForMenus re-syncs every role holding one of menuIDs.
func rebuildPoliciesForMenus(ctx context.Context, menuIDs []uint) error {
	roles, err := repository.NewRoleRepo().ListByMenuIDs(ctx, menuIDs)
	if err != nil {
		return err
	}
	return rebuildRolePolicies(ctx, roles)
}
