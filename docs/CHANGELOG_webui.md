# Changelog (Web UI) / 更新日志 (Web UI)

tagfile 项目 Web 界面的版本变更记录。格式参考 [Keep a Changelog](https://keepachangelog.com)。

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
