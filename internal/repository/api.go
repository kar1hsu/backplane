package repository

import (
	"context"

	"github.com/kar1hsu/backplane/internal/model"
)

type ApiRepo struct {
	BaseRepo[model.SysAPI]
}

func NewApiRepo() *ApiRepo {
	return &ApiRepo{}
}

// SysAPI has no associations, so Create/GetByID/Delete/PageList come from BaseRepo.

func (d *ApiRepo) Update(ctx context.Context, api *model.SysAPI) error {
	return dbFrom(ctx).Save(api).Error
}

func (d *ApiRepo) ListAll(ctx context.Context) ([]model.SysAPI, error) {
	var apis []model.SysAPI
	if err := dbFrom(ctx).Order("`group` ASC, id ASC").Find(&apis).Error; err != nil {
		return nil, err
	}
	return apis, nil
}
