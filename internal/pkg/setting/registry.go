package setting

import (
	"context"

	"github.com/kar1hsu/frame/internal/model"
	"github.com/kar1hsu/frame/internal/repository"
)

// definition is a known config: its default value, type and metadata. The
// registry is the single source for seeding the DB and for the fallback default
// when both cache and DB miss.
type definition struct {
	Group    string
	Key      string
	Name     string
	Type     string // string | int | bool | float | text | json | select
	Value    string
	Options  string
	Remark   string
	IsPublic bool
}

// registry lists the built-in configs. Add new tunables here; they are seeded on
// next startup (idempotently) and editable from the admin UI afterwards.
var registry = []definition{
	{Group: "站点", Key: "site.name", Name: "站点名称", Type: "string", Value: "Frame Admin", IsPublic: true, Remark: "登录页 / 浏览器标题"},
	{Group: "站点", Key: "site.logo", Name: "站点 Logo", Type: "string", Value: "", IsPublic: true, Remark: "Logo 图片 URL"},
	{Group: "站点", Key: "site.description", Name: "站点描述", Type: "text", Value: "后台管理系统", IsPublic: true},
	{Group: "站点", Key: "site.copyright", Name: "版权信息", Type: "string", Value: "", IsPublic: true},
	{Group: "站点", Key: "site.icp", Name: "ICP 备案号", Type: "string", Value: "", IsPublic: true},

	{Group: "用户", Key: "user.allow_register", Name: "允许注册", Type: "bool", Value: "false", Remark: "是否开放自助注册"},
	{Group: "用户", Key: "user.default_role", Name: "默认角色编码", Type: "string", Value: "", Remark: "注册用户默认角色"},

	{Group: "安全", Key: "security.login_fail_limit", Name: "登录失败锁定次数", Type: "int", Value: "5"},
	{Group: "安全", Key: "security.login_lock_minutes", Name: "锁定时长(分钟)", Type: "int", Value: "15"},
	{Group: "安全", Key: "security.password_min_length", Name: "密码最小长度", Type: "int", Value: "6"},

	{Group: "日志", Key: "log.operation_retain_days", Name: "操作日志保留天数", Type: "int", Value: "30", Remark: "留存清理任务读取此值，0 表示不清理"},
}

// defaultValue returns the compiled-in default for key, or "" if unknown.
func defaultValue(key string) string {
	for i := range registry {
		if registry[i].Key == key {
			return registry[i].Value
		}
	}
	return ""
}

// seedDefaults inserts any registry entry not already present (matched by key),
// so existing deployments pick up newly added configs without overwriting values
// an admin has changed.
func seedDefaults(ctx context.Context) error {
	for _, d := range registry {
		exists, err := repo.Exists(ctx, &repository.QueryOptions{
			Where: map[string]interface{}{"key": d.Key},
		})
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := repo.Create(ctx, &model.SysConfig{
			Group:    d.Group,
			Key:      d.Key,
			Name:     d.Name,
			Type:     d.Type,
			Value:    d.Value,
			Options:  d.Options,
			Remark:   d.Remark,
			IsPublic: d.IsPublic,
			Editable: true,
			Builtin:  true,
		}); err != nil {
			return err
		}
	}
	return nil
}
