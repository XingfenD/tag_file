# 设计稿：舞台氛围底 + 第三方媒体查看器（2026-11-22）

## 意图与原因

详情页媒体舞台现在永远铺一层写死的暗绿渐变底，图片/视频比例越极端、空白越刺眼。两件事：

1. 舞台底色改为取自内容本身（方向 A，用户已选定），任何比例都不再摊在纯色板上。
2. 引入第三方媒体查看器，点舞台主体可放大查看（缩放/拖拽/旋转；视频带控制条播放）。

## 目标 / 非目标 / 反目标

- 目标：竖图、横条、方图观感一致地「不空」；点开即全屏细看；无构建步骤、离线可用。
- 非目标：查看器内不做同目录上/下一张翻页（用户已明确不要）；文件树行不挂查看器；音频与非媒体不改。
- 反目标：切换文件时布局跳动（否决方向 B 的原因）；查看器与全站 modal/toast 体系抢层级。

## 约束

- 依赖按项目惯例 vendor 进 `cmd/webui/static/vendor/`，无 CDN、无构建（同 Alpine/SortableJS）。
- 不新增后端接口；查看器数据就是 `/api/files/raw/{id}`。
- WebUI 改动 → CHANGELOG_webui 一条短条目 + `WebuiVersion` 同 commit bump。

## 方案

- 已选 A「氛围底」：媒体自身的模糊放大拷贝铺满舞台底层，主体 `contain` 居中浮起。
  - 落选 B「舞台贴合内容」：面板宽度随文件变，切文件布局跳。
  - 落选 C「只换中性底色」：比例问题原样存在。
- 查看器库：PhotoSwipe 5.4.4（MIT、活跃维护、ESM 单文件 vendor + 动态 import；滚轮/触控板捏合缩放、拖拽平移内置；本地视频经 contentLoad 动态内容挂 `<video>`）。
  - 初选 GLightbox 3.3.1 已实施后被替换：其 3.3.1 包内既无 `zoomTo` API 也无关闭事件，触控板 `ctrl+wheel` 捏合不被处理（浏览器拿去缩放整页且无法程序化还原），桌面缩放只能外挂自研 transform 与其 touch 缩放互相打架；库已两年未发版。
  - 落选 lightgallery v2（插件体系重、主题 CSS 与站内设计语言冲突）。

## 设计

1. **氛围底层**（`file_detail.html` + `workbench.css`）
   - 图片：`<img class="stage-ambient">`（同 src，`object-fit: cover`，`filter: blur(32px) saturate(1.25) brightness(0.5)`，`inset: -36px` 外扩吃掉模糊白边，`pointer-events: none`）+ `<img class="stage-media">`（contain + 投影）两层。
   - 视频：ambient 用同一 URL 的 `<video muted playsinline preload="auto">`（首帧模糊底，浏览器共享媒体缓存）；主体仍是带 controls 的 `<video>`。
   - Alpine `template x-if` 只认单根：两层包进 `.stage-layer { display: contents }`，定位仍归 `.stage-view`。
   - 音频 / 不支持格式 / 非媒体：保留现有深色渐变底（无可采样画面）。
2. **查看器**（`static/vendor/photoswipe*` + `pages/file/detail.js`）
   - base.html 挂载 photoswipe.css；PhotoSwipe 模块经动态 `import()` 首开时加载（无构建步骤的 ESM 用法）。
   - `.stage-media`（图片）加 `cursor: zoom-in` 与点击打开；视频用独立 `.stage-expand` 按钮打开（点画面仍是原生播停）。
   - 打开前量媒体像素尺寸（PhotoSwipe 定缩放边界所需）；实例整页复用一个，`loadAndOpen(0, [item])` 每次换数据，切换文件无残留。
   - 图片内置滚轮/捏合缩放 + 拖拽平移；视频在 contentLoad 里挂 `<video controls autoplay playsinline>`。
   - 与工作台就地切换文件兼容：dataSource 每次重设，旧文件 src 不留存。
3. **回归门**（`static_test.go`）
   - `file_detail.html` 含 `stage-ambient`/`stage-media`（图片、视频两处）；`workbench.css` 含 `.stage-layer`/`object-fit: cover`/`object-fit: contain`；vendor 为 photoswipe 三件且 glightbox 不得回流；detail.js 含 `photoswipe-lightbox.esm.min.js`/`loadAndOpen`/`contentLoad`。

## 留给实现者的裁量

- blur/saturate/brightness 具体数值可按观感微调（初值 32px/1.25/0.5）。
- GLightbox 版本号取当前最新稳定版 vendor 即可。
- 版本条目并入本迭代一条（Added/Changed 短句，英文+中文各一行）。
