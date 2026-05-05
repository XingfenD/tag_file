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

- `--source` flag to specify working directory.
- `--source` 参数支持指定工作目录。

- Performance logging: key operations log timing when > 50ms.
- 性能日志：关键操作超过 50ms 时自动打印耗时。

### Fixed / 修复

- Data serialization: all model types now converted to maps with consistent JS-friendly field names, fixing undefined IDs and edit errors.
- 数据序列化：所有模型类型统一转换为 JS 友好的 map，修复 ID 显示为 undefined 和编辑报错问题。

- Media streaming via local HTTP server instead of loading files into memory, preventing OOM crash on large files.
- 媒体文件通过本地 HTTP 服务器流式传输，不再将大文件读入内存，避免 OOM 崩溃。

- File list N+1 query: added 5-second in-memory cache, reducing repeated `ListFiles` calls from 4 to 1 per page load.
- 文件列表 N+1 查询：新增 5 秒内存缓存，将页面加载时重复的 `ListFiles` 调用从 4 次减少到 1 次。
