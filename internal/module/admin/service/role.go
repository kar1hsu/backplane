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
	Code   string `json:"code" binding:"required,max=64"`
	Sort   int    `json:"sort"`
	Status *int8  `json:"status" binding:"omitempty,oneof=0 1"`
	Remark string `json:"remark"`
}

// UpdateRoleRequest: nil fields are left unchanged; a non-nil empty remark
// clears it.
type UpdateRoleRequest struct {
	Name   *string `json:"name" binding:"omitempty,min=1"`
	Sort   *int    `json:"sort"`
	Status *int8   `json:"status" binding:"omitempty,oneof=0 1"`
	Remark *string `json:"remark"`
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
	return repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.roleRepo.Create(ctx, role); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return errors.New("角色编码已存在")
			}
			return err
		}
		if status == 0 {
			// Create skips zero values on columns with a default (status has
			// default:1), so a disabled role would be stored as enabled.
			role.Status = 0
			return s.roleRepo.Update(ctx, role)
		}
		return nil
	})
}

func (s *RoleService) GetByID(ctx context.Context, id uint) (*model.SysRole, error) {
	return s.roleRepo.GetByID(ctx, id)
}

func (s *RoleService) Update(ctx context.Context, op Operator, id uint, req *UpdateRoleRequest) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "角色不存在")
	}
	if err := checkRoleWritable(op, role); err != nil {
		return err
	}
	// AdminAuth only honors enabled roles, so disabling this one would lock
	// every super admin out.
	if role.Code == model.SuperAdminRoleCode && req.Status != nil && *req.Status != 1 {
		return forbidden("超级管理员角色不可禁用")
	}

	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Sort != nil {
		role.Sort = *req.Sort
	}
	statusChanged := req.Status != nil && *req.Status != role.Status
	if statusChanged {
		role.Status = *req.Status
	}
	if req.Remark != nil {
		role.Remark = *req.Remark
	}
	if err := s.roleRepo.Update(ctx, role); err != nil {
		return err
	}
	if statusChanged {
		// Menus and button permissions are computed from enabled roles only.
		cache.ClearAllPermissionCache()
	}
	return nil
}

func (s *RoleService) Delete(ctx context.Context, op Operator, id uint) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "角色不存在")
	}
	if role.Code == model.SuperAdminRoleCode {
		return forbidden("系统内置超级管理员角色不可删除")
	}
	if err := checkRoleWritable(op, role); err != nil {
		return err
	}

	if err := s.roleRepo.Delete(ctx, id); err != nil {
		return err
	}
	if _, err := app.Enforcer.RemoveFilteredPolicy(0, role.Code); err != nil {
		return reloadPolicyError("清除角色权限策略失败", err)
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
// the APIs of its enabled menus (see rebuildRolePolicies). Unknown menu IDs
// are rejected rather than inserted as blank menus by Association.Replace.
//
// A non-super admin may only grant menus they hold themselves, so the "edit
// role" permission cannot be used to hand out more than the operator has.
func (s *RoleService) SetMenus(ctx context.Context, op Operator, roleID uint, menuIDs []uint) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return notFoundOr(err, "角色不存在")
	}
	if err := checkRoleWritable(op, role); err != nil {
		return err
	}
	if !op.IsSuperAdmin() {
		owned, err := s.roleRepo.GetMenuIDsByCodes(ctx, op.RoleCodes)
		if err != nil {
			return err
		}
		ownedSet := make(map[uint]struct{}, len(owned))
		for _, id := range owned {
			ownedSet[id] = struct{}{}
		}
		for _, id := range menuIDs {
			if _, ok := ownedSet[id]; !ok {
				return forbidden("只能分配自己拥有的菜单权限")
			}
		}
	}

	menuIDs = uniqueIDs(menuIDs)
	if n, err := repository.NewMenuRepo().CountIDs(ctx, menuIDs); err != nil {
		return err
	} else if n != int64(len(menuIDs)) {
		return errors.New("菜单不存在")
	}

	if err := s.roleRepo.SetMenus(ctx, roleID, menuIDs); err != nil {
		return err
	}
	return rebuildRolePolicies(ctx, []model.SysRole{*role})
}

// SetAPIs grants raw API rules, bypassing menus, so it is super-admin only.
func (s *RoleService) SetAPIs(ctx context.Context, op Operator, roleID uint, apis []RoleAPIItem) error {
	if err := requireSuperAdmin(op, "只有超级管理员可以直接配置角色的 API 权限"); err != nil {
		return err
	}
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return notFoundOr(err, "角色不存在")
	}

	if _, err := app.Enforcer.RemoveFilteredPolicy(0, role.Code); err != nil {
		return reloadPolicyError("清除旧权限策略失败", err)
	}
	var policies rolePolicies
	for _, api := range apis {
		policies.add(role.Code, api.Path, api.Method)
	}
	if len(policies.rules) > 0 {
		if _, err := app.Enforcer.AddPolicies(policies.rules); err != nil {
			return reloadPolicyError("写入权限策略失败", err)
		}
	}
	cache.ClearAllPermissionCache()
	return nil
}

// rolePolicies collects (role, path, method) rules without duplicates. Menus
// often share an API (用户详情 and 编辑用户 both grant GET /admin/users/:id),
// and casbin_rule has a unique index, so one duplicate fails the whole batch
// insert after the role's old rules were already removed.
type rolePolicies struct {
	rules [][]string
	seen  map[[3]string]struct{}
}

func (p *rolePolicies) add(roleCode, path, method string) {
	key := [3]string{roleCode, path, method}
	if _, ok := p.seen[key]; ok {
		return
	}
	if p.seen == nil {
		p.seen = make(map[[3]string]struct{})
	}
	p.seen[key] = struct{}{}
	p.rules = append(p.rules, []string{roleCode, path, method})
}

// checkRoleWritable: only a super admin may change the super-admin role, and a
// non-super admin may not change a role they hold, which would let them grant
// themselves more permissions.
func checkRoleWritable(op Operator, role *model.SysRole) error {
	if op.IsSuperAdmin() {
		return nil
	}
	if role.Code == model.SuperAdminRoleCode {
		return forbidden("只有超级管理员可以修改超级管理员角色")
	}
	if op.holdsRole(role.Code) {
		return forbidden("不能修改自己所属的角色")
	}
	return nil
}

func reloadPolicyError(operation string, cause error) error {
	if err := app.Enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("%s: %w; 回滚内存策略失败: %v", operation, cause, err)
	}
	return fmt.Errorf("%s: %w", operation, cause)
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
