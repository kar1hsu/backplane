package repository

import (
	"context"

	"github.com/kar1hsu/backplane/internal/model"
	"gorm.io/gorm/clause"
)

type ApiRepo struct {
	BaseRepo[model.SysAPI]
}

func NewApiRepo() *ApiRepo {
	return &ApiRepo{}
}

// GroupOrder sorts by the "group" column, quoted per dialect: it is a reserved
// word, and a hand-written MySQL backtick is a syntax error on PostgreSQL.
var GroupOrder = []clause.OrderByColumn{
	{Column: clause.Column{Name: "group"}},
	{Column: clause.Column{Name: "id"}},
}

// Update writes the editable columns only. Save would fall back to an upsert
// and resurrect a row soft-deleted in the meantime.
func (d *ApiRepo) Update(ctx context.Context, api *model.SysAPI) error {
	return d.BaseRepo.Update(ctx, api, "Path", "Method", "Group", "Description")
}

// Delete soft-deletes the API and unbinds it from every menu (sys_menu_api).
func (d *ApiRepo) Delete(ctx context.Context, id uint) error {
	return Transaction(ctx, func(ctx context.Context) error {
		if err := dbFrom(ctx).Exec("DELETE FROM sys_menu_api WHERE sys_api_id = ?", id).Error; err != nil {
			return err
		}
		return d.BaseRepo.Delete(ctx, id)
	})
}

func (d *ApiRepo) ListAll(ctx context.Context) ([]model.SysAPI, error) {
	var apis []model.SysAPI
	q := &QueryOptions{OrderColumns: GroupOrder}
	if err := q.applyAll(dbFrom(ctx).Model(&model.SysAPI{})).Find(&apis).Error; err != nil {
		return nil, err
	}
	return apis, nil
}
