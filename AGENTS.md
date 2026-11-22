# AGENTS.md

## Safety Rules
- Dev branch naming: `{feat|fix|docs|chore}/{branch-name}` (e.g. `feat/file-tag-done`, `fix/tree-render`).
- Before `git commit`: run `git branch --show-current`. If on `master`, do NOT commit — ask user for a branch name (suggest one based on the changes, e.g. `docs/simplify-branch-workflow`), create it, commit there.
- Core changes → `docs/CHANGELOG.md` (files under `internal/`, `cmd/cli/`, `pkg/`); WebUI changes → `docs/CHANGELOG_webui.md` (files under `cmd/webui/`). Higher versions on top.
- CHANGELOG entry format: same entry has English line then Chinese line on consecutive lines (no blank line between them); different entries are separated by a blank line. Keep each revision concise — one short line per language stating what changed, no background narrative or rationale paragraphs.

## Project Meta
- Module: `github.com/XingfenD/tag_file` (capital `D`). Go 1.25.0 (see `go.mod`). No linter, no CI.
- SQLite via GORM; DB at `.tagfile/tagfile.db`; schema via `AutoMigrate` only — no raw SQL migrations.
- Version constants: `internal/constant.CoreVersion` (core), `cmd/webui/constant.WebuiVersion` (webui). When bumping CHANGELOG, update the corresponding constant in the same commit.

## Build & Test
- CLI: `make build` → `tagfile.out`. WebUI: `make build-webui` → `tagfile-webui.out`; run with `go run ./cmd/webui --source <dir> --addr :8088`.
- Test: `make test` (sets up `temp/test_workspace/`, then `go test ./... -v`).
- Never build ad-hoc test binaries — use `go test`, or the Makefile-built `tagfile.out`.

## Core Gotchas (`internal/`)
- `internal/` is not importable externally. Uniqueness is enforced at the service layer, and it is not symmetric: category names are globally unique, **root tag names are globally unique too** (not per-category), while a nested tag is scoped only to its parent. The duplicate check is `tagRepo.Find().ByParentAndName(parentID, name)` (`internal/repository/tag/find.go`), which filters parent + name and never `category_id` — so `tag.Service.AddTag` rejects a new root whose name already exists as a root in a *different* category with 「标签 'X' 在此层级已存在」. The AI review gate inherits this (a suggestion naming another category's root tag fails per-row and stays pending). Loosening it is a product decision, not a cleanup.
- Repositories use chainable builders for the list/update-style operations that take options: `repo.List().ByCategory(id).Offset(n).Limit(n).Exec()`, and a builder's option setters and `Exec` sit in the same file as its entrypoint. Files are organized by concern and most hold one operation (`config/set.go`, `category/list.go`), but several plain entrypoints do share one file (`file/list.go`, `ai/task_claim.go`, `taggroup/group_find.go`).
- Tag tree: `Tag.ParentID` + `ListTagTree()`; search resolves subtrees via `GetDescendantIDs()`. Traversal is in-memory (load all tags once, no per-node DB calls).
- `File.TagCnt` is a redundant counter — the repo exposes only `IncrementTagCnt(fileID, delta)` (`internal/repository/file/update.go`); there is no `DecrementTagCnt`, a removal passes a negative delta (`IncrementTagCnt(fileID, -1)`).
- Two independent tag-value systems: `TagValue` (predefined dictionary) vs `TagGroupTag.TagValue` (the value stored on one association; part of its `idx_group_tag_val` unique index).

## WebUI Gotchas (`cmd/webui/`)
- Routing is Gin, not `net/http`: `api.Router.RegisterRoutes` mounts the JSON API on the `/api` group, `handlers.Server.RegisterRoutes` mounts the HTML pages. Read input from `c.Param`/`c.Query`/`c.PostForm`/`c.PostFormArray`, never from `r.URL.Query()`. Service imports aliased, e.g. `categorySvc ".../internal/service/category"`.
- Every handler answers JSON through `respond(c, msg, msgType, extra...)`, which flattens to `{ok, msg, ...extra}` and returns HTTP 400 when `msgType` is `"error"`. Request bodies are `application/x-www-form-urlencoded` (`API.post`/`API.put` set the header and wrap the payload in `URLSearchParams`); repeatable keys need a real `URLSearchParams` instance because an object literal collapses them into one comma-joined value. `API.post`/`put`/`del` return `r.json()` whatever the status, so JS judges a call by `data.ok`, not by `try`/`catch`.
- `API.del` sends a bodyless `DELETE` with its parameters in the URL query (`fetch(url, {method:'DELETE'})`; Go parses a request body only for POST/PUT/PATCH), so a DELETE handler must read parameters with `c.Query` and `c.Param` — never `c.PostForm`, which is always empty there. The rule protects three operations: `HandleUntagFile` (`DELETE /api/files/tag`, `file_path`/`tag_id`/`tag_value`/`group_id` from the query), `HandleRemoveGroup` (`DELETE /api/files/group/:id`, `file_path` from the query, `id` from the path) and `HandleRemoveTagValue` (`DELETE /api/tags/values/:id`, `tag_name`/`value` from the query). A `c.PostForm` read in any of them yields `""` and the handler refuses the call before touching the database (「无效的文件路径」, 「文件 '' 未注册」, 「标签名和值不能为空」).
- A Gin `*path` wildcard hands `c.Param("path")` *with* a leading slash while `File.FilePath` is stored relative, so `/api/files/detail/*path` and `/api/files/groups/*path` `TrimPrefix` it before looking the file up.
- Templates: `{{define}}` cannot be nested — every block sits at the top level of its own file (`content` in each page, `base` in `base.html`, and the `empty`/`config_group`/`config_field` partials under `templates/partials/`, all parsed in `cmd/webui/main.go`), and a page pulls one in with `{{template "empty" ...}}` from inside its own `content` block. A recursive structure is not a template at all: the tag tree is rendered by JS (`cmd/webui/static/js/pages/tag/main.js`) because Alpine has no recursive templates.
- Inline editing is the Alpine `inlineEdit` component (`cmd/webui/static/js/core/components.js`): each cell carries its own `x-data="inlineEdit({ value, onSave })"`, so there is no positional cell array kept in sync with table columns.
- A new page needs three registrations at once: the page-name slice in `cmd/webui/main.go` (where its template is parsed into `pages`), a route in `handlers.Server.RegisterRoutes`, and the nav entry in `cmd/webui/templates/base.html`.
