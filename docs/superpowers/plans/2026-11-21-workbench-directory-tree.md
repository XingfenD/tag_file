# 工作台左栏完整折叠目录树 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把文件列表页已有的完整折叠目录树抽成共享 Alpine 组件 `directoryTree`，列表页行为零漂移，工作台左栏用它直接替换「当前目录两张平面列表」。

**Architecture:** 新组件 `static/js/core/tree.js` 承载全部树机制（`GET /api/files` 加载、逐级排序、`<details>` 渲染、`tagfile_tree_open:*` 折叠记忆、`GET /api/files/expand?dir=` 未索引懒加载、行构造、toggle 监听重挂），页面通过 config 闭包注入差异（勾选列开关、文件点击行为、当前路径高亮）。后端 API 零改动。

**Tech Stack:** Go 1.25 + Gin 模板 + Alpine.js（无构建、无 JS 测试框架；JS 约定靠 `cmd/webui/*_test.go` 的静态哨兵钉住）+ CSS 分层（tree.css 已有树样式，workbench.css 管工作台）。

**Spec:** `docs/superpowers/specs/2026-11-21-workbench-directory-tree-design.md`

## Global Constraints

- 分支 `feat/detail-workbench`；提交前 `git branch --show-current` 不得在 master。
- 只改 `cmd/webui/`；WebUI 变更记 `docs/CHANGELOG_webui.md`（英文行紧跟中文行为同一条，条目间空一行，高版本在上），本特性开 **5.21.0** 条目并在同一 commit bump `cmd/webui/constant/WebuiVersion`。
- Alpine 无递归模板：树必须保持「JS 拼 HTML 字符串 + x-html、渲染后 `afterRender` 重挂监听」的现有模式。
- 模板规矩：`{{define}}` 不嵌套；页面级注册三件套（main.go 模板名、routes、nav）本任务不涉及（无新页面）。
- `handler 读参` 规矩不涉及（无新 endpoint）。
- 测试命令：`go test ./cmd/webui/...`；部署验证走 Docker（`docker build -t tagfile-webui . && docker rm -f tagfile-webui && docker run -d --name tagfile-webui -p 8088:8088 -v "/Users/xingfend/Movies/暂时:/data" --restart unless-stopped tagfile-webui`）。

---

### Task 1: 共享组件 `core/tree.js`（新建 + 挂载 + 存在性哨兵）

**Files:**
- Create: `cmd/webui/static/js/core/tree.js`
- Modify: `cmd/webui/templates/base.html`（在 `/static/js/core/components.js` 之后加一行 `<script src="/static/js/core/tree.js"></script>`）
- Test: `cmd/webui/static_test.go`（新增 `TestDirectoryTreeComponent`）

**Interfaces:**
- Produces: `Alpine.data('directoryTree', function(cfg) {...})`，config 面（Task 2/3 全靠这个契约，名字与语义不得偏离）：
  - `cfg.sort: () => ({field, order})` — 每次渲染时读的闭包，读的是**页面作用域**的排序状态（Alpine 元素作用域编译，闭包内 `sortField`/`sortOrder` 保持响应式）。
  - `cfg.picks: boolean` — 为真时每行渲染 `.tree-pick` 勾选格，仅已索引媒体文件带 checkbox。
  - `cfg.onTogglePick: (path, checked) => void` — checkbox change 回调（勾选状态的所有权留在页面）。
  - `cfg.getPicks: () => string[]` — afterRender 同步 checkbox 时读取。
  - `cfg.onFileClick: (node) => void | null` — 非空时点击 `.tree-file a` 被 `preventDefault` 后调用（`node` 含 `{name, path}`）；为空时保持原生 `<a href>` 整页导航。
  - `cfg.getCurrent: () => string | ''` — 当前文件相对路径，渲染后打 `.tree-current`。
  - 组件方法（页面经 `Alpine.$data(el)` 调用）：`load()`（重取全树，清 `expandedDirs`）、`reveal(path)`（展开祖先 `<details>`、打 `.tree-current`、`scrollIntoView({block:'nearest'})`）、`refreshFileRow(fields)`（`fields = {path, tag_cnt?, tag_done?, ai_status?}`：在 `this.tree` 里按 path 找到节点、`Object.assign` 后按 `[data-file-path]` 换掉那一行）。
  - 状态：`tree`、`loading`、`error`、`expandedDirs`、方法 `treeHtml()`/`fileRowHtml(n)`/`sortedNodes(nodes)`/`afterRender()`/`expandDir(detailsEl, dirPath)`——签名与 `pages/file/tree.js:54-272` 现有实现一致，逐段搬迁而非重写。
  - DOM 契约：行 `<div class="tree-file" data-file-path="…">`（新加属性）；目录 `<details class="tree-dir" data-tree-path="…">`（不变）；折叠记忆键 `'tagfile_tree_open:' + dirPath`（不变，与现列表页共用命名空间，两页互通是 spec 决策）。
  - 根元素 `<div class="tree-root" x-show="!loading && !error" x-html="treeHtml()">`；`afterRender` 容器查找由页面的 `#file-tree .tree-root` 改为组件自身根内 `querySelector('.tree-root')`。

- [ ] **Step 1: 写失败哨兵** `static_test.go` 新增 `TestDirectoryTreeComponent`：读 `staticFS`，断言 `static/js/core/tree.js` 含 `Alpine.data('directoryTree'`、`tagfile_tree_open:`、`data-file-path`；再断言 `templates/base.html` 含 `/static/js/core/tree.js`。

- [ ] **Step 2: 跑失败** `go test ./cmd/webui/ -run TestDirectoryTreeComponent -v` → FAIL（文件不存在）。

- [ ] **Step 3: 搬迁实现。** 从 `pages/file/tree.js` 复制 `load/sortedNodes/treeHtml/fileRowHtml/aiBadgeHtml/isAiMedia/afterRender/bindPickEvents/syncPicks/expandDir` 进新文件，按 Interfaces 改参数来源：排序读 `cfg.sort()`；勾选列由 `cfg.picks` 开关、change 事件转 `cfg.onTogglePick`、同步读 `cfg.getPicks()`；行构造给 `.tree-file` 加 `data-file-path`；bindPickEvents 容器上追加一个 click 委托：`cfg.onFileClick` 存在时拦 `.tree-file a`（`e.preventDefault(); cfg.onFileClick(...)`，path→node 用 data 属性构造 `{name, path}`，name 从行内 `a` 的 textContent 取）。`fileRowHtml` 里的 `<a href>` 保持 `fileDetailHref(n.path)` 不变。`init()` 调 `load()` 并做一次 `afterRender` 挂 `$watch('tree', …)`。组件不碰 `selectedPaths/scanDir/import/aiSubmitting`——那些留在页面。

- [ ] **Step 4: base.html 加 script 行**（位置：components.js 之后、`{{block}}` 之前）。

- [ ] **Step 5: 跑绿** 同 Step 2 命令 → PASS；`go test ./cmd/webui/...` 全绿（此时列表页还用旧逻辑，双份机制是本 task 的预期瞬态，Task 2 消灭）。

- [ ] **Step 6: Commit** `feat(webui): shared directoryTree component carries every tree mechanic`

### Task 2: 文件列表页改挂组件（行为零漂移）

**Files:**
- Modify: `cmd/webui/static/js/pages/file/tree.js`（删除已搬迁的机制，保留页面职责）
- Modify: `cmd/webui/templates/file.html`（`#file-tree` 区块换成组件元素）
- Modify: `cmd/webui/static_test.go`（扩展哨兵）

**Interfaces:**
- Consumes: Task 1 的完整 config 契约与组件方法。
- Produces: 页面 `fileTreePage` 对外行为不变；新增页面方法 `reloadTree()`（`this.selectedPaths = []; Alpine.$data(this.$refs.tree).load()`），scan/import/refresh/submitSelected 成功后统一调它——Task 3 的详情页也消费同名模式（`$refs.tree` 取组件实例）。

- [ ] **Step 1: 扩展哨兵（失败）** `TestDirectoryTreeComponent` 追加断言：`pages/file/tree.js` **不再**含 `treeHtml`、`fileRowHtml`、`expandDir`（机制单源闸门）；`templates/file.html` 含 `directoryTree(`。跑 `-run TestDirectoryTreeComponent` → FAIL。

- [ ] **Step 2: 改 `file.html`。** `#file-tree` 内改为：

```html
<div id="file-tree" x-ref="tree" x-data="directoryTree({
        picks: true,
        sort: () => ({field: sortField, order: sortOrder}),
        onTogglePick: (path, checked) => { if (checked) { if (selectedPaths.indexOf(path) < 0) selectedPaths.push(path); } else { selectedPaths = selectedPaths.filter(p => p !== path); } },
        getPicks: () => selectedPaths
     })">
    {{template "empty" (dict "Text" "加载中..." "Icon" "file" "Show" "loading")}}
    {{template "empty" (dict "TextExpr" "error" "Icon" "warning" "Show" "!loading && error")}}
    {{template "empty" (dict "Text" "暂无文件" "Icon" "file" "Hint" "请先扫描目录或导入文件" "Show" "!loading && !error && tree.length === 0")}}
    <div class="tree-root" x-show="!loading && !error" x-html="treeHtml()"></div>
</div>
```

  闭包里的 `sortField/sortOrder/selectedPaths` 解析到外层 `fileTreePage` 作用域，这是 Alpine 嵌套 x-data 的既定语义。批量按钮、sort 控件、card-header 一律不动。

- [ ] **Step 3: 瘦身 `pages/file/tree.js`。** 删掉已搬迁的九个方法与 `afterRender/bindPickEvents/syncPicks/togglePick/expandedDirs`；保留 `tree` 页面不再持有（组件状态）——`load()` 换成上面的 `reloadTree()`；init 里 `$watch(['tree','sortField','sortOrder'] → afterRender)` 整段删除（组件自管）。`selectedPaths/aiSubmitting/scanDir/scanning/importing/importPath/sortField/sortOrder/applySort/submitSelected/submitScan/submitImport/refreshTree/onFilePicked` 原样保留；`refreshTree()` 保持「POST /api/files/refresh 成功后 reloadTree()」。

- [ ] **Step 4: 哨兵转绿 + 全量** `go test ./cmd/webui/...` → PASS。

- [ ] **Step 5: 部署手测（列表页回归）** Docker 重建部署后逐项核对：①展开/折叠记忆与改前互通（键没变）；②未索引子项展开时懒加载追加、且不比上面的行左移；③勾选→批量识别按钮计数、提交后树 reload 且勾选清空；④排序切换全树逐级生效；⑤点文件名整页进入详情。任何一项漂移=本 task 未完成，回 Step 2/3 修。

- [ ] **Step 6: Commit** `refactor(webui): file list page runs on the shared directoryTree component`

### Task 3: 工作台左栏替换为树

**Files:**
- Modify: `cmd/webui/templates/file_detail.html`（`aside.dir-sidebar` 整块重写；舞台区不动）
- Modify: `cmd/webui/static/js/pages/file/detail.js`（目录列表遗留清理 + 树接线 + 指纹收缩）
- Modify: `cmd/webui/static/css/components/workbench.css`（侧栏树适配样式 + `.tree-current`）
- Modify: `cmd/webui/static_test.go`（新增 `TestWorkbenchSidebarIsDirectoryTree`）

**Interfaces:**
- Consumes: Task 1 config 契约（`onFileClick`、`getCurrent` 首次启用）；Task 2 的 `Alpine.$data($refs.tree)` 取实例模式与组件方法 `reveal/refreshFileRow/load`。
- Produces: 工作台侧栏 DOM 契约：`#wb-dir-tree`（组件根、`x-ref="tree"`）、`.tree-current`（当前文件行）。`detail.js` 新增页面方法 `revealSidebar()`、`refreshSidebarRow(f)`；删除 `onSidebarClick`；`applySort` 不再调 `load()`。

- [ ] **Step 1: 写失败哨兵** `static_test.go` 新增 `TestWorkbenchSidebarIsDirectoryTree`：断言 `templates/file_detail.html` 含 `directoryTree(` 且**不含** `dir-file-list`、`dir-indexed`、`dir-unindexed`；断言 `pages/file/detail.js` 的 `detailStamp` 函数体不再含 `dir_files`/`dir_unindexed`（对 `function detailStamp` 到 `return parts.join` 区间做 substring 断言）。`go test -run TestWorkbenchSidebarIsDirectoryTree` → FAIL。

- [ ] **Step 2: 重写 `file_detail.html` 侧栏。** `aside.panel.dir-sidebar.scroll` 内改为两块：

```html
<div class="dir-header flex-between">
    <h4>{{template "icon" (dict "Name" "folder" "Class" "folder-icon")}}
        <code x-text="detail ? detail.dir_path : ''"></code></h4>
    {{template "sortControlsField" (dict "IdRoot" "sort" "Prefix" "tagfile_detail")}}
</div>
<div class="dir-section">
    <div id="wb-dir-tree" x-ref="tree" x-data="directoryTree({
            sort: () => ({field: sortField, order: sortOrder}),
            getCurrent: () => filePath,
            onFileClick: (node) => { if (node.path !== filePath) navigateToFile(node.path); }
         })">
        {{template "empty" (dict "Text" "加载中..." "Icon" "file" "Show" "loading")}}
        {{template "empty" (dict "TextExpr" "error" "Icon" "warning" "Show" "!loading && error")}}
        <div class="tree-root" x-show="!loading && !error" x-html="treeHtml()"></div>
    </div>
</div>
```

  （`picks` 不传=关闭；不出现「未索引」独立小节——懒加载在树内。dir-header 保留目录面包屑是 spec 决策 4；排序控件搬进 header 右侧，IdRoot/Prefix 原样保留，即 `tagfile_detail_*` 存储键不变。）

- [ ] **Step 3: 改 `detail.js`。**
  - `detailStamp(d)`：删掉 `dir_files`/`dir_unindexed` 两段 forEach（侧栏不再渲染这些字段；其余字段一律不动）。
  - `init()`：删 `window.addEventListener('popstate'…)` 之外的 `this.$el.addEventListener('click', this.onSidebarClick.bind(this))`，并**保留** popstate 监听（浏览器前进后退仍要重走 `navigateToFile` 语义，`onPopState` 留在页面）。
  - 删除 `onSidebarClick` 整个方法；`applySort(field, order)` 只写 `this.sortField/sortOrder`，不再调 `this.load()`（排序现在纯客户端排树）。
  - `fetchDetail`/`silentRefresh` 的 opts 去掉 `sort`/`order` 两项（服务端 `sortDirFiles` 已无人消费）。
  - 新增两个页面方法并在两处挂载点调用：

```js
revealSidebar: function() {
  var el = this.$refs.tree; if (!el || !el.__x) return;
  var t = Alpine.$data(el);
  if (t.loading || !t.tree.length) return;
  t.reveal(this.filePath);
},
refreshSidebarRow: function(f) {
  var el = this.$refs.tree; if (!el || !el.__x) return;
  Alpine.$data(el).refreshFileRow({path: f.path, tag_cnt: f.tag_cnt, tag_done: f.tag_done, ai_status: f.ai_status});
},
```

  调用点：①`applyDetail(d)` 尾部 `var self = this; this.$nextTick(function(){ self.revealSidebar(); });`；②`mirrorFileField(f, key, val)` 尾部 `this.refreshSidebarRow(f)`。

- [ ] **Step 4: `workbench.css` 追加侧栏适配**（窄栏里 `.tree-name` 省略号、`.tree-dir/.tree-summary` 字号走 `--text-sm`、去掉 `.tree-size/.tree-pick` 在窄栏的多余留白——照文件列表页样式做减法；再钉高亮）：

```css
.dir-sidebar .tree-file.tree-current { background: var(--accent-bg); }
.dir-sidebar .tree-current .tree-name { color: var(--accent); font-weight: 600; }
```

- [ ] **Step 5: 哨兵转绿 + 全量测试** `go test ./cmd/webui/...` → PASS。

- [ ] **Step 6: 部署手测（工作台回归，spec 验收逐条）**：
  - 流水线跨目录连抽：树自动展开祖先、`.tree-current` 跟着走、视口滚到当前行。
  - 点树里**同目录/跨目录**文件：URL pushState、舞台换片、`recSeen/recScope` 不丢（顶部「本次已推荐」计数不变）、无整页刷新。
  - 打标 +1 / 删除标签 / 完成并下一个 / AI 识别：侧栏对应行的计数与「已完成/AI」徽章就地变化，不整树重取（Network 面板确认无额外 `GET /api/files`）。
  - 空闲页 SSE：一次 keepalive 后 DOM 无跳动（指纹已不含目录列表，若还有闪烁 = 指纹删多了，回看 Step 3）。
  - 浏览器前进/后退仍正确换文件。
  - 折叠开合在列表页与工作台间互通（同一份 sessionStorage 键）。

- [ ] **Step 7: Commit** `feat(webui): workbench sidebar becomes the full collapsible directory tree`

### Task 4: 5.21.0 记账 + 收口部署

**Files:**
- Modify: `docs/CHANGELOG_webui.md`（5.21.0 新条目置顶）
- Modify: `cmd/webui/constant/version.go`（`WebuiVersion = "5.21.0"`，同一 commit）

- [ ] **Step 1: 写 CHANGELOG 条目**，AGENTS.md 格式（一条目 = 英文行紧跟中文行，与 5.20.0 之间空一行，5.21.0 在文件最上）。内容覆盖：机制抽为 `core/tree.js` 共享组件（列表页零行为变化）、工作台左栏替换为完整折叠树、树内点击就地换文件不丢流水线、自动展开祖先并高亮、折叠记忆两页共用、打标/完成态乐观更新只改当前行、`.dir-*` 样式因 tag 页共用而保留。
- [ ] **Step 2: bump 常量** `5.20.0 → 5.21.0`；`go test ./cmd/webui/...` 全绿。
- [ ] **Step 3: Commit** `chore(webui): changelog 5.21.0 + version bump`
- [ ] **Step 4: 最终部署**：Docker 重建重跑（Global Constraints 里的命令），`curl localhost:8088/files/recommend` 含 `directoryTree(`，页面版本号渲染 `5.21.0`；请用户在浏览器实跑一轮流水线。

## Self-Review 记录

- spec 目标四条：组件（T1）、三参数面（T1 契约：onFileClick/picks/getCurrent+reveal，勾选与排序走闭包注入）、工作台自动展开+就地行更新（T3）、列表页零漂移（T2 Step 5）。spec 的「非目标」在计划里对应：无任何 API/服务端改动步骤。
- `_builder 裁量`四条全部落到具体步骤：排序键与控件（T3 Step 2）、全树只取一次（T3 Step 3 无 load 接线 + Step 6「不整树重取」验证）、乐观更新单行（T3 组件 `refreshFileRow` + `mirrorFileField` 调用点）、折叠记忆共享（T1 DOM 契约）。
- 已知瞬态：T1 结束时机制存在两份（哨兵只查新文件存在，T2 的单源闸门才查旧文件移除）——这是搬迁类重构的最小可测切片，接受。
