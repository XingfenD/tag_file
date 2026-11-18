# WebUI 响应式布局 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 WebUI 在手机（≤768px）与平板（≤1024px）宽度下可用：侧栏变汉堡抽屉、数据表堆叠成卡片、其余组件收紧。

**Architecture:** 纯表现层改动——媒体查询就近写在各 CSS 组件文件里（沿用 `config.css` 720px 规则的风格），`base.html` 加汉堡按钮 + 遮罩 + 一段内联脚本，表格单元补 `data-label` 供卡片化标签渲染。不改任何 Go handler、API 或路由。

**Tech Stack:** CSS `@layer`（现有结构不动）、Alpine.js（现有，不新增逻辑）、Go `html/template`（现有模板）、Go testing（模板标记不变量测试，仿 `templates_test.go` 的 `strings.Contains` 风格）。

**Spec:** 无独立 spec 文件——设计于 2026-11-18 聊天中确认（汉堡抽屉、表格卡片堆叠、断点 768/1024），关键决定已在下方 Global Constraints 与任务中逐条落定。

## Global Constraints

- 断点只有两个，均为 `@media (max-width: …)` 形式：`768px`（手机主断点）、`1024px`（平板轻收紧）。
- 不新增依赖、不新增 JS 组件文件；抽屉脚本以内联 `<script>` 写在 `base.html` 末尾。
- 保持 `@layer` 包裹（reset/base/layout/components），媒体查询写在对应层文件内。
- 不动 Go handler、路由、API；唯一 Go 侧改动是常量 `cmd/webui/constant/version.go` 的 `WebuiVersion`。
- 分支：从当前 `chore/webui-frontend-refactor` 切 `feat/responsive-layout`，全部提交落在新分支；提交前照例 `git branch --show-current` 确认不在 `master`。
- CHANGELOG：`docs/CHANGELOG_webui.md` 同一条目先英文行后中文行（两行紧邻无空行），高版本在上；版本 bump 与常量修改同一提交。本次版本号定为 **5.11.0**。
- 验证命令基线：`go test ./cmd/webui/...`（含模板解析/渲染冒烟）；`make build-webui` 必须通过。

## Review Focus

计划实现完成后，整分支评审时优先看这几类（现有测试不覆盖、但使用者会第一时间撞上）：

1. **触屏没有 hover** —— 行操作列 `.col-actions-inner` 原本只在 `:hover` 显示，手机上永远点不到；期望非 hover 设备常显。
2. **卡片化后的横向溢出** —— 文件路径、标签长串在 `td` 里会把卡片撑破屏幕；期望值区可换行或截断（`min-width:0` + `overflow-wrap`/ellipsis）。
3. **抽屉打开后的关闭路径** —— 期望点遮罩、按 Esc、点任意导航链接都能收起抽屉；漏一个就是「菜单关不掉」。
4. **详情页 100vh 硬高度** —— `.detail-layout` 的 `calc(100vh - 154px)` 在手机浏览器地址栏伸缩下不可靠；堆叠后应改回自然流式高度（`body` 用 `100dvh` 兜底）。
5. **键值信息表不该被卡片化** —— `file_detail.html` 基本信息表自带行头 `<th>` 且无 `.table-wrap` 包裹；期望堆叠 CSS 只命中 `.table-wrap` 内的表，键值表保持原样。

## File Structure

| 文件 | 职责 | 任务 |
|---|---|---|
| `cmd/webui/templates/base.html` | 汉堡按钮、遮罩 div、内联抽屉脚本 | 1 |
| `cmd/webui/static/css/layout/layout.css` | 侧栏抽屉化、`.main`/`page-title` 收紧、`body` dvh | 1 |
| `cmd/webui/responsive_test.go`（新） | `data-label` 覆盖不变量测试 | 2 |
| `cmd/webui/static/css/components/table.css` | ≤768px 卡片堆叠规则、hover 可见性修复 | 2 |
| `cmd/webui/templates/{category,tag,file_groups,file_detail}.html` | `<td>` 补 `data-label` | 2 |
| `cmd/webui/static/js/pages/review/main.js` | 两张 JS 拼表补 `data-label` | 2 |
| `cmd/webui/static/css/{components/forms.css,components/modal.css,components/notifications.css,components/query-builder.css}` | 窄屏收紧 | 3 |
| `cmd/webui/constant/version.go` + `docs/CHANGELOG_webui.md` | 5.11.0 | 4 |

---

### Task 1: 汉堡抽屉侧栏与壳层收紧

**Files:**
- Modify: `cmd/webui/templates/base.html`
- Modify: `cmd/webui/static/css/layout/layout.css`

**Interfaces:**
- Produces: DOM 契约 `<button id="sidebar-toggle">`、`<div class="sidebar-overlay">`、`body.sidebar-open` 类名；Task 3 的媒体查询沿用同一断点写法。

- [ ] **Step 1: 建分支**

```bash
git checkout -b feat/responsive-layout
```

- [ ] **Step 2: base.html 加按钮、遮罩、脚本**

在 `<body>` 内、`<aside class="sidebar">` 之前加汉堡按钮（桌面隐藏，仅靠 CSS 控制显隐），`</aside>` 之后加 `<div class="sidebar-overlay"></div>`；`</body>` 前加内联脚本。脚本行为（约 15 行原生 JS）：

- 点击按钮 → `document.body.classList.toggle('sidebar-open')`；
- 点击遮罩 → 移除 `sidebar-open`；
- `Escape` 键 → 移除 `sidebar-open`；
- `.sidebar nav a` 任意点击 → 移除 `sidebar-open`。

按钮文案用 `☰`，带 `aria-label="菜单"`。

- [ ] **Step 3: layout.css 写抽屉媒体查询**

在 `@layer layout` 内追加 `@media (max-width: 768px)` 块：`.sidebar` 变 `position: fixed; inset: 0 auto 0 -240px; height: 100dvh; z-index: 100; width: 230px`，`body.sidebar-open .sidebar { transform: translateX(240px); }` 配 `transition: transform var(--transition)`；`.sidebar-overlay` 半透明遮罩默认 `display:none`，`body.sidebar-open` 时全屏显示（`z-index: 99`）；`#sidebar-toggle` 默认 `display:none`、768px 下 `display:flex`（绝对定位左上角即可，z-index 高于内容）；`.main` padding 收至 `16px`，`body { height: 100dvh }`。同文件再加一个 `@media (max-width: 1024px)`：`.main` padding `20px 24px`、`.dir-sidebar` width `200px`。

- [ ] **Step 4: 跑测试验证未破坏**

Run: `go test ./cmd/webui/... -v`
Expected: 全部 PASS（模板解析/渲染冒烟不受影响）。

- [ ] **Step 5: 手动冒烟（可选但有浏览器就做）**

`make build-webui && ./tagfile-webui.out --source temp/test_workspace --addr :8099`，浏览器把窗口收到 <768px：☰ 出现、抽屉可开可关（遮罩/Esc/导航链接三路）、桌面宽度下侧栏与按钮恢复原样。

- [ ] **Step 6: Commit**

```bash
git add cmd/webui/templates/base.html cmd/webui/static/css/layout/layout.css
git commit -m "feat(webui): hamburger drawer sidebar and shell tightening for <=768px"
```

---

### Task 2: 表格卡片堆叠 + 触屏可见性（测试先行）

**Files:**
- Create: `cmd/webui/responsive_test.go`
- Modify: `cmd/webui/static/css/components/table.css`
- Modify: `cmd/webui/templates/category.html`、`tag.html`、`file_groups.html`、`file_detail.html`
- Modify: `cmd/webui/static/js/pages/review/main.js`

**Interfaces:**
- Consumes: 表格已在 `.table-wrap` 内（`table.css:7`）；`file_detail.html` 键值表无 `.table-wrap`，因此天然不被堆叠命中（Review Focus 5）。
- Produces: 所有会被堆叠的 `<td>` 均携带 `data-label`（卡片内标签由 CSS `::before` 读取）；不变量由 `TestStackedTableDataLabels` 长期守住。

- [ ] **Step 1: 写失败测试**

新建 `cmd/webui/responsive_test.go`（`package main`，风格仿 `templates_test.go`）。`TestStackedTableDataLabels`：对文件清单 `templates/category.html`、`templates/tag.html`、`templates/file_groups.html`、`templates/file_detail.html`（经 `templateFS` 读，JS 文件 `static/js/pages/review/main.js` 经嵌入 FS 或 `os.ReadFile`——选测试里现成可行的方式）逐个正则匹配所有 `<td` 起始标签，断言其开标签内必含 `data-label`；命中即记录文件名+片段方便定位。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./cmd/webui/ -run TestStackedTableDataLabels -v`
Expected: FAIL，报出各文件缺 `data-label` 的 `<td>`。

- [ ] **Step 3: 模板补 data-label**

值照各表 `<th>` 文案逐字取：

- `category.html`：`ID` / `名称` / `描述` / `操作`
- `tag.html`：`ID` / `标签名` / `所属分类` / `父标签` / `预设值` / `描述` / `操作`
- `file_groups.html`：`组名` / `标签数` / `类型` / `操作`
- `file_detail.html` 标签表：`标签名` / `值` / `分类` / `操作`；键值表各行 `<td>` 以其行头 `<th>` 文案为 `data-label`（`ID`/`路径`/`大小`/`类型`/`标签 (n)` 用固定串 `标签`）——它不被堆叠命中，补上只为让整文件不变量成立。

- [ ] **Step 4: review/main.js 补 data-label**

`taskRowsHtml` 七格照 `<thead>` 取 `任务/文件/状态/帧数/待审/更新时间/操作`；`sugRowsHtml` 首格复选框用 `data-label=""`，其余 `文件/分类/标签/值/识别匹配/操作`。只改字符串模板里的开标签，不动事件委托。

- [ ] **Step 5: 跑测试确认通过**

Run: `go test ./cmd/webui/... -v`
Expected: `TestStackedTableDataLabels` 及既有测试全部 PASS。

- [ ] **Step 6: table.css 写堆叠与 hover 修复**

`@layer components` 内追加 `@media (max-width: 768px)`：作用域限定 `.table-wrap table`——`thead` 隐藏；`tr` 变块级卡片（复用玻璃配方声明：`var(--glass-bg-strong)`、`border`、`radius`、`margin-bottom`）；`td` 变 `display:flex; justify-content:space-between; gap:12px; border-bottom:none`，配 `td::before { content: attr(data-label); font-weight:500; color: var(--ink-muted); flex-shrink:0 }` 与值区 `min-width:0; overflow-wrap:anywhere`（Review Focus 2）。同文件把现有 `tbody tr:hover .col-actions-inner { opacity: 1 }` 的反面——基础态 `.col-actions-inner{opacity:0}`——改为仅在 `@media (hover: hover)` 下生效（Review Focus 1），并把 `tbody tr:hover` 高亮一并移入该媒体查询。

- [ ] **Step 7: 构建验证**

Run: `make build-webui && go test ./cmd/webui/... `
Expected: 构建成功、测试全 PASS。浏览器（或 DevTools 窄屏）看分类/标签/文件详情/待审页：表格成卡片、标签在左值在右、操作按钮常显可点。

- [ ] **Step 8: Commit**

```bash
git add cmd/webui/responsive_test.go cmd/webui/static/css/components/table.css cmd/webui/templates/ cmd/webui/static/js/pages/review/main.js
git commit -m "feat(webui): stack tables into cards on <=768px with data-label invariant test"
```

---

### Task 3: 详情页与通用组件窄屏收紧

**Files:**
- Modify: `cmd/webui/static/css/layout/layout.css`（detail 段）
- Modify: `cmd/webui/static/css/components/forms.css`、`modal.css`、`notifications.css`、`query-builder.css`、`card.css`

**Interfaces:**
- Consumes: Task 1 的断点约定与 `--transition`/`--glass-*` 变量。

- [ ] **Step 1: 详情页堆叠**

`layout.css` 的 768px 块内：`.detail-layout { flex-direction: column; height: auto; }`（脱离 100vh 硬高度，Review Focus 4），`.dir-sidebar { width: auto; max-height: 30vh; }`；`.main:has(.detail-layout) { overflow: auto; }` 覆盖桌面段的 `overflow:hidden`，让详情页恢复自然滚动。

- [ ] **Step 2: 组件收紧**

各文件内追加 768px 媒体查询：

- `forms.css`：`.input-w-sm/.input-w-md/.input-w-lg/.input-w-xl { max-width: 100%; }`
- `modal.css`：`.modal { padding: 20px 18px; }`，`.modal-form`/`.modal-lg` `max-width: 94vw`
- `notifications.css`：`.toast { max-width: calc(100vw - 32px); }`，`.toast-container { right: 16px; }`
- `query-builder.css`：`.qb-type/.qb-op { width: auto; flex: 1 1 90px; }`，六个 `.qb-val-*` 固定宽度全部 `flex: 1 1 100%`（独占一行）
- `card.css`：`.media-preview { max-height: 40vh; }`，`.card { padding: 16px; }`

- [ ] **Step 3: 验证**

Run: `go test ./cmd/webui/... && make build-webui`
Expected: 全 PASS、构建成功。窄屏过一遍：搜索页查询构造器逐行堆叠不溢出、文件详情页目录列表落顶部、模态/toast 不出屏。

- [ ] **Step 4: Commit**

```bash
git add cmd/webui/static/css/
git commit -m "feat(webui): stack detail layout and tighten forms/modals/toasts/query-builder on phones"
```

---

### Task 4: 版本号与 CHANGELOG

**Files:**
- Modify: `cmd/webui/constant/version.go`（`WebuiVersion` `"5.10.0"` → `"5.11.0"`）
- Modify: `docs/CHANGELOG_webui.md`

**Interfaces:**
- Consumes: Task 1–3 的全部改动描述。

- [ ] **Step 1: 写 CHANGELOG 条目并 bump 常量**

`docs/CHANGELOG_webui.md` 顶部加 `## [5.11.0] - 2026-11-18`，`### Added / 新增` 段一条：英文行、紧随中文行（之间不空行），内容概括——汉堡抽屉侧栏、≤768px 表格卡片堆叠、触屏操作列常显、详情页与表单/模态/toast/查询构造器窄屏适配。常量同值改 `5.11.0`。

- [ ] **Step 2: 验证 + 提交**

Run: `go test ./cmd/webui/...`
Expected: PASS。

```bash
git add cmd/webui/constant/version.go docs/CHANGELOG_webui.md
git commit -m "chore(webui): bump WebuiVersion to 5.11.0 with changelog entry"
```
