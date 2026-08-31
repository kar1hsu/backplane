# Backplane

[![CI](https://github.com/kar1hsu/backplane/actions/workflows/ci.yml/badge.svg?branch=naive-ui)](https://github.com/kar1hsu/backplane/actions/workflows/ci.yml)

简体中文 | [English](README.en-US.md)

Backplane 是一个可以直接用于业务开发的 Go 模块化后台管理框架。后端基于 Gin、GORM、Casbin 和 JWT，管理端使用 Vue 3 + Naive UI，并通过 Go `embed` 打包进同一个二进制文件。

它已经准备好后台项目最常见、也最容易重复造轮子的部分：用户与角色、按钮级 RBAC、权限策略自动同步、运行时系统配置、操作审计、任务队列、文件上传和 Docker 部署。你可以保留这些基础能力，把精力放在自己的业务模块上。

> 当前为 `naive-ui` 分支，管理端已使用 [Naive UI](https://www.naiveui.com/) 重构。

## 界面预览

| 登录 | 控制台 |
| --- | --- |
| ![Naive UI 登录页](docs/images/login-naive-ui.png) | ![Naive UI 控制台](docs/images/admin-naive-ui.png) |

<details>
<summary>查看核心功能页面</summary>

| 用户管理 | 角色管理 |
| --- | --- |
| ![Naive UI 用户管理](docs/images/users-naive-ui.png) | ![Naive UI 角色管理](docs/images/roles-naive-ui.png) |

| 菜单与按钮权限 | 系统配置 |
| --- | --- |
| ![Naive UI 菜单管理](docs/images/menus-naive-ui.png) | ![Naive UI 系统配置](docs/images/settings-naive-ui.png) |

![Naive UI 操作日志](docs/images/operation-logs-naive-ui.png)

</details>

## 核心功能

- **认证与安全**：JWT 登录、Token 黑名单、密码 bcrypt 加密、登录失败限流。
- **RBAC 权限**：Casbin 路由鉴权，支持目录、菜单、按钮三级权限和路径参数匹配。
- **权限自动同步**：角色分配菜单后，根据菜单关联的 API 自动维护 Casbin Policy。
- **系统管理**：用户、角色、菜单、API 和运行时配置统一管理。
- **操作审计**：记录操作人、模块、请求与响应、执行结果、来源 IP 和耗时，支持检索与清理。
- **配置与缓存**：系统配置持久化到数据库，并通过 Redis 缓存实现多实例即时生效。
- **任务系统**：基于 Asynq 的即时、延迟、唯一任务与 Cron 定时任务。
- **文件上传**：本地文件存储、日期目录、随机文件名、大小与 MIME 双重校验。
- **交付与部署**：前端嵌入 Go 二进制，支持 MySQL / PostgreSQL 和 Docker Compose。

## 技术栈

| 后端 | 前端 | 基础设施 |
| --- | --- | --- |
| Go · Gin · GORM · Casbin · JWT | Vue 3 · Naive UI · Vite · Pinia | MySQL / PostgreSQL · Redis · Asynq · Docker |

## 3 分钟运行

### Docker Compose

需要提前安装 Git 和 Docker：

```bash
git clone -b naive-ui git@github.com:kar1hsu/backplane.git
cd backplane/deploy

cp .env.example .env
cp ../config/config.yaml.example config.yaml
docker compose up -d --build
```

启动后访问 [http://localhost:8080](http://localhost:8080)。

默认管理员账号：`admin` / `admin123`。首次启动会自动创建数据表和基础权限数据，生产环境部署前请修改默认密码、数据库密码和 `jwt.secret`。

### 本地开发

需要 Go 1.21+、Node.js 20+、MySQL 8+ 或 PostgreSQL 14+，以及 Redis 6+。

```bash
git clone -b naive-ui git@github.com:kar1hsu/backplane.git
cd backplane

cp config/config.yaml.example config/config.yaml
# 修改 config/config.yaml 中的数据库、Redis 和 JWT 配置

cd web/admin
npm ci
npm run build
cd ../..

go run ./cmd/server
```

前端联调可在 `web/admin` 下运行 `npm run dev`，默认访问 [http://localhost:5173](http://localhost:5173)。

任务消费者和定时调度器按需独立启动：

```bash
go run ./cmd/worker
go run ./cmd/scheduler
```

## 开始写业务

Backplane 按模块注册路由。建议参考 `internal/module/api` 创建自己的业务模块，并保持 Handler、Service、Repository 的职责边界。

一个模块只需要实现 `server.Module`：

```go
type Module interface {
	Name() string
	RegisterRoutes(rg *gin.RouterGroup)
}
```

然后在 `cmd/server/main.go` 注册：

```go
router := server.NewRouter(
	backplane.AdminDist,
	admin.New(),
	api.New(),
	yourmodule.New(),
)
```

推荐的开发顺序：

1. 在 `internal/module/<name>` 中编写路由、Handler 和 Service。
2. 在 `internal/model` 与 `internal/repository` 中补充模型和数据访问逻辑。
3. 将新模型加入迁移，并在需要时补充初始化数据。
4. 在管理端添加页面、路由、菜单和权限标识。
5. 为菜单关联 API，角色授权时会自动同步对应的 Casbin Policy。

## 项目结构

```text
backplane/
├── cmd/                    # server、worker、scheduler 入口
├── config/                 # 配置示例与 Casbin 模型
├── internal/
│   ├── app/                # 数据库、Redis、Casbin、任务等初始化
│   ├── middleware/         # JWT、RBAC、日志、审计、CORS
│   ├── model/              # 数据模型
│   ├── module/             # admin、api 与业务模块
│   ├── pkg/                # cache、setting、task、storage 等公共能力
│   ├── repository/         # 数据访问层
│   └── server/             # 模块注册与静态资源服务
├── web/admin/              # Vue 3 + Naive UI 管理端
├── deploy/                 # Dockerfile 与 Docker Compose
└── embed.go                # 嵌入管理端构建产物
```

## 质量检查

GitHub Actions 会执行以下检查：

```bash
go test ./...
go vet ./...
golangci-lint run
npm run build --prefix web/admin
```

## 致谢

感谢 [Gin](https://gin-gonic.com/)、[GORM](https://gorm.io/)、[Casbin](https://casbin.org/)、[Vue](https://vuejs.org/) 以及 [Naive UI](https://www.naiveui.com/) 等开源项目。本分支的管理端重构建立在 Naive UI 之上。

如果 Backplane 对你有帮助，欢迎在 [GitHub](https://github.com/kar1hsu/backplane) 点一个 Star，也欢迎提交 Issue 和 Pull Request。
