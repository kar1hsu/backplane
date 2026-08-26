# Backplane — A Modular Go Admin Framework

[![CI](https://github.com/kar1hsu/backplane/actions/workflows/ci.yml/badge.svg)](https://github.com/kar1hsu/backplane/actions/workflows/ci.yml)

English | [简体中文](README.md)

Backplane is a modular admin/back-office framework for Go. It pairs a **Gin + GORM + Casbin + JWT** backend with a **Vue 3 + Element Plus** panel compiled straight into the binary — a single `go build` ships the whole app. Out of the box: **button-level RBAC** with auto-generated Casbin policies, a **DB-backed operation/audit log**, **runtime system config** (DB as source of truth, Redis-cached, instant multi-instance updates), and an **Asynq distributed task queue** with cron.

> **Naive UI redesign branch:** The redesigned admin panel is available on the [`naive-ui`](https://github.com/kar1hsu/backplane/tree/naive-ui) branch, with new login, application layout, and system-management views. Run `git switch naive-ui` to try it locally.

If Backplane is useful to you, please consider giving the [GitHub repository](https://github.com/kar1hsu/backplane) a **Star**. It helps the project keep improving.

## Tech Stack

| Layer | Technology | Purpose |
|-------|------------|---------|
| Web framework | Gin | Routing, middleware, request handling |
| ORM | GORM | MySQL / PostgreSQL, switchable via config |
| Auth | golang-jwt | JWT token issuing & verification |
| Authorization | Casbin | RBAC model with path-parameter matching |
| Config | Viper | YAML / ENV multi-environment config |
| Logging | Zap + Lumberjack | Structured logs with rotation |
| Cache | Redis (go-redis) | Token blacklist, permission cache, login throttling, config cache |
| Task queue | Asynq | Distributed queue + cron, backed by Redis |
| Passwords | bcrypt | Password hashing |
| Frontend | Vue 3 + Element Plus + Vite | Admin panel, embedded via Go embed |
| Deployment | Docker + docker-compose | One-command containerized deploy |

## Features

- **Modular architecture** — extend via the `Module` interface; ships with Admin and API modules
- **JWT auth** — Bearer-token authentication with a logout token blacklist
- **Login protection** — Redis-backed failure counting; lock after 5 failures for 15 minutes
- **RBAC** — Casbin-based role permissions with `keyMatch2` path-parameter matching
- **Button-level permissions** — menus map to APIs for fine-grained list/query/add/edit/delete control
- **Auto policy sync** — assigning menus to a role auto-generates the matching Casbin API policies, zero manual config
- **Permission cache** — Redis caches per-user permission lists, cleared automatically on role change
- **User / Role / Menu / API management** — full CRUD, tree menus (directory / menu / button), DB-driven permission config
- **Operation log** — automatic audit of every write (operator / module / params / result / latency), including failed logins; searchable & purgeable
- **System config** — DB-driven runtime config with a DB-as-source-of-truth + Redis shared cache; changes take effect instantly and stay consistent across instances; typed access from code
- **Message queue** — Asynq + Redis distributed task queue (immediate / delayed / unique tasks)
- **Cron jobs** — Asynq Scheduler with cron syntax in a dedicated process; optional `Unique` dedup against multi-instance double-enqueue
- **Embedded panel** — Element Plus UI compiled into the Go binary via `embed`
- **Multi-database** — switch between MySQL and PostgreSQL via config

## Quick Start

### Requirements

- Go 1.21+
- Node.js 20+ (to build the frontend)
- MySQL 8.0+ or PostgreSQL 14+
- Redis 6+

### 1. Clone

```bash
git clone git@github.com:kar1hsu/backplane.git
cd backplane
```

### 2. Configure

Edit `config/config.yaml` and set your database and Redis connection details.

### 3. Build the frontend

```bash
cd web/admin
npm install
npm run build
cd ../..
```

### 4. Run

```bash
go run cmd/server/main.go
```

Once started:
- Admin panel: http://localhost:8080
- Admin API: http://localhost:8080/admin/*
- Public API: http://localhost:8080/api/*

### 5. Default account

| Username | Password | Role |
|----------|----------|------|
| admin | admin123 | Super Admin |

> Change the default password immediately after first login.

### 6. Frontend dev mode

```bash
cd web/admin
npm run dev
```

Vite dev server runs at `http://localhost:5173` and proxies API calls to the backend at `http://localhost:8080`.

## Docker

```bash
cd deploy
cp .env.example .env          # set DB/Redis passwords and timezone
docker compose up -d
```

Notes:
- `deploy/config.yaml` is already wired for the compose network (`host: mysql` / `host: redis`) and is mounted into the app containers — edit app settings there, **not** in the root `config/config.yaml`.
- `.env` configures the MySQL/Redis containers. If you change `MYSQL_ROOT_PASSWORD` / `REDIS_PASSWORD`, set the matching `database.password` / `redis.password` in `deploy/config.yaml` too — the app reads credentials from the mounted config, not from those env vars.

## Project Structure

```
backplane/
├── cmd/
│   ├── server/main.go          # Web server entrypoint (producer)
│   ├── worker/main.go          # Worker entrypoint (consumer, multi-instance)
│   └── scheduler/main.go       # Scheduler entrypoint (cron dispatch, single instance)
├── config/
│   ├── config.yaml             # Application config
│   └── rbac_model.conf         # Casbin RBAC model
├── internal/
│   ├── app/                    # Bootstrap (Config/Logger/DB/Redis/Casbin/Task, AutoMigrate/Seed)
│   ├── middleware/             # Middleware (JWT/Casbin/CORS/Logger/OperationLog)
│   ├── model/                  # Data models (User/Role/Menu/API/Config/OperationLog)
│   ├── server/                 # HTTP server, route registration, static files
│   ├── repository/             # Data-access layer (generic BaseRepo[T] + QueryOptions)
│   ├── tasks/                  # Task definitions & registration (handlers + cron jobs)
│   ├── module/
│   │   ├── admin/              # Admin module
│   │   │   ├── handler/        # Request handlers (Auth/User/Role/Menu/API/Config/OperationLog)
│   │   │   ├── service/        # Business logic
│   │   │   └── router.go       # Admin route registration
│   │   └── api/                # Public API module
│   └── pkg/                    # Internal shared packages
│       ├── jwt/                # JWT issue/parse
│       ├── cache/              # Cache layer (Store interface / RedisStore / domain caches)
│       ├── setting/            # Runtime system config (typed accessor + Redis cache + defaults registry)
│       ├── task/               # Task system (Client/Worker/Scheduler/Manager)
│       ├── response/           # Unified response envelope
│       ├── errcode/            # Error codes
│       └── utils/              # Helpers (password hashing / pagination)
├── web/admin/                  # Vue 3 frontend project
├── embed.go                    # Go embed for the frontend
└── deploy/                     # Docker deployment files
```

## Route Layers

| Layer | Middleware | Description | Example |
|-------|-----------|-------------|---------|
| Public | none | No login required | `POST /admin/login` |
| Authenticated | JWT | Any logged-in user (profile, dropdowns) | `GET /admin/profile`, `/permissions`, `/roles/all` |
| Protected | JWT + Casbin | RBAC-authorized management actions | `GET /admin/users`, `POST /admin/users` |

## API Overview

### Auth

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| POST | /admin/login | Login (with failure throttling) | none |
| POST | /admin/logout | Logout (token blacklisted) | JWT |
| GET | /admin/profile | Current user info | JWT |
| GET | /admin/permissions | Current user permission codes (cached) | JWT |

### Users

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | /admin/users | User list | JWT + RBAC |
| POST | /admin/users | Create user | JWT + RBAC |
| GET | /admin/users/:id | User detail | JWT + RBAC |
| PUT | /admin/users/:id | Update user | JWT + RBAC |
| DELETE | /admin/users/:id | Delete user | JWT + RBAC |

### Roles

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | /admin/roles | Role list (paginated) | JWT + RBAC |
| GET | /admin/roles/all | All roles (dropdown) | JWT |
| POST | /admin/roles | Create role | JWT + RBAC |
| GET | /admin/roles/:id | Role detail | JWT + RBAC |
| PUT | /admin/roles/:id | Update role | JWT + RBAC |
| DELETE | /admin/roles/:id | Delete role | JWT + RBAC |
| PUT | /admin/roles/:id/menus | Assign menus (auto-syncs Casbin policy) | JWT + RBAC |

### Menus

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | /admin/menus/tree | Full menu tree | JWT |
| GET | /admin/menus/user | Current user's menu tree | JWT |
| POST | /admin/menus | Create menu (can link APIs) | JWT + RBAC |
| GET | /admin/menus/:id | Menu detail (with linked APIs) | JWT + RBAC |
| PUT | /admin/menus/:id | Update menu | JWT + RBAC |
| DELETE | /admin/menus/:id | Delete menu | JWT + RBAC |

### APIs

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | /admin/apis/all | All APIs (dropdown) | JWT |
| GET | /admin/apis | API list (paginated) | JWT + RBAC |
| POST | /admin/apis | Create API | JWT + RBAC |
| PUT | /admin/apis/:id | Update API | JWT + RBAC |
| DELETE | /admin/apis/:id | Delete API | JWT + RBAC |

### System Config

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | /admin/configs | Config list (optional `?group=`) | JWT + RBAC |
| POST | /admin/configs | Create a custom config | JWT + RBAC |
| PUT | /admin/configs | Batch-save values `{items:[{key,value}]}` | JWT + RBAC |
| DELETE | /admin/configs/:id | Delete (built-ins protected) | JWT + RBAC |
| POST | /admin/configs/refresh | Refresh cache (`?key=` for one, otherwise all) | JWT + RBAC |
| GET | /api/configs/public | Public config key→value (no auth, for the login page / app bootstrap) | none |

### File Upload

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| POST | /api/upload | Upload a file (multipart: required `file`, optional `folder`); returns `resource_domain` and file `url` for client-side composition | none |

### Operation Log

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | /admin/operation-logs | Log list (filter by operator / module / result / time range / keyword) | JWT + RBAC |
| GET | /admin/operation-logs/:id | Log detail | JWT + RBAC |
| DELETE | /admin/operation-logs/:id | Delete one | JWT + RBAC |
| DELETE | /admin/operation-logs | Clear all | JWT + RBAC |

## Permission System

### Menu types

| Type | `type` value | Meaning |
|------|--------------|---------|
| Directory | 0 | A group, e.g. "System" |
| Menu | 1 | A page, e.g. "Users" |
| Button | 2 | An action permission, e.g. "Add User" / "Delete User" |

### Permission code convention

```
module:resource:action
```

Standard actions per menu:

| Action | Example | Meaning |
|--------|---------|---------|
| list | system:user:list | View list (menu-level, controls sidebar visibility) |
| query | system:user:query | View detail (read a single record) |
| add | system:user:add | Create |
| edit | system:user:edit | Update |
| delete | system:user:delete | Delete |

### Menu field rules

| Field | Directory | Menu | Button |
|-------|-----------|------|--------|
| Route path | `/module` | `/module/resource` | empty |
| Component path | empty | `module/resource/index` | empty |
| Permission code | empty | `module:resource:list` | `module:resource:action` |
| Icon | yes | yes | empty |
| Linked APIs | none | list endpoint | the matching endpoint(s) |

### Permission workflow (pure admin UI, no code)

1. **API management** — register the API endpoint records
2. **Menu management** — create menus & buttons, linking the matching APIs
3. **Role management** — assign menus/buttons to a role; Casbin policies are generated automatically

### Redis Cache

#### Design

The cache is abstracted behind an interface so business code never depends on go-redis directly:

```
Business code (cache.BlacklistToken / cache.GetUserPermissions ...)
    │
    ▼
cache.Store interface (String/Hash/List/Set)
    │
    ▼
cache.RedisStore (wraps go-redis, handles the key prefix)
```

#### Global key prefix

Configured in `config.yaml` so multiple projects can share one Redis instance:

```yaml
redis:
  key_prefix: "backplane:"
```

Stored keys look like `backplane:token:blacklist:eyJhb...`, `backplane:perm:user:1`.

#### Built-in caches

| Feature | Key format | TTL | Notes |
|---------|-----------|-----|-------|
| Token blacklist | `token:blacklist:{token}` | matches JWT expiry | invalidates token on logout |
| Permission cache | `perm:user:{userID}` | 10 min | reduces permission queries |
| Login throttle | `login:fail:{username}` | 15 min | lock after 5 failures |
| System config | `config` (hash) | none | shared runtime-config cache |

#### Using the cache

```go
// Domain helpers
cache.BlacklistToken(token, expiration)
cache.IsTokenBlacklisted(token)
cache.SetUserPermissions(userID, perms)

// Or any Redis op via the Store interface
store := cache.GetStore()
store.HSet("user:profile:1", "name", "Alice", "age", "25")
store.LPush("task:queue", taskJSON)
```

Swap the backend (tests, in-memory, cluster) by implementing `cache.Store` and calling `cache.InitStore(...)`.

## System Config (Runtime Dynamic Config)

Unlike `config/config.yaml` (**infrastructure** config: database, Redis, JWT secret), system config is **stored in the DB, editable from the admin UI, and applied instantly without a restart** — things like site name, whether self-registration is open, minimum password length, log retention days, etc.

> Boundary: keep secrets (JWT secret, DB password) in `config.yaml` — do **not** move them into DB config.

### High availability

```text
Startup:  DB(sys_config) ──load──▶ Redis hash(backplane:config) warm-up
Read:     setting.GetXxx() ─▶ Redis hit ─▶ miss → query DB & backfill ─▶ still missing → compiled-in default
Write:    admin save ─▶ write DB (source of truth) ─▶ write-through Redis
Cluster:  all instances share one Redis cache → consistent, no pub/sub needed
Degrade:  Redis down → read DB directly;  DB/key missing → registry default (a read never brings the app down)
```

- **DB is the source of truth, Redis is a shared cache** — every instance reads the same Redis, so a change anywhere is global; no broadcast required.
- **Three-tier fallback** — Redis → DB → compiled-in default; any layer can hiccup without breaking reads.
- **Manual refresh** — refresh a single key or refresh everything, for out-of-band DB edits or to force a cache rebuild.

### Reading config in code

Use the typed accessors in `internal/pkg/setting` — zero boilerplate (the Redis → DB → default fallback is handled internally):

```go
import "github.com/kar1hsu/backplane/internal/pkg/setting"

siteName := setting.GetString("site.name")                  // string
allowReg := setting.GetBool("user.allow_register")          // bool ("true"/"1" → true)
minLen   := setting.GetInt("security.password_min_length")  // int
retain   := setting.GetInt64("log.operation_retain_days")   // int64
rate     := setting.GetFloat("some.rate")                   // float64
```

### Adding a config key

Add one line to `registry` in `internal/pkg/setting/registry.go`. It is both the **seed source** and the **fallback-default source**:

```go
var registry = []definition{
    // Group (frontend tab), Key (unique), Name (label), Type, Value (default), IsPublic
    {Group: "Site", Key: "site.name", Name: "Site Name", Type: "string", Value: "Backplane Admin", IsPublic: true},
    {Group: "Mail", Key: "mail.smtp_host", Name: "SMTP Host", Type: "string", Value: ""}, // ← new
}
```

On startup, `setting.Init` **idempotently** inserts missing keys (without overwriting values an admin has changed), so new keys propagate to existing databases automatically.

### Config types

`type` drives both the UI control and how the value is parsed:

| type | UI control | Accessor |
|------|-----------|----------|
| string | input | `GetString` |
| int / float | input | `GetInt` / `GetInt64` / `GetFloat` |
| bool | switch | `GetBool` |
| text / json | textarea | `GetString` |
| select | dropdown (`options` is a JSON array) | `GetString` |

### Model fields (sys_config)

| Field | Description |
|-------|-------------|
| group | Group; the frontend renders one tab per group |
| key | Unique key, e.g. `site.name` |
| value | Value (always stored as a string) |
| type | Type (see table above) |
| options | Select options / validation rules (JSON) |
| is_public | Readable without auth (see public endpoint) |
| editable | Whether the admin UI may edit it |
| builtin | Built-in (cannot be deleted; registry-seeded ones are `true`) |

### Public endpoint (no auth)

Keys with `is_public=true` can be read without authentication — handy for the login page / app bootstrap (site name, logo, …):

```
GET /api/configs/public  →  { "code":0, "data": { "site.name":"...", "site.logo":"...", "site.resource_domain":"..." } }
```

### Admin UI

Under **System → System Config**: grouped into tabs, each value rendered by its type. **Save** submits changes in a batch and refreshes the cache; each row can be **refreshed individually**, and the top-right button **refreshes the whole cache**. Non-built-in entries can be deleted.

## File Uploads and Static Assets

### Storage and validation

Local files are stored under `storage/uploads` relative to the application working directory. The directory is created automatically at startup, and uploaded files are excluded from Git by the directory's `.gitignore`.

```yaml
storage:
  directory: storage/uploads
  public_url: /uploads
  max_size: 10 # per-file limit in MB
  allowed_types:
    jpg: image/jpeg
    jpeg: image/jpeg
    png: image/png
    gif: image/gif
    webp: image/webp
```

The uploader validates both the extension and the MIME detected from the file header; the client-provided `Content-Type` is not trusted. SVG is disabled by default to avoid same-origin script injection. Stored files use random names rather than client-provided filenames.

Without `folder`, files are stored under a `Y/m/d` path in the application timezone. A custom folder may contain multiple segments such as `users/avatars`; each segment may contain letters, digits, `_`, and `-`. Absolute paths, empty segments, and `../` traversal are rejected.

### Upload API and resource domain

`POST /api/upload` accepts `multipart/form-data`:

```bash
curl -X POST http://localhost:8080/api/upload \
  -F "file=@avatar.png" \
  -F "folder=users/avatars"
```

Successful response:

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

**System → System Config → Site → Resource Domain** maps to `site.resource_domain`. Leave it empty to use the current site origin. When set, use only the scheme and host without a trailing slash, for example `https://cdn.example.com`. The upload endpoint reads this runtime setting for every response, so admin changes take effect without a restart.

Build the complete asset URL on the client:

```ts
const resourceURL = `${data.resource_domain}${data.url}`
```

### Docker and host Nginx

When Nginx runs on the host rather than in Compose, bind the application upload directory to the host. `deploy/.env.example` provides this default (copy it to `.env` for deployment):

```dotenv
UPLOAD_DIR=../storage/uploads
```

Compose mounts it at `/app/storage/uploads` inside the application container. Production may use an absolute path such as `/srv/backplane/uploads`; the host Nginx configuration then becomes:

```nginx
server {
    listen 80;
    server_name example.com;

    # Keep this slightly above storage.max_size for multipart overhead
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

The `alias` must point to the host directory represented by `UPLOAD_DIR`, and its trailing `/` is required. Recommended permissions are `0755` for directories and `0644` for files; do not use `chmod 777`.

`/api/upload` is currently a public API and does not pass through Admin JWT/RBAC. In production, add business authentication as appropriate and rate-limit this path at Nginx.

## System Runtime Logs (File Logs)

Zap logs are written to both stdout and a file. File logs are managed by Lumberjack and default to `logs/app.log`. They are not split by day proactively; instead, the active file is rotated when it reaches `log.max_size`, old files get timestamped names and gzip compression, `log.max_backups` limits how many old files are kept, and `log.max_age` limits how many days old files are retained (`0` disables age-based cleanup).

`log.max_age` only affects runtime file logs. It does not affect the database-backed operation audit log; DB operation-log retention is controlled separately by the system config key `log.operation_retain_days`.

In Docker deployments, `server`, `worker`, and `scheduler` all mount `/app/logs` to the same `app_logs` volume, so their file logs share that volume. Their stdout is also captured by Docker's logging driver. If you need separate process log files, configure different log directories per process or inspect each container's stdout.

## Operation Log (Audit)

Every write (POST/PUT/DELETE/PATCH) is recorded to the database and searchable from the admin UI. This is a **queryable audit trail in the DB**, distinct from the Zap file logs used for ops/debugging — the two coexist.

### How it works

- **Auto-captured by middleware** — `middleware.OperationLog()` sits between auth and RBAC, so even denied (403) attempts leave a trace.
- **Reuses sys_api** — matches `method + route` against `sys_api` to fill the module/action names automatically; no extra mapping to maintain.
- **Redacts & truncates** — sensitive JSON fields in requests/responses (`password`, `token`, …) are stored as `***`; request bodies are capped at 8 KB, and response bodies over 8 KB store a truncation notice instead of the full payload.
- **Success detection** — parses the response business code (0 = success) plus the HTTP status.
- **Best-effort** — written synchronously, but a logging failure only goes to Zap and never affects the main request.
- **Login audit** — login (including failures, with the attempted username) and logout are folded into the operation log under the "Auth" module.

Recorded fields include: operator & role snapshot, module/action, method/route/path, target ID, request params, response params, HTTP status / business code / success, error message, IP/UA, and latency.

### Retention

Retention days come from the `log.operation_retain_days` config (default 30). It only affects the `sys_operation_log` table, not runtime file logs such as `logs/app.log`. Every day at 02:00, the Scheduler enqueues `system:cleanup`; a Worker consumes it and calls `OperationLogRepo.DeleteBefore(t)` for a hard delete by time. Values `0` or below disable cleanup. Scheduled cleanup requires both `cmd/scheduler` and `cmd/worker` to be running; the Web server alone does not execute cleanup jobs.

## Task System (Queue + Cron)

Built on Asynq + Redis; supports distributed deployment with automatic load balancing across multiple Worker instances.

### Architecture

```
Web server (producer)   Scheduler (cron dispatch)   Worker (consumer)
cmd/server/main.go      cmd/scheduler/main.go        cmd/worker/main.go
  │ Client.Enqueue()      │ dispatch by cron          │ tasks.RegisterHandlers()
  │                       │ (single instance + Unique) │ (multi-instance, balanced)
  ▼                       ▼                            ▲
┌──────────────────────────────────────────────────────┐
│                        Redis                          │
│            queues: critical / default / low           │
└──────────────────────────────────────────────────────┘
```

### Processes

The web server, Scheduler, and Worker are three independent processes that can be deployed separately:

```bash
# Terminal 1: web server (producer)
go run cmd/server/main.go

# Terminal 2: Scheduler (cron dispatcher)
go run cmd/scheduler/main.go

# Terminal 3: Worker (consumer)
go run cmd/worker/main.go
```

**Scaling & multiple instances**:

- **Workers scale freely** — multiple Workers consume the same Redis queues with automatic load balancing; true distributed consumption.
- **Scheduler must be a single instance** — `asynq.Scheduler` has no leader election, so N instances dispatch each cron job N times. Run exactly one Scheduler in production. As a safety net, give cron jobs a `Unique` TTL (below) so Redis dedups duplicate dispatches even if a second instance is started by mistake.

### Enqueue tasks (producer)

From any handler / service:

```go
// immediate
app.TaskMgr.Client.Enqueue("email:send", EmailPayload{To: "user@example.com", Subject: "Welcome"})

// delayed (run in 10 minutes)
app.TaskMgr.Client.EnqueueDelay("email:send", payload, 10*time.Minute)

// unique (dedup the same task within 1 hour)
app.TaskMgr.Client.EnqueueUnique("report:generate", payload, 1*time.Hour)

// to a specific (high-priority) queue
app.TaskMgr.Client.EnqueueToQueue("order:notify", payload, "critical")
```

### Define a handler (consumer)

Under `internal/tasks/`:

```go
// internal/tasks/types.go — the task type name
const TypeOrderNotify = "order:notify"

// internal/tasks/order.go — the handler
func HandleOrderNotify(ctx context.Context, payload []byte) error {
    var p OrderPayload
    json.Unmarshal(payload, &p)
    // ...
    return nil
}

// internal/tasks/register.go — register it
func RegisterHandlers(w *task.Worker) {
    w.Handle(TypeOrderNotify, HandleOrderNotify)
}
```

### Cron jobs

Registered in `internal/tasks/register.go` and loaded by the `cmd/scheduler` process:

```go
func RegisterCronJobs(s *task.Scheduler) {
    // daily at 02:00 (Unique TTL < interval → dedup across instances)
    s.Register(task.CronTask{Cron: "0 2 * * *", TypeName: TypeCleanup, Unique: 23 * time.Hour})
    // every 5 minutes
    s.Register(task.CronTask{Cron: "@every 5m", TypeName: TypeSyncData, Unique: 4 * time.Minute})
    // a specific queue
    s.Register(task.CronTask{Cron: "0 8 * * 1", TypeName: TypeWeeklyReport, Queue: "low"})
}
```

> `Unique` is optional: set it slightly below the trigger interval and Redis will let only one task enter the queue even if multiple Schedulers dispatch at once.

### Queue priority

Configured in `config.yaml`; earlier means higher priority:

```yaml
task:
  concurrency: 10
  queues:
    - critical    # weight 3 (highest)
    - default     # weight 2
    - low         # weight 1 (lowest)
```

## Extending Modules

Implement the `Module` interface to add a new module:

```go
type Module interface {
    Name() string
    RegisterRoutes(rg *gin.RouterGroup)
}
```

Register it in `main.go`:

```go
router := server.NewRouter(
    backplane.AdminDist,
    admin.New(),
    api.New(),
    yourmodule.New(), // new module
)
```

## License

MIT
