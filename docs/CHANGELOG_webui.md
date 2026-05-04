# Changelog (Web UI) / 更新日志 (Web UI)

tagfile 项目 Web 界面的版本变更记录。格式参考 [Keep a Changelog](https://keepachangelog.com)。

## [0.4.4] - 2026-05-05

### Changed / 变更

- Tag add and delete operations use AJAX (fetch + location.reload) instead of form POST redirects; page stays at scroll position after each operation.
- 标签添加和删除改用 AJAX（fetch + location.reload）替代表单 POST 重定向，操作后页面保持在原滚动位置。

### Fixed / 修复

- Flat view parent column now shows full hierarchical path (e.g. `猫 / 英短`) via recursive `buildTagPath` instead of only immediate parent name.
- 平铺视图父标签列现在通过递归 `buildTagPath` 展示完整层级路径（如 `猫 / 英短`），而非仅直接父标签名。
- Tree view tag delete now preserves URL query params (`category_id`, `view`) via `location.search` in onsubmit, fixing category filter reset on deletion.
- 树形视图删除标签后通过 `onsubmit` 中的 `location.search` 保留 URL 查询参数（`category_id`、`view`），修复删除后分类筛选被重置。

## [0.4.3] - 2026-05-05

### Changed / 变更

- Tidied button CSS: added `.btn` base class, `.btn-xs` for tiny buttons, `.badge` for tag counts; removed global `border:none` conflict; added hover states for all variants.
- 整理按钮样式：新增 `.btn` 基类、`.btn-xs` 超小按钮、`.badge` 标签数徽章；移除全局 `border:none` 冲突；所有变体增加 hover 态。

## [0.4.2] - 2026-05-05

### Added / 新增

- Add tag form category follows URL `?category_id` param: preselected on page load with parent tags auto-loaded.
- 添加标签表单的所属分类跟随 URL `?category_id` 预选，页面加载时自动加载父标签选项。

### Fixed / 修复

- Restore `Preload("Category")` on `FindChildren` / `FindRootsByCategory` lost during chainable builder refactor, fixing blank category names in tree view.
- 修复链式 Builder 重构后 `FindChildren` / `FindRootsByCategory` 丢失 `Preload("Category")` 导致树形视图分类名为空。

## [0.4.1] - 2026-05-05

### Added / 新增

- Tag count badge in file tree view: indexed files show red badge with tag count next to filename.
- 文件目录树展示标签数徽章：已索引文件名旁显示红色数字徽章。

- Tag count badge in detail page directory sidebar: same badge shown for sibling files.
- 详情页目录侧边栏同样展示文件标签数徽章。

## [0.4.0] - 2026-05-05

### Added / 新增

- Tag group tabs on file detail page: switch between groups via AJAX without page reload.
- 文件详情页标签组 Tab 切换：通过 AJAX 切换标签组，不刷新页面。

- Tag group management page (`GET /files/groups/{path...}`): list all groups with tag counts, inline rename, create/delete groups.
- 标签组管理页：展示文件所有标签组及标签数量，支持内联重命名、创建和删除。

- Create/delete tag group via AJAX from file detail page: `+` button expands inline form, new tab inserted into DOM; `×` button on non-default tabs removes group.
- 文件详情页 AJAX 创建/删除标签组：`+` 按钮展开内联表单，新组直接插入 DOM；非默认组 `×` 按钮删除。

- Rename tag group: `POST /files/group/rename` endpoint and inline edit on management page.
- 重命名标签组：新增路由和内联编辑。

- Tag tagging/un-tagging scoped to selected group via `group_id` parameter.
- 打标/去标操作通过 `group_id` 限定到选中标签组。

### Changed / 变更

- `HandleTagFile` / `HandleUntagFile` accept `group_id` form value; fall back to default group when not provided.
- 打标/去标 Handler 接受 `group_id` 参数，未提供时回退到默认组。

- `HandleFileDetail` passes `Groups` and `GroupID` to template; default selected group is first in list.
- 文件详情 Handler 传递 `Groups` 和 `GroupID` 到模板，默认选中第一组。

- `renderTagTable` AJAX partial respects `group_id` for paginated tag listing.
- AJAX 分页标签列表支持 `group_id` 过滤。

### Fixed / 修复

- Adapted to core refactor: `model.FileTag` → `model.TagGroupTag`, added `UpdateTagValue` to taggroup repository and file service.
- 适配核心重构：`FileTag` → `TagGroupTag`，新增 `UpdateTagValue` 方法。

## [0.3.0] - 2026-05-04

### Added / 新增

- Reusable partial templates: `pagination`, `empty`, `tagBadge` components under `templates/partials/`.
- 可复用组件模板：`templates/partials/` 下新增 `pagination`（分页）、`empty`（空状态）、`tagBadge`（标签徽章）组件。

- Shared JavaScript library: `flashMsg`, `confirmAction`, `setupTagSearch` in `partials/shared_js.html`, injected before every page content.
- 共享 JS 库：`partials/shared_js.html` 提供 `flashMsg`、`confirmAction`、`setupTagSearch`，注入到所有页面。

- Unified tag autocomplete: `setupTagSearch(input, dropdown, onInput)` replaces both `<datalist>` and custom dropdown across all tag selectors. Supports substring filtering, mouse hover highlight, and keyboard navigation (`↑↓` cycle, `Enter` select first/highlighted, `Escape` dismiss).
- 统一标签搜索组件：`setupTagSearch` 替代所有页面的 datalist 和自定义下拉。支持子串过滤、鼠标悬停高亮、键盘导航（上下键循环、Enter 选中、Escape 关闭）。

- Parent tag selector preloads all tags on page load (server-rendered `allTags` array), no async fetch delay.
- 父标签选择器页面加载时预渲染所有标签为 `allTags` 数组，无异步请求延迟。

- `/api/tags` JSON endpoint now includes `cat` (category name) and `cid` (category ID) fields.
- `/api/tags` JSON 接口新增 `cat`（分类名）和 `cid`（分类 ID）字段。

### Changed / 变更

- `main.go`: `//go:embed` changed from `templates/*` to `templates` to support `templates/partials/` subdirectory.
- `main.go`: embed 指令从 `templates/*` 改为 `templates`，支持 `templates/partials/` 子目录。

- All page templates parse all partials, enabling `{{template "pagination" .}}` and `{{template "empty"}}` everywhere.
- 所有页面模板均解析全部 partials，`{{template "pagination" .}}` 和 `{{template "empty"}}` 可在任意页面使用。

- Pagination in `category.html`, `config.html`, `tag.html`, `file_detail.html` replaced with `{{template "pagination" .}}` (supports both link-based and JS `onclick` callback via `.PageFn`).
- `category.html`、`config.html`、`tag.html`、`file_detail.html` 的分页替换为 `{{template "pagination" .}}`（支持链接跳转和 JS 回调两种模式）。

- Empty states in all 6 content pages replaced with `{{template "empty" "..."}}`.
- 全部 6 个内容页的空状态替换为 `{{template "empty" "..."}}`。

- `file_detail.html`: removed local `flashMsg()`, `onTagInput()`, `hideTagDropdown()` — replaced by shared `setupTagSearch` and global `flashMsg`.
- `file_detail.html`：移除本地的 `flashMsg`、`onTagInput`、`hideTagDropdown`，改用共享函数。

- `tag.html`: parent tag selector replaced `<input list>` + `<datalist>` with `<div>` dropdown wired via `setupTagSearch`; category change filters from preloaded `allTags` instead of `fetch()`.
- `tag.html`：父标签选择器从 datalist 改为 `setupTagSearch` 自定义下拉；切换分类改为从预加载 `allTags` 客户端过滤。

- `renderTagTable()` in `handlers/file.go`: pagination HTML generation replaced with `template.Lookup("pagination").Execute()`.
- `handlers/file.go` 中 `renderTagTable()` 的分页 HTML 生成改为模板执行。

## [0.2.1] - 2026-05-04

### Added / 新增

- Custom filtered tag dropdown: replaces native `<datalist>` with client-side filtered list, hidden when input is empty.
- 自定义过滤标签下拉框：替代原生 datalist，输入为空时隐藏，仅匹配时展示过滤结果。
- 标签路径展示：API 返回直接父标签/自身路径（如 `a/b`），所有选择器中以路径形式展示。

- Parent tag selectors use searchable `<input>` + `<datalist>` with path display.
- 父标签选择器使用可搜索 input + datalist，展示路径。

- Tag path display: API returns immediate parent/self path (e.g. `a/b`), shown in all tag selectors.
- 标签路径展示：API 返回直接父标签/自身路径（如 `a/b`），所有选择器中以路径形式展示。

- Quick-add child tag in tree view: `+` button on each node expands inline input, auto-fills parent/category.
- 树形视图快速添加子标签：每个节点 `+` 按钮展开内联输入框，自动填充父标签和分类。

- Redesigned file detail tag section: paginated tag table below media preview with add row, inline value edit and delete.
- 重构文件详情标签区域：媒体预览下方分页标签表格，首行为添加标签，每行支持编辑值和删除。

- Custom filtered tag dropdown: replaces native `<datalist>` with client-side filtered list, hidden when input is empty.
- 自定义过滤标签下拉框：替代原生 datalist，输入为空时隐藏，仅匹配时展示过滤结果。

### Changed / 变更

- Tag list page defaults to tree view; search still auto-switches to flat view.
- 标签管理页默认展示树形结构；搜索时自动切换为平铺视图。

### Fixed / 修复

- Fixed category filter dropdown not showing selected state: added `selected` attribute matching `?category_id`.
- 修复分类筛选下拉不显示选中状态：为匹配 `?category_id` 的选项添加 `selected` 属性。

- Fixed flat view inline edit parent tag: changed from text input to `<select>` dropdown populated by API.
- 修复平铺视图行内编辑父标签：从文本输入改为 API 加载的 `<select>` 下拉。

- Fixed tag value edit creating duplicates: replaced untag+retag with direct `UPDATE` via `POST /files/tag/update`.
- 修复编辑标签值产生重复条目：将先删后增改为直接 UPDATE。

- Fixed tag table pagination causing page reload: switched to `GET ?partial=tags` AJAX for paging and refresh.
- 修复标签表格分页导致页面刷新：分页和刷新改为 `GET ?partial=tags` AJAX 局部加载。

- Fixed tag/untag on file detail causing video reset: replaced form POST with `fetch()` + DOM partial refresh.
- 修复文件详情打标签导致视频重置：表单提交改为 `fetch()` + DOM 局部刷新。

- Fixed file paths with spaces in detail page URLs: added `urlEncode` template function, URL-encode redirect paths, decode `PathValue` in handler.
- 修复文件名含空格时详情页链接：新增 `urlEncode` 模板函数，Redirect 和模板链接均 URL 编码，Handler 解码 `PathValue`。

- Fixed flash params accumulating in URL across multiple operations: strip `flash`/`flash_type` from `_return` before redirect.
- 修复多次操作后 URL 中 flash 参数不断累积：Redirect 前从 `_return` 中去除 flash 参数。

- Fixed directory sidebar on detail page not scrolling independently: use `position: sticky` so sidebar stays visible while main content scrolls.
- 修复详情页目录侧边栏不能独立滚动：改用 `position: sticky`，右侧滚动时左侧保持可见。

### Changed / 变更

- `Redirect()` now preserves page state via hidden `_return` field in forms, carrying `view`, `category_id`, `q` params across add/remove operations.
- `Redirect()` 通过表单隐藏域 `_return` 保留页面筛选状态（`view`、`category_id`、`q`），操作后不丢失。

- Repository layer: `ListByCategory`, `List`, `SearchByName`, `FindChildren`, `FindRootsByCategory` now preload `Category` and/or `Parent` relations.
- Repository 层：列表/搜索/树形查询方法增加 `Category`/`Parent` 关联预加载。

## [0.2.0] - 2026-05-04

### Removed / 移除

- Removed attribute management: deleted `handlers/attribute.go`, `templates/attribute.html`, all `/attributes` routes, and "属性管理" navigation link.
- 移除属性管理模块：删除 `attribute.go`、`attribute.html`、所有 `/attributes` 路由及导航入口。

### Added / 新增

- Tag tree support: tags can have parent-child hierarchy; tree view toggle (`?view=tree`) on tag list page uses `ListTagTree` to render collapsible hierarchy.
- 标签树形支持：标签支持父子层级；标签列表页新增平铺/树形切换，使用 `ListTagTree` 渲染可折叠树。

- Parent tag selector with dropdown: add form auto-loads available parent tags via `GET /api/tags?category_id=X` when category changes; inline edit form also supports parent tag selection.
- 父标签下拉选择器：新增标签时根据所选分类通过 API 动态加载可选父标签；行内编辑也支持父标签下拉选择。

- `GET /api/tags` JSON endpoint: returns tag list filtered by `category_id` for dynamic parent tag selection.
- 新增 `GET /api/tags` JSON 接口，按分类返回标签列表供动态下拉选择。

- Tag value management: inline value list per tag with add/remove on tag page; `GET /api/tags/values?tag_name=X` API for value dropdown.
- 标签预设值管理：标签页内联展示每个标签的值列表，支持添加/删除；`GET /api/tags/values` API 提供值下拉数据。

- File tagging with value: detail page tag form shows value dropdown if tag has predefined values, else text input; tags display as `TagName:Value` format.
- 文件打标签支持值：详情页打标签表单根据标签预设值动态切换下拉/文本输入；已打标签以 `标签:值` 格式展示。

- Category filter dropdown on tag list page: filter tags by category, auto-submit on select, preserved across flat/tree view toggle.
- 标签列表页新增分类筛选下拉框：按分类过滤标签，选择后自动提交，平铺/树形切换时保持筛选。

### Fixed / 修复

- File tree view now respects `scan_exclude` config: replaces hardcoded dot-prefix exclusion with regex matching from the config table, consistent with scan service behavior.
- 文件目录树视图现在遵循 `scan_exclude` 配置：将硬编码的点文件排除改为读取数据库配置的正则匹配，与扫描服务行为一致。

- Fixed category inline edit save button: replaced `<form>` split across `<td>` cells with `fetch()`-based submit.
- 修复分类行内编辑保存按钮无反应：跨 `<td>` 的 `<form>` 改为 `fetch()` 提交。

- Fixed superfluous `WriteHeader` warnings: replaced `http.Error` after partial template render with stderr logging to avoid double header writes.
- 修复 superfluous WriteHeader 告警：模板渲染失败后不重复写响应头，改为 stderr 日志。

- Fixed tag tree render error: replaced invalid `$.CategoryID` in recursive sub-template with JS `location.search` based `_return` population.
- 修复标签树渲染错误：递归子模板中 `$.CategoryID` 改为 JS onsubmit 读取 URL query。

## [0.1.1] - 2026-05-04

### Added / 新增

- Added directory tree view for file list, with collapsible folders, indexed/unindexed markers, based on actual filesystem.
- 新增文件列表目录树视图，基于实际文件系统，支持折叠目录、索引/未索引状态标记。

- `webui.hide_extensions` config to filter file extensions from directory tree view (empty dirs auto-pruned).
- 新增 `webui.hide_extensions` 配置项，可在目录树视图中过滤指定后缀的文件（空目录自动隐藏）。

- Added file picker for import, supporting both file upload and manual path input in a single field.
- 新增文件选择器导入，同一输入框支持文件选取和手动输入路径。

- Tree structure cached to `.tagfile/webui_tree.json` to avoid repeated filesystem scans; auto-invalidated on scan/import/rename.
- 目录树缓存至 `.tagfile/webui_tree.json`，避免反复扫盘；扫描/导入/重命名后自动失效。

- Inline edit on category, tag, attribute, config list pages; search bar on all list pages.
- 分类/标签/属性/配置列表页支持行内编辑和搜索。

- File rename form on detail page.
- 文件详情页新增重命名表单。

- Added directory sidebar on file detail page, showing indexed and unindexed sibling files in the same directory.
- 文件详情页新增目录侧边栏，展示同目录下已索引和未索引的兄弟文件。

### Changed / 变更

- Moved file tag/un-tag operations from file list page to `/files/detail/{path...}` detail page with tag dropdowns.
- 文件打/去标签操作从列表页移至详情页，URL 使用文件路径，标签下拉选择。

- Detail page URL changed from `/files/detail/{id}` to `/files/detail/{path...}` using file path.
- 详情页 URL 改用文件路径 `/files/detail/{path...}`。

- Removed flat table file list, directory tree view only.
- 移除平铺表格文件列表，仅保留目录树视图。

- Filename links in both tree and table views navigate to detail page.
- 文件名点击可跳转至详情页。

### Fixed / 修复

- Fixed flash message display: `Redirect()` now URL-encodes flash text, `Render()` preserves handler-set flash values instead of overwriting them.
- 修复提示消息不显示：`Redirect()` 对消息文本进行 URL 编码，`Render()` 保留 handler 设置的 flash 值而非覆盖。

- Updated webui import paths to match refactored service/repository sub-packages.
- 更新 webui 的 import 路径以适配重构后的 service/repository 子包。

- Fixed video/audio seeking: replaced `os.ReadFile` with `http.ServeFile` to support HTTP Range requests (206 Partial Content).
- 修复视频/音频无法快进：改用 `http.ServeFile` 支持 HTTP Range 请求。

## [0.1.0] - 2026-05-03

### Added / 新增

- Added `cmd/webui/handlers/` package with modular handler files (category, tag, attribute, file, search, config).
- 新增 `cmd/webui/handlers/` 包，处理函数拆分为独立模块文件。

- Added `GET /files/detail/{id}` dedicated route for file detail page with inline media playback.
- 新增 `GET /files/detail/{id}` 独立路由，文件详情页支持图片/视频/音频内联播放。

- Added `GET /files/raw` endpoint to serve file bytes with proper MIME type.
- 新增 `GET /files/raw` 端点，用正确的 MIME 类型提供文件原始内容。

- Added `--source` and `--addr` CLI flags for web UI.
- 新增 `--source` 和 `--addr` 命令行参数。

- Added `file_detail.html` standalone template for file detail view.
- 新增 `file_detail.html` 独立模板。

### Changed / 变更

- Refactored `cmd/webui/main.go` from single 900-line file into `handlers/` package with 9 files.
- 将 `cmd/webui/main.go` 从单文件 900 行重构为 `handlers/` 包内 9 个文件。

- Updated pagination to 10 items per page, fixed page clamping bug when total unknown.
- 分页改为每页 10 条，修复 total 未知时页码被钳制回第1页的问题。

- Extracted shared helpers: `UpdatePagination`, `IsMediaType`, `Paginate`, `ParseInt`.
- 提取共享 helper 函数。
