# Changelog (Web UI) / 更新日志 (Web UI)

tagfile 项目 Web 界面的版本变更记录。格式参考 [Keep a Changelog](https://keepachangelog.com)。

## [5.23.0] - 2026-11-22

### Changed / 变更

- The workbench sidebar header carries only the sort control; the current file's place is told by its own row, and every detail landing expands the ancestors and scrolls that row into view — switching files follows along the same tree. The SSE detail fingerprint tracks rendered fields only.
- 工作台侧栏头部只留排序控件；当前文件的位置由它自己那一行说明，每次详情落位自动展开祖先目录并把该行滚进视口，站内切换文件时侧栏同步跟随。SSE 详情指纹只盯渲染中的字段。
- The current-file marker is one full-bleed band in the category sidebar's selection language: it glides between rows (0.25 s) and re-seats onto the row after folds, lazy appends, refreshes, and re-renders. In-place row repaints (tag/done/AI) keep content without replaying the row fade-in.
- 当前文件标记是一条通栏色带，与分类侧栏选中态同一套视觉语言：在行间平移落位（0.25s），折叠、懒追加、刷新、重渲染后都重新贴合所在行。打标/完成/AI 的就地重画只换内容，不重放行级淡入。

### Fixed / 修复

- `decodePath` is the exact inverse of `encodePath` (`decodeURIComponent`), so a file opened from a URL with reserved characters (`#`, `,`, `?`) resolves to the tree row's path and highlights correctly.
- `decodePath` 是 `encodePath` 的严格逆运算（`decodeURIComponent`）：带保留字符（`#`、`,`、`?`）的 URL 直开文件能落回与树行一致的路径，高亮定位正常。

## [5.22.0] - 2026-11-22

### Added / 新增

- The directory sidebar is now resizable: a slim col-resize handle sits in the seam between the sidebar and the main area on both pages that carry one (the workbench detail and the tag page). Dragging sets any width between 180 and 480 px (clamped live); releasing persists it — the preference lives in localStorage (`tagfile_dir_width`) and lands as a root CSS variable, so both pages share one width and it survives restarts. Double-click or Enter restores the default, arrow keys step 16 px, and the handle hides in the stacked narrow-screen layout. The wider sidebar is the counterpart to 5.21.0's tree: long directory and file names get more room before the ellipsis bites.
- 目录侧栏现在可以调宽窄：两个带侧栏的页面（工作台详情、标签页）在侧栏与主区的缝隙里放了一条 col-resize 细把手——拖动实时设定 180–480px 之间的宽度（越界夹住），松手即持久化：偏好存 localStorage（`tagfile_dir_width`）并写进根元素 CSS 变量，两页天然共用同一份宽度、跨会话保留。双击或 Enter 恢复默认，方向键步进 16px，窄屏堆叠布局下把手自动隐去。更宽的侧栏正是 5.21.0 目录树的配套：长目录名/文件名在省略号生效前能多露一截。

## [5.21.0] - 2026-11-21

### Changed / 变更

- The directory tree is now one mechanism instead of two: full-tree loading, per-level sorting, `<details>` rendering, the `tagfile_tree_open:*` fold memory, and lazy loading of unindexed children moved out of the file list page into a shared Alpine component (`core/tree.js`, `directoryTree`), parameterized only by file-click behavior, the batch-recognition pick column, and the current-path highlight — the list page runs on it with zero behavior change. The workbench sidebar swaps the current directory's indexed/unindexed flat lists for that full collapsible tree: clicking a file switches it in place (pushState; the recommend session state is never reset), each detail landing repaints and reveals the current row — ancestors auto-expand and the row scrolls into view — and the tag/done/AI updates repaint just that row instead of refetching the tree. Directory names became a shrinking cell with an ellipsis (trailing `/` kept outside the shrink region) and the sidebar reserves its scrollbar gutter, so expanding a tall directory no longer reflows rows the way the old flat list never could. The dir-header breadcrumb shows only the leaf directory name (full path on hover), and the current file's row is now unmistakably highlighted — inset accent bar, tint, and accent-colored name, the last one because tree.css pins `.tree-name a` to `--ink` and the highlight had to override the anchor itself. Fold memory is shared between the two pages by design; the sidebar sort control now orders tree levels client-side (the server-side directory sort has no consumer left), and the SSE fingerprint dropped the directory lists the page no longer renders. `.dir-*` styles stay because the tag page still uses them.
- 目录树机制收成一份：全树加载、逐级排序、`<details>` 渲染、`tagfile_tree_open:*` 折叠记忆、未索引子项懒加载，从文件列表页整体搬进共享 Alpine 组件 `core/tree.js`（`directoryTree`），组件差异只剩文件点击行为、批量识别勾选列、当前路径高亮三处——列表页换挂后行为零变化。工作台左栏用这棵完整的折叠树替换当前目录的已索引/未索引两张平面列表：点文件就地切换（pushState，推荐流水线的会话状态不被重置），每次详情落位重画并定位当前行——祖先目录自动展开、视口滚到那一行——打标/完成/AI 的乐观更新只重绘该行而不重取全树。目录名改为可收缩单元格加省略号（行尾 `/` 留在收缩区外），侧栏为滚动条预留 gutter，展开长目录时不再出现旧平面列表不会有的挤行折行。dir-header 面包屑只显示目录的最后一级名字（完整路径悬停可查），当前文件行的高亮改为一眼可辨——内侧主色竖条 + 浅色底 + 主色名（此前 tree.css 把 `.tree-name a` 钉成 `--ink`，高亮不直接盖住锚就永远看不出）。折叠记忆按设计与两页互通；侧栏排序控件改为客户端排树层级（服务端目录排序已无消费者），SSE 指纹随之放下页面不再渲染的目录两栏；`.dir-*` 样式因标签页仍在使用而保留。

## [5.20.0] - 2026-11-21

### Changed / 变更

- The file detail page is rebuilt as a tagging workbench around the recommend → view → tag → done → next pipeline, while still opening any single file from its URL or the directory sidebar. The old single scrolling card (metadata table, always-visible rename form, preview buried mid-card, tag table at the bottom) becomes three regions: a command header carrying the file identity and the pipeline's next step, a dark media stage where image/video/audio is the first thing you see, and a dedicated tagging panel (group tabs, add-tag form, editable tag rows) that never scrolls out of reach. In recommend mode the primary button is 「完成并下一个」 — it marks the file done and draws the next one in a single click (「下一个」 once already done, 「跳过」 to advance without marking); direct-open mode keeps the 「标记完成」 toggle. Renaming moved into the filename itself (click to edit, same inline-edit component as table cells; at rest the name renders as wrapping text so long filenames stay fully visible, switching to the single-line input only while editing), the ID/path/size/type metadata table collapsed into a mono meta line plus a stage footer carrying type and size, the redundant five-chip tag preview is gone, and the tag table became compact rows (path + category + inline-editable value + remove) that fit the narrow panel. The old `.media-preview`/`.tag-preview`/`.tag-lg` styles retired with it; new panels join the shared `revealUp` entrance and its reduced-motion guard, and the responsive sentinels now track the table-free page.
- 文件详情页围绕「推荐 → 看内容 → 打标 → 完成 → 下一个」流水线重排为工作台，同时保留从 URL / 目录侧栏直接打开某个文件。原来一张滚动的长卡（元数据表、常驻的重命名表单、压在中段的预览、垫底的标签表）拆成三块：命令头承载文件身份与流水线的下一步，深色媒体舞台让图片/视频/音频第一眼可见，独立的打标面板（组 tab、加标签表单、可行内编辑的标签行）永不滚出伸手可及的范围。推荐模式主按钮是「完成并下一个」——一次点击标记完成并抽下一个文件（已完成时变「下一个」，「跳过」则不标记直接前进）；直开模式保留「标记完成」开关。重命名收进文件名本身（点击即改，与表格单元格同一个行内编辑组件；展示态折行完整显示长文件名，点击才切成单行输入框），ID/路径/大小/类型元数据表收成一行 mono 元信息加舞台脚注的类型与大小，冗余的五枚标签预览胶囊删除，标签表改为适配窄面板的紧凑行（路径 + 分类 + 行内改值 + 删除）。旧的 `.media-preview`/`.tag-preview`/`.tag-lg` 样式随之退役；新面板接入共用的 `revealUp` 入场与 reduced-motion 兜底，响应式哨兵测试同步改为追踪这个无表格的页面。

- The detail workbench now live-updates over the existing AI SSE stream instead of a manual refresh button: a `change` event silently re-reads the current detail, and a fingerprint covering every rendered field (file identity/status, media capability, groups, the current page of tags, pagination, both directory lists) gates the apply — an idle page costs one cheap read per keepalive and zero DOM churn, while recognition progress and approvals landed elsewhere appear on their own. Optimistic updates recompute the fingerprint so the following push does not re-apply identical data; environments without EventSource fall back to a low-frequency silent poll. The 「刷新」 button is deleted.
- 详情工作台改由既有的 AI SSE 流推送刷新，不再靠手动刷新按钮：`change` 事件静默重读当前详情，一枚覆盖全部渲染字段（文件身份/状态、媒体能力、组、当前页标签、分页、目录两栏）的指纹闸住落位——空闲页面每个保活周期只花一次廉价读取、零 DOM 改动，而识别进度与别处审核落库的结果会自己出现。乐观更新后重算指纹，随后的推送不会把相同数据再落一遍；没有 EventSource 的环境退回低频静默轮询。「刷新」按钮删除。

## [5.19.1] - 2026-11-21

### Fixed / 修复

- The directory sidebar now reveals like every other panel: it ran `fadeUpIn` — the same animation the rows inside it use — so its own entrance was indistinguishable from the page appearing and only the rows' later fade read as animation. It now uses the shared `revealUp 0.45s forwards` recipe of `.card`/`.tag-content-panel`, and both reduced-motion guards list the two panels so their `opacity: 0` baseline can never strand them invisible.
- 目录栏现在与其他面板同一套入场动画：它此前用的是 `fadeUpIn`——和栏内目录行完全相同的动画——面板自身的淡入淹没在整页出现的过程里，只有数据到达后目录行的二次淡入看得见，于是显得「面板没有动画」。现在改用与 `.card`/`.tag-content-panel` 相同的 `revealUp 0.45s forwards` 配方；两处 reduced-motion 兜底也补上了这两个面板，避免动画关掉后卡在 `opacity: 0` 的基线上。

## [5.19.0] - 2026-11-21

### Changed / 变更

- The container language is now one rule instead of a habit: one content region = one raised panel, regions inside a panel are separated by `.card-section`, and panels never sit inside panels. Four sites broke it and are fixed — the search page's two empty states floated on the page background while its results sat in a panel (all three now share one panel, the title just hides itself when there is nothing), the detail/recommend page's loading, no-recommendation and error states floated outside any panel (now one placeholder panel), the unsupported-video notice was a panel nested inside the file panel (now a section of it), and the tag page's add-tag form was a dashed panel inside the content panel (now a bottom section). The query builder's nested groups keep their dashed outline as a region marker but lost the glass recipe, and the recommend toolbar's inline padding became `.card-slim`.
- 容器语言收成一条规则：一块内容区 = 一块凸起面板，面板内部用 `.card-section` 分区，面板绝不套面板。四处破例已改：搜索页两个空态原本摊在背景上、结果却在面板里（现在三者同处一块面板，没结果时标题自己隐去），详情/随机打标的加载中、无推荐、出错三种状态原本在面板之外（现在归入一块占位面板），不支持播放的提示是文件面板里又套了一块面板（改为面板内的分区），标签页的加标签表单是内容面板里的虚线面板（改为底部分区）。查询构建器的嵌套分组保留虚线框作为层级标记但去掉玻璃配方，随机打标操作条的内联 padding 收进 `.card-slim`。

## [5.18.0] - 2026-11-21

### Added / 新增

- An icon system replaces the ad-hoc glyphs: a 30-symbol stroke sprite (`templates/partials/icons.html`, one 24-grid, no presentation attributes inside) plus two ways to reach it — `{{template "icon" (dict "Name" … "Class" … "Label" …)}}` in templates and `iconHtml(name, cls)` in JS — with the whole look (line weight, caps, joins, `currentColor`) declared once as `.icon` in `core/icons.css` and every size pinned at its mount point. The `empty` partial now takes an icon *name* instead of an emoji, and `TestIconSpriteMatchesEveryUsage` fails the build when a call site names an icon the sprite lacks or a symbol has no call site at all.
- 新增图标系统取代各处随手的字符图形：30 枚描边图标集中在一份 sprite（`templates/partials/icons.html`，统一 24 格、symbol 内不写表现属性），两种调用方式——模板侧 `{{template "icon" (dict "Name" … "Class" … "Label" …)}}`、JS 侧 `iconHtml(name, cls)`；线宽、端点、圆角、`currentColor` 只在 `core/icons.css` 的 `.icon` 定义一次，尺寸一律钉在挂载点。`empty` partial 的 `Icon` 参数由 emoji 改为图标名，并新增 `TestIconSpriteMatchesEveryUsage`：调用点引用了 sprite 里没有的图标、或 sprite 里有图标没人调用，都直接判构建失败。

### Changed / 变更

- Every emoji and unicode glyph that stood in for an icon is gone: the sidebar nav and the mobile menu button, the home stat labels, all empty-state icons, the file-type marks in the file tree and the detail sidebar, the tag-tree row glyph and drag handle, the two drag-and-drop zone markers, the disclosure triangles in both trees (now the same chevron the config groups use), the select-menu check, the group tab add/close, the chip remove, the toast medallions and close button, the info bubble, the sort and collapse arrows, the back arrows in page titles, the SQL export button, and the theme toggle's sun/moon (which moved into the sprite too).
- 所有拿 emoji 与 unicode 字符充当图标的地方全部换成描边图标：侧栏导航与窄屏菜单按钮、首页统计卡标签、各处空态图标、文件树与详情侧栏的文件类型标记、标签树行首图标与拖拽抓手、两个拖放提示区、两棵树的展开三角（改用与配置分组同一个 chevron）、下拉列表的选中勾、标签组的加/删、预设值芯片的移除、toast 的图标底座与关闭按钮、说明气泡、排序与折叠箭头、页标题的返回箭头、SQL 导出按钮，深浅色切换的太阳/月亮也一并进 sprite。

### Fixed / 修复

- The tag-tree drag handle now actually fades in on hover: the reveal rule (`.tag-tree-row-main:hover .drag-handle { opacity: 1 }`) had no base opacity to reveal from, because the handle's styling lived in an inline style attribute on the generated markup. The base rule now sits in `tree.css` at 0.35 opacity with the shared fast transition, matching the row's action buttons.
- 标签树的拖拽抓手现在悬停才真正显形：原来的揭示规则（`.tag-tree-row-main:hover .drag-handle { opacity: 1 }`）没有可揭示的基线透明度——抓手的样式一直挂在生成标记的 inline style 上。基线规则改放进 `tree.css`，0.35 起步、共用 fast 过渡档，与同一行的操作按钮一致。

## [5.17.1] - 2026-11-21

### Changed / 变更

- Two empty-state sub-hints are gone: 随机打标's 「这个范围内没有可推荐的文件」 no longer carries 「要么都已完成打标，要么还没有索引任何文件——换个范围试试，或先去文件管理扫描目录」, and the search page's initial 「添加条件开始搜索」 no longer lists 「支持文件名、标签、标签值、文件大小、类型、日期组合」 — the title already says what to do, and the second line only restates the form sitting right below it.
- 删掉两处空态副文案：随机打标的「这个范围内没有可推荐的文件」不再带「要么都已完成打标，要么还没有索引任何文件——换个范围试试，或先去文件管理扫描目录」，搜索页初始的「添加条件开始搜索」也不再列「支持文件名、标签、标签值、文件大小、类型、日期组合」——主文案已说清该做什么，第二行只是把眼前的表单重述一遍。

## [5.17.0] - 2026-11-20

### Added / 新增

- The review page's directory field is a combobox now: focusing it or typing opens a dropdown of workspace-relative directories that narrows as you type, with ArrowUp/ArrowDown, Enter/Tab and Escape handled from the keyboard; a newly added `GET /api/files/dirs` serves the candidates, already pruned by `scan_exclude`, so hidden or configured-away subtrees never show up. The list is fetched once per page life and filtered client-side, and a failed fetch only means no suggestions — manual input still works.
- 审核页的目录输入框改为组合框：聚焦或输入即展开工作区相对目录的下拉候选，随输入收窄，支持上下键、Enter/Tab 与 Escape；新增的 `GET /api/files/dirs` 提供候选并已按 `scan_exclude` 剪枝，隐藏或被配置排除的子树绝不出现。列表每个页面生命周期只取一次、在客户端过滤；拉取失败只意味着没有联想，手工输入照常可用。

### Changed / 变更

- The 发起识别 form is laid out on a two-row grid: the 目录 label owns the first row, and the input, both checkboxes and the submit button share the second row aligned on their control centres — the old flex-end alignment pinned the 20px checkbox line to the 34px button's bottom edge, leaving its text visibly higher. Narrow screens fall back to a single vertical column.
- 「发起识别」表单改为两行网格：目录标签独占第一行，输入框、两个复选框与提交按钮同处第二行、按控件中心对齐——旧的 flex-end 对齐把 20px 高的复选框贴到 34px 按钮的底边，文字高出一截。窄屏回退为单列纵向排列。

- Two floating-layer fixes the new list forced into the open: the enqueue card is raised (z-index) only while its dropdown is open, because a glass card's backdrop-filter makes it its own stacking context and the next card painted over the overflowing list; and `--menu-bg` goes from 0.94 to 0.98 alpha — at 0.94 the card behind still read through the panel as a 6% ghost of its dark text.
- 新列表逼出的两处浮层修正：下拉打开期间才抬高「发起识别」卡片（z-index）——玻璃卡的 backdrop-filter 让它自成层叠上下文，越界的列表被下一张卡片盖住；`--menu-bg` 由 0.94 抬到 0.98 透明度——0.94 时后面卡片的深色文字仍以 6% 残影透在面板上。

## [5.16.1] - 2026-11-20

### Changed / 变更

- The sidebar theme toggle's sun/moon emoji is replaced by a paired stroke-icon set. Both 16px SVGs stack in one slot and CSS swaps them from `data-theme` — the sun rotates and fades out as the moon springs in on the app's exponential ease-out — so no JavaScript rewrites the icon and the line weight matches the toast and favicon icons. The 浅色/深色 label is a CSS-swapped pair as well, so `theme.js` only writes the attribute: the head-time call no longer touches body elements that do not exist yet, which used to leave the emoji and label reading 浅色 under a dark system theme until the first toggle. Reduced motion swaps instantly.
- 侧边栏主题切换不再使用太阳/月亮 emoji，改用一对描边图标。两枚 16px SVG 叠在同一槽位，由 CSS 依 `data-theme` 互换——太阳旋转淡出、月亮沿全站统一的指数缓出曲线弹入——图标不再由 JS 改写，线宽与 toast、favicon 一致。「浅色/深色」文案同样由 CSS 互换，`theme.js` 只写属性：head 阶段的调用不再去碰尚不存在的 body 元素（此前系统为深色时，首次切换前图标与文案会一直停在浅色）。reduced-motion 下瞬时切换。

## [5.16.0] - 2026-11-20

### Added / 新增

- The AI task API grew the stage-two surface: `POST /api/ai/tasks/refine` queues a re-refine of a task whose raws are stored — a malformed id is refused 「无效的任务ID」 before the service sees it, while every service refusal (unknown id, a status that is neither done nor failed, no stored raws, the refine endpoint off, a file with a run in flight) is the core service's message verbatim, e.g. 「未开启整理端点, 请先在配置页配置」 — and `GET /api/ai/tasks/:id/raw` returns the stored first-pass answers frame-ordered, resolving a refine row to its vision source so the list never advertises a raw that an id-then-fetch would not deliver. Task list rows now carry `source_task_id`, `refine_status`, `has_raw` and `can_refine`; `has_raw` is computed server-side from one aggregated GROUP BY read over the page's deduplicated source ids, never one query per row, and `can_refine` is that existence ANDed with the server-side refine-endpoint check (one Settings read per page).
- AI 任务接口补上第二段的面：`POST /api/ai/tasks/refine` 为已存原文的任务入队一次「只重新整理」——非法 id 在触达服务之前就被「无效的任务ID」挡下，服务侧的每种拒绝（未知 id、非 done/failed 状态、库里没有原文、整理端点未开启、该文件已有任务在飞）都逐字转述 core 服务文案，如「未开启整理端点, 请先在配置页配置」；`GET /api/ai/tasks/:id/raw` 按帧序返回存下的识图原文，整理行自动解析到它的识图源——列表 advertise 的原文，按 id 取必能拿到。任务行新增 `source_task_id`、`refine_status`、`has_raw`、`can_refine` 四字段；`has_raw` 由服务端整页一次 GROUP BY 聚合读出（源 id 先去重），绝不逐行查询；`can_refine` 则是有原文与服务器侧整理端点开关的合取（整页只读一次 Settings）。

- The review page gained the stage-two controls: 「原文」 shows on done and failed rows whose `has_raw` the server computed — viewing stored answers is never gated on the refine endpoint, and while a running row may already hold partial raws, the button deliberately waits for a terminal status — while 「重新整理」 additionally requires the server-computed `can_refine` (has_raw with the endpoint on, so an off-mode page never advertises an always-refused button; the frontend reads no config itself). 「原文」 opens a large modal with each frame's verbatim answer as a read-only static field labelled 第 N 帧, and 「重新整理」 posts the new refine endpoint and reports through the shared toast convention. A task whose stage two failed carries a second 「整理失败」 badge beside its status badge; the pending hint is deliberately not rendered, because a leftover pending proves nothing while failed names the outcome.
- 审核页接入第二段的操作：「原文」出现在服务端算好 `has_raw` 的 done/failed 行——看原文从不受整理端点开关限制，running 行其实可能已存下部分原文，按钮仍刻意等终态才出现；「重新整理」还要求服务端算好的 `can_refine`（有原文且端点已开启，off 模式的页面因此不会挂出必被拒绝的按钮），前端自己不读配置。「原文」打开大号模态，每帧的逐字回答以只读 static 字段呈现、标注「第 N 帧」；「重新整理」提交到新 refine 端点，经共享 toast 约定播报结果。阶段二失败的任务在状态徽章旁多一枚「整理失败」徽章；pending 提示刻意不渲染——残留的 pending 说明不了什么，failed 才是终局。

- The settings page renders the new refine endpoint group with no frontend change at all: the `ai_refine_*` descriptors ship through the schema API, so 「整理端点类型」 (off by default) and its conditional URL/model/key fields slot into the existing per-provider layout — same in-place swap, same required/secret handling as the vision group.
- 配置页零前端改动地渲染出整理端点组：`ai_refine_*` 描述符随 schema 接口一并下发，「整理端点类型」（默认关闭）与其条件显隐的地址/模型/密钥字段落进既有的按 provider 分槽布局——与识图组同一套原位互换、必填徽章与密钥掩码。

### Changed / 变更

- Conditional config fields no longer reserve an invisible placeholder row for a slot the current value has no candidate in: the empty slot collapses entirely — divider line and all — so switching endpoint type makes rows appear and vanish in place while every other field keeps its relative position. The old anti-jump sizer bought stable height with a blank row wherever a provider lacks a field (Ollama has no API key); appearance and disappearance is the normal semantics of conditional fields, and the whitespace was the worse price.
- 配置页的条件字段不再为「当前取值在该槽没有候选」的槽位保留隐形占位行：空槽整槽收起——连同挂在槽上的分隔线一起消失——切换端点类型时段位在原地出现/消失，其余字段的相对位置照旧。旧的防跳动占位行用一行空白换稳定高度，provider 缺字段处（如 Ollama 没有密钥）必留白；而「出现/消失」本就是条件字段的正常语义，空白是更差的代价。

- Height changes on the settings page now animate in the app's motion language: conditional slots slide open and shut on a grid-row transition (1fr→0fr) when the endpoint type flips, and a group body collapses the same way when its card head is clicked — both on the shared `--transition` token, with the collapsed content fading (divider line included) and marked inert so it can never take focus. The collapsed body's padding rides the same transition, because a static 14px padding would otherwise clamp the box and leave a 15px ghost height. Clipping is a transition-only affair: `overflow:hidden` is switched on by transitionrun and off by the grid-row transitionend, so an open dropdown can always outgrow its row when idle (a permanent clip cut the option list off at the row boundary). `prefers-reduced-motion` switches everything instantly — and with no transition there is nothing to clip.
- 配置页的高度变化现已纳入全站动效语言：切换端点类型时，条件槽位以 grid 行过渡（1fr→0fr）滑开滑合；点击卡片头折叠分组走同一套动画——两者共用 `--transition` 令牌，收起内容连同分隔线一起淡出，并置 inert 使其绝不接收焦点。收起态的内边距也纳入过渡：静态的 14px padding 会把 border-box 夹住，留下 15px 幽灵高度。裁剪只属于过渡：transitionrun 点亮 `overflow:hidden`、行过渡的 transitionend 即撤——静止展开态永远 visible，下拉列表才能越出行界（常驻裁剪曾把选项列表剪断在行边界）。`prefers-reduced-motion` 下全部瞬时切换——没有过渡，也就无须裁剪。

### Fixed / 修复

- The dropdown caret of select-menu triggers no longer drifts on focus and blur. The focus rule set `background:` (shorthand), which reset `background-position` to `0 0` and the caret image to none, while the base rule transitioned the whole `background` shorthand — so the arrow animated from its right-edge anchor toward the top-left corner (and vanished mid-transition, reappearing there on blur). Focus now changes only `background-color`, and the transition list is narrowed to `background-color`, so a future shorthand cannot drag the caret position into an animation again.
- select-menu 触发器的下拉倒三角不再在聚焦/失焦时漂移。焦点态规则写的是 `background:` 简写，会把 `background-position` 重置为 `0 0`、图标重置为 none，而基础规则对整个 `background` 简写做了过渡——三角于是从右缘锚点一路动画滑向左上角（中途还会消失，失焦时又从左上角出现滑回去）。现在焦点只改 `background-color`，过渡属性也收窄为 `background-color`，未来简写也无法再把三角位置拖进动画。
- Floating menus no longer let the page read through them. The dropdown recipe reused `--glass-bg-strong` (65–72% alpha) and leaned on backdrop blur for separation — but a menu's backdrop sampling is cut off at the nearest effect-bearing ancestor (the glass card), so content outside that card was never blurred; it simply showed through the translucent panel. Menus now paint a dedicated near-opaque `--menu-bg` (light/dark paired) while the blur stays for whatever genuinely sits behind them inside the card.
- 浮层菜单不再透出底下文字。下拉列表沿用 `--glass-bg-strong`（65–72% 透明度）玻璃配方、指望 backdrop 模糊隔开底层——但列表的采样源在最近的带效果祖先（玻璃卡片）处截断，卡片之外的内容从未被模糊，只是直接透过半透明面板显形。列表改用专门的近不透明 `--menu-bg`（浅/深成对），模糊保留——卡内确有内容垫在后面时它仍然起作用。

## [5.15.0] - 2026-11-20

### Changed / 变更

- Toasts are redesigned in the app's glass language: each notification becomes a glass card with a semantic-tinted icon medallion, the message in ink (entity chips kept), a hover-revealed close button and a lifetime progress rail along the bottom. One `--toast-c` variable per type derives the icon, border tint, rail and glow; errors linger 5s, success/info 3s. Hovering pauses the countdown and the rail together; dismissal is two-stage (exit animation, then removal) keyed by a per-toast id, replacing the index-keyed DOM reuse and the CSS-delay/JS-timeout desync of the old bar. The stack caps at five with the oldest auto-dismissed; touch screens keep the close button visible, reduced-motion drops the animations, and the container announces via `aria-live`. `tests/js/stores.test.mjs` pins the lifecycle with fake timers.
- Toast 按全站玻璃设计语言重做：每条通知成为玻璃卡片——语义色图标底座、墨色正文（实体芯片保留）、悬停浮现的关闭按钮、底部寿命进度条。每种类型只钉一个 `--toast-c` 变量，图标、边框染色、进度条与光晕全部由它派生；错误停留 5 秒，成功/提示 3 秒。悬停时倒计时与进度条一同暂停；退场改为两段式（先播退场动画再移除），以每条 toast 的 id 为键——替换了旧彩条按索引复用 DOM、CSS 延迟与 JS 计时器不同步的机制。堆叠上限 5 条，超出时最早一条自动退场；触屏关闭按钮常显，reduced-motion 关闭动画，容器经 `aria-live` 播报。`tests/js/stores.test.mjs` 用假计时器钉住生命周期。

## [5.14.0] - 2026-11-19

### Added / 新增

- Task rows gain a 「删除」 action (`DELETE /api/ai/tasks/:id`): the shared confirm dialog names the task as an entity chip and warns that its unreviewed suggestions go with it (reviewed ones stay); a live running task is refused server-side, a stalled one is deletable. The task list now pages at 5 rows per page; the suggestion list keeps 20.
- 任务行新增「删除」操作（`DELETE /api/ai/tasks/:id`）：走全站统一确认弹窗，任务名渲染为实体芯片，并提示未审核的建议会一并删除（已审核的保留）；正在执行的任务由服务端拒绝，无人认领的可删。任务列表改为每页 5 条，建议列表维持每页 20 条。

### Changed / 变更

- The review queue's 3-second polling is replaced by a Server-Sent Events stream (`GET /api/ai/events`): the server pushes a `change` event whenever task or suggestion state actually mutates (via the core change hub), plus a 10-second keepalive as a resync safety net; an initial event on connect means a dropped-and-reconnected `EventSource` catches up immediately. `loadPaged` gains an optional `stampKey`/`stampOf` fingerprint guard covering every rendered field and the pagination position, so an unchanged fetch skips array assignment entirely — an idle page rebuilds zero DOM (reassigning an identical string to `x-html` still recreates the nodes). Environments without `EventSource` fall back to the old polling. The task card's stale 「每 3 秒自动刷新」 hint and both manual 「刷新」 buttons (task card, suggestion card) are removed — pushed updates make them redundant. Task error reasons no longer sit truncated inline in the status column: the badge gains a dotted-underline 「详情」 trigger opening a detail modal that shows the file path and the full reason as plain selectable text (a new `static` form-modal field type — no input chrome), with a single 「关闭」 button. `change_test.go`, `ai_events_test.go` and the `paged.test.mjs` stamp tests pin the hub, the push path and the guard.
- 待审队列的 3 秒轮询改为 SSE 长连接（`GET /api/ai/events`）：任务或建议状态真正变更时服务端经 core 变更 hub 推一个 `change` 事件，另有 10 秒保活作补齐兜底；连接即发初始事件，断线重连的 `EventSource` 立刻补上错过的变更。`loadPaged` 新增可选 `stampKey`/`stampOf` 指纹闸，覆盖行渲染的全部字段与分页位置，数据没变就完全跳过数组赋值——空闲页面零 DOM 重建（`x-html` 对相同字符串重新赋值也会重建节点）。无 `EventSource` 的环境退回原轮询。任务卡上过时的「每 3 秒自动刷新」提示与两处手动「刷新」按钮（任务卡、建议卡）一并删除——推送刷新让它们成为冗余。任务错误原因不再截断内联在状态列：徽章旁新增虚线下划线的「详情」入口，点开详情模态以纯文本展示文件路径与完整原因（form 模态新增 `static` 字段类型——没有任何输入框外观），页脚只留一个「关闭」按钮。`change_test.go`、`ai_events_test.go` 与 `paged.test.mjs` 的指纹测试钉住 hub、推送链路与闸门。

## [5.13.0] - 2026-11-19

### Added / 新增

- Every settings field label now carries a required/optional badge driven by the schema API's new `required` flag: red 「必填」 when an empty value is illegal, muted 「选填」 otherwise (model names, API keys, the exclude list). The frontend reads the flag as shipped — no local re-derivation of constraint semantics.
- 设置页每个字段标签旁新增必填/选填徽章，取自 schema 接口新下发的 `required` 标志：空值非法显示红色「必填」，其余（模型名、API 密钥、排除列表）显示灰色「选填」。前端直接读下发标志，不在本地重新推导约束语义。

### Changed / 变更

- Entity names in confirm dialogs and toasts are no longer wrapped in «» guillemets — they render as a shared `.entity-chip` (the same accent-on-accent-bg chip language as tag chips, minus the interactive hover), so the operated-on entity is highlighted by the design system instead of punctuation. Confirm dialogs take the name as an explicit store channel (`show(..., name)`, forwarded by `confirmDelete` via `opts.name`) across all six sites (settings group reset, tag/value/category deletion, both tag-group deletions); toasts parse the «name» marker that backend `respond` copy already carries, so every success/info/error toast gains the chip with no backend change. `tests/js/stores.test.mjs` pins both segment splitters and the forwarding.
- 确认弹窗与 toast 里的实体名不再用书名号 «» 包裹，改渲染为共享的 `.entity-chip` 芯片——与标签芯片同一套 accent 配色、去掉交互 hover——被操作实体由设计语言而非标点来突出。确认弹窗走显式 name 通道（`show(..., name)`，`confirmDelete` 经 `opts.name` 透传），六处确认点全部跟进（设置分组恢复默认、标签/预设值/分类删除、两处标签组删除）；toast 解析后端 `respond` 文案本就携带的 «名字» 标记，于是每条 success/info/error toast 都获得芯片而无需改动后端。`tests/js/stores.test.mjs` 钉住两套分段器与透传。

### Fixed / 修复

- Switching the inference endpoint type no longer moves the other settings items. Conditional fields sharing a `show_if` controller are grouped into per-ordinal slots (`buildLayout`): the three providers' URL fields share slot 1, models slot 2, API keys slot 3, and candidates swap in place inside their slot; a slot with no candidate for the current provider (Ollama has no API key) keeps its height through an invisible sizer row built from the same field geometry. Field separators moved from the field onto the slot wrapper so the lines stay put too. `tests/js/configlayout.test.mjs` pins the slot structure, in-place swap and required pass-through.
- 切换推理端点类型不再带动其他配置项移位。共享同一 `show_if` 控制键的条件字段按序位分槽（`buildLayout`）：三个 provider 的地址同槽 1、模型同槽 2、密钥同槽 3，候选字段在槽内原位互换；当前 provider 在某槽没有候选（Ollama 无 API 密钥）时，由同一套字段几何构成的隐形占位行撑住槽高。字段分隔线从字段移到槽包装层，线的位置同样不随显隐移动。`tests/js/configlayout.test.mjs` 钉住槽结构、原位互换与 required 透传。

## [5.12.0] - 2026-11-19

### Fixed / 修复

- Video AI recognition never worked in the published container image: the runtime stage was `FROM scratch`, which ships neither ffmpeg/ffprobe nor a writable `/tmp`, so every video task failed with 「未找到 ffmpeg」. The runtime is now Alpine with the distro `ffmpeg` package (ffprobe included) plus ca-certificates and tzdata; the build stage is unchanged and the application code is untouched.
- 已发布的容器镜像里视频 AI 识别从来跑不起来：运行层是 `FROM scratch`，既没有 ffmpeg/ffprobe 也没有可写 `/tmp`，视频任务只会报「未找到 ffmpeg」。运行层改为 Alpine 并安装发行版 `ffmpeg` 包（附带 ffprobe）及 ca-certificates、tzdata；构建层不变，应用代码零改动。

## [5.11.0] - 2026-11-18

### Changed / 变更

- Every dropdown is now an app-styled listbox: the nine native `<select>` sites (search query builder type/operator/unit, config oneof, both sort controls, the modal select field, tag-page category, file-detail recommend scope and tag-value picker) mount a new shared `selectMenu` Alpine component whose open state is a glass `.dropdown` panel instead of the OS-drawn menu that CSS cannot reach. The skeleton lives once in `templates/partials/select_menu.html` (the search builder keeps a documented JS-string second source); options/value are live getters so async-loaded choices appear, picks write back through `onChange` and dispatch `change` for side effects.
- 全部下拉改为应用自绘 listbox：九处原生 `<select>`（搜索构建器的类型/操作符/单位、配置 oneof、两处排序、弹窗 select 字段、标签页分类、文件详情的推荐范围与标签值选择）统一挂载新的共享 `selectMenu` Alpine 组件，展开态是玻璃 `.dropdown` 面板，而非 CSS 够不着的系统菜单。骨架单源在 `templates/partials/select_menu.html`（搜索构建器保留一份有注释约定的 JS 字符串第二源）；options/value 用活 getter，异步加载的选项能进表，选择经 `onChange` 写回并派发 `change` 供副作用。

- Keyboard and ARIA come with the component: combobox trigger + listbox with `aria-activedescendant`, ↑/↓/Home/End wrap, Enter/Space pick, Esc/outside-click/Tab close, single-character type-ahead, focus returns to the trigger; the closed control keeps the existing glass look and `--select-arrow`. The now-dead native `select` CSS (global base/focus/arrow rules, `.sort-select option`, modal/config width selectors) is purged or retargeted onto `.select-menu`; the tag-page category loses native `required`, replaced by a submit guard following the modal's 「label+不能为空」 copy convention.
- 键盘与 ARIA 随组件自带：combobox trigger + listbox 的 `aria-activedescendant` 模式，↑/↓/Home/End 回绕、Enter/Space 选中、Esc/外点/Tab 关闭、单字符 type-ahead、选中后焦点归还 trigger；闭合态沿用既有玻璃外观与 `--select-arrow`。已死的原生 `select` CSS（全局 base/focus/箭头规则、`.sort-select option`、modal/config 宽度选择器）清除或改挂 `.select-menu`；标签页分类失去原生 `required`，改由提交守卫承担，文案沿用弹窗「label+不能为空」约定。

### Added / 新增

- Responsive layout: at ≤768px the sidebar becomes a hamburger drawer (☰ button + overlay, closed by mask click / Esc / nav click) and data tables stack into glass cards — field names render from `data-label` attributes now carried by every `<td>`, including the two review tables built in JS; an invariant test (`TestStackedTableDataLabels`) keeps the attributes in place. Row actions, previously hover-only, stay visible on touch devices (hover reveal now gated behind `@media (hover: hover)`). The file-detail page stacks its directory sidebar on top and returns to natural scrolling (dropping the hard `100vh` height, `100dvh` on phones), and forms, modals, toasts, query-builder rows and media previews all tighten at the breakpoint; a lighter 1024px pass trims main padding and sidebar widths. No handler, route or API changed.
- 响应式布局：≤768px 时侧栏变为汉堡抽屉（☰ 按钮 + 遮罩，点遮罩 / Esc / 点导航均可收起），数据表堆叠成玻璃卡片——字段名由现在每个 `<td>`（含待审页两张 JS 拼接表）携带的 `data-label` 渲染，不变量测试 `TestStackedTableDataLabels` 防止漏标；原先只在 hover 显示的行操作在触屏上改为常显（hover 揭示逻辑收进 `@media (hover: hover)`）。文件详情页目录侧栏堆到顶部、脱离 100vh 硬高度恢复自然滚动（手机上用 `100dvh` 兜底），输入框、弹窗、toast、查询构造器、媒体预览同在断点处收紧；1024px 轻度档收窄主区留白与侧栏宽度。handler、路由与 API 零改动。

- `tests/js/selectmenu.test.mjs` pins the component contract (live options/value getters, pick write-back + change dispatch + refocus, keyboard wrap/Home/End/Esc/Tab/type-ahead, disabled gate); search builder type changes now reset the operator to the new type's first op so the listbox never shows an empty label.
- `tests/js/selectmenu.test.mjs` 钉住组件契约（options/value 活 getter、pick 写回 + change 派发 + 焦点归还、键盘回绕/Home/End/Esc/Tab/type-ahead、disabled 闸门）；搜索构建器切换类型时把操作符重置为新类型的首个 op，listbox 不再显示空标签。

### Fixed / 修复

- The tree sort dropdown's compact styling was dead CSS: `.sort-select` lived in `@layer layout` while the form control base rule lives in `@layer components`, and layer order outranks specificity, so the control rendered at full input size (13px / 7px padding) next to its tiny sort-order button. The override now sits in the components layer beside the rule it overrides and the 2xs compact look actually applies.
- 树排序下拉的紧凑样式一直是死 CSS：`.sort-select` 在 `@layer layout`，而表单控件基础档在 `@layer components`，层序压过 specificity，于是控件以完整输入框尺寸（13px / 7px 内边距）渲染在旁边那颗小小的排序按钮边上。现将覆盖档移到 components 层、紧挨被覆盖的规则，2xs 紧凑外观真正生效。

## [5.10.1] - 2026-11-18

### Fixed / 修复

- Patched four regressions left by the convergence waves. The file-detail tag-value toast flashed a bare "ok" because `HandleUpdateTagValue` answered `respond(c, "ok", ...)` while `reportResult` prefers the server `msg` — it now returns 「标签值已更新」 like its add/remove siblings. The config page's `optionsOfRules` had been folded onto `findRule` (first-match-wins), so a corrupt leading `oneof` rule hid every later valid option list; it walks all rules again, skipping unparseable entries. The search query-builder's five `delegate` handlers lost the old `onBuilderClick`'s `preventDefault`; each now calls it (matching the tag page), so builder buttons never trigger default actions. And `make fe-test` invoked `node --test tests/js/`, which Node 24 treats as a module entry (MODULE_NOT_FOUND); it now globs `tests/js/*.test.mjs`.
- 补上收敛波次遗留的四处回归。文件详情改标签值的 toast 原先闪一句裸 "ok"——`HandleUpdateTagValue` 回的是 `respond(c, "ok", ...)`，而 `reportResult` 优先播服务端 `msg`；现与添加/移除同族，回「标签值已更新」。配置页 `optionsOfRules` 曾被收进 `findRule`（首命中即停），首条 `oneof` 指令损坏会遮住后面所有合法选项表；现重新逐条遍历、跳过解析失败的条目。搜索条件构建器五个 `delegate` handler 丢了旧 `onBuilderClick` 的 `preventDefault`；现逐个补回（与标签页一致），构建器按钮不再触发默认行为。`make fe-test` 原跑 `node --test tests/js/`，Node 24 把目录参数当模块入口（MODULE_NOT_FOUND）；现改 glob `tests/js/*.test.mjs`。

### Added / 新增

- Regression tests pin the touched contracts: a Go assertion that `PUT /api/files/tag` answers 「标签值已更新」, and node:test cases for `confirmDelete`'s default `info` toast, `reportResultOrThrow`'s server-msg-over-fallback priority, `loadPaged`'s silent failed-envelope quietness, `pagedList.clearSearch`/`goToPage`, and `API.fileDetail` keeping `group_id=0` (the default group is a legal value, not a falsy drop).
- 回归测试钉住本次触及的契约：Go 断言 `PUT /api/files/tag` 回「标签值已更新」；node:test 覆盖 `confirmDelete` 默认 `info` 播报、`reportResultOrThrow` 服务端 msg 优先于兜底文案、`loadPaged` 静默失败信封不弹 toast、`pagedList.clearSearch`/`goToPage`，以及 `API.fileDetail` 保留 `group_id=0`（默认组是合法值，不能被真值判断丢掉）。

### Changed / 变更

- The hint line under each config field is gone: `fieldHint`, the `.config-field-hint` span and its CSS are deleted, so the meta row holds only the error text. Usage copy reaches the ⓘ tooltip fed by `description`, which is why 「每行一条，正则表达式」 was dropped rather than moved — the row editor already shows one input per regex, and 「扫描排除正则列表」 became 「输入正则表达式」.
- 字段下方的提示行移除：删除 `fieldHint`、`.config-field-hint` 及其 CSS，meta 行只剩错误文案。用法说明由 `description` 驱动的 ⓘ 气泡承担，因此「每行一条，正则表达式」是被丢弃而非搬运——行编辑器本身就是一条正则一个输入框；「扫描排除正则列表」改为「输入正则表达式」。

## [5.10.0] - 2026-11-18

### Changed / 变更

- Scalar design tokens landed: a seven-step type ramp (`--text-2xs..2xl`), a blur ramp (`--glass-blur-xs/sm/lg` next to the panel's `--glass-blur`), three accent-glow box-shadow recipes (`--glow-sm/md/lg`), `--radius-xs`/`--radius-pill` and `--transition-fast` — every occurrence of the mapped literals in components/layout/base CSS now reads from the tokens (~114 replacements, values unchanged; `--radius-pill: 999px` replaces 9px/20px radii that already rendered as pills). Input widths got four named tiers (`--input-w` classes sm 120 / md 200 / lg 280 / xl 360) that absorb a dozen-plus ad-hoc inline width values — a deliberate convergence, individual fields grow or shrink by up to 40px; table column widths (`col-w-*`) and the modal's help text (`form-help`) moved off inline styles as well. The token gate test pins the mapped literals' absence; unmapped component-identity values (15px serif titles, 40px stat digits, blur 12/16px) stay literal by design.
- 标量设计 token 落地：七级字号阶梯（`--text-2xs..2xl`）、模糊阶梯（`--glass-blur-xs/sm/lg`，与主面板档 `--glass-blur` 并立）、三档 accent 光晕阴影配方（`--glow-sm/md/lg`）、`--radius-xs`/`--radius-pill` 与 `--transition-fast`——components/layout/base 里所有映射表内的字面值改读 token（约 114 处替换，数值本身不变；9px/20px 圆角换成 `--radius-pill: 999px`，它们原本就已渲染成胶囊）。输入框宽度归为四档命名类（sm 120 / md 200 / lg 280 / xl 360），吸收 12 种随手内联宽度——刻意的设计收敛，个别字段宽窄变化至多 40px；表头列宽（`col-w-*`）与弹窗帮助文案（`form-help`）也离开内联。token 回归门钉住映射内字面值零残留；映射外的组件身份值（15px 衬线标题、40px 统计数字、blur 12/16px）按设计保留字面。

## [5.9.0] - 2026-11-18

### Changed / 变更

- CSS componentization wave: the glass-panel recipe now has exactly one source — the new `.panel` class (`card.css`); stat cards, both sidebars, the tag content panel, the inline-add strip and the search query-builder groups mount it and keep only their differences. The card family (`card/card-header/card-title/stats/stat-card`) moved from `layout.css` into `card.css`, the modal section into a new `modal.css`, and inline-edit styles into `forms.css` (with the modal's readonly-field look replacing a template `:style` ternary via `.modal-form input[readonly]`). A `.toolbar` class replaces ~10 hand-rolled flex toolbars (a few gained 2-10px of gap — deliberate convergence) and `card-section`/`media-preview` classes retire the detail page's section-line and media-size inlines. Dead CSS is gone and synonyms merged: `.scan-result` (no callers), `.config-invalid-badge` (now `badge badge-red`), the four `tree-status-ai-*` color pairs (tree rows mount `tree-status badge-ai-*`; `AI_STATUS_META` lost its `treeClass` field), `.tag-chip` (now `.tag.tag-lg`) and `.dir-sidebar-static` (`.dir-sidebar` + `.scroll` modifier); `@keyframes fadeUpIn` moved home to `animations.css`.
- CSS 组件化波次：玻璃面板配方如今只有一个来源——新 `.panel` 类（`card.css`）；统计卡、两侧栏、标签内容面板、行内新增条与搜索条件构建器统一挂载它，自己只留差异。card 家族（`card/card-header/card-title/stats/stat-card`）从 `layout.css` 迁入 `card.css`，弹窗段独立成 `modal.css`，行内编辑样式并入 `forms.css`（弹窗只读字段外观由 `.modal-form input[readonly]` 规则接管，替代模板里的 `:style` 三元）。`.toolbar` 类取代约十处手写 flex 工具栏（个别间距收敛了 2-10px，属刻意归一），`card-section`/`media-preview` 两个类消灭详情页分节线与媒体尺寸的写死内联。死 CSS 出清、同义合并：`.scan-result`（无人调用）、`.config-invalid-badge`（改 `badge badge-red`）、四条 `tree-status-ai-*` 配色（树行现挂 `tree-status badge-ai-*`，`AI_STATUS_META` 删掉 `treeClass` 字段）、`.tag-chip`（改 `.tag.tag-lg`）、`.dir-sidebar-static`（`.dir-sidebar` + `.scroll` 修饰）；`@keyframes fadeUpIn` 迁回 `animations.css`。

## [5.8.0] - 2026-11-18

### Changed / 变更

- Wave 14 converged the JS idioms onto shared core helpers. Delete confirmations (6 sites) and the inline-edit throw contract (3 sites) now run through `confirmDelete`/`reportResultOrThrow` — all delete toasts uniformly report as `info` (the file-detail group delete previously flashed `success`; deliberate normalization) and a failed delete no longer refreshes the tag list (it reports through `API.onError` instead). A node:test harness (`make fe-test`) pins the silent-poll, clamp and rollback contracts. Pagination loads run through `loadPaged`/`pagedList`/`clampToLastPage` — the out-of-range "fell off the last page" protection that only the review queue had now reaches every pager, and deleting the last row of the final page lands you back on a page with content. `API.enqueueAi` absorbs three verbatim failure branches; `delegate`/`createSelection` single-source the data-action dispatch and the multi-select filter/concat idiom; `decodePath`/`fileDetailHref`/`readStoredJSON`/`readSortState` retire four copy-paste families; and the config page's rule scanners and DynamicArray kept-rows share one traversal each.
- 第 14 波把 JS 惯用式收进 core 共享件。删除确认（6 处）与行内编辑 throw 契约（3 处）统一走 `confirmDelete`/`reportResultOrThrow`——删除播报一律 `info`（文件详情删标签组原先闪 `success`，属刻意归一），删除失败不再顺手刷新标签列表（改经 `API.onError` 报错）；node:test 基建（`make fe-test`）钉死静默轮询、越界回退与回滚契约。分页加载统一 `loadPaged`/`pagedList`/`clampToLastPage`——只有待审队列曾经的「末页掉页」保护现在普及到所有分页，删掉末页最后一行会自动落回有内容的页。`API.enqueueAi` 吸收三处逐字失败分支；`delegate`/`createSelection` 单源化 data-action 分发与多选 filter/concat 惯用式；`decodePath`/`fileDetailHref`/`readStoredJSON`/`readSortState` 消灭四族复制粘贴；配置页指令扫描器与 DynamicArray 保留行集各自单源一次遍历。

### Fixed / 修复

- The file detail page keeps the user's sort after tag/group mutations: the post-mutation refresh (`reloadGroupsAndTags`) fetched without `sort`/`order` and the list silently snapped back to name-ascending. Detail fetches now share one `API.fileDetail` + `applyDetail` path, and the sidebar `tag_done`/`ai_status` mirrors write through one helper.
- 文件详情页在标签/组操作后不再丢用户排序：操作后的局部刷新（`reloadGroupsAndTags`）此前不带 `sort`/`order` 请求，列表悄悄弹回按名升序。详情抓取现共用一条 `API.fileDetail` + `applyDetail` 路径，侧栏 `tag_done`/`ai_status` 镜像也归一进同一 helper。

## [5.7.0] - 2026-11-18

### Changed / 变更

- The `empty` partial now speaks a dict dialect (`Text`/`Icon`/`Hint`/`Show`, plus `TextExpr` for x-text error states) and all 20 empty-state sites — 18 hand-written blocks across six pages and the two review-page string calls — render from it; a Go render test pins the escaped x-show/x-text round-trip. Inline-edit cells collapse into the new `inlineEditCell` partial (4 sites), and a pile of dead inline styles is purged: redundant `margin:0` on card titles, duplicated flex properties, undefined `reveal-delay-*` classes, a bare `btn-ghost` link missing its `.btn` base, and the review page's two hand-rolled checkbox labels and three loading hints which became real `.checkbox-label`/`.loading-hint` classes.
- `empty` partial 改说 dict 方言（`Text`/`Icon`/`Hint`/`Show`，错误态另有 `TextExpr`），20 处空态全部收编——六页 18 处手写块加待审队列两处字符串调用；新增 Go 渲染测试钉住 x-show/x-text 转义往返。行内编辑单元格收进新 `inlineEditCell` partial（4 处）。死内联样式清扫：card 标题冗余 `margin:0`、重复 flex 属性、无定义的 `reveal-delay-*`、缺 `.btn` 基类的裸 `btn-ghost` 链接；待审页两个手写复选框 label 与三处加载提示落为真实的 `.checkbox-label`/`.loading-hint` 类。

### Fixed / 修复

- Inline edit now rolls back the displayed value when the save is rejected: `inlineEdit.save()` snapshotted `this.value` at save time, but `x-model` keeps syncing it with every keystroke — the "undo" restored the dirty value. Rollback now uses the pre-edit `original`.
- 行内编辑保存被服务端拒绝后显示值现在会正确回滚：`inlineEdit.save()` 的撤销快照取自保存时刻的 `this.value`，而 `x-model` 每次击键都在同步它——回滚恢复的是脏值。现改用编辑前的 `original` 快照。

## [5.6.0] - 2026-11-18

### Fixed / 修复

- Four dark-theme bugs where light values were hardcoded into accent surfaces. Text on accent/danger backgrounds now follows a new `--on-accent` token (`#fff` light, `#0c1a1e` dark): white-on-`#2dd4bf` measured ~1.9:1 in dark mode, the dark ink lifts it to ~9:1. The danger button's hover glow builds from `color-mix(… var(--danger) …)` instead of a frozen light RGB, the three flash-toast borders likewise mix from the semantic tokens, and the select arrow moved into a theme-paired `--select-arrow` image token so it no longer paints the light muted grey onto dark glass.
- 修复四处把浅色写死进 accent 表面的暗色主题 bug。accent/danger 实底上的文字改用新 token `--on-accent`（light `#fff`、dark `#0c1a1e`）：暗色下白字配亮青仅约 1.9:1，换深色字后约 9:1。危险按钮悬停光晕改由 `color-mix(… var(--danger) …)` 生成，不再写死浅色 RGB；三条 flash 提示边框同理由三个语义色 mix 而来；select 下拉箭头收进按主题配对的 `--select-arrow` image token，不再在暗色玻璃上画浅色灰。

## [5.5.0] - 2026-11-18

### Changed / 变更

- Common form controls converged to single sources. Checkboxes finally have one global rule (16px square, `accent-color` from the theme, no inherited glass padding/blur): the three inline `14px` patches on the review page and the config page's private sizing rule are gone, and the modal's checkbox — which never had a patch and rendered deformed — is fixed by exempting it from the modal's `width:100%` field rule. Clickable tag chips share a new `.tag-clickable` class instead of per-site inline `cursor`/`text-decoration`. The autocomplete dropdown now renders from the `tagSearchField` partial in all three server-side sites (tag add form, file detail add-tag, and the form modal's autocomplete field, the last via new `Bare`/`PlaceholderBind` params); the search builder's runtime-string copy is documented as the unavoidable second source (Go template vs JS string cannot share code). The two sort-control DOM blocks (file tree, file detail) collapse into a `sortControlsField` partial — the four sort options are single-sourced, ids preserved (`tree-sort-*`, `sort-*`).
- 常用表单控件归一单源。复选框终有一条全局规则（16px 方形、勾色随主题 `accent-color`、不再继承 glass 的 padding/blur）：待审队列页三处内联 14px 补偿与配置页的私有尺寸规则删除，弹窗里的复选框（唯一从没打补丁、一直变形的）经 modal `width:100%` 规则豁免后修复。可点击标签 chip 共用新 `.tag-clickable` 类，取代各站内联 cursor/text-decoration。标签补全下拉在服务端渲染的三处全部走 `tagSearchField` partial（标签管理加标签、文件详情加标签、以及表单弹窗的 autocomplete 字段——后一处靠新增 `Bare`/`PlaceholderBind` 参数适配）；搜索构建器的运行时字符串版是无法避免的第二源（Go 模板与 JS 字符串无法共享代码，已加交叉引用注释钉住）。两处排序控件 DOM 块（文件树、文件详情）收进 `sortControlsField` partial——四个排序选项单源，id 钩子原样保留（`tree-sort-*`、`sort-*`）。

## [5.4.0] - 2026-11-18

### Changed / 变更

- All six inline pagination blocks (categories, tags flat, search, file detail tags, and both review tables) now render from one shared `pager` template partial speaking the server envelope uniformly (`has_prev/has_next/prev_page/next_page`): the review page's numeric-comparison dialect — provably equivalent to the envelope, since the server derives `has_prev = page > 1` and `has_next = page < total_pages` — is gone along with its two `taskPages()`/`sugPages()` helpers. Wording standardized everywhere to 「« 上一页 / 下一页 »」 and 「(共 N 条)」, so five pages see a small label change (前页/后页 → 上一页/下一页, N 项 → 共 N 条). The page-count text gained a real `.pagination .page-info` CSS rule (it had none — the class was decorative) and all six pagers now share the `--ink-muted` color; the categories page's counter darkened by one step to match the majority.
- 六处内联分页块（分类、标签平铺、搜索、文件详情标签、待审队列两表）统一由新的 `pager` partial 渲染，全部说服务端信封字段（`has_prev/has_next/prev_page/next_page`）：待审队列的数字比较方言——`has_prev = page > 1`、`has_next = page < total_pages` 由服务端同一公式产出，等价性可证——连同 `taskPages()`/`sugPages()` 两个辅助方法一并消失。全站文案归一为「« 上一页 / 下一页 »」与「(共 N 条)」，五个页面措辞微变（前页/后页 → 上一页/下一页，N 项 → 共 N 条）。页码计数文字获得真实 CSS 规则 `.pagination .page-info`（此前这个类没有任何样式，纯装饰），六处分页统一为 `--ink-muted` 着色；分类页的计数自本版起加深一档，与多数派对齐。

## [5.3.0] - 2026-11-18

### Added / 新增

- Second-round dedupe: `reportResult(d, okMsg, failMsg, okType)` in `core/utils.js` absorbs ~20 copies of the `ok→toast(msg||fallback) else toast(error)` envelope across tag/category/review/detail/tree/groups; the form-modal store now validates `required` fields centrally (trim semantics, per-field `requiredMsg` override), so footer-button saves no longer bypass the HTML `required` constraint that five hand-rolled guards used to patch individually; `API.loadAllTagsSafe()` collapses three identical fetch-with-empty-fallback blocks; `fe-check` grows a fourth gate flagging any partial `{{define}}` no template invokes — it immediately caught the long-dead `tagBadge` and server-side `pagination` partials, both removed.
- 二轮去重：`core/utils.js` 的 `reportResult(d, okMsg, failMsg, okType)` 收编 tag/category/review/detail/tree/groups 约 20 处「ok→toast(msg‖fallback) 否则 error」样板；表单弹窗 store 集中校验 `required` 字段（trim 语义，字段级 `requiredMsg` 可覆盖文案），页脚按钮保存不再绕过 HTML `required` 约束（此前靠五处手工守卫各自打补丁）；`API.loadAllTagsSafe()` 合并三处同形「拉全量标签失败回退空数组」；`fe-check` 新增第四项门：partial 里没人调用的 `{{define}}` 直接报错——它当场揪出早已死掉的 `tagBadge` 与服务端 `pagination` 两个 partial，均已删除。

- The tag-autocomplete block (input + dropdown + x-for) shared verbatim between the tag page add-form and the file detail add-tag form now renders from the new `tagSearchField` partial, and the two list-page search forms (categories, tags) from the new `searchForm` partial; templates gained a `dict` FuncMap helper for multi-parameter partial calls.
- 标签管理加标签表单与文件详情加标签表单里逐字重复的补全块（input + dropdown + x-for）改由新 partial `tagSearchField` 渲染，分类/标签两页的搜索表单改由新 partial `searchForm` 渲染；模板侧新增 `dict` FuncMap 辅助函数，支撑多参数 partial 调用。

### Changed / 变更

- Creating a tag group with an empty name is now blocked client-side with 「标签组名称不能为空」 instead of firing a POST that the server rejected with 「参数不完整」 (approved tightening; it was the one `required` modal field that had no guard).
- 详情页创建标签组留空现在本地拦截并提示「标签组名称不能为空」，不再发 POST 吃服务端「参数不完整」（经批准的行为收紧；它正是唯一带 `required` 却没有手工守卫的弹窗字段）。

## [5.2.0] - 2026-11-18

### Added / 新增

- `make fe-check` is the new frontend safety net: `node --check` over every JS file under `static/js` plus a two-way JS↔template reference check (a JS file loaded by zero or more than one template fails the gate) — together with the existing Go API tests they lock behavior for frontend refactors.
- 新增 `make fe-check` 前端安全网：对 `static/js` 下每个 JS 做 `node --check` 语法检查，外加 JS↔模板双向引用一致性（某 JS 被 0 个或多个模板加载即失败）——与既有 Go API 测试共同为前端重构锁住行为。

### Changed / 变更

- Sort controls on the file tree and file detail pages are now driven by the shared Alpine `sortControls` component, replacing two hand-copied implementations; the session keys (`tagfile_tree_*`, `tagfile_detail_*`), persistence and re-render wiring are unchanged, and the checkbox-sync-after-sort guard keeps working through the page mirror. The batch-enqueue skip summary (`skipSummary`) now lives once in `core/utils.js`, shared by the file tree and review pages, and the review page's two polling lists collapsed into one `LISTS`-driven loader with identical silent-poll and out-of-range clamp semantics. `core/api.js` groups its verbs before its helpers, and the core scripts' load-order contract is documented in their headers.
- 文件树与文件详情页的排序控件收敛到共享 Alpine 组件 `sortControls`，替换掉两份手写副本；会话键（`tagfile_tree_*`、`tagfile_detail_*`）、持久化与重渲染接线不变，「换排序后勾选同步」的护栏经页面镜像继续生效。批量识别的跳过播报（`skipSummary`）在 `core/utils.js` 只存一份，文件树与待审队列共用；待审队列两张轮询表合并为一条由 `LISTS` 驱动的加载骨架，静默轮询与越界 clamp 语义逐字一致。`core/api.js` 把动词归组到辅助函数之前，core 各脚本的加载顺序契约写进文件头注释。
- Search-history summaries format file sizes with the shared `formatSize`, so a value renders as e.g. 「1.5 MB」 — rounded with a space — instead of the previous unrounded, spaceless 「1.53515625MB」.
- 搜索历史摘要的大小格式化改用共享的 `formatSize`，取值现在渲染为「1.5 MB」——四舍五入且带空格——取代此前不四舍五入、无空格的「1.53515625MB」。

## [5.1.0] - 2026-11-17

### Changed / 变更

- The settings page now speaks a rule-based contract: `GET /api/config/schema` answers blocks, each with `input_boxes` carrying `value_type`/`container` and three rule lists — `rules` (value/item), `positional_rules` (fixed-array positions) and `container_rules` (`item_count`/`fixed_arity`) — instead of the old `type`/`item_type` fields. Every rule carries `kind`, `role`, `params`, `hint`, `message` and `widget`, all rendered backend-side, so the wire shape maps 1:1 onto the descriptor with no second projection.
- 设置页改用基于规则的契约：`GET /api/config/schema` 返回块，每块含 `input_boxes`，每个 input box 带 `value_type`/`container` 与三组规则——`rules`（值级/项级）、`positional_rules`（定长数组逐位）与 `container_rules`（`item_count`/`fixed_arity`）——取代旧的 `type`/`item_type` 字段。每条规则带 `kind`、`role`、`params`、`hint`、`message`、`widget`，全部由后端渲染填入，因此网络层结构与描述符一一对应，无需第二份投影。

- The frontend is now a generic rule executor: it only knows the `kind` primitives and their role, runs local best-effort pre-validation through `ruleCheckers`, drives the control effect from the backend's `widget` (select/password/textarea), and silently ignores any unknown kind — the server still rejects a bad value and echoes `invalid_reason` with the same message, so degradation is safe by construction. Secret masking and leave-blank-keeps behavior are enforced server-side and never depend on the frontend recognising the `secret` kind (the value is never even sent).
- 前端成为通用指令执行器：只认识 `kind` 原语及其角色，经 `ruleCheckers` 做尽力而为的本地预校验，从后端下发的 `widget` 驱动控件效果（select/password/textarea），并静默忽略任何未知 kind——服务端仍会拒绝非法取值并回填 `invalid_reason`（文案一致），因此降级天然安全。secret 掩码与「留空不覆盖」由服务端强制，从不依赖前端是否认识 `secret` 指令（值根本不会被下发）。

- Zero copy of backend text and control mapping: every hint, failure message and widget effect — 「每行一条，正则表达式」, 「已设置，留空不修改」 and all validation messages — now comes from the single backend rule registry, so the JS and templates hold no Chinese validation copy and no `item_type` literal; `shift`/`options` are read from the rule `params`, and the widget from the rule effect, with only structural defaults (number/checkbox/rows/input) staying in the frontend.
- 后端文案与控件映射零副本：每条用法提示、失败文案与控件效果——「每行一条，正则表达式」、「已设置，留空不修改」及所有校验文案——都来自后端唯一规则注册表，因此 JS 与模板不再持有任何中文校验文案，也没有 `item_type` 字面量；`shift`/`options` 从规则 `params` 读取，widget 从规则效果读取，前端仅保留结构默认（number/checkbox/rows/input）。

## [5.0.0] - 2026-11-17

### Removed / 移除

- Every gradient paint is gone from the WebUI. The page background is now a flat solid per theme (`--bg`: light `#e6f0ed`, dark `#0c1a1e`) replacing `--bg-gradient` plus the three layered radial blobs; the two ambient `body::before`/`body::after` glow circles and the stat-card hover specular sweep (pseudo-element plus `specularSweep` keyframes) are deleted; the skeleton loader pulses a solid `--border-light` fill via `skeletonPulse` instead of shimmering a moving linear-gradient; and the favicon tag is flat `#0d9488` with its `linearGradient` defs removed. The now-unused `--bg-gradient`, `--bg-blobs`, `--glow-a`, `--glow-b` and `--sweep-color` variables are deleted, and `--paper` points at `--bg`. Glass blur, insets and the grain overlay are untouched.
- WebUI 清除全部渐变：页面背景改为每主题纯色（`--bg`：亮 `#e6f0ed`、暗 `#0c1a1e`），取代 `--bg-gradient` 与三层径向色斑；删除 `body::before`/`body::after` 两团环境光晕和 stat-card hover 的镜面扫光（伪元素及 `specularSweep` 关键帧）；骨架屏改为纯色 `--border-light` 填充做 `skeletonPulse` 透明度脉冲，不再用移动线性渐变做 shimmer；favicon 标签改为纯 `#0d9488`，移除 `linearGradient` 定义。随之删除不再使用的 `--bg-gradient`、`--bg-blobs`、`--glow-a`、`--glow-b`、`--sweep-color` 变量，`--paper` 改指 `--bg`。玻璃模糊、inset 亮边与噪点层保持不变。
- The review page's enqueue card dropped its two explanatory strings: the directory label is now just 「目录」 (the 「（相对扫描根目录，. 表示全部）」 parenthetical is gone) and the footnote 「只有已注册且扩展名受支持的图片/视频会入队；「强制重跑」会丢弃该文件尚未审核的建议。」 div is deleted. The force-rerun confirmation dialog on the file detail page is untouched — it warns about a destructive action rather than describing the form.
- 待审队列页「发起识别」卡片删除两处说明文案：目录 label 只剩「目录」（去掉「（相对扫描根目录，. 表示全部）」括注），并整块删除「只有已注册且扩展名受支持的图片/视频会入队；「强制重跑」会丢弃该文件尚未审核的建议。」脚注 div。文件详情页强制重跑的确认弹窗保留——它是对破坏性操作的警告，而非表单说明。

## [4.5.0] - 2026-11-16

### Changed / 变更

- The WebUI theme switched from indigo-purple to teal: light accent `#0d9488`, dark accent `#2dd4bf`, with new background gradients (teal → misty blue → warm sand for light, deep teal-ink for dark), teal-tinted neutral inks and a teal favicon. All previously hardcoded indigo and pink values in `layout.css`, `tags.css` and `base.css` now reference `--accent-*` / `--glow-*` theme variables, so future re-theming touches `variables.css` only. Dark glass fills and the sidebar were re-tinted from indigo-navy to teal-ink so no surface keeps the old palette.
- WebUI 主题色由靛紫整体切换为青碧：亮色强调 `#0d9488`、暗色 `#2dd4bf`，背景渐变改为「青碧→雾蓝→暖沙」（暗色为深青墨），中性色微调为灰青调，favicon 同步换色。`layout.css`、`tags.css`、`base.css` 中硬编码的靛紫与粉色全部改引 `--accent-*` / `--glow-*` 主题变量，此后换主题只需改 `variables.css`。暗色玻璃填充与侧栏由靛蓝海军色重染为青墨色，确保没有任何表面残留旧色板。

### Added / 新增

- Glass surfaces gained the full material treatment per the glassmorphism design spec (`docs/superpowers/specs/2026-11-16-webui-teal-glassmorphism-design.md`): layered radial background blobs so backdrop blur has real content to frost, inset top-highlight/bottom-shade baked into `--glass-shadow`, one global `feTurbulence` grain overlay (opacity 0.045) to kill gradient banding, and a transform-only specular sweep on stat-card hover. Fallbacks: `prefers-reduced-transparency` renders solid panels and hides the grain, `@supports not (backdrop-filter)` raises panel opacity, and the existing reduced-motion rule covers the sweep.
- 按玻璃拟态设计规格（`docs/superpowers/specs/2026-11-16-webui-teal-glassmorphism-design.md`）补齐玻璃材质：背景叠加多层径向色斑让 backdrop blur 有真实内容可霜化；`--glass-shadow` 内置顶部亮边/底部阴影 inset；全局单层 `feTurbulence` 噪点（opacity 0.045）消除渐变色带；stat-card hover 增加仅动画 transform 的镜面扫光。降级：`prefers-reduced-transparency` 下实心面板并隐藏噪点，`@supports not (backdrop-filter)` 下提高面板不透明度，扫光由现有 reduced-motion 规则覆盖。

## [4.4.2] - 2026-11-16

### Changed / 变更

- The settings page's block frame is now shared instead of inline: `templates/partials/config_group.html` owns the card (collapse head with the invalid-count badge, body, and the 保存 / 恢复默认 footer) and `templates/partials/config_field.html` owns the per-type control dispatch, so `config.html` renders every schema group with one `{{template "config_group"}}` inside the existing `x-for`. The collapse state moved out of `configPage()` into a new Alpine component, `configGroup({ id })` in `core/components.js`, which keeps persisting it under the same `tagfile_config_collapsed` key, so nothing observable changed; adding a settings block is now a schema registration plus that one template line rather than a copy of the card markup.
- 设置页的板块外壳改为复用而非内联：新增 `templates/partials/config_group.html` 承载卡片（折叠头 + 非法项数徽标 + 主体 + 保存 / 恢复默认 页脚），新增 `templates/partials/config_field.html` 承载按类型分派的控件渲染，于是 `config.html` 只需在原有 `x-for` 里写一行 `{{template "config_group"}}` 就能渲染 schema 里的每个分组。折叠状态从 `configPage()` 移入新的 Alpine 组件 `configGroup({ id })`（位于 `core/components.js`），仍沿用同一个 `tagfile_config_collapsed` 存储键，因此可见行为不变；此后新增一个配置板块只要注册 schema 加上这一行模板，不必再复制卡片代码。

## [4.4.1] - 2026-11-16

### Fixed / 修复

- `make docker-build` failed at the scratch stage's `COPY --from=builder /usr/share/zoneinfo`: `golang:1.25-alpine` ships no tzdata, so that path does not exist in the builder and the build aborted with `copier: stat: "/usr/share/zoneinfo": no such file or directory`. The builder now runs `apk add --no-cache tzdata` after the compile step, which is the last builder layer and therefore invalidates no cached download or build. Deployed containers should also pass `-e TZ=Asia/Shanghai`, since `time.Local` otherwise falls back to UTC and the log timestamps drift from the host.
- `make docker-build` 此前在 scratch 阶段的 `COPY --from=builder /usr/share/zoneinfo` 处失败：`golang:1.25-alpine` 不带 tzdata，builder 里根本没有该路径，构建以 `copier: stat: "/usr/share/zoneinfo": no such file or directory` 中止。现在改为在编译步骤之后 `apk add --no-cache tzdata`，而编译层是 builder 的最后一层，因此不会让已缓存的依赖下载与构建失效。部署容器还应带上 `-e TZ=Asia/Shanghai`，否则 `time.Local` 回落 UTC，日志时间戳与宿主不一致。

### Changed / 变更

- The 「高级 / 自定义配置」 card is gone from the settings page, together with its search form, table, pagination, add form and edit/delete dialogs. Unregistered keys stay readable and writable through `POST/PUT/DELETE /api/config` and the CLI, but the page no longer offers a second editing entry for them.
- 设置页删除「高级 / 自定义配置」卡片，连同其搜索表单、表格、分页、新增表单与编辑/删除弹窗。未注册的键仍可由 `POST/PUT/DELETE /api/config` 与 CLI 读写，但页面不再为它们提供第二个编辑入口。

- 「恢复默认」 is now one button per group card, in the footer next to 保存, instead of one per field. It asks for confirmation, resets every field of that group to its declared default locally, and toasts 「已恢复默认值，尚未保存」 — nothing is written until 保存. A secret field resets to blank, which the group save then omits, so the stored secret survives. The per-field key chip and the 「默认 X · 范围 …」 hint line are also removed: the key name is a developer identifier, and both the default and the legal range are already reachable through 恢复默认 and the invalid-value message 「取值超出合法范围，应为 [1, 200]」.
- 「恢复默认」 改为每个分组卡片一个按钮，位于页脚「保存」旁，而不再逐字段放置。它会先请求确认，然后把该组每个字段在本地重置为声明的默认值，并提示「已恢复默认值，尚未保存」——不点保存就不落库。secret 字段重置为空串，而分组保存会跳过它，因此已存密钥不会被抹掉。字段行的键名 chip 与「默认 X · 范围 …」说明行同样移除：键名是开发者标识，而默认值与合法范围已经由「恢复默认」和非法值报错「取值超出合法范围，应为 [1, 200]」两处体现。

## [4.4.0] - 2026-11-16

### Added / 新增

- `GET /api/config/schema` answers the whole settings page in one request: every registered group in `Order` with its fields, each carrying `type`, `label`, `description`, `default`, `min`/`max` with `min_exclusive`/`max_exclusive`, `pattern`, `options`, `item_type`, `unit` and `show_if`, plus the live state — `value`, `has_value`, `invalid` and `invalid_reason`. The snapshot is taken from the whole `configs` table in a single read rather than one request per card, so a card can no longer be saved from a set of defaults that a still-outstanding read would have replaced. A key missing from the table answers with its declared default; `has_value` says whether a non-empty value is stored. A secret field's value and default are always blank and only `has_value` is sent, since the default itself may hold a plaintext key — no secret-typed key is registered in this release, so that masking path is exercised by the API tests rather than by a production key. A stored value that the current rules reject is still returned, marked `invalid` with the reason, so the page can show the operator what broke instead of silently replacing it. A failed read fails the whole endpoint instead of answering a schema assembled from defaults, because the page would then render plausible defaults as if they were what the database holds.
- `GET /api/config/schema` 一次请求返回整个设置页：所有已注册分组按 `Order` 排列，每个字段带 `type`、`label`、`description`、`default`、`min`/`max` 与 `min_exclusive`/`max_exclusive`、`pattern`、`options`、`item_type`、`unit`、`show_if`，以及当前状态 `value`、`has_value`、`invalid`、`invalid_reason`。快照由整张 `configs` 表一次读出，而不是每张卡片单独请求，因此卡片不会再拿一组内置默认值去保存、而那次迟迟未回的读取本应把它们替换掉。库中缺失的键返回其声明的默认值；`has_value` 表示是否存有非空值。secret 字段的值与默认值一律留空、只回传 `has_value`，因为默认值本身就可能带明文密钥——本版本没有注册任何 secret 类型的键，因此该掩码路径由 API 测试覆盖，而不是由某个线上键触发。库中已存、但按当前规则非法的取值照旧返回，并标为 `invalid` 且附上原因——让页面把坏在哪告诉运维，而不是静默替换掉。读取失败时整个端点报错，而不是返回一份由默认值拼成的 schema，否则页面会把看似合理的默认值当成库中已存的值渲染出来。

- `POST /api/config/group` saves one group strictly, as form fields `group=id` plus one `key=value` per field. It forwards only the group's own keys, read with `GetPostForm` so a missing field stays distinguishable from an empty one; an unknown group answers its own error before anything is written. On success nothing may have been skipped: a single bad or missing field rejects the whole call with HTTP 400 and `{msg:"配置校验失败", failures:[{key,msg}]}`, failures sorted by key, and no key is written. A clean save answers 「分组 «X» 已保存」 with the group's display title. All-or-nothing matters because a card posts every field it renders: a half-applied save would leave the page showing values the database does not hold.
- `POST /api/config/group` 严格保存单个分组，表单为 `group=id` 加每个字段一条 `key=value`。它只转交本分组自己的键，并用 `GetPostForm` 读取，以区分「字段缺失」与「值为空」；未知分组在写入前就以独立错误拒绝。成功时不允许有任何遗漏：任一字段非法或缺失都会令整次调用失败，返回 HTTP 400 与 `{msg:"配置校验失败", failures:[{key,msg}]}`，failures 按键排序，且一个键也不写入。全部合法时返回「分组 «X» 已保存」，X 为分组显示名。全有全无是必要的，因为卡片会提交它渲染的每一个字段：只写一半会让页面显示的值与库中实际值不一致。

- The settings page is rebuilt schema-driven instead of hand-written per feature card. A page-level load guard disables every input and save button, and offers 「重新加载」 once the read has actually failed, because a form still holding built-in defaults would blank the real configuration. Each field gets the control its type declares — text, textarea, password, number, checkbox, select, or a row editor for lists — and number inputs deliberately carry no `min`/`max`/`step` HTML attributes: the browser's step grid would judge a legal stored value invalid and block submission, so bounds are checked in JS honouring exclusive ends, with the server as backstop. Every field's description is an ⓘ tooltip opened on hover, keyboard focus and tap. Group cards collapse, with the collapsed set persisted in `localStorage` under `tagfile_config_collapsed`, and a group holding invalid values shows a visible 「N 项非法」 badge even while collapsed, so folding a card cannot hide the reason its save button is disabled. Each field has 恢复默认; each group saves on its own, and a rejected save marks the offending fields by key instead of only toasting the first message. The `scan_exclude` list is edited as regex rows, each row marked live once it fails to compile and serialized back to a JSON string array on save. The advanced area now holds only registry-unknown keys, with search, pagination, add, edit and delete, and the same info tips; the free-form description input is gone, since descriptions belong to the declaration, and the data export card is kept as it was. The old 「AI 识别」 card and the free KV form over known keys are removed — those keys are edited only through their schema group.
- 设置页改为由 schema 驱动重建，而不再按功能逐张手写卡片。页面级加载守卫会禁用所有输入与保存按钮，并在读取确实失败后给出「重新加载」，因为仍停留在内置默认值的表单会把真实配置清空。每个字段按声明的类型渲染控件——文本框、多行文本、密码框、数字框、勾选框、下拉选择，或列表的行编辑器——数字输入刻意不带 `min`/`max`/`step` 等 HTML 属性：浏览器的步进网格会把合法的存量值判为非法并阻止提交，因此边界改由 JS 判定且尊重开闭端，服务端兜底。每个字段的描述都是 ⓘ 提示，悬停、键盘聚焦与点按都能打开。分组卡片可折叠，折叠状态持久化在 `localStorage` 的 `tagfile_config_collapsed` 下；含非法值的分组即使处于折叠状态也显示可见的「N 项非法」徽标，折叠卡片因此无法隐藏保存按钮为何被禁用。每个字段都有「恢复默认」；每组独立保存，保存被拒时按配置键标出出错的字段，而不是只弹出第一条消息。`scan_exclude` 以正则行编辑，无法编译的行会即时标出，保存时序列化为 JSON 字符串数组。高级区现在只放注册表之外的键，保留搜索、分页、新增、编辑、删除与同样的信息提示；自由填写的描述输入框已移除，因为描述属于声明，数据导出卡片则原样保留。旧的「AI 识别」卡片与针对已知键的通用 KV 表单已删除——这些键只经各自的 schema 分组编辑。

### Changed / 变更

- `GET /api/config` items gain `description`, `?unknown=1` now lists only registry-unknown keys through the new service filters (with `q` searching that same subset and totals counted after filtering), and a registered secret key is masked: `value` is always empty and `has_value` replaces it, so the list view can never leak a stored key. Non-secret items are unchanged apart from `description`.
- `GET /api/config` 的条目新增 `description`；`?unknown=1` 改为经新的服务层过滤只列出注册表之外的键（`q` 在同一子集内搜索，总数在过滤后统计）；已注册的 secret 键一律掩码——`value` 恒为空、由 `has_value` 代替，列表视图因此不可能泄漏已存的密钥。非 secret 条目除 `description` 外不变。

- Known keys are edited through their schema group on the settings page, while the advanced area lists only unknown keys; the advanced add (`POST /api/config`) and `PUT /api/config/:key` endpoints still accept a known key name and validate it, and known keys remain protected by write validation and the delete guard. Secret values are never rendered: the page only learns whether a secret exists, and the group save omits an untouched hidden secret, which the service treats as leave-unchanged, so an untouched save cannot erase a stored key.
- 已知键在设置页经各自的 schema 分组编辑，高级区只列出注册表之外的键；高级新增（`POST /api/config`）与 `PUT /api/config/:key` 仍接受已知键名并照常校验，已知键继续受写入校验与删除守卫保护。secret 的值永不渲染：页面只得知某密钥是否存在，分组保存会省略未改动的隐藏密钥、服务端将其视为「保持不变」，因此未改动的保存不会抹掉已存密钥。

## [4.3.0] - 2026-11-14

### Added / 新增

- Seven AI endpoints in their own `/api/ai` group (`api.Router.RegisterRoutes`): `POST /tasks`, `GET /tasks`, `POST /tasks/retry`, `GET /suggestions`, `PUT /suggestions/:id`, `POST /suggestions/approve`, `POST /suggestions/reject`. The separate group is the feature's own namespace, a design choice rather than a routing constraint: gin only refuses a sibling at a `*path` wildcard's own position, and `/api/files` itself carries 14 more routes beside its two wildcards (`GET /detail/*path`, `GET /groups/*path`) — the one collision on record was that same-position case, `PUT /api/files/*path` moved to `PUT /api/files/rename` in `34bf78c`.
- 新增 7 个 AI 接口，独立挂在 `/api/ai` 分组下（`api.Router.RegisterRoutes`）：`POST /tasks`、`GET /tasks`、`POST /tasks/retry`、`GET /suggestions`、`PUT /suggestions/:id`、`POST /suggestions/approve`、`POST /suggestions/reject`。单开分组是给这个特性一份自己的命名空间，属于设计选择而非路由限制：gin 只拒绝与 `*path` 通配符同一位置的同级路由，而 `/api/files` 自己在两个通配符（`GET /detail/*path`、`GET /groups/*path`）之外还挂着 14 条路由——记录在案的冲突正是同位置那一例，即 `34bf78c` 把 `PUT /api/files/*path` 改成 `PUT /api/files/rename`。

- `POST /api/ai/tasks` takes either repeatable `paths` or one `dir` (plus `recursive`), both relative to the scan root exactly like `POST /api/files/scan`, and an optional `force`; `paths` wins and `dir` is then ignored rather than merged. It answers `{queued, skipped}` where `skipped` is a list of `{path, reason}`, so a partially accepted batch is a success. The call only writes queue rows and returns immediately — recognition itself happens in the background worker.
- `POST /api/ai/tasks` 接受可重复的 `paths`，或单个 `dir`（配 `recursive`），两者都与 `POST /api/files/scan` 一样相对扫描根目录，另有可选的 `force`；同时提交时以 `paths` 为准且不再合并 `dir`。响应为 `{queued, skipped}`，`skipped` 是 `{path, reason}` 列表，因此部分接受算成功。该请求只写队列表、立即返回，识别本身由后台 worker 完成。

- `GET /api/ai/tasks?status=&page=` lists tasks page size 20 with items `{task_id, file_id, file_path, file_name, status, error, frame_count, pending_count, created_at, updated_at}` plus the usual `pagination`. Timestamps are RFC3339 in UTC so both carry a `Z` and can be sorted or diffed as strings; `error` is the column verbatim and is not a failure flag — only `status` decides. `POST /api/ai/tasks/retry` takes a `task_id` and accepts a failed row or a `running` row no recognition is executing (a live one comes back 「正在执行中」), creating a new queued row rather than rewriting the old one so the error stays in the history.
- `GET /api/ai/tasks?status=&page=` 以每页 20 条返回任务，条目为 `{task_id, file_id, file_path, file_name, status, error, frame_count, pending_count, created_at, updated_at}` 并附常规 `pagination`。时间戳统一为 UTC 的 RFC3339，两者都带 `Z`，可直接按字符串排序或相减；`error` 是该列原样内容，并非失败标志——判定与否只看 `status`。`POST /api/ai/tasks/retry` 接收 `task_id`，接受失败任务，也接受没有任何识别在执行中的 `running` 任务（真正在跑的那条会返回「正在执行中」）；它新建一条 queued 行而不改写旧行，因此失败原因会留在历史里。

- `GET /api/ai/suggestions?status=&file_id=&page=` lists proposals, defaulting to `pending` i.e. the review queue; each item carries both halves — `category_name`/`tag_name`/`tag_value` exactly as the model said, never rewritten, and the reviewer-editable `final_category_name`/`final_tag_name`/`final_value`, plus `matched_tag_path` when the term already exists in the tree. `PUT /api/ai/suggestions/:id` rewrites the three effective values of a pending row only.
- `GET /api/ai/suggestions?status=&file_id=&page=` 列出建议，`status` 默认为 `pending`（即待审队列）；每条同时给出两半——模型原样输出且永不被改写的 `category_name`/`tag_name`/`tag_value`，以及审核者可改的 `final_category_name`/`final_tag_name`/`final_value`，若该词条已在标签树中则附 `matched_tag_path`。`PUT /api/ai/suggestions/:id` 只允许改写 pending 行的三个生效值。

- `POST /api/ai/suggestions/approve` and `/reject` take repeatable `ids` and answer `{approved|rejected, failures}`, where `failures` is a list of `{id, msg}` and is always an array rather than `null` when the batch was clean. A row that cannot be filed does not drag the batch down — including the case where the term collides with an existing root tag of another category, which the tag service refuses and the review row reports while staying pending.
- `POST /api/ai/suggestions/approve` 与 `/reject` 接收可重复的 `ids`，返回 `{approved|rejected, failures}`；`failures` 为 `{id, msg}` 列表，整批无一失败时返回空数组而非 `null`。单行入库失败不会拖垮整批——包括词条与其他分类下已有根标签重名的情形，标签服务会拒绝，审核行会原样上报并保持 pending。

- The recognition worker is started in `cmd/webui/main.go` after the database is open and before the listener takes a request, so its restart sweep cannot race with the first enqueue; `StopWorker` is deferred, which given `r.Run` means it runs when the listener fails rather than on Ctrl-C — a killed process leaves its `running` row for the next start's sweep to mark 重启中断, and one left behind inside a long-lived process is retryable from the review page.
- 识别 worker 在 `cmd/webui/main.go` 中于数据库打开之后、监听器接收首个请求之前启动，因而其重启清扫不会与首次入队竞争；`StopWorker` 以 defer 注册，考虑到 `r.Run` 会阻塞，它实际是在监听失败时执行而非 Ctrl-C——被杀掉的进程会遗留 `running` 行，交由下次启动的清扫标为 重启中断；而长期运行的进程内部遗留的 `running` 行也可在审核页直接重试。

- New 「待审队列」 page at `/review`, linked from the sidebar: an 发起识别 card (directory, 递归子目录, 强制重跑), a 任务状态 table that refreshes every 3 s and shows each task's status, frame count and failure reason with a per-row retry offered on 失败 and 识别中 rows (the server still refuses one the worker is executing), and a flat suggestion table with per-row checkboxes, select-all, inline editing of category/tag/value, and batch 通过/拒绝.
- 新增 `/review` 「待审队列」 页面，侧边栏有入口：包含「发起识别」卡片（目录、递归子目录、强制重跑）、每 3 秒自动刷新的「任务状态」表格（展示状态、抽帧数、失败原因，并对 失败 与 识别中 两种行给出逐行重试按钮——真正在执行中的那条仍由服务端拒绝），以及平铺的建议表格，支持逐行勾选、全选、行内修改分类/标签/值与批量「通过/拒绝」。

- The config page gains an 「AI 识别」 card for `ollama_url`, `ollama_model`, `ai_frame_interval`, `ai_frame_diff_threshold` and `ai_max_frames`, with the numeric ranges the backend enforces (threshold 1..64, max frames 1..200; the interval field's `min="0.1"` is a form constraint — the backend accepts any positive number) and a note that out-of-range or unparseable values fall back to defaults server-side. The interval field uses `step="0.1"` rather than `0.5` because a number input steps from its `min`, and with `min="0.1"` a `step` of 0.5 would judge the stored default `1` invalid and block the whole form from submitting. Saving is gated on a completed read (`aiLoaded`): the card posts all five keys at once, so a form still holding its built-in defaults would blank a stored model name, and while the read is outstanding or has failed the fields and the submit button are disabled and a banner says so, offering 「重新加载」 once the read has actually failed. The five keys are read page by page by the card itself rather than through `API.getAllPages`, because that helper answers with a bare item array and treats a page it refuses like the last one, so a truncated read would look exactly like a table that holds few rows.
- 配置页新增 「AI 识别」 卡片，覆盖 `ollama_url`、`ollama_model`、`ai_frame_interval`、`ai_frame_diff_threshold`、`ai_max_frames`，数值范围与后端强制的上界一致（阈值 1..64、最大帧数 1..200；间隔输入框的 `min="0.1"` 只是表单约束，后端接受任意正数），并附说明：越界或无法解析的值会在服务端回落到默认值。间隔输入用 `step="0.1"` 而非 `0.5`，因为 number 输入以 `min` 为步进基准，`min="0.1"` 配 0.5 的步进会把已存默认值 `1` 判为非法，导致整个表单无法提交。保存以“读取完成”（`aiLoaded`）为前提：该卡片会一次性提交五个键，仍停留在内置默认值的表单会把已存的模型名称清空，因此读取进行中或失败时，输入框与保存按钮一律禁用，横幅会说明这一点，并在确实读取失败后给出「重新加载」入口。这五个键由卡片自己逐页读取而没有用 `API.getAllPages`，因为后者只返回一个条目数组，遇到它不接受的那一页会像对待最后一页那样停下，读到一半被截断与表里本来条目就少，在调用方看来完全一样。

- File tree rows and the detail page show AI status as a dimension independent of the manual 已完成 state: `running`/`pending`/`approved`/`failed` render as 识别中/待审/AI 通过/识别失败, and a file with no `ai_status` renders no badge at all, so 「未识别」 is never invented as a fifth state. Indexed media rows gain a checkbox and the tree toolbar a batch 「识别选中(N)」 button (`tree-ai-batch-btn`), disabled while no row is checked and reading 「提交中...」 while the request is in flight; the detail page offers 「AI 识别」 or, for a file already recognised, 「重新识别」 behind a confirmation (`force=true`) and updates its own badge optimistically instead of reloading, since the page navigates with `pushState`.
- 文件树行与详情页把 AI 状态作为独立于人工「已完成」的维度展示：`running`/`pending`/`approved`/`failed` 分别渲染为 识别中/待审/AI 通过/识别失败，没有 `ai_status` 的文件不渲染任何徽章，因此「未识别」不会被臆造成第五种状态。已索引的媒体行新增勾选框，工具栏新增批量「识别选中(N)」按钮（`tree-ai-batch-btn`），未勾选任何行时禁用、提交过程中文案改为「提交中...」；详情页在从未识别或识别失败时给出「AI 识别」，否则给出需二次确认的「重新识别」（`force=true`），并在成功后就地乐观更新徽章而不整页刷新，因为该页用 `pushState` 导航。

### Changed / 变更

- The existing file endpoints gain one optional key and nothing else: `FileItem` (from `GET /api/files/expand` and the `file`/`dir_files` of `GET /api/files/detail/*path`) and `FileTreeNode` (from `GET /api/files`) now carry `ai_status`. It is `omitempty`, so a file that has never been recognised presents an existing consumer with exactly the keys it saw in 4.2.1. No existing key was renamed, retyped or removed.
- 既有文件接口只新增一个可选键：`FileItem`（来自 `GET /api/files/expand` 与 `GET /api/files/detail/*path` 的 `file`/`dir_files`）与 `FileTreeNode`（来自 `GET /api/files`）新增 `ai_status`。该键为 `omitempty`，因此从未识别过的文件在既有消费方看来键集合与 4.2.1 完全一致。没有任何既有键被重命名、改类型或移除。

## [4.2.5] - 2026-11-13

### Changed / 变更

- File detail and tag-group wildcard paths now share one normalizer that strips the leading slash gin adds before URL-decoding. Regression tests cover top-level, nested, spaced and percent-encoded registered paths for both endpoints.
- 文件详情与标签组通配路径改用同一个规范化函数，在 URL 解码前剥离 gin 添加的前导斜杠。两个端点的回归测试均覆盖顶层、嵌套、含空格与百分号编码的已注册路径。

## [4.2.4] - 2026-11-12

### Added / 新增

- New "随机打标" page at `/files/recommend`: draws a random file and opens the full tagging UI in place — preview, tag search/add/remove, tag groups, rename, mark done. A scope selector (待完成打标 / 尚无标签 / 全部文件), a 换一个 button, and a readout of pool size, remaining candidates and how many were shown this session.
- 新增“随机打标”页 `/files/recommend`：随机抽一个文件并就地展开完整打标界面——预览、标签搜索/增删、标签组、重命名、标记完成。带范围选择（待完成打标 / 尚无标签 / 全部文件）、“换一个”按钮，以及候选总量、剩余数量、本次已推荐数的显示。
- Files already recommended are remembered in `sessionStorage` (capped at 500) and skipped, so 换一个 does not repeat until the pool runs out; then it restarts from the whole scope with a toast. An empty scope shows its own hint instead of an error.
- 已推荐过的文件记在 `sessionStorage`（上限 500）并被跳过，因此“换一个”在候选耗尽前不会重复；耗尽后从整个范围重新开始并弹提示。范围为空时显示专门的提示，而不是报错。
- `GET /api/files/random?scope=&exclude=` returns the picked file plus `pool_total`, `remaining` and `exhausted`. An empty scope is `ok:true` with `file:null`; an unknown scope is a 400.
- `GET /api/files/random?scope=&exclude=` 返回抽中的文件以及 `pool_total`、`remaining`、`exhausted`。范围为空是 `ok:true` + `file:null`；未知范围返回 400。
- Integration tests for the endpoint (scope reach-through, exclusion, exhaustion, empty pool, unknown scope) and the route added to the smoke net.
- 新增该接口的集成测试（范围透传、排除、耗尽、空池、未知范围），并把路由纳入 smoke 防护网。

### Changed / 变更

- The detail page component now takes a recommend flag, so `/files/recommend` and `/files/detail/*path` share one template and one JS module rather than duplicating the tagging UI.
- 详情页组件新增 recommend 标志，`/files/recommend` 与 `/files/detail/*path` 共用同一模板与同一 JS 模块，不再复制打标界面。

## [4.2.3] - 2026-11-12

### Fixed / 修复

- File tree lazy expansion works again: the re-render hook watched `treeHtml`, which is a method and therefore never changes, so `afterRender` never ran and freshly rendered `<details>` nodes had no `toggle` listener — opening a directory fetched nothing. The watch now covers `tree`, `sortField` and `sortOrder`.
- 文件树目录懒加载恢复正常：重渲染钩子监听的是 `treeHtml`，而那是一个方法、永远不会变，于是 `afterRender` 从不执行，刚渲染出的 `<details>` 节点没有 `toggle` 监听器——展开目录什么都取不到。现在监听 `tree`、`sortField`、`sortOrder`。
- The config page export link points at `/api/export`. The route is registered on the `/api` group, so the old `/export` href only returned 404.
- 配置页导出链接改指 `/api/export`。该路由注册在 `/api` 分组下，原先的 `/export` 只会返回 404。
- `FilesystemRoutes` smoke test scans inside the temporary workspace instead of a sibling temp dir, matching the path basis the core now enforces.
- `FilesystemRoutes` smoke 测试改为在临时工作目录内扫描，不再用平级的另一个临时目录，以匹配核心层现在强制的路径基准。

### Changed / 变更

- Dockerfile base image is `golang:1.25-alpine`. `go.mod` requires `go 1.25.0`, and the official 1.24 image sets `GOTOOLCHAIN=local`, so the build failed before compiling.
- Dockerfile 基础镜像改为 `golang:1.25-alpine`。`go.mod` 要求 `go 1.25.0`，而官方 1.24 镜像设置了 `GOTOOLCHAIN=local`，构建在编译前就会失败。

## [4.2.2] - 2026-11-11

### Fixed / 修复

- DELETE endpoints (`/api/files/tag`, `/api/files/group/:id`, `/api/tags/values/:id`) read parameters from the URL query again. The gin migration switched `r.FormValue` to `c.PostForm`, which never parses a DELETE request, so untagging, group removal and predefined-value removal all failed silently.
- DELETE 接口（`/api/files/tag`、`/api/files/group/:id`、`/api/tags/values/:id`）恢复从 URL query 取参。gin 迁移把 `r.FormValue` 换成 `c.PostForm`，而后者根本不解析 DELETE 请求，导致取消打标、删除标签组、删除预定义值全部静默失败。

- Wildcard routes (`/api/files/detail/*path`, `/api/files/groups/*path`) now strip the leading slash gin adds to `c.Param("path")`. Without it the lookup key never matched the indexed relative path, so the file detail and tag group pages were empty for every file.
- 通配路由（`/api/files/detail/*path`、`/api/files/groups/*path`）现在会剥掉 gin 给 `c.Param("path")` 附加的前导斜杠。此前查找键永远匹配不上索引里的相对路径，导致所有文件的详情页与标签组页都是空的。

### Added / 新增

- First integration tests for the WebUI API layer: DELETE parameter contract, wildcard path resolution, and a smoke net covering every registered `/api` route.
- WebUI API 层首批集成测试：DELETE 参数契约、通配路径解析，以及覆盖全部已注册 `/api` 路由的 smoke 防护网。

## [4.2.1] - 2026-08-29

### Fixed / 修复

- Tag management page right panel no longer overflows its card background when viewport height is short.
- 标签管理页面右侧面板在视口高度较小时不再超出卡片背景。

## [4.2.0] - 2026-08-29

### Fixed / 修复

- `HandleTagFile` no longer silently falls back to default group when tagging to a specified group fails. The error is now returned to the user.
- 修复 `HandleTagFile` 在指定标签组打标失败时静默回退到默认组的问题，现在错误会返回给用户。
- `HandleUntagFile` same fix: no silent fallback to default group on error.
- `HandleUntagFile` 同样修复：错误时不再静默回退到默认组。
- `HandleRemoveTag` now deletes by tag ID instead of tag name, preventing potential cross-category data loss when tags share names.
- `HandleRemoveTag` 改为按标签 ID 删除而非按名字删除，避免跨分类同名标签误删。
- `HandleRemoveCategory`, `HandleAddTag`, `HandleListTags` now use `FindByID` instead of loading all categories to find one by ID.
- `HandleRemoveCategory`、`HandleAddTag`、`HandleListTags` 改用 `FindByID` 替代加载全部分类来按 ID 查找。

### Changed / 变更

- Tag path building optimized: `HandleFileDetail`, `HandleSearch`, and `HandleListTags` (flat view) now load only the tags needed for path building (target tags + ancestors) instead of all tags.
- 标签路径构建优化：`HandleFileDetail`、`HandleSearch`、`HandleListTags`（平铺视图）现在只加载构建路径所需的标签（目标标签 + 祖先标签），不再加载全部标签。
- Search now uses SQL-level pagination (`SearchByQueryPaged`) instead of loading all matching files into memory then slicing in Go.
- 搜索改用 SQL 层面分页（`SearchByQueryPaged`），不再将所有匹配文件加载到内存后在 Go 层切片。

## [4.1.0] - 2026-08-29

### Changed / 变更

- File tree no longer performs a full recursive filesystem scan (`filepath.WalkDir`) on page load. The tree is now built from the database; unindexed files in a directory are discovered lazily via `GET /api/files/expand` when the user expands that directory (single-level `os.ReadDir`).
- 文件树不再在页面加载时执行全量递归文件系统扫描（`filepath.WalkDir`）。现在文件树从数据库构建；展开目录时通过 `GET /api/files/expand` 懒加载该目录的未索引文件（单层 `os.ReadDir`）。

### Added / 新增

- New API endpoint `GET /api/files/expand?dir=xxx` returns indexed and unindexed direct children of a directory. Backed by `file.Service.ExpandDir()`.
- 新增 API `GET /api/files/expand?dir=xxx`，返回目录下的已索引和未索引直接子项。由 `file.Service.ExpandDir()` 支持。

## [4.0.0] - 2026-08-28

### Changed / 变更

- Refactored HTTP layer from `net/http` to Gin framework with route groups, Logger and Recovery middleware.
- 重构 HTTP 层，从 `net/http` 迁移到 Gin 框架，引入路由分组、Logger 和 Recovery 中间件。
- CLI argument parsing now uses cobra (consistent with the CLI binary).
- 命令行参数解析改用 cobra（与 CLI 保持一致）。

## [3.3.0] - 2026-08-16

### Added / 新增

- File tree: directories now show a nested done/total counter ("✓2/4"), aggregating the whole subtree — done = files marked 打标完成, total = all files (indexed or not). Computed in `computeDirCounts` after `markTreeIndexed`; the FS-tree cache is untouched.
- 文件树：目录行现在显示嵌套的"已完成/总数"计数（如"✓2/4"），递归统计整棵子树——已完成=标记为打标完成的文件数，总数=全部文件数（含未索引）。在 `markTreeIndexed` 后由 `computeDirCounts` 计算；文件系统树缓存不受影响。

## [3.2.9] - 2026-08-16

### Changed / 变更

- Tag tree drag zones: "拖到这里成为顶级标签" and "拖到这里取消移动" are now side by side in one fixed row above the scrolling list (previously stacked above/below the tree, scrolling with it). Drag logic unchanged — zones are located by id/class.
- 标签树拖拽区域："拖到这里成为顶级标签"和"拖到这里取消移动"现在并排固定在滚动列表上方的一行中（之前上下分列于树两侧并随列表滚动）。拖拽逻辑不变——按 id/class 定位。

## [3.2.8] - 2026-08-16

### Changed / 变更

- Tag page: only the tag list/tree area scrolls now. The content panel became a flex column (`#tag-list-wrap` is the sole scroll container); the controls row and the inline-add form stay fixed. Removed the inline `min-height` on the layout that broke the bounded height on small viewports.
- 标签页：现在只有标签列表/树区域滚动。内容面板改为 flex 列布局（`#tag-list-wrap` 是唯一滚动容器），顶部控件行和底部内联添加表单保持固定。移除了布局上内联的 `min-height`（小视口下会破坏高度约束）。

## [3.2.7] - 2026-08-16

### Fixed / 修复

- File detail page: after adding a tag, the tag search input kept the picked tag name and highlight. The shared `tagSearch` component now supports an optional `reset` getter (`cfg.reset`); the page bumps a counter after a successful add, clearing the input, selection highlight, and dropdown state.
- 文件详情页：添加标签后，标签搜索框仍保留已选标签名和高亮。共享 `tagSearch` 组件现在支持可选的 `reset` getter（`cfg.reset`）；页面在添加成功后递增计数，清空输入框、选中高亮和下拉状态。

## [3.2.6] - 2026-08-16

### Fixed / 修复

- Video preview still showed the previous file when switching between videos: the previous `x-for` key trick did not recreate the media element in Alpine 3.14, and changing `<source>` src does not trigger a browser reload. `src`/`type` are now bound directly on the `<video>`/`<audio>` element, which triggers the media load algorithm on attribute change (verified: `currentSrc` switches in headless Chrome). Also guarded sidebar `x-for` expressions against null `detail`, removing startup exceptions.
- 视频预览在视频文件间切换时仍显示旧文件：之前的 `x-for` key 方案在 Alpine 3.14 下不会重建媒体元素，而改 `<source>` 的 src 不会触发浏览器重新加载。现在 `src`/`type` 直接绑定在 `<video>`/`<audio>` 元素上，属性变化会触发媒体加载算法（headless Chrome 实测 `currentSrc` 已切换）。同时给侧边栏 `x-for` 表达式加了 null 守卫，消除启动时异常。

## [3.2.5] - 2026-08-16

### Fixed / 修复

- Tag page inline-add form: specifying a parent tag had no effect. The `tag-selected` listener was on the input, but `tagSearch` dispatches the event on its root element (which bubbles upward, never reaching the child input). Listener moved to the `x-data` root; manually typed parent names are also synced to the form state.
- 标签页内联添加表单：指定父标签不生效。`tag-selected` 监听挂在 input 上，但 `tagSearch` 在根元素上派发事件（向上冒泡，子元素 input 永远收不到）。监听移到 `x-data` 根元素，手动输入的父标签名也会同步到表单状态。

## [3.2.4] - 2026-08-16

### Fixed / 修复

- Modal form inputs could not accept text: Alpine 3.14.9 coerces `undefined` to `""` for dotted expressions, and `""` is truthy for boolean attributes, so `:readonly="f.readonly"` set `readonly` on every field. Fixed at the data layer — the modal store now normalizes `readonly`/`required` to explicit booleans when building fields, keeping template bindings declarative.
- 弹窗表单无法输入文字：Alpine 3.14.9 会把点路径表达式的 `undefined` 强转为 `""`，而 `""` 对布尔属性为真，导致 `:readonly="f.readonly"` 给所有字段都加上了 readonly。在数据层修复——弹窗 store 构建字段时将 `readonly`/`required` 规范化为显式布尔值，模板绑定保持声明式。

## [3.2.3] - 2026-08-16

### Fixed / 修复

- Inline edit (tag values on file detail, category names/descriptions, group names) never fired its save request: `save()` compared the typed value against the live `x-model` value, which is always equal. It now compares against the original value captured at init.
- 行内编辑（文件详情的标签值、分类名称/描述、标签组名称）从未发起保存请求：`save()` 将输入值与实时同步的 `x-model` 值比较，恒相等导致提前返回。现改为与初始化时记录的原值比较。

## [3.2.2] - 2026-08-16

### Fixed / 修复

- File detail page: tag preview chips are now vertically aligned with the "标签" label in the info table. Removed excess `margin-bottom` from `.tag-preview`.
- 文件详情页：信息表中标签预览芯片与"标签"标签现在垂直对齐，移除了 `.tag-preview` 多余的 `margin-bottom`。

- File detail page: fixed `onTagPicked` crash caused by accessing `evt.detail` when the handler already receives `$event.detail` as its argument.
- 文件详情页：修复 `onTagPicked` 因访问 `evt.detail`（应直接使用 `evt`）导致的崩溃。

- File detail page: toggling "标记完成" now immediately syncs the ✓ badge in the left sidebar file list.
- 文件详情页：切换"标记完成"后，左侧目录页的 ✓ 徽标会实时同步更新。

## [3.2.1] - 2026-08-16

### Added / 新增

- Version constant centralized in `cmd/webui/constant.WebuiVersion`. `main.go` imports from this package instead of hardcoding the version string.
- 版本常量集中在 `cmd/webui/constant.WebuiVersion`。`main.go` 从此包导入，不再硬编码版本号。

### Fixed / 修复

- File detail page: the inner sidebar (file list) and the right content panel now scroll independently. The outer `.main` container no longer scrolls when the detail layout is active.
- 文件详情页：内部侧边栏（文件列表）与右侧内容面板现在独立滚动。激活详情布局时，外层 `.main` 容器不再滚动。

## [3.2.0] - 2026-08-16

### Added / 新增

- Tag tree nodes with children now show a toggle caret (▶/▼) for collapse/expand. Clicking the caret collapses the subtree; clicking again expands it. Collapse state persists across re-renders (e.g., after drag moves or search).
- 标签树有子节点的标签现在显示折叠/展开三角（▶/▼）。点击三角可折叠子树，再次点击展开。折叠状态在拖拽移动或搜索后重新渲染时保持不变。
- Added "全部折叠" (Collapse All) and "全部展开" (Expand All) buttons in the tree view controls.
- 树形视图控制区新增"全部折叠"和"全部展开"按钮。

## [3.1.0] - 2026-08-16

### Changed / 变更

- Tag tree drag redesigned to target-highlight mode: SortableJS is now a pure drag engine (`onMove` always returns false, so the DOM never mutates during drag — no more jitter). The floating clone is styled as a compact chip (values/actions/subtree hidden); the original row shows as a dashed ghost marking the source position. Hovering a node highlights its whole row (drop = become its child, own subtree excluded); a dashed "拖到这里成为顶级标签" zone sits above the tree and "拖到这里取消移动" below it. Removed the per-leaf empty drop zones (tree is compact again) and the misleading sibling-reorder behavior (order is not persisted).
- 标签树拖拽重新设计为目标高亮模式：SortableJS 仅作拖拽引擎（`onMove` 恒返回 false，拖动过程 DOM 零变动——不再抽搐）。跟随光标的克隆样式化为紧凑小片（隐藏预设值/操作/子树）；原位置显示为虚线残影。悬停节点时整行高亮（放下即成为其子标签，自动排除自身子树）；树上方新增虚线"拖到这里成为顶级标签"区，下方为"拖到这里取消移动"区。删除叶子节点的空放置区（树恢复紧凑）与误导性的同级排序行为（顺序本就不持久化）。

## [3.0.3] - 2026-08-16

### Changed / 变更

- During tag tree drag, the ghost at the original position is now a slim accent-colored insertion line (4px) instead of a half-opacity full row; a dashed "拖到这里取消移动" drop zone was added below the tree — dropping a dragged tag there reverts the move without any API call.
- 标签树拖拽时落点指示改为 4px 高亮细线（原为半透明整行虚影）；树下方新增虚线"拖到这里取消移动"放置区，把标签拖入该区域即撤销移动，不发起任何 API 请求。

## [3.0.2] - 2026-08-16

### Fixed / 修复

- Drag handle on tag tree rows is now always visible (was `opacity:0` until row hover), and every node — including leaves — renders its children list as an empty drop zone (`min-height: 12px`), so a tag can be dragged to become the child of a leaf node.
- 标签树行的拖拽把手改为常显（原为悬停才显示），且每个节点（含叶子）都渲染子列表作为空放置区（`min-height: 12px`），现在可以把标签拖成叶子节点的子标签。

## [3.0.1] - 2026-08-16

### Fixed / 修复

- Tag tree drag-and-drop could never move a tag to root level: the drop target was resolved via `elementFromPoint`, so any drop landed on a sibling tag and made the dragged tag its child. Drop target is now the list the item was dropped into (SortableJS `evt.to`): dropping into the root list makes it a top-level tag, dropping into a node's children list makes it a child of that node.
- 标签树拖拽无法把标签移到根层：放置目标用 `elementFromPoint` 解析，任何放置都会命中某个同级标签并把被拖标签变成它的子标签。现改为按放置进入的列表（SortableJS `evt.to`）判定：放进根列表成为顶级标签，放进某节点的子列表成为该节点的子标签。

## [3.0.0] - 2026-08-16

### Changed / 变更

- Full WebUI refactor onto Alpine.js 3.14 + SortableJS 1.15 (vendored under `static/vendor/`, no CDN, no build step): all pages migrated from imperative vanilla JS to declarative Alpine components; tag tree drag-and-drop replaced with nested SortableJS; toast/confirm/form-modal rewritten as Alpine stores with the same public API (`showToast`/`showConfirm`/`Modal.show`); tag search autocomplete, inline edit, and sort controls extracted as shared Alpine components (`tagSearch`/`inlineEdit`). 24 vanilla JS files reduced to per-page component files. UI appearance unchanged.
- WebUI 全面重构到 Alpine.js 3.14 + SortableJS 1.15（本地 vendor 引入 `static/vendor/`，无 CDN、无构建步骤）：全部页面从命令式 vanilla JS 迁移为声明式 Alpine 组件；标签树拖拽替换为嵌套 SortableJS；toast/confirm/表单弹窗重写为 Alpine store（对外 API 不变：`showToast`/`showConfirm`/`Modal.show`）；标签搜索自动补全、内联编辑、排序控件抽为共享 Alpine 组件（`tagSearch`/`inlineEdit`）。24 个 vanilla JS 文件收敛为每页一个组件文件。UI 外观不变。

### Fixed / 修复

- Creating a new tag group always failed with "查询标签组失败: record not found" (`CreateGroup` treated gorm's ErrRecordNotFound as fatal).
- 创建新标签组始终失败并提示"查询标签组失败: record not found"（`CreateGroup` 将 gorm 的 record not found 误判为致命错误）。

## [2.2.0] - 2026-08-15

### Added / 新增

- Mark files as "tagging done": `POST /api/files/done` endpoint, toggle button on the file detail card, "已完成" status badge in the file tree, and ✓ badge in the detail sidebar. `FileItem`/`FileTreeNode` now carry `tag_done`.
- 文件"打标完成"标记：新增 `POST /api/files/done` 接口、文件详情卡片上的标记/取消按钮、文件树"已完成"状态徽标、详情页侧边栏 ✓ 徽标。`FileItem`/`FileTreeNode` 新增 `tag_done` 字段。

## [2.1.1] - 2026-07-12
### Fixed / 修复

- File tree items invisible after `staggerChildren()` refactor: `.tree-file` and `.tree-dir` missing `animation` property caused them to stay at `opacity: 0` from `.stagger > *` CSS rule. Added `animation: fadeUpIn 0.35s ease both` to match other staggered elements.
- 文件树列表不可见：`staggerChildren()` 重构后 `.tree-file` 和 `.tree-dir` 缺少 `animation` 属性，导致被 `.stagger > *` 的 `opacity: 0` 永久隐藏。补上 `fadeUpIn` 动画，与其他 stagger 元素保持一致。

### Added / 新增

- `clearChildren(el)` utility in `core/utils.js`: replaces 35 instances of `while(x.firstChild) x.removeChild(x.firstChild)` across 13 files. Uses native `replaceChildren()` for better performance.
- `clearChildren(el)` 工具函数：替代 13 个文件中 35 处重复的 DOM 清空模式，使用原生 `replaceChildren()` 提升性能。

- `staggerChildren(parent)` utility + `.stagger` CSS class: replaces 55 hardcoded `nth-child` CSS stagger rules with JS-driven `--stagger-i` custom property. Works with existing per-element `animation` rules.
- `staggerChildren()` 工具函数 + `.stagger` CSS 类：替代 55 条硬编码 nth-child stagger 规则，改用 JS 驱动的 CSS 自定义属性。

- `initSortControls(opts)` in `core/sort-controls.js`: shared sort control initialization with sessionStorage persistence, replacing duplicate implementations in file detail and file tree pages.
- `initSortControls(opts)` 共享排序控件初始化，消除文件详情和文件树页面的重复实现。

- `API.getAllPages(url, maxPages)` in `core/api.js`: generic paginated fetcher with 100-page safety limit, replacing 3 manual recursive pagination loops.
- `API.getAllPages()` 通用分页加载器，带 100 页安全上限，替代 3 处手动递归翻页。

- `API.loadAllTags()` in `core/api.js`: unified tag fetch+map helper, replacing duplicate implementations in file detail and search pages.
- `API.loadAllTags()` 统一标签加载+映射，消除文件详情和搜索页的重复逻辑。

- `API.onError` in `core/api.js`: shared error handler for `.catch()` chains, replacing 12 identical `showToast('请求失败', 'error')` catch blocks.
- `API.onError` 共享错误处理器，替代 12 处相同的 catch+toast 模式。

- `GET /api/tags/values/batch?names=A,B,C` bulk endpoint: returns all tag preset values in one request instead of N individual calls. Reduces tag management page from N HTTP requests to 1.
- `GET /api/tags/values/batch` 批量接口：一次请求返回所有标签预设值，标签管理页从 N 次请求降为 1 次。

- Tag management page `initTagValues()` refactored: collects unique tag names, fires one batch request, distributes results via `renderTagValueChips()`.
- 标签管理页 `initTagValues()` 重构：收集去重标签名，单次批量请求，通过 `renderTagValueChips()` 分发渲染。

### Changed / 变更

- Search page builder: global `onTagInputCreated` hook replaced with `initBuilder(opts)` configuration pattern. Tag autocomplete now wired immediately on input creation, fixing a race condition where restored search history inputs lost autocomplete.
- 搜索页构建器：全局 `onTagInputCreated` 钩子改为 `initBuilder(opts)` 配置模式。标签自动补全在 input 创建时立即绑定，修复历史搜索恢复时 input 缺失 autocomplete 的竞态问题。

- Error handling: button-reset + error-toast patterns in scan/import/search forms refactored to `.finally()` for button state + `.catch(API.onError)` for error display.
- 错误处理：扫描/导入/搜索表单的按钮重置+错误提示模式重构为 `.finally()` 管理按钮状态 + `.catch(API.onError)` 显示错误。

- `fetchAllTagsFromServer` wrapper in file detail page inlined and removed — callers now use `API.loadAllTags()` directly.
- 文件详情页 `fetchAllTagsFromServer` 包装函数内联删除，调用方直接使用 `API.loadAllTags()`。

- Raw `fetch()` calls in `detail-tags.js` (tag values) and `config/main.js` (config save) replaced with `API.get()` / `API.put()`.
- `detail-tags.js` 和 `config/main.js` 中的 raw `fetch()` 替换为 `API.get()` / `API.put()`。

- Search page `initBuilder()` moved from `main.js` to `builder.js` — builder owns its own initialization.
- 搜索页 `initBuilder()` 从 `main.js` 移至 `builder.js`，构建器自管初始化。

### Fixed / 修复

- File rename broken by `API.put()` double-wrapping `URLSearchParams`: passed plain object `{new_name: newName}` instead.
- 文件重命名功能修复：`API.put()` 双重包装 `URLSearchParams` 导致请求体损坏，改传 plain object。

- `showConfirm` event listener leak: `keydown` handler for Escape was not removed when modal was closed via backdrop click or cancel button. Now removed in `cleanup()`.
- `showConfirm` 事件监听器泄漏：通过背景点击或取消按钮关闭弹窗时 Escape 监听器未移除，现在统一在 `cleanup()` 中移除。

### Removed / 移除

- 7 dead legacy stub functions: `saveTag()`, `loadParentTags()`, `editTagTree()`, `saveTagTree()`, `quickAddChild()`, `saveQuickChild()`, `doAddGroup()`, `confirmAction()`, `addTagValue()`, `renderPagination()`.
- 7 个废弃遗留桩函数：`saveTag()`、`loadParentTags()`、`editTagTree()` 等。

## [2.0.3] - 2026-07-12
### Added / 新增

- SVG favicon: tag-shaped icon with indigo gradient, displayed in browser tab and sidebar brand.
- SVG 网站图标：标签形状靛蓝渐变图标，显示在浏览器标签页和侧边栏品牌名旁。

- Static file handler now serves `.svg` files with correct `image/svg+xml` Content-Type.
- 静态文件处理器现在正确设置 `.svg` 文件的 `image/svg+xml` MIME 类型。

## [2.0.2] - 2026-07-12
### Fixed / 修复

- File detail page TOC sidebar: adjusted sticky top offset to avoid overlapping page title, added independent internal scrolling.
- 文件详情页 TOC 侧边栏：调整 sticky 顶部偏移避免遮挡页面标题，添加独立内部滚动。

## [2.0.1] - 2026-07-12
### Fixed / 修复

- Page title header: fixed sticky positioning for proper scroll pinning, added rounded corners and glassmorphism background, adjusted width to match content cards.
- 页面标题栏：修复 sticky 定位实现滚动固定，添加圆角和毛玻璃背景，宽度与内容卡片对齐。

- Sidebar navigation: replaced side-stripe border with rounded pill-style tabs and glassmorphism active state.
- 侧边导航：将侧边条替换为圆角胶囊样式标签页，激活态使用毛玻璃效果。

## [2.0.0] - 2026-07-12

### Added / 新增

- Glassmorphism theme: all components (cards, sidebar, buttons, inputs, tags, tables, modals, toasts) now use translucent backgrounds with `backdrop-filter: blur()` and soft shadows.
- 玻璃拟态主题：所有组件（卡片、侧边栏、按钮、输入框、标签、表格、弹窗、通知）使用半透明背景 + `backdrop-filter: blur()` 柔和阴影。

- Dark/light theme toggle: sun/moon button in sidebar footer, persists to localStorage, auto-detects `prefers-color-scheme`.
- 深浅主题切换：侧边栏底部太阳/月亮按钮，localStorage 持久化，自动跟随系统 `prefers-color-scheme`。

- CSS Layers (`@layer reset, base, layout, components, utilities`) for style organization and specificity control.
- CSS Layers（`@layer reset, base, layout, components, utilities`）组织样式层级和优先级。

- Ambient glow effects: radial gradient blobs behind the page background for depth.
- 环境光效：页面背景后方的径向渐变光斑，增强层次感。

- Page title underline accent decoration.
- 页面标题下划线装饰。

### Changed / 变更

- Color system: replaced steel-blue (`#5a8db8`) with indigo (`#6366f1` / `#818cf8`) accent. Dual-theme variables with OKLCH-tinted neutrals.
- 色彩系统：强调色从钢蓝（`#5a8db8`）改为靛蓝（`#6366f1` / `#818cf8`）。双主题变量，中性色带品牌色调。

- Border radius: increased from 2px to 8px globally.
- 圆角：全局从 2px 提升至 8px。

- Background: replaced flat `#fafaf8` with multi-stop gradient (`#e8f0fe` → `#ede7f6` → `#fce4ec`).
- 背景：从纯色 `#fafaf8` 改为多段渐变（`#e8f0fe` → `#ede7f6` → `#fce4ec`）。

- Sidebar: translucent glass with inner glow on active nav item.
- 侧边栏：半透明玻璃质感，活跃导航项带内发光。

- Buttons: primary button gets glow shadow; default button gets glass backdrop.
- 按钮：主按钮带发光阴影；默认按钮带玻璃模糊背景。

- Inputs: glass background with accent glow on focus ring.
- 输入框：玻璃背景，聚焦时带强调色光环。

- Scrollbar: custom thin glass-style scrollbar.
- 滚动条：自定义细玻璃风格。

## [1.3.0] - 2026-07-12

### Added / 新增

- Structured query builder for search: replaces tag-name-only search with a visual condition builder supporting file name, tag, tag value, file size, file extension, and modification date filters. Conditions can be combined with AND/OR logic and nested in groups.
- 结构化查询构建器：将仅支持标签名的搜索替换为可视化条件构建器，支持文件名、标签、标签值、文件大小、文件扩展名、修改日期过滤。条件可 AND/OR 组合并嵌套分组。

- `POST /api/search` now accepts structured `query` JSON parameter instead of flat `include`/`exclude`/`logic`. Supports recursive condition groups with arbitrary nesting depth.
- `POST /api/search` 改为接受结构化 `query` JSON 参数，替代原有的 `include`/`exclude`/`logic`。支持任意嵌套深度的递归条件组。

- Search history stores full query objects, restores complete builder state on click.
- 搜索历史存储完整查询对象，点击可恢复完整的构建器状态。

### Changed / 变更

- Search API breaking change: `include`, `exclude`, `logic` parameters removed. Only `query` JSON string accepted.
- 搜索 API 不兼容变更：移除 `include`、`exclude`、`logic` 参数，仅接受 `query` JSON 字符串。

## [1.2.7] - 2026-06-01

### Added / 新增

- `POST /api/search` JSON endpoint: tag-based file search with pagination. Accepts `include`, `exclude` (comma-separated tag names), `logic` (and/or), `page`, `page_size`. Returns paginated results with file metadata and tag paths.
- `POST /api/search` JSON 端点：基于标签的文件搜索，支持分页。接受 `include`、`exclude`（逗号分隔标签名）、`logic`（and/or）、`page`、`page_size`。返回分页结果含文件元数据和标签路径。

- Search page tag autocomplete: both include and exclude inputs show filtered tag dropdown while typing, with comma-separated multi-tag support. Tags already entered are excluded from candidates.
- 搜索页标签自动补全：包含和排除输入框在输入时显示过滤后的标签下拉列表，支持逗号分隔多标签。已输入的标签自动排除。

- Search page result pagination: paginated result display with prev/next navigation.
- 搜索页结果分页：分页展示搜索结果，支持前后翻页。

- Search history: recent searches saved to localStorage (max 10), displayed as clickable chips below the search form. Click restores conditions and triggers search. "清除" button clears history.
- 搜索历史：最近搜索保存至 localStorage（最多 10 条），以可点击 chip 形式展示在搜索表单下方。点击恢复条件并触发搜索。「清除」按钮清空历史。

### Changed / 变更

- Search page converted from server-side form POST rendering to AJAX-driven client-side rendering. All result rendering uses `createElement` (zero innerHTML), consistent with other pages since v1.2.0.
- 搜索页从服务端 form POST 渲染改为 AJAX 驱动的客户端渲染。结果渲染全部使用 `createElement`（零 innerHTML），与 v1.2.0 以来其他页面一致。

- Handler layer (`handlers/`) no longer holds any service dependencies. `Server.SearchSvc` removed; `POST /search` route and `HandleSearch` deleted. Handler layer is now a pure template shell.
- Handler 层（`handlers/`）不再持有任何 service 依赖。`Server.SearchSvc` 已移除；`POST /search` 路由和 `HandleSearch` 已删除。Handler 层现在是纯模板渲染壳。

### Removed / 移除

- `handlers/search.go`: server-side search handler replaced by `api/search.go`.
- `handlers/search.go`：服务端搜索 handler 已被 `api/search.go` 替代。

## [1.2.6] - 2026-05-17

### Changed / 变更

- Tag management page sidebar restructured: old tag tree panel replaced with category filter list sidebar matching file detail page `.dir-sidebar` style — "全部" + per-category items with tag counts. Category dropdown removed from right panel search form; category filtering now exclusively through sidebar clicks.
- 标签管理页侧边栏重构：旧的分类标签树面板替换为与文件详情页 `.dir-sidebar` 风格一致的分类筛选列表 — "全部" + 各分类项附带标签计数。右侧面板搜索表单中移除分类下拉框，分类筛选统一通过侧边栏点击。

- Tag page inline `<style>` block extracted to proper CSS files: `.tag-content-panel` / `.tag-tree-node` / `.tag-tree-row-main` / drag-drop styles → `tree.css`; table row animations + `.col-actions-inner` → `table.css`; `.dir-sidebar-static` → `layout.css`. Removed dead `.tag-tree-panel` / `.tag-tree-row` / `.tag-tree-list` / `.tag-tree-children` CSS from `tree.css`.
- 标签页内联 `<style>` 块迁移至 CSS 文件：`.tag-content-panel` / `.tag-tree-node` / `.tag-tree-row-main` / 拖拽样式 → `tree.css`；表格行动画 + `.col-actions-inner` → `table.css`；`.dir-sidebar-static` → `layout.css`。清理 `tree.css` 中已废弃的 `.tag-tree-panel` / `.tag-tree-row` / `.tag-tree-list` / `.tag-tree-children` 样式。

- Sidebar and tag content panel now have fade-up entrance animation on page load.
- 侧边栏和标签内容面板页面加载时新增淡入动画。

- File detail sidebar removed `←` current-file arrow marker; highlight alone is sufficient.
- 文件详情侧边栏移除当前文件的 `←` 箭头标记，高亮效果已足够。

- File tree sort: directories now participate in same-level sorting — descending puts dirs first then files, ascending puts files first then dirs. Name used as tiebreaker when primary field values are equal (e.g. dirs sorted by size).
- 文件树排序：目录参与同层级排序 — 降序时目录在前文件在后，升序时文件在前目录在后。主字段值相等时（如目录按大小排序）以名称作为次级排序依据。

- Sort settings persist across page reloads via sessionStorage (field + order) on both file tree page and file detail sidebar.
- 排序设置在页面重载后通过 sessionStorage 保持（字段 + 方向），文件树页面和文件详情侧边栏均生效。

- File detail page: navigating between files in the same directory now updates in-place via History API without full page reload — only the content panel refreshes, sidebar stays and just updates the current-file highlight.
- 文件详情页：同目录下切换文件通过 History API 原地更新，不再整页重载 — 仅内容面板刷新，侧边栏保持不变仅更新当前文件高亮。

- File detail page: missing file now shows a dedicated "file not found" empty state with hint text instead of a generic error.
- 文件详情页：文件不存在时显示专门的"文件不存在"空状态提示，不再显示通用错误。

## [1.2.5] - 2026-05-17

### Added / 新增

- File list sorting: sort controls on file tree page and file detail sidebar — field dropdown (name/size/tag count/type) + order toggle button (asc/desc). Sidebar sorting uses server-side `?sort=&order=` params; tree page sorts client-side within each directory level. Sort change on detail page only refreshes sidebar, not the main panel.
- 文件列表排序：文件树页面和详情页侧边栏新增排序控件 — 字段下拉（名称/大小/标签数/类型）+ 方向切换按钮（升序/降序）。侧边栏排序通过服务端 `?sort=&order=` 参数；树页面客户端目录内排序。详情页切换排序仅刷新侧边栏，不重载主内容面板。

- File info card tag preview auto-refresh: after tag CRUD in the tag list below, the tag preview chips and count in the file info card update synchronously.
- 文件信息卡片标签预览自动刷新：下方标签列表 CRUD 后，文件信息卡中的标签预览 chips 和计数同步更新。

- Sidebar directory file list staggered fade-in animation: each file item fades in with incremental delay on page load.
- 侧边栏目录文件列表渐入动画：页面加载时文件项依次淡入。

### Changed / 变更

- Sort controls use `<select>` for field picker + `<button>` for order toggle (with text label `升序 ▴` / `降序 ▾`), avoiding unnecessary UI clutter.
- 排序控件使用 `<select>` 字段下拉 + `<button>` 方向切换（带文字 `升序 ▴` / `降序 ▾`），保持界面简洁。

## [1.2.4] - 2026-05-17

### Fixed / 修复

- Button/input height mismatch: removed `btn-sm` from buttons placed next to full-size inputs across 4 locations — file detail add-tag form, category/tag/config search forms, config cancel button.
- 按钮/输入框高度不一致：移除 4 处与正常大小输入框相邻的 `btn-sm`（文件详情添加标签、分类/标签/配置搜索表单、配置取消按钮）。

- File import form: removed `align-self:end` from browse button so it aligns naturally with the adjacent path input.
- 文件导入表单：移除浏览按钮的 `align-self:end`，与路径输入框自然对齐。

## [1.2.3] - 2026-05-17

### Changed / 变更

- File groups page: removed "重命名" and "查看标签" buttons. Group name is now editable via click-to-edit (click the cell directly). Only "删除" button remains for non-default groups.
- 标签组管理页：删除「重命名」和「查看标签」按钮，组名改为 click-to-edit（直接点击单元格编辑）。非默认组仅保留「删除」按钮。

## [1.2.2] - 2026-05-17

### Changed / 变更

- Confirm messages use actual Unicode characters `«` `»` instead of HTML entities `&laquo;` `&raquo;`, eliminating the need for entity decoding.
- 确认弹窗消息直接使用 Unicode 字符 `«` `»` 替代 HTML 实体，不再需要解码处理。

## [1.2.1] - 2026-05-17

### Fixed / 修复

- Confirm modal HTML entity decoding: `showConfirm()` used `textContent` which does not decode HTML entities — `&laquo;` / `&raquo;` displayed as literal text. Fixed by decoding via a temp div's `innerHTML` before setting `textContent`.
- 确认弹窗 HTML 实体解码修复：`showConfirm()` 使用 `textContent` 不会解码 `&laquo;` / `&raquo;`，改为通过临时 div 的 `innerHTML` 解码后再设为 `textContent`。

## [1.2.0] - 2026-05-17

### Changed / 变更

- All frontend JS files rewritten to eliminate raw HTML string concatenation (`innerHTML = '<div>...'`). Every DOM node is now created via `document.createElement` / `createEl()` helper. Zero `innerHTML` assignments with inline HTML remain in the codebase.
- 前端 JS 全面重写，消除所有 HTML 字符串拼接（`innerHTML = '<div>...'`）。所有 DOM 节点改用 `document.createElement` / `createEl()` 创建，代码库中不再有 innerHTML 赋值内联 HTML。

- New `createEl(tag, attrs, ...children)` utility in `core/utils.js`: lightweight DOM builder supporting className, textContent, style string, `on*` event listeners, and mixed text/Node children.
- 新增 `createEl(tag, attrs, ...children)` 工具函数：轻量 DOM 构建器，支持 className、textContent、style 字符串、事件监听器、文本/Node 混合子节点。

- Extracted three reusable DOM builder helpers in `core/utils.js`:
  - `makeEmptyState(icon, text, hint?)` — centered empty state placeholder (used in ~20 locations)
  - `makePagination(pg, onPageFn, opts?)` — pagination bar with prev/next + page info (used in 5 pages)
  - `makeActionBtn(text, className, onClick)` — action button shorthand
- 提取三个可复用 DOM 构建 helper：
  - `makeEmptyState(icon, text, hint?)` — 居中空状态占位（~20 处使用）
  - `makePagination(pg, onPageFn, opts?)` — 分页栏（5 个页面使用）
  - `makeActionBtn(text, className, onClick)` — 操作按钮简写

- 13 JS files refactored: `core/app.js`, `core/utils.js`, `pages/category/main.js`, `pages/config/main.js`, `pages/file/detail-core.js`, `pages/file/detail-tags.js`, `pages/file/groups.js`, `pages/file/tree.js`, `pages/tag/list-flat.js`, `pages/tag/list-tree.js`, `pages/tag/main.js`, `pages/tag/sidebar.js`, `pages/tag/values.js`.
- 13 个 JS 文件重构完成。

- `onclick` handlers switched from string-based `onclick="fn(...)"` to closure-based `onclick: function() { fn(...); }` for proper variable scoping.
- `onclick` 处理从字符串 `onclick="fn(...)"` 改为闭包 `onclick: function() { fn(...); }`，确保变量作用域正确。

- Tree rendering functions (`renderTreeNodes`, `renderSideTreeNodes`) changed from returning HTML strings to appending directly to a parent DOM node.
- 树渲染函数从返回 HTML 字符串改为直接向父节点追加 DOM。

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
