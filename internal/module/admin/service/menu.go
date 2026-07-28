package service

import (
	"context"
	"errors"

	"github.com/kar1hsu/backplane/internal/model"
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
	Name       string `json:"name" binding:"required"`
	Path       string `json:"path"`
	Component  string `json:"component"`
	Icon       string `json:"icon"`
	Sort       int    `json:"sort"`
	Type       int8   `json:"type"`
	Permission string `json:"permission"`
	Visible    int8   `json:"visible"`
	Status     int8   `json:"status"`
	APIIDs     []uint `json:"api_ids"`
}

type UpdateMenuRequest struct {
	ParentID   *uint   `json:"parent_id"`
	Name       string  `json:"name"`
	Path       string  `json:"path"`
	Component  string  `json:"component"`
	Icon       string  `json:"icon"`
	Sort       int     `json:"sort"`
	Type       int8    `json:"type"`
	Permission string  `json:"permission"`
	Visible    int8    `json:"visible"`
	Status     int8    `json:"status"`
	APIIDs     *[]uint `json:"api_ids"`
}

func (s *MenuService) Create(ctx context.Context, req *CreateMenuRequest) error {
	menu := &model.SysMenu{
		ParentID:   req.ParentID,
		Name:       req.Name,
		Path:       req.Path,
		Component:  req.Component,
		Icon:       req.Icon,
		Sort:       req.Sort,
		Type:       req.Type,
		Permission: req.Permission,
		Visible:    req.Visible,
		Status:     req.Status,
	}
	return repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.menuRepo.Create(ctx, menu); err != nil {
			return err
		}
		if len(req.APIIDs) > 0 {
			return s.menuRepo.SetAPIs(ctx, menu.ID, req.APIIDs)
		}
		return nil
	})
}

func (s *MenuService) GetByID(ctx context.Context, id uint) (*model.SysMenu, error) {
	return s.menuRepo.GetByID(ctx, id)
}

func (s *MenuService) Update(ctx context.Context, id uint, req *UpdateMenuRequest) error {
	menu, err := s.menuRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "菜单不存在")
	}

	if req.ParentID != nil {
		menu.ParentID = *req.ParentID
	}
	if req.Name != "" {
		menu.Name = req.Name
	}
	if req.Path != "" {
		menu.Path = req.Path
	}
	if req.Component != "" {
		menu.Component = req.Component
	}
	if req.Icon != "" {
		menu.Icon = req.Icon
	}
	menu.Sort = req.Sort
	menu.Type = req.Type
	if req.Permission != "" {
		menu.Permission = req.Permission
	}
	menu.Visible = req.Visible
	menu.Status = req.Status

	return repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.menuRepo.Update(ctx, menu); err != nil {
			return err
		}
		if req.APIIDs != nil {
			return s.menuRepo.SetAPIs(ctx, id, *req.APIIDs)
		}
		return nil
	})
}

func (s *MenuService) Delete(ctx context.Context, id uint) error {
	has, err := s.menuRepo.HasChildren(ctx, id)
	if err != nil {
		return err
	}
	if has {
		return errors.New("存在子菜单，无法删除")
	}
	return s.menuRepo.Delete(ctx, id)
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
	for _, role := range user.Roles {
		menus, err := s.roleRepo.GetMenusByRoleID(ctx, role.ID)
		if err != nil {
			continue
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

	// Super admin has all permissions
	for _, role := range user.Roles {
		if role.Code == model.SuperAdminRoleCode {
			return []string{"*"}, nil
		}
	}

	menuIDSet := make(map[uint]bool)
	for _, role := range user.Roles {
		menus, err := s.roleRepo.GetMenusByRoleID(ctx, role.ID)
		if err != nil {
			continue
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
