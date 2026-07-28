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
