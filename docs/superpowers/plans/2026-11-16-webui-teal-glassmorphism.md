# WebUI 青碧主题 + 玻璃拟态质感升级 — 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 WebUI 主题色从靛紫整体切换为青碧 Teal，并按 glassmorphism 技能补齐玻璃四要素（有料背景、inset 亮边、防色带噪点、镜面扫光）与三条可访问性降级路径。

**Architecture:** 纯静态层改动——全部颜色/阴影/背景集中在 `variables.css` 的 CSS 变量，组件 CSS 只清理硬编码残留并消费变量；噪点是 `base.html` 里一个全局 fixed div；降级用文件末尾非分层的 `@media` / `@supports` 块覆盖。不改任何 Go 逻辑与 JS。

**Tech Stack:** 手写 CSS（`@layer` 组织）、Go html/template、SVG favicon、headless Chrome 截图验收。

**Spec:** `docs/superpowers/specs/2026-11-16-webui-teal-glassmorphism-design.md`

## Global Constraints

- 分支 `feat/webui-teal-glass`；每次 `git commit` 前跑 `git branch --show-current` 确认不在 master。
- 只改这些文件：`cmd/webui/static/css/core/variables.css`、`core/base.css`、`layout/layout.css`、`components/tags.css`、`cmd/webui/static/favicon.svg`、`cmd/webui/templates/base.html`、`cmd/webui/constant/version.go`、`docs/CHANGELOG_webui.md`。
- 不动 Go 逻辑、路由、API、JS（含 `theme.js`）；`base.html` 唯一改动是加 grain div。
- 保留现有 `@layer reset, base, layout, components, utilities` 组织；变量改动写在对应 layer 内，降级块写在文件末尾**非分层**位置（unlayered 优先级高于 layered）。
- 任何 `backdrop-filter` 必须成对带 `-webkit-backdrop-filter`；已有的不许删。
- 不动画 blur 半径；扫光只动画 `transform`；噪点全局仅一层，`opacity ≤ 0.06`。
- 版本 `4.4.2 → 4.5.0`；CHANGELOG 条目英文行紧跟中文行（中间无空行），不同条目之间空一行；版本 bump 与 CHANGELOG 同一 commit。
- 语义色 danger/success/warning 不动。

## Review Focus

按 spec 反推、无任务测试直接覆盖、最可能咬人的五类，各自钉在所属任务里：

1. **对比度**：`--accent` 文字（链接、stat 数字、徽章）在背景最亮区域（右上 teal 光晕 + 暖沙色斑处）仍 ≥ 4.5:1（40px stat 数字按大字 ≥ 3:1）→ Task 5 截图逐页目检。
2. **噪点层**：不拦截任何点击（modal/toast/dropdown 均可交互）、reduced-transparency 下消失、肉眼只是"去色带"而非"脏点" → Task 2 样式断言 + Task 5 截图。
3. **扫光**：光带从文字下方掠过不遮挡 `.num`/`.label`；每次 hover 只扫一次不循环 → Task 4 CSS 层级断言。
4. **暗色侧栏**：`--sidebar-active` teal 在深底上可读，active 项的 `--accent-bg` 底色在亮/暗两主题都不刺眼 → Task 3 变量替换断言 + Task 5 暗色截图。
5. **紫色零残留**：全仓 grep 旧紫色值（含 favicon.svg）零命中 → Task 3 与 Task 5 各一次全仓 grep。

---

### Task 1: Teal 色板与玻璃阴影变量（variables.css）

**Files:**
- Modify: `cmd/webui/static/css/core/variables.css`

**Interfaces:**
- Consumes: 无（一切后续任务的变量源头）。
- Produces: 后续任务引用的变量——既有的 `--accent`、`--accent-dim`、`--accent-bg`、`--accent-glow`、`--glass-shadow`、`--glass-shadow-hover`、`--glass-border`、`--sidebar-active`、`--ink*`，及**新增**的 `--glow-a`、`--glow-b`、`--bg-blobs`、`--sweep-color`。

- [ ] **Step 1: 改亮色块（`:root`，`@layer base` 内）**

按 spec §6.1 表逐项替换，并新增以下变量（精确值）：

```css
--bg-gradient: linear-gradient(135deg, #dff2ee 0%, #e2edf6 50%, #f6efe1 100%);
--accent: #0d9488;
--accent-dim: #0f766e;
--accent-bg: rgba(13, 148, 136, 0.10);
--accent-glow: rgba(13, 148, 136, 0.25);
--ink: #182422;
--ink-light: #475855;
--ink-muted: #7e918d;
--sidebar-active: #2dd4bf;
--glass-border: rgba(255, 255, 255, 0.55);
--glass-border-hover: rgba(255, 255, 255, 0.70);
--glass-shadow: 0 8px 32px rgba(24, 54, 50, 0.08),
  inset 0 1px 0 rgba(255, 255, 255, 0.50),
  inset 0 -1px 0 rgba(24, 54, 50, 0.06);
--glass-shadow-hover: 0 12px 40px rgba(24, 54, 50, 0.12),
  inset 0 1px 0 rgba(255, 255, 255, 0.60),
  inset 0 -1px 0 rgba(24, 54, 50, 0.08);
--glow-a: rgba(13, 148, 136, 0.25);
--glow-b: rgba(214, 158, 86, 0.12);
--sweep-color: rgba(255, 255, 255, 0.35);
--bg-blobs:
  radial-gradient(48vw 48vw at 12% 18%, rgba(13, 148, 136, 0.14), transparent 60%),
  radial-gradient(42vw 42vw at 85% 30%, rgba(56, 132, 255, 0.10), transparent 60%),
  radial-gradient(55vw 55vw at 70% 90%, rgba(214, 158, 86, 0.12), transparent 65%);
```

其余未列出的变量（字体、radius、语义色、`--glass-bg`、`--glass-blur` 等）保持原值。

- [ ] **Step 2: 改暗色块（`[data-theme="dark"]`）**

```css
--bg-gradient: linear-gradient(135deg, #0a1416 0%, #0d2129 50%, #101a22 100%);
--accent: #2dd4bf;
--accent-dim: #14b8a6;
--accent-bg: rgba(45, 212, 191, 0.12);
--accent-glow: rgba(45, 212, 191, 0.18);
--ink: #e6f0ee;
--ink-light: #aac2bc;
--ink-muted: #66837d;
--sidebar-active: #5eead4;
--glass-shadow: 0 8px 32px rgba(0, 0, 0, 0.25),
  inset 0 1px 0 rgba(255, 255, 255, 0.10),
  inset 0 -1px 0 rgba(0, 0, 0, 0.15);
--glass-shadow-hover: 0 12px 40px rgba(0, 0, 0, 0.35),
  inset 0 1px 0 rgba(255, 255, 255, 0.14),
  inset 0 -1px 0 rgba(0, 0, 0, 0.20);
--glow-a: rgba(45, 212, 191, 0.14);
--glow-b: rgba(45, 212, 191, 0.07);
--sweep-color: rgba(255, 255, 255, 0.16);
--bg-blobs:
  radial-gradient(48vw 48vw at 12% 18%, rgba(45, 212, 191, 0.10), transparent 60%),
  radial-gradient(42vw 42vw at 85% 30%, rgba(56, 132, 255, 0.08), transparent 60%),
  radial-gradient(55vw 55vw at 70% 90%, rgba(214, 158, 86, 0.06), transparent 65%);
```

`--glass-border`（暗）保持 `rgba(255,255,255,0.08)` 不变。

- [ ] **Step 3: 验证**

```bash
grep -c 'inset 0 1px 0' cmd/webui/static/css/core/variables.css   # 期望 4
grep -E 'glow-a|glow-b|bg-blobs|sweep-color' cmd/webui/static/css/core/variables.css | wc -l  # 期望 ≥ 8（两主题各 4）
grep -E '6366f1|818cf8|a5b4fc|4f46e5|99, 102, 241|129, 140, 248' cmd/webui/static/css/core/variables.css  # 期望零命中
```

- [ ] **Step 4: Commit**

```bash
git branch --show-current   # 必须是 feat/webui-teal-glass
git add cmd/webui/static/css/core/variables.css
git commit -m "feat(webui): switch theme palette to teal, add inset glass shadows and glow/blob variables"
```

---

### Task 2: 背景色斑、grain 噪点层、降级三件套（base.css + base.html）

**Files:**
- Modify: `cmd/webui/static/css/core/base.css`
- Modify: `cmd/webui/templates/base.html`

**Interfaces:**
- Consumes: Task 1 的 `--glow-a`、`--glow-b`、`--bg-blobs`、`--bg-gradient`。
- Produces: `.grain` 类（`base.html` 的 div 消费）；降级块覆盖 Task 1 全部玻璃变量。

- [ ] **Step 1: body 背景叠加色斑**

`body` 的 `background: var(--bg-gradient);` 改为 `background: var(--bg-blobs), var(--bg-gradient);`（`background-attachment: fixed` 保留，作用于全部层）。

- [ ] **Step 2: 两个环境光晕改变量、放大**

`body::before`：`background: radial-gradient(circle, var(--accent-glow), transparent 70%)` → `var(--glow-a)`；`width/height: 500px` → `640px`。
`body::after`：`rgba(236, 72, 153, 0.12)` → `var(--glow-b)`；`width/height: 400px` → `520px`。

- [ ] **Step 3: `.grain` 样式（加在 `@layer base` 的 Utility 区）**

```css
.grain {
  position: fixed; inset: 0; z-index: 10000; pointer-events: none;
  opacity: 0.045;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='300' height='300'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.85' numOctaves='2' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
}
```

- [ ] **Step 4: 降级块（base.css 文件末尾，`@layer` 之外）**

```css
/* ── Reduced transparency: solid panels, no frost, no grain ── */
@media (prefers-reduced-transparency: reduce) {
  :root {
    --glass-bg: #ffffff; --glass-bg-strong: #ffffff;
    --surface: #f2f5f4; --sidebar-bg: #14201e;
  }
  [data-theme="dark"] {
    --glass-bg: #1c2a28; --glass-bg-strong: #20302d;
    --surface: #1a2725; --sidebar-bg: #0a1211;
  }
  *, *::before, *::after {
    backdrop-filter: none !important;
    -webkit-backdrop-filter: none !important;
  }
  .grain { display: none; }
}

/* ── No backdrop-filter support: opaque-enough glass ── */
@supports not ((backdrop-filter: blur(1px)) or (-webkit-backdrop-filter: blur(1px))) {
  :root {
    --glass-bg: rgba(255, 255, 255, 0.88);
    --glass-bg-strong: rgba(255, 255, 255, 0.92);
  }
  [data-theme="dark"] {
    --glass-bg: rgba(23, 37, 35, 0.92);
    --glass-bg-strong: rgba(28, 44, 41, 0.95);
  }
}
```

放在现有 `prefers-reduced-motion` 块之后；该块已有的规则不动。

- [ ] **Step 5: base.html 加 grain div**

`<body>` 之后、`<aside class="sidebar">` 之前插入：

```html
<div class="grain" aria-hidden="true"></div>
```

- [ ] **Step 6: 验证**

```bash
go test ./cmd/webui/...   # 模板解析与路由 smoke 测试全绿（base.html 改动唯一的自动化回归面）
grep -n 'bg-blobs\|glow-a\|glow-b' cmd/webui/static/css/core/base.css        # 3 处命中
grep -n 'class="grain"' cmd/webui/templates/base.html                        # 1 处命中
grep -c 'prefers-reduced-transparency' cmd/webui/static/css/core/base.css    # 1
grep -c '@supports not' cmd/webui/static/css/core/base.css                   # 1
grep -n '236, 72, 153' cmd/webui/static/css/core/base.css                    # 零命中
```

- [ ] **Step 7: Commit**

```bash
git branch --show-current
git add cmd/webui/static/css/core/base.css cmd/webui/templates/base.html
git commit -m "feat(webui): layered background blobs, global grain overlay, reduced-transparency and no-backdrop-filter fallbacks"
```

---

### Task 3: 硬编码紫色清理（layout.css、tags.css、favicon.svg）

**Files:**
- Modify: `cmd/webui/static/css/layout/layout.css:57-64`（`.sidebar nav a.active`）
- Modify: `cmd/webui/static/css/components/tags.css:15,24,59,66`
- Modify: `cmd/webui/static/favicon.svg:4-5`

**Interfaces:**
- Consumes: Task 1 的 `--accent-bg`、`--accent-glow`。
- Produces: 无新接口；产出是「主题色单一来源」不变量（全仓紫色 grep 零命中）。

- [ ] **Step 1: layout.css 侧栏 active**

```css
/* 旧 */ background: rgba(129,140,248,0.12);
/* 新 */ background: var(--accent-bg);
/* 旧 */ box-shadow: 0 2px 8px rgba(129,140,248,0.15), inset 0 0 0 1px rgba(129,140,248,0.18);
/* 新 */ box-shadow: 0 2px 8px var(--accent-glow), inset 0 0 0 1px var(--accent-glow);
```

- [ ] **Step 2: tags.css 四处**

`border: 1px solid rgba(99, 102, 241, 0.12)`（×2）→ `border: 1px solid var(--accent-bg)`；
`background: rgba(99, 102, 241, 0.18)`（×2）→ `background: var(--accent-glow)`。

- [ ] **Step 3: favicon.svg 渐变**

`stop-color="#818cf8"` → `#2dd4bf`；`stop-color="#6366f1"` → `#0d9488`。

- [ ] **Step 4: 验证（全仓紫色零残留——Review Focus #5 第一道）**

```bash
grep -rEn '6366f1|818cf8|a5b4fc|4f46e5|129, ?140, ?248|99, ?102, ?241|236, ?72, ?153' cmd/webui/   # 零命中
go test ./cmd/webui/...
```

- [ ] **Step 5: Commit**

```bash
git branch --show-current
git add cmd/webui/static/css/layout/layout.css cmd/webui/static/css/components/tags.css cmd/webui/static/favicon.svg
git commit -m "refactor(webui): replace hardcoded indigo with accent variables, teal favicon"
```

---

### Task 4: stat-card 镜面扫光（layout.css）

**Files:**
- Modify: `cmd/webui/static/css/layout/layout.css`（`.stat-card` 区块，约 199-227 行）

**Interfaces:**
- Consumes: Task 1 的 `--sweep-color`。
- Produces: `@keyframes specularSweep`（仅本任务使用）。

- [ ] **Step 1: 写扫光样式**

`.stat-card` 现有规则加 `position: relative; overflow: hidden;`，并追加：

```css
.stat-card::after {
  content: '';
  position: absolute; top: -20%; bottom: -20%; left: 0; width: 40%;
  background: linear-gradient(100deg, transparent, var(--sweep-color), transparent);
  transform: translateX(-150%) skewX(-20deg);
  z-index: 0; pointer-events: none;
}
.stat-card:hover::after { animation: specularSweep 0.7s ease; }
.stat-card .num, .stat-card .label { position: relative; z-index: 1; }

@keyframes specularSweep {
  from { transform: translateX(-150%) skewX(-20deg); }
  to   { transform: translateX(350%) skewX(-20deg); }
}
```

注意：只动画 `transform`；`animation` 挂在 `:hover` 上意味着每次进入扫一次、不循环（Review Focus #3）；`.num`/`.label` 的 `z-index:1` 保证光带从文字下方掠过。现有 `prefers-reduced-motion` 全局规则（base.css）自动把动画压到 0.01ms，无需新增。

- [ ] **Step 2: 验证**

```bash
grep -n 'specularSweep\|overflow: hidden' cmd/webui/static/css/layout/layout.css  # 扫光 3 处 + stat-card overflow 命中
grep -n 'z-index: 1' cmd/webui/static/css/layout/layout.css                       # num/label 层级命中
go test ./cmd/webui/...
```

- [ ] **Step 3: Commit**

```bash
git branch --show-current
git add cmd/webui/static/css/layout/layout.css
git commit -m "feat(webui): specular sweep on stat card hover (transform-only)"
```

---

### Task 5: 版本 bump、CHANGELOG、端到端验收

**Files:**
- Modify: `cmd/webui/constant/version.go`
- Modify: `docs/CHANGELOG_webui.md`（置顶新增 `[4.5.0]`）

**Interfaces:**
- Consumes: Task 1-4 的全部产出。
- Produces: 发布的 4.5.0 版本记录。

- [ ] **Step 1: 版本常量**

`const WebuiVersion = "4.4.2"` → `"4.5.0"`。

- [ ] **Step 2: CHANGELOG 条目（置于 `## [4.4.2]` 之前，格式遵守 Global Constraints）**

```markdown
## [4.5.0] - 2026-11-16

### Changed / 变更

- The WebUI theme switched from indigo-purple to teal: light accent `#0d9488`, dark accent `#2dd4bf`, with new background gradients (teal → misty blue → warm sand for light, deep teal-ink for dark), teal-tinted neutral inks and a teal favicon. All previously hardcoded indigo values in `layout.css`, `tags.css` and `base.css` now reference `--accent-*` variables, so future re-theming touches `variables.css` only.
- WebUI 主题色由靛紫整体切换为青碧：亮色强调 `#0d9488`、暗色 `#2dd4bf`，背景渐变改为「青碧→雾蓝→暖沙」（暗色为深青墨），中性色微调为灰青调，favicon 同步换色。`layout.css`、`tags.css`、`base.css` 中硬编码的靛紫全部改引 `--accent-*` 变量，此后换主题只需改 `variables.css`。

### Added / 新增

- Glass surfaces gained the full material treatment per the glassmorphism design spec (`docs/superpowers/specs/2026-11-16-webui-teal-glassmorphism-design.md`): layered radial background blobs so backdrop blur has real content to frost, inset top-highlight/bottom-shade baked into `--glass-shadow`, one global `feTurbulence` grain overlay (opacity 0.045) to kill gradient banding, and a transform-only specular sweep on stat-card hover. Fallbacks: `prefers-reduced-transparency` renders solid panels and hides the grain, `@supports not (backdrop-filter)` raises panel opacity, and the existing reduced-motion rule covers the sweep.
- 按玻璃拟态设计规格（`docs/superpowers/specs/2026-11-16-webui-teal-glassmorphism-design.md`）补齐玻璃材质：背景叠加多层径向色斑让 backdrop blur 有真实内容可霜化；`--glass-shadow` 内置顶部亮边/底部阴影 inset；全局单层 `feTurbulence` 噪点（opacity 0.045）消除渐变色带；stat-card hover 增加仅动画 transform 的镜面扫光。降级：`prefers-reduced-transparency` 下实心面板并隐藏噪点，`@supports not (backdrop-filter)` 下提高面板不透明度，扫光由现有 reduced-motion 规则覆盖。
```

- [ ] **Step 3: 构建与全量测试**

```bash
make build-webui        # 期望产出 tagfile-webui.out，零错误
go test ./...           # 全绿（本改动不触碰 Go 逻辑，任何失败都是回归）
```

- [ ] **Step 4: 起服务，headless Chrome 截图 8 页 × 亮/暗**

```bash
./tagfile-webui.out --source temp/test_workspace --addr :8088 &   # 后台，等 1-2s
mkdir -p /tmp/opencode/shots
for p in "" categories tags files files/recommend review search config; do
  n=${p:-home}
  google-chrome --headless=new --disable-gpu --window-size=1440,900 \
    --screenshot=/tmp/opencode/shots/$n-light.png "http://127.0.0.1:8088/$p"
  google-chrome --headless=new --disable-gpu --window-size=1440,900 \
    --force-prefers-color-scheme=dark \
    --screenshot=/tmp/opencode/shots/$n-dark.png "http://127.0.0.1:8088/$p"
done
kill %1
```

- [ ] **Step 5: 读图验收（Review Focus #1/#2/#4 在此钉死）**

用 `read` 工具逐张查看至少 `home-light.png`、`home-dark.png`、`files-light.png`、`config-dark.png`，确认：

- 无紫色残留（侧栏 active、徽章、按钮、favicon、光晕全为 teal 系）；
- 玻璃面板亮边（顶部 1px 白线）与浮起阴影可见；背景色斑下 blur 可感、无色带；
- 正文与 accent 文字在背景最亮区域清晰可读（对比度目检不达标 → 回 Task 1 调 `--accent-bg`/光晕透明度）；
- 暗色下侧栏 active teal 可读、不刺眼。

再截一张 reduced-motion 验证扫光静止：

```bash
google-chrome --headless=new --disable-gpu --force-prefers-reduced-motion \
  --window-size=1440,900 --screenshot=/tmp/opencode/shots/home-rm.png "http://127.0.0.1:8088/"
```

- [ ] **Step 6: 全仓紫色终检（Review Focus #5 第二道）**

```bash
grep -rEn '6366f1|818cf8|a5b4fc|4f46e5|129, ?140, ?248|99, ?102, ?241|236, ?72, ?153' cmd/ docs/CHANGELOG_webui.md
# 仅允许命中 CHANGELOG 4.5.0 条目里描述旧色的文字（如无则零命中）；static/ 下必须零命中
```

- [ ] **Step 7: Commit**

```bash
git branch --show-current
git add cmd/webui/constant/version.go docs/CHANGELOG_webui.md
git commit -m "chore(webui): bump version to 4.5.0 with teal glassmorphism changelog"
```
