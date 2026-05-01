# AGENTS.md

## Build & Run
- Build: `make build` or `go build -o tagfile.out ./cmd/cli`
- Run: `make run`
- Test: `make test` (runs `go test ./... -v`)
- Install: `make install` (go install ./cmd/cli)

## Module
- Path: `github.com/XingfenD/tag_file` (note capital `D`)
- Go version: `1.24.1`

## Architecture
- **Entry point**: `cmd/cli/main.go` — creates a `cobra.Command` named `tagfile`, registers subcommands from `cmd/cli/commands/`
- **Layered (partially stubbed)**: `cmd/cli` → `internal/service/` → `internal/repository/` → `internal/model/`
  - `internal/service/` and `internal/repository/` are currently **empty stubs** — business logic should go here
  - `internal/model/` contains GORM model definitions with `TableName()` methods
- `internal/` enforces Go package encapsulation (not importable externally)

## Database
- SQLite via GORM (`gorm.io/gorm` + `gorm.io/driver/sqlite`)
- DB path: `".tag_file.db"`, hardcoded in `internal/bootstrap/init.go`
- Schema managed exclusively through `AutoMigrate` — no raw SQL migrations
- Models: `File`, `FileTag`, `Tag`, `TagCategory`, `Attribute`, `AttributeValue`, `FileAttribute`, `Config`

## Commands
- `tagfile category {add, remove, list}` — implemented in `cmd/cli/commands/category.go`
- `tagfile tag {add, remove, list}` — implemented in `cmd/cli/commands/tag.go`
- `tagfile file {scan, import, list, tag, untag, info, search}` — implemented in `cmd/cli/commands/file.go`
- `tagfile search tag <tags> --and/--or/--not` — implemented in `cmd/cli/commands/search.go`
- Use `cobra.ExactArgs` and `cobra.MinimumNArgs` for positional arg validation

## Conventions
- Binary named `tagfile.out` (unusual — not just `tagfile`)
- No linter config, no CI, no tests exist yet
- No import aliasing convention
- `internal/bootstrap` initializes DB and calls AutoMigrate on startup

## Project Plan

### Goal
跨平台文件整理与检索工具，对指定目录内的文件进行可配置的分类打标，支持快速检索。

### Phase 1: 分类项与标签管理 (当前)
- [x] `internal/repository/category_repo.go` — TagCategory CRUD
- [x] `internal/repository/tag_repo.go` — Tag CRUD
- [ ] `internal/repository/file_repo.go` — File CRUD
- [x] `internal/service/category_service.go` — 分类项业务逻辑（唯一性校验）
- [x] `internal/service/tag_service.go` — 标签业务逻辑（同分类下标签名唯一）
- [x] `cmd/cli/commands/category.go` — CLI 命令
- [x] 重写 `cmd/cli/commands/tag.go` — 替换 stub 为真实实现
- [x] 更新 `cmd/cli/main.go` — 注册 CategoryCmd

CLI commands:
- `tagfile category add <名称>` — 新增分类项
- `tagfile category remove <名称>` — 删除分类项
- `tagfile category list` — 列出所有分类项
- `tagfile tag add <分类项> <标签名>` — 在某分类下新增标签
- `tagfile tag remove <标签名>` — 删除标签
- `tagfile tag list [分类项]` — 列出标签

### Phase 2: 文件扫描与打标
- [ ] `internal/service/file_service.go` — 文件扫描、索引、导入
- [ ] `internal/service/search_service.go` — 搜索逻辑
- [ ] `cmd/cli/commands/file.go` — file scan/import/list 命令
- [ ] 重写 `cmd/cli/commands/search.go` — 真实搜索实现

CLI commands:
- `tagfile file scan [目录]` — 扫描目录注册文件
- `tagfile file tag <文件> <标签>` — 给文件打标
- `tagfile file untag <文件> <标签>` — 移除标签
- `tagfile file list` — 列出已注册文件
- `tagfile file import <外部路径>` — 从外部引入文件
- `tagfile search <条件>` — 按标签搜索

### Design Decisions
- DB: 每个目录独立的 `.tag_file.db`（相对路径，天然支持）
- 文件: 只索引不移动（外部导入时可移动到统一 `import_dir`）
- 分类项名称全局唯一，标签名在同分类下唯一（service 层校验）
- 文件路径存储相对路径，以当前目录为根
