# Changelog / 更新日志

tagfile 项目的版本变更记录。格式参考 [Keep a Changelog](https://keepachangelog.com)，版本号遵循 [Semantic Versioning](https://semver.org)。

## [0.3.0] - 2026-05-04

### Changed / 变更

- Repository layer refactored to chainable builder pattern: `Create()`, `Find()`, `List()`, `Update()`, `Delete()`, `Search()` with one file per builder.
- Repository 层重构为链式 Builder 模式，每个 Builder 一个文件。

### Added / 新增

- `Update(model).SetXxx(v).SetYyy(v).Exec()` for selective field updates via `Select()`.
- `Update(model).SetXxx(v).Exec()` 选择性字段更新。

## [0.3.1] - 2026-05-04

### Added / 新增

- `File.TagCnt` field: automatic maintenance on tag add/remove via `IncrementTagCnt` in service layer.
- `File.TagCnt` 冗余字段：service 层在打标/去标时自动增减，repo 层提供 `IncrementTagCnt` 原子操作。

## [0.2.0] - 2026-05-04

### Added / 新增

- `TagGroup` / `TagGroupTag` models: files have tag groups, tags belong to groups. Default group auto-created on scan/import.
- 新增标签组模型：文件通过标签组关联标签，扫描/导入时自动创建默认组。

- `TagGroupTag` redundant `file_id` column avoids JOIN to `tag_groups` for search and listing.
- `TagGroupTag` 冗余 `file_id`，搜索和列表无需 JOIN `tag_groups`。

- `file group {add,remove,list,tag,untag}` CLI commands for multi-group management.
- `file group` CLI 命令支持多标签组管理。

### Removed / 移除

- `FileTag` model removed; all tag-file associations migrated to `TagGroupTag`.
- 移除 `FileTag` 模型，标签关联迁移至 `TagGroupTag`。

## [0.1.2] - 2026-05-04

### Added / 新增

- `ListFileTags` with pagination returns `FileTag` rows including `tag_value`, Preload `Tag.Category`.
- `ListFileTags` 分页返回 `FileTag` 行（含 `tag_value`），Preload `Tag.Category`。

- `file tags <path>` CLI command with `--limit/--offset`.
- 新增 `file tags <路径>` CLI 命令。

## [0.1.1] - 2026-05-04

### Added / 新增

- `tag update <ID>` command with `--name`, `--desc`, `--parent` flags; cycle detection prevents setting descendant as parent.
- 新增 `tag update <ID>` 命令，支持 `--name`、`--desc`、`--parent`；含循环检测防止子孙设为父标签。

- Tag value system: `TagValue` model, `tag value {add,remove,list}`, `file tag --value`, display as `标签:值`.
- 标签值系统：`TagValue` 模型，`tag value {add,remove,list}`，`file tag --value`，展示为 `标签:值`。

- Regex-based scan/diff exclusion patterns loaded from `scan_exclude` Config; defaults seeded on DB init (`$RECYCLE.BIN`, `System Volume Information`, dot-files).
- 扫描排除规则改为正则，存储在 `scan_exclude` 配置项；DB 初始化时写入默认值。

- Error logging added to all service layer methods via `log.Printf`.
- 所有 service 层方法新增错误日志。

### Changed / 变更

- Tag name uniqueness scoped to `(parent_id, tag_name)` within category. `FindByParentNameAndCategory` added.
- 标签唯一性改为 `(parent_id, tag_name)` 维度。

- File tag uniqueness at `(tag_id, tag_value)` level; same tag with different values can coexist on one file.
- 文件标记改为 `(tag_id, tag_value)` 维度唯一。

## [0.1.0] - 2026-05-04

### Removed / 移除

- Removed the entire attribute system: `Attribute`, `AttributeValue`, `FileAttribute` models, repo, service, and CLI commands.
- 移除属性系统：`Attribute`、`AttributeValue`、`FileAttribute` 模型、仓库、服务及 CLI 命令。

### Added / 新增

- Tree-structured tags: `Tag.ParentID` self-referencing FK, `--parent` flag in `tag add`, `--tree` flag in `tag list`.
- 标签支持树形结构：`Tag.ParentID` 自引外键，`tag add --parent`，`tag list --tree`。

- Search auto-expands parent tags to include all descendant tags (AND groups by tag family, OR flattens).
- 搜索父标签时自动包含所有子孙标签（AND 按标签族分组，OR 平铺）。

- `ListTagTree`, `GetDescendantIDs` in tag service; `SearchByTagGroups` in file repo.
- tag service 新增 `ListTagTree`、`GetDescendantIDs`；file repo 新增 `SearchByTagGroups`。

### Changed / 变更

- Merged `"具体品种"` category into `"动物类型"`; `"风景"` tag moved to `"场景内容"`.
- `"具体品种"` 分类合并入 `"动物类型"`；`"风景"` 标签移入 `"场景内容"`。

- Updated setup script, test fixtures, and documentation to reflect tree structure.
- 更新测试工作区脚本、测试数据和文档。

## [0.0.7] - 2026-05-04

### Changed / 变更

- Refactored `internal/service/` and `internal/repository/` into domain-based subdirectories, with large files split by concern.
- 重构 `internal/service/` 和 `internal/repository/` 为按领域划分子目录，大文件按职责拆分。

  - `internal/service/file/`: service, scan, import, list, tag, diff
  - `internal/service/attr/`: service, attribute, value
  - `internal/service/tag/`, `category/`, `config/`, `search/`
  - `internal/repository/file/`: repo, tag, search
  - `internal/repository/attr/`: repo, attribute, value
  - `internal/repository/tag/`, `category/`, `config/`

### Added / 新增

- Update methods for all entities (File, Tag, Category, Config, Attribute, AttributeValue) with uniqueness validation at service layer.
- 为所有实体新增更新方法，service 层带唯一性校验。

- Fuzzy search (LIKE) for all entities: `SearchFiles`, `SearchTags`, `SearchCategories`, `SearchConfigs`, `SearchAttributes`, `SearchValues`.
- 所有实体支持模糊查询。

- File rename: `RenameFile` renames on disk and updates DB path atomically.
- 文件重命名：磁盘重命名与数据库路径原子更新。

### Fixed / 修复

- `CreateBatch` now splits inserts into chunks of 100 to avoid SQLite's max 999 host parameters limit.
- `CreateBatch` 按每批 100 条分片写入，避免超出 SQLite 最大 999 个 host 参数限制。

## [0.0.6] - 2026-05-03

### Changed / 变更

- Split `.tagfile/last_scan` into `.tagfile/scan_state` (scan metadata) and `.tagfile/diff_state` (diff results), no longer sharing the same file.
- `.tagfile/last_scan` 拆分为 `.tagfile/scan_state`（扫描元数据）和 `.tagfile/diff_state`（差异结果），不再共用同一文件。

- `file scan` cache invalidation: auto-full scan when root directory mtime > last scan time (directory structure changed).
- `file scan` 缓存失效：根目录 mtime > 上次扫描时间时自动全量扫描（检测到目录结构变化）。

- Reorganized CLI commands into subdirectories: `cmd/cli/commands/{category,tag,file,search,attribute,config,util}/`.
- CLI 命令按子目录重组：`cmd/cli/commands/{category,tag,file,search,attribute,config,util}/`。

- Moved shared helpers `FormatSize` / `Truncate` into `cmd/cli/commands/util/` package.
- 共享工具函数 `FormatSize` / `Truncate` 移至 `cmd/cli/commands/util/` 包。

### Fixed / 修复

- Removed unused scan filter code (`getScanFilters` / `passesFilter`) from service layer.
- 移除 service 层未使用的扫描过滤器代码。

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
