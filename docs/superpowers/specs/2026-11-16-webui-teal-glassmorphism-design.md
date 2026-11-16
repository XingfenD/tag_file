# WebUI 青碧主题 + 玻璃拟态质感升级 — 设计文档

日期：2026-11-16
状态：待需求方审读

## 1. 意图与背景

WebUI 已有一套玻璃拟态设计系统：`cmd/webui/static/css/core/variables.css` 集中定义亮/暗两套主题变量，各组件 CSS（layout/buttons/tags/table/…）全部通过变量引用。但存在两个问题：

**主题色不受欢迎。** 现行主题色是靛紫（亮色 `#6366f1`、暗色 `#818cf8`），需求方明确表示不好看，要求换色。且紫色有硬编码残留，散落在变量体系之外，单纯改 `variables.css` 换不干净：

| 位置 | 残留 |
|---|---|
| `layout.css:58,63` 侧栏 active | `rgba(129,140,248,…)` ×3 |
| `tags.css:15,24,59,66` 标签徽章 | `rgba(99,102,241,…)` ×4 |
| `base.css:51` 左下环境光晕 | 粉色 `rgba(236,72,153,0.12)` |
| `favicon.svg:4-5` | 渐变 `#818cf8 → #6366f1` |

**玻璃质感不完整。** 对照 glassmorphism 技能的可信玻璃四要素（霜化 / 半透明 / 玻璃亮边 / 分层深度），现状缺：

- 背景太平滑（浅色三段渐变 + 两个小光晕），`backdrop-filter: blur` 采样不到内容，玻璃感出不来；
- 面板只有一层 drop shadow，没有 `inset` 顶部亮边高光和底部阴影，玻璃没有「厚度」；
- 无防色带噪点，大面积重模糊下渐变可能出现 contour banding；
- 无 `prefers-reduced-transparency`、`@supports not (backdrop-filter)` 降级路径；
- 无任何镜面高光动效。

新主题色已由需求方选定：**青碧 Teal**。

## 2. 目标

- 亮/暗两套主题整体切换到 Teal 色系：强调色、背景渐变、环境光晕、侧栏、favicon 全部换色，中性色（ink）从灰紫调微调为灰青调。
- 消灭主题色硬编码：上表残留全部改用 `var(--accent-*)` 变量或新增主题变量，今后换色只动 `variables.css`。
- 补齐玻璃四要素：有内容的背景、inset 亮边高光、防色带噪点、hover 镜面扫光。
- 补齐可访问性降级：`prefers-reduced-transparency` → 实心面板；`@supports not (backdrop-filter)` → 高不透明度背景；`prefers-reduced-motion` 覆盖新动效。
- 性能不劣化：不动画 blur 半径，扫光只动画 `transform`，全局噪点层只有一个。

## 3. 非目标

- 不改任何 Go 逻辑、路由、API；不改 JS 交互逻辑（`theme.js` 等零改动）。
- 不引入 CSS 框架、预处理器或构建步骤，保持手写 CSS + `@layer` 现有组织。
- 不做 liquid glass SVG 折射（`feDisplacementMap`）hero——那是营销页效果，管理工具性能优先。
- 不重构组件 CSS 结构与模板 DOM（唯一例外：`base.html` 新增一个噪点层 `div`）。
- 不做第三套主题或运行时取色器。

## 4. 反目标（做成了也算失败）

- 换色后任何页面仍有紫色残留（含 favicon）。
- 文字对比度跌破 WCAG：正文 ≥ 4.5:1、大号字 ≥ 3:1，按背景最亮处校验。
- 噪点层肉眼可辨为「脏点」、拦截点击、或在 reduced-transparency 下仍出现。
- 滚动/交互可感知掉帧（动画了 blur 半径、扫光循环播放、噪点层过多）。
- `prefers-reduced-transparency` 生效时面板仍半透明。

## 5. 已确认的决策（需求方拍板）

| 决策点 | 结论 |
|---|---|
| 主题色方向 | 青碧 Teal（需求方从 Teal/Sky/Emerald/Amber 中选定） |
| 实现方式 | 沿用现有 CSS 变量体系，换变量 + 补质感，不重构 |
| liquid glass 折射 | 不做 |

## 6. 设计细节

### 6.1 Teal 色板（`variables.css`）

**亮色主题（`:root`）：**

| 变量 | 现值 | 新值 |
|---|---|---|
| `--accent` | `#6366f1` | `#0d9488`（teal-600） |
| `--accent-dim` | `#4f46e5` | `#0f766e`（teal-700） |
| `--accent-bg` | `rgba(99,102,241,0.1)` | `rgba(13,148,136,0.10)` |
| `--accent-glow` | `rgba(99,102,241,0.25)` | `rgba(13,148,136,0.25)` |
| `--bg-gradient` | 蓝→紫→粉 | `linear-gradient(135deg, #dff2ee 0%, #e2edf6 50%, #f6efe1 100%)`（青碧→雾蓝→暖沙） |
| `--sidebar-active` | `#818cf8` | `#2dd4bf`（teal-400，深底上够亮） |
| `--ink` / `--ink-light` / `--ink-muted` | 灰紫调 | `#182422` / `#475855` / `#7e918d`（灰青调） |

**暗色主题（`[data-theme="dark"]`）：**

| 变量 | 现值 | 新值 |
|---|---|---|
| `--accent` | `#818cf8` | `#2dd4bf`（teal-400） |
| `--accent-dim` | `#6366f1` | `#14b8a6`（teal-500） |
| `--accent-bg` | `rgba(129,140,248,0.12)` | `rgba(45,212,191,0.12)` |
| `--accent-glow` | `rgba(129,140,248,0.2)` | `rgba(45,212,191,0.18)` |
| `--bg-gradient` | 深蓝紫 | `linear-gradient(135deg, #0a1416 0%, #0d2129 50%, #101a22 100%)`（深青墨） |
| `--sidebar-active` | `#a5b4fc` | `#5eead4`（teal-300） |
| `--ink` / `--ink-light` / `--ink-muted` | 灰紫调 | `#e6f0ee` / `#aac2bc` / `#66837d`（灰青调） |

新增两个主题变量，供 `base.css` 环境光晕引用（消灭粉色硬编码）：

- `--glow-a`：主光晕，亮 `var(--accent-glow)` 级 teal，暗同；
- `--glow-b`：副光晕，亮色用暖沙 `rgba(214,158,86,0.12)`（teal 的暖互补，避免全冷），暗色用 `rgba(45,212,191,0.07)`。

danger/success/warning 语义色不动（红绿橙与 teal 不冲突）。

### 6.2 硬编码清理

| 文件 | 改法 |
|---|---|
| `layout.css` 侧栏 active | `rgba(129,140,248,0.12)` → `var(--accent-bg)`；box-shadow 两处 → `var(--accent-glow)` 派生 |
| `tags.css` ×4 | `rgba(99,102,241,0.12/0.18)` → `var(--accent-bg)` / `var(--accent-glow)` |
| `base.css:36,51` 两光晕 | 分别改引 `var(--glow-a)` / `var(--glow-b)` |
| `favicon.svg` | 渐变改 `#2dd4bf → #0d9488` |

清理完成后 `grep -rE '6366f1|818cf8|a5b4fc|4f46e5|129,140,248|99,102,241|236,72,153' cmd/webui` 应为零命中。

### 6.3 玻璃质感升级

**(a) 有内容的背景。** 新增主题变量 `--bg-blobs`（亮暗各定义一份）：多层 `radial-gradient` 色斑（teal / 雾蓝 / 暖沙，直径 40–60vw，透明度 0.10–0.18），`body` 的 `background` 简写中叠加在 `--bg-gradient` 之上，同为 fixed 层。两个光晕 blob（`body::before/::after`）从 500/400px 放大到 640/520px。blur 从此采样得到真实色彩变化，玻璃感立现。

**(b) 亮边高光（inset）。** 直接改 `--glass-shadow` / `--glass-shadow-hover` 变量为复合阴影，所有引用处（card / stat-card / dir-sidebar / scan-result / btn-default / …）自动获得，无需逐文件改：

```
亮色 --glass-shadow:
  0 8px 32px rgba(24,54,50,0.08),          /* float */
  inset 0 1px 0 rgba(255,255,255,0.50),    /* 顶部亮边 */
  inset 0 -1px 0 rgba(24,54,50,0.06);      /* 底部阴影 */
暗色 inset 高光减弱为 rgba(255,255,255,0.10) / 0.05。
```

`--glass-border` 微调更亮（亮色 `rgba(255,255,255,0.55)`），保持「边框比填充亮」。

**(c) 防色带噪点。** `base.html` 在 `<body>` 首个子元素位置加 `<div class="grain" aria-hidden="true"></div>`；样式进 `base.css`：`position:fixed; inset:0; pointer-events:none; z-index:10000`（现有层级为 toast 9999 / modal 1000 / dropdown 100，噪点取最顶层统一覆盖，`pointer-events:none` 保证不拦截任何点击），背景为内联 SVG `feTurbulence` data-URI（`fractalNoise`，`baseFrequency=0.85`，`numOctaves=2`），`opacity:0.045`。全局仅此一层，不给每个面板挂 `::after`。

**(d) 镜面扫光。** `.stat-card` 加 `position:relative; overflow:hidden` + `::after` 伪元素：一条 `skewX(-20deg)` 的白色渐变光带，默认 `translateX(-150%)`，hover 时 `transform` 扫到 `translateX(150%)`，`0.7s ease`，单次不循环，只动画 `transform`（合成层，不触发重排/重绘 blur）。层级：`::after { z-index:0 }`，内容 `.num` / `.label` 加 `position:relative; z-index:1`，光带从文字下方掠过不遮挡。

### 6.4 降级与可访问性（`base.css` 末尾）

```css
@media (prefers-reduced-transparency: reduce) {
  /* 玻璃变量重定义为实心：--glass-bg → #fff（暗 #1c2a28），
     --glass-bg-strong 同理；所有 backdrop-filter 类统一 none；
     .grain 隐藏；阴影保留（不依赖透明度） */
}
@supports not ((backdrop-filter: blur(1px)) or (-webkit-backdrop-filter: blur(1px))) {
  /* --glass-bg → rgba(255,255,255,0.88)；暗 → rgba(23,37,35,0.92)，足够不透明兜底 */
}
```

现有 `prefers-reduced-motion` 全局规则（`animation-duration:0.01ms`）自动覆盖扫光，无需新增；补充确认光晕/色斑不受影响即可。

对比度验收点：`--accent` 亮色 `#0d9488` 对近白玻璃底 ≈ 4.5:1（达标）；stat 数字（40px 大号字）≥ 3:1 宽松；暗色 `#2dd4bf` 对深底 > 7:1。

## 7. 实施与验收

**文件清单**（按改动顺序）：

1. `cmd/webui/static/css/core/variables.css` — 色板 + inset 阴影变量 + glow 变量
2. `cmd/webui/static/css/core/base.css` — 背景色斑、光晕改引变量、`.grain`、降级块
3. `cmd/webui/static/css/layout/layout.css` — 侧栏 active 去硬编码、stat-card 扫光
4. `cmd/webui/static/css/components/tags.css` — 去硬编码
5. `cmd/webui/static/favicon.svg` — teal 渐变
6. `cmd/webui/templates/base.html` — grain div
7. `docs/CHANGELOG_webui.md` + `cmd/webui/constant/version.go` — 版本 `4.4.2 → 4.5.0`（视觉主题级变更，minor），同一 commit

**验收标准：**

1. `go run ./cmd/webui --source <dir> --addr :8088` 启动，逐一打开 8 个页面（首页/分类/标签/文件/随机打标/待审/搜索/配置），亮暗两套主题下：无紫色残留、玻璃亮边可见、背景色斑下 blur 可感、无 banding。
2. `grep` 校验硬编码零命中（见 6.2）。
3. stat-card hover 扫光顺滑、不循环；页面滚动无可感知掉帧。
4. 浏览器模拟 `prefers-reduced-transparency: reduce` → 面板实心可读；`prefers-reduced-motion` → 无扫光动画。
5. `make build-webui` 编译通过；`go test ./cmd/webui/...` 不受影响（模板仅加一个静态 div）。

**分支约定：** 当前在 `master`，不可直接提交。需新建分支 `feat/webui-teal-glass`（含本设计文档的提交）。
