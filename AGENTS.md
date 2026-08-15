# AGENTS.md

## Safety Rules
- Before `git commit`: run `git branch --show-current`. If branch name lacks `feat`/`refactor`/`docs`/`fix`, do NOT commit — ask user for a branch name, create it, commit there.
- Cross-domain guard: if `cmd/webui/` does NOT exist, warn before touching it (belongs to other repo); if it exists, warn before touching `internal/` (belongs to core repo).
- Core changes → `docs/CHANGELOG.md` only; higher versions on top.

## Project Meta
- Module: `github.com/XingfenD/tag_file` (capital `D`). Go 1.24.1. No linter, no CI.
- SQLite via GORM; DB at `.tagfile/tagfile.db`; schema via `AutoMigrate` only — no raw SQL migrations.

## Build & Test
- CLI: `make build` → `tagfile.out`. WebUI: `make build-webui` → `tagfile-webui.out`; run with `go run ./cmd/webui --source <dir> --addr :8088`.
- Test: `make test` (sets up `temp/test_workspace/`, then `go test ./... -v`).
- Never build ad-hoc test binaries — use `go test`, or the Makefile-built `tagfile.out`.

## Core Gotchas (`internal/`)
- `internal/` is not importable externally. Uniqueness (tag names per category, category names globally) is enforced at the service layer.
- Repositories use chainable builders: `repo.List().ByCategory(id).Offset(n).Limit(n).Exec()`, one file per operation.
- Tag tree: `Tag.ParentID` + `ListTagTree()`; search resolves subtrees via `GetDescendantIDs()`. Traversal is in-memory (load all tags once, no per-node DB calls).
- `File.TagCnt` is a redundant counter — maintain it via `IncrementTagCnt`/`DecrementTagCnt` on tag add/remove.
- Two independent tag-value systems: `TagValue` (predefined) vs `TagGroupTag.Value` (per-association).

## WebUI Gotchas (`cmd/webui/`)
- Go 1.22+ routing: `mux.HandleFunc("METHOD /path", handler)`. Service imports aliased, e.g. `categorySvc ".../internal/service/category"`.
- Templates: `{{define}}` cannot be nested (`tagTreeNode` must live outside `content`); inline-edit `cells[]` indices must match table columns; forms carry a `_return` hidden field to preserve query state across POST redirects.
