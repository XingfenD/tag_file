# Changelog / 更新日志

All notable changes to this template should be documented in this file.
本模板的重要变更建议统一记录在此文件中。

The format loosely follows Keep a Changelog and can be adapted to the team's habits.
本文档参考了 Keep a Changelog 的思路，也可以根据团队习惯调整。

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
