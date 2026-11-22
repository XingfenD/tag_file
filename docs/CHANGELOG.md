# Changelog / 更新日志

tagfile 项目的版本变更记录。格式参考 [Keep a Changelog](https://keepachangelog.com)，版本号遵循 [Semantic Versioning](https://semver.org)。

## [1.0.0] - 2026-11-22

### Added / 新增

- Tagging schema engine: `tag_schemas`/`tag_schema_fields` models, schema repository, and `schema.Service` — configurable scenarios whose fields bind existing tags (single/bool/multi) and materialize into the default group with takeover semantics via `taggroup.ReplaceRows` (transactional row diff, TagCnt kept in step, no-ops silent).
- 打标场景 schema 引擎：新增 `tag_schemas`/`tag_schema_fields` 模型、schema 仓储与 `schema.Service`——可配置场景，字段显式绑定既有标签（单选/开关/多选），经 `taggroup.ReplaceRows`（事务内行级 diff、同步 TagCnt、no-op 静默）以接管式覆盖物化进默认标签组。

## [0.14.1] - 2026-11-21

### Fixed / 修复

- Scan and import resolve file types through one `fileTypeByExt` (environment MIME table → built-in media fallback → literal extension), so `.mp4` scans on musl containers store a real media type; `file.RepairFileTypes` repairs slash-less stored types at startup, writing back only changed values.
- 扫描与导入的文件类型统一经 `fileTypeByExt` 判定（环境 MIME 表 → 内置媒体兜底表 → 字面扩展名），musl 容器下 `.mp4` 不再以裸扩展名入库；启动时 `file.RepairFileTypes` 修复无斜杠的存量类型，仅值变化才写回。

## [0.14.0] - 2026-11-20

### Added / 新增

- `file.Service.SuggestDirs` lists workspace-relative directories a scan would walk, pruning `scan_exclude` matches with their subtrees; root excluded, symlinks not descended, parents before children, capped at 500.
- 新增 `file.Service.SuggestDirs`：列出扫描会走到的工作区相对目录，命中 `scan_exclude` 的目录连同子树剪除；根目录不作候选、符号链接不深入、父先于子、上限 500 条。

## [0.13.0] - 2026-11-20

### Added / 新增

- Recognition is a two-stage pipeline: every successful frame's raw answer is archived verbatim in `ai_frame_outputs`; with the refine endpoint on, a text-model pass batches stored frames (20/call) into the tags JSON contract (synonyms normalised, cross-frame duplicates merged); with it off the pipeline stays single-stage over the same archive. Tasks carry `refine_status`.
- 识别拆为「识图 + 整理」两段：每个成功帧的原始回答逐字归档进 `ai_frame_outputs`；开启整理端点后由文本模型把库中帧按每批 20 帧整理进标签 JSON 契约（归一同义词、跨帧合并），关闭时保持单段、直接解析同一份归档。任务行携带 `refine_status`。

- The refine stage has its own `ai_refine_*` endpoint group (9 keys, provider default `off`): the same three endpoint formats as vision, conditional URL/model/key fields, a closed enum that falls back to `off` on garbage, and per-task client construction so config edits apply to the next run.
- 整理端点拥有独立配置组 `ai_refine_*`（9 键，provider 默认 `off`）：与识图相同的三种端点格式、条件显隐的地址/模型/密钥字段、脏值一律回落 `off` 的封闭枚举，客户端按任务构建、改配置下一任务即生效。

- `ai.Service.RefineTask` queues a stage-two-only rerun over a finished task's stored raws (no file read, no frame extraction, no vision call), inserting a new task whose `source_task_id` points at the original vision task and refusing live rows, non-terminal statuses, raw-less sources, refine-off config, and files with a newer run in flight. `DeleteTask` cascades the frame outputs.
- `ai.Service.RefineTask` 入队一次只跑第二段的重新整理（不读文件、不抽帧、不调视觉模型），新建任务的 `source_task_id` 指向原始识图任务，并拒绝在飞行、非终态、无原文、端点未开启与已有更新任务在飞的文件。`DeleteTask` 级联删除帧原文。

### Changed / 变更

- The OpenAI-compatible client's `response_format: json_object` is mode-scoped: sent in off mode and for text-only refine calls, omitted from vision requests in refine mode (the field is now a pointer with `omitempty`).
- OpenAI 兼容客户端的 `response_format: json_object` 按模式分流：off 模式与纯文本整理请求照旧携带，refine 模式的识图请求不再携带（字段改指针加 `omitempty`）。

## [0.12.0] - 2026-11-19

### Added / 新增

- `ai.Service` gained an in-process change hub (`SubscribeChange`/`notifyChange`): every task/suggestion mutation pokes subscribers with a coalescing, non-blocking signal meaning "re-read state"; no schema change.
- `ai.Service` 新增进程内变更 hub（`SubscribeChange`/`notifyChange`）：任务与建议的每次变更都向订阅者发一个可合并、不阻塞的「重读状态」信号；不改 schema。

- `ai.Service.DeleteTask` removes a task with its still-pending suggestions in one status-conditional transaction: a live running task is refused, a claim landing mid-delete loses, and approved/rejected rows stay as history.
- `ai.Service.DeleteTask` 在单事务内按状态条件删除任务及其未审建议：执行中的 running 直接拒绝、删除途中被 worker 抢走则落空拒绝，已通过/已拒绝行保留为审核历史。

### Fixed / 修复

- Vision requests cap answers at 4096 tokens on all three providers (`max_tokens` / `options.num_predict`), ending 「解析模型输出失败: unexpected end of JSON input」 truncations.
- 三个 provider 的视觉请求统一设 4096 token 回答上限（`max_tokens` / `options.num_predict`），终结「解析模型输出失败: unexpected end of JSON input」截断。

- `ParseSuggestions` repairs `["category":…]`-style malformed tags-array elements on JSON syntax errors only, and the repaired text must still decode.
- `ParseSuggestions` 仅在 JSON 语法错误时修复 `["category":…]` 式的畸形 tags 元素，修复后仍须解码成功。

## [0.11.0] - 2026-11-19

### Changed / 变更

- Config copy is minimal: labels state what to fill, descriptions are optional and kept only for facts the label cannot carry; no description means no ⓘ bubble.
- 配置文案极简化：标签直接说明填什么，描述可选且只保留标签传达不了的事实；无描述即无 ⓘ 气泡。

### Added / 新增

- The config schema projects a `required` boolean per input box ("an empty value is illegal"), probed with `""` for singles and `[]` for containers.
- 配置 schema 为每个输入框下发 `required`（语义「空值非法」）：Single 探 `""`、容器探 `[]`。

### Fixed / 修复

- The block registry keeps descriptor declaration order (the settings-page layout contract), and the Ollama fields are declared URL-first to match the other providers' slot order.
- 块注册表保持描述符声明顺序（设置页布局契约），Ollama 字段改为地址在前，与其他 provider 的槽序一致。

## [0.10.0] - 2026-11-19

### Added / 新增

- Vision recognition supports Ollama native (default), OpenAI-compatible, and Anthropic endpoints, selected by `ai_provider` with per-provider conditional URL/model/key fields, masked left-empty-to-keep secrets, hand-written stdlib clients built per task, and byte-sniffed image media types.
- 视觉识别支持 Ollama 原生（默认）、OpenAI 兼容与 Anthropic 端点，经 `ai_provider` 选择，各端点条件显隐地址/模型/密钥，secret 掩码留空不覆盖；客户端为按任务构建的手写标准库 HTTP，图片媒体类型按字节嗅探。

## [0.9.0] - 2026-11-19

### Changed / 变更

- Video recognition feeds the vision model one frame per request; suggestions merge and dedupe across frames, and the task succeeds if any frame did, with failed frames summarised in `error`.
- 视频识别改为逐帧请求视觉模型；建议跨帧合并去重，至少一帧成功即完成，失败帧汇总进 `error`。

- Frame extraction covers the whole video: interval stretches to `max(configured, duration/(maxFrames*10))`, ffmpeg hard-cut detection adds shot boundaries, candidates are thinned to the frame budget, extracted at ≤768px, with a duration-scaled deadline (max 120 s, 2×duration, cap 30 min).
- 抽帧覆盖全片：间隔拉伸为 `max(配置间隔, 时长/(maxFrames*10))`，ffmpeg 硬切检测纳入分镜切换点，候选按帧预算均匀薄化、≤768px 定点抽取，超时随时长伸缩（max(120s, 2×时长)，封顶 30min）。

- The near-duplicate filter uses area-averaged 9×8 dHash, compares against all kept frames, never early-stops, thins uniformly when over budget, and streams from disk keeping only ≤maxFrames in memory.
- 近似帧过滤改用区域平均 9×8 dHash，与所有已保留帧比对、不早停、超预算均匀薄化，流式读盘、内存只留最终 ≤maxFrames 帧。

## [0.8.0] - 2026-11-17

### Changed / 变更

- The config schema moved from a closed `FieldType` enum to an orthogonal model (`ValueType` × `ContainerType` × typed constraints) in the leaf `schema` package, validated at registration; `ai_frame_interval` migrated to a fixed-point wire via an idempotent one-time migration (`ai_frame_interval_fixedpoint_v1`).
- 配置 schema 由封闭 `FieldType` 枚举改为正交模型（`ValueType` × `ContainerType` × 类型化约束），在叶子包 `schema` 声明、注册期校验；`ai_frame_interval` 经一次性幂等迁移（`ai_frame_interval_fixedpoint_v1`）改存定点数。

- Validation, rendering and behaviour are one backend-sourced rule protocol: descriptors decompose into `Rule`s (validate/render/behavior) from a single registry, and server errors and frontend copy render from the same templates.
- 校验、渲染与行为统一为后端单一规则协议：描述符分解为来自唯一注册表的 `Rule`（validate/render/behavior），服务端错误与前端文案同模板渲染。

- Field registration became a block registry (`RegisterBlock`/`Blocks`/`BlockOf`/`Lookup`/`All`) with startup panics on duplicates or illegal defaults; `SetGroup` saves per block and secrets are recognised via `IsSecret()`.
- 字段注册改为块注册表（`RegisterBlock`/`Blocks`/`BlockOf`/`Lookup`/`All`），重复或非法声明启动即 panic；`SetGroup` 按块保存，secret 经 `IsSecret()` 识别。

- Rule `Hint` carries behaviour copy only; per-field usage instructions belong to the descriptor `Description`.
- 规则 `Hint` 只承载行为文案；逐字段用法说明归属描述符 `Description`。

### Removed / 移除

- The legacy `FieldType`/`Field`/`Group` model and its registration/validation surface are deleted; transitional `*InputBox` names folded back to `Validate`/`Lookup`/`All`.
- 旧 `FieldType`/`Field`/`Group` 模型及其注册/校验接口删除；过渡名 `*InputBox` 收拢回 `Validate`/`Lookup`/`All`。

## [0.7.1] - 2026-11-16

### Removed / 移除

- `Service.ListUnknown`/`SearchUnknown` (the advanced area's data source) are gone and `GET /api/config` no longer accepts `?unknown=1`; unknown keys still pass through `Set`/`Update`/`Delete` for the CLI.
- 删除 `Service.ListUnknown`/`SearchUnknown`（高级区数据源），`GET /api/config` 不再接受 `?unknown=1`；未注册键仍经 `Set`/`Update`/`Delete` 透传，CLI 不受影响。

## [0.7.0] - 2026-11-16

### Added / 新增

- Leaf package `internal/service/config/schema` declares the config schema: a closed `FieldType` set and a `Field` carrying label, description, default, bounds, pattern, options, item type, unit, and `ShowIf`.
- 新增叶子包 `internal/service/config/schema` 声明配置结构：封闭 `FieldType` 集合，`Field` 携带显示名、描述、默认值、边界、模式、选项、项类型、单位与 `ShowIf`。

- `schema.Validate(field, value)` checks stored values per type with Chinese messages spelling out the legal range; URL/list/regex rules included, NaN/±Inf rejected, empty string legal for `string`/`text`/`secret`.
- `schema.Validate(field, value)` 按类型校验存量取值，中文报错写明合法区间；含 URL/列表/正则规则，拒绝 NaN 与 ±Inf，`string`/`text`/`secret` 空串合法。

- The registry self-checks at declaration time (duplicate group/key or a rejected default panics at startup) and orders deterministically (`Groups()` by Order+ID, `FieldsOf`/`All` by key).
- 注册表在声明期自检：重复分组/键、非法默认值启动即 panic；排序确定（`Groups()` 按 Order+ID，`FieldsOf`/`All` 按键）。

- Consumers register their own keys in `init`: the `ai` group (5 recognition keys with bounds taken from runtime constants) and `scan` with `scan_exclude`; `bootstrap` blank-imports ai so the CLI seeds the AI keys too.
- 各消费方在 `init` 登记自己的键：`ai` 组 5 个识别键（边界取自运行时常量）与 `scan` 的 `scan_exclude`；`bootstrap` 空导入 ai，CLI 同样播种 AI 键。

- Strict group save: `config.ValidationErrors` plus `Service.SetGroup` — every non-secret field must be present and valid, secrets may be omitted (= unchanged), any failure writes nothing, and a clean map lands in one transaction.
- 整组严格保存：`config.ValidationErrors` + `Service.SetGroup`——非 secret 字段必须出现且合法，secret 可省略（保持不变），任一失败整组零写入，全部通过单事务落库。

- `Service.ListUnknown`/`SearchUnknown` listed registry-unknown keys with filtering before pagination (removed again in 0.7.1).
- `Service.ListUnknown`/`SearchUnknown` 列出未注册键，先过滤后分页（0.7.1 中又删除）。

### Changed / 变更

- `Service.Set`/`Update` validate registered keys before writing (「配置 <key> 不合法: <原因>」); `Delete` refuses known keys and points at 恢复默认; unknown keys still pass through.
- `Service.Set`/`Update` 写入前校验已注册键（「配置 <key> 不合法: <原因>」）；`Delete` 拒绝已知键并指向「恢复默认」；未注册键照旧透传。

- `bootstrap.seedDefaultConfig` walks `schema.All()` and inserts only missing keys; the hand-written defaults list is gone and no table migration is needed.
- `bootstrap.seedDefaultConfig` 遍历 `schema.All()` 只补缺失键；手写默认值清单删除，无需表迁移。

- `ai.MaxFramesLimit` and `ai.MaxDiffThreshold` are exported so schema and WebUI reuse the runtime bounds.
- `ai.MaxFramesLimit`/`ai.MaxDiffThreshold` 改为导出，schema 与 WebUI 复用运行时上界。

## [0.6.0] - 2026-11-14

### Added / 新增

- Ollama vision auto-tagging under `internal/service/ai`: `AITask` (`ai_tasks`) and `AISuggestion` (`ai_suggestions`) tables via `AutoMigrate`, repository in `internal/repository/ai` (chainable builders for list/update), and the worker pipeline (frame extraction, dHash filtering, Ollama client, prompt + parser, sequential `StartWorker`/`StopWorker` with a start sweep that fails orphaned `running` rows 重启中断).
- `internal/service/ai` 落地 Ollama 视觉自动打标：`AITask`/`AISuggestion` 两表随 `AutoMigrate`；`internal/repository/ai` 仓储（列表/更新走链式构造器）；worker 流水线（抽帧、dHash 过滤、Ollama 客户端、提示词与解析、单协程 `StartWorker`/`StopWorker`，启动清扫把遗留 running 行标为 重启中断）。

- `ai.IsMediaFile(name)` gates queueing by extension; `EnqueuePaths`/`EnqueueDir` report every refusal as `SkippedFile{path, reason}` and gate under one mutex so overlapping batches cannot double-queue.
- `ai.IsMediaFile(name)` 按扩展名将开关卡入队；`EnqueuePaths`/`EnqueueDir` 逐条上报 `SkippedFile{path, reason}`，判定在同一把锁下完成，重叠批次不会重复入队。

- Five seeded config keys — `ollama_url`, `ollama_model`, `ai_frame_interval`, `ai_frame_diff_threshold`, `ai_max_frames` — read by `ai.Service.Settings()` with per-key fallback to defaults (threshold 1..64, max frames 1..200 enforced).
- 新增并播种 5 个配置键——`ollama_url`、`ollama_model`、`ai_frame_interval`、`ai_frame_diff_threshold`、`ai_max_frames`——`ai.Service.Settings()` 逐键读取、越界即回退默认（阈值 1..64、帧数 1..200）。

- Suggestions write once per task: capped at 50 rows with the discarded remainder recorded in `error`, parsed fields clamped to column widths in runes; `RetryTask` also accepts stalled `running` rows while a live one is refused 「正在执行中」.
- 建议每任务一次写入：上限 50 条、丢弃部分记入 `error`，解析字段按列宽以 rune 裁剪；`RetryTask` 同时接受无人认领的 `running` 行，执行中的返回「正在执行中」。

- The human review gate: `Approve`/`Reject`/`UpdateSuggestion` — nothing enters the library unapproved; approvals resolve or create tags inside the reviewer's chosen category via the tag/file services, never auto-creating categories or writing the `TagValue` dictionary; per-row failures don't fail the batch.
- 人工审核关口：`Approve`/`Reject`/`UpdateSuggestion`——未经通过不进标签库；通过在审核者选定分类内复用或创建标签（经 tag/file 服务保持一致性），不静默建分类、不写 `TagValue` 字典；单行失败不拖垮整批。

- `ai.Service.FileStatuses(fileIDs)` derives each file's display status from its newest task in two queries; `File.TagDone` stays a manual dimension.
- `ai.Service.FileStatuses(fileIDs)` 以两次查询按最新任务推导文件展示状态；`File.TagDone` 保持人工维度。

### Changed / 变更

- SQLite DSN sets `busy_timeout(5000)` so a second writer waits instead of erroring; the connection pool stays unlimited.
- SQLite DSN 显式 `busy_timeout(5000)`，后发写入者等待而非报锁错误；连接池保持不限。

- 75 new tests across the ai service/repository, `pkg/storage` and `internal/bootstrap`.
- 新增 75 个测试，覆盖 ai 服务/仓储、`pkg/storage` 与 `internal/bootstrap`。

## [0.5.8] - 2026-11-12

### Added / 新增

- `file.Service.RandomFile(scope, exclude)` draws one indexed file for tagging over scopes `todo`/`untagged`/`all`, reporting pool size, remaining, and an `Exhausted` flag; backed by the `file.Repository.Random()` builder (`ORDER BY RANDOM()`) with tests.
- `file.Service.RandomFile(scope, exclude)` 随机抽取已索引文件（范围 `todo`/`untagged`/`all`），返回池量、剩余与 `Exhausted` 标记；底层为 `file.Repository.Random()` 构造器（`ORDER BY RANDOM()`），配测试。

## [0.5.7] - 2026-11-12

### Fixed / 修复

- `ScanDirectory`/`DiffUnindexed` index paths relative to the working directory; directories outside it are rejected.
- `ScanDirectory`/`DiffUnindexed` 按相对工作目录的路径索引；工作目录之外的目录直接拒绝。

- Unbuildable search conditions (unknown `type`/`op`, bad date, empty extension list) return an error instead of silently widening the query; a no-match tag name collapses `has` and no-ops `not_has`.
- 无法构建的搜索条件（未知 `type`/`op`、坏日期、空扩展名列表）返回错误而非静默放宽查询；匹配不到的标签名使 `has` 收敛为空集、`not_has` 不设约束。

- Deleting a tag group decrements `File.TagCnt` by its association count; deleting a tag cascades its `tag_group_tags`/`tag_values` rows, re-parents children, and fixes counters, in one transaction.
- 删除标签组按其关联数递减 `File.TagCnt`；删除标签在单事务内级联清理 `tag_group_tags`/`tag_values`、子标签提根并修正计数。

- `TagFileToGroup`/`UntagFileFromGroup` reject foreign groups, matching the existing ownership checks; `RenameFile` rejects names containing path separators; `ImportFile` creates the default tag group for copied files.
- `TagFileToGroup`/`UntagFileFromGroup` 拒绝外部标签组，与既有归属校验一致；`RenameFile` 拒绝含路径分隔符的名字；`ImportFile` 为复制进来的文件创建默认标签组。

- Regression tests cover the path basis, condition building, deletion cascades, group ownership, and rename constraints.
- 回归测试覆盖路径基准、条件构建、删除级联、组归属与重命名约束。

## [0.5.6] - 2026-08-29

### Changed / 变更

- Added indexes on file path/name, tag name/category/parent, and group-tag `tag_id`; file-tree building O(n²)→O(n); descendant resolution batched into one recursive CTE (`GetDescendantIDsMultiMap`); `ImportFile` and `loadTagsForFiles` switched to single lookups.
- 为文件路径/名、标签名/分类/父级、组关联 `tag_id` 建索引；文件树构建 O(n²)→O(n)；后代解析合并为单条递归 CTE（`GetDescendantIDsMultiMap`）；`ImportFile` 与 `loadTagsForFiles` 改单次查找。

### Added / 新增

- `tag.Repository.FindByIDs`, `GetDescendantIDsMultiMap`, and `search.Service.SearchByQueryPaged` for SQL-level pagination.
- 新增 `tag.Repository.FindByIDs`、`GetDescendantIDsMultiMap` 与 SQL 层分页的 `search.Service.SearchByQueryPaged`。

### Fixed / 修复

- `tag.FindOperation.Exec()` no longer nil-panics without `ByID()`.
- `tag.FindOperation.Exec()` 未调 `ByID()` 时不再空指针 panic。

## [0.5.5] - 2026-08-29

### Added / 新增

- `file.Service.BuildTreeFromDB()` (tree from the database, no filesystem scan) and `ExpandDir(dir)` (indexed + unindexed children via single-level `os.ReadDir` + DB query).
- 新增 `file.Service.BuildTreeFromDB()`（纯数据库建树、不扫文件系统）与 `ExpandDir(dir)`（单层 `os.ReadDir` + DB 查询返回已/未索引子项）。

## [0.5.4] - 2026-08-16

### Added / 新增

- Core version recorded in `.tagfile/core_version`; on a version change the DB is backed up before migration. Version constants centralized (`internal/constant.CoreVersion`, `cmd/webui/constant.WebuiVersion`).
- 核心版本记录于 `.tagfile/core_version`，版本变化时迁移前自动备份数据库；版本常量集中（`internal/constant.CoreVersion`、`cmd/webui/constant.WebuiVersion`）。

## [0.5.3] - 2026-08-16

### Fixed / 修复

- `tag.UpdateTag` returns an explicit cross-category move error with both category names, and `Find().Exec()` applies preloads so names are never blank.
- `tag.UpdateTag` 对跨分类移动返回带两侧分类名的明确错误；`Find().Exec()` 正确应用预加载，分类名不再为空。

## [0.5.2] - 2026-08-16

### Fixed / 修复

- `tag.UpdateTag` takes a parent pointer (`nil` unchanged, `""` to root, name re-parents); root-level duplicate checks filter `parent_id IS NULL`; WebUI treats present-but-empty as move-to-root.
- `tag.UpdateTag` 父标签改指针语义（`nil` 不变、空串移根、传名改父）；根层重名检查按 `parent_id IS NULL` 过滤；WebUI 把「存在但为空」视为移到根层。

## [0.5.1] - 2026-08-16

### Fixed / 修复

- `file.CreateGroup` treats `ErrRecordNotFound` as "no existing group", matching `RenameGroup`.
- `file.CreateGroup` 将 `ErrRecordNotFound` 视为「无同名组」，与 `RenameGroup` 约定一致。

## [0.5.0] - 2026-08-15

### Added / 新增

- `File.TagDone` column with repository/service setters and CLI `file done <path>` (`--undo`); `file info` shows the status.
- 新增 `File.TagDone` 字段与仓储/服务写方法，CLI `file done <路径>`（`--undo` 取消）；`file info` 展示打标状态。

## [0.4.2] - 2026-07-12

### Changed / 变更

- Descendant queries moved to SQLite recursive CTEs (`GetDescendantIDs`/`GetDescendantIDsMulti`); `IncrementTagCnt` uses `gorm.Expr`; `Config.Set` became an upsert with unique indexes on config/category names; search extracted shared `requireTags`/`loadTagsForFiles`.
- 后代查询改用 SQLite 递归 CTE（`GetDescendantIDs`/`GetDescendantIDsMulti`）；`IncrementTagCnt` 改用 `gorm.Expr`；`Config.Set` 改 upsert 并给配置/分类名加 uniqueIndex；搜索提取共享 `requireTags`/`loadTagsForFiles`。

### Fixed / 修复

- Silent error swallowing fixed in 7 places (now logged); group operations distinguish DB errors from not-found; `DeleteGroup` wrapped in a transaction; export table names quoted; BFS queue leak replaced; TOCTOU races on unique constraints handled.
- 修复 7 处静默吞错（现记日志）；标签组操作区分 DB 错误与不存在；`DeleteGroup` 事务包裹；导出表名加引号；BFS 队列内存泄漏修复；唯一约束 TOCTOU 竞态兜底。

## [0.4.1] - 2026-07-12

### Added / 新增

- Batch tag-value lookups: `ListValuesByTagIDs` (repository, single `WHERE IN`) and `ListTagValuesBatch` (service, two queries regardless of input size).
- 标签预设值批量查询：`ListValuesByTagIDs`（单条 `WHERE IN`）与 `ListTagValuesBatch`（服务层固定两次查询）。

## [0.4.0] - 2026-07-12

### Added / 新增

- Structured query builder: `SearchQuery`/`SearchCondition` with recursive nesting over filename/tag/tag_value/size/ext/date conditions; `SearchByQuery` and CLI `tagfile search query`.
- 结构化查询构建器：`SearchQuery`/`SearchCondition` 递归嵌套，支持文件名/标签/标签值/大小/扩展名/日期；新增 `SearchByQuery` 与 CLI `tagfile search query`。

## [0.3.3] - 2026-05-17

### Fixed / 修复

- Eliminated N+1 patterns in file search, tag descendants, default-group creation, and tag-name resolution; `ListByDir` handles the `"."` root path.
- 消除文件搜索、标签后代、默认组创建、标签名解析中的 N+1 查询；`ListByDir` 处理 `"."` 根路径。

### Added / 新增

- `FindByNames`, `CreateGroupsBatch`, `FindByPath`, `SearchFilesWithTags`.
- 新增 `FindByNames`、`CreateGroupsBatch`、`FindByPath`、`SearchFilesWithTags`。

### Removed / 移除

- `FindDescendantIDs`, `FindChildren`, `FindRootsByCategory` (replaced by in-memory adjacency traversal).
- 移除 `FindDescendantIDs`、`FindChildren`、`FindRootsByCategory`（改用内存邻接遍历）。

## [0.3.2] - 2026-05-04

### Added / 新增

- `export` command (database → SQL file).
- 新增 `export` 命令（数据库导出 SQL 文件）。

### Changed / 变更

- CLI tag/untag accept tag names, listing matches on ambiguity.
- CLI 打/去标签支持标签名，重名时列出所有匹配。

### Fixed / 修复

- `TagFile`/`UntagFile` resolve by `tag_id` instead of name.
- `TagFile`/`UntagFile` 改用 `tag_id`，避免同名歧义。

## [0.3.1] - 2026-05-04

### Added / 新增

- `File.TagCnt` maintained automatically via `IncrementTagCnt`.
- 新增 `File.TagCnt` 冗余计数，经 `IncrementTagCnt` 自动维护。

## [0.3.0] - 2026-05-04

### Changed / 变更

- Repository layer restructured into the chainable builder pattern (`Create`/`Find`/`List`/`Update`/`Delete`/`Search`, one file per builder) with selective field updates.
- 仓储层重构为链式 Builder 模式（`Create`/`Find`/`List`/`Update`/`Delete`/`Search`，一 Builder 一文件），支持选择性字段更新。

## [0.2.0] - 2026-05-04

### Added / 新增

- Tag groups: `TagGroup`/`TagGroupTag` models (default group auto-created on scan/import, redundant `file_id` avoids JOINs) and `file group {add,remove,list,tag,untag}` CLI commands.
- 标签组：`TagGroup`/`TagGroupTag` 模型（扫描/导入自动建默认组，冗余 `file_id` 免 JOIN）与 `file group {add,remove,list,tag,untag}` CLI 命令。

### Removed / 移除

- `FileTag`; associations migrated to `TagGroupTag`.
- 移除 `FileTag`，关联迁移至 `TagGroupTag`。

## [0.1.2] - 2026-05-04

### Added / 新增

- `ListFileTags` with pagination and `file tags <path>` CLI command.
- 新增 `ListFileTags` 分页查询与 CLI `file tags <路径>`。

## [0.1.1] - 2026-05-04

### Added / 新增

- `tag update` with cycle detection; the `TagValue` system (`tag value {add,remove,list}`, `file tag --value`, `标签:值` display); regex `scan_exclude` config with seeded defaults; error logging across services.
- 新增 `tag update`（含循环检测）；`TagValue` 标签值体系（`tag value {add,remove,list}`、`file tag --value`、`标签:值` 展示）；正则 `scan_exclude` 配置及默认值播种；服务层全面补错误日志。

### Changed / 变更

- Tag uniqueness scoped to `(parent_id, tag_name)`; file tags unique at `(tag_id, tag_value)`.
- 标签唯一性收敛到 `(parent_id, tag_name)`；文件标签按 `(tag_id, tag_value)` 唯一。

## [0.1.0] - 2026-05-04

### Removed / 移除

- The entire attribute system (models, repo, service, CLI).
- 移除属性系统（模型、仓储、服务、CLI）。

### Added / 新增

- Tree-structured tags (`Tag.ParentID`, `tag add --parent`, `tag list --tree`); search auto-expands parent tags to descendants; `ListTagTree`, `GetDescendantIDs`, `SearchByTagGroups`.
- 树形标签（`Tag.ParentID`、`tag add --parent`、`tag list --tree`）；搜索自动把父标签展开为子孙集合；新增 `ListTagTree`、`GetDescendantIDs`、`SearchByTagGroups`。

### Changed / 变更

- Seed categories merged (`"具体品种"` → `"动物类型"`, `"风景"` → `"场景内容"`); setup scripts, fixtures, and docs updated.
- 种子分类合并（「具体品种」并入「动物类型」，「风景」移入「场景内容」）；初始化脚本、测试数据与文档同步更新。

## [0.0.7] - 2026-05-04

### Changed / 变更

- `internal/service/` and `internal/repository/` reorganised into domain subdirectories, files split by concern.
- `internal/service/` 与 `internal/repository/` 按领域划分子目录，大文件按职责拆分。

### Added / 新增

- Update methods for all entities with service-layer uniqueness checks; fuzzy search for all entities; `RenameFile` (disk + DB atomic).
- 全实体更新方法（服务层唯一性校验）；全实体模糊查询；`RenameFile` 磁盘与数据库原子更新。

### Fixed / 修复

- `CreateBatch` chunks inserts at 100 to stay under SQLite's parameter limit.
- `CreateBatch` 按 100 条分片写入，避免超出 SQLite 参数上限。

## [0.0.6] - 2026-05-03

### Changed / 变更

- Scan state split into `.tagfile/scan_state` and `.tagfile/diff_state`; auto full scan when root mtime moves past the last scan; CLI commands reorganised into sub-packages with shared utils.
- 扫描状态拆分为 `.tagfile/scan_state` 与 `.tagfile/diff_state`；根目录 mtime 晚于上次扫描时间时自动全量扫描；CLI 命令按子包重组并归拢共享工具函数。

### Fixed / 修复

- Removed unused scan-filter code from the service layer.
- 删除服务层未使用的扫描过滤器代码。

## [0.0.5] - 2026-05-01

### Added / 新增

- `file exists`, `file diff [-r]`, and `file scan --full`.
- 新增 `file exists`、`file diff [-r]`、`file scan --full`。

### Changed / 变更

- DB moved to `.tagfile/tagfile.db`; scan state stored in `.tagfile/last_scan`.
- 数据库移至 `.tagfile/tagfile.db`；扫描状态写入 `.tagfile/last_scan`。

### Fixed / 修复

- Scan filters are case-insensitive.
- 扫描过滤器不再区分大小写。

### Performance / 性能

- Scan N+1 eliminated (single indexed-path map load), inserts batched, `WalkDir` replaces `Walk`, redundant `os.Stat` calls dropped.
- 消除扫描 N+1（单次加载已索引路径 map）、批量写入、`WalkDir` 替代 `Walk`、去掉冗余 `os.Stat`。

## [0.0.4] - 2026-05-01

### Added / 新增

- Attribute and config layers (`AttrRepository`/`AttrService`, `ConfigRepository`/`ConfigService`) with `tagfile attribute` and `tagfile config` CLI commands, plus 17 tests.
- 属性与配置两层（`AttrRepository`/`AttrService`、`ConfigRepository`/`ConfigService`）及 `tagfile attribute`、`tagfile config` 命令，附 17 个测试。

### Changed / 变更

- `ImportFile` copies into CWD instead of an `imports/` subdirectory; test workspace and docs updated.
- `ImportFile` 直接复制到当前目录，不再走 `imports/` 子目录；测试工作区与文档同步。

## [0.0.3] - 2026-05-01

### Added / 新增

- File management CLI (`tagfile file {scan,import,list,tag,untag,info,search}`), tag search (`tagfile search tag`), `FileRepository`/`FileService`/`SearchService`, test workspace script, and modular test targets.
- 文件管理 CLI（`tagfile file {scan,import,list,tag,untag,info,search}`）、标签搜索（`tagfile search tag`）、`FileRepository`/`FileService`/`SearchService`、测试工作区脚本与模块化测试目标。

### Fixed / 修复

- `GetTags` preloads `Category`; `ScanDirectory` skips dot entries.
- `GetTags` 预加载 `Category`；`ScanDirectory` 跳过点号项。

## [0.0.2] - 2026-05-01

### Added / 新增

- Category and tag CLIs with repository/service layers (uniqueness validation), a standalone test entry point, and `AGENTS.md`.
- 分类与标签 CLI 及仓储/服务层（唯一性校验）、独立测试入口、`AGENTS.md`。

### Changed / 变更

- GORM debug logging silenced for cleaner CLI output.
- GORM 调试日志改静默，命令行输出更干净。

## [0.0.1] - 2026-04-29

### Added / 新增

- Project-template READMEs (EN/ZH) and this changelog.
- 项目模板 README（中英双语）与本更新日志。

### Notes / 说明

- The repository provides structure and placeholders rather than a finished implementation.
- 当前仓库主要提供目录结构与占位文件。
