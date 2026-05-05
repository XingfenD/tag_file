# AGENTS.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Safety Rules
- **Branch safety**: If the current branch is NOT a dev branch (name does NOT contain `feat`, `refactor`, `docs`, `fix`, `release`), do NOT commit directly — warn the user first. `release/*` branches are safe to commit.
- **Commit SOP**: Before every `git commit`, run `git branch --show-current`. If the branch is not a dev/release branch, ask the user for a branch name, create it, and commit there.
- **Cross-domain guard**: If `cmd/webui/` does NOT exist, warn before modifying `cmd/webui/` files (they belong in the other repo). If `cmd/webui/` exists, warn before modifying `internal/` files (they belong in the core repo).
- **CHANGELOG**: Core changes → `docs/CHANGELOG.md`. GUI changes → `docs/CHANGELOG_gui.md`. Higher versions on TOP, lower versions below. No Co-Authored-By trailer in commits.

---

## Project Meta
- Module: `github.com/XingfenD/tag_file` (note capital `D`)
- Go: `1.24.1`
- No linter config, no CI
- Changelogs: `docs/CHANGELOG.md` (core), `docs/CHANGELOG_gui.md` (gui), `docs/CHANGELOG_webui.md` (webui)

---

## Core (`internal/`)

### Models
- `File` — indexed file metadata; `TagCnt` is a redundant count maintained by `IncrementTagCnt`
- `Tag` — `ParentID *uint` self-referential FK for tree hierarchy
- `TagCategory` — flat structure (no tree), globally unique names
- `TagGroup` — groups tags on a file (file_id + group_name); each file has a default group
- `TagGroupTag` — joins group + tag with optional value
- `TagValue` — predefined/enum values for a tag
- `Config` — key-value config store

### Service Layer (`internal/service/{file,category,tag,search,config,taggroup,export}/`)
- Uniqueness enforced at service layer (tag names per category, category names globally)
- `IncrementTagCnt` / `DecrementTagCnt` maintain `File.TagCnt` on tag add/remove
- `ListTagTree()` → `[]TagTreeNode{Tag, Children}` for tree rendering
- `GetDescendantIDs()` resolves tag subtree to flat ID list for search

### Repository Layer (`internal/repository/{file,category,tag,config,taggroup}/`)
- **Chainable builder pattern** — `repo.List().ByCategory(id).Offset(n).Limit(n).Exec()`, one file per operation
- `Find()` returns single result, `Search()` does LIKE queries
- Tag tree methods in `tree.go`: `FindRootsByCategory`, `FindChildren`, `FindDescendantIDs`
- `FindChildren` / `FindRootsByCategory` must `Preload("Category")` for tree view

### Database
- SQLite via GORM (`gorm.io/gorm` + `github.com/glebarez/sqlite`)
- DB path: `.tagfile/tagfile.db`, hardcoded in `internal/bootstrap/init.go`
- Schema by `AutoMigrate` only — no raw SQL migrations
- GORM log level: `Silent` (`pkg/storage/sqlite.go`)
- `internal/bootstrap` calls `AutoMigrate` on startup; seeds default `scan_exclude` config

### Key Design Decisions
- **Tag tree**: `ParentID` + `ListTagTree()`; search resolves descendants via `resolveTagIDsWithDescendants`
- **Tag groups**: replaced old `FileTag`; tags scoped to named groups on a file
- **Tag values**: two independent systems — `TagValue` (predefined) vs `TagGroupTag.Value` (per-association)
- **Attributes removed**: no `internal/service/attr/` package
- `internal/` packages are not importable externally

---

## CLI (`cmd/cli/`)

### Build & Run
- Build: `make build` or `go build -o bin/tagfile.exe ./cmd/cli`
- Run: `make run`
- Install: `make install` (go install ./cmd/cli)

### Entrypoint
- `cmd/cli/main.go` — cobra root command
- Subcommands in `cmd/cli/commands/{category,tag,file,search,config}/`
- Shared helpers (`FormatSize`, `Truncate`) in `cmd/cli/commands/util/`

### Commands
- `tagfile category {add, remove, list}`
- `tagfile tag {add, remove, list, value}` — `--parent` for tree, `--tree` for tree view, `value add/remove/list` for predefined values
- `tagfile file {scan, import, list, tag, untag, info, search}` — `tag --value` for tag value
- `tagfile search tag <tags> --and/--or/--not`
- `tagfile config {set, get, list, unset}`
- `tagfile export [output.sql]`
- `tagfile completion {bash,zsh,fish,powershell}`
- Use `cobra.ExactArgs` / `cobra.MaximumNArgs` for positional arg validation

---

## Web UI (`cmd/webui/`)

### Build & Run
- Build: `make build-webui` or `go build -o bin/tagfile-webui.exe ./cmd/webui`
- Run: `go run ./cmd/webui --source <dir> --addr :8088`

### Entrypoint
- `cmd/webui/main.go` — `net/http` + `html/template` with `//go:embed templates/*`
- Templates loaded via `ParseFS`; partials in `templates/partials/` loaded alongside each page

### Routes (`handlers/handlers.go:RegisterRoutes`)
| Category | Routes |
|---|---|
| Categories | `GET /categories`, `POST /categories/{add,remove,update}` |
| Tags | `GET /tags`, `POST /tags/{add,remove,update}`, `POST /tags/values/{add,remove}` |
| API | `GET /api/tags?category_id=X` `{id,name,path}`, `GET /api/tags/values?tag_name=X` `{id,value}` |
| Files | `GET /files` (tree), `POST /files/{scan,refresh,import,tag,untag,rename}` |
| Files | `POST /files/tag/update`, `POST /files/group/{add,remove,rename}` |
| Files | `GET /files/detail/{path...}` (`?partial=tags&page=X` for AJAX), `GET /files/groups/{path...}` |
| Files | `GET /files/raw?id=X`, `GET /files/info?id=X` |
| Search | `GET /search`, `POST /search` |
| Config | `GET /config`, `POST /config/{set,unset,update}` |

### Handler Conventions
- `mux.HandleFunc("METHOD /path", handler)` — Go 1.22+ routing syntax
- Service imports use alias: `categorySvc "github.com/XingfenD/tag_file/internal/service/category"`
- `Redirect()` strips `flash`/`flash_type` from `_return` hidden field via `cleanQuery()` to prevent URL accumulation
- `HandleFileDetail` → `renderTagTable()` for `?partial=tags` AJAX table reload
- Pagination: `Paginate(r, total, pageSize)`, `UpdatePagination(&pg, total, count)`, `PaginationBaseURL(r)`

### Template Conventions
- `FuncMap`: `formatSize`, `iconForType`, `urlEncode`
- `{{define "tagTreeNode"}}` **must** be outside `{{define "content"}}` — Go templates disallow nested defines
- Inline edit in tag table uses `cells[]` indices; adding/removing columns requires updating indices
- Button CSS: `.btn` base, `.btn-primary`, `.btn-danger`, `.btn-default`, `.btn-sm`, `.btn-xs`
- Tag count: `{{if gt .File.TagCnt 0}}<span class="badge tag-cnt">{{.File.TagCnt}}</span>{{end}}`
- Form `_return` hidden field preserves `view`, `category_id`, `q` across POST redirects
- `sharedJS` template loaded in `base.html` via `{{template "sharedJS" .}}`

---

## GUI (`cmd/gui/`)

### Build & Run
- Build: `make build-gui` (dev), `make build-gui-prod` (production)
- Run: `make run-gui` or `./bin/tagfile-gui.exe --source <dir>`
- **Must use `wails build`**, not plain `go build` (Wails requires specific build tags)
- Output: `bin/tagfile-gui.exe`

### Entrypoint
- `cmd/gui/main.go` — Wails v2 app, calls `bootstrap.InitDB()`, embeds `frontend/` via `//go:embed`
- `cmd/gui/app.go` — `App` struct with 40+ Wails bindings, mirrors webui `Server` handlers
- `cmd/gui/app_video.go` — VLC player bindings (play/pause/stop/state)
- `cmd/gui/filetree.go` — File tree building utilities (copied from webui)
- `cmd/gui/player/player.go` — VLC external process launcher, cross-platform paths
- `cmd/gui/wails.json` — Wails project config (required by `wails build`)

### Frontend
- `frontend/index.html` — SPA shell with sidebar nav (6 sections)
- `frontend/js/app.js` — Router, flash messages, navigation
- `frontend/js/api.js` — JS wrappers for all Go Wails bindings
- `frontend/js/components/` — pagination, tagSearch, fileTree, tagTree, flashMsg, videoPlayer
- `frontend/js/pages/` — 8 pages: home, category, tag, file, fileDetail, fileGroups, search, config
- `frontend/css/app.css` — All styles (extracted from webui `base.html`)
- `frontend/wailsjs/` — Auto-generated by `wails build`, **gitignored**

### Data Serialization
- **Wails serializes Go structs using JSON tags** (snake_case: `category_id`, `tag_name`)
- All binding methods MUST convert model slices to `map[string]any` with consistent JS-friendly keys before returning
- Converter functions: `catToMap`, `cfgToMap`, `tagToFlatMap`, `groupToMap`, `tagGroupTagToMap`, `resultToMap`, `fileToMap`
- JS code uses camelCase keys: `id`, `name`, `desc`, `categoryId`, `categoryName`, `tagPath`, `groupID`, etc.

### Key Architecture Decisions
- **File list cache**: 5-second TTL in-memory cache (`App.fileCache`) to avoid repeated `ListFiles(0,0)` calls. Invalidated on mutation (scan, import, tag, untag, rename).
- **Media streaming**: Local HTTP server on random port serves images/audio via URL (no memory load). `GetFileRawData` has 50MB limit.
- **Video player**: VLC launched as external process. HTML5 `<video>` fallback via streaming URL when VLC unavailable.
- **File tree cache**: Filesystem tree cached to `.tagfile/webui_tree.json` with scan pattern hash for invalidation.
- **Wails v2.12**: `Assets` field is `fs.FS` (not `http.FileSystem`). Use `fs.Sub(embedFS, "frontend")`.
- **No npm/build tool**: frontend is plain HTML/CSS/JS, no bundler.

### Makefile Targets
| Target | Output |
|---|---|
| `make build` | `bin/tagfile.exe` |
| `make build-webui` | `bin/tagfile-webui.exe` |
| `make build-gui` | `bin/tagfile-gui.exe` |
| `make build-gui-prod` | `bin/tagfile-gui.exe` (optimized) |
| `make build-linux` | `dist/tagfile-linux` |
| `make clean` | removes `bin/`, `dist/`, `.tagfile/`, `cmd/gui/build/` |

---

## Testing
- `make test` → `make setup` (creates `temp/test_workspace/`) → integration/unit binary
- Custom test binary at `cmd/test/` (`main.go`, `unit.go`, `integration.go`)
- Files in `cmd/test/` must **NOT** end with `_test.go`
- Standard tests: `go test ./internal/service/... -v`
- `scripts/setup_test_workspace.sh` — demo workspace with 9 files, 4 categories, 13 tags
- 30 tests passing
