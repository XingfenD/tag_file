# AGENTS.md

## Build & Run
- Build: `make build` or `go build -o tagfile.out ./cmd/cli`
- Run: `make run`
- Test: `make test` (runs `make setup` then integration/unit test binary)
- Install: `make install` (go install ./cmd/cli)

## Module
- Path: `github.com/XingfenD/tag_file` (note capital `D`)
- Go version: `1.24.1`

## Architecture
- **Entry point**: `cmd/cli/main.go` — creates a `cobra.Command` named `tagfile`, registers subcommands from `cmd/cli/commands/{category,tag,file,search,config}/`
- **Layered**: `cmd/cli` → `internal/service/` → `internal/repository/` → `internal/model/`
- `internal/` enforces Go package encapsulation (not importable externally)

## Database
- SQLite via GORM (`gorm.io/gorm` + `gorm.io/driver/sqlite`)
- DB path: `.tagfile/tagfile.db`, hardcoded in `internal/bootstrap/init.go`
- Schema managed exclusively through `AutoMigrate` — no raw SQL migrations
- GORM log level: `Silent` (set in `pkg/storage/sqlite.go`)
- Models: `File`, `FileTag`, `Tag`, `TagCategory`, `Config`

## Commands
All commands are fully implemented (no stubs).

- `tagfile category {add, remove, list}` — `cmd/cli/commands/category/`
- `tagfile tag {add, remove, list}` — `cmd/cli/commands/tag/` (supports `--parent` and `--tree` flags for tree-based tag structure)
- `tagfile file {scan, import, list, tag, untag, info, search}` — `cmd/cli/commands/file/`
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

## Conventions
- Binary named `tagfile.out` (unusual — not just `tagfile`)
- No linter config, no CI, no import aliasing convention
- `internal/bootstrap` initializes DB and calls AutoMigrate on startup
- Each directory gets its own `.tagfile/` containing `tagfile.db` and `last_scan` state file
- File paths stored as relative paths from CWD
- File import copies to CWD (not a subdirectory)
- Category names globally unique, tag names unique per category (enforced in service layer)
- Incremental scan tracks last scan time in `.tagfile/last_scan`, use `file scan --full` to force full scan
