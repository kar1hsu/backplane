package repository

import (
	"context"

	"github.com/kar1hsu/backplane/internal/model"
)

type ConfigRepo struct {
	BaseRepo[model.SysConfig]
}

func NewConfigRepo() *ConfigRepo {
	return &ConfigRepo{}
}

// GetByKey fetches a config by its unique key (returns gorm.ErrRecordNotFound
// when absent, so callers can errors.Is it).
func (d *ConfigRepo) GetByKey(ctx context.Context, key string) (*model.SysConfig, error) {
	return d.GetOne(ctx, &QueryOptions{Where: map[string]interface{}{"key": key}})
}

// ListAll returns every config ordered for stable display.
func (d *ConfigRepo) ListAll(ctx context.Context) ([]model.SysConfig, error) {
	return d.List(ctx, &QueryOptions{Order: []string{"sort ASC", "id ASC"}})
}

// ListPublic returns only configs flagged is_public (for the unauthenticated
// bootstrap endpoint).
func (d *ConfigRepo) ListPublic(ctx context.Context) ([]model.SysConfig, error) {
	return d.List(ctx, &QueryOptions{
		Where: map[string]interface{}{"is_public": true},
		Order: []string{"sort ASC", "id ASC"},
	})
}

// UpdateValue updates just the value column for key. `key` is backtick-escaped
// because it is a SQL reserved word.
func (d *ConfigRepo) UpdateValue(ctx context.Context, key, value string) error {
	return dbFrom(ctx).Model(&model.SysConfig{}).
		Where("`key` = ?", key).Update("value", value).Error
}
