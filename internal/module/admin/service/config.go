package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/setting"
	"github.com/kar1hsu/backplane/internal/repository"
)

type ConfigService struct {
	repo *repository.ConfigRepo
}

func NewConfigService() *ConfigService {
	return &ConfigService{repo: repository.NewConfigRepo()}
}

var allowedConfigTypes = map[string]bool{
	"string": true, "int": true, "float": true,
	"bool": true, "text": true, "json": true, "select": true,
}

// validateConfigDef ensures the type is known and, for select, that options is a
// non-empty JSON array — so bad definitions can't reach the DB/UI.
func validateConfigDef(typ, options string) error {
	if !allowedConfigTypes[typ] {
		return fmt.Errorf("不支持的配置类型: %s", typ)
	}
	if typ == "select" {
		if strings.TrimSpace(options) == "" {
			return errors.New("select 类型必须提供选项(options)")
		}
		var arr []json.RawMessage
		if err := json.Unmarshal([]byte(options), &arr); err != nil {
			return errors.New("options 必须是 JSON 数组")
		}
		if len(arr) == 0 {
			return errors.New("select 选项不能为空")
		}
	}
	return nil
}

func (s *ConfigService) List(ctx context.Context, group string) ([]model.SysConfig, error) {
	q := &repository.QueryOptions{Order: []string{"sort ASC", "id ASC"}}
	if group != "" {
		q.Where = map[string]interface{}{"group": group}
	}
	return s.repo.List(ctx, q)
}

type ConfigItem struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

// BatchUpdate writes many values in one transaction, then re-syncs the cache.
func (s *ConfigService) BatchUpdate(ctx context.Context, items []ConfigItem) error {
	if len(items) == 0 {
		return nil
	}
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		for _, it := range items {
			if err := s.repo.UpdateValue(ctx, it.Key, it.Value); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return setting.RefreshAll(context.Background())
}

type CreateConfigRequest struct {
	Group    string `json:"group"`
	Key      string `json:"key" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Value    string `json:"value"`
	Type     string `json:"type"`
	Options  string `json:"options"`
	IsPublic bool   `json:"is_public"`
	Remark   string `json:"remark"`
	Sort     int    `json:"sort"`
}

func (s *ConfigService) Create(ctx context.Context, req *CreateConfigRequest) error {
	typ := req.Type
	if typ == "" {
		typ = "string"
	}
	if err := validateConfigDef(typ, req.Options); err != nil {
		return err
	}
	exists, err := s.repo.Exists(ctx, &repository.QueryOptions{
		Where: map[string]interface{}{"key": req.Key},
	})
	if err != nil {
		return err
	}
	if exists {
		return errors.New("配置键已存在")
	}
	c := &model.SysConfig{
		Group: req.Group, Key: req.Key, Name: req.Name, Value: req.Value,
		Type: typ, Options: req.Options, IsPublic: req.IsPublic,
		Remark: req.Remark, Sort: req.Sort, Editable: true, Builtin: false,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return err
	}
	_ = setting.RefreshKey(context.Background(), req.Key)
	return nil
}

type UpdateConfigRequest struct {
	Group    string `json:"group"`
	Name     string `json:"name" binding:"required"`
	Value    string `json:"value"`
	Type     string `json:"type"`
	Options  string `json:"options"`
	IsPublic bool   `json:"is_public"`
	Remark   string `json:"remark"`
	Sort     int    `json:"sort"`
}

// Update edits a config's metadata and value (the key is immutable), then
// refreshes the cache for that key.
func (s *ConfigService) Update(ctx context.Context, id uint, req *UpdateConfigRequest) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "配置不存在")
	}
	typ := req.Type
	if typ == "" {
		typ = "string"
	}
	if err := validateConfigDef(typ, req.Options); err != nil {
		return err
	}
	c.Group = req.Group
	c.Name = req.Name
	c.Value = req.Value
	c.Type = typ
	c.Options = req.Options
	c.IsPublic = req.IsPublic
	c.Remark = req.Remark
	c.Sort = req.Sort
	if err := s.repo.Update(ctx, c, "Group", "Name", "Value", "Type", "Options", "IsPublic", "Remark", "Sort"); err != nil {
		return err
	}
	_ = setting.RefreshKey(context.Background(), c.Key)
	return nil
}

func (s *ConfigService) Delete(ctx context.Context, id uint) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "配置不存在")
	}
	if c.Builtin {
		return errors.New("内置配置不可删除")
	}
	// Hard delete: configs are reference data, and a soft-deleted row would keep
	// the unique key, blocking re-creating the same key later.
	if err := s.repo.HardDelete(ctx, id); err != nil {
		return err
	}
	_ = setting.RefreshKey(context.Background(), c.Key) // drops the cache field
	return nil
}

// Refresh re-syncs the cache: a single key when given, otherwise everything.
func (s *ConfigService) Refresh(ctx context.Context, key string) error {
	if key != "" {
		return setting.RefreshKey(ctx, key)
	}
	return setting.RefreshAll(ctx)
}
