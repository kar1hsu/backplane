package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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

// validateConfigValue checks value against the config's type, so a typo like
// "30天" in an int setting is rejected instead of silently read back as 0.
func validateConfigValue(typ, options, value string) error {
	switch typ {
	case "int":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return fmt.Errorf("值必须是整数: %q", value)
		}
	case "float":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("值必须是数字: %q", value)
		}
	case "bool":
		if value != "true" && value != "false" {
			return fmt.Errorf("值必须是 true 或 false: %q", value)
		}
	case "json":
		if value != "" && !json.Valid([]byte(value)) {
			return errors.New("值必须是合法的 JSON")
		}
	case "select":
		allowed := selectOptionValues(options)
		if len(allowed) > 0 && !allowed[value] {
			return fmt.Errorf("值不在可选项中: %q", value)
		}
	}
	return nil
}

// selectOptionValues extracts the allowed values of a select config, whose
// options are a JSON array of strings or of {"label","value"} objects.
func selectOptionValues(options string) map[string]bool {
	var items []json.RawMessage
	if err := json.Unmarshal([]byte(options), &items); err != nil {
		return nil
	}
	allowed := make(map[string]bool, len(items))
	for _, item := range items {
		var str string
		if json.Unmarshal(item, &str) == nil {
			allowed[str] = true
			continue
		}
		var obj struct {
			Value interface{} `json:"value"`
		}
		if json.Unmarshal(item, &obj) == nil && obj.Value != nil {
			allowed[fmt.Sprint(obj.Value)] = true
		}
	}
	return allowed
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
// Every key must exist, be editable and hold a value valid for its type.
func (s *ConfigService) BatchUpdate(ctx context.Context, items []ConfigItem) error {
	if len(items) == 0 {
		return nil
	}
	keys := make([]string, 0, len(items))
	for _, it := range items {
		keys = append(keys, it.Key)
	}
	existing, err := s.repo.ListByKeys(ctx, keys)
	if err != nil {
		return err
	}
	byKey := make(map[string]model.SysConfig, len(existing))
	for _, c := range existing {
		byKey[c.Key] = c
	}
	for _, it := range items {
		c, ok := byKey[it.Key]
		if !ok {
			return fmt.Errorf("配置不存在: %s", it.Key)
		}
		if err := checkValueChange(&c, it.Value); err != nil {
			return err
		}
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
	if err := validateConfigValue(typ, req.Options, req.Value); err != nil {
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
		typ = c.Type // keep the current type rather than resetting it to string
	}
	if c.Builtin && (typ != c.Type || req.Options != c.Options) {
		return errors.New("内置配置不可修改类型或选项")
	}
	if err := validateConfigDef(typ, req.Options); err != nil {
		return err
	}
	c.Type, c.Options = typ, req.Options
	if err := checkValueChange(c, req.Value); err != nil {
		return err
	}
	c.Group = req.Group
	c.Name = req.Name
	c.Value = req.Value
	c.IsPublic = req.IsPublic
	c.Remark = req.Remark
	c.Sort = req.Sort
	if err := s.repo.Update(ctx, c, "Group", "Name", "Value", "Type", "Options", "IsPublic", "Remark", "Sort"); err != nil {
		return err
	}
	_ = setting.RefreshKey(context.Background(), c.Key)
	return nil
}

// checkValueChange rejects changing a non-editable config and values invalid
// for the config's type. Re-saving the current value is always allowed.
func checkValueChange(c *model.SysConfig, value string) error {
	if value == c.Value {
		return nil
	}
	if !c.Editable {
		return fmt.Errorf("配置不允许修改: %s", c.Key)
	}
	if err := validateConfigValue(c.Type, c.Options, value); err != nil {
		return fmt.Errorf("%s: %w", c.Key, err)
	}
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
