package service

import (
	"context"
	"errors"

	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/cache"
	"github.com/kar1hsu/backplane/internal/repository"
)

type MenuService struct {
	menuRepo *repository.MenuRepo
	roleRepo *repository.RoleRepo
}

func NewMenuService() *MenuService {
	return &MenuService{
		menuRepo: repository.NewMenuRepo(),
		roleRepo: repository.NewRoleRepo(),
	}
}

type CreateMenuRequest struct {
	ParentID   uint   `json:"parent_id"`
	Name       string `json:"name" binding:"required,max=64"`
	Path       string `json:"path"`
	Component  string `json:"component"`
	Icon       string `json:"icon"`
	Sort       int    `json:"sort"`
	Type       int8   `json:"type" binding:"oneof=0 1 2"`
	Permission string `json:"permission"`
	Visible    *int8  `json:"visible" binding:"omitempty,oneof=0 1"` // nil = visible
	Status     *int8  `json:"status" binding:"omitempty,oneof=0 1"`  // nil = enabled
	APIIDs     []uint `json:"api_ids"`
}

// UpdateMenuRequest: nil fields are left unchanged; a non-nil empty string
// clears the field (e.g. removing an icon or permission).
type UpdateMenuRequest struct {
	ParentID   *uint   `json:"parent_id"`
	Name       *string `json:"name" binding:"omitempty,min=1,max=64"`
	Path       *string `json:"path"`
	Component  *string `json:"component"`
	Icon       *string `json:"icon"`
	Sort       *int    `json:"sort"`
	Type       *int8   `json:"type" binding:"omitempty,oneof=0 1 2"`
	Permission *string `json:"permission"`
	Visible    *int8   `json:"visible" binding:"omitempty,oneof=0 1"`
	Status     *int8   `json:"status" binding:"omitempty,oneof=0 1"`
	APIIDs     *[]uint `json:"api_ids"`
}

// Binding APIs to a menu decides which endpoints every role holding the menu
// may call, so only a super admin may set or change a menu's api_ids.
const menuAPIsForbiddenMsg = "只有超级管理员可以配置菜单关联的 API"

func (s *MenuService) Create(ctx context.Context, op Operator, req *CreateMenuRequest) error {
	if len(req.APIIDs) > 0 {
		if err := requireSuperAdmin(op, menuAPIsForbiddenMsg); err != nil {
			return err
		}
	}
	if err := s.validateParent(ctx, 0, req.ParentID); err != nil {
		return err
	}
	apiIDs, err := s.checkAPIIDs(ctx, req.APIIDs)
	if err != nil {
		return err
	}

	visible, status := int8(1), int8(1)
	if req.Visible != nil {
		visible = *req.Visible
	}
	if req.Status != nil {
		status = *req.Status
	}
	menu := &model.SysMenu{
		ParentID:   req.ParentID,
		Name:       req.Name,
		Path:       req.Path,
		Component:  req.Component,
		Icon:       req.Icon,
		Sort:       req.Sort,
		Type:       req.Type,
		Permission: req.Permission,
		Visible:    visible,
		Status:     status,
	}
	return repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.menuRepo.Create(ctx, menu); err != nil {
			return err
		}
		if visible == 0 || status == 0 {
			// Create skips zero values on columns with a default (visible and
			// status default to 1), so write the requested values explicitly.
			menu.Visible, menu.Status = visible, status
			if err := s.menuRepo.Update(ctx, menu); err != nil {
				return err
			}
		}
		if len(apiIDs) > 0 {
			return s.menuRepo.SetAPIs(ctx, menu.ID, apiIDs)
		}
		return nil
	})
}

func (s *MenuService) GetByID(ctx context.Context, id uint) (*model.SysMenu, error) {
	return s.menuRepo.GetByID(ctx, id)
}

// Update edits a menu. When its APIs or status change, the Casbin rules of
// every role holding it are rebuilt so the change takes effect immediately.
func (s *MenuService) Update(ctx context.Context, op Operator, id uint, req *UpdateMenuRequest) error {
	menu, err := s.menuRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "菜单不存在")
	}

	var apiIDs []uint
	apisChanged := false
	if req.APIIDs != nil {
		currentIDs := make([]uint, 0, len(menu.APIs))
		for _, a := range menu.APIs {
			currentIDs = append(currentIDs, a.ID)
		}
		if apisChanged = !sameIDs(currentIDs, *req.APIIDs); apisChanged {
			if err := requireSuperAdmin(op, menuAPIsForbiddenMsg); err != nil {
				return err
			}
			if apiIDs, err = s.checkAPIIDs(ctx, *req.APIIDs); err != nil {
				return err
			}
		}
	}
	if req.ParentID != nil {
		if err := s.validateParent(ctx, id, *req.ParentID); err != nil {
			return err
		}
		menu.ParentID = *req.ParentID
	}

	setString(&menu.Name, req.Name)
	setString(&menu.Path, req.Path)
	setString(&menu.Component, req.Component)
	setString(&menu.Icon, req.Icon)
	setString(&menu.Permission, req.Permission)
	if req.Sort != nil {
		menu.Sort = *req.Sort
	}
	if req.Type != nil {
		menu.Type = *req.Type
	}
	if req.Visible != nil {
		menu.Visible = *req.Visible
	}
	statusChanged := req.Status != nil && *req.Status != menu.Status
	if req.Status != nil {
		menu.Status = *req.Status
	}

	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.menuRepo.Update(ctx, menu); err != nil {
			return err
		}
		if apisChanged {
			return s.menuRepo.SetAPIs(ctx, id, apiIDs)
		}
		return nil
	}); err != nil {
		return err
	}

	if apisChanged || statusChanged {
		return rebuildPoliciesForMenus(ctx, []uint{id})
	}
	// Name, path, permission or visibility may have changed.
	cache.ClearAllPermissionCache()
	return nil
}

// Delete removes a leaf menu, revokes it from every role and rebuilds those
// roles' Casbin rules.
func (s *MenuService) Delete(ctx context.Context, id uint) error {
	has, err := s.menuRepo.HasChildren(ctx, id)
	if err != nil {
		return err
	}
	if has {
		return errors.New("存在子菜单，无法删除")
	}
	roles, err := s.roleRepo.ListByMenuIDs(ctx, []uint{id})
	if err != nil {
		return err
	}
	if err := s.menuRepo.Delete(ctx, id); err != nil {
		return err
	}
	return rebuildRolePolicies(ctx, roles)
}

// validateParent: the parent must be the root (0) or an existing directory or
// menu, and — when moving menu id — neither id itself nor one of its
// descendants, which would form a cycle and drop the subtree from every tree.
// id is 0 for a new menu.
func (s *MenuService) validateParent(ctx context.Context, id, parentID uint) error {
	if parentID == 0 {
		return nil
	}
	if parentID == id {
		return errors.New("上级菜单不能是自身")
	}
	menus, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return err
	}
	byID := make(map[uint]model.SysMenu, len(menus))
	for _, m := range menus {
		byID[m.ID] = m
	}
	parent, ok := byID[parentID]
	if !ok {
		return errors.New("上级菜单不存在")
	}
	if parent.Type == 2 {
		return errors.New("按钮不能作为上级菜单")
	}
	if id == 0 {
		return nil
	}
	seen := make(map[uint]bool)
	for cur := parentID; cur != 0 && !seen[cur]; cur = byID[cur].ParentID {
		if cur == id {
			return errors.New("上级菜单不能是自身的子菜单")
		}
		seen[cur] = true
	}
	return nil
}

// checkAPIIDs dedupes ids and rejects unknown ones, which Association.Replace
// would otherwise insert as blank API records.
func (s *MenuService) checkAPIIDs(ctx context.Context, ids []uint) ([]uint, error) {
	ids = uniqueIDs(ids)
	n, err := repository.NewApiRepo().CountIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if n != int64(len(ids)) {
		return nil, errors.New("API 不存在")
	}
	return ids, nil
}

func setString(dst *string, src *string) {
	if src != nil {
		*dst = *src
	}
}

func (s *MenuService) GetTree(ctx context.Context) ([]*model.SysMenu, error) {
	menus, err := s.menuRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return repository.BuildMenuTree(menus, 0), nil
}

func (s *MenuService) GetUserMenuTree(ctx context.Context, userID uint) ([]*model.SysMenu, error) {
	userDAO := repository.NewUserRepo()
	user, err := userDAO.GetByID(ctx, userID)
	if err != nil {
		return nil, notFoundOr(err, "用户不存在")
	}

	menuIDSet := make(map[uint]bool)
	for _, role := range enabledRoles(user.Roles) {
		menus, err := s.roleRepo.GetMenusByRoleID(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		for _, m := range menus {
			menuIDSet[m.ID] = true
		}
	}

	if len(menuIDSet) == 0 {
		return []*model.SysMenu{}, nil
	}

	ids := make([]uint, 0, len(menuIDSet))
	for id := range menuIDSet {
		ids = append(ids, id)
	}

	menus, err := s.menuRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Filter: only directories (0) and menus (1), visible and enabled
	visible := make([]model.SysMenu, 0, len(menus))
	for _, m := range menus {
		if m.Type <= 1 && m.Visible == 1 && m.Status == 1 {
			visible = append(visible, m)
		}
	}

	return repository.BuildMenuTree(visible, 0), nil
}

// GetUserPermissions returns all permission identifiers (including buttons) for a user.
func (s *MenuService) GetUserPermissions(ctx context.Context, userID uint) ([]string, error) {
	userDAO := repository.NewUserRepo()
	user, err := userDAO.GetByID(ctx, userID)
	if err != nil {
		return nil, notFoundOr(err, "用户不存在")
	}

	roles := enabledRoles(user.Roles)

	// Super admin has all permissions
	if hasSuperAdminRole(roles) {
		return []string{"*"}, nil
	}

	menuIDSet := make(map[uint]bool)
	for _, role := range roles {
		menus, err := s.roleRepo.GetMenusByRoleID(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		for _, m := range menus {
			menuIDSet[m.ID] = true
		}
	}

	if len(menuIDSet) == 0 {
		return []string{}, nil
	}

	ids := make([]uint, 0, len(menuIDSet))
	for id := range menuIDSet {
		ids = append(ids, id)
	}

	menus, err := s.menuRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	perms := make([]string, 0)
	for _, m := range menus {
		if m.Permission != "" && m.Status == 1 {
			perms = append(perms, m.Permission)
		}
	}
	return perms, nil
}

// enabledRoles drops disabled roles, matching what AdminAuth enforces.
func enabledRoles(roles []model.SysRole) []model.SysRole {
	out := make([]model.SysRole, 0, len(roles))
	for _, r := range roles {
		if r.Status == 1 {
			out = append(out, r)
		}
	}
	return out
}
