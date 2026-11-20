# AI 两段式识别：识图原文落库 + 文本模型批量整理

日期：2026-11-20
状态：待批准
涉及：`internal/service/ai/`、`internal/model/ai.go`、`internal/repository/ai/`、`cmd/webui/`（配置页、审核页、API）
版本：core 0.12.0 → 0.13.0；webui 5.15.0 → 5.16.0

## 1. 意图与背景

现役识图模型 joycaption-beta-one 是标注专用模型，文本输出能力弱：无视 JSON
grammar、tags 数组系统性方括号畸形、编造分类名。现有单段管线把整个标签库 +
JSON 契约塞进识图提示词（`prompt.go`），解析容错（`parse.go`）追着模型的畸形
输出打补丁，仍有文件失败。

本设计把识别拆成两段：

- **阶段一（识图）**：识图模型只管看图，自由文本输出，不再要求 JSON、不再展示标签库。
- **阶段二（整理）**：独立的文本模型端点，把识图原文批量整理成系统契约
  `{"tags":[{"category","tag","value"}]}`，负责对齐现有分类/标签库（含同义词归一，
  如「沙滩」→「海滩」）、跨帧语义去重。

识图原文按帧落库、按任务保留，一个文件多次任务的原文共存；整理失败或想换文本
模型/提示词时，可「只重新整理」，不重跑 GPU。

## 2. 目标 / 非目标 / 反目标

**目标**
1. 识图原文持久化：每任务每帧一行，任务历史不覆盖（RetryTask/强制重识别本就创建新任务行）。
2. 阶段二批量：一个任务的全部帧原文一次文本调用整理（超长按帧数分批，批间走现有合并）。
3. 「只重新整理」入口：基于已存原文重跑阶段二，不重跑阶段一。
4. 文本端点独立一组配置（provider/地址/模型/密钥），默认关闭；关闭时行为与今天完全一致。
5. 标签库 + JSON 契约从识图提示词搬进整理提示词（仅当整理开启）。

**非目标**
- 不动审核门：分类必须已存在、根标签全局唯一、人工 Approve/Reject、raw/Final 双份存储，全部照旧。
- 不做并发识别/整理：单 worker 串行不变（显存是稀缺资源，反目标「并发打爆显存」沿用）。
- 不做原文自动清理/保留策略：原文随任务历史无限保留。
- 不 loosening 解析容错：`ParseSuggestions` 及其修复逻辑原样保留，整理输出仍走它。

**反目标**
- 整理模型静默改写语义之外的发明：提示词要求它只基于识图原文整理，可映射到库内已有名字，不得凭空添加原文没有的内容。
- 半套建议入库：建议写入仍是单语句原子写（50 条上限沿用），整理失败不产生半套待审。

## 3. 约束

- Schema 只走 GORM `AutoMigrate`，无裸 SQL 迁移。
- 配置走块注册表规则协议（`config_schema.go` + `internal/service/config/schema`）：
  封闭枚举、`ShowIf` 单键条件、per-provider 字段按「地址 → 模型 → 密钥」同序声明。
  `ShowIf` 只支持单键等值条件，故「整理关闭」必须是枚举里的一个值，不能是独立布尔键。
- 部分成功语义沿用：一帧坏只丢那一帧，任务 done + Error 说明；全败才 failed。
- 建议写入原子性沿用：`maxTaskSuggestions = 50`，低于 repo 100 行批边界。
- WebUI 响应走 `respond(c, ...)`；DELETE 读 query；SSE 推送 + 指纹守卫，无轮询。
- 客户端按任务构建（配置编辑下一个任务生效），整理客户端同此。

## 4. 备选方案（已决）

- **A. 逐帧串联**：每帧 识图→整理→解析，走现有合并。改动最小，但每帧一次文本调用，
  且重整理必须重跑 GPU。**未选**。
- **B. 整任务批量（无存储）**：攒齐全部帧原文一次整理。调用最少，但一次失败全丢，
  重试要重跑 GPU。**未选**，其批量思想被 C 吸收。
- **C. 两段分储 + 批量整理（用户选定）**：原文落库，整理独立成段，支持只重整理。
  存储兜住了 B 的「一次失败全丢」——原文在库里，重整理不花 GPU。代价：新表、
  任务模型扩列、重整理入口与 UI。

用户明确要求：一个文件多次任务的识图原文都要存；整理采用 B 的批量方式。

## 5. 设计

### 5.1 配置（`config.go` / `config_schema.go`）

新增 9 个键，镜像现有 provider 组：

```
ai_refine_provider          封闭枚举: off | ollama | openai | anthropic，默认 off
ai_refine_ollama_url        ShowIf provider=ollama    默认 http://127.0.0.1:11434
ai_refine_ollama_model      ShowIf provider=ollama
ai_refine_openai_url        ShowIf provider=openai    默认 https://api.openai.com/v1
ai_refine_openai_model      ShowIf provider=openai
ai_refine_openai_api_key    ShowIf provider=openai    Secret
ai_refine_anthropic_url     ShowIf provider=anthropic 默认 https://api.anthropic.com
ai_refine_anthropic_model   ShowIf provider=anthropic
ai_refine_anthropic_api_key ShowIf provider=anthropic Secret
```

- 「推理端点类型」现有块内新增一个「整理端点类型」枚举字段，per-provider 字段
  `ShowIf` 挂在 `ai_refine_provider` 上，与识图组互不干扰。
- `Settings` 增加对应字段；`refineEnabled(settings)` = provider != off。
- provider=off 时其 per-provider 字段隐藏；已存的旧值不清空（切回即恢复）。
- 整理端点配置了 provider 但模型名为空：与识图同形，任务报错
  「未配置整理端点的 X 模型名, 请先在配置页填写」，不静默回落单段。
- 文案遵循最小化原则：只给非显而易见的事实写描述（如 OpenAI 地址须含 /v1）。

### 5.2 存储（`internal/model/ai.go`，AutoMigrate）

新表 `ai_frame_outputs`：

```go
type AIFrameOutput struct {
    OutputID   uint   // pk
    TaskID     uint   // index，识图任务（阶段一）的 id
    FileID     uint   // index
    FrameIndex int    // 1 起，原始帧序号（失败帧不占行，序号可有空洞）
    RawText    string // type:text，识图模型原文，逐字保存
    CreatedAt  time.Time
}
```

- **逐帧即写**：阶段一每帧 Recognize 成功立即写一行，**无论输出是否为合法 JSON**——
  原文逐字保存正是落库的价值，畸形 JSON 的帧日后开启整理再重整理即可救回。
  「失败帧」仅指传输层失败（Recognize 报错），不指解析失败。进程中途崩溃/重启
  已识别的帧不丢。建议写入仍是末尾一次性，原文写入是追加型，两者原子性要求不同。
- 传输失败帧不写行；失败帧号与原因沿用现有 Error 注记（「第 X 帧识别失败…」）。
- 原文只属于识图任务；重整理任务通过 `SourceTaskID` 引用，不复制原文。
- **任务删除**：现有 DeleteTask 事务内级联删除该任务的原文行（与「删任务连带删
  建议」的既有语义一致）。被引用的源任务删除后，指向它的在途重整理任务执行时
  以「源任务没有识图原文」失败；已完成重整理任务的建议不受影响（建议属于它自己）。

`AITask` 扩两列：

```go
SourceTaskID uint   // 0=普通识别任务；非 0=只重整理任务，指向原文所属的识图任务
RefineStatus string // size:20，''=未启用整理；pending/done/failed
```

- `Status` 语义不变（queued/running/done/failed），仍由 `processTask` 独占终态写入。
- `RefineStatus` 记录阶段二结果：整理关闭时恒 ''；开启时阶段二开始前置 pending，
  成功 done，失败 failed。任务整体 failed 而原文已存时，重整理是廉价恢复路径。
- 重启清扫（`FailRunning`）语义不变；原文不受影响。

### 5.3 管线（`recognize.go` 拆分）

`recognize(task)` 按 `SourceTaskID` 分流：

**阶段一（SourceTaskID == 0）**
1. file → settings → 识图 client → loadFrames（全部照旧）。
2. 提示词分流：整理开启 → 自由描述提示词（见 5.4）；关闭 → 现有 BuildPrompt 原样。
3. 逐帧 Recognize；成功即写 `ai_frame_outputs`；失败帧记录沿用现状。
4. 全帧皆败 → 任务 failed（无原文）。有任一帧成功 → 进入阶段二。
5. `FrameCount` 照旧写入。

**阶段二（两个入口共用）**
- 输入：本任务（或 SourceTaskID 指向任务）的全部原文行，按 FrameIndex 排序。
- 整理关闭：逐帧原文直接走现有 `ParseSuggestions` → `mergeRawTags`（今天的行为，
  只是输入从内存换成库里的原文）。**解析失败按帧级注记报告**，措辞与今天完全一致
  （「第 X 帧识别失败(n/N), 首条原因…」）；全部帧都解析不出 → 任务 failed，同今天。
  「关闭时行为一致」指提示词、解析、Error 文案、建议结果与请求体全部照旧，
  不要求内部循环结构照旧。
- 整理开启：
  1. 构建整理提示词（标签库 + 契约 + 全部帧原文，见 5.4）。
  2. 输入超长按帧数分批（批大小是建造者可调的常量，以「每帧原文 × 批帧数 +
     标签库提示词」不超出文本模型常见上下文为准），每批一次纯文本调用
     （`Recognize(ctx, prompt, nil)`，三家客户端已支持无图调用）。
  3. 每批输出走 `ParseSuggestions`；一批失败只丢该批（注记进 Error），全批皆败
     才任务 failed——部分成功语义从帧级平移到批级。
  4. 跨批 `mergeRawTags` → 50 条上限截断（注记沿用）→ `buildSuggestions` →
     单语句写入 → done。
- `RefineStatus` 随阶段二结果写入；整理失败时任务 failed、原文完好，
  Error 写明「整理失败, 可重新整理」。

**重整理任务（SourceTaskID != 0）**：跳过阶段一，直接以源任务原文跑阶段二。
跳过 `checkProviderModel`（识图端点）校验——它不发起识图调用；`FrameCount` 沿用
源任务的值。执行时整理端点已被切到 off：任务 failed，原因「整理端点未开启」
（按任务构建客户端的先例，不回落单段）。源任务无原文行（含源任务已被删除）→
任务 failed，原因「源任务没有识图原文」。

### 5.4 提示词（`prompt.go`）

- **识图自由描述提示词**（整理开启时）：要求描述画面内容并列出看到的标签，
  自由文本，无标签库、无 JSON 契约、无输出格式约束。具体措辞建造者定。
- **整理提示词**：现有 `BuildPrompt` 的标签库渲染（分类清单 + 标签树路径 + 预设值）
  整体复用，加上：
  - 全部帧原文，按「第 N 帧:」标注序号；
  - 现有 `promptContract` 与 `promptRules` 改造：category/tag 逐字用库内名字、
    同义词归一到库内已有标签、跨帧重复语义合并、只基于原文整理不得凭空发明、
    只输出 JSON。
- 整理关闭时两个新提示词都不参与，现有提示词一字不动。

### 5.5 客户端（`openai.go` / `anthropic.go` / `ollama.go`）

- 整理调用复用 `VisionClient.Recognize(ctx, prompt, nil)`，不新增传输层。
- OpenAI 客户端现在无条件发送 `response_format: json_object`。改为**按整理模式
  分流**：整理关闭时请求体与今天逐字节一致（回归零差异）；整理开启时有图调用
  不带 response_format（自由描述提示词与 json_object 自相矛盾，joycaption 类模型
  无视甚至可能被该字段干扰），纯文本整理调用保留 json_object。模式标志由
  `newVisionClient(settings)` 传入构造器。Anthropic/Ollama 客户端现状无此字段，不动。
- 整理客户端按任务构建（`newRefineClient(settings)`，镜像 `newVisionClient`），
  超时沿用 `visionTimeout`；测试注入 seam 与 `Service.Client` 同形新增。

### 5.6 重整理入口（`enqueue.go` 语义镜像）

`RefineTask(taskID uint) error`，镜像 `RetryTask`：

- 前置：任务存在、**源任务**有原文行（源 = `SourceTaskID` 非 0 时取其值，否则取
  任务自身——重整理任务可再次重整理，见下）、整理端点已开启（off 时拒绝：
  「未开启整理端点」）、非 live 执行中、该文件无更新的后继在途任务
  （`inFlightTask` 守卫沿用）。
- 允许对 done/failed 的任务重整理（done 且有待审建议时，重整理替换待审，
  与强制重识别替换待审的先例一致）；queued/running 拒绝。
- 动作：`DeletePendingByFile` → 创建新任务行
  `{FileID, Status: queued, SourceTaskID: 源识图任务 id}` → wake + notifyChange。
  旧行永不改写，历史保留；「最新任务拥有待审集合」不变式沿用。
- **重整理任务可再次重整理**：失败的重整理必须能廉价重试，否则只能 RetryTask
  重跑 GPU，违背本功能意图。新任务的 `SourceTaskID` 一律归一化为最初的识图任务
  id，不形成链。`has_raw` 与原文查看接口按同一「源解析」规则判定。

### 5.7 API 与 WebUI

- `POST /api/ai/tasks/:id/refine` → `RefineTask`，respond 惯例。
- 任务 JSON 增加 `source_task_id`、`refine_status`、`has_raw`（是否有原文行，
  列表查询一次聚合，不逐任务 N+1）。
- `GET /api/ai/tasks/:id/raw` → 原文行列表（frame_index + raw_text），供查看。
- 审核页任务区：有原文的任务显示「重新整理」按钮（整理端点 off 时隐藏）；
  任务详情可展开查看逐帧原文（简单文本展示，样式建造者定）；
  `refine_status=failed` 与失败帧/截断注记同区展示。
- 配置页：规则协议自动渲染新字段组，无前端硬编码。
- `FileStatuses` 映射不变：queued/running→running，failed→failed，
  done+pending>0→pending，否则 approved。RefineStatus 不参与文件级状态。
- SSE 推送沿用 `notifyChange`，无新轮询。

### 5.8 版本与变更记录

- core 0.13.0（`internal/constant.CoreVersion`）→ `docs/CHANGELOG.md`。
- webui 5.16.0（`cmd/webui/constant.WebuiVersion`）→ `docs/CHANGELOG_webui.md`。
- 条目格式：英文行 + 中文行 consecutive，不同条目空行分隔，高版本在上。

## 6. 测试要点

- 配置：off 默认、枚举回落、ShowIf 声明序、模型名为空的报错路径。
- 存储：逐帧即写（含畸形 JSON 帧）、传输失败帧无行、重启后原文可重整理、
  DeleteTask 级联删原文、源被删后在途重整理任务的失败原因。
- 管线：整理 off 行为与今天一致（提示词、解析、Error 文案、建议结果、请求体回归）；
  开启时自由描述提示词生效；批量分批与批级部分成功；全批失败任务 failed 且原文完好；
  重整理任务跳过识图校验、FrameCount 沿用源值、执行时端点 off 的失败原因。
- RefineTask：守卫矩阵（无原文/off/在途/queued-running）、待审替换、新任务行
  SourceTaskID 归一化到最初识图任务（重整理任务再重整理不形成链）、旧行不改写。
- OpenAI 客户端：off 模式请求体逐字节照旧（含 response_format）；on 模式有图调用
  无 response_format、纯文本调用有。
- 审核门回归：Approve/Reject/唯一性规则不受影响。

## 7. 留给建造者

- 分批批大小的具体数值与估算方式。
- 自由描述提示词与整理提示词的具体措辞（契约与规则要点已定）。
- 原文查看 UI 的具体样式、`has_raw` 聚合的查询形态。
- 整理客户端测试 seam 的命名与注入方式（镜像现有 `Service.Client`）。
- repo 新文件按 concern 拆分（`frame_output_create.go` 等），遵循链式 builder 惯例。