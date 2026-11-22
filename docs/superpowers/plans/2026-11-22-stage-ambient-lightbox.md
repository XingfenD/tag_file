# 舞台氛围底 + GLightbox 查看器 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 详情页媒体舞台以内容自身的模糊底适配任意比例，并接入 GLightbox 实现点击放大查看。

**Architecture:** 氛围底 = 舞台内两层（`.stage-ambient` cover 模糊铺底 + `.stage-media` contain 主体），Alpine `x-if` 单根限制用 `.stage-layer { display: contents }` 包装解决；查看器 = vendor 的 GLightbox 3.3.1，图片点击主体打开、视频用独立展开按钮打开（不劫持原生控制条），每次点击新建单元素实例。

**Tech Stack:** Go 1.25 + Gin 模板、Alpine.js（已 vendor）、纯 CSS、GLightbox 3.3.1（vendor）。

**Spec:** `docs/superpowers/specs/2026-11-22-stage-ambient-lightbox-design.md`（已批准；补充裁决：舞台图片不加圆角）

## Global Constraints

- 第三方库一律 vendor 进 `cmd/webui/static/vendor/`（扁平命名，同 `alpinejs.min.js`/`sortable.min.js`），无 CDN、无构建步骤。
- vendor 下载源用 unpkg（本机网络到 jsdelivr 不通）：`https://unpkg.com/glightbox@3.3.1/dist/js/glightbox.min.js`、`https://unpkg.com/glightbox@3.3.1/dist/css/glightbox.min.css`。
- WebUI 改动记 `docs/CHANGELOG_webui.md`：一条英文短句紧跟一行中文短句，不同条目空行分隔；本迭代合并为一条 [6.1.0]，`cmd/webui/constant/version.go` 与首次落条目同 commit。
- 测试只用 `go test ./cmd/webui/...`（PATH 需含 `/usr/local/go/bin`）；验证容器 `tagfile-webui` 重建：`docker build -t tagfile-webui . && docker rm -f tagfile-webui && docker run -d --name tagfile-webui -p 8088:8088 -v "/Users/xingfend/Movies/暂时:/data" tagfile-webui`；:8088 只读验证，绝不 wipe `/data/.tagfile`。
- commit 前 `git branch --show-current` 确认不在 master。

## Review Focus

- 极端比例（竖图/横条）：ambient 必须盖满整个舞台（cover + 外扩 inset 吃模糊白边），主体 contain 不被裁——钉在 Task 1 的 CSS token 断言。
- 视频原生控制条：点视频画面是播放/暂停，不得误触查看器——查看器入口是独立展开按钮，钉在 Task 2 的模板断言。
- 就地切换文件（pushState）：查看器每次点击新建实例、src 取当前 `detail.file.id`，不留旧文件残影——钉在 Task 2 的 detail.js token。
- 音频 / 不支持格式 / 非媒体：不渲染 ambient，保留现有深色底——钉在 Task 1 的模板计数断言（ambient 恰两处）。
- 浮层互斥：GLightbox 遮罩须盖过全站 `.modal-overlay`/toast，Esc 可关——Task 2 浏览器结构验证步骤覆盖（z-index 由库自带，验证加载顺序在全部站内 CSS 之后）。

---

### Task 1: 舞台氛围底（ambient backdrop）

**Files:**
- Modify: `cmd/webui/templates/file_detail.html`（image / video 两个 `template x-if` 块）
- Modify: `cmd/webui/static/css/components/workbench.css`（`.stage-view` 区，约 86-101 行）
- Modify: `cmd/webui/static_test.go`（新增 `TestStageAmbientBackdrop`）
- Modify: `docs/CHANGELOG_webui.md`、`cmd/webui/constant/version.go`

**Interfaces:**
- Produces: DOM 契约 `.stage-layer` / `.stage-ambient` / `.stage-media`（Task 2 的点击绑定与样式依赖这三个类名）。

- [ ] **Step 1: 写失败测试** `TestStageAmbientBackdrop`（放 `TestTreeStickyHierarchy` 之后）
  - `templates/file_detail.html`：`strings.Count(html, `class="stage-ambient"`) == 2` 且 `strings.Count(html, `class="stage-media"`) == 2`（图片+视频各一，音频/fallback 不加——计数即断言）。
  - `static/css/components/workbench.css`：包含 `.stage-layer`、`display: contents`、`.stage-ambient`、`object-fit: cover`、`inset: -36px`、`pointer-events: none`、`.stage-media`、`object-fit: contain`。
  - 无圆角裁决：截取 `.stage-media` 规则体（从 `.stage-media {` 到下一个 `}`），其中不得含 `border-radius`。

- [ ] **Step 2: 跑测试确认失败** `go test ./cmd/webui/ -run TestStageAmbientBackdrop` → FAIL。

- [ ] **Step 3: 实现**
  - `file_detail.html`：image 块改为 `<div class="stage-layer">` 内两个 `<img>`（`.stage-ambient` 带 `aria-hidden="true"`、空 `alt`，与主体同 `:src="'/api/files/raw/' + detail.file.id"`）；video 块同理 `.stage-ambient` 用 `<video muted playsinline preload="auto" aria-hidden="true">` 同 src，主体保留 `controls`。
  - `workbench.css`：原 `.stage-view img, .stage-view video` 规则改为只作用 `.stage-media`（`position: relative; max-width: 100%; max-height: 100%; object-fit: contain; box-shadow: 0 12px 40px rgba(0,0,0,0.35)`，无 border-radius）；新增 `.stage-layer { display: contents; }` 与 `.stage-ambient { position: absolute; inset: -36px; width: auto; height: auto; object-fit: cover; filter: blur(32px) saturate(1.25) brightness(0.5); pointer-events: none; }`；`.stage-view` 补 `position: relative`。
  - `docs/CHANGELOG_webui.md` 顶部新增 [6.1.0] - 2026-11-22 / Added：氛围底一句（英+中）；`version.go` → `"6.1.0"`。

- [ ] **Step 4: 跑测试确认通过** `go test ./cmd/webui/...` → 全 ok（含既有门不回归）。

- [ ] **Step 5: 视觉验证** 重建容器；打开索引图片详情页（如 `/files/detail/「泪如萌下@leileidwq /P/-6210492276337742667_121.jpg`），`evaluate_script` 断言 `.stage-ambient` computed `filter` 含 `blur(32px)`、`objectFit: cover`、与 `.stage-media` 同 src；内嵌浏览器可见时截图确认无暗绿空底。

- [ ] **Step 6: Commit**（模板 + CSS + 测试 + CHANGELOG + 版本常量同 commit）
  `feat(webui): ambient media-derived stage backdrop`

### Task 2: GLightbox 点击放大查看

**Files:**
- Create: `cmd/webui/static/vendor/glightbox.min.js`、`cmd/webui/static/vendor/glightbox.min.css`（unpkg 下载，见 Global Constraints）
- Modify: `cmd/webui/templates/base.html`（head CSS 链接 + 底部脚本）
- Modify: `cmd/webui/templates/file_detail.html`（`.stage-media` 绑定）
- Modify: `cmd/webui/static/js/pages/file/detail.js`（`openViewer()`）
- Modify: `cmd/webui/static/css/components/workbench.css`（`cursor: zoom-in`、`.stage-expand` 按钮样式）
- Modify: `cmd/webui/static_test.go`（新增 `TestStageLightbox`）
- Modify: `docs/CHANGELOG_webui.md`（并入 [6.1.0] 条目，不再 bump）

**Interfaces:**
- Consumes: Task 1 的 `.stage-media` / `.stage-view` 类名；`fileDetailPage` 的 `detail.file.id`、`detail.media_type`、`detail.video_playable`。
- Produces: `openViewer()`（`fileDetailPage` 方法）；vendor 路径 `/static/vendor/glightbox.min.js`、`/static/vendor/glightbox.min.css`。

- [ ] **Step 1: 写失败测试** `TestStageLightbox`
  - `staticFS.ReadFile("static/vendor/glightbox.min.js")`、`.../glightbox.min.css` 存在且非空。
  - `templates/base.html` 含 `/static/vendor/glightbox.min.css` 与 `/static/vendor/glightbox.min.js`，且 CSS 链接位于 `/static/css/core/animations.css` 之后（加载顺序断言：`strings.Index` 比较）。
  - `templates/file_detail.html`：image 主体含 `@click="openViewer()"`；video 主体不含 `@click`（防劫持控制条），存在 `class="stage-expand"` 按钮且其 `@click="openViewer()"`。
  - `js/pages/file/detail.js`：含 `openViewer`、`new GLightbox(`、`type: 'image'`、`type: 'html'`。

- [ ] **Step 2: 跑测试确认失败** `go test ./cmd/webui/ -run TestStageLightbox` → FAIL（vendor 缺失）。

- [ ] **Step 3: vendor 下载** 两条 unpkg `curl -sL -o`（Global Constraints 的确切 URL），`ls -la` 确认字节数 ~56KB/~14KB。

- [ ] **Step 4: 实现**
  - `base.html`：head 内 `/static/css/core/animations.css` 之后加 glightbox CSS link；底部脚本区 `/static/js/core/api.js` 之前加 glightbox `<script>`。
  - `file_detail.html`：image `.stage-media` 加 `@click="openViewer()"`；video 块在 `.stage-layer` 内加 `<button type="button" class="stage-expand" title="放大查看" aria-label="放大查看" @click="openViewer()">`（图标用 sprite `search`）；video `.stage-media` 不加 click。
  - `detail.js`：`fileDetailPage` 新增 `openViewer: function()`——image 时 `new GLightbox({ elements: [{ href: '/api/files/raw/' + this.detail.file.id, type: 'image' }] }).open()`；video（`video_playable`）时 `type: 'html'`，content 为 `<video controls autoplay playsinline src="...">` 字符串（src 只有数字 id，无需转义）；每次点击新建实例（切换文件天然无残留），非媒体/不可播直接 return。
  - `workbench.css`：`.stage-media`（img）加 `cursor: zoom-in`；`.stage-expand` 绝对定位右上角、玻璃小钮样式。
  - CHANGELOG [6.1.0] Added 追加一句（英+中）：舞台可点击放大查看（GLightbox vendor）。

- [ ] **Step 5: 跑测试确认通过** `go test ./cmd/webui/...` → 全 ok。

- [ ] **Step 6: 浏览器结构验证** 重建容器，打开图片详情页：`evaluate_script` 触发 `.stage-media` click，断言 DOM 出现 `.glightbox-container` 且含同 src 的 img；再开一个视频详情页，断言 video `.stage-media` 无 click 绑定路径（点它不产生 glightbox 容器）、点 `.stage-expand` 产生容器。可见时截图确认放大态。

- [ ] **Step 7: Commit**（vendor + 全部改动 + CHANGELOG 追加同 commit）
  `feat(webui): click-to-zoom media viewer via vendored GLightbox`
