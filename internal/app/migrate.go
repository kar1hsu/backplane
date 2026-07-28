package app

import (
	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/utils"
	"gorm.io/gorm"
)

// AutoMigrate creates or updates the schema for every model.
func AutoMigrate() error {
	return DB.AutoMigrate(
		&model.SysUser{},
		&model.SysRole{},
		&model.SysMenu{},
		&model.SysAPI{},
		&model.SysConfig{},
		&model.SysOperationLog{},
	)
}

// SeedData populates the default super-admin plus the full RBAC tree — every
// built-in menu and API (user / role / menu / API management, operation log,
// system config) — and grants it all to the admin role.
//
// It is a one-shot, fresh-install seed: if the admin role already exists it
// returns immediately, so it's safe to call on every startup. The runtime
// config VALUES (site.name, …) are seeded separately by setting.Init from the
// defaults registry, which also provides their fallback values.
func SeedData() error {
	var count int64
	DB.Model(&model.SysRole{}).Where("code = ?", "admin").Count(&count)
	if count > 0 {
		return nil
	}

	Log.Info("seeding initial data...")

	if err := DB.Transaction(func(tx *gorm.DB) error {
		apis := defaultAPIs()
		for i := range apis {
			if err := tx.Create(&apis[i]).Error; err != nil {
				return err
			}
		}

		adminRole := &model.SysRole{
			Name:   "超级管理员",
			Code:   "admin",
			Sort:   0,
			Status: 1,
			Remark: "超级管理员，拥有所有权限",
		}
		if err := tx.Create(adminRole).Error; err != nil {
			return err
		}

		hashed, err := utils.HashPassword("admin123")
		if err != nil {
			return err
		}
		adminUser := &model.SysUser{
			Username: "admin",
			Password: hashed,
			Nickname: "管理员",
			Status:   1,
			Roles:    []model.SysRole{*adminRole},
		}
		if err := tx.Create(adminUser).Error; err != nil {
			return err
		}

		menus := defaultMenus(apis)
		for i := range menus {
			if err := tx.Create(&menus[i]).Error; err != nil {
				return err
			}
		}

		// Grant every built-in menu to the super-admin role.
		if err := tx.Model(adminRole).Association("Menus").Replace(menus); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	Log.Warn("已创建默认管理员 admin/admin123，请登录后立即修改密码")
	Log.Info("seed data completed")
	return nil
}

// defaultAPIs is the full set of built-in API records. IDs are explicit and
// stable so menus can reference them by index below (apis[id-1]).
func defaultAPIs() []model.SysAPI {
	return []model.SysAPI{
		// 用户管理 (1-5)
		{ID: 1, Path: "/admin/users", Method: "GET", Group: "用户管理", Description: "用户列表"},
		{ID: 2, Path: "/admin/users", Method: "POST", Group: "用户管理", Description: "创建用户"},
		{ID: 3, Path: "/admin/users/:id", Method: "GET", Group: "用户管理", Description: "用户详情"},
		{ID: 4, Path: "/admin/users/:id", Method: "PUT", Group: "用户管理", Description: "更新用户"},
		{ID: 5, Path: "/admin/users/:id", Method: "DELETE", Group: "用户管理", Description: "删除用户"},
		// 角色管理 (6-11)
		{ID: 6, Path: "/admin/roles", Method: "GET", Group: "角色管理", Description: "角色列表"},
		{ID: 7, Path: "/admin/roles", Method: "POST", Group: "角色管理", Description: "创建角色"},
		{ID: 8, Path: "/admin/roles/:id", Method: "GET", Group: "角色管理", Description: "角色详情"},
		{ID: 9, Path: "/admin/roles/:id", Method: "PUT", Group: "角色管理", Description: "更新角色"},
		{ID: 10, Path: "/admin/roles/:id", Method: "DELETE", Group: "角色管理", Description: "删除角色"},
		{ID: 11, Path: "/admin/roles/:id/menus", Method: "PUT", Group: "角色管理", Description: "分配菜单"},
		// 菜单管理 (12-15)
		{ID: 12, Path: "/admin/menus", Method: "POST", Group: "菜单管理", Description: "创建菜单"},
		{ID: 13, Path: "/admin/menus/:id", Method: "GET", Group: "菜单管理", Description: "菜单详情"},
		{ID: 14, Path: "/admin/menus/:id", Method: "PUT", Group: "菜单管理", Description: "更新菜单"},
		{ID: 15, Path: "/admin/menus/:id", Method: "DELETE", Group: "菜单管理", Description: "删除菜单"},
		// API管理 (16-19)
		{ID: 16, Path: "/admin/apis", Method: "GET", Group: "API管理", Description: "API列表"},
		{ID: 17, Path: "/admin/apis", Method: "POST", Group: "API管理", Description: "创建API"},
		{ID: 18, Path: "/admin/apis/:id", Method: "PUT", Group: "API管理", Description: "更新API"},
		{ID: 19, Path: "/admin/apis/:id", Method: "DELETE", Group: "API管理", Description: "删除API"},
		// 系统配置 (20-25)
		{ID: 20, Path: "/admin/configs", Method: "GET", Group: "系统配置", Description: "配置列表"},
		{ID: 21, Path: "/admin/configs", Method: "POST", Group: "系统配置", Description: "新增配置"},
		{ID: 22, Path: "/admin/configs", Method: "PUT", Group: "系统配置", Description: "保存配置"},
		{ID: 23, Path: "/admin/configs/:id", Method: "DELETE", Group: "系统配置", Description: "删除配置"},
		{ID: 24, Path: "/admin/configs/refresh", Method: "POST", Group: "系统配置", Description: "刷新配置缓存"},
		{ID: 25, Path: "/admin/configs/:id", Method: "PUT", Group: "系统配置", Description: "编辑配置"},
		// 操作日志 (26-29)
		{ID: 26, Path: "/admin/operation-logs", Method: "GET", Group: "操作日志", Description: "日志列表"},
		{ID: 27, Path: "/admin/operation-logs/:id", Method: "GET", Group: "操作日志", Description: "日志详情"},
		{ID: 28, Path: "/admin/operation-logs/:id", Method: "DELETE", Group: "操作日志", Description: "删除日志"},
		{ID: 29, Path: "/admin/operation-logs", Method: "DELETE", Group: "操作日志", Description: "清空日志"},
	}
}

// defaultMenus is the full built-in menu tree (directories, menus and buttons),
// each wiring in the APIs it authorizes. a is the slice from defaultAPIs, so
// a[n-1] is the API with ID n.
func defaultMenus(a []model.SysAPI) []model.SysMenu {
	return []model.SysMenu{
		// ── 系统管理（目录）──
		{ID: 1, ParentID: 0, Name: "系统管理", Path: "/system", Icon: "Setting", Sort: 1, Type: 0, Visible: 1, Status: 1},

		// ── 用户管理 ──
		{ID: 2, ParentID: 1, Name: "用户管理", Path: "/system/user", Component: "system/user/index", Icon: "User", Sort: 1, Type: 1, Permission: "system:user:list", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[0]}},
		{ID: 20, ParentID: 2, Name: "用户详情", Sort: 1, Type: 2, Permission: "system:user:query", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[2]}},
		{ID: 21, ParentID: 2, Name: "新增用户", Sort: 2, Type: 2, Permission: "system:user:add", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[1]}},
		{ID: 22, ParentID: 2, Name: "编辑用户", Sort: 3, Type: 2, Permission: "system:user:edit", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[2], a[3]}},
		{ID: 23, ParentID: 2, Name: "删除用户", Sort: 4, Type: 2, Permission: "system:user:delete", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[4]}},

		// ── 角色管理 ──
		{ID: 3, ParentID: 1, Name: "角色管理", Path: "/system/role", Component: "system/role/index", Icon: "UserFilled", Sort: 2, Type: 1, Permission: "system:role:list", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[5]}},
		{ID: 30, ParentID: 3, Name: "角色详情", Sort: 1, Type: 2, Permission: "system:role:query", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[7]}},
		{ID: 31, ParentID: 3, Name: "新增角色", Sort: 2, Type: 2, Permission: "system:role:add", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[6]}},
		{ID: 32, ParentID: 3, Name: "编辑角色", Sort: 3, Type: 2, Permission: "system:role:edit", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[7], a[8], a[10]}},
		{ID: 33, ParentID: 3, Name: "删除角色", Sort: 4, Type: 2, Permission: "system:role:delete", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[9]}},

		// ── 菜单管理 ──
		{ID: 4, ParentID: 1, Name: "菜单管理", Path: "/system/menu", Component: "system/menu/index", Icon: "Menu", Sort: 3, Type: 1, Permission: "system:menu:list", Visible: 1, Status: 1},
		{ID: 40, ParentID: 4, Name: "菜单详情", Sort: 1, Type: 2, Permission: "system:menu:query", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[12]}},
		{ID: 41, ParentID: 4, Name: "新增菜单", Sort: 2, Type: 2, Permission: "system:menu:add", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[11]}},
		{ID: 42, ParentID: 4, Name: "编辑菜单", Sort: 3, Type: 2, Permission: "system:menu:edit", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[12], a[13]}},
		{ID: 43, ParentID: 4, Name: "删除菜单", Sort: 4, Type: 2, Permission: "system:menu:delete", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[14]}},

		// ── 系统配置 ──
		{ID: 5, ParentID: 1, Name: "系统配置", Path: "/system/config", Component: "system/config/index", Icon: "Tools", Sort: 4, Type: 1, Permission: "system:config:list", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[19]}},
		{ID: 50, ParentID: 5, Name: "保存配置", Sort: 1, Type: 2, Permission: "system:config:edit", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[21], a[23], a[24]}},
		{ID: 51, ParentID: 5, Name: "新增配置", Sort: 2, Type: 2, Permission: "system:config:add", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[20]}},
		{ID: 52, ParentID: 5, Name: "删除配置", Sort: 3, Type: 2, Permission: "system:config:delete", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[22]}},

		// ── 操作日志 ──
		{ID: 6, ParentID: 1, Name: "操作日志", Path: "/system/operlog", Component: "system/operlog/index", Icon: "Document", Sort: 5, Type: 1, Permission: "system:operlog:list", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[25], a[26]}},
		{ID: 60, ParentID: 6, Name: "删除日志", Sort: 1, Type: 2, Permission: "system:operlog:delete", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[27]}},
		{ID: 61, ParentID: 6, Name: "清空日志", Sort: 2, Type: 2, Permission: "system:operlog:clear", Visible: 1, Status: 1,
			APIs: []model.SysAPI{a[28]}},
	}
}
