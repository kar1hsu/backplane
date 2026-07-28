package repository

import (
	"context"

	"github.com/kar1hsu/backplane/internal/model"
)

type MenuRepo struct {
	BaseRepo[model.SysMenu]
}

func NewMenuRepo() *MenuRepo {
	return &MenuRepo{}
}

// GetByID overrides the generic version to preload APIs.
func (d *MenuRepo) GetByID(ctx context.Context, id uint) (*model.SysMenu, error) {
	return d.BaseRepo.GetByID(ctx, id, "APIs")
}

// Update writes only base columns; APIs is managed by SetAPIs.
func (d *MenuRepo) Update(ctx context.Context, menu *model.SysMenu) error {
	return d.BaseRepo.Update(ctx, menu, "ParentID", "Name", "Path", "Component", "Icon", "Sort", "Type", "Permission", "Visible", "Status")
}

// Delete removes the menu and its API associations (sys_menu_api).
func (d *MenuRepo) Delete(ctx context.Context, id uint) error {
	return dbFrom(ctx).Select("APIs").Delete(&model.SysMenu{ID: id}).Error
}

func (d *MenuRepo) ListAll(ctx context.Context) ([]model.SysMenu, error) {
	var menus []model.SysMenu
	if err := dbFrom(ctx).Order("sort ASC, id ASC").Find(&menus).Error; err != nil {
		return nil, err
	}
	return menus, nil
}

func (d *MenuRepo) GetByIDs(ctx context.Context, ids []uint) ([]model.SysMenu, error) {
	var menus []model.SysMenu
	if err := dbFrom(ctx).Preload("APIs").Where("id IN ?", ids).Order("sort ASC, id ASC").Find(&menus).Error; err != nil {
		return nil, err
	}
	return menus, nil
}

func (d *MenuRepo) SetAPIs(ctx context.Context, menuID uint, apiIDs []uint) error {
	menu := &model.SysMenu{ID: menuID}
	var apis []model.SysAPI
	for _, id := range apiIDs {
		apis = append(apis, model.SysAPI{ID: id})
	}
	return dbFrom(ctx).Model(menu).Association("APIs").Replace(apis)
}

func (d *MenuRepo) HasChildren(ctx context.Context, id uint) (bool, error) {
	var count int64
	if err := dbFrom(ctx).Model(&model.SysMenu{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func BuildMenuTree(menus []model.SysMenu, parentID uint) []*model.SysMenu {
	tree := make([]*model.SysMenu, 0)
	for i := range menus {
		if menus[i].ParentID == parentID {
			node := &menus[i]
			node.Children = BuildMenuTree(menus, node.ID)
			tree = append(tree, node)
		}
	}
	return tree
}
