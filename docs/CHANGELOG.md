# Changelog / 更新日志

All notable changes to this template should be documented in this file.
本模板的重要变更建议统一记录在此文件中。

The format loosely follows Keep a Changelog and can be adapted to the team's habits.
本文档参考了 Keep a Changelog 的思路，也可以根据团队习惯调整。

## [0.0.6] - 2026-05-03

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

## [0.0.5] - 2026-05-01

### Added / 新增

- Added `file exists <path>` command to check if a file is indexed.
- 新增 `file exists <路径>` 命令，判断文件是否已索引。

- Added `file diff [-r]` command: list unindexed files, `-r` for recursive tree view.
- 新增 `file diff [-r]` 命令：列出未索引文件，`-r` 递归树形展示。

- Added `file scan --full` flag for forced full scan.
- 新增 `file scan --full` 强制全量扫描。

### Changed / 变更

- Moved database from `.tag_file.db` to `.tagfile/tagfile.db`. Scan state stored in `.tagfile/last_scan`.
- 数据库从 `.tag_file.db` 移至 `.tagfile/tagfile.db`。扫描状态写入 `.tagfile/last_scan`。

### Fixed / 修复

- File scan filter now case-insensitive (config `mp4` matches `.MP4`).
- 文件扫描过滤器无视大小写（配置 `mp4` 可匹配 `.MP4`）。

### Performance / 性能

- Eliminated N+1 SQL in file scan: load all indexed paths into in-memory map with single query.
- 消除文件扫描中的 N+1 SQL：单次查询全部索引路径到内存 map。

- Batch insert new files instead of per-file INSERT.
- 批量写入新文件替代逐条 INSERT。

- `filepath.Walk` → `filepath.WalkDir` (avoids per-file `os.Stat`).
- `filepath.Walk` 替换为 `filepath.WalkDir`（免 per-file Stat 调用）。

- Removed redundant `os.Stat` in scan — reuse `DirEntry.Info()`.
- 移除扫描中的冗余 `os.Stat`，复用 `DirEntry.Info()`。

## [0.0.4] - 2026-05-01

### Added / 新增

- Added `AttrRepository` and `AttrService` for managing tag attributes and attribute values.
- 新增 `AttrRepository` 和 `AttrService`，用于管理标签属性定义和属性值。

- Added `ConfigRepository` and `ConfigService` with upsert/get/list/delete operations.
- 新增 `ConfigRepository` 和 `ConfigService`，支持配置项的 upsert/get/list/delete。

- Added CLI command `tagfile attribute {add, remove, list}` for attribute management.
- 新增 CLI 命令 `tagfile attribute {add, remove, list}`，用于属性管理。

- Added CLI command `tagfile config {set, get, list, unset}` for configuration management.
- 新增 CLI 命令 `tagfile config {set, get, list, unset}`，用于配置项管理。

- Added `cmd/test/attr_config.go` with 17 new tests for attribute and config layers.
- 新增 `cmd/test/attr_config.go`，包含 17 项属性和配置层测试。

### Changed / 变更

- Simplified `ImportFile` to copy files to CWD instead of a separate `imports/` subdirectory.
- 简化 `ImportFile`，将外部文件直接复制到当前目录，不再使用 `imports/` 子目录。

- Updated `setup_test_workspace.sh` and integration tests to reflect new import behavior.
- 更新 `setup_test_workspace.sh` 和集成测试以匹配新的导入行为。

- Updated `AGENTS.md` with Phase 3 completion and accurate project status.
- 更新 `AGENTS.md`，标记 Phase 3 完成并同步项目状态。

## [0.0.3] - 2026-05-01

### Added / 新增

- Implemented CLI command `tagfile file {scan, import, list, tag, untag, info, search}` for file management.
- 实现 CLI 命令 `tagfile file {scan, import, list, tag, untag, info, search}`，用于文件管理。

- Implemented CLI command `tagfile search tag <tags> --and/--or/--not` for tag-based file search.
- 实现 CLI 命令 `tagfile search tag <tags> --and/--or/--not`，支持 AND/OR/NOT 组合搜索。

- Added `FileRepository` with file CRUD, tag association, and tag-based search queries.
- 新增 `FileRepository`，包含文件 CRUD、标签关联和按标签搜索。

- Added `FileService` for directory scanning, file importing, tagging, and untagging.
- 新增 `FileService`，支持目录扫描、外部文件导入、打标和取消标签。

- Added `SearchService` for tag-based file search with AND/OR/NOT logic.
- 新增 `SearchService`，支持 AND/OR/NOT 逻辑的标签搜索。

- Added `scripts/setup_test_workspace.sh` to create a demo workspace with test files and tags.
- 新增 `scripts/setup_test_workspace.sh`，一键创建包含测试文件和标签的演示工作区。

- Added `cmd/test/unit.go` and `cmd/test/integration.go`, split test binary into modular files.
- 新增 `cmd/test/unit.go` 和 `cmd/test/integration.go`，将测试拆分为模块化文件。

- Added `Makefile setup` target for workspace creation, `test` target depends on `setup`.
- 新增 `Makefile` 的 `setup` 目标和 `test` → `setup` 依赖。

### Changed / 变更

- Updated `Makefile` test target to depend on `setup` and use the shared workspace.
- 更新 `Makefile`，`test` 目标依赖 `setup`，使用共享工作区。

- Updated `.gitignore` to exclude `imports/` and `temp/` directories.
- 更新 `.gitignore`，排除 `imports/` 和 `temp/` 目录。

- Updated `AGENTS.md` with Phase 2 completion and updated command documentation.
- 更新 `AGENTS.md`，标记 Phase 2 完成并更新命令文档。

### Fixed / 修复

- Fixed `GetTags` in `FileRepository` to preload `Category` association for proper tag display.
- 修复 `FileRepository.GetTags`，正确预加载 `Category` 关联以显示分类项名称。

- Fixed `ScanDirectory` to properly skip dot-prefixed files and directories.
- 修复 `ScanDirectory`，正确跳过点号前缀的文件和目录。

## [0.0.2] - 2026-05-01

### Added / 新增

- Implemented CLI command `tagfile category {add, remove, list}` for managing tag categories.
- 实现 CLI 命令 `tagfile category {add, remove, list}`，用于管理标签分类项。

- Implemented CLI command `tagfile tag {add, remove, list}` with real database-backed logic.
- 实现 CLI 命令 `tagfile tag {add, remove, list}`，替换占位代码为真实数据库操作。

- Added `CategoryRepository` and `TagRepository` for data access (CRUD).
- 新增 `CategoryRepository` 和 `TagRepository` 数据访问层。

- Added `CategoryService` and `TagService` with uniqueness validation (category names globally unique, tag names unique within a category).
- 新增 `CategoryService` 和 `TagService` 业务逻辑层，包含唯一性校验（分类项名称全局唯一，同分类下标签名唯一）。

- Added `cmd/test/main.go` as a standalone test entry point covering repository and service layers.
- 新增 `cmd/test/main.go` 独立测试入口，覆盖 repository 和 service 层。

- Added `AGENTS.md` with project plan, architecture notes, and development conventions.
- 新增 `AGENTS.md`，包含项目规划、架构说明和开发约定。

### Changed / 变更

- Muted GORM SQL debug logging for cleaner CLI output.
- GORM SQL 调试日志改为静默模式，优化命令行输出。

- Updated `.gitignore` to exclude additional IDE and tool directories.
- 更新 `.gitignore`，排除更多 IDE 和工具目录。

- Updated `Makefile` with `test-run` target.
- 更新 `Makefile`，新增 `test-run` 目标。

## [0.0.1] - 2026-04-29

### Added / 新增

- Added an English project-template README in `docs/README.md`.
- 在 `docs/README.md` 中补充了英文版项目模板说明。

- Added a Chinese project-template README in `docs/README_zh.md`.
- 在 `docs/README_zh.md` 中补充了中文版项目模板说明。

- Added this changelog file for future template maintenance.
- 新增本更新日志文件，便于后续持续维护模板。

### Notes / 说明

- The repository currently provides structure and placeholder files rather than a finished implementation.
- 当前仓库主要提供目录结构和占位文件，尚不是一个已完成功能实现的成品项目。

- Future updates should record framework selection, startup steps, deployment workflow, and major documentation changes.
- 后续若补充了技术栈、启动流程、部署方式或重要文档内容，建议继续记录在本文件中。
