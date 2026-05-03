# AGENTS.md

## Build & Run
- Build: `make build` or `go build -o tagfile.out ./cmd/cli`
- Run: `make run`
- Test: `make test` (runs `make setup` then integration/unit test binary)
- Install: `make install` (go install ./cmd/cli)
- Web UI: `go run ./cmd/webui` → http://localhost:8088

## Module
- Path: `github.com/XingfenD/tag_file` (note capital `D`)
- Go version: `1.24.1`

## Architecture
- **Entry point**: `cmd/cli/main.go` — creates a `cobra.Command` named `tagfile`, registers subcommands from `cmd/cli/commands/`
- **Layered**: `cmd/cli` → `internal/service/` → `internal/repository/` → `internal/model/`
- `internal/` enforces Go package encapsulation (not importable externally)

## Database
- SQLite via GORM (`gorm.io/gorm` + `gorm.io/driver/sqlite`)
- DB path: `.tagfile/tagfile.db`, hardcoded in `internal/bootstrap/init.go`
- Schema managed exclusively through `AutoMigrate` — no raw SQL migrations
- GORM log level: `Silent` (set in `pkg/storage/sqlite.go`)
- Models: `File`, `FileTag`, `Tag`, `TagCategory`, `Attribute`, `AttributeValue`, `FileAttribute`, `Config`

## Commands
All commands are fully implemented (no stubs).

- `tagfile category {add, remove, list}` — `cmd/cli/commands/category.go`
- `tagfile tag {add, remove, list}` — `cmd/cli/commands/tag.go`
- `tagfile file {scan, import, list, tag, untag, info, search}` — `cmd/cli/commands/file.go`
- `tagfile search tag <tags> --and/--or/--not` — `cmd/cli/commands/search.go`
- `tagfile attribute {add, remove, list}` — `cmd/cli/commands/attribute.go`
- `tagfile attribute value {add, remove, list}` — subcommands of attribute
- `tagfile config {set, get, list, unset}` — `cmd/cli/commands/config.go`
- `tagfile completion {bash,zsh,fish,powershell}` — cobra built-in
- Use `cobra.ExactArgs` and `cobra.MaximumNArgs` for positional arg validation

## Testing
- `make test` depends on `make setup` — creates `temp/test_workspace/` first
- Custom test binary at `cmd/test/` (entry: `main.go`, split into `unit.go`, `integration.go`, `attr_config.go`)
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
