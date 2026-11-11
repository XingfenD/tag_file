# Changelog / 更新日志

tagfile 项目的版本变更记录。格式参考 [Keep a Changelog](https://keepachangelog.com)，版本号遵循 [Semantic Versioning](https://semver.org)。

## [0.5.7] - 2026-11-12

### Fixed / 修复

- `ScanDirectory` and `DiffUnindexed` index files by their path relative to the working directory — the basis the tree, detail, expand and raw readers all resolve against. Scanning a subdirectory used to store absolute or `../`-prefixed paths, which duplicated rows for the same file and left that subdirectory invisible in the tree.
- `ScanDirectory`、`DiffUnindexed` 按相对于工作目录的路径索引文件——文件树、详情页、展开、原始读取都以这个基准解析。此前扫描子目录会存入绝对路径或 `../` 前缀路径，同一文件产生重复行，该子目录在树中也看不到。
- Directories outside the working directory are rejected instead of indexed: a stored `../..` path resolves for no reader, so indexing it would poison every lookup.
- 工作目录之外的目录直接报错而不再索引：存成 `../..` 的路径任何读取方都解析不了，索引它等于污染全部查询。
- Search conditions that cannot be built — unknown `type`/`op`, unparsable date, empty extension list — now return an error instead of being dropped, which used to turn a mistyped filter into a query returning every file. A tag name matching no tag now narrows the result to nothing (`has`) or leaves it unconstrained (`not_has`) instead of deleting the condition.
- 无法构建的搜索条件——未知 `type`/`op`、日期解析失败、扩展名列表为空——现在返回错误而不再被丢弃，此前一个拼错的筛选条件会退化成返回全部结果。匹配不到任何标签的标签名现在 `has` 时收敛为空集、`not_has` 时不设约束，而不是整条条件消失。
- Deleting a tag group decrements `File.TagCnt` by the number of associations it held, so the counter no longer stays above the real tag count.
- 删除标签组现在按其承载的关联数递减 `File.TagCnt`，计数不再长期高于真实标签数。
- Deleting a tag now clears its `tag_group_tags` and `tag_values` rows, re-parents children to the root, and decrements `File.TagCnt` for every affected file, in one transaction. The database runs with `PRAGMA foreign_keys = 0`, so nothing cascaded before.
- 删除标签现在会在单个事务内清除其 `tag_group_tags`、`tag_values` 关联行、把子标签提升为根标签，并递减每个受影响文件的 `File.TagCnt`。数据库以 `PRAGMA foreign_keys = 0` 运行，此前一切都不级联。
- `TagFileToGroup` and `UntagFileFromGroup` reject a group belonging to another file, matching the ownership check `DeleteGroup` and `RenameGroup` already made. Tagging through a foreign group used to write a cross-file association and inflate the wrong file's counter.
- `TagFileToGroup`、`UntagFileFromGroup` 现在拒绝属于其他文件的标签组，与 `DeleteGroup`、`RenameGroup` 原有的归属校验一致。此前通过外部标签组打分会写入跨文件关联，并抬高错误文件的计数。
- `RenameFile` rejects a new name containing a path separator, which used to move the file out of its indexed directory and leave the database pointing at a path that no longer exists.
- `RenameFile` 拒绝包含路径分隔符的新文件名，此前它会把文件移出被索引的目录，让数据库指向一个已不存在的路径。
- `ImportFile` creates the default tag group for files it copies into the workspace, as the in-workspace registration path already did; those files had no group, so the tagging UI had nowhere to write.
- `ImportFile` 现在为复制进工作目录的文件创建默认标签组，与工作目录内注册的路径保持一致；此前这类文件没有标签组，打标界面无处可写。

### Added / 新增

- Regression tests covering the scan/diff path basis, search condition building, tag and group deletion cascades, group ownership, and rename constraints.
- 新增回归测试，覆盖扫描/差异比对的路径基准、搜索条件构建、标签与标签组删除级联、标签组归属、重命名约束。

## [0.5.6] - 2026-08-29

### Changed / 变更

- Added database indexes on `files.file_path`, `files.file_name`, `tags.tag_name`, `tags.category_id`, `tags.parent_id`, and `tag_group_tags.tag_id` for significantly faster queries.
- 新增 `files.file_path`、`files.file_name`、`tags.tag_name`、`tags.category_id`、`tags.parent_id`、`tag_group_tags.tag_id` 数据库索引，显著提升查询性能。
- File tree building optimized from O(n²) to O(n) using directory path map lookup instead of linear children scan.
- 文件树构建从 O(n²) 优化为 O(n)，使用目录路径 map 查找替代线性遍历子节点。
- Search tag descendant resolution batched from N individual recursive CTEs into a single query via `GetDescendantIDsMultiMap`.
- 搜索标签子孙解析从 N 次独立递归 CTE 合并为单次批量查询（`GetDescendantIDsMultiMap`）。
- `ImportFile` now uses single `Find().ByPath()` lookup instead of loading all indexed paths into memory.
- `ImportFile` 改用单次 `Find().ByPath()` 查询，不再将所有已索引路径加载到内存。
- `loadTagsForFiles` inner loop fixed from O(n×m) to O(n) using direct map lookup.
- `loadTagsForFiles` 内层循环从 O(n×m) 修复为 O(n)，使用直接 map 查找。

### Added / 新增

- `tag.Repository.FindByIDs(ids)` for batch tag lookup by IDs.
- `tag.Repository.FindByIDs(ids)` 支持按 ID 批量查询标签。
- `tag.Repository.GetDescendantIDsMultiMap(tagIDs)` returns per-root descendant ID map in a single recursive CTE.
- `tag.Repository.GetDescendantIDsMultiMap(tagIDs)` 通过单次递归 CTE 返回按根标签分组的子孙 ID map。
- `search.Service.SearchByQueryPaged(query, offset, limit)` adds SQL-level pagination to structured search.
- `search.Service.SearchByQueryPaged(query, offset, limit)` 为结构化搜索增加 SQL 层面分页。

### Fixed / 修复

- `tag.FindOperation.Exec()` no longer panics with nil pointer dereference when `ByID()` was not called.
- 修复 `tag.FindOperation.Exec()` 在未调用 `ByID()` 时的空指针 panic。

## [0.5.5] - 2026-08-29

### Added / 新增

- `file.Service.BuildTreeFromDB()` builds the file tree structure from the database without filesystem scanning. Returns `[]*TreeNode` with directory hierarchy inferred from file paths.
- `file.Service.BuildTreeFromDB()` 从数据库构建文件树结构，无需扫描文件系统。返回 `[]*TreeNode`，目录层级从文件路径推断。
- `file.Service.ExpandDir(dir)` returns indexed and unindexed direct children of a directory using a single-level `os.ReadDir` combined with DB query.
- `file.Service.ExpandDir(dir)` 返回指定目录的已索引和未索引直接子项，使用单层 `os.ReadDir` 结合数据库查询。

## [0.5.4] - 2026-08-16

### Added / 新增

- DB protection mechanism: core version is recorded in `.tagfile/core_version`. On startup, if the running version differs from the recorded version, the SQLite DB is automatically backed up to `tagfile.db.bak.<timestamp>` before migration.
- 数据库保护机制：核心版本号记录在 `.tagfile/core_version` 中。启动时如果运行版本与记录版本不同，会在迁移前自动将 SQLite 数据库备份为 `tagfile.db.bak.<时间戳>`。
- Version constants centralized: `internal/constant.CoreVersion` (core) and `cmd/webui/constant.WebuiVersion` (webui). Both CLI and WebUI import from these packages.
- 版本常量集中管理：`internal/constant.CoreVersion`（core）和 `cmd/webui/constant.WebuiVersion`（webui）。CLI 和 WebUI 均从这两个包导入。

## [0.5.3] - 2026-08-16

### Fixed / 修复

- `tag.UpdateTag` 跨分类移动标签时报错不明确（"父标签不存在"）：改为返回明确的跨分类错误提示（"不能跨分类移动标签"），并显示父标签和当前标签各自的分类名称。
- `tag.UpdateTag` now returns a clear cross-category error message ("cannot move tag across categories") instead of "parent tag not found" when attempting to set a parent tag from a different category.
- 修复跨分类错误信息中当前标签分类名称显示为空的问题：`Find().Exec()` 方法现在正确应用预加载（Preload）。
- Fixed empty category name for current tag in cross-category error messages: `Find().Exec()` now correctly applies preloads.

## [0.5.2] - 2026-08-16

### Fixed / 修复

- `tag.UpdateTag` could not move a tag back to root level (empty `parent_tag_name` meant "unchanged"). It now takes a pointer (`nil` = unchanged, empty string = move to root, name = re-parent); `tag update` CLI passes it only when `--parent` is set; `ByParentAndName(nil, …)` now filters `parent_id IS NULL` so root-level duplicate checks work. WebUI `PUT /api/tags/{id}` treats a present-but-empty `parent_tag_name` as an explicit move to root.
- `tag.UpdateTag` 无法把标签移回根层（空 `parent_tag_name` 被当作"不变"）。现改为指针语义（`nil`=不变、空串=移到根层、名称=改父级）；CLI `tag update` 仅在显式传入 `--parent` 时生效；`ByParentAndName(nil, …)` 现在按 `parent_id IS NULL` 过滤，根层重名检查可用。WebUI `PUT /api/tags/{id}` 将"字段存在但为空"视为显式移到根层。

## [0.5.1] - 2026-08-16

### Fixed / 修复

- `file.CreateGroup` treated gorm's `ErrRecordNotFound` from `FindGroupByName` as fatal, so creating a new tag group always failed with "查询标签组失败: record not found". Now follows the same "not found = no existing group" convention already used by `RenameGroup`.
- `file.CreateGroup` 将 `FindGroupByName` 返回的 gorm `ErrRecordNotFound` 误判为致命错误，导致创建新标签组始终失败（"查询标签组失败: record not found"）。现与 `RenameGroup` 采用相同约定：未找到即视为不存在同名组。

## [0.5.0] - 2026-08-15

### Added / 新增

- Files can now be marked as "tagging done": new `File.TagDone` column (auto-migrated), `SetTagDone` in file repository update builder, `file.SetFileTagDone` service method, and CLI `file done <文件路径>` command (`--undo` to unmark). `file info` shows the 打标状态.
- 文件支持标记为"打标完成"：新增 `File.TagDone` 字段（自动迁移）、file repository 新增 `SetTagDone` 更新构造器、`file.SetFileTagDone` 服务方法，CLI 新增 `file done <文件路径>` 命令（`--undo` 取消标记）。`file info` 展示打标状态。

## [0.4.2] - 2026-07-12

### Changed / 变更

- Tag descendant queries replaced in-memory BFS with SQLite recursive CTE (`WITH RECURSIVE`): `GetDescendantIDs` and `GetDescendantIDsMulti` in tag repository, single SQL instead of full-table load + Go traversal.
- 标签后代查询改用 SQLite 递归 CTE（`WITH RECURSIVE`）：tag repository 新增 `GetDescendantIDs`/`GetDescendantIDsMulti`，单条 SQL 替代全表加载 + Go BFS。

- `IncrementTagCnt` replaced `fmt.Sprintf` SQL string building with `gorm.Expr("MAX(tag_cnt + ?, 0)", delta)`, handling all delta values uniformly.
- `IncrementTagCnt` 改用 `gorm.Expr("MAX(tag_cnt + ?, 0)", delta)` 替代 `fmt.Sprintf` SQL 拼接，统一处理所有 delta 值。

- `Config.Set` replaced check-then-insert with GORM `ON CONFLICT DO UPDATE` upsert; added `uniqueIndex` to `Config.ConfigName` and `TagCategory.CategoryName`.
- `Config.Set` 改用 GORM `ON CONFLICT DO UPDATE` 原子操作替代 check-then-insert；`Config.ConfigName` 和 `TagCategory.CategoryName` 新增 uniqueIndex。

- `search/service.go` extracted `requireTags` (shared tag-name validation), `loadTagsForFiles` (shared batch tag loading), eliminated duplicate `buildTagChildrenMap`/`collectDescendantIDs` and ~80 lines of copy-pasted batch-loading code.
- `search/service.go` 提取 `requireTags`（共享标签名校验）和 `loadTagsForFiles`（共享批量标签加载），消除重复的 `buildTagChildrenMap`/`collectDescendantIDs` 和约 80 行复制粘贴代码。

- `tag/service.go` `GetDescendantIDs` and `getDescendantIDsInMem` now delegate to repository CTE methods, removing inline childrenMap construction and BFS traversal.
- `tag/service.go` 的 `GetDescendantIDs` 和 `getDescendantIDsInMem` 改为调用 repository CTE 方法，移除内联 childrenMap 构建和 BFS 遍历。

### Fixed / 修复

- Silent error swallowing in 7 locations: `loadExcludePatterns`, `saveScanState`, `saveDiffState`, `incrTagCnt`/`decrTagCnt`, `GetFileInfo`, `GetDefaultGroup`, `createDefaultGroup` — all now log errors via `log.Printf`.
- 7 处静默吞错修复：`loadExcludePatterns`、`saveScanState`、`saveDiffState`、`incrTagCnt`/`decrTagCnt`、`GetFileInfo`、`GetDefaultGroup`、`createDefaultGroup` — 全部改为 `log.Printf` 记录错误。

- `taggroup/service.go` `RemoveGroup`/`TagGroup`/`UntagGroup`/`GetTags` now distinguish DB errors from "not found" (`FindGroupByName` error vs nil result).
- `taggroup/service.go` 的 `RemoveGroup`/`TagGroup`/`UntagGroup`/`GetTags` 区分 DB 错误和"不存在"（`FindGroupByName` error vs nil）。

- `taggroup/service.go` `GetDefaultGroup` now checks `CreateDefaultGroup` error return instead of silently discarding it.
- `taggroup/service.go` 的 `GetDefaultGroup` 检查 `CreateDefaultGroup` 返回的错误，不再静默丢弃。

- `taggroup/group_delete.go` `DeleteGroup` now wraps cascade delete + parent delete in a transaction, preventing orphaned `tag_group_tags` rows.
- `taggroup/group_delete.go` 的 `DeleteGroup` 改用事务包裹级联删除和父记录删除，防止孤立 `tag_group_tags` 记录。

- `export/service.go` SQL injection surface: unquoted table name in `SELECT * FROM` now uses `fmt.Sprintf("%q", table)`.
- `export/service.go` SQL 注入风险：`SELECT * FROM` 中未转义的表名改用 `fmt.Sprintf("%q", table)`。

- `repository/tag/find.go` dead code branch: `if op.pID == nil` inside `if op.pID != nil` was unreachable; removed `pName` field and `ByParentName` method (unused).
- `repository/tag/find.go` 死代码分支：`if op.pID != nil` 内的 `if op.pID == nil` 永远不可达；移除未使用的 `pName` 字段和 `ByParentName` 方法。

- BFS memory leak in `collectDescendantIDs`: `queue[1:]` never releases underlying array; replaced with index-based iteration.
- `collectDescendantIDs` 内存泄漏：`queue[1:]` 不释放底层数组；改用索引遍历。

- TOCTOU race conditions: `AddCategory`, `CreateGroup` now handle unique constraint violations as fallback when concurrent insert wins the race.
- TOCTOU 竞态修复：`AddCategory`、`CreateGroup` 在并发插入竞态失败时回退检查 unique constraint 违规。

## [0.4.1] - 2026-07-12

### Added / 新增

- `ListValuesByTagIDs(tagIDs []uint)` in tag repository: batch-queries tag preset values by multiple tag IDs in a single SQL `WHERE tag_id IN (?)`, returning `map[uint][]TagValue`.
- tag repository 新增 `ListValuesByTagIDs`：按多个 tag_id 批量查询预设值，单条 SQL `WHERE tag_id IN (?)`，返回 `map[uint][]TagValue`。

- `ListTagValuesBatch(tagNames []string)` in tag service: orchestrates `FindByNames` + `ListValuesByTagIDs` to resolve tag names → IDs → values in two DB queries regardless of input count.
- tag service 新增 `ListTagValuesBatch`：编排 `FindByNames` + `ListValuesByTagIDs`，无论输入多少标签名，固定 2 次 DB 查询完成解析。

## [0.4.0] - 2026-07-12

### Added / 新增

- Structured query builder: `SearchQuery`/`SearchCondition` model with recursive nesting, supporting `filename`, `tag`, `tag_value`, `size`, `ext`, `date` condition types with AND/OR logic and nested groups.
- 结构化查询构建器：`SearchQuery`/`SearchCondition` 模型支持递归嵌套，支持 `filename`、`tag`、`tag_value`、`size`、`ext`、`date` 条件类型，AND/OR 逻辑和嵌套分组。

- `SearchByQuery` method in search service: recursive GORM query builder that resolves tag descendants and batch-loads file tags.
- search service 新增 `SearchByQuery`：递归 GORM 查询构建器，解析标签后代并批量加载文件标签。

- `tagfile search query [file]` CLI command: accepts structured JSON query from file or stdin.
- CLI 新增 `tagfile search query [file]`：从文件或 stdin 读取结构化 JSON 查询。

## [0.3.3] - 2026-05-17

### Fixed / 修复

- File search N+1: replaced client-side filtering + per-file queries with `SearchFilesWithTags` batch loading.
- 文件搜索 N+1：用 `SearchFilesWithTags` 批量加载替代客户端过滤 + 逐文件查询。

- Tag descendant N+1: replaced per-node BFS `FindDescendantIDs` with in-memory adjacency map traversal.
- 标签后代 N+1：用内存邻接表遍历替代逐节点 BFS `FindDescendantIDs`。

- Scan default group N+1: replaced per-file `createDefaultGroup` with `CreateGroupsBatch` single INSERT.
- 扫描默认组 N+1：用 `CreateGroupsBatch` 单次 INSERT 替代逐文件 `createDefaultGroup`。

- Tag name resolution N+1: replaced per-name `Find().ByName()` with `FindByNames` single `WHERE IN` query.
- 标签名解析 N+1：用 `FindByNames` 单次 `WHERE IN` 查询替代逐个 `Find().ByName()`。

- CLI group commands waste: replaced `GetFileInfo` (4 queries) with lightweight `FindByPath` (1 query).
- CLI 标签组命令浪费：用轻量 `FindByPath`（1 次查询）替代 `GetFileInfo`（4 次查询）。

- `ListByDir` root directory bug: `filepath.Dir` returns `"."`, causing `LIKE './%'` to miss DB paths; added explicit root handling.
- `ListByDir` 根目录 bug：`filepath.Dir` 返回 `"."`，`LIKE './%'` 匹配不到 DB 路径；增加显式根目录处理。

### Added / 新增

- `FindByNames` to tag repository: batch tag lookup by name list.
- tag repository 新增 `FindByNames`：批量按名称查找标签。

- `CreateGroupsBatch` to tag group repository: batch create tag groups.
- tag group repository 新增 `CreateGroupsBatch`：批量创建标签组。

- `FindByPath`, `SearchFilesWithTags` to file service: lightweight file lookup and batch search with tags.
- file service 新增 `FindByPath`、`SearchFilesWithTags`：轻量文件查找和批量标签搜索。

### Removed / 移除

- `FindDescendantIDs`, `FindChildren`, `FindRootsByCategory` from tag repository: replaced by in-memory adjacency map.
- 移除 tag repository 的 `FindDescendantIDs`、`FindChildren`、`FindRootsByCategory`：改用内存邻接表。

## [0.3.2] - 2026-05-04

### Added / 新增

- `export` command: export database to SQL file.
- `export` 命令：将数据库导出为 SQL 文件。

### Changed / 变更

- CLI tag/untag accepts tag name: unique name auto-resolves, ambiguous name lists all matches with ID and category.
- CLI tag/untag 支持标签名：唯一名称自动解析，重名则列出所有匹配（ID + 分类）。

### Fixed / 修复

- `TagFile` / `UntagFile` now use `tag_id` instead of `tag_name` to avoid ambiguity when same-named tags exist in different categories.
- `TagFile` / `UntagFile` 改为传入 `tag_id` 而非 `tag_name`，解决同名标签在不同分类下的歧义问题。

## [0.3.1] - 2026-05-04

### Added / 新增

- `File.TagCnt` field: automatic maintenance on tag add/remove via `IncrementTagCnt` in service layer.
- `File.TagCnt` 冗余字段：service 层在打标/去标时自动增减，repo 层提供 `IncrementTagCnt` 原子操作。

## [0.3.0] - 2026-05-04

### Changed / 变更

- Repository layer refactored to chainable builder pattern: `Create()`, `Find()`, `List()`, `Update()`, `Delete()`, `Search()` with one file per builder.
- Repository 层重构为链式 Builder 模式，每个 Builder 一个文件。

### Added / 新增

- `Update(model).SetXxx(v).SetYyy(v).Exec()` for selective field updates via `Select()`.
- `Update(model).SetXxx(v).Exec()` 选择性字段更新。

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
