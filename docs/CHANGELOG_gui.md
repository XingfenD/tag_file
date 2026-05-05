# Changelog / 更新日志 — GUI

tagfile-gui 桌面应用的版本变更记录。

## [0.1.0] - 2026-05-05

### Added / 新增

- GUI desktop application (`cmd/gui/`): Wails v2 + WebView2, reuses all `internal/` services.
- 新增桌面 GUI 应用（`cmd/gui/`）：基于 Wails v2 + WebView2，复用全部 `internal/` 服务。

- VLC external player integration for wide video format support (MKV, AVI, MOV, HEVC, etc.) without transcoding.
- VLC 外部播放器集成，支持无转码播放广泛视频格式（MKV, AVI, MOV, HEVC 等）。

- HTML5 video fallback for browser-playable formats when VLC is not installed.
- VLC 未安装时回退到 HTML5 视频播放。

- SPA-style frontend: same 8 pages as webui (home, category, tag, file, file detail, file groups, search, config).
- SPA 风格前端：与 webui 相同的 8 个页面（首页、分类、标签、文件、文件详情、标签组、搜索、配置）。

- Makefile targets: `build-gui`, `run-gui`, `build-gui-prod`.
- Makefile 新增 `build-gui`、`run-gui`、`build-gui-prod` 目标。
