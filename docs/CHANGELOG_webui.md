# Changelog (Web UI) / 更新日志 (Web UI)

tagfile 项目 Web 界面的版本变更记录。格式参考 [Keep a Changelog](https://keepachangelog.com)。

## [6.0.0] - 2026-11-22

### Added / 新增

- The home page gained a feature-entry card carrying the 随机打标 button.
- 首页新增功能入口卡，内含「随机打标」按钮。
- The desktop sidebar collapses to an icon rail via the brand-row button; the state persists in localStorage (`tagfile_sidebar_collapsed`) and restores on first paint. Labels return as link `title` tooltips.
- 桌面端侧栏可由品牌行按钮折叠为图标栏；折叠态存 localStorage（`tagfile_sidebar_collapsed`）且首帧即恢复，折叠后导航名以链接 title 提示保留。
- Directory rows in the tree stick while scrolling and stack one per depth (`--td`), so the ancestor chain stays visible; a row releases at its subtree's end.
- 目录树支持黏性层级：滚动时目录行吸顶、按深度逐层叠放，祖先链常驻可见，滚过本目录子树末各自离场。
- The media stage lays each image/video's own blurred, enlarged copy under the content as ambient backdrop, so any aspect ratio sits on content-derived color instead of the flat dark field; stage media keeps square corners.
- 媒体舞台以内容自身的模糊放大拷贝作氛围底：任何比例的图片/视频都垫在取自画面本身的底色上，不再摊在纯色暗板上；主体画面保持方角。
- Clicking stage media opens a full-screen viewer (vendored PhotoSwipe 5): wheel and trackpad pinch zoom the image itself — never the page — with pan and drag; videos play with controls via a dedicated expand button; no cross-file paging by design.
- 点击舞台媒体打开全屏查看器（vendor PhotoSwipe 5）：滚轮与触控板捏合缩放的是图片本身而非整页，支持拖拽平移；视频经独立展开按钮在查看器内播控；按设计不做跨文件翻页。

## [5.25.0] - 2026-11-22

### Changed / 变更

- Random tagging is entered from the file-list toolbar (dice button) as a feature entry; the sidebar nav no longer carries it.
- 随机打标改为文件列表工具栏里的功能入口（骰子按钮），侧栏导航不再单列此项。

## [5.24.0] - 2026-11-22

### Changed / 变更

- Expanding or folding a directory in the tree animates its content height (0.3 s); the caret rotation and the height unfold share the same beat.
- 目录树展开/折叠带 0.3s 内容高度动画，箭头旋转与高度开合同一拍。

## [5.23.0] - 2026-11-22

### Changed / 变更

- The workbench sidebar header carries only the sort control; every detail landing expands the ancestors and scrolls the current row into view. The SSE detail fingerprint tracks rendered fields only.
- 工作台侧栏头部只留排序控件；每次详情落位自动展开祖先目录并把当前行滚进视口。SSE 详情指纹只盯渲染中的字段。
- The current-file marker is a full-bleed band that glides between rows (0.25 s) and re-seats after folds, lazy appends, refreshes, and re-renders; in-place row repaints keep content without replaying the row fade-in.
- 当前文件标记是一条通栏色带，行间平移落位（0.25s），折叠、懒追加、刷新、重渲染后重新贴合所在行；打标/完成/AI 的就地重画只换内容，不重放行级淡入。

### Fixed / 修复

- `decodePath` uses `decodeURIComponent` — the exact inverse of `encodePath` — so URLs containing reserved characters (`#`, `,`, `?`) resolve to the tree row and highlight correctly.
- `decodePath` 改用 `decodeURIComponent`（`encodePath` 的严格逆运算），带保留字符（`#`、`,`、`?`）的 URL 直开能落回与树行一致的路径并正确高亮。

## [5.22.0] - 2026-11-22

### Added / 新增

- The directory sidebar is resizable by a col-resize handle (180–480 px) on the workbench and tag pages; the width persists in localStorage (`tagfile_dir_width`) as a root CSS variable shared by both pages. Double-click/Enter restores the default, arrow keys step 16 px.
- 目录侧栏可在缝隙把手上拖宽（180–480px，工作台与标签页），宽度存 localStorage（`tagfile_dir_width`）并写进根 CSS 变量、两页共用。双击/Enter 恢复默认，方向键步进 16px。

## [5.21.0] - 2026-11-21

### Changed / 变更

- The directory tree is one shared Alpine component (`core/tree.js`, `directoryTree`): full-tree loading, per-level sorting, `<details>` rendering, `tagfile_tree_open:*` fold memory, and lazy loading of unindexed children have a single source; the list page runs on it unchanged.
- 目录树机制单源进共享 Alpine 组件 `core/tree.js`（`directoryTree`）：全树加载、逐级排序、`<details>` 渲染、`tagfile_tree_open:*` 折叠记忆、未索引子项懒加载；列表页换挂后行为不变。
- The workbench sidebar uses that full collapsible tree instead of flat directory lists: clicking a file switches it in place, each detail landing reveals the current row, and tag/done/AI updates repaint just that row.
- 工作台侧栏用完整折叠树替换平面目录列表：点文件就地切换，每次详情落位定位当前行，打标/完成/AI 更新只重绘该行。
- Directory and file names ellipsize and the sidebar reserves its scrollbar gutter; the dir-header breadcrumb shows only the leaf directory name (full path on hover); the sidebar sort control orders tree levels client-side.
- 目录名/文件名省略号截断、侧栏预留滚动条 gutter；头部面包屑只显示末级目录名（悬停看全路径）；侧栏排序控件改为客户端排树层级。

## [5.20.0] - 2026-11-21

### Changed / 变更

- The file detail page is rebuilt as a tagging workbench: a command header (file identity + pipeline step), a dark media stage, and a dedicated tagging panel; recommend mode drives 「完成并下一个 / 下一个 / 跳过」, direct-open keeps the 「标记完成」 toggle.
- 文件详情页重排为打标工作台：命令头（文件身份 + 流水线动作）、深色媒体舞台、独立打标面板；推荐模式走「完成并下一个 / 下一个 / 跳过」，直开模式保留「标记完成」开关。
- Renaming moved into the filename itself (inline edit), the metadata table collapsed to a meta line plus stage footer, and the tag table became compact rows that fit the narrow panel.
- 重命名收进文件名行内编辑；元数据表收成一行信息加舞台脚注；标签表改为适配窄面板的紧凑行。
- The workbench live-updates over the AI SSE stream: a `change` event silently re-reads the detail and a fingerprint over the rendered fields gates the apply; environments without EventSource fall back to a low-frequency silent poll; the 「刷新」 button is removed.
- 工作台经既有 AI SSE 流推送刷新：`change` 事件静默重读详情，渲染字段指纹闸住落位；无 EventSource 的环境退回低频静默轮询；「刷新」按钮删除。

## [5.19.1] - 2026-11-21

### Fixed / 修复

- The directory sidebar uses the shared `revealUp 0.45s forwards` entrance like the other panels, with both reduced-motion guards listing it.
- 目录侧栏改用与其他面板相同的 `revealUp 0.45s forwards` 入场配方，并纳入两处 reduced-motion 兜底。

## [5.19.0] - 2026-11-21

### Changed / 变更

- One content region = one raised panel; panels never nest. The search page states, the detail/recommend placeholders, the unsupported-video notice, and the tag add-form now sit as sections inside their panels; nested query-builder groups keep the dashed marker without the glass recipe.
- 容器语言一条规则：一块内容区 = 一块面板，面板不套面板。搜索页空态、详情/随机打标占位、不支持视频提示、标签页加标签表单四处归位到面板内分区；查询构建器嵌套分组保留虚线标记、去掉玻璃配方。

## [5.18.0] - 2026-11-21

### Added / 新增

- An icon system: a 30-symbol stroke sprite (`partials/icons.html`) reached via `{{template "icon"}}` in templates and `iconHtml()` in JS, styled once as `.icon`; a test pins sprite↔call-site parity and the `empty` partial takes icon names.
- 新增图标系统：30 枚描边图标集中为 sprite（`partials/icons.html`），模板侧 `{{template "icon"}}`、JS 侧 `iconHtml()` 调用，外观单源于 `.icon`；测试钉住 sprite 与调用点一一对应，`empty` partial 改收图标名。

### Changed / 变更

- Every emoji/unicode glyph that stood in for an icon now uses a sprite icon (nav, stat labels, empty states, file types, drag handles, disclosure carets, toasts, sort/back arrows, theme toggle).
- 所有以 emoji/unicode 字符充当图标之处（导航、统计标签、空态、文件类型、抓手、展开箭头、toast、排序/返回箭头、主题开关）换为 sprite 图标。

### Fixed / 修复

- The tag-tree drag handle has its base opacity in `tree.css`, so the hover reveal rule takes effect.
- 标签树拖拽抓手的基线透明度落进 `tree.css`，悬停显形规则真正生效。

## [5.17.1] - 2026-11-21

### Changed / 变更

- Two redundant empty-state sub-hints (recommend scope, search initial) are removed.
- 删除两处冗余空态副文案（随机打标范围提示、搜索初始提示）。

## [5.17.0] - 2026-11-20

### Added / 新增

- The review page's directory field is a combobox with keyboard-navigable candidates from a new `GET /api/files/dirs`, pruned by `scan_exclude`; a failed fetch only drops the suggestions, manual input still works.
- 审核页目录输入框改为组合框，候选来自新增的 `GET /api/files/dirs`（已按 `scan_exclude` 剪枝），支持键盘导航；拉取失败只失去联想，不挡手工输入。

### Changed / 变更

- The 发起识别 form is a two-row grid with input, checkboxes, and button aligned on control centres; single column on narrow screens.
- 「发起识别」表单改为两行网格，输入框、复选框与按钮按控件中心对齐；窄屏回退单列。
- The enqueue card raises its z-index only while its dropdown is open, and `--menu-bg` moved to 0.98 alpha, fixing the two overflow/blending layers.
- 下拉打开期间才抬高「发起识别」卡片层级，`--menu-bg` 提到 0.98 透明度，修复越界遮挡与底层透出。

## [5.16.1] - 2026-11-20

### Changed / 变更

- The theme toggle's sun/moon and 浅色/深色 labels are CSS-swapped from `data-theme` (animated pair, instant under reduced motion); `theme.js` only writes the attribute, so a dark system theme shows the right icon and label from first paint.
- 主题开关的日月图标与「浅色/深色」文案改由 CSS 依 `data-theme` 互换（带过渡，reduced-motion 下瞬时），`theme.js` 只写属性；深色系统主题下首屏即为正确图标与文案。

## [5.16.0] - 2026-11-20

### Added / 新增

- `POST /api/ai/tasks/refine` queues a stage-two re-refine for tasks with stored raws, and `GET /api/ai/tasks/:id/raw` returns the stored first-pass answers frame-ordered; task rows carry `source_task_id`, `refine_status`, `has_raw`, `can_refine`, computed with one aggregated read per page.
- AI 任务新增第二段接口：`POST /api/ai/tasks/refine` 为已存原文的任务入队重新整理；`GET /api/ai/tasks/:id/raw` 按帧序返回识图原文。任务行新增 `source_task_id`/`refine_status`/`has_raw`/`can_refine`（整页聚合读出）。
- The review page gained 「原文」 and 「重新整理」 controls (both gated on the server-computed flags) plus a 「整理失败」 badge; the config page renders the `ai_refine_*` group with no frontend change.
- 审核页接入「原文」「重新整理」（均由服务端字段把关）与「整理失败」徽章；配置页零前端改动渲染 `ai_refine_*` 分组。

### Changed / 变更

- An empty conditional config slot collapses entirely instead of keeping an invisible placeholder row.
- 配置页空的条件槽整槽收起，不再保留隐形占位行。
- Settings height changes animate on grid-row transitions (1fr→0fr) with collapsed content marked inert and clipping active only during the transition; instant under reduced motion.
- 配置页高度变化走 grid 行过渡（1fr→0fr），收起内容置 inert、裁剪仅在过渡期生效；reduced-motion 下瞬时。

### Fixed / 修复

- The select-menu caret no longer drifts on focus/blur: focus changes only `background-color` and the transition list is narrowed to match.
- select-menu 下拉三角不再在聚焦/失焦时漂移：焦点只改 `background-color`，过渡列表同步收窄。
- Floating menus paint a dedicated near-opaque `--menu-bg`, so page content no longer reads through them.
- 浮层菜单改用专门的近不透明 `--menu-bg`，不再透出底下内容。

## [5.15.0] - 2026-11-20

### Changed / 变更

- Toasts are glass cards with a tinted icon medallion, hover close, and a lifetime rail; errors linger 5 s, others 3 s; hover pauses, dismissal is id-keyed two-stage, the stack caps at five, `aria-live` announces, and tests pin the lifecycle.
- Toast 重做为玻璃卡片：语义色图标底座、悬停关闭、寿命进度条；错误停 5 秒、其余 3 秒；悬停暂停、按 id 两段式退场、堆叠上限 5、`aria-live` 播报，生命周期有测试钉住。

## [5.14.0] - 2026-11-19

### Added / 新增

- Task rows gained a 「删除」 action (`DELETE /api/ai/tasks/:id`) through the shared confirm dialog; a live running task is refused. The task list pages at 5 rows.
- 任务行新增「删除」（`DELETE /api/ai/tasks/:id`，走统一确认弹窗），执行中的任务由服务端拒绝；任务列表每页 5 条。

### Changed / 变更

- The review queue switched from 3-second polling to an SSE stream (`GET /api/ai/events`) with a fingerprint guard in `loadPaged` so idle pages rebuild zero DOM; polling remains as fallback and the refresh hint/buttons are removed.
- 待审队列由 3 秒轮询改为 SSE 推送（`GET /api/ai/events`），`loadPaged` 加指纹闸使空闲页零 DOM 重建；无 EventSource 退回轮询，刷新提示与按钮删除。
- Task error reasons moved from truncated inline text to a 「详情」 modal with selectable plain text.
- 任务错误原因不再截断内联，改由「详情」模态以纯文本完整展示。

## [5.13.0] - 2026-11-19

### Added / 新增

- Settings field labels carry a schema-driven 必填/选填 badge from the `required` flag.
- 设置页字段标签新增 schema 驱动的必填/选填徽章。

### Changed / 变更

- Entity names in confirm dialogs and toasts render as `.entity-chip` chips instead of «guillemets».
- 确认弹窗与 toast 里的实体名改渲染为 `.entity-chip` 芯片，不再用书名号。

### Fixed / 修复

- Switching the endpoint type no longer moves other settings rows: conditional fields sharing a `show_if` key swap in place inside per-ordinal slots.
- 切换端点类型不再带动其他配置行：共享 `show_if` 的条件字段按序位分槽、槽内原位互换。

## [5.12.0] - 2026-11-19

### Fixed / 修复

- The container runtime is Alpine with ffmpeg (plus ca-certificates, tzdata) instead of `FROM scratch`, so video AI recognition works in published images.
- 容器运行层由 `FROM scratch` 改为 Alpine + ffmpeg（含 ca-certificates、tzdata），已发布镜像里的视频 AI 识别可用。

## [5.11.0] - 2026-11-18

### Changed / 变更

- All nine native `<select>` sites mount a shared `selectMenu` Alpine component — an app-styled glass listbox with full keyboard/ARIA support — replacing the OS-drawn menu; dead native-select CSS is purged and the test `selectmenu.test.mjs` pins the contract.
- 九处原生 `<select>` 统一挂载共享 `selectMenu` Alpine 组件——应用自绘的玻璃 listbox，键盘与 ARIA 齐备；死原生 select 样式清除，`selectmenu.test.mjs` 钉住契约。

### Added / 新增

- Responsive layout: at ≤768px the sidebar becomes a hamburger drawer and data tables stack into cards via `data-label` (pinned by `TestStackedTableDataLabels`); row actions stay visible on touch, the detail page stacks and scrolls naturally, and a 1024px pass trims paddings.
- 响应式布局：≤768px 侧栏变汉堡抽屉、数据表按 `data-label` 堆叠成卡片（`TestStackedTableDataLabels` 防漏标）、触屏行操作常显、详情页堆叠后自然滚动；1024px 轻度档收窄留白。

### Fixed / 修复

- The tree sort dropdown's compact styling moved into the components layer, where it outranks the form-control base rule and actually applies.
- 树排序下拉的紧凑样式移入 components 层，层序压过表单基础规则后 2xs 紧凑外观真正生效。

## [5.10.1] - 2026-11-18

### Fixed / 修复

- Four convergence regressions patched: the tag-value update toast message, config option scanning, builder `preventDefault`, and `make fe-test`'s glob path.
- 补回四处回归：改标签值的 toast 文案、配置选项逐条遍历、构建器按钮 `preventDefault`、`make fe-test` 的 glob 路径。

### Added / 新增

- Regression tests pin the touched contracts (Go handler assertion + node:test store/paged cases).
- 为本次触及的契约补回归测试（Go 接口断言 + node:test store/分页用例）。

### Changed / 变更

- Per-field hint lines under config fields are deleted; usage copy lives in the ⓘ tooltip fed by `description`.
- 配置字段下方提示行删除；用法说明由 `description` 驱动的 ⓘ 气泡承担。

## [5.10.0] - 2026-11-18

### Changed / 变更

- Scalar design tokens landed: a seven-step type ramp, blur ramp, glow recipes, radius/transition tokens; input widths became four named tiers and table column widths and form help moved off inline styles.
- 标量设计 token 落地：七级字号阶梯、模糊阶梯、光晕配方、圆角/过渡 token；输入框宽度归为四档命名类，列宽与 form-help 离开内联样式。

## [5.9.0] - 2026-11-18

### Changed / 变更

- CSS componentized: a single `.panel` glass recipe, card/modal/toolbar/forms split into their files, `.toolbar` replacing ~10 ad-hoc flex toolbars; dead CSS purged and synonyms merged.
- CSS 组件化：玻璃面板配方单源 `.panel`，card/modal/toolbar/forms 分文件落地，约十处手写工具栏改挂 `.toolbar`；死 CSS 清除、同义类合并。

## [5.8.0] - 2026-11-18

### Changed / 变更

- JS idioms converged on shared core helpers: `confirmDelete`/`reportResultOrThrow`, `loadPaged`/`pagedList`/`clampToLastPage`, `delegate`/`createSelection`, `decodePath`/`fileDetailHref`/`readStoredJSON`/`readSortState`; a node:test harness (`make fe-test`) pins the contracts.
- JS 惯用式收进 core 共享件：`confirmDelete`/`reportResultOrThrow`、`loadPaged`/`pagedList`/`clampToLastPage`、`delegate`/`createSelection`、`decodePath` 等四族；node:test 基建 `make fe-test` 钉住契约。

### Fixed / 修复

- The file detail page keeps the user's sort across tag/group mutations (shared `API.fileDetail` + `applyDetail` path).
- 文件详情页在标签/组操作后不再丢用户排序（详情抓取共用一条路径）。

## [5.7.0] - 2026-11-18

### Changed / 变更

- The `empty` partial speaks a dict dialect and all 20 empty-state sites render from it; inline-edit cells collapsed into an `inlineEditCell` partial; dead inline styles purged.
- `empty` partial 改说 dict 方言，20 处空态全部收编；行内编辑单元格收进 `inlineEditCell` partial；死内联样式清扫。

### Fixed / 修复

- Inline edit rolls back to the pre-edit snapshot when the save is rejected.
- 行内编辑保存被拒后回滚到编辑前快照，不再恢复敲脏的实时值。

## [5.6.0] - 2026-11-18

### Fixed / 修复

- Four dark-theme hardcoded-light fixes: accent-surface text follows a new `--on-accent` token, danger hover glow and flash-toast borders mix from semantic tokens, and the select arrow became a theme-paired `--select-arrow` image token.
- 修复四处暗色主题写死浅色：accent 实底文字改用 `--on-accent` token，danger 光晕与 flash 边框改由语义色 `color-mix`，下拉箭头收进按主题配对的 `--select-arrow`。

## [5.5.0] - 2026-11-18

### Changed / 变更

- Form controls converged to single sources: one global checkbox rule, `.tag-clickable` for clickable chips, the `tagSearchField` partial for all autocomplete dropdowns, and a `sortControlsField` partial for both sort blocks.
- 表单控件归一单源：复选框一条全局规则、可点击 chip 共用 `.tag-clickable`、补全下拉统一 `tagSearchField` partial、两处排序控件收进 `sortControlsField` partial。

## [5.4.0] - 2026-11-18

### Changed / 变更

- All six pagers render from one `pager` partial on server envelope fields, with wording standardized to 「« 上一页 / 下一页 » (共 N 条)」 and one shared page-info color.
- 六处分页统一由 `pager` partial 渲染（服务端信封字段），文案归一「« 上一页 / 下一页 » (共 N 条)」、页码计数统一着色。

## [5.3.0] - 2026-11-18

### Added / 新增

- Second-round dedupe: `reportResult` absorbs ~20 toast-envelope copies, the form modal validates `required` centrally, `API.loadAllTagsSafe()` merges three fetch blocks, and `fe-check` gained a gate for uncalled partials (two dead partials removed).
- 二轮去重：`reportResult` 收编约 20 处 toast 样板、弹窗集中校验 `required`、`API.loadAllTagsSafe()` 合并三处同形拉取、`fe-check` 新增无调用 partial 门（两个死 partial 删除）。
- New `tagSearchField` and `searchForm` partials single-source the autocomplete block and the two list-page search forms; templates gained a `dict` helper for multi-parameter partial calls.
- 新增 `tagSearchField` 与 `searchForm` partial，补全块与两页搜索表单单源；模板侧新增 `dict` 多参辅助函数。

### Changed / 变更

- Creating a tag group with an empty name is blocked client-side with 「标签组名称不能为空」.
- 创建标签组留空改为本地拦截并提示「标签组名称不能为空」。

## [5.2.0] - 2026-11-18

### Added / 新增

- `make fe-check`: `node --check` over every JS file plus a two-way JS↔template reference check.
- 新增 `make fe-check`：全量 JS 语法检查 + JS↔模板双向引用一致性检查。

### Changed / 变更

- Sort controls became one shared `sortControls` component; `skipSummary` single-sourced; the review page's two polling lists merged into one `LISTS`-driven loader; search history formats sizes with the shared `formatSize`.
- 排序控件收敛到共享 `sortControls` 组件；`skipSummary` 单源；待审两张轮询表合并为 LISTS 驱动加载；搜索历史大小改用共享 `formatSize`。

## [5.1.0] - 2026-11-17

### Changed / 变更

- The settings page speaks a rule-based contract: `/api/config/schema` ships `rules`/`positional_rules`/`container_rules` with kind, role, params, hint, message and widget, replacing the old `type`/`item_type` fields.
- 设置页改用基于规则的契约：`/api/config/schema` 下发 `rules`/`positional_rules`/`container_rules`（kind/role/params/hint/message/widget），取代旧 `type`/`item_type` 字段。
- The frontend is a generic rule executor: local pre-validation through `ruleCheckers`, control effects driven by `widget`, unknown kinds ignored safely; every backend string comes from the single rule registry with zero copies.
- 前端成为通用规则执行器：`ruleCheckers` 本地预校验、控件效果由 `widget` 驱动、未知 kind 安全忽略；后端文案单源零副本。

## [5.0.0] - 2026-11-17

### Removed / 移除

- All gradient paint is gone: flat per-theme `--bg`, ambient glow circles and the stat-card specular sweep deleted, the skeleton pulses a solid fill, the favicon is flat teal; unused gradient variables removed. Glass blur, insets and grain are untouched.
- 清除全部渐变：背景改每主题纯色 `--bg`，环境光晕与扫光删除，骨架屏纯色脉冲，favicon 单色；无用渐变变量移除。玻璃模糊、inset 与噪点保持不变。
- The review enqueue card's two explanatory strings are removed; the destructive-action confirmation on the detail page stays.
- 待审页「发起识别」卡片删除两处说明文案；详情页破坏性操作的确认警告保留。

## [4.5.0] - 2026-11-16

### Changed / 变更

- The theme switched from indigo to teal (light `#0d9488`, dark `#2dd4bf`); hardcoded colors moved onto `--accent-*`/`--glow-*` variables so re-theming touches `variables.css` only.
- 主题由靛紫换为青碧（亮 `#0d9488`、暗 `#2dd4bf`）；硬编码色改引 `--accent-*`/`--glow-*` 变量，换主题只改 `variables.css`。

### Added / 新增

- Glass material completed per the design spec: layered background blobs, inset edge in `--glass-shadow`, a global grain overlay, and a hover specular sweep, with reduced-transparency and no-blur fallbacks.
- 按设计规格补齐玻璃材质：背景多层色斑、`--glass-shadow` 内嵌 inset、全局噪点层与 hover 扫光，含降低透明度/无模糊降级。

## [4.4.2] - 2026-11-16

### Changed / 变更

- The settings card and field markup moved into `config_group`/`config_field` partials and the collapse state into a `configGroup` Alpine component (same storage key); a new settings block is now a schema registration plus one template line.
- 设置页卡片与字段模板收进 `config_group`/`config_field` partial，折叠状态归 `configGroup` 组件（存储键不变）；新增配置板块只需注册 schema 加一行模板。

## [4.4.1] - 2026-11-16

### Fixed / 修复

- `make docker-build` failed copying `/usr/share/zoneinfo` from a tzdata-less builder; the builder now installs tzdata, and containers should pass `-e TZ=Asia/Shanghai`.
- `make docker-build` 在拷贝 `/usr/share/zoneinfo` 处失败（builder 无 tzdata）；现由 builder 安装 tzdata，部署容器加 `-e TZ=Asia/Shanghai`。

### Changed / 变更

- The 「高级 / 自定义配置」 card is removed from the settings page; unknown keys stay readable/writable through the API and CLI.
- 设置页删除「高级 / 自定义配置」卡片；未注册键仍走 API 与 CLI。
- 「恢复默认」 became one confirmed, local, per-group button in the card footer; the per-field key chip and default/range hint line are removed.
- 「恢复默认」改为分组级按钮（确认后本地重置，不保存不落库）；字段行键名 chip 与默认/范围提示行删除。

## [4.4.0] - 2026-11-16

### Added / 新增

- `GET /api/config/schema` answers the whole settings page in one read: fields, declared constraints, live values, `invalid`/`invalid_reason` for stored-but-illegal values, and masked secrets (`has_value` only). `POST /api/config/group` saves one group all-or-nothing with per-key `failures`.
- `GET /api/config/schema` 一次读出整页配置：字段、声明约束、实时取值、非法值标记与原因、secret 掩码（只回 `has_value`）；`POST /api/config/group` 分组全有全无保存并逐键返回失败。
- The settings page is rebuilt schema-driven: per-type controls, ⓘ tooltips, collapsible group cards with invalid-count badges, a regex row editor for `scan_exclude`, and per-group save with rejected fields marked by key.
- 设置页改由 schema 驱动重建：按类型渲染控件、ⓘ 提示、可折叠分组带非法计数徽标、`scan_exclude` 正则行编辑器、分组级保存且按键标出非法字段。

### Changed / 变更

- `GET /api/config` items carry `description`; `?unknown=1` lists only registry-unknown keys via service filters; registered secret values are masked in the list view.
- `GET /api/config` 条目新增 `description`；`?unknown=1` 经服务层过滤只列未注册键；已注册 secret 值在列表视图掩码。
- Known keys are edited only through their schema group; the advanced area holds unknown keys; an untouched secret survives a group save (server treats omission as leave-unchanged).
- 已知键只经各自 schema 分组编辑；高级区只列未注册键；未改动的 secret 在分组保存后保留（服务端把省略视为不变）。

## [4.3.0] - 2026-11-14

### Added / 新增

- Seven AI endpoints under `/api/ai`: task create/list/retry and suggestion list/update/approve/reject.
- `/api/ai` 分组新增 7 个接口：任务创建/列表/重试，建议列表/修改/通过/拒绝。
- `POST /api/ai/tasks` takes repeatable `paths` or one `dir` (+`recursive`, optional `force`) and answers `{queued, skipped}`; it only enqueues — recognition runs in a background worker started with the server, whose start sweep marks orphaned `running` rows 重启中断.
- `POST /api/ai/tasks` 接受可重复 `paths` 或单个 `dir`（配 `recursive`、可选 `force`），返回 `{queued, skipped}`；只入队，识别由随服务启动的后台 worker 执行，启动清扫把遗留 running 行标为 重启中断。
- `GET /api/ai/tasks` and `GET /api/ai/suggestions` list pages with status filters; each suggestion keeps the model's original wording and the reviewer's final values side by side; approve/reject take repeatable `ids` and report per-row `{id, msg}` failures without dragging the batch down.
- 任务与建议分页列表接口（状态过滤）；建议同时携带模型原样输出与审核者终值；批量通过/拒绝逐条上报 `{id, msg}` 失败、不拖垮整批。
- New 「待审队列」 page at `/review`: enqueue card (directory, recursive, force), auto-refreshing task table with per-row retry, and a suggestion table with checkboxes, inline editing, and batch approve/reject.
- 新增 `/review` 待审队列页：发起识别卡片（目录、递归、强制重跑）、自动刷新并可逐行重试的任务表、支持勾选/行内编辑/批量通过拒绝的建议表。
- The config page gained an 「AI 识别」 card for `ollama_url`, `ollama_model`, `ai_frame_interval`, `ai_frame_diff_threshold` and `ai_max_frames`, with load-gated saving.
- 配置页新增「AI 识别」卡片（识图地址/模型、抽帧间隔/阈值/最大帧数），保存以读取完成为前提。
- File tree rows and the detail page render AI status badges (识别中/待审/AI 通过/识别失败); indexed media rows gained checkboxes with a batch 「识别选中」 button, and the detail page offers 「AI 识别」/「重新识别」 with an optimistic badge update.
- 文件树与详情页渲染 AI 状态徽章（识别中/待审/AI 通过/识别失败）；已索引媒体行新增勾选与批量「识别选中」按钮；详情页提供「AI 识别」/「重新识别」并乐观更新徽章。

### Changed / 变更

- `FileItem` and `FileTreeNode` gained the optional `ai_status` key (`omitempty`); no existing key changed.
- `FileItem`/`FileTreeNode` 新增可选 `ai_status`（`omitempty`），既有键不变。

## [4.2.5] - 2026-11-13

### Changed / 变更

- File detail and tag-group wildcard paths share one normalizer that strips gin's leading slash before URL-decoding; regression tests cover both endpoints.
- 文件详情与标签组通配路径共用同一个规范化函数，在 URL 解码前剥离 gin 添加的前导斜杠；两个端点补回归测试。

## [4.2.4] - 2026-11-12

### Added / 新增

- New 「随机打标」 page at `/files/recommend`: draws a random file and opens the full tagging UI in place, with a scope selector (待完成打标/尚无标签/全部文件), 换一个, and pool/remaining/shown readout; already-shown files are remembered in `sessionStorage` (cap 500) and skipped until the pool exhausts.
- 新增「随机打标」页 `/files/recommend`：随机抽文件就地完整打标，带范围选择（待完成打标/尚无标签/全部文件）、「换一个」与池量/剩余/已推荐显示；本次已推荐记入 `sessionStorage`（上限 500）并跳过，直至候选耗尽后重开。
- `GET /api/files/random?scope=&exclude=` returns the pick plus `pool_total`, `remaining`, `exhausted`; an empty scope answers `ok` with `file:null`. Integration tests cover the endpoint.
- 新增 `GET /api/files/random?scope=&exclude=`，返回抽中文件与 `pool_total`/`remaining`/`exhausted`；空范围回 `ok` + `file:null`。配集成测试。

### Changed / 变更

- The detail component takes a recommend flag; `/files/recommend` and `/files/detail/*path` share one template and one JS module.
- 详情页组件新增 recommend 标志，`/files/recommend` 与 `/files/detail/*path` 共用同一模板与 JS 模块。

## [4.2.3] - 2026-11-12

### Fixed / 修复

- File tree lazy expansion works again: the re-render hook watched the never-changing `treeHtml` method; it now watches `tree`, `sortField` and `sortOrder`.
- 文件树懒加载恢复：重渲染钩子原盯永不变化的方法 `treeHtml`，改盯 `tree`/`sortField`/`sortOrder`。
- The config export link points at `/api/export` (the old `/export` 404s).
- 导出链接改指 `/api/export`（原 `/export` 404）。
- The `FilesystemRoutes` smoke test scans inside the temporary workspace, matching the core's path basis.
- `FilesystemRoutes` smoke 测试改在临时工作目录内扫描，对齐核心层路径基准。

### Changed / 变更

- Dockerfile base image bumped to `golang:1.25-alpine` (the 1.24 image pins `GOTOOLCHAIN=local` and fails the build).
- Dockerfile 基础镜像升为 `golang:1.25-alpine`（1.24 镜像锁 `GOTOOLCHAIN=local` 会构建失败）。

## [4.2.2] - 2026-11-11

### Fixed / 修复

- DELETE endpoints (`/api/files/tag`, `/api/files/group/:id`, `/api/tags/values/:id`) read parameters from the URL query again; `c.PostForm` never parses a DELETE body, which had silently broken untagging, group removal, and value removal.
- DELETE 接口（`/api/files/tag`、`/api/files/group/:id`、`/api/tags/values/:id`）恢复从 URL query 取参；`c.PostForm` 不解析 DELETE body，此前取消打标/删组/删预设值全部静默失败。
- Wildcard routes strip the leading slash gin adds to `c.Param("path")`, restoring the file detail and tag group pages.
- 通配路由剥掉 gin 给 `c.Param("path")` 附加的前导斜杠，详情页与标签组页恢复。

### Added / 新增

- First WebUI API integration tests: DELETE parameter contract, wildcard path resolution, and a smoke net over every registered `/api` route.
- WebUI API 首批集成测试：DELETE 参数契约、通配路径解析、覆盖全部 `/api` 路由的 smoke 防护网。

## [4.2.1] - 2026-08-29

### Fixed / 修复

- Tag management page right panel no longer overflows its card background when viewport height is short.
- 标签管理页面右侧面板在视口高度较小时不再超出卡片背景。

## [4.2.0] - 2026-08-29

### Fixed / 修复

- `HandleTagFile`/`HandleUntagFile` no longer silently fall back to the default group when a specified group fails; the error is returned.
- `HandleTagFile`/`HandleUntagFile` 指定标签组失败时不再静默回退默认组，错误返回给用户。
- `HandleRemoveTag` deletes by tag ID instead of name, avoiding cross-category same-name deletion.
- `HandleRemoveTag` 按标签 ID 删除而非名字，避免跨分类同名误删。
- `HandleRemoveCategory`, `HandleAddTag`, `HandleListTags` use `FindByID` instead of loading all categories.
- `HandleRemoveCategory`、`HandleAddTag`、`HandleListTags` 改用 `FindByID`。

### Changed / 变更

- Tag path building loads only the needed tags (target + ancestors) in detail, search, and flat list.
- 标签路径构建只加载所需标签（目标 + 祖先）。
- Search uses SQL-level pagination (`SearchByQueryPaged`) instead of in-memory slicing.
- 搜索改用 SQL 层分页（`SearchByQueryPaged`）。

## [4.1.0] - 2026-08-29

### Changed / 变更

- The file tree builds from the database on load; unindexed files are discovered lazily per directory expansion via `GET /api/files/expand?dir=xxx` (backed by `file.Service.ExpandDir()`).
- 文件树从数据库构建；未索引文件在展开目录时经 `GET /api/files/expand?dir=xxx` 懒加载（`file.Service.ExpandDir()`）。

## [4.0.0] - 2026-08-28

### Changed / 变更

- HTTP layer migrated from `net/http` to Gin with route groups, Logger and Recovery middleware; CLI flags now parse with cobra.
- HTTP 层从 `net/http` 迁移到 Gin（路由分组、Logger、Recovery）；命令行参数改用 cobra。

## [3.3.0] - 2026-08-16

### Added / 新增

- Directory rows show a nested done/total counter (e.g. 「✓2/4」) aggregating the whole subtree.
- 目录行显示子树聚合的已完成/总数计数（如「✓2/4」）。

## [3.2.9] - 2026-08-16

### Changed / 变更

- The two tag-tree drag zones are side by side in one fixed row above the scrolling list instead of stacked around the tree.
- 标签树两个拖拽区域并排固定在滚动列表上方一行，不再上下分列随树滚动。

## [3.2.8] - 2026-08-16

### Changed / 变更

- Only the tag list/tree area scrolls on the tag page; the controls row and inline-add form stay fixed.
- 标签页只有列表/树区域滚动，控件行与内联添加表单固定。

## [3.2.7] - 2026-08-16

### Fixed / 修复

- The `tagSearch` component gained an optional `reset` getter; the file detail add-tag input clears after a successful add.
- `tagSearch` 组件新增可选 `reset` getter；文件详情加标签输入框在添加成功后清空。

## [3.2.6] - 2026-08-16

### Fixed / 修复

- Media elements bind `src`/`type` directly, so switching video files actually reloads; sidebar `x-for` expressions got null guards.
- 媒体元素直接绑定 `src`/`type`，切换视频文件真正重载；侧栏 `x-for` 表达式加 null 守卫。

## [3.2.5] - 2026-08-16

### Fixed / 修复

- The tag page inline-add form listens for `tag-selected` on the `x-data` root (where `tagSearch` dispatches), so parent-tag selection works.
- 标签页内联添加表单在 `x-data` 根元素监听 `tag-selected`（事件派发处），父标签选择恢复生效。

## [3.2.4] - 2026-08-16

### Fixed / 修复

- The modal store normalizes `readonly`/`required` to explicit booleans, so Alpine no longer coerces `undefined` to a truthy `""` and locks every field.
- 弹窗 store 把 `readonly`/`required` 规范化为显式布尔，Alpine 不再把 `undefined` 强转成真值 `""` 锁死所有字段。

## [3.2.3] - 2026-08-16

### Fixed / 修复

- Inline edit compares against the init-time original value, so save actually fires.
- 行内编辑与初始化时的原值比较，保存真正发起。

## [3.2.2] - 2026-08-16

### Fixed / 修复

- Tag preview chips align with the 标签 label in the detail info table.
- 文件详情信息表的标签预览芯片与「标签」垂直对齐。
- `onTagPicked` no longer double-reads `evt.detail`.
- `onTagPicked` 不再重复取 `evt.detail`。
- Toggling 标记完成 immediately syncs the sidebar ✓ badge.
- 切换「标记完成」实时同步侧栏 ✓ 徽标。

## [3.2.1] - 2026-08-16

### Added / 新增

- The WebUI version constant lives in `cmd/webui/constant.WebuiVersion`.
- WebUI 版本常量集中于 `cmd/webui/constant.WebuiVersion`。

### Fixed / 修复

- The detail sidebar and content panel scroll independently; the outer `.main` no longer scrolls.
- 详情页侧栏与内容面板独立滚动，外层 `.main` 不再滚动。

## [3.2.0] - 2026-08-16

### Added / 新增

- Tag tree collapse carets with state persisting across re-renders, plus 全部折叠/全部展开 buttons.
- 标签树新增折叠三角（状态跨重渲染保持）与「全部折叠/全部展开」按钮。

## [3.1.0] - 2026-08-16

### Changed / 变更

- Tag tree drag is target-highlight mode: the DOM never mutates mid-drag, hovering highlights the drop parent, and fixed zones handle root drops and cancellation.
- 标签树拖拽改为目标高亮模式：拖动过程 DOM 零变动，悬停高亮落点父节点，根层放置与取消走固定区域。

## [3.0.3] - 2026-08-16

### Changed / 变更

- The drag insertion indicator became a 4px accent line, and a 「拖到这里取消移动」 zone below the tree reverts a move without an API call.
- 拖拽落点指示改为 4px 主色细线，树下新增「拖到这里取消移动」区，拖入即撤销且不发起 API。

## [3.0.2] - 2026-08-16

### Fixed / 修复

- The drag handle is always visible and every node renders an empty children drop zone, so tags can be dragged onto leaves.
- 拖拽把手常显、每个节点渲染空子列表放置区，可把标签拖成叶子的子标签。

## [3.0.1] - 2026-08-16

### Fixed / 修复

- Drag drop targets resolve from SortableJS `evt.to` (the list entered) instead of `elementFromPoint`, so dropping into the root list makes a tag top-level.
- 拖拽落点改按进入的列表（SortableJS `evt.to`）判定而非坐标命中，放入根列表即成为顶级标签。

## [3.0.0] - 2026-08-16

### Changed / 变更

- Full refactor onto Alpine.js 3.14 + SortableJS 1.15 (vendored, no build step): declarative components on every page, Alpine stores for toast/confirm/modal, shared `tagSearch`/`inlineEdit`; UI unchanged.
- WebUI 全面重构到 Alpine.js 3.14 + SortableJS 1.15（本地 vendor、无构建）：各页面声明式组件化，toast/confirm/modal 改 Alpine store，`tagSearch`/`inlineEdit` 抽共享件；UI 外观不变。

### Fixed / 修复

- Creating a tag group no longer fails on gorm's ErrRecordNotFound being treated as fatal.
- 创建标签组不再因 gorm ErrRecordNotFound 被误判致命而失败。

## [2.2.0] - 2026-08-15

### Added / 新增

- Mark files as tagging-done: `POST /api/files/done`, the detail toggle button, tree 「已完成」 badge and sidebar ✓; `tag_done` joins `FileItem`/`FileTreeNode`.
- 打标完成标记：`POST /api/files/done`、详情页切换按钮、树「已完成」徽标与侧栏 ✓；`FileItem`/`FileTreeNode` 新增 `tag_done`。

## [2.1.1] - 2026-07-12

### Fixed / 修复

- File tree items no longer stay invisible under the stagger rule (missing `animation` on `.tree-file`/`.tree-dir`).
- 补上 `.tree-file`/`.tree-dir` 缺失的 `animation`，文件树不再整列隐形。

### Added / 新增

- Shared utilities land in `core/utils.js` and `core/sort-controls.js`: `clearChildren`, `staggerChildren`, `initSortControls`, `API.getAllPages`, `API.loadAllTags`, `API.onError`, and `GET /api/tags/values/batch` (N requests → 1 on the tag page).
- `core/utils.js` 与 `core/sort-controls.js` 落地共享工具：`clearChildren`、`staggerChildren`、`initSortControls`、`API.getAllPages`、`API.loadAllTags`、`API.onError`，新增 `GET /api/tags/values/batch`（标签页 N 次请求降为 1 次）。

### Changed / 变更

- Search builder initialization moved to a configuration pattern, fixing the lost autocomplete on restored history inputs.
- 搜索构建器初始化改配置模式，修复历史恢复输入框丢失自动补全的竞态。
- Error handling standardized on `.finally()` + `.catch(API.onError)`; raw `fetch()` calls replaced with the `API` wrapper.
- 错误处理统一 `.finally()` + `.catch(API.onError)`；raw `fetch()` 改走 `API` 封装。

### Removed / 移除

- Ten dead legacy stub functions deleted.
- 删除十个废弃遗留桩函数。

## [2.0.3] - 2026-07-12

### Added / 新增

- SVG favicon served with the correct `image/svg+xml` content type.
- SVG 网站图标，静态服务正确返回 `image/svg+xml`。

## [2.0.2] - 2026-07-12

### Fixed / 修复

- The detail TOC sidebar no longer overlaps the page title and scrolls independently.
- 详情页 TOC 侧栏不再遮挡标题，且可独立滚动。

## [2.0.1] - 2026-07-12

### Fixed / 修复

- Sticky page title with glass background; sidebar nav switched to pill-style active tabs.
- 页标题 sticky 固定 + 玻璃底；侧栏导航改胶囊激活态。

## [2.0.0] - 2026-07-12

### Added / 新增

- Glassmorphism theme across all components, a dark/light toggle with localStorage and system preference, CSS Layers, ambient radial glows, and an indigo accent replacing steel blue.
- 玻璃拟态主题覆盖全组件，深浅色切换（localStorage + 跟随系统），CSS Layers，环境径向光晕，强调色由钢蓝改靛蓝。

### Changed / 变更

- Global radius raised to 8px; background switched to a multi-stop gradient; buttons, inputs, and scrollbars re-styled in the glass language.
- 全局圆角升至 8px；背景改多段渐变；按钮、输入框、滚动条统一玻璃语言。

## [1.3.0] - 2026-07-12

### Added / 新增

- Structured query builder for search: file name, tag, tag value, size, extension, and date conditions with AND/OR and nested groups; `POST /api/search` now takes the `query` JSON (breaking change: `include`/`exclude`/`logic` removed); history restores full builder state.
- 搜索结构化条件构建器：文件名、标签、标签值、大小、扩展名、日期，AND/OR 与嵌套分组；`POST /api/search` 改收 `query` JSON（不兼容变更：移除 `include`/`exclude`/`logic`）；历史可恢复完整构建器状态。

## [1.2.7] - 2026-06-01

### Added / 新增

- `POST /api/search` JSON endpoint with pagination, tag autocomplete on both inputs, result paging, and a localStorage search history (max 10) rendered as chips.
- `POST /api/search` JSON 端点：分页搜索、双输入框标签自动补全、结果翻页、localStorage 搜索历史（最多 10 条）芯片展示。

### Changed / 变更

- The search page became AJAX client-side rendering; the server-side search handler was replaced by `api/search.go`, leaving the handler layer a pure template shell.
- 搜索页改 AJAX 客户端渲染；服务端搜索 handler 由 `api/search.go` 替代，handler 层成为纯模板壳。

## [1.2.6] - 2026-05-17

### Changed / 变更

- Tag page sidebar became a category filter list with counts; the category dropdown was removed.
- 标签页侧栏改为带计数的分类筛选列表，移除分类下拉。
- Tag page inline styles moved to their CSS files; dead tree CSS removed; panels gained entrance animation.
- 标签页内联样式迁入 CSS 文件，死树样式清除，面板加入场动画。
- File tree sorting now orders directories at each level with a name tiebreaker; sort state persists via sessionStorage; detail navigation between files updates in place via the History API; missing files show a dedicated empty state.
- 文件树目录参与层级排序（名称作次级键）；排序状态 sessionStorage 持久化；同目录切换文件经 History API 原地更新；文件不存在显示专门空态。

## [1.2.5] - 2026-05-17

### Added / 新增

- File list sorting on the tree page (client-side per level) and detail sidebar (server-side `?sort=&order=`), with a field dropdown and order toggle.
- 文件列表排序：树页面客户端层级排序、详情侧栏服务端 `?sort=&order=`，字段下拉 + 方向切换。
- The file info card's tag preview and count refresh after tag CRUD; sidebar file items fade in staggered.
- 标签 CRUD 后文件信息卡预览区与计数同步刷新；侧栏文件项依次淡入。

## [1.2.4] - 2026-05-17

### Fixed / 修复

- Buttons next to full-size inputs no longer use `btn-sm` (height mismatch); the import browse button aligns with its input.
- 与正常尺寸输入框相邻的按钮去掉 `btn-sm`（高度不齐）；导入浏览按钮与输入框对齐。

## [1.2.3] - 2026-05-17

### Changed / 变更

- The groups page became click-to-edit rename with only 删除 left as a row action.
- 标签组页组名改为点击单元格编辑，行操作只留「删除」。

## [1.2.2] - 2026-05-17

### Changed / 变更

- Confirm messages use literal `«` `»` characters instead of HTML entities.
- 确认消息直接用 `«` `»` 字符，不再经 HTML 实体。

## [1.2.1] - 2026-05-17

### Fixed / 修复

- `showConfirm` decodes HTML entities before setting text.
- `showConfirm` 先解码 HTML 实体再设置文本。

## [1.2.0] - 2026-05-17

### Changed / 变更

- All frontend JS rebuilt on `createElement` — zero innerHTML string concatenation — with new `createEl`, `makeEmptyState`, `makePagination`, `makeActionBtn` helpers and 13 page files refactored; tree rendering appends DOM directly.
- 前端 JS 全面改 `createElement` 构建（零 innerHTML 拼接），新增 `createEl`、`makeEmptyState`、`makePagination`、`makeActionBtn` 工具，13 个页面文件重构；树渲染直接追加 DOM。

## [1.1.1] - 2026-05-17

### Changed / 变更

- Inline edit rewritten around a permanent `<input readonly>`: click to edit, no DOM replacement, no layout shift; applied to category, tag value, and group rename, dropping their per-row edit buttons.
- 行内编辑改为常驻 `<input readonly>`：点击即编辑、不换 DOM、无排版跳动；应用于分类、标签值、组重命名并去掉对应行内编辑按钮。
- The redundant 「详情」 button is gone next to indexed tree files (the name already links).
- 已索引文件行去掉冗余「详情」按钮（文件名本身即链接）。

## [1.1.0] - 2026-05-17

### Changed / 变更

- A global singleton form modal (`Modal.show`) replaced innerHTML editing across all 6 CRUD modules; `makeEditable` brought click-to-edit to single-field cells; all forms submit via AJAX with zero page reloads; the detail page reloads only groups + tags after mutations.
- 全局单例表单弹窗（`Modal.show`）替代 6 个 CRUD 模块的 innerHTML 编辑；`makeEditable` 提供单字段点击编辑；全部表单 AJAX 提交零刷新；详情页变更后只刷新标签组与标签。
- Tag preset values and group creation moved from `prompt()`/inline forms to the modal.
- 标签预设值与创建标签组从 `prompt()`/内联表单改走弹窗。

### Fixed / 修复

- DELETE endpoints moved their parameters to the URL query (Go ignores DELETE bodies).
- DELETE 接口参数改走 URL query（Go 不解析 DELETE body）。

## [1.0.1] - 2026-05-17

### Changed / 变更

- Static assets reorganized: CSS split into core/layout/components, JS into core/pages, and `core/utils.js` extracts six shared helpers.
- 静态资源重组：CSS 分 core/layout/components，JS 分 core/pages，`core/utils.js` 提取六个共享工具函数。

## [1.0.0] - 2026-05-17

### Added / 新增

- `api/` package: 29 RESTful `/api/...` endpoints (categories, tags, tag values, files, config, export) with a consistent `{ok,msg}` envelope; per-page JS controllers and a shared `API` fetch wrapper + toast.
- 新增 `api/` 包：29 个 RESTful `/api/...` 端点（分类、标签、标签值、文件、配置、导出），统一 `{ok,msg}` 信封；各页面 JS 控制器 + 共享 `API` 封装与 toast。

### Changed / 变更

- Handler/API separation: page handlers render empty shells, all data access moved to `api/`; templates dropped server-side rendering (~1000 lines); CRUD routes became RESTful by HTTP method.
- 前后端分离：页面 handler 只渲染空壳，数据访问全部移入 `api/`；模板移除服务端渲染（约千行）；CRUD 路由改 RESTful HTTP 方法分发。

### Removed / 移除

- Five data-handling handler files, the flash/redirect error passing, and four server-side sub-templates.
- 删除五个数据处理 handler 文件、flash/redirect 错误传递与四个服务端子模板。

## [0.5.1] - 2026-05-17

### Changed / 变更

- All POST submissions moved from POST-Redirect-GET to `fetch()` + JSON `respond()`; the URL-param flash system and `_return`/`ajax=1` template hacks are deleted; the category list refreshes via `?partial=list`.
- 所有 POST 提交从 POST-Redirect-GET 迁移到 `fetch()` + JSON `respond()`；删除 URL 参数 flash 系统与 `_return`/`ajax=1` 模板补丁；分类列表经 `?partial=list` 局部刷新。

### Fixed / 修复

- Non-ASCII flash text no longer breaks cookie encoding.
- 非 ASCII flash 文本不再引发 cookie 编码错误。

## [0.5.0] - 2026-05-17

### Added / 新增

- Editorial redesign: steel-blue palette, serif/sans typography, split static CSS, staggered card reveals, toast system, custom confirm modal, dual-panel tag page with drag re-parenting, hover-revealed row actions, enhanced empty states, `Ctrl+K` search focus.
- Editorial 风格重设计：钢笔墨蓝色系、衬线/黑体排版、静态 CSS 分模块、卡片依次渐入、toast 通知、自定义确认弹窗、标签页双面板 + 拖拽改父级、悬停显示行操作、强化空态、`Ctrl+K` 聚焦搜索。

### Fixed / 修复

- Static file route fixed for Go 1.22+ path parameters.
- 静态文件路由适配 Go 1.22+ 路径参数语法。

## [0.4.5] - 2026-05-05

### Added / 新增

- Focus returns to the add-tag input after tagging; directory rows show an aggregated tag-count badge; unsupported video formats show a notice instead of a broken player.
- 加标签后焦点回到输入框；目录行显示聚合标签数徽章；不支持的视频格式显示提示而非空播放器。

### Changed / 变更

- Tree expand/collapse persists via sessionStorage; the tag search input clears `tag_id` on typing and auto-selects the first candidate on Tab.
- 目录树展开状态 sessionStorage 持久化；标签搜索输入时清 `tag_id`、Tab 自动选第一候选。

## [0.4.4] - 2026-05-05

### Changed / 变更

- Tag CRUD became AJAX with partial DOM replacement (scroll preserved); detail tag edits sync the preview strip and count.
- 标签 CRUD 改 AJAX + DOM 局部替换（保持滚动位置）；详情页标签编辑同步预览区与计数。

### Fixed / 修复

- Flat view shows full tag paths; deletion keeps category/view URL params; detail tags dedupe by display text and show parent paths; template-escaped pagination callbacks fixed.
- 平铺视图显示完整标签路径；删除保留 URL 筛选参数；详情标签按展示文本去重并显示父路径；模板转义导致的分页回调失效修复。

## [0.4.3] - 2026-05-05

### Changed / 变更

- Button CSS tidied: `.btn` base, `.btn-xs`, `.badge`, hover states for all variants.
- 按钮样式整理：`.btn` 基类、`.btn-xs`、`.badge`、各变体 hover 态。

## [0.4.2] - 2026-05-05

### Added / 新增

- The add-tag form preselects the category from `?category_id` and loads parent options.
- 添加标签表单按 `?category_id` 预选分类并加载父标签选项。

### Fixed / 修复

- `Preload("Category")` restored in the tree finders (blank category names in tree view).
- 树形查询恢复 `Preload("Category")`（修复树视图分类名为空）。

## [0.4.1] - 2026-05-05

### Added / 新增

- Tag-count badges in the file tree and the detail sidebar.
- 文件树与详情侧栏的标签数徽章。

## [0.4.0] - 2026-05-05

### Added / 新增

- Tag groups: detail-page group tabs, the `/files/groups/{path}` management page, AJAX create/rename/delete, and `group_id`-scoped tag/untag.
- 标签组：详情页组 Tab、`/files/groups/{path}` 管理页、AJAX 建组/改名/删组、打标/去标按 `group_id` 限定。

### Fixed / 修复

- Adapted to the core `FileTag` → `TagGroupTag` refactor, adding `UpdateTagValue`.
- 适配核心 `FileTag` → `TagGroupTag` 重构，新增 `UpdateTagValue`。

## [0.3.0] - 2026-05-04

### Added / 新增

- Shared partials (`pagination`, `empty`, `tagBadge`), a shared JS lib (`flashMsg`, `confirmAction`, `setupTagSearch` with keyboard navigation), server-preloaded `allTags`, and `cat`/`cid` fields on `/api/tags`.
- 共享 partial（`pagination`/`empty`/`tagBadge`）、共享 JS 库（`flashMsg`、`confirmAction`、带键盘导航的 `setupTagSearch`）、服务端预载 `allTags`、`/api/tags` 附 `cat`/`cid` 字段。

### Changed / 变更

- All pages parse all partials and switched their pagination/empty states/datalists to the shared components; embed directive extended to `templates/partials/`.
- 各页面统一解析全部 partial，分页/空态/datalist 改用共享组件；embed 指令扩到 `templates/partials/`。

## [0.2.1] - 2026-05-04

### Added / 新增

- A custom filtered tag dropdown (replacing `<datalist>`), searchable parent selectors with tag-path display, quick-add child in tree view, and a redesigned detail tag table with inline value editing.
- 自定义过滤标签下拉（替代 `<datalist>`）、可搜索父标签选择器（路径展示）、树视图快速加子标签、详情页标签表重做支持行内改值。

### Changed / 变更

- The tag page defaults to tree view; `Redirect()` preserves filter state via the `_return` field; repository queries preload Category/Parent.
- 标签页默认树形；`Redirect()` 经 `_return` 保留筛选状态；仓储查询预加载 Category/Parent。

### Fixed / 修复

- Category dropdown selection, edit-value duplicates, pagination/form reload breaks, video reset on tagging, spaced file paths, and flash param accumulation.
- 修复分类下拉选中态、改值产生重复、分页/表单整页刷新、打标导致视频重置、含空格路径、flash 参数累积。

## [0.2.0] - 2026-05-04

### Removed / 移除

- The attribute module (`handlers/attribute.go`, its template, routes, nav entry).
- 移除属性模块（handler、模板、路由与导航入口）。

### Added / 新增

- Tag tree hierarchy with a flat/tree toggle, parent-tag dropdowns, `GET /api/tags`, tag preset value management, value-aware file tagging (`标签:值`), and a category filter dropdown on the tag list.
- 标签树层级 + 平铺/树形切换、父标签下拉、`GET /api/tags`、标签预设值管理、带值打标（`标签:值`）、标签列表分类筛选下拉。

### Fixed / 修复

- `scan_exclude` respected in the tree view; category inline-edit save, double WriteHeader warnings, and recursive template errors.
- 树视图遵循 `scan_exclude`；修复分类行内编辑保存、重复 WriteHeader 告警与递归模板错误。

## [0.1.1] - 2026-05-04

### Added / 新增

- The directory tree file list (collapsible, indexed/unindexed markers), `webui.hide_extensions` config, an import file picker, a tree cache in `.tagfile/webui_tree.json`, inline edit + search on list pages, a rename form, and the detail-page directory sidebar.
- 目录树文件列表（可折叠、索引状态标记）、`webui.hide_extensions` 配置、导入文件选择器、`.tagfile/webui_tree.json` 树缓存、列表页行内编辑与搜索、重命名表单、详情页目录侧栏。

### Changed / 变更

- Tagging moved to the path-based detail page `/files/detail/{path...}`; the flat table list was dropped.
- 打标操作迁到路径制详情页 `/files/detail/{path...}`；移除平铺表格列表。

### Fixed / 修复

- Flash messages encode correctly; media files stream with HTTP Range (seeking works).
- 修复 flash 消息编码；媒体支持 HTTP Range 拖动进度。

## [0.1.0] - 2026-05-03

### Added / 新增

- The `cmd/webui/handlers/` package, a dedicated file detail route with inline media playback, `GET /files/raw`, and `--source`/`--addr` flags.
- 新增 `cmd/webui/handlers/` 包、文件详情独立路由（内联播放）、`GET /files/raw`、`--source`/`--addr` 参数。

### Changed / 变更

- `main.go` (900 lines) split into a 9-file handlers package; pagination set to 10/page with a clamping fix; shared helpers extracted.
- `main.go`（900 行）拆为 9 文件的 handlers 包；分页每页 10 条并修复钳制 bug；提取共享 helper。
