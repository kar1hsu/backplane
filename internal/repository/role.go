package repository

import (
	"context"

	"github.com/kar1hsu/backplane/internal/model"
	"gorm.io/gorm"
)

type RoleRepo struct {
	BaseRepo[model.SysRole]
}

func NewRoleRepo() *RoleRepo {
	return &RoleRepo{}
}

// GetByID overrides the generic version to preload Menus.
func (d *RoleRepo) GetByID(ctx context.Context, id uint) (*model.SysRole, error) {
	return d.BaseRepo.GetByID(ctx, id, "Menus")
}

func (d *RoleRepo) GetByCode(ctx context.Context, code string) (*model.SysRole, error) {
	var role model.SysRole
	if err := dbFrom(ctx).Where("code = ?", code).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// Update writes only base columns; Menus is managed by SetMenus.
func (d *RoleRepo) Update(ctx context.Context, role *model.SysRole) error {
	return d.BaseRepo.Update(ctx, role, "Name", "Sort", "Status", "Remark")
}

// Delete soft-deletes the role, clears its menu associations (sys_role_menu),
// and mangles the code so the unique index is freed — letting the same code be
// reused later. The soft-deleted row is kept for audit.
func (d *RoleRepo) Delete(ctx context.Context, id uint) error {
	return Transaction(ctx, func(ctx context.Context) error {
		if err := dbFrom(ctx).Model(&model.SysRole{}).Where("id = ?", id).
			Update("code", gorm.Expr("CONCAT('del#', id, '#', LEFT(code, 40))")).Error; err != nil {
			return err
		}
		return dbFrom(ctx).Select("Menus").Delete(&model.SysRole{ID: id}).Error
	})
}

func (d *RoleRepo) GetByIDs(ctx context.Context, ids []uint) ([]model.SysRole, error) {
	var roles []model.SysRole
	if len(ids) == 0 {
		return roles, nil
	}
	if err := dbFrom(ctx).Where("id IN ?", ids).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// GetMenuIDsByCodes returns the IDs of every menu granted to the enabled roles
// with the given codes.
func (d *RoleRepo) GetMenuIDsByCodes(ctx context.Context, codes []string) ([]uint, error) {
	var ids []uint
	if len(codes) == 0 {
		return ids, nil
	}
	err := dbFrom(ctx).Table("sys_role_menu").
		Joins("JOIN sys_role ON sys_role.id = sys_role_menu.sys_role_id").
		Where("sys_role.code IN ? AND sys_role.status = ? AND sys_role.deleted_at IS NULL", codes, 1).
		Distinct().Pluck("sys_role_menu.sys_menu_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (d *RoleRepo) ListAll(ctx context.Context) ([]model.SysRole, error) {
	var roles []model.SysRole
	if err := dbFrom(ctx).Where("status = 1").Order("sort ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (d *RoleRepo) SetMenus(ctx context.Context, roleID uint, menuIDs []uint) error {
	role := &model.SysRole{ID: roleID}
	var menus []model.SysMenu
	for _, id := range menuIDs {
		menus = append(menus, model.SysMenu{ID: id})
	}
	return dbFrom(ctx).Model(role).Association("Menus").Replace(menus)
}

func (d *RoleRepo) GetMenusByRoleID(ctx context.Context, roleID uint) ([]model.SysMenu, error) {
	role := &model.SysRole{ID: roleID}
	var menus []model.SysMenu
	if err := dbFrom(ctx).Model(role).Association("Menus").Find(&menus); err != nil {
		return nil, err
	}
	return menus, nil
}
