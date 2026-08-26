# Backplane - Go 模块化后台管理框架

[![CI](https://github.com/kar1hsu/backplane/actions/workflows/ci.yml/badge.svg)](https://github.com/kar1hsu/backplane/actions/workflows/ci.yml)

简体中文 | [English](README.en-US.md)

Backplane 是一个模块化的 Go 后台管理框架。后端基于 **Gin + GORM + Casbin + JWT**，前端 **Vue 3 + Element Plus** 通过 `embed` 编进二进制——一次 `go build` 即可交付整个应用。开箱即用：**按钮级 RBAC**（自动生成 Casbin 策略）、**数据库驱动的操作审计日志**、**运行时系统配置**（DB 为准、Redis 缓存、多实例即时生效），以及基于 **Asynq 的分布式任务队列**（含定时任务）。

> **Naive UI 重构分支**：管理后台的 Naive UI 版本位于 [`naive-ui`](https://github.com/kar1hsu/backplane/tree/naive-ui) 分支，已重写登录页、整体布局和系统管理页面。可执行 `git switch naive-ui` 在本地体验。

如果 Backplane 对你有帮助，欢迎在 [GitHub 仓库](https://github.com/kar1hsu/backplane) 点一个 **Star**，你的支持会帮助项目持续完善。

## 技术栈

| 层级 | 技术 | 用途 |
|------|------|------|
| Web 框架 | Gin | 路由、中间件、请求处理 |
| ORM | GORM | 支持 MySQL / PostgreSQL，通过配置切换 |
| 鉴权 | golang-jwt | JWT Token 签发与验证 |
| 权限管理 | Casbin | RBAC 权限模型，支持路径参数匹配 |
| 配置管理 | Viper | YAML/ENV 多环境配置 |
| 日志 | Zap + Lumberjack | 结构化日志，自动轮转 |
| 缓存 | Redis (go-redis) | Token 黑名单、权限缓存、登录限流 |
| 任务队列 | Asynq | 分布式消息队列 + 定时任务，基于 Redis |
| 密码 | bcrypt | 密码哈希加密 |
| 前端 | Vue 3 + Element Plus + Vite | 管理面板，通过 Go embed 嵌入 |
| 部署 | Docker + docker-compose | 一键容器化部署 |

## 功能特性

- **模块化架构** - 通过 `Module` 接口扩展，自带 Admin 和 API 两个模块
- **JWT 鉴权** - Bearer Token 认证，支持退出登录 Token 黑名单
- **登录保护** - Redis 记录登录失败次数，5 次失败后锁定 15 分钟
- **RBAC 权限** - 基于 Casbin 的角色权限管理，支持 `keyMatch2` 路径参数匹配
- **按钮级权限** - 菜单关联 API，支持 list/query/add/edit/delete 细粒度控制
- **权限自动同步** - 分配菜单时自动生成对应的 Casbin API 策略，零手动配置
- **权限缓存** - Redis 缓存用户权限列表，角色变更时自动清除
- **用户管理** - 用户 CRUD、密码加密、角色分配
- **角色管理** - 角色 CRUD、菜单权限分配（含按钮粒度）
- **菜单管理** - 树形菜单管理（目录/菜单/按钮三级）、动态菜单
- **API 管理** - API 接口注册，与菜单/按钮关联，数据库驱动的权限配置
- **操作日志** - 自动审计所有写操作（操作人/模块/参数/结果/耗时），含登录失败审计，支持检索与清理
- **系统配置** - 数据库驱动的运行时配置，DB 为准 + Redis 共享缓存，改完即时生效、多实例一致，代码内类型化读取
- **消息队列** - 基于 Asynq + Redis 的分布式任务队列，支持即时/延迟/唯一任务
- **定时任务** - Asynq Scheduler，cron 语法，独立 Scheduler 进程；可选 Unique 去重防多实例重复投递
- **前端面板** - Element Plus 管理界面，通过 Go embed 内嵌到二进制
- **多数据库** - 通过配置切换 MySQL 或 PostgreSQL

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 20+（构建前端）
- MySQL 8.0+ 或 PostgreSQL 14+
- Redis 6+

### 1. 克隆项目

```bash
git clone git@github.com:kar1hsu/backplane.git
cd backplane
```

### 2. 配置

编辑 `config/config.yaml`，修改数据库和 Redis 连接信息。

### 3. 构建前端

```bash
cd web/admin
npm install
npm run build
cd ../..
```

### 4. 启动服务

```bash
go run cmd/server/main.go
```

服务启动后：
- 后台管理面板: http://localhost:8080
- Admin API: http://localhost:8080/admin/*
- 对外 API: http://localhost:8080/api/*

### 5. 默认账号

| 用户名 | 密码 | 角色 |
|--------|------|------|
| admin | admin123 | 超级管理员 |

### 6. 前端开发模式

```bash
cd web/admin
npm run dev
```

Vite 开发服务器启动后访问 `http://localhost:5173`，API 自动代理到后端 `http://localhost:8080`。

## Docker 部署

```bash
cd deploy
cp .env.example .env          # 设置数据库/Redis 密码、时区
docker compose up -d
```

说明：
- `deploy/config.yaml` 已按容器网络配好（`host: mysql` / `host: redis`）并挂载进应用容器，应用配置改这里，**不要**动根目录的 `config/config.yaml`。
- `.env` 用于 MySQL/Redis 容器。若修改了 `MYSQL_ROOT_PASSWORD` / `REDIS_PASSWORD`，需在 `deploy/config.yaml` 同步改 `database.password` / `redis.password`——应用从挂载的配置读取凭据，而非这些环境变量。

## 项目结构

```
backplane/
├── cmd/
│   ├── server/main.go          # Web 服务入口（生产者）
│   ├── worker/main.go          # Worker 进程入口（消费者，可多实例）
│   └── scheduler/main.go       # Scheduler 进程入口（定时投递，单实例）
├── config/
│   ├── config.yaml             # 应用配置
│   └── rbac_model.conf         # Casbin RBAC 模型
├── internal/
│   ├── app/                    # 应用初始化 (Config/Logger/DB/Redis/Casbin/Task)
│   ├── middleware/             # 中间件 (JWT/Casbin/CORS/Logger/OperationLog)
│   ├── model/                  # 数据模型 (User/Role/Menu/API/Config/OperationLog)
│   ├── server/                 # HTTP 服务 & 路由注册 & 静态文件
│   ├── repository/             # 数据访问层
│   ├── tasks/                  # 任务定义与注册（Handler + 定时任务）
│   ├── module/
│   │   ├── admin/              # Admin 后台模块
│   │   │   ├── handler/        # 请求处理 (Auth/User/Role/Menu/API/Config/OperationLog)
│   │   │   ├── service/        # 业务逻辑
│   │   │   └── router.go       # Admin 路由注册
│   │   └── api/                # 对外 API 模块
│   └── pkg/                    # 内部公共包
│       ├── jwt/                # JWT 签发/解析
│       ├── cache/              # 缓存层 (Store接口/RedisStore/业务缓存)
│       ├── setting/            # 运行时系统配置 (类型化访问器 + Redis缓存 + 默认值注册表)
│       ├── task/               # 任务系统 (Client/Worker/Scheduler/Manager)
│       ├── response/           # 统一响应格式
│       ├── errcode/            # 错误码
│       └── utils/              # 工具 (密码哈希/分页)
├── web/admin/                  # Vue 3 前端项目
├── embed.go                    # Go embed 嵌入前端
└── deploy/                     # Docker 部署文件
```

## 路由层级设计

| 层级 | 中间件 | 说明 | 示例 |
|------|--------|------|------|
| 公开 | 无 | 无需登录 | `POST /admin/login` |
| 已登录 | JWT | 登录即可访问（个人信息、下拉选项） | `GET /admin/profile`, `/permissions`, `/roles/all` |
| 管理权限 | JWT + Casbin | 需要 RBAC 授权的管理操作 | `GET /admin/users`, `POST /admin/users` |

## API 概览

### 认证

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | /admin/login | 登录（含登录失败限流） | 无 |
| POST | /admin/logout | 退出登录（Token 加入黑名单） | JWT |
| GET | /admin/profile | 获取当前用户信息 | JWT |
| GET | /admin/permissions | 获取当前用户权限标识列表（带缓存） | JWT |

### 用户管理

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /admin/users | 用户列表 | JWT + RBAC |
| POST | /admin/users | 创建用户 | JWT + RBAC |
| GET | /admin/users/:id | 用户详情 | JWT + RBAC |
| PUT | /admin/users/:id | 更新用户 | JWT + RBAC |
| DELETE | /admin/users/:id | 删除用户 | JWT + RBAC |

### 角色管理

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /admin/roles | 角色列表（分页） | JWT + RBAC |
| GET | /admin/roles/all | 全部角色（下拉选项） | JWT |
| POST | /admin/roles | 创建角色 | JWT + RBAC |
| GET | /admin/roles/:id | 角色详情 | JWT + RBAC |
| PUT | /admin/roles/:id | 更新角色 | JWT + RBAC |
| DELETE | /admin/roles/:id | 删除角色 | JWT + RBAC |
| PUT | /admin/roles/:id/menus | 分配菜单（自动同步 Casbin 策略） | JWT + RBAC |

### 菜单管理

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /admin/menus/tree | 完整菜单树 | JWT |
| GET | /admin/menus/user | 当前用户菜单树 | JWT |
| POST | /admin/menus | 创建菜单（支持关联 API） | JWT + RBAC |
| GET | /admin/menus/:id | 菜单详情（含关联的 API） | JWT + RBAC |
| PUT | /admin/menus/:id | 更新菜单 | JWT + RBAC |
| DELETE | /admin/menus/:id | 删除菜单 | JWT + RBAC |

### API 管理

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /admin/apis/all | 全部 API（下拉选项） | JWT |
| GET | /admin/apis | API 列表（分页） | JWT + RBAC |
| POST | /admin/apis | 创建 API | JWT + RBAC |
| PUT | /admin/apis/:id | 更新 API | JWT + RBAC |
| DELETE | /admin/apis/:id | 删除 API | JWT + RBAC |

### 系统配置

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /admin/configs | 配置列表（可选 `?group=`） | JWT + RBAC |
| POST | /admin/configs | 新增自定义配置 | JWT + RBAC |
| PUT | /admin/configs | 批量保存值 `{items:[{key,value}]}` | JWT + RBAC |
| DELETE | /admin/configs/:id | 删除配置（内置不可删） | JWT + RBAC |
| POST | /admin/configs/refresh | 刷新缓存（`?key=` 单个，否则全部） | JWT + RBAC |
| GET | /api/configs/public | 公开配置 key→value（免鉴权，供登录页/前端启动） | 无 |

### 文件上传

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | /api/upload | 上传文件（multipart：`file` 必填，`folder` 可选）；返回 `resource_domain` 与文件 `url` 供前端拼接 | 无 |

### 操作日志

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /admin/operation-logs | 操作日志列表（按操作人/模块/成败/时间段/关键字过滤） | JWT + RBAC |
| GET | /admin/operation-logs/:id | 日志详情 | JWT + RBAC |
| DELETE | /admin/operation-logs/:id | 删除单条 | JWT + RBAC |
| DELETE | /admin/operation-logs | 清空日志 | JWT + RBAC |

## 权限系统

### 菜单类型

| 类型 | type 值 | 说明 |
|------|---------|------|
| 目录 | 0 | 菜单分组，如"系统管理" |
| 菜单 | 1 | 页面入口，如"用户管理" |
| 按钮 | 2 | 操作权限，如"新增用户"、"删除用户" |

### 权限标识命名规范

```
模块:资源:操作
```

每个菜单下的标准操作：

| 操作 | 标识示例 | 含义 |
|------|----------|------|
| list | system:user:list | 查看列表（菜单级，控制侧边栏显示） |
| query | system:user:query | 查看详情（只读查看单条记录） |
| add | system:user:add | 新增 |
| edit | system:user:edit | 编辑 |
| delete | system:user:delete | 删除 |

### 菜单字段填写规则

| 字段 | 目录 | 菜单 | 按钮 |
|------|------|------|------|
| 路由路径 | `/模块` | `/模块/资源` | 留空 |
| 组件路径 | 留空 | `模块/资源/index` | 留空 |
| 权限标识 | 留空 | `模块:资源:list` | `模块:资源:操作` |
| 图标 | 填写 | 填写 | 留空 |
| 关联API | 无 | 列表接口 | 对应的接口 |

### 权限配置流程（纯后台操作，零代码）

1. **API 管理** — 注册新的 API 接口记录
2. **菜单管理** — 创建菜单和按钮子项，关联对应的 API
3. **角色管理** — 给角色分配菜单/按钮，系统自动生成 Casbin 策略

### Redis 缓存

#### 架构设计

缓存层通过接口抽象，业务代码不直接依赖 go-redis：

```
业务代码 (cache.BlacklistToken / cache.GetUserPermissions ...)
    │
    ▼
cache.Store 接口 (String/Hash/List/Set 全类型支持)
    │
    ▼
cache.RedisStore 实现 (封装 go-redis，自动处理 key 前缀)
```

#### 全局 Key 前缀

通过 `config.yaml` 配置，多项目共用 Redis 时不会冲突：

```yaml
redis:
  key_prefix: "backplane:"
```

实际存储的 key 示例：`backplane:token:blacklist:eyJhb...`、`backplane:perm:user:1`

#### Store 接口支持的数据类型

| 类型 | 方法 |
|------|------|
| String | `Get` `Set` `Del` `Exists` `Incr` `Decr` `Expire` `TTL` `Scan` |
| Hash | `HGet` `HSet` `HDel` `HGetAll` `HExists` `HIncrBy` `HKeys` `HLen` `HMGet` |
| List | `LPush` `RPush` `LPop` `RPop` `LRange` `LLen` `LRem` `LIndex` `LTrim` |
| Set | `SAdd` `SRem` `SMembers` `SIsMember` `SCard` |

#### 内置缓存功能

| 功能 | Key 格式 | TTL | 说明 |
|------|----------|-----|------|
| Token 黑名单 | `token:blacklist:{token}` | 与 JWT 过期时间一致 | 退出登录后 Token 立即失效 |
| 权限缓存 | `perm:user:{userID}` | 10 分钟 | 减少权限查询的数据库压力 |
| 登录限流 | `login:fail:{username}` | 15 分钟 | 5 次失败后锁定 |

#### 业务中使用缓存

```go
// 直接调用封装好的业务方法
cache.BlacklistToken(token, expiration)
cache.IsTokenBlacklisted(token)
cache.SetUserPermissions(userID, perms)

// 或通过 Store 接口使用任意 Redis 操作
store := cache.GetStore()
store.HSet("user:profile:1", "name", "张三", "age", "25")
store.LPush("task:queue", taskJSON)
store.SAdd("online:users", "user_1")
```

#### 自定义实现

测试或切换缓存方案时，实现 `cache.Store` 接口即可：

```go
cache.InitStore(myMemoryStore)  // 替换为内存实现
cache.InitStore(myRedisCluster) // 替换为集群实现
```

## 系统配置（运行时动态配置）

与 `config/config.yaml`（数据库、Redis、JWT 密钥等**基础设施配置**）不同，系统配置是**存数据库、后台可改、改完即时生效、无需重启**的运行期配置——站点名称、是否开放注册、密码最小长度、日志保留天数等。

> 边界：机密（JWT 密钥、数据库密码）仍放 `config.yaml`，不要进数据库配置。

### 高可用设计

```text
启动:   DB(sys_config) ──载入──▶ Redis Hash(backplane:config) 预热
读取:   setting.GetXxx() ─▶ Redis 命中 ─▶ 未命中查 DB 并回填 ─▶ 仍无则用代码内默认值
写入:   后台保存 ─▶ 写 DB(为准) ─▶ 写穿 Redis
多实例: 所有实例共享同一个 Redis 缓存，天然一致，无需 Pub/Sub
降级:   Redis 挂 → 直接查 DB；DB/键缺失 → 用 registry 默认值（读取永不致命）
```

- **DB 为准，Redis 作共享缓存** —— 多实例读同一个 Redis，改一处全局生效，不需要广播。
- **三级兜底** —— Redis → DB → 代码默认值，任意一层抖动都不影响读取。
- **手动刷新** —— 提供「单个刷新」与「一键全部刷新」，用于库被旁路修改、或需强制重建缓存的场景。

### 在业务代码中读取配置

通过 `internal/pkg/setting` 的类型化访问器读取，零样板（内部自动走「Redis → DB → 默认值」）：

```go
import "github.com/kar1hsu/backplane/internal/pkg/setting"

siteName := setting.GetString("site.name")                  // 字符串
allowReg := setting.GetBool("user.allow_register")          // 布尔（"true"/"1" 为真）
minLen   := setting.GetInt("security.password_min_length")  // 整数
retain   := setting.GetInt64("log.operation_retain_days")   // int64
rate     := setting.GetFloat("some.rate")                   // 浮点
```

### 新增一个配置项

在 `internal/pkg/setting/registry.go` 的 `registry` 里加一行即可——它同时是**种子来源**和**兜底默认值来源**：

```go
var registry = []definition{
    // Group 分组(前端按它分 Tab), Key 唯一键, Name 显示名, Type 类型, Value 默认值, IsPublic 是否公开
    {Group: "站点", Key: "site.name", Name: "站点名称", Type: "string", Value: "Backplane Admin", IsPublic: true},
    {Group: "邮件", Key: "mail.smtp_host", Name: "SMTP 主机", Type: "string", Value: ""}, // ← 新增
}
```

启动时 `setting.Init` 会**幂等补齐**缺失的键（不会覆盖管理员改过的值），所以新增配置会自动同步到已有库。

### 配置类型

`Type` 决定前端用什么控件渲染、以及取值如何解析：

| type | 前端控件 | 取值方法 |
|------|----------|----------|
| string | 输入框 | `GetString` |
| int / float | 输入框 | `GetInt` / `GetInt64` / `GetFloat` |
| bool | 开关 | `GetBool` |
| text / json | 多行文本框 | `GetString` |
| select | 下拉（`options` 为 JSON 数组） | `GetString` |

### 数据模型字段（sys_config）

| 字段 | 说明 |
|------|------|
| group | 分组，前端按它分 Tab |
| key | 唯一键，如 `site.name` |
| value | 值（统一以字符串存储） |
| type | 类型，见上表 |
| options | select 选项 / 校验规则（JSON） |
| is_public | 是否免鉴权可读（见公开端点） |
| editable | 是否允许后台编辑 |
| builtin | 内置项（不可删除；registry 种子均为 true） |

### 公开端点（免鉴权）

`is_public=true` 的项可被公开读取，供登录页/前端启动时拿站点名、Logo 等：

```
GET /api/configs/public  →  { "code":0, "data": { "site.name":"...", "site.logo":"...", "site.resource_domain":"..." } }
```

### 后台使用

「系统管理 → 系统配置」：按分组 Tab 展示、按类型渲染控件；**保存**批量提交并自动刷新缓存；每项可**单独刷新缓存**，右上角可**一键刷新全部缓存**；非内置项可删除。完整接口见上文 [API 概览 · 系统配置](#系统配置)。

## 文件上传与静态资源

### 存储与校验

本地文件默认保存在项目运行目录的 `storage/uploads`。应用启动时会自动创建目录，目录内的实际上传文件由 `.gitignore` 排除，不进入 Git。

```yaml
storage:
  directory: storage/uploads
  public_url: /uploads
  max_size: 10 # 单文件上限（MB）
  allowed_types:
    jpg: image/jpeg
    jpeg: image/jpeg
    png: image/png
    gif: image/gif
    webp: image/webp
```

上传组件同时校验扩展名和服务端读取文件头后探测出的真实 MIME，客户端提交的 `Content-Type` 不作为可信依据。SVG 默认不允许，避免同域脚本注入。最终文件使用随机名称，不使用客户端原始文件名。

未传 `folder` 时按应用时区保存到 `Y/m/d` 目录；传入时可使用 `users/avatars` 这类由字母、数字、`_`、`-` 组成的多级目录。绝对路径、空路径段和 `../` 路径会被拒绝。

### 上传接口与资源域名

`POST /api/upload` 使用 `multipart/form-data`：

```bash
curl -X POST http://localhost:8080/api/upload \
  -F "file=@avatar.png" \
  -F "folder=users/avatars"
```

成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "original_name": "avatar.png",
    "file_name": "6f31a8b51c294e0d936487b282e404ed.png",
    "path": "users/avatars/6f31a8b51c294e0d936487b282e404ed.png",
    "url": "/uploads/users/avatars/6f31a8b51c294e0d936487b282e404ed.png",
    "size": 12345,
    "content_type": "image/png",
    "resource_domain": "https://cdn.example.com"
  }
}
```

「系统管理 → 系统配置 → 站点 → 资源域名」对应 `site.resource_domain`。留空表示使用当前站点域名；填写时建议只填协议和域名，不带末尾 `/`，例如 `https://cdn.example.com`。上传接口每次从运行时配置读取该值，后台保存后无需重启。

前端完整资源地址：

```ts
const resourceURL = `${data.resource_domain}${data.url}`
```

### Docker 与宿主机 Nginx

Nginx 不在 Compose 中时，应用容器必须把上传目录绑定到宿主机。`deploy/.env.example` 提供的默认配置为（复制为 `.env` 后生效）：

```dotenv
UPLOAD_DIR=../storage/uploads
```

Compose 将其挂载到容器内的 `/app/storage/uploads`。生产环境也可以填写绝对目录，例如 `/srv/backplane/uploads`，此时宿主机 Nginx 配置为：

```nginx
server {
    listen 80;
    server_name example.com;

    # 应略大于 storage.max_size，为 multipart 边界和表单字段留出空间
    client_max_body_size 11m;

    location ^~ /uploads/ {
        alias /srv/backplane/uploads/;
        autoindex off;
        access_log off;
        add_header Cache-Control "public, max-age=31536000, immutable";
        add_header X-Content-Type-Options "nosniff" always;
    }

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

`alias` 必须指向 `UPLOAD_DIR` 对应的宿主机目录，且末尾 `/` 不能省略。目录建议为 `0755`、文件为 `0644`，不要使用 `chmod 777`。

`/api/upload` 当前属于公开 API，不经过 Admin JWT/RBAC。生产环境应根据业务增加鉴权，并在 Nginx 对该路径配置请求频率限制。

## 系统运行日志（文件日志）

Zap 日志同时写入 stdout 和文件。文件日志由 Lumberjack 管理，默认写入 `logs/app.log`，不是按天主动新建文件，而是达到 `log.max_size` 后轮转为带时间戳的旧文件并 gzip 压缩；`log.max_backups` 控制保留的旧文件数量，`log.max_age` 控制旧文件最长保留天数（`0` 表示不按天数清理旧文件）。

`log.max_age` 只影响运行文件日志，不影响数据库中的操作审计日志；数据库操作日志的留存由系统配置 `log.operation_retain_days` 单独控制。

Docker 部署时，`server` / `worker` / `scheduler` 都把 `/app/logs` 挂到同一个 `app_logs` volume，因此三个进程的文件日志会写到同一个卷里；同时容器 stdout 也会被 Docker 自身日志驱动采集。需要区分进程日志时，建议给不同进程配置不同的日志目录或查看各自容器 stdout。

## 操作日志（审计）

自动把所有写操作（POST/PUT/DELETE/PATCH）记入数据库，后台可检索。与 Zap 文件日志（运维排障）不同，这是**入库、可查询的审计轨迹**，两者并存。

### 工作方式

- **中间件自动采集** —— `middleware.OperationLog()` 挂在管理路由上，位于鉴权与 RBAC 之间（被拒的 403 也会留痕）。
- **复用 sys_api** —— 用「请求方法 + 路由」匹配 `sys_api` 的分组/描述，自动填模块名与操作名，无需额外维护映射。
- **脱敏 + 截断** —— 请求/响应 JSON 中 `password`/`token` 等敏感字段记为 `***`；请求体最多记录 8KB，响应体超过 8KB 时跳过完整内容，仅记录已截断提示。
- **判定成败** —— 解析响应业务码（0 为成功）+ HTTP 状态。
- **best-effort** —— 同步写入，但落库失败只记 Zap、不影响主请求。
- **登录审计** —— 登录（含失败，记下尝试的用户名）、登出并入操作日志（模块「认证」）。

记录字段包含：操作人/角色快照、模块/操作、方法/路由/路径、目标 ID、请求参数、响应参数、HTTP 状态/业务码/成败、错误信息、IP/UA、耗时。

### 留存清理

保留天数由系统配置 `log.operation_retain_days`（默认 30 天）控制；它只影响 `sys_operation_log` 表，不影响 `logs/app.log` 等运行文件日志。每天凌晨 2 点由 Scheduler 投递 `system:cleanup`，Worker 消费后调用 `OperationLogRepo.DeleteBefore(t)` 按时间硬删。配置值 `0` 或负数表示不清理。定时清理需要同时运行 `cmd/scheduler` 和 `cmd/worker`；只运行 Web 服务不会执行清理任务。

## 任务系统（消息队列 + 定时任务）

基于 Asynq + Redis，支持分布式部署，多 Worker 实例自动负载均衡。

### 架构

```
Web 服务 (生产者)       Scheduler (定时投递)      Worker (消费者)
cmd/server/main.go     cmd/scheduler/main.go     cmd/worker/main.go
  │ Client.Enqueue()     │ 按 cron 投递            │ tasks.RegisterHandlers()
  │                      │ (单实例 + Unique 去重)  │ (可多实例，自动负载均衡)
  ▼                      ▼                         ▲
┌─────────────────────────────────────────────────────┐
│                       Redis                         │
│            队列: critical / default / low            │
└─────────────────────────────────────────────────────┘
```

### 启动进程

Web 服务、Scheduler、Worker 是三个独立进程，可分开部署：

```bash
# 终端 1: Web 服务（生产者）
go run cmd/server/main.go

# 终端 2: Scheduler（定时任务投递端）
go run cmd/scheduler/main.go

# 终端 3: Worker（消费者）
go run cmd/worker/main.go
```

**水平扩展与多实例**：

- **Worker 可任意多实例** — 多个 Worker 消费同一 Redis 队列，任务自动负载均衡，是真正的分布式消费。
- **Scheduler 必须单实例** — `asynq.Scheduler` 没有选主机制，N 个实例会让每个 cron 任务被投递 N 次。生产环境只部署一个 Scheduler。作为兜底，给 cron 任务设置 `Unique` TTL（见下），即使误起第二个实例，Redis 也会对重复投递去重。

### 投递任务（生产者）

在任意 Handler / Service 中调用：

```go
// 即时任务
app.TaskMgr.Client.Enqueue("email:send", EmailPayload{To: "user@example.com", Subject: "Welcome"})

// 延迟任务（10 分钟后执行）
app.TaskMgr.Client.EnqueueDelay("email:send", payload, 10*time.Minute)

// 去重任务（1 小时内同样的任务只投递一次）
app.TaskMgr.Client.EnqueueUnique("report:generate", payload, 1*time.Hour)

// 指定队列（高优先级）
app.TaskMgr.Client.EnqueueToQueue("order:notify", payload, "critical")
```

### 定义任务处理器（消费者）

在 `internal/tasks/` 中创建：

```go
// internal/tasks/types.go — 定义任务类型名
const TypeOrderNotify = "order:notify"

// internal/tasks/order.go — 实现处理逻辑
func HandleOrderNotify(ctx context.Context, payload []byte) error {
    var p OrderPayload
    json.Unmarshal(payload, &p)
    // 处理逻辑...
    return nil
}

// internal/tasks/register.go — 注册
func RegisterHandlers(w *task.Worker) {
    w.Handle(TypeOrderNotify, HandleOrderNotify)
}
```

### 定时任务

在 `internal/tasks/register.go` 中注册，由 `cmd/scheduler` 进程加载：

```go
func RegisterCronJobs(s *task.Scheduler) {
    // 每天凌晨 2 点清理（Unique TTL < 触发间隔，多实例下去重）
    s.Register(task.CronTask{Cron: "0 2 * * *", TypeName: TypeCleanup, Unique: 23 * time.Hour})
    // 每 5 分钟执行
    s.Register(task.CronTask{Cron: "@every 5m", TypeName: TypeSyncData, Unique: 4 * time.Minute})
    // 指定队列
    s.Register(task.CronTask{Cron: "0 8 * * 1", TypeName: TypeWeeklyReport, Queue: "low"})
}
```

> `Unique` 字段可选：设为略小于触发间隔的值后，即便有多个 Scheduler 实例同时投递，Redis 也只会让一个任务进入队列。

### 队列优先级

在 `config.yaml` 中配置，排在前面的优先级更高：

```yaml
task:
  concurrency: 10
  queues:
    - critical    # 权重 3（最高）
    - default     # 权重 2
    - low         # 权重 1（最低）
```

## 扩展模块

实现 `Module` 接口即可添加新模块：

```go
type Module interface {
    Name() string
    RegisterRoutes(rg *gin.RouterGroup)
}
```

在 `main.go` 中注册：

```go
router := server.NewRouter(
    backplane.AdminDist,
    admin.New(),
    api.New(),
    yourmodule.New(), // 新模块
)
```

## License

MIT
