package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/cache"
	"github.com/kar1hsu/backplane/internal/repository"
	"gorm.io/gorm"
)

type RoleService struct {
	roleRepo *repository.RoleRepo
}

func NewRoleService() *RoleService {
	return &RoleService{roleRepo: repository.NewRoleRepo()}
}

type CreateRoleRequest struct {
	Name   string `json:"name" binding:"required"`
	Code   string `json:"code" binding:"required"`
	Sort   int    `json:"sort"`
	Status *int8  `json:"status"`
	Remark string `json:"remark"`
}

type UpdateRoleRequest struct {
	Name   string `json:"name"`
	Sort   int    `json:"sort"`
	Status *int8  `json:"status"`
	Remark string `json:"remark"`
}

type SetRoleMenusRequest struct {
	MenuIDs []uint `json:"menu_ids" binding:"required"`
}

type SetRoleAPIsRequest struct {
	APIs []RoleAPIItem `json:"apis" binding:"required"`
}

type RoleAPIItem struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

func (s *RoleService) Create(ctx context.Context, req *CreateRoleRequest) error {
	if req.Code == model.SuperAdminRoleCode {
		return errors.New("该角色编码为系统保留，不可使用")
	}
	_, err := s.roleRepo.GetByCode(ctx, req.Code)
	if err == nil {
		return errors.New("角色编码已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	status := int8(1)
	if req.Status != nil {
		status = *req.Status
	}

	role := &model.SysRole{
		Name:   req.Name,
		Code:   req.Code,
		Sort:   req.Sort,
		Status: status,
		Remark: req.Remark,
	}
	if err := s.roleRepo.Create(ctx, role); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.New("角色编码已存在")
		}
		return err
	}
	return nil
}

func (s *RoleService) GetByID(ctx context.Context, id uint) (*model.SysRole, error) {
	return s.roleRepo.GetByID(ctx, id)
}

func (s *RoleService) Update(ctx context.Context, id uint, req *UpdateRoleRequest) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "角色不存在")
	}

	if req.Name != "" {
		role.Name = req.Name
	}
	role.Sort = req.Sort
	if req.Status != nil {
		role.Status = *req.Status
	}
	if req.Remark != "" {
		role.Remark = req.Remark
	}
	return s.roleRepo.Update(ctx, role)
}

func (s *RoleService) Delete(ctx context.Context, id uint) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "角色不存在")
	}
	if role.Code == model.SuperAdminRoleCode {
		return errors.New("系统内置超级管理员角色不可删除")
	}

	if err := s.roleRepo.Delete(ctx, id); err != nil {
		return err
	}
	if _, err := app.Enforcer.RemoveFilteredPolicy(0, role.Code); err != nil {
		app.Enforcer.LoadPolicy() // 回滚内存策略到 DB 状态
		return fmt.Errorf("清除角色权限策略失败: %w", err)
	}
	cache.ClearAllPermissionCache()
	return nil
}

func (s *RoleService) List(ctx context.Context, page, pageSize int) ([]model.SysRole, int64, error) {
	return s.roleRepo.PageList(ctx, page, pageSize, &repository.QueryOptions{
		Order: []string{"sort ASC", "id ASC"},
	})
}

func (s *RoleService) ListAll(ctx context.Context) ([]model.SysRole, error) {
	return s.roleRepo.ListAll(ctx)
}

// SetMenus assigns menus to a role and rebuilds the role's Casbin policies from
// the menus' associated APIs. Casbin cannot share the DB transaction, so its
// operations are error-checked and, on failure, the in-memory policy is reloaded
// from DB to stay consistent.
func (s *RoleService) SetMenus(ctx context.Context, roleID uint, menuIDs []uint) error {
	if err := s.roleRepo.SetMenus(ctx, roleID, menuIDs); err != nil {
		return err
	}
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}

	if _, err := app.Enforcer.RemoveFilteredPolicy(0, role.Code); err != nil {
		app.Enforcer.LoadPolicy()
		return fmt.Errorf("清除旧权限策略失败: %w", err)
	}
	if len(menuIDs) > 0 {
		menus, err := repository.NewMenuRepo().GetByIDs(ctx, menuIDs)
		if err != nil {
			app.Enforcer.LoadPolicy()
			return err
		}
		var policies [][]string
		for _, m := range menus {
			for _, api := range m.APIs {
				policies = append(policies, []string{role.Code, api.Path, api.Method})
			}
		}
		if len(policies) > 0 {
			if _, err := app.Enforcer.AddPolicies(policies); err != nil {
				app.Enforcer.LoadPolicy()
				return fmt.Errorf("写入权限策略失败: %w", err)
			}
		}
	}
	if err := app.Enforcer.SavePolicy(); err != nil {
		app.Enforcer.LoadPolicy()
		return fmt.Errorf("保存权限策略失败: %w", err)
	}
	cache.ClearAllPermissionCache()
	return nil
}

func (s *RoleService) SetAPIs(ctx context.Context, roleID uint, apis []RoleAPIItem) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return notFoundOr(err, "角色不存在")
	}

	if _, err := app.Enforcer.RemoveFilteredPolicy(0, role.Code); err != nil {
		app.Enforcer.LoadPolicy()
		return fmt.Errorf("清除旧权限策略失败: %w", err)
	}
	var policies [][]string
	for _, api := range apis {
		policies = append(policies, []string{role.Code, api.Path, api.Method})
	}
	if len(policies) > 0 {
		if _, err := app.Enforcer.AddPolicies(policies); err != nil {
			app.Enforcer.LoadPolicy()
			return fmt.Errorf("写入权限策略失败: %w", err)
		}
	}
	if err := app.Enforcer.SavePolicy(); err != nil {
		app.Enforcer.LoadPolicy()
		return fmt.Errorf("保存权限策略失败: %w", err)
	}
	cache.ClearAllPermissionCache()
	return nil
}

func (s *RoleService) GetAPIs(ctx context.Context, roleID uint) ([]RoleAPIItem, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, notFoundOr(err, "角色不存在")
	}

	policies := app.Enforcer.GetFilteredPolicy(0, role.Code)
	items := make([]RoleAPIItem, 0, len(policies))
	for _, p := range policies {
		if len(p) >= 3 {
			items = append(items, RoleAPIItem{Path: p[1], Method: p[2]})
		}
	}
	return items, nil
}
