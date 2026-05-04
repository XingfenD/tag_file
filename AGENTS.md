# AGENTS.md

## Build & Run
- Build: `make build` or `go build -o tagfile.out ./cmd/cli`
- Build web UI: `make build-webui` or `go build -o tagfile-webui.out ./cmd/webui`
- Run: `make run`
- Test: `make test` (runs `make setup` then integration/unit test binary)
- Install: `make install` (go install ./cmd/cli)
- Web UI: `go run ./cmd/webui --source <dir> --addr :8088`

## Module
- Path: `github.com/XingfenD/tag_file` (note capital `D`)
- Go version: `1.24.1`

## Architecture
- **CLI entry**: `cmd/cli/main.go` — cobra root, subcommands in `cmd/cli/commands/{category,tag,file,search,config}/`
- **Web UI entry**: `cmd/webui/main.go` — `net/http` + `html/template` with `//go:embed templates/*`
- **Layered**: `cmd/...` → `internal/service/` → `internal/repository/` → `internal/model/`
- **Service sub-packages**: `internal/service/{file,category,tag,search,config}/`
- **Repository sub-packages**: `internal/repository/{file,category,tag,config}/`
- `internal/` enforces Go package encapsulation (not importable externally)

## Database
- SQLite via GORM (`gorm.io/gorm` + `gorm.io/driver/sqlite`)
- DB path: `.tagfile/tagfile.db`, hardcoded in `internal/bootstrap/init.go`
- Schema managed exclusively through `AutoMigrate` — no raw SQL migrations
- GORM log level: `Silent` (set in `pkg/storage/sqlite.go`)
- Models: `File`, `FileTag`, `Tag`, `TagCategory`, `TagValue`, `Config`
  - `Tag` has `ParentID *uint` self-referential FK for tree hierarchy
  - `TagValue` holds predefined/enum values for a tag (unique index on `tag_id,value`)
  - `FileTag.TagValue` stores the actual tag value on file associations

## Key Design Decisions
- **Tag tree**: Tags support parent-child via `ParentID`; `ListTagTree()` returns `[]TagTreeNode{Tag, Children}`
- **Tag values**: Two systems exist:
  1. `TagValue` in `tag_values` table — predefined/allowed values for a tag (managed via `tagfile tag value add/remove/list`)
  2. `FileTag.TagValue` in `file_tags` table — arbitrary value stored per file-tag association (set via `tagfile file tag --value` or web UI)
  - These two are independent (no FK validation between them)
- **Category**: Flat structure (no tree). Global uniqueness on category names.
- **Tag uniqueness**: Tag names unique per category (enforced in service layer)
- **Search**: `SearchByTags` supports `--and/--or/--not` logic; descendant resolution via `resolveTagIDsWithDescendants`
- **Attributes removed**: No `internal/service/attr/` package exists

## Web UI Endpoints
All routes registered in `cmd/webui/handlers/handlers.go:RegisterRoutes`:
- `GET /categories`, `POST /categories/{add,remove,update}`
- `GET /tags`, `POST /tags/{add,remove,update}`, `POST /tags/values/{add,remove}`
- `GET /api/tags?category_id=X` (JSON: tags for parent dropdown)
- `GET /api/tags/values?tag_name=X` (JSON: predefined values for a tag)
- `GET /files` (tree view), `POST /files/{scan,refresh,import,tag,untag,rename}`
- `GET /files/detail/{path...}`, `GET /files/raw?id=X`, `GET /files/info?id=X`
- `GET /search`, `POST /search`
- `GET /config`, `POST /config/{set,unset,update}`

## Tag Value API (Web UI)
- `GET /api/tags/values?tag_name=X` → `[{id, value}]` — used to populate value dropdowns
- `POST /tags/values/add` (form: `tag_name`, `value`) — add predefined value
- `POST /tags/values/remove` (form: `tag_name`, `value`) — remove predefined value
- File detail page: tag form shows value dropdown if tag has predefined values, else text input
- Values displayed as `TagName:Value` in file detail tags list

## Template Notes
- `cmd/webui/templates/tag.html` — flat/table + tree toggle; inline edit via JS fetch()
- `cmd/webui/templates/file_detail.html` — tag:value display, value input with API-driven dropdown
- `FuncMap` in `handlers/handlers.go`: `formatSize`, `iconForType` (emoji icons by MIME)
- `{{define "tagTreeNode"}}` is a recursive sub-template — must be outside `{{define "content"}}`
- Inline edit in tag table uses `cells[]` indices; adding/removing columns requires updating indices

## Commands
- `tagfile category {add, remove, list}` — `cmd/cli/commands/category/`
- `tagfile tag {add, remove, list, value}` — `cmd/cli/commands/tag/` (supports `--parent`, `--tree`; `value` subcommand for predefined values)
- `tagfile file {scan, import, list, tag, untag, info, search}` — `cmd/cli/commands/file/` (`tag --value` for tag value)
- `tagfile search tag <tags> --and/--or/--not` — `cmd/cli/commands/search/`
- `tagfile config {set, get, list, unset}` — `cmd/cli/commands/config/`
- `tagfile completion {bash,zsh,fish,powershell}` — cobra built-in
- Use `cobra.ExactArgs` and `cobra.MaximumNArgs` for positional arg validation
- Shared helpers (`FormatSize`, `Truncate`) in `cmd/cli/commands/util/`

## Testing
- `make test` depends on `make setup` — creates `temp/test_workspace/` first
- Custom test binary at `cmd/test/` (entry: `main.go`, split into `unit.go`, `integration.go`)
- Files in `cmd/test/` must NOT end with `_test.go` (Go would treat them as test-only, invisible to `go build`)
- Standard Go tests exist at `internal/service/*_test.go` — run with `go test ./internal/service/... -v`
- `scripts/setup_test_workspace.sh` creates a demo workspace in `temp/test_workspace/` with 9 files, 4 categories, 13 tags
- 30 tests currently passing (unit + integration)

## Conventions
- Binary named `tagfile.out` (CLI), `tagfile-webui.out` (Web UI)
- No linter config, no CI, no import aliasing convention
- `internal/bootstrap` initializes DB and calls AutoMigrate on startup
- Each directory gets its own `.tagfile/` containing `tagfile.db` and `last_scan` state file
- File paths stored as relative paths from CWD
- File import copies to CWD (not a subdirectory)
- Incremental scan tracks last scan time in `.tagfile/last_scan`, use `file scan --full` to force full scan
- `mux.HandleFunc("METHOD /path", handler)` — Go 1.22+ routing syntax with method prefix
- Service/repository imports use alias pattern: `categorySvc "github.com/XingfenD/tag_file/internal/service/category"`
