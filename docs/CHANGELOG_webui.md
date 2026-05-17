# Changelog (Web UI) / 更新日志 (Web UI)

tagfile 项目 Web 界面的版本变更记录。格式参考 [Keep a Changelog](https://keepachangelog.com)。

## [1.1.1] - 2026-05-17

### Changed / 变更

- Inline-edit rewritten: cells now contain a permanent `<input readonly>` styled as plain text (transparent background, no border). Click removes `readonly` and shows a subtle dashed underline. No DOM replacement, no layout shift. Blur to save, Esc to cancel, Enter to confirm. Eliminates the visibility-hidden overlay approach that caused text ghosting.
- Click-to-Edit 重写：单元格始终持有一个 `<input readonly>` 组件，默认态样式消隐如普通文字（透明背景、无边框）。点击解除 `readonly` 并显示虚线底边标识编辑态。不再替换 DOM，无排版变化。失焦保存，Esc 取消，Enter 确认。消除之前 visibility-hidden 叠加导致的文字重影问题。

- Category management: removed per-row "Edit" button. Category name and description columns now use click-to-edit — click directly on the cell to modify.
- 分类管理：删除每行的「编辑」按钮，名称和描述列改为 click-to-edit，直接点击单元格即可修改。

- File detail tag table: removed "Edit value" button. The value column already supports click-to-edit.
- 文件详情标签列表：删除「编辑值」按钮，值列已支持 click-to-edit。

- File tree view: removed redundant "Details" button next to indexed files. The filename is already a clickable link to the detail page.
- 文件目录树：删除已索引文件旁的「详情」按钮，文件名本身已是详情页链接。

## [1.1.0] - 2026-05-17

### Changed / 变更

- Modal system replaces innerHTML edit patterns across all CRUD pages: new `js/core/modal.js` provides a global singleton form modal with `Modal.show({title, fields, onSave})` API, building forms entirely with `createElement` (zero innerHTML). Supports text, select, textarea, and autocomplete field types. All 6 CRUD modules refactored — `cells[n].innerHTML` patterns removed. Enter to confirm, Esc to cancel.
- 全局 Modal 系统替代 innerHTML 编辑模式：新增 `js/core/modal.js`，`Modal.show({title, fields, onSave})` 接口，全 `createElement` 构建表单（零 innerHTML），支持 text/select/textarea/autocomplete 字段类型。6 个 CRUD 模块全部重构，删除 `cells[n].innerHTML` 模式。Enter 确认，Esc 取消。

- Click-to-Edit for single-field edits: new `js/core/inline-edit.js` with `makeEditable(el, {onSave})` API. Click cell to switch to input, blur to auto-save, Esc to cancel. Input styled subtly (transparent background, dashed bottom border, content-based width) to avoid stretching table columns. Applied to file tag value editing and group renaming.
- Click-to-Edit 单字段原地编辑：新增 `js/core/inline-edit.js`，`makeEditable(el, {onSave})` 接口，点击单元格切换为 input、失焦自动保存、Esc 取消。输入框样式消隐（透明背景、虚线底边框、宽度按内容自适应），不撑宽表格列。应用于文件标签值编辑与标签组重命名。

- Form submission overhaul: all `<form>` elements stripped of `onsubmit`, `method="GET"`, and `action` attributes. Every form now uses `addEventListener('submit', e.preventDefault())` with AJAX submission — zero page refreshes.
- 表单提交流程重写：所有 `<form>` 移除 `onsubmit`/`method="GET"`/`action` 属性，统一改为 `addEventListener('submit', e.preventDefault())` AJAX 提交，零页面刷新。

- File detail page partial reload: new `reloadGroupsAndTags()` only refreshes the tag table area. Tag add/remove/edit and group create/remove/switch no longer re-render the file info card, directory sidebar, or media preview.
- 文件详情页局部刷新：新增 `reloadGroupsAndTags()` 只刷新标签区域。标签增删改、标签组创建/删除/切换均不再重绘文件信息卡片、目录侧边栏、媒体预览。

- Create tag group via Modal dialog instead of inline expandable form. Removed `add-group-form` DOM built inside `renderGroupsAndTags()`.
- 创建标签组从内嵌表单改为 Modal 弹窗，移除 `renderGroupsAndTags()` 中 `add-group-form` 内嵌 DOM。

- Tag preset value input changed from browser `prompt()` to Modal dialog.
- 标签预设值添加从浏览器 `prompt()` 改为 Modal 弹窗输入。

### Fixed / 修复

- DELETE request parameter passing: Go `r.FormValue()` ignores request body for DELETE requests, only reads URL query string. Fixed 3 DELETE endpoints (remove file tag, remove tag group ×2, remove tag preset value) — parameters moved from request body to URL query string.
- DELETE 请求参数传递修复：Go `r.FormValue()` 对 DELETE 请求不解析 body，仅读 URL query。修复 3 处 DELETE（移除文件标签、删除标签组 ×2、删除标签预设值），参数从 body 改为 URL query string.

## [1.0.1] - 2026-05-17

### Changed / 变更

- **Static assets reorganization**: CSS moved into `css/core/`, `css/layout/`, `css/components/` (7 focused files from the monolithic `components.css`). JS moved into `js/core/` and `js/pages/`. New `js/core/utils.js` extracts 6 duplicated utility functions (`escHtml`, `escJsStr`, `formatSize`, `iconForMime`, `iconForName`, `encodePath`) from 4 page files.
- **静态资源重构**：CSS 按 core/layout/components 三级目录拆分（`components.css` 拆为 7 个独立文件）。JS 按 core/pages 两级目录组织。新增 `js/core/utils.js` 提取 4 个页面文件中重复的 6 个工具函数。

- **Page JS split into per-concern files**: each page JS file split into its own subdirectory with methods grouped by category. `file/` (5 files: main, tree, detail-core, detail-tags, groups), `tag/` (7 files: main, sidebar, list-tree, list-flat, values, crud, drag-drop), `category/` (2 files: main, crud), `config/` (2 files: main, crud), `home/` (1 file: main).
- **页面 JS 按职能拆分**：每个页面 JS 文件拆为独立子目录，方法按类别分文件。`file/`（5 个文件）、`tag/`（7 个文件）、`category/`（2 个文件）、`config/`（2 个文件）、`home/`（1 个文件）。

## [1.0.0] - 2026-05-17

### Added / 新增

- `api/` package (6 files, 1538 lines): JSON data layer with 29 RESTful `/api/...` endpoints covering categories, tags, tag values, files, config, and export. All handlers return typed structs with consistent `{"ok":bool,"msg":"..."}` envelope.
- `api/` 包（6 个文件，1538 行）：JSON 数据层，29 个 RESTful `/api/...` 端点，覆盖分类、标签、标签值、文件、配置、导出。统一返回 typed struct + `{"ok":bool,"msg":"..."}` 信封。

- `static/js/api.js`: shared fetch wrapper (`API.get/post/put/del`) + toast notification helper, loaded globally via `base.html`.
- `static/js/api.js`：共享 fetch 封装 + toast 通知，通过 `base.html` 全局加载。

- `static/js/home.js`: fetches dashboard counts from `/api/...` endpoints on page load.
- `static/js/home.js`：页面加载时从 API 获取首页仪表盘计数。

- `static/js/category.js` (152 lines): full CRUD with inline edit, pagination, and search via `/api/categories`.
- `static/js/category.js`（152 行）：分类 CRUD，内联编辑、分页、搜索全部通过 API。

- `static/js/config.js` (191 lines): full CRUD with set/update mode toggle via `/api/config`.
- `static/js/config.js`（191 行）：配置 CRUD，支持 set/update 模式切换。

- `static/js/tag.js` (714 lines): tree view, flat table, tag value CRUD, drag-and-drop parent reassignment, sidebar category tree with filter — all via `/api/tags`.
- `static/js/tag.js`（714 行）：树形/平铺视图、标签值 CRUD、拖拽改父标签、左侧分类树筛选，全部通过 API。

- `static/js/file.js` (827 lines): three page controllers (file tree with collapsible `<details>`, file detail with media preview + group tabs + paginated tag table, file groups management) — all via `/api/files`.
- `static/js/file.js`（827 行）：三个页面控制器（可折叠文件树、文件详情含媒体预览+标签组Tab+分页标签表、标签组管理），全部通过 API。

- `api.Router.RegisterRoutes()` + `handlers.Server.RegisterRoutes()`: clean separation of API routes and page routes in `main.go`.
- `api.Router.RegisterRoutes()` + `handlers.Server.RegisterRoutes()`：API 路由和页面路由在 `main.go` 中清晰分离。

- `respondStatus()` variant for non-400 error codes (e.g. 500 for export failures).
- `respondStatus()` 变体支持非 400 错误状态码（如导出失败返回 500）。

### Changed / 变更

- **Handler-API separation**: `handlers/` package stripped from ~1800 lines (9 files) to ~291 lines (3 files). All data-access logic moved to `api/` package. Page handlers now render empty HTML shells only.
- **前后端分离**：`handlers/` 包从 ~1800 行（9 个文件）精简至 ~291 行（3 个文件）。所有数据访问逻辑移至 `api/` 包。页面 handler 仅渲染空 HTML 壳。

- Server struct no longer holds service dependencies (except SearchSvc for POST /search, pending future overhaul).
- Server 结构体不再持有 service 依赖（POST /search 的 SearchSvc 除外，待后续大改）。

- `HandlePage(active, page)` closure factory replaces 7 individual page handler methods.
- `HandlePage(active, page)` 闭包工厂替代 7 个独立页面 handler 方法。

- Templates: all `{{range}}` loops, `{{if .Data}}` conditionals, and server-side data rendering removed. Templates now contain only container DOM + page-specific `<script>` tags. Total template code reduced by ~994 lines.
- 模板：移除所有 `{{range}}` 循环、`{{if}}` 条件渲染和服务端数据注入。模板仅保留容器 DOM + 页面 `<script>` 标签。模板代码净减 ~994 行。

- All POST/PUT/DELETE operations migrated from form-based `/module/action` routes to RESTful `/api/module` + HTTP method routing (Go 1.22+ syntax).
- 所有增删改操作从 `/module/action` 表单路由迁移至 RESTful `/api/module` + HTTP 方法路由（Go 1.22+ 语法）。

- `Redirect()` flash-based error passing removed; errors now returned as JSON responses.
- 移除 `Redirect()` flash 错误传递；错误统一通过 JSON 响应返回。

- `buildTagPath` shared helper centralized in `api/api.go`, eliminating duplication between tag and file handlers.
- `buildTagPath` 共享辅助函数集中至 `api/api.go`，消除 tag 和 file handler 之间的重复。

- Export handler moved from `handlers/export.go` to `api/export.go`; uses buffer before setting headers to avoid write-order bugs.
- 导出 handler 从 `handlers/export.go` 移至 `api/export.go`；先缓冲再设响应头，避免写入顺序 bug。

- `base.html` loads `api.js` globally; per-template duplicate `<script>` tags removed.
- `base.html` 全局加载 `api.js`；移除各模板中重复的 `<script>` 标签。

### Removed / 移除

- Deleted `handlers/category.go`, `handlers/tag.go`, `handlers/file.go`, `handlers/config.go`, `handlers/export.go` — all data logic moved to `api/`.
- 删除 5 个 handler 文件，数据逻辑全部移至 `api/`。

- Removed `Redirect()` method, `respond()` from handlers (now in api package), all `flash`-related logic.
- 移除 `Redirect()` 方法、handlers 中的 `respond()`（现属 api 包）、所有 flash 相关逻辑。

- Removed file tree helper functions from `handlers/file.go` (buildFileTreeFromFS, sortTree, pruneEmptyDirs, etc.) — ported to api package.
- 移除 `handlers/file.go` 中的文件树辅助函数（已移至 api 包）。

- Removed `{{define "categoryList"}}`, `{{define "tagTreeNode"}}`, `{{define "sideTagTreeNode"}}`, `{{define "treeNode"}}` sub-templates — now rendered by JS.
- 移除 4 个 Go 子模板，改由 JS 渲染。

## [0.5.1] - 2026-05-17

### Changed / 变更

- All POST form submissions migrated from traditional form POST-Redirect-GET to `fetch()` + `respond()` JSON: category add/remove/update, config set/unset/update, file scan/import/refresh/rename, tag CRUD, tag value CRUD, group create/remove/rename.
- 所有 POST 表单提交从传统 form POST-Redirect-GET 迁移为 `fetch()` + `respond()` JSON 模式。

- `Redirect()` now only used for GET handler error redirects; new `respond(w, msg, type, extra...)` returns JSON for all POST handlers.
- `Redirect()` 仅用于 GET handler 错误重定向；新增 `respond()` 统一 POST handler JSON 返回。

- Removed URL-param flash system: deleted `GetFlash()`, `cleanQuery()`, `setFlashCookie()`, flash banner from `base.html`.
- 移除 URL 参数 flash 通知系统：删除 `GetFlash()`、`cleanQuery()`、`setFlashCookie()` 及 `base.html` 的 flash 横幅。

- Category list page: add/remove/update dynamically refresh table via `?partial=list` AJAX instead of `location.reload()`.
- 分类管理页：增删改通过 `?partial=list` 局部刷新表格，不再 `location.reload()`。

- Removed `_return` hidden fields and `ajax=1` hacks from all form templates.
- 移除所有表单的 `_return` 隐藏字段和 `ajax=1` 参数。

### Fixed / 修复

- Fixed non-ASCII characters (Chinese) causing cookie encoding errors in flash messages.
- 修复非 ASCII 字符导致 cookie 编码错误。

### Removed / 移除

- Removed `flashMsg()` from `app.js`.

## [0.5.0] - 2026-05-17

### Added / 新增

- Complete editorial redesign with steel-blue color system, Noto Serif/Sans typography, refined spacing.
- 全面 Editorial 风格重设计：钢笔墨蓝色系、思源宋体/黑体排版、克制间距。

- Independent static asset serving (`/static/`) with split CSS architecture: variables, base, layout, components, animations.
- 独立静态资源服务 (`/static/`)，CSS 拆分为 variables/base/layout/components/animations 五个模块。

- Staggered reveal animations on all page cards using CSS `nth-child` delays.
- 所有页面卡片均支持渐入动效（CSS nth-child 延迟 stagger）。

- Toast notification system replacing banner flash for async operations.
- Toast 通知系统替代横幅 flash，右上角滑入/滑出。

- Custom confirm modal with editorial styling, replacing native `confirm()`.
- 自定义确认弹窗（Editorial 风格），替代原生 confirm()。

- Tag management: dual-panel layout with category tag tree sidebar + segmented flat/tree toggle.
- 标签管理：双面板布局（左侧分类标签树 + 右侧内容区），segmented control 切换平铺/树形。

- Tag tree drag-and-drop: drag any tag onto another to re-parent, drop zone to make root-level.
- 标签树拖拽操作：拖拽标签到目标标签下成为子标签，拖到放置区成为顶级标签。

- Tag tree action buttons (edit/add/delete) only appear on row hover.
- 标签树操作按钮（编辑/+添加/删除）仅 hover 时显示。

- Sidebar shows separate core and webui version with git describe.
- 侧边栏分开展示 core 和 webui 的 git 版本号。

- Enhanced empty states with centered icons and guidance text across all pages.
- 所有空状态优化：居中图标 + 引导文案。

- File detail: tag preview chips, underline-style group tabs.
- 文件详情：标签预览 chip、下划线风格标签组 tab。

- Global `Ctrl+K` shortcut to focus search input.
- 全局 Ctrl+K 快捷聚焦搜索框。

### Changed / 变更

- Extracted shared JS to `static/app.js` (toast, confirm modal, tag search, keyboard shortcuts).
- 共享 JS 提取至 `static/app.js`（toast、确认弹窗、标签搜索、快捷键）。

- Removed `templates/partials/shared_js.html`, JS logic now in static file with defer loading.
- 删除 `shared_js.html`，JS 逻辑移至独立文件。

- Refined pagination, file tree, sidebar, and all form/table component styling.
- 优化分页、文件树、侧栏和所有表单/表格组件样式。

- Home page: 2×2 grid dashboard with staggered reveal.
- 首页：2×2 网格仪表盘 + 渐入动效。

### Fixed / 修复

- Static file route pattern corrected for Go 1.22+ path parameter syntax.
- 修复 Go 1.22+ 路由模式下静态文件 `{path...}` 路径匹配。

## [0.4.5] - 2026-05-05

### Added / 新增

- Focus returns to tag search input after adding a tag on file detail page, enabling quick consecutive tagging.
- 文件详情添加标签后焦点回到搜索输入框，便于连续打标。

- File tree directory nodes now show aggregated tag count badge (sum of all descendant file tag counts).
- 文件目录树中目录节点展示汇总标签数徽章（子树所有文件标签数之和）。

- Unsupported video formats show a notice message instead of a broken player; no download button shown.
- 不支持的视频格式显示提示信息（无下载按钮），避免空白播放器。

### Fixed / 修复

- File tagging now uses tag_id instead of tag_name to avoid ambiguity when same-named tags exist under different parents.
- 文件打标改用 tag_id 替代 tag_name，避免同名标签在不同父标签下的歧义。

- Fixed double tag value display in file detail preview (e.g. 作者:A:A) caused by previewPaths already including value.
- 修复文件详情标签预览区 value 重复显示问题（如 作者:A:A），previewPaths 不再包含 value。

### Changed / 变更

- File tree expand/collapse state persists via sessionStorage when navigating to/from file detail.
- 文件目录树的展开/折叠状态通过 sessionStorage 保持。

- Tag search input clears tag_id on typing, highlights when selected, first candidate auto-selected on Tab.
- 标签搜索输入框输入时清除 tag_id，选中后高亮，Tab 自动选第一候选项。

- Tree view: replaced +值 button with inline edit tag name button; removed add-value + from tag values cell.
- 树形视图：+值改为编辑标签名按钮，标签值区域去掉添加值 + 号。

## [0.4.4] - 2026-05-05

### Changed / 变更

- Tag CRUD (add/delete/edit/value/quick-add) now uses AJAX with DOM partial replacement (`refreshTagList` + `DOMParser`) instead of form redirects or `location.reload()`. Scroll position naturally preserved.
- 标签 CRUD 全部改为 AJAX + DOM 局部替换（`refreshTagList` + `DOMParser`），不再触发页面重载或 `location.reload()`，滚动位置自然保持。
- File detail tag value edit updates table cell inline; add/remove/edit all sync the preview tag section and count.
- 文件详情标签值编辑内联更新单元格；添加/删除/编辑均同步顶部标签预览区和计数。

### Fixed / 修复

- Flat view parent column now shows full hierarchical path (e.g. `猫 / 英短`) via recursive `buildTagPath` instead of only immediate parent name.
- 平铺视图父标签列现在通过递归 `buildTagPath` 展示完整层级路径（如 `猫 / 英短`），而非仅直接父标签名。
- Tree view tag delete now preserves URL query params (`category_id`, `view`), fixing category filter reset on deletion.
- 树形视图删除标签后保留 URL 查询参数，修复删除后分类筛选被重置。
- File detail tag preview deduplicated by display text (tagName:value) when same tag appears in multiple groups.
- 文件详情标签预览按展示文本去重，同一标签在多组中出现时只显示一次。
- File detail tags now display full parent path (e.g. `猫 / 英短`) in both preview section and tag table; raw tag name preserved in data attributes for API calls.
- 文件详情所有标签展示改为 `父标签 / 当前标签` 路径格式（如 `猫 / 英短`），原始标签名通过 data 属性保留用于 API。
- Fixed tag table pagination: next page onclick was broken by Go html/template auto-escaping PageFn as JS string literal; switched to template.JS type.
- 修复标签表格分页：Go 模板自动将 PageFn 转义为 JS 字符串字面量导致 onclick 失效，改用 template.JS 类型。

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

- `Redirect()` now preserves page state via hidden `_return` field in forms, carrying `view`, `category_id`, `q` params across add/remove operations.
- `Redirect()` 通过表单隐藏域 `_return` 保留页面筛选状态（`view`、`category_id`、`q`），操作后不丢失。

- Repository layer: `ListByCategory`, `List`, `SearchByName`, `FindChildren`, `FindRootsByCategory` now preload `Category` and/or `Parent` relations.
- Repository 层：列表/搜索/树形查询方法增加 `Category`/`Parent` 关联预加载。

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
