# WebUI 设计系统重构（Wave 12–16）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 cmd/webui 前端剩余的重复代码抽成共享组件/helper，并把散落的默认样式收敛为 token 化的设计系统（含 4 个暗色主题真 bug 修复）。

**Architecture:** 五条独立可交付的波次：主题修复 → 模板 partial → JS 加载侧组件 → CSS 组件化/死代码 → 标量 token。JS 共享 helper 全部落在现有 core/utils.js 与 core/api.js（不新增 JS 文件，避免 fe-check 加载契约变更）；新 partial 为顶层 `{{define}}` 独立文件并在 main.go 注册；新 CSS 组件文件在 base.html 挂链接。每波结束记 CHANGELOG_webui.md 并同步 WebuiVersion。

**Tech Stack:** Go 1.25 + Gin + html/template（dict FuncMap）、Alpine.js 3（无构建步骤，全局脚本）、原生 CSS（@layer + CSS 变量）、node:test（Node 24，新增 JS 单测）、browser-use MCP（浏览器验收）。

**Spec:** 无独立 spec 文档；依据为本文件「审计依据」一节（2026-11-18 会话审计，关键断言已逐条 grep/读码核实）。

## Global Constraints

- 分支：当前 `chore/webui-frontend-refactor`。每次 commit 前 `git branch --show-current`；若在 master 停下问用户。
- CHANGELOG：WebUI 改动记 `docs/CHANGELOG_webui.md`，高版本在上；同一条目英文行紧跟中文行（中间无空行），条目间空行。bump CHANGELOG 的同一 commit 更新 `cmd/webui/constant/constant.go` 的 `WebuiVersion`（当前 `5.5.0`）。
- 版本号：Wave 12→`5.6.0`，13→`5.7.0`，14→`5.8.0`，15→`5.9.0`，16→`5.10.0`。（历史已用至 wave 11 / 5.5.0，本计划从 12 续编号。）
- 模板：`{{define}}` 不能嵌套；新 partial 必须顶层独立文件 + 注册进 `cmd/webui/main.go:58` 的 partials 切片。
- JS 加载契约：core 顺序 api.js → utils.js → stores.js → components.js（base.html:124-127）；utils.js 的全局函数运行时才引用 API/showToast。不新增 static/js 文件（fe-check 第 3 门要求每个 JS 恰好被一个模板加载）。
- 门禁：每个任务 commit 前跑 `make fe-check` 与 `go test ./cmd/webui/...`；每波最后一个任务跑全量 `make test`。
- DELETE 请求参数永远在 URL query（Go 只解析 POST/PUT/PATCH body）；`API.del` 无 body。
- 重复键参数（如 AI 入队 paths）必须用 `URLSearchParams.append`，对象字面量会塌成逗号串。
- 注释风格跟随仓库：只在契约/不变量非显然处写中文契约注释，不写流水账。
- 明确不做（本次范围外）：间距阶梯 token 化（设计决策，另行立项）、sidebar 白色系 token 化（两主题皆深底，纯装饰）、tag 树/file 树渲染器合并、search tagSearchHtml 与 Go partial 合并（跨系统双源已有契约注释）、`isAiMedia` 与 `iconForName` 合并、card_header partial（Go 模板无法包裹调用方内容，用 .toolbar 类替代）。

## Review Focus

1. **删掉末页唯一一条后停在空白页** —— 用户期望自动退到最后一页。T6 的 node 测试钉 clamp，浏览器验收在分类页复现删除场景。
2. **inlineEdit 保存失败必须回滚显示值** —— onSave 失败不 throw 就会把脏值留在单元格上。T5 node 测试钉 throw 契约，浏览器用重名分类触发服务端拒绝验证回滚。
3. **review 轮询（silent）不得闪 loading/弹 toast** —— 每 3 秒闪一次空态即回归。T6 node 测试钉 silent 分支。
4. **新 partial 里 Alpine 表达式被 html/template 转义破坏** —— 箭头函数/引号进属性后必须原样可用。T3 Go 测试用真实表达式断言渲染产物，浏览器冒烟行内编辑。
5. **暗色主题 accent 表面文字对比度** —— 亮青底白字约 1.9:1。T1 浏览器双主题 computed-style 断言。

## 审计依据（condensed）

已完成的波次（至 wave 11 / 5.5.0）不要重做：pager/searchForm/tagSearchField/sortControlsField partial、reportResult、modal required gate、checkbox 单源、AI_STATUS_META 单源。剩余问题（均核实）：

- **JS**：删除确认 6 处同骨架（catch 策略三种并存）；分页加载骨架 5 页各一份（仅 review 有越界 clamp，review/main.js:93-99）；AI 入队失败分支 3 处逐字（tree.js:231/detail.js:297/review 走 reportResult）；事件委托 4 处；inlineEdit 的 reportResult-or-throw 3 处（detail.js:373 手写旁路）；detail 详情抓取 URL 拼装 2 份且 reload 不带 sort（bug）；decodePath 4 处、fileDetailHref 4 处、readStoredJSON 4 处、sort 键镜像 3 处；selection 2 页逐字；config keptRows 2 处（校验/提交漂移风险）、findRule 6 个扫描器。
- **模板**：`.empty` 手写 19 处 vs partial 仅 2 处调用；inlineEdit 单元格 4 份逐字；死内联 `margin:0`×5、冗余 flex 属性×3、`reveal-delay-1..4` 无定义；search.html:17 裸 `btn-ghost` 缺 `.btn`。
- **CSS 真 bug**：forms.css:37 select 箭头 `fill='%237e918d'` 写死浅色；buttons.css:23/40/89 + navigation.css:19/95 accent 上 `color:#fff`；buttons.css:42/47 danger 光晕写死浅色 RGB；notifications.css:14-16 flash 边框写死三语义色。
- **CSS 收敛**：qb-* 零 CSS 定义（全内联在 search/main.js:175-252）；玻璃面板配方手抄 6 份；`.config-invalid-badge`≈`badge badge-red`；`tree-status-ai-*` 与 `badge-ai-*` 配色双份；`.tag`/`.tag-chip` 99% 同；`.dir-sidebar`/`-static` 仅差 overflow；`.scan-result` 死代码；fadeUpIn 定义在 layout.css 被跨文件引用；字号/blur/光晕/圆角无 token；输入框内联宽度 12 种随手值。

## File Structure

| 文件 | 动作 | 职责 |
|---|---|---|
| `cmd/webui/static/css/core/variables.css` | 改 | 全部新 token 唯一定义点（两主题） |
| `cmd/webui/static/js/core/utils.js` | 改 | 新共享 helper：confirmDelete/reportResultOrThrow/loadPaged/pagedList/clampToLastPage/delegate/createSelection/decodePath/pathFromLocation/fileDetailHref/readStoredJSON/readSortState |
| `cmd/webui/static/js/core/api.js` | 改 | API.enqueueAi / API.fileDetail |
| `cmd/webui/templates/partials/empty.html` | 改 | dict 方言空态 |
| `cmd/webui/templates/partials/inline_edit.html` | 新 | inlineEditCell |
| `cmd/webui/static/css/components/card.css` | 新 | .panel 基类 + card 家族（自 layout.css 迁出） |
| `cmd/webui/static/css/components/toolbar.css` | 新 | .toolbar |
| `cmd/webui/static/css/components/modal.css` | 新 | modal 段（自 notifications.css 迁出） |
| `cmd/webui/static/css/components/query-builder.css` | 新 | qb-*（自 search/main.js 内联迁出） |
| `cmd/webui/static_test.go` | 新 | CSS 回归门（package main，读 staticFS） |
| `cmd/webui/templates_test.go` | 新 | partial 渲染测试（package main，读 templateFS） |
| `tests/js/helpers.mjs` + `tests/js/*.test.mjs` | 新 | node:test 单测（vm 沙箱加载 core 脚本） |
| `Makefile` | 改 | fe-test 目标 |

---

## Wave 12 — 主题修复（→ 5.6.0）

### Task 1: 四个暗色主题 bug + CSS 回归门

**Files:**
- Create: `cmd/webui/static_test.go`
- Modify: `cmd/webui/static/css/core/variables.css`、`components/buttons.css:23,40,42,47,89`、`components/navigation.css:19,95`、`components/notifications.css:14-16`、`components/forms.css:30-45`（select 规则）
- Modify: `docs/CHANGELOG_webui.md`、`cmd/webui/constant/constant.go`

**Interfaces:**
- Produces: token `--on-accent`（light `#fff`，dark `#0c1a1e`），后续任务凡 accent 底上的文字一律用它。

- [ ] **Step 1: 写失败的 CSS 回归测试**

`cmd/webui/static_test.go`（package main，用现有 `staticFS embed.FS`）：

```go
func TestThemeTokens(t *testing.T) {
	read := func(p string) string {
		b, err := staticFS.ReadFile("static/" + p)
		if err != nil { t.Fatal(err) }
		return string(b)
	}
	cases := []struct{ file, absent string }{
		{"css/components/buttons.css", "color: #fff"},
		{"css/components/navigation.css", "color: #fff"},
		{"css/components/buttons.css", "rgba(224, 82, 82"},
		{"css/components/forms.css", "%237e918d"},
		{"css/components/notifications.css", "rgba(34,197,94"},
		{"css/components/notifications.css", "rgba(224,82,82"},
		{"css/components/notifications.css", "rgba(245,158,11"},
	}
	for _, c := range cases {
		if strings.Contains(read(c.file), c.absent) {
			t.Errorf("%s still contains hardcoded %q", c.file, c.absent)
		}
	}
	if !strings.Contains(read("css/core/variables.css"), "--on-accent") {
		t.Error("variables.css missing --on-accent")
	}
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./cmd/webui/ -run TestThemeTokens -v`，预期 FAIL（8 处硬编码均在）。
- [ ] **Step 3: 修复**
  - variables.css：`:root` 加 `--on-accent: #fff;`，`[data-theme="dark"]` 加 `--on-accent: #0c1a1e;`。
  - 5 处 `color: #fff` → `color: var(--on-accent)`。
  - buttons.css:42/47 光晕 → `color-mix(in srgb, var(--danger) 20%, transparent)` / `25%`（不新增 token，自动随主题）。
  - notifications.css:14-16 边框 → `color-mix(in srgb, var(--success) 15%, transparent)` 等三处。
  - forms.css select 箭头：data-URI 从 `background-image` 改为 `mask-image`（URI 中 fill 改黑色即可，mask 只取 alpha），配 `background-color: var(--ink-muted)`，保留原 size/position/repeat；`-webkit-mask-image` 同步。
- [ ] **Step 4: 跑测试确认通过** — 同 Step 2 命令，预期 PASS；`make fe-check` OK。
- [ ] **Step 5: 浏览器双主题验收** — `make setup` 后 `go run ./cmd/webui --source temp/test_workspace --addr :8088`（后台）。browser-use 打开 `/categories`，`evaluate_script` 断言：light 下 `.btn-primary` computed `color === 'rgb(255, 255, 255)'`；切 `document.documentElement.dataset.theme='dark'` 后 `color === 'rgb(12, 26, 30)'` 且 select 的 `background-color` 等于 `getComputedStyle(document.documentElement).getPropertyValue('--ink-muted')` 对应值。截图两主题留档。
- [ ] **Step 6: CHANGELOG 5.6.0 + WebuiVersion 同 commit** — 条目说明四修复与对比度数字；`make test` 全量通过。
- [ ] **Step 7: Commit** — `chore(webui): wave 12 — theme-aware on-accent/danger-glow/flash-border/select-arrow tokens fix four dark-mode bugs`

---

## Wave 13 — 模板 partial（→ 5.7.0）

### Task 2: empty partial dict 方言 + 收编 21 处

**Files:**
- Create: `cmd/webui/templates_test.go`
- Modify: `cmd/webui/templates/partials/empty.html`、`category.html:25-29`、`file.html:45-57`、`file_detail.html:64-77,170-174,203-206`、`file_groups.html:24-35`、`search.html:29-38`、`tag.html:53-68`、`review.html:45,72`（两处旧字符串调用改 dict）

**Interfaces:**
- Produces: partial `empty`，方言 `{{template "empty" (dict "Text" "暂无分类" "Icon" "🗃" "Hint" "在上方添加第一个分类" "Show" "!loading && items.length === 0" "TextExpr" "error")}}`。参数：`Icon` 缺省 `&#9888;`；`Text` 静态文案；`TextExpr` 渲染为 `.empty-text` 上的 `x-text`（与 Text 互斥，TextExpr 优先）；`Hint` 渲染 `.empty-hint`；`Show` 渲染根节点 `x-show`。

- [ ] **Step 1: 写失败的渲染测试** — `cmd/webui/templates_test.go`（package main）：用 `template.New("t").Funcs(handlers.FuncMap).ParseFS(templateFS, "templates/partials/empty.html")` 后 `ExecuteTemplate(&buf, "empty", map[string]any{...})`。三个用例断言：
  1. `{"Text":"暂无分类","Icon":"🗃","Hint":"在上方添加第一个分类","Show":"!loading && items.length === 0"}` → 输出含 `x-show="!loading &amp;&amp; items.length === 0"`（html/template 属性转义）、`🗃`、`empty-hint`。
  2. `{"TextExpr":"error","Show":"error"}` → 含 `x-text="error"`。
  3. `{"Text":"空"}` → 含 `&#9888;`（缺省图标），不含 `x-show`/`empty-hint`。
- [ ] **Step 2: 跑测试确认失败** — `go test ./cmd/webui/ -run TestEmptyPartial -v`（现 partial 只认字符串，dict 用例报错/缺属性）。
- [ ] **Step 3: 重写 partial 并迁移全部站点** — 按 Interfaces 的方言实现；19 处手写块逐处替换（保留原图标字符、文案、x-show 表达式；error 态用 TextExpr）；review 两处 `{{template "empty" "文案"}}` 改 `(dict "Text" "文案")`。
- [ ] **Step 4: 验证** — 测试 PASS；`make fe-check`（第 4 门：empty 仍被调用）；浏览器逐页空态冒烟（分类空列表、搜索无结果、file_detail 错误态）。
- [ ] **Step 5: Commit** — `chore(webui): wave 13a — empty partial speaks dict dialect, absorbs 19 hand-written empty states`

### Task 3: inlineEditCell partial + 4 处收编

**Files:**
- Create: `cmd/webui/templates/partials/inline_edit.html`
- Modify: `cmd/webui/main.go:58-66`（partials 切片加一行）、`category.html:39-44,47-52`、`file_groups.html:42-47`、`file_detail.html:183-188`、`cmd/webui/static/js/core/components.js:98-105`（用法注释改为指向 partial）
- Test: `cmd/webui/templates_test.go`

**Interfaces:**
- Produces: partial `inlineEditCell`，参数 `Value`/`OnSave` 为 Alpine 表达式字符串：`{{template "inlineEditCell" (dict "Value" "item.name" "OnSave" "(v) => updateCategory(item, { name: v })")}}`。渲染产物与现 6 行结构逐字等价（span x-data + @click.outside + input x-ref/readonly/@click/@blur/@keydown）。

- [ ] **Step 1: 写失败测试** — 同文件加 `TestInlineEditCellPartial`：ParseFS 新 partial，传上面的真实 dict，断言输出含 `inlineEdit({ value: item.name, onSave: (v) =&gt; updateCategory(item, { name: v }) })`（**注意**：html/template 对属性内 `>` 的转义形式以实际输出为准——先跑一次看产物，断言浏览器可解析的形式；若转义破坏表达式，改用 `template.HTMLAttr` 包装参数并在 partial 注释钉住原因）。另断言含 `@blur="editing &amp;&amp; save()"` 与 `x-ref="input"`。
- [ ] **Step 2: 确认失败**（partial 不存在，ParseFS 报错）。
- [ ] **Step 3: 实现 partial + 注册 + 迁移 4 处 + 更新 components.js 注释。**
- [ ] **Step 4: 验证** — 测试 PASS、fe-check OK；浏览器在分类页行内改名成功一次、改成已有分类名失败一次（值回滚、toast 报错）——这是 Review Focus 2/4 的验收。
- [ ] **Step 5: Commit** — `chore(webui): wave 13b — inlineEditCell partial dedupes four verbatim inline-edit cells`

### Task 4: 模板死内联清理 + 小类收编（波尾）

**Files:**
- Modify: `category.html:21`、`file.html:34`、`review.html:27,52`、`file_detail.html:83`（删 `style="margin:0"`，layout.css:186 已有）；`tag.html:4`（删冗余 align-items:stretch）；`file_detail.html:48,82`（删 .flex-between 上冗余属性）；`home.html:5,9,13,17`（删无定义的 reveal-delay-*）；`search.html:17`（`btn-ghost` → `btn btn-ghost`）；`file_detail.html:138`（内联 font-size:11px → `btn-xs`）；`review.html:14-19`（两个 checkbox label 内联 → 新 `.checkbox-label`）；`review.html:44,71`、`tag.html:15`（加载中内联 → 新 `.loading-hint`）
- Modify: `cmd/webui/static/css/components/forms.css`（加 `.checkbox-label { display:flex; align-items:center; gap:5px; font-size:13px; cursor:pointer }`）、`core/base.css`（加 `.loading-hint { color:var(--ink-muted); font-size:12px; padding:8px 0 }`，用 `.text-muted` 组合亦可——实现时看 base.css 现有 utility，取最小新增）
- Modify: `docs/CHANGELOG_webui.md`、`cmd/webui/constant/constant.go`（5.7.0）

- [ ] **Step 1: 逐处替换/删除**（上表即清单，无遗漏项）。
- [ ] **Step 2: 验证** — `make fe-check`、`go test ./cmd/webui/...`；浏览器看 review 复选框行、search 清除按钮（补 .btn 后应有 padding）、home 卡片动画仍正常。
- [ ] **Step 3: 全量 make test + CHANGELOG 5.7.0（EN+ZH）+ WebuiVersion。**
- [ ] **Step 4: Commit** — `chore(webui): wave 13 wrap-up — dead inline styles purged, checkbox-label/loading-hint classes + changelog 5.7.0`

---

## Wave 14 — JS 共享组件（→ 5.8.0）

### Task 5: node 测试基建 + confirmDelete + reportResultOrThrow

**Files:**
- Create: `tests/js/helpers.mjs`、`tests/js/utils.test.mjs`
- Modify: `Makefile`（`fe-test: node --test tests/js/`，加 .PHONY）、`cmd/webui/static/js/core/utils.js`
- Modify: 6 个删除站点 `pages/tag/main.js:396-403,495-503`、`pages/category/main.js:72-79`、`pages/file/detail.js:353-366,398-409`、`pages/file/groups.js:68-77`；3 个 throw 站点 `pages/file/detail.js:368-377`（saveFileTagValue 手写旁路）、`pages/file/groups.js:64`、`pages/category/main.js:68`

**Interfaces:**
- Produces（utils.js 新全局函数，后续任务同文件追加）:
  - `confirmDelete(msg, url, opts)` — opts `{okMsg='', failMsg='', okType='info', onOk}`；内部 `showConfirm(msg, () => API.del(url).then(d => { if (reportResult(d, okMsg, failMsg, okType) && onOk) onOk(); }).catch(API.onError))`。
  - `reportResultOrThrow(d, okFallback, failFallback, okType)` — `if (!reportResult(...)) throw new Error((d && d.msg) || failFallback);`，供 inlineEdit onSave。
  - `tests/js/helpers.mjs` 导出 `loadCore(files, stubs)`：vm 沙箱按序执行 `cmd/webui/static/js/core/` 下指定文件，返回沙箱（顶层 function/var 声明成为沙箱属性）；stubs 缺省提供 `showToast`/`showConfirm`/`console`/`URLSearchParams`/`setTimeout`。

- [ ] **Step 1: 写失败测试** — `tests/js/utils.test.mjs`：

```js
import test from 'node:test';
import assert from 'node:assert/strict';
import { loadCore } from './helpers.mjs';

test('confirmDelete keeps params in URL query and refreshes only on ok', async () => {
  const calls = [];
  const s = loadCore(['utils.js'], {
    showConfirm: (msg, onOk) => onOk(),
    showToast: () => {},
    API: { del: async (url) => { calls.push(url); return { ok: true, msg: '已删' }; }, onError: () => {} },
  });
  let refreshed = 0;
  s.confirmDelete('确定删除？', '/api/files/group/3?file_path=a%2Fb', { onOk: () => refreshed++ });
  await new Promise(r => setTimeout(r, 0));
  assert.equal(calls[0], '/api/files/group/3?file_path=a%2Fb');
  assert.equal(refreshed, 1);
});

test('reportResultOrThrow throws on failure so inlineEdit rolls back', () => {
  const s = loadCore(['utils.js'], { showToast: () => {} });
  assert.throws(() => s.reportResultOrThrow({ ok: false, msg: '重名' }, '', '更新失败'), /重名/);
  assert.doesNotThrow(() => s.reportResultOrThrow({ ok: true }, '已更新', ''));
});
```

  另加：失败信封不触发 onOk；`API.del` reject 走 onError 不 throw 出 confirmDelete。
- [ ] **Step 2: 确认失败** — `node --test tests/js/`，预期 FAIL（函数不存在）。
- [ ] **Step 3: 实现两函数 + Makefile fe-test + 迁移 9 个站点** — 各站点保留原 msg/URL 构造与 okMsg/failMsg 字面量；**统一 okType='info'**（detail removeGroup 原为 success，属刻意归一，changelog 注明）；tag deleteTag 的 catch 从 self.refresh() 归一为 API.onError（失败不再刷新），removeTagValue 补上 catch。saveFileTagValue 的手写两分支改 `reportResultOrThrow(data, '值已更新', '更新失败')`。
- [ ] **Step 4: 验证** — `make fe-test` PASS、`make fe-check` OK；浏览器删一个分类（确认弹窗→toast→列表刷新）、行内改值失败回滚。
- [ ] **Step 5: Commit** — `chore(webui): wave 14a — confirmDelete/reportResultOrThrow + node:test harness, nine sites unified`

### Task 6: 分页加载骨架 loadPaged/pagedList/clampToLastPage

**Files:**
- Modify: `cmd/webui/static/js/core/utils.js`
- Modify: `pages/review/main.js:84-109`（loadList 体）、`pages/category/main.js`（整页瘦身）、`pages/search/main.js:306-330`（doSearch）、`pages/tag/main.js:120-123`（flat 分支加 clamp）、`pages/file/detail.js:173-213`（load/reload 加 clamp）
- Test: `tests/js/paged.test.mjs`

**Interfaces:**
- Produces（utils.js）:
  - `clampToLastPage(pagination, page)` → 越界时返回 `pagination.total_pages`，否则 `null`（判据与 review/main.js:94 逐字：`page > 1 && total_pages > 0 && page > total_pages`）。
  - `async loadPaged(self, spec, silent)` — review loadList 的泛化，行为逐字等价：spec `{url?: string, fetch?: async fn(self), pageKey='page', itemsKey='items', pagKey='pagination', loadingKey='loading', errMsg='加载失败', warnMsg='[paged] 静默加载失败，下一次继续:', after?: fn(self), clamp=true}`。url 字符串时 fetch 缺省为 `self => API.get(spec.url + '?page=' + self[pageKey])`。silent 不翻 loading、不弹 toast、catch 走 console.warn；!ok 非 silent 弹 `d.msg || errMsg`；clamp 命中改 `self[pageKey]` 后递归重取；成功写 items/pagination 再调 after。
  - `pagedList(cfg)` — cfg `{url: fn(self), errMsg, after}`；返回标准命名状态与方法 `{q:'', queryInput:'', page:1, items:[], pagination:{}, loading:true, refresh(silent), doSearch(), clearSearch(), goToPage(page)}`，refresh 即 `loadPaged(this, spec, silent)`，搜索三件套与 category/main.js:37-53 逐字同义。页面用 `Object.assign(pagedList({...}), {页面私有})` 组合。
- Consumes: T5 的 loadCore（测试加载 utils.js，stub API.get/post）。

- [ ] **Step 1: 写失败测试** — `tests/js/paged.test.mjs` 用例：
  1. clamp：`clampToLastPage({total_pages:2}, 3) === 2`；page=1、total_pages=0、page<=pages 均回 null。
  2. 越界重取：stub API.get 第一次回 `{ok:true, items:[], pagination:{page:2,total_pages:2}}` 且 self.page=3 → 第二次请求 URL 含 `page=2`，items 落位。
  3. silent：loadingKey 不翻转、showToast 零调用、reject 时 console.warn 而非 onError（Review Focus 3）。
  4. !ok 非 silent：toast `d.msg || errMsg`，items 不被清空。
  5. pagedList.doSearch：queryInput 带空白 → q 被 trim、page 归 1、触发 refresh。
- [ ] **Step 2: 确认失败** — `make fe-test`。
- [ ] **Step 3: 实现三函数。**
- [ ] **Step 4: 迁移五页**
  - review：LISTS 表原样保留（url/keys/after/warnMsg 即 spec），`loadList` 体缩为 `return loadPaged(this, LISTS[which], silent);`。
  - category：`Alpine.data('categoryPage', () => Object.assign(pagedList({ url: self => '/api/categories?page=' + self.page + '&page_size=10' + (self.q ? '&q=' + encodeURIComponent(self.q) : '') }), { addName:'', addDesc:'', init(){this.refresh();}, submitAdd, updateCategory, removeCategory }))`——后三个方法体不变。
  - search：doSearch 保留条件校验与 page 参数；取数段改 `this.page = page || 1; this._lastQuery = query; await loadPaged(this, SEARCH_SPEC, false);`，SEARCH_SPEC `{fetch: self => API.post('/api/search', {query: JSON.stringify(self._lastQuery), page: self.page, page_size: 20}), itemsKey:'results', loadingKey:'searching', errMsg:'搜索失败', after: self => { self.searched = true; self.saveHistory(self._lastQuery); }}`。pager 的 Fn="doSearch" 契约不变。
  - tag flat 分支与 detail load/reload：成功赋值前 `var c = clampToLastPage(data.pagination, this.page); if (c) { this.page = c; return this.refresh(); }`（detail 用 `data.detail.pagination`/`currentPage`，重取调自身）。两页其余语义（error 键、tree 分支、loadValues）不动——这是刻意决策：双视图/嵌套信封不强行套 loadPaged，只共享 clamp。
- [ ] **Step 5: 验证** — `make fe-test`、`make fe-check`；浏览器：分类页造 11 条数据翻到第 2 页删光该页条目 → 自动回第 1 页（Review Focus 1）；review 页轮询 3 秒无空态闪烁；search 翻页正常。
- [ ] **Step 6: Commit** — `chore(webui): wave 14b — loadPaged/pagedList/clampToLastPage unify five loaders, clamp protection reaches every pager`

### Task 7: AI 入队共享失败分支 API.enqueueAi

**Files:**
- Modify: `cmd/webui/static/js/core/api.js`、`pages/file/tree.js:222-246`、`pages/file/detail.js:288-311`、`pages/review/main.js:130-149`
- Test: `tests/js/api.test.mjs`

**Interfaces:**
- Produces: `API.enqueueAi(params)`（params 为 URLSearchParams）→ `d = await API.post('/api/ai/tasks', params); if (!d || !d.ok) { showToast((d && d.msg) || '发起识别失败', 'error'); return null; } return d;`。成功侧语义（queued===0→info、乐观更新、refresh）留在各页。

- [ ] **Step 1: 写失败测试** — loadCore(['api.js','utils.js'])，stub fetch 回 400+`{ok:false,msg:'目录不存在'}`：enqueueAi 回 null 且 toast 用服务端 msg；ok:true 时原样回 d（含 skipped/queued）。
- [ ] **Step 2: 确认失败 → Step 3: 实现 + 三站点改调** — 各页删掉逐字失败分支（含抄了两遍的注释），URLSearchParams 构造保留在调用点。
- [ ] **Step 4: 验证** — fe-test/fe-check；浏览器文件树勾选一个媒体文件批量识别成功、对不存在目录发起识别报错。
- [ ] **Step 5: Commit** — `chore(webui): wave 14c — API.enqueueAi absorbs three verbatim failure branches`

### Task 8: delegate 事件委托 + createSelection 多选集合

**Files:**
- Modify: `cmd/webui/static/js/core/utils.js`、`pages/tag/main.js:231-234`、`pages/search/main.js:107-111`、`pages/review/main.js:262-265,278-281`（四个 data-action 站点）；`pages/review/main.js:242-259`、`pages/file/tree.js:209-220`（selection）
- Test: `tests/js/interaction.test.mjs`

**Interfaces:**
- Produces:
  - `delegate(root, evtName, handlers)` — root.addEventListener，`el = e.target.closest('[data-action]')`，命中且 `handlers[el.getAttribute('data-action')]` 存在则调 `fn(el, e)`。仅覆盖单 data-action 方言；search onBuilderInput（复合选择器）与 tree bindPickEvents（once-bound + stopPropagation）**保持原样**，不强行参数化。
  - `createSelection(key)` — 返回 `{[key]: [], toggle(id, checked), toggleAll(ids, checked), allSelected(ids), prune(aliveIds)}`；toggle 即 filter/concat 惯用式单源。review/tree 保留现方法名（toggleOne/togglePick/pruneSelected/syncPicks 等）作薄包装，模板契约不动。

- [ ] **Step 1: 写失败测试** — delegate：stub event `{target:{closest: sel => fakeEl}}`，断言按 action 分发、未命中不调；createSelection('selected')：toggle 增删、toggleAll、prune 只留 aliveIds、allSelected 边界（空 ids 回 false）。
- [ ] **Step 2: 确认失败 → Step 3: 实现 + 迁移**（四处 if/else 链改 handlers 表；两页 selection 改包装）。
- [ ] **Step 4: 验证** — fe-test/fe-check；浏览器：review 全选/反选/批量通过、tag 面板行内按钮、search 构建器增删条件。
- [ ] **Step 5: Commit** — `chore(webui): wave 14d — delegate + createSelection single-source the panel dispatch and multi-select idioms`

### Task 9: 路径/存储/排序键工具函数

**Files:**
- Modify: `cmd/webui/static/js/core/utils.js`、`core/components.js:202-205`（sortControls init 改调 readSortState）
- Modify: decode 4 处 `pages/file/detail.js:41,72,85`、`pages/file/groups.js:17`；href 4 处 `pages/review/main.js:155-158,210-213`、`pages/file/tree.js:123`、`pages/file/groups.js:21-23`；storage 4 处 `core/components.js:241-243`、`pages/file/detail.js:98-106`、`pages/search/main.js:334-344,350-352`；sort 镜像 2 处 `pages/file/tree.js:27-28`、`pages/file/detail.js:44-45`
- Test: `tests/js/pathstore.test.mjs`

**Interfaces:**
- Produces（utils.js）:
  - `decodePath(raw)` — try decodeURI catch 回原值。
  - `pathFromLocation(prefix)` — `decodePath(window.location.pathname.substring(prefix.length))`。
  - `fileDetailHref(path)` — `'/files/detail/' + encodePath(path)`（HTML 拼接处调用方自行 escHtml，四处统一包）。
  - `readStoredJSON(storage, key, fallback)` — try JSON.parse(storage.getItem(key)) catch/空回 fallback（对象/数组由调用方给缺省）。
  - `readSortState(prefix, defaults)` — `{field: sessionStorage.getItem(prefix+'_sort_field') || defaults.field, order: ...}`；键名 schema 唯一来源改为组件与该函数共用同一拼接式（components.js init 改调它，树/详情两页 init 也调它，键副本消失）。

- [ ] **Step 1: 写失败测试** — decodePath 坏 URI（`'%E0%A4%A'`）回原值；readStoredJSON 坏 JSON/缺键回 fallback；readSortState 用 stub sessionStorage 断言键名与缺省。
- [ ] **Step 2: 确认失败 → Step 3: 实现 + 迁移全部站点。**
- [ ] **Step 4: 验证** — fe-test/fe-check；浏览器：含中文/空格路径的文件详情页与 groups 页正常加载；排序切换后刷新页面排序保持；search 历史仍在。
- [ ] **Step 5: Commit** — `chore(webui): wave 14e — decodePath/fileDetailHref/readStoredJSON/readSortState kill four idiom families`

### Task 10: 文件详情抓取合并 API.fileDetail + applyDetail

**Files:**
- Modify: `cmd/webui/static/js/core/api.js`、`pages/file/detail.js:173-213,256-258,303-305`、`pages/file/groups.js:28-31`
- Test: `tests/js/api.test.mjs`

**Interfaces:**
- Produces:
  - `API.fileDetail(path, opts)` — opts `{groupId, page, sort, order}`，拼 `/api/files/detail/` + encodePath(path) + 非空参数 query，返回 API.get 信封。
  - detail.js 页内 `applyDetail(d)`：currentGroupID===0 时选首组 + `this.detail = d`（两份逐字逻辑合一）；`mirrorFileField(f, key, val)`：写 f 与 dir_files 侧栏镜像（tag_done/ai_status 两处合一）。
- **附带 bug 修复**：reloadGroupsAndTags 改走与 load 相同的参数构造（**带 sort/order**），刷新后不再丢用户排序。

- [ ] **Step 1: 写失败测试** — API.fileDetail 拼 URL：opts 全给时 query 含 `group_id=2&page=3&sort=size&order=desc`；缺省只带传入项；path 含中文被 encodePath。
- [ ] **Step 2: 确认失败 → Step 3: 实现 + detail.js 合并** — load(sortOnly) 与 reloadGroupsAndTags 共用内部取数（URL 构造 + applyDetail + clamp），差异仅 error 处理与 renameName；groups.js 取名字改 `API.fileDetail(this.filePath)`。
- [ ] **Step 4: 验证** — fe-test/fe-check；浏览器：详情页切排序→删一个标签→列表刷新后排序保持（修复点）；切组/翻页正常；侧栏 tag_done/AI 徽章镜像仍同步。
- [ ] **Step 5: Commit** — `chore(webui): wave 14f — API.fileDetail + applyDetail merge detail fetches, reload keeps user sort`

### Task 11: config 页内抽取 + 波尾（→ 5.8.0）

**Files:**
- Modify: `pages/config/main.js:148-215`（六个规则扫描器）、`:313-316`（fieldError）、`:450-454`（containerPayload）
- Modify: `docs/CHANGELOG_webui.md`、`cmd/webui/constant/constant.go`（5.8.0）

**Interfaces:**
- Produces（config 页内，不进 core）: `keptRows(f)`（DynamicArray trim+滤空行，校验与提交共用——漂移即本地放行服务端拒收）；`findRule(rules, kind)`（回首个匹配 rule 或 null，六个扫描器改调）。

- [ ] **Step 1: 抽取并替换**（页内函数，无跨页消费者；fe-check 语法门 + 浏览器验证即测试）。
- [ ] **Step 2: 浏览器验证** — config 页 DynamicArray 字段加空行/空白行保存，校验提示与提交行为一致；secret 字段 hint、widget 渲染、shift 步进各冒烟一次。
- [ ] **Step 3: 全量 make test + make fe-test + CHANGELOG 5.8.0（EN+ZH，覆盖 T5-T11，注明 okType 归一与 clamp 普及两处行为变化）+ WebuiVersion。**
- [ ] **Step 4: Commit** — `chore(webui): wave 14 wrap-up — config keptRows/findRule + changelog 5.8.0`

---

## Wave 15 — CSS 组件化与死代码（→ 5.9.0）

### Task 12: 死代码清除 + 同义变体合并

**Files:**
- Modify: `layout/layout.css:371-386`（删 .scan-result）、`:243-270`（dir-sidebar 合并）、`:318-321`（fadeUpIn 迁 animations.css）、`:364-368`（.result-item 保留）
- Modify: `components/config.css:56-61`（删 .config-invalid-badge）、`partials/config_group.html:11`（改 `badge badge-red`）、`components/tree.css:57-60`（删四条配色）、`components/tags.css:55-70`（.tag-chip 并入 .tag + .tag-lg 修饰）、`core/animations.css`（收 fadeUpIn）
- Modify: `pages/file/tree.js:152`（aiBadgeHtml 改 `meta.badgeClass`）、`core/utils.js:55-60`（AI_STATUS_META 删 treeClass 字段——先 grep 确认 tree.js 是唯一消费者）、`file_detail.html:101`（tag-chip → tag tag-lg）、`file.html`（dir-sidebar 加 scroll 修饰）、`search.html:42-46`（改挂 .result-item/.file-name/.file-meta，删覆盖内联）
- Test: `cmd/webui/static_test.go`

**Interfaces:**
- Produces: `.dir-sidebar` 基类（无 overflow）+ `.dir-sidebar.scroll { overflow-y:auto }`；`.tag-lg`（padding 3px 11px、gap 4px）；AI 状态配色唯一来源 = tags.css 的 `badge-ai-*`（tree 行挂 `tree-status badge-ai-*`，.tree-status 只留尺寸）。

- [ ] **Step 1: 写失败回归测试** — static_test.go 加用例：layout.css 不含 `.scan-result`；config.css 不含 `config-invalid-badge`；tree.css 不含 `tree-status-ai-`；tags.css 不含 `.tag-chip`；animations.css 含 `@keyframes fadeUpIn`；layout.css 不含 `@keyframes fadeUpIn`。
- [ ] **Step 2: 确认失败 → Step 3: 按 Files 清单执行合并/删除/迁移。**
- [ ] **Step 4: 验证** — go test/fe-check/fe-test；浏览器：文件树 AI 徽章四色不变、config 无效徽章仍红、详情页标签 chip 尺寸不变、侧栏滚动（file 页可滚、detail 页不滚）、search 结果行样式、表格/树入场动画仍在。
- [ ] **Step 5: Commit** — `chore(webui): wave 15a — dead CSS purged, badge/tag/sidebar variants merged to single sources`

### Task 13: query-builder.css —— 最大的隐形组件显形

**Files:**
- Create: `cmd/webui/static/css/components/query-builder.css`
- Modify: `pages/search/main.js:175-252`（qb-* 内联 style 全部迁出为类）、`templates/base.html:8-19`（挂新链接，置于 tree.css 之后）
- Test: `cmd/webui/static_test.go`

**Interfaces:**
- Produces: `.qb-row/.qb-group/.qb-header/.qb-type/.qb-op/.qb-val-*` 正式类（@layer components）。玻璃面板配方本任务先按字面值落 CSS，T14 引入 .panel 后 qb-group 改挂 .panel 删配方。

- [ ] **Step 1: 写失败测试** — static_test.go：query-builder.css 存在且含 `.qb-group`；search/main.js 不含 `style="`（qb 渲染段全部类化；若个别一次性定位保留，测试改为断言 `blur(8px)` 不在 search/main.js）。
- [ ] **Step 2: 确认失败 → Step 3: 逐段把 JS 字符串里的内联样式搬进 CSS 类**（约 15 段；两份玻璃面板配方合为 .qb-group 一条）。
- [ ] **Step 4: 验证** — go test/fe-check；浏览器 search 页：加条件/嵌套组/切 and-or/删组，视觉与重构前截图一致；双主题各看一次。
- [ ] **Step 5: Commit** — `chore(webui): wave 15b — query-builder styles leave JS strings for a real component file`

### Task 14: card/panel/toolbar/modal 组件化 + 波尾（→ 5.9.0）

**Files:**
- Create: `components/card.css`（.panel 基类 + card 家族自 layout.css:152-227 迁出）、`components/toolbar.css`（`.toolbar { display:flex; align-items:center; gap:8px }` + `.toolbar-wrap { flex-wrap:wrap }`）、`components/modal.css`（notifications.css:39-84 modal 段迁出）
- Modify: `layout/layout.css`（card 家族删出）、`components/notifications.css`（modal 段删出；.inline-edit-input :87-112 迁 forms.css）、`components/forms.css`（收 inline-edit-input；加 `input[readonly]` 规则取代 base.html:105 的 :style 三元）、`components/tree.css:77-88`（tag-content-panel 改挂 .panel）、`forms.css:65-77`（inline-add 改 .panel dashed 变体）
- Modify: `templates/base.html`（三个新链接；:105 readonly 绑定删）、`file_detail.html:107,133`（内联 → 新 `.card-section`）、`:118,121,130`（内联 → 新 `.media-preview`）、约 10 处内联 flex 工具栏 div（category/file/review/search/file_detail，见审计 A3 清单）→ `.toolbar`；`file.html`/`layout.css` stat-card、dir-sidebar 与 `pages/search/main.js` qb-group 挂 .panel
- Modify: `docs/CHANGELOG_webui.md`、`cmd/webui/constant/constant.go`（5.9.0）

**Interfaces:**
- Produces: `.panel`（glass-bg + blur(var(--glass-blur)) + glass-border + radius + glass-shadow 唯一配方）；`.card-section { margin-top:16px; padding-top:16px; border-top:1px solid var(--border) }`；`.media-preview`（max-height:600px 等，按 file_detail 现内联值）；`.toolbar`。

- [ ] **Step 1: 写失败测试** — static_test.go：card.css 含 `.panel`；notifications.css 不含 `.modal`/`.inline-edit-input`；layout.css 不含 `.card-header`；base.html 含三个新 css 链接。
- [ ] **Step 2: 确认失败 → Step 3: 迁移与替换**（.panel 各消费者只留差异声明；CSS 链接顺序：card/toolbar/modal 插在现有 components 之后、animations 之前）。
- [ ] **Step 4: 验证** — go test/fe-check/fe-test；浏览器全站九页冒烟 + 双主题：首页统计卡、弹窗（打开/保存/Esc）、行内编辑输入框、详情页媒体预览与分节线、各页工具栏间距。
- [ ] **Step 5: 全量 make test + CHANGELOG 5.9.0 + WebuiVersion。**
- [ ] **Step 6: Commit** — `chore(webui): wave 15 wrap-up — panel/card/toolbar/modal component files, glass recipe single-sourced + changelog 5.9.0`

---

## Wave 16 — 标量 token（→ 5.10.0）

### Task 15: 字号/blur/光晕/圆角/过渡 token

**Files:**
- Modify: `core/variables.css`（两主题共用的标量放 :root）
- Modify: `components/*.css`、`layout/layout.css`、`core/base.css`、`core/animations.css`、`pages/search/main.js`（残留内联 blur/圆角）
- Test: `cmd/webui/static_test.go`

**Interfaces:**
- Produces（:root 新增）:
  - `--text-2xs:10px; --text-xs:11px; --text-sm:12px; --text-md:13px; --text-lg:14px; --text-xl:16px; --text-2xl:22px`
  - `--glass-blur-xs:4px; --glass-blur-sm:8px; --glass-blur-lg:24px`（--glass-blur:18px 保留）
  - `--glow-sm: 0 2px 8px var(--accent-glow); --glow-md: 0 4px 16px var(--accent-glow); --glow-lg: 0 6px 24px var(--accent-glow)`
  - `--radius-xs:4px; --radius-pill:999px`；`--transition-fast:0.15s ease`
- 替换映射：10/11/12/13/14/16/22px 字号→对应 token（约 90 处）；blur(4/8/24px)→xs/sm/lg（约 22 处，含 JS 内联）；`0 2px 8px var(--accent-glow)` 等三配方→glow token（9 处）；硬写 6px 圆角→var(--radius-sm)、4px→--radius-xs、badge 9px 与 toggle 20px→--radius-pill；`0.15s ease`→--transition-fast（4 处）。**不改任何数值本身**——纯等价替换，视觉零变化。

- [ ] **Step 1: 写失败测试** — static_test.go：components/*.css 与 layout.css 中不匹配正则 `font-size:\s*\d+px` 与 `blur\(\d+px\)`（variables.css 除外）；variables.css 含 `--text-md` 与 `--glow-sm`。
- [ ] **Step 2: 确认失败 → Step 3: 定义 token + 全站等价替换**（逐文件过一遍，替换后 grep 复核零残留）。
- [ ] **Step 4: 验证** — go test/fe-check；浏览器双主题抽查首页/文件树/search/config 四页截图，与替换前对比无视觉差异。
- [ ] **Step 5: Commit** — `chore(webui): wave 16a — text/blur/glow/radius/transition scalar tokens, ~130 literals tokenized`

### Task 16: 输入框宽度/列宽/form-help 类 + 波尾（→ 5.10.0）

**Files:**
- Modify: `components/forms.css`（`.input-w-sm:120px / .input-w-md:200px / .input-w-lg:280px / .input-w-xl:360px`；`.form-help { font-size:var(--text-xs); display:block; margin-top:2px }`）、`components/table.css`（`.col-w-60/.col-w-80/.col-w-100`）
- Modify: 约 15 处内联宽度（category.html:10,13；file.html:10,21；file_groups.html:15；review.html:11；tag.html:117,122,131；file_detail.html:111,158,164；partials/search_form.html:8）按最近档归档：110-140→sm，170-220→md，240-280→lg，320-360→xl；9 处 `th style="width:…"` → 列宽类；`base.html:110` help 内联 → .form-help
- Modify: `docs/CHANGELOG_webui.md`、`cmd/webui/constant/constant.go`（5.10.0）
- Test: `cmd/webui/static_test.go`、`cmd/webui/templates_test.go`

- [ ] **Step 1: 写失败测试** — templates_test.go：渲染 category.html content 不含 `style="width:`（需 base+partials 全量 ParseFS，参照 main.go:53-76 的组装方式）；static_test.go：forms.css 含 `.input-w-md`。
- [ ] **Step 2: 确认失败 → Step 3: 加类 + 替换**（宽度归档是刻意的设计收敛，个别输入框 ±20-40px 属预期，changelog 注明）。
- [ ] **Step 4: 验证** — go test/fe-check/fe-test；浏览器各表单页目测输入框宽度协调、表头列宽不变形。
- [ ] **Step 5: 全量 make test + CHANGELOG 5.10.0 + WebuiVersion。**
- [ ] **Step 6: Commit** — `chore(webui): wave 16 wrap-up — input width tiers, column-width and form-help classes + changelog 5.10.0`

---

## Self-Review 记录

- 覆盖：审计 JS 10 项 → T5-T11（页内 walk/mirror 中 mirrorFileField 入 T10，walkTagTree 明确范围外）；模板 A1-A7 → T2/T3/T4/T16（A4 整卡 partial 不做，宽度档+submitBtn 中 submitBtn 收益小不做）；CSS B1-B5 → T1/T12/T13/T14/T15/T16（sidebar token、间距阶梯明确范围外）。
- 类型一致：loadPaged spec 键名与 review LISTS 现有键逐字对齐；pagedList 标准命名与 searchForm/pager partial 方言（queryInput/doSearch/goToPage/pagination）对齐；confirmDelete opts 与 reportResult 参数序一致。
- Review Focus 五条各有归属测试：1→T6、2→T5、3→T6、4→T3、5→T1。