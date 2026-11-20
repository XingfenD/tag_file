# AI 两段式识别（识图原文落库 + 批量整理 + 只重整理）实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 识图模型自由输出并按帧落库原文，独立文本模型端点批量整理成系统标签 JSON，支持不重跑 GPU 的「只重新整理」。

**Architecture:** 现有单 worker 管线拆成阶段一（逐帧识图、原文即写 `ai_frame_outputs`）与阶段二（批量整理或关闭时直接解析），共用现有 parse/merge/match/入库与审核门。重整理是镜像 RetryTask 的新任务行（`SourceTaskID` 引用源原文）。整理端点默认 off，off 时行为与今天完全一致。

**Tech Stack:** Go 1.25.0、Gin、GORM/SQLite（AutoMigrate）、Alpine.js WebUI、`go test`。

**Spec:** `docs/superpowers/specs/2026-11-20-ai-two-stage-refine-design.md`

## Global Constraints

- 分支 `feat/ai-two-stage-refine`（已建）；禁止在 master 提交；提交前 `git branch --show-current`。
- Schema 只走 GORM `AutoMigrate`，无裸 SQL 迁移。
- 审核门不动：分类必须已存在、根标签全局唯一、Approve/Reject、raw/Final 双份存储。
- 建议写入原子性沿用：`maxTaskSuggestions = 50`，单语句 INSERT。
- 单 worker 串行不变；客户端按任务构建（配置编辑下一任务生效）。
- 配置走块注册表规则协议：封闭枚举、`ShowIf` 单键等值、per-provider 字段按「地址 → 模型 → 密钥」同序声明；文案最小化（只描述非显而易见的事实）。
- WebUI：`respond(c, msg, msgType, extra...)`；DELETE 参数读 `c.Query`/`c.Param`；SSE + 指纹守卫，无轮询；`stampOf` 必须覆盖行渲染读到的每个字段。
- 测试：`make test`；禁止 ad-hoc 测试二进制。core 变更 → `docs/CHANGELOG.md`，webui 变更 → `docs/CHANGELOG_webui.md`；条目英文行+中文行连续、条目间空行、高版本在上；版本常量同 commit 更新。
- off 模式回归零差异：提示词、解析、Error 文案、建议结果、OpenAI 请求体全部照旧。

## Review Focus

1. **off 模式遇到畸形 JSON 帧**（joycaption 常态）：原文仍落库、Error 注记与今天逐字一致、好帧建议不受影响 → Task 5 回归测试钉住。
2. **大帧数视频开启整理**（最多 200 帧）：必须分批，单批输入不撑爆上下文，批间去重走现有 merge → Task 5 分批测试钉住。
3. **源任务被删除后的重整理任务**：执行时以「源任务没有识图原文」失败，不 panic、不孤儿读 → Task 2 级联测试 + Task 5 执行测试钉住。
4. **重复点击「重新整理」/与识别提交并发**：`enqueueMu` 串行化，待审集合不产生双份，在途守卫拒绝后继 → Task 7 守卫测试钉住。
5. **运行中改配置**（整理端点切 off、模型名清空）：在途重整理任务以明确原因失败，原文完好可恢复 → Task 5 执行期校验测试钉住。

## File Structure

**Create:**
- `internal/repository/ai/frame_output_create.go` / `frame_output_list.go` / `frame_output_count.go` — 原文写入/读取/按源任务计数
- `internal/service/ai/refine.go` — 阶段二：分批、整理调用、off 直接解析、注记
- `internal/service/ai/refine_prompt.go` — 自由描述提示词 + 整理提示词
- `internal/service/ai/refine_task.go` — RefineTask 入口（镜像 RetryTask）
- `internal/service/ai/refine_test.go` / `refine_task_test.go` / `refine_prompt_test.go`
- `cmd/webui/api/ai_refine_test.go`

**Modify:**
- `internal/model/ai.go` — AIFrameOutput 新模型；AITask 加 SourceTaskID/RefineStatus
- 所有含 `&model.AITask{}` 的 AutoMigrate 清单（bootstrap/init.go:225 及各测试 setup，`grep -rn "AutoMigrate(" --include="*.go"` 定位）
- `internal/repository/ai/task_delete.go`（级联）、`task_update.go`（SetRefineStatus）
- `internal/service/ai/config.go` / `config_schema.go` — 9 键 + Settings + refineEnabled + checkRefineModel
- `internal/service/ai/openai.go` / `provider.go` — response_format 按模式分流
- `internal/service/ai/prompt.go` — 抽出 renderPromptLibrary（输出逐字节不变）
- `internal/service/ai/recognize.go` — 阶段一/二拆分
- `internal/service/ai/service.go`（RefineClient seam）、`query.go`（两个读透传）、`delete.go`（无需改，级联在 repo）
- `cmd/webui/api/ai.go`（AITaskItem 三字段 + 两个 handler）、`api.go`（路由）
- `cmd/webui/static/js/pages/review/main.js` — 重新整理/原文按钮、指纹、模态
- `internal/constant/version.go`、`cmd/webui/constant/version.go`、两份 CHANGELOG

---

### Task 1: 模型与迁移清单

**Files:**
- Modify: `internal/model/ai.go`
- Modify: 所有含 `&model.AITask{}` 的 AutoMigrate 清单（bootstrap/init.go:225、internal/service/ai/ai_service_test.go:21、parse_test.go:39、internal/repository/ai/ai_repo_test.go:19、cmd/webui/api/router_test.go:30、pkg/storage/sqlite_test.go:22，以及 grep 出的其余）
- Test: `internal/repository/ai/ai_repo_test.go`

**Interfaces:**
- Produces: `model.AIFrameOutput{OutputID uint pk, TaskID uint index, FileID uint index, FrameIndex int, RawText string type:text, CreatedAt time.Time}`，TableName `ai_frame_outputs`；`model.AITask` 新字段 `SourceTaskID uint`（column `source_task_id`，default 0）、`RefineStatus string`（column `refine_status`，size:20，default ''），json tag `source_task_id`/`refine_status`。

- [ ] **Step 1: 写失败测试** `TestAutoMigrate_FrameOutputTable`（ai_repo_test.go）：setup 后 `storage.GetDB().Migrator().HasTable(&model.AIFrameOutput{})` 为 true；`HasColumn(&model.AITask{}, "source_task_id")`、`HasColumn(&model.AITask{}, "refine_status")` 为 true。
- [ ] **Step 2: 运行确认失败** `go test ./internal/repository/ai/ -run TestAutoMigrate_FrameOutputTable -v` → FAIL（表/列不存在）。
- [ ] **Step 3: 实现**：按 Interfaces 的字段与 tag 写模型（gorm tag 风格照抄同文件 AITask）；把 `&model.AIFrameOutput{}` 加进每一处含 AITask 的 AutoMigrate 清单。
- [ ] **Step 4: 运行确认通过** 同 Step 2 命令 → PASS；`go build ./...` 通过。
- [ ] **Step 5: Commit** `feat(ai): add frame-output model and task refine columns`

### Task 2: 原文仓储 + 任务删除级联 + RefineStatus 写入

**Files:**
- Create: `internal/repository/ai/frame_output_create.go`、`frame_output_list.go`、`frame_output_count.go`
- Modify: `internal/repository/ai/task_delete.go`、`task_update.go`
- Test: `internal/repository/ai/ai_repo_test.go`

**Interfaces:**
- Consumes: Task 1 模型。
- Produces:
  - `(r *Repository) CreateFrameOutput(o *model.AIFrameOutput) error`（回填 OutputID）
  - `(r *Repository) ListFrameOutputs(taskID uint) ([]model.AIFrameOutput, error)`（按 frame_index 升序，无行返回空切片）
  - `(r *Repository) RawCountsBySource(sourceIDs []uint) (map[uint]int64, error)`（`SELECT task_id, COUNT(*) ... WHERE task_id IN ? GROUP BY task_id`；空入参返回空 map；无行任务不在 map 中）
  - `(*TaskUpdateOperation) SetRefineStatus(v string) *TaskUpdateOperation`（cols 白名单同形）
  - `DeleteTask` 事务内追加 `tx.Where("task_id = ?", id).Delete(&model.AIFrameOutput{})`（仅 affected>0 分支）。

- [ ] **Step 1: 写失败测试**（ai_repo_test.go，setup 沿用该文件既有 helper）：
  - `TestFrameOutput_CreateAndList`：插 3 行（FrameIndex 乱序 3,1,2），List 返回按 1,2,3 排序、RawText 逐字；空任务 List 返回非 nil 空切片。
  - `TestRawCountsBySource`：两任务各 2/1 行，map 计数正确；缺席任务不在 map；空入参 → 空 map。
  - `TestDeleteTask_CascadesFrameOutputs`：任务+原文+pending 建议+approved 建议，DeleteTask 后原文与 pending 消失、approved 留存；条件删除（status 不匹配）时原文不动。
  - `TestUpdateTask_SetRefineStatus`：只写 refine_status 列，status/error 不被覆盖。
- [ ] **Step 2: 运行确认失败** `go test ./internal/repository/ai/ -run "TestFrameOutput|TestRawCounts|TestDeleteTask_Cascades|TestUpdateTask_SetRefineStatus" -v` → FAIL（方法不存在）。
- [ ] **Step 3: 实现** 四个仓储方法 + 级联 + setter，签名照 Interfaces；文件按 concern 拆分，风格照 task_create.go/task_list.go。
- [ ] **Step 4: 运行确认通过** 同 Step 2 → PASS。
- [ ] **Step 5: Commit** `feat(ai): frame-output repository, cascade delete, refine-status setter`

### Task 3: 整理端点配置（9 键）

**Files:**
- Modify: `internal/service/ai/config.go`、`config_schema.go`
- Test: `internal/service/ai/config_test.go`、`config_schema_test.go`

**Interfaces:**
- Consumes: 现有 `stringSetting`/`providerSetting` 模式、schema 注册表。
- Produces:
  - 键常量：`KeyRefineProvider = "ai_refine_provider"`；`KeyRefineOllamaURL/Model = "ai_refine_ollama_url"/"ai_refine_ollama_model"`；`KeyRefineOpenAIURL/Model/APIKey = "ai_refine_openai_url"/"ai_refine_openai_model"/"ai_refine_openai_api_key"`；`KeyRefineAnthropicURL/Model/APIKey` 同形。
  - 枚举值复用 `ProviderOllama/OpenAI/Anthropic` + `RefineOff = "off"`；`DefaultRefineProvider = RefineOff`；URL 默认值与识图组相同。
  - `Settings` 新字段：`RefineProvider string`、`RefineOllamaURL/RefineOllamaModel string`、`RefineOpenAIURL/RefineOpenAIModel/RefineOpenAIAPIKey string`、`RefineAnthropicURL/RefineAnthropicModel/RefineAnthropicAPIKey string`。
  - `refineEnabled(settings Settings) bool`（`RefineProvider != RefineOff`）。
  - `checkRefineModel(settings Settings) error`：off → nil；否则镜像 checkProviderModel，错误文案「未配置整理端点的 %s 模型名, 请先在配置页填写」。
  - schema：ai 块内、识图 provider 组之后、抽帧字段之前，按「枚举 → ollama 地址/模型 → openai 地址/模型/密钥 → anthropic 地址/模型/密钥」声明序注册 9 个 InputBox；枚举 Label「整理端点类型」，Options：off=「关闭」、ollama=「Ollama」、openai=「OpenAI 兼容」、anthropic=「Anthropic」；per-provider 字段 `ShowIf{Key: KeyRefineProvider, Equals: ...}`；openai 地址 Description 照抄识图组（/v1 前缀），密钥 `Secret: true`，地址 `Required+FormatURL`。

- [ ] **Step 1: 写失败测试**：
  - config_test.go `TestSettings_RefineDefaults`：fresh DB → RefineProvider=off、URL 默认值；`TestSettings_RefineProviderFallback`：写非法值 → 回落 off（日志同形）；`TestRefineEnabled`、`TestCheckRefineModel`（off→nil；openai 空模型→错误文案含「整理端点」；填模型→nil）。配置写入用 `configSvc.NewService().Set(key, value)`（该文件既有模式）。
  - config_schema_test.go：ai 块含 9 个新键；声明序为枚举→地址→模型→密钥；ShowIf 挂 KeyRefineProvider；off 为枚举首项且默认。
- [ ] **Step 2: 运行确认失败** `go test ./internal/service/ai/ -run "Refine" -v` → FAIL。
- [ ] **Step 3: 实现** 常量/Settings/读取/校验/注册，照 Interfaces；`refineProviderSetting()` 镜像 providerSetting 的封闭枚举回落。bootstrap 种子由注册表驱动（TestSeedCoversRegistry 自动覆盖新键），无需改 bootstrap。
- [ ] **Step 4: 运行确认通过** Step 2 命令 + `go test ./internal/bootstrap/ -v` → PASS。
- [ ] **Step 5: Commit** `feat(ai): independent refine-endpoint config group (default off)`

### Task 4: OpenAI response_format 按模式分流

**Files:**
- Modify: `internal/service/ai/openai.go`、`provider.go`、`openai_test.go`、`client_internal_test.go`（凡 NewOpenAIClient 调用点）
- Test: `internal/service/ai/openai_test.go`

**Interfaces:**
- Consumes: Task 3 `refineEnabled`。
- Produces: `NewOpenAIClient(baseURL, model, apiKey string, refineMode bool) VisionClient`；`newVisionClient(settings)` 传 `refineEnabled(settings)`。请求体规则：`refineMode==false` → 与今天逐字节一致（含 `response_format: json_object`）；`refineMode==true` 且 len(images)>0 → 不带 response_format 字段（结构体字段改 `*openAIResponseFormat` + omitempty，或等价手段）；`refineMode==true` 且无图 → 带 json_object。Anthropic/Ollama 不动（已支持无图调用）。

- [ ] **Step 1: 写失败测试**（openai_test.go，httptest 捕获请求体，沿用该文件既有手法）：
  - `TestOpenAI_RefineOffKeepsResponseFormat`：refineMode=false + 1 图 → body 含 `"response_format":{"type":"json_object"}`（现有断言迁移）。
  - `TestOpenAI_RefineOnImageOmitsResponseFormat`：true + 1 图 → body 不含 `response_format` 键。
  - `TestOpenAI_RefineOnTextOnlyKeepsResponseFormat`：true + nil 图 → 含 json_object，content 仅一个 text part。
- [ ] **Step 2: 运行确认失败** `go test ./internal/service/ai/ -run TestOpenAI_Refine -v` → FAIL（签名/行为）。
- [ ] **Step 3: 实现** 构造器加参 + 条件 response_format；更新 provider.go 与全部调用点（grep `NewOpenAIClient(`）。
- [ ] **Step 4: 运行确认通过** Step 2 命令 + `go test ./internal/service/ai/ -v` 全绿 → PASS。
- [ ] **Step 5: Commit** `feat(ai): scope OpenAI response_format to refine mode`

### Task 5: 提示词拆分（自由描述 + 整理提示词）

**Files:**
- Modify: `internal/service/ai/prompt.go`
- Create: `internal/service/ai/refine_prompt.go`
- Test: `internal/service/ai/refine_prompt_test.go`（prompt 既有测试若在别处，保持全绿）

**Interfaces:**
- Consumes: 现有 BuildPrompt 的库渲染、`promptContract`、`model.AIFrameOutput`。
- Produces:
  - `renderPromptLibrary() (string, error)`：从 BuildPrompt 抽出的库渲染（分类清单+标签树+预设值），**输出逐字节等于今天 BuildPrompt 的对应段落**；BuildPrompt 改为调用它，整体输出不变。
  - `visionFreePrompt` 常量：自由描述提示词，措辞要点「观察图片, 描述画面内容并列出你看到的标签, 自由文本即可, 不需要任何格式」。
  - `(s *Service) BuildRefinePrompt(frames []model.AIFrameOutput) (string, error)`：renderPromptLibrary + 按「第 %d 帧:」逐帧附 RawText（FrameIndex 原值）+ 整理规则。
  - 整理规则要点（文案建造者润色，语义钉死）：只输出 JSON、格式固定 `promptContract`；category/tag 逐字用库内名字、同义词归一到库内已有标签（如「沙滩」→「海滩」）；跨帧重复语义合并成一条；value 优先预设值否则空串；只基于原文整理、不得凭空添加原文没有的内容；确实无合适库标签时可提新词但不重复已有标签。

- [ ] **Step 1: 写失败测试**：
  - `TestBuildPrompt_UnchangedAfterExtract`：seed 分类/标签/预设值（沿用 prompt 既有测试的 seed 手法），断言 BuildPrompt 输出与抽取前捕获的期望串逐字一致（期望串在测试里以现有常量拼出，或直接断言含 promptHead/promptRules 与库段落且无整理字样）。
  - `TestBuildRefinePrompt`：seed 库 + 2 帧原文（FrameIndex 2,5）→ 输出含分类名、标签路径、「第 2 帧:」「第 5 帧:」、两帧原文逐字、promptContract、同义词归一与「不得凭空」规则要点。
  - `TestVisionFreePrompt`：不含 promptContract、不含「JSON」。
- [ ] **Step 2: 运行确认失败** `go test ./internal/service/ai/ -run "Prompt" -v` → 新测试 FAIL。
- [ ] **Step 3: 实现** 抽取 + 两个新提示词，照 Interfaces。
- [ ] **Step 4: 运行确认通过** Step 2 命令全绿（含既有 prompt 测试）→ PASS。
- [ ] **Step 5: Commit** `feat(ai): free-form vision prompt and batch refine prompt`

### Task 6: 管线拆分（阶段一落库 + 阶段二批量/直接解析）

**Files:**
- Modify: `internal/service/ai/recognize.go`、`service.go`、`provider.go`
- Create: `internal/service/ai/refine.go`
- Test: `internal/service/ai/refine_test.go`

**Interfaces:**
- Consumes: Task 2 仓储、Task 3 配置、Task 4 客户端、Task 5 提示词；现有 `ParseSuggestions`/`mergeRawTags`/`buildSuggestions`/`maxTaskSuggestions`/`frameIndexList`、fakeClient/perFrameClient（worker_test.go）。
- Produces:
  - `Service.RefineClient VisionClient` 导出 seam（NewService 留 nil，镜像 Client）；`newRefineClient(settings Settings) VisionClient`（镜像 newVisionClient，读 Refine* 字段，refineMode 恒传 true）。
  - `refineBatchSize = 20`（常量，refine.go）。
  - 阶段二输入统一从库读：普通任务 `ListFrameOutputs(task.TaskID)`，重整理任务读源任务——不在内存里传递原文。
  - `recognize(task)` 分流：`task.SourceTaskID != 0` → `recognizeRefineTask`（跳过识图与 checkProviderModel；FindTask 源任务，nil 或无原文行 → 「源任务没有识图原文」；FrameCount 沿用源值）；否则阶段一。
  - 阶段一：settings → checkProviderModel → client → loadFrames → 提示词分流（refineEnabled → visionFreePrompt，否则 BuildPrompt）→ 逐帧 Recognize，**成功即 `CreateFrameOutput`（无论 JSON 是否合法）**，传输失败帧记 failedFrames（注记文案照旧）；全帧皆败 → failed；有成功帧 → 阶段二。
  - 阶段二 `refineStage(task, outputs []model.AIFrameOutput, settings) error`（refine.go）：
    - off：逐帧 ParseSuggestions；解析失败帧记入 failedFrames，**注记与今天逐字同形**「第 %s 帧识别失败(%d/%d), 首条原因: %v」（frameIndexList）；全失败 → 「全部 %d 帧识别失败, 首条原因: %w」；成功帧 mergeRawTags。
    - on：执行期校验 `refineEnabled`（off → 「整理端点未开启」）+ `checkRefineModel`；RefineStatus 置 pending；按 refineBatchSize 切批，每批 BuildRefinePrompt + `RefineClient.Recognize(ctx, prompt, nil)` + ParseSuggestions；一批失败只丢该批，注记「第 %s 帧整理失败(%d/%d), 首条原因: %v」（帧号列表用该批首尾帧的 FrameIndex）；全批皆败 → failed + 「整理失败, 可重新整理」；跨批 mergeRawTags。
    - 共同尾部：50 条截断（注记文案照旧）→ buildSuggestions → CreateSuggestions → 终态写入（SetStatus/SetFrameCount/SetError/SetRefineStatus；off 时 RefineStatus 恒 ''，on 成功 done、失败 failed）。
  - 部分成功注记合并沿用 `strings.Join(notes, "；")`。

- [ ] **Step 1: 写失败测试**（refine_test.go，setupTestDB + seedFile + fakeExtractor/fakeClient/perFrameClient，worker 驱动沿用 worker_test.go 的 StartWorker/StopWorker 手法；配置用 configSvc.Set）：
  - `TestRecognize_OffModeRegression`：perFrameClient 帧 1 合法 JSON、帧 2 畸形 → 任务 done、建议只含帧 1、Error 逐字等于「第 2 帧识别失败(1/2), 首条原因: …」（首条原因用 ParseSuggestions 对该畸形串的实际错误断言前缀）；**原文 2 行都在库**（含畸形帧）。
  - `TestRecognize_OnModeFreePromptAndRefine`：开启整理（openai+模型名），fakeClient 返回自由文本、RefineClient 返回合法 tags JSON → 识图收到的 prompt 不含 promptContract、整理收到的 prompt 含标签库与帧原文；建议来自整理输出；RefineStatus=done；原文落库。
  - `TestRecognize_OnModeChunking`：fakeExtractor 给 25 帧、整理 fake 按调用计数返回不同 tags → 整理调用 2 次（20+5）、每次 prompt 只含本批帧号、建议为两批合并去重。
  - `TestRecognize_OnModeChunkPartialFailure`：30 帧、第 2 批整理报错 → done、建议含第 1 批、Error 含「整理失败」帧段注记、RefineStatus=done。
  - `TestRecognize_OnModeAllChunksFail`：整理全报错 → failed、Error 含「整理失败, 可重新整理」、**原文完好**、RefineStatus=failed。
  - `TestRecognize_RefineTaskSkipsVision`：手建 SourceTaskID 任务+源原文 → 识图 fake 零调用、建议产出、FrameCount=源值。
  - `TestRecognize_RefineTaskSourceGone`：SourceTaskID 指向不存在任务 → failed「源任务没有识图原文」。
  - `TestRecognize_RefineTaskEndpointOffAtExecution`：建任务后把 ai_refine_provider 设 off → failed「整理端点未开启」。
- [ ] **Step 2: 运行确认失败** `go test ./internal/service/ai/ -run TestRecognize_ -v` → FAIL。
- [ ] **Step 3: 实现** recognize 分流 + refine.go + seam + newRefineClient，照 Interfaces。off 路径的注记重建是回归关键：failedFrames/firstErr 的聚合顺序与今天 recognize 循环一致（帧序遍历、首条原因保留 %w 归因链）。
- [ ] **Step 4: 运行确认通过** Step 2 命令 + `go test ./internal/service/ai/ -v` 全绿（既有 worker/parse/merge 测试不得回归）→ PASS。
- [ ] **Step 5: Commit** `feat(ai): split recognition into stored-vision and batch-refine stages`

### Task 7: RefineTask 入口

**Files:**
- Create: `internal/service/ai/refine_task.go`
- Test: `internal/service/ai/refine_task_test.go`

**Interfaces:**
- Consumes: Task 2/3/6；现有 `enqueueMu`/`liveTask`/`inFlightTask`/`DeletePendingByFile`/`notifyWake`/`notifyChange`。
- Produces: `(s *Service) RefineTask(taskID uint) error`，语义镜像 RetryTask：
  - enqueueMu 全程；FindTask → nil 报「识别任务 %d 不存在」。
  - 状态守卫：仅 done/failed 可重整理；queued/running 报「仅已完成或失败的任务可重新整理,当前状态 '%s'」；running 且 liveTask 命中报「正在执行中」（同 RetryTask 文案）。
  - 源解析：`sourceID := task.TaskID; if task.SourceTaskID != 0 { sourceID = task.SourceTaskID }`；源任务不存在或 `ListFrameOutputs(sourceID)` 为空 → 「该任务没有识图原文, 无法重新整理」。
  - `refineEnabled(s.Settings())` 为 false → 「未开启整理端点, 请先在配置页配置」。
  - `inFlightTask(task.FileID, task.TaskID)` 非 nil → 「该文件已有进行中的识别任务 %d,请勿重复提交」（照抄）。
  - 动作：DeletePendingByFile → CreateTask `{FileID: task.FileID, Status: TaskQueued, SourceTaskID: sourceID}`（**归一化到最初识图任务，不链式**）→ notifyWake + notifyChange。旧行永不改写。

- [ ] **Step 1: 写失败测试**（refine_task_test.go）：
  - `TestRefineTask_Guards`：矩阵——不存在 id；queued 任务；running+live；无原文的 done 任务；off 配置；文件有更新在途任务 → 各自错误文案断言。
  - `TestRefineTask_ReplacesPendingAndKeepsHistory`：done 任务+原文+2 条 pending → RefineTask 后 pending 清空、新 queued 任务 SourceTaskID=源 id、旧任务行状态/建议历史不变。
  - `TestRefineTask_NormalizesSourceChain`：对 SourceTaskID=A 的重整理任务再 RefineTask → 新任务 SourceTaskID 仍=A。
  - `TestRefineTask_ConcurrentClicks`：两 goroutine 并发 RefineTask 同一任务 → 恰一个成功（另一个被在途守卫拒绝），queued 行只 1 条。
- [ ] **Step 2: 运行确认失败** `go test ./internal/service/ai/ -run TestRefineTask_ -v` → FAIL。
- [ ] **Step 3: 实现** refine_task.go，照 Interfaces。
- [ ] **Step 4: 运行确认通过** Step 2 命令 → PASS。
- [ ] **Step 5: Commit** `feat(ai): RefineTask entry mirroring RetryTask semantics`

### Task 8: WebUI API（任务字段 + refine/raw 端点）

**Files:**
- Modify: `cmd/webui/api/ai.go`、`api.go`、`internal/service/ai/query.go`
- Test: `cmd/webui/api/ai_refine_test.go`

**Interfaces:**
- Consumes: Task 2/7；现有 `respond`/`parsePagination`/`filesOf`、router_test.go 的 newTestEngine 手法。
- Produces:
  - `AITaskItem` 增 `SourceTaskID uint json:"source_task_id"`、`RefineStatus string json:"refine_status"`、`HasRaw bool json:"has_raw"`。
  - HandleListAITasks：收集页内任务的 sourceID（同 Task 7 解析规则）→ `AISvc.RawCountsBySource(ids)` → `HasRaw = counts[sourceID] > 0`（一次聚合，无 N+1）。
  - query.go 透传：`(s *Service) ListFrameOutputs(taskID uint) ([]model.AIFrameOutput, error)`、`(s *Service) RawCountsBySource(sourceIDs []uint) (map[uint]int64, error)`（纯委托，风格照现有三个）。
  - `HandleRefineAITask`：POST `/api/ai/tasks/refine`，PostForm `task_id`（镜像 HandleRetryAITask），成功文案「任务已加入重新整理队列」。
  - `HandleAITaskRaw`：GET `/api/ai/tasks/:id/raw`，Param id；源解析后 `ListFrameOutputs(sourceID)`；响应 `{"ok":true,"msg":"ok","items":[{"frame_index":N,"raw_text":"…"}]}`（items 非 nil）；任务不存在 → 「识别任务 %d 不存在」error。
  - api.go 路由组追加两条（POST refine 在 DELETE :id 之前注册，与 retry 同形）。

- [ ] **Step 1: 写失败测试**（ai_refine_test.go，newTestEngine）：
  - `TestListAITasks_RefineFields`：seed 任务+原文 → 列表项含 source_task_id/refine_status/has_raw=true；无原文任务 has_raw=false；重整理任务 has_raw 按源解析。
  - `TestRefineEndpoint`：合法请求 → ok + 新 queued 任务；非法 task_id → 400「无效的任务ID」；守卫错误 → 400 + 服务文案。
  - `TestRawEndpoint`：GET 返回按帧序的 items；重整理任务 id 返回源的原文；不存在 id → 400。
- [ ] **Step 2: 运行确认失败** `go test ./cmd/webui/api/ -run "TestListAITasks_RefineFields|TestRefineEndpoint|TestRawEndpoint" -v` → FAIL。
- [ ] **Step 3: 实现** 字段/透传/handler/路由，照 Interfaces。
- [ ] **Step 4: 运行确认通过** Step 2 命令 + `go test ./cmd/webui/api/ -v` 全绿 → PASS。
- [ ] **Step 5: Commit** `feat(webui): refine/raw endpoints and task refine fields`

### Task 9: 审核页 UI（重新整理 / 原文查看 / 整理状态）

**Files:**
- Modify: `cmd/webui/static/js/pages/review/main.js`
- Test: 浏览器手工验证（`go run ./cmd/webui --source <临时目录> --addr :8088`，配置页开整理端点指向 fake/真实服务；无模型时用 API 直接 seed 任务+原文验证按钮与模态）

**Interfaces:**
- Consumes: Task 8 端点与字段；现有 `delegate`/`Modal.show`/`API.post`/`reportResult`/`confirmDelete`/`loadPaged`。
- Produces（全部在 reviewPage 组件内）：
  - `LISTS.tasks.stampOf` 追加 `t.source_task_id, t.refine_status || '', t.has_raw ? 1 : 0`（指纹覆盖新渲染字段）。
  - `taskRowsHtml` 操作列：`(t.has_raw && (t.status === 'done' || t.status === 'failed'))` 时渲染「重新整理」按钮（data-action="refine"）与「原文」按钮（data-action="task-raw"），位于重试/删除之前；按钮显隐完全由 has_raw 驱动（服务端已含 off 判定），前端不读配置。
  - `taskBadgeHtml`：`t.refine_status === 'failed'` 时主徽章后追加 `<span class="badge badge-red">整理失败</span>`。
  - delegate 增 `refine` → `refineTask(id)`：`API.post('/api/ai/tasks/refine', {task_id})` + reportResult（成功文案「任务已加入重新整理队列」）+ refresh；`task-raw` → `showTaskRaw(t)`。
  - `showTaskRaw`：`API.get('/api/ai/tasks/' + id + '/raw')` → Modal.show（title「任务 #N · 识图原文」，size lg，每帧一个 static 字段 label「第 N 帧」value=raw_text，saveLabel 关闭）；空 items → 单字段「(无原文)」。
  - 不新增轮询、不新增刷新按钮；SSE 推送沿用。

- [ ] **Step 1: 实现** 上述改动。
- [ ] **Step 2: 浏览器验证 golden path**：配置页出现「整理端点类型」及条件字段组（切 provider 显隐正确）；审核页有原文任务显示两按钮；点「原文」模态按帧展示；点「重新整理」→ toast + 新任务行出现（SSE 推送，无手动刷新）；整理失败任务显示「整理失败」徽章。
- [ ] **Step 3: 边界验证**：off 配置下无「重新整理」按钮；queued/running 行无按钮；删除任务后其原文模态不再可达（行已消失）；空闲页面零请求零重画（Network 面板核对指纹守卫）。
- [ ] **Step 4: 回归**：重试/删除/建议审核/分页/SSE 断线重连照常。
- [ ] **Step 5: Commit** `feat(webui): re-refine and raw-output actions on review page`

### Task 10: 版本、CHANGELOG、全量验证

**Files:**
- Modify: `internal/constant/version.go`（0.12.0 → 0.13.0）、`cmd/webui/constant/version.go`（5.15.0 → 5.16.0）、`docs/CHANGELOG.md`、`docs/CHANGELOG_webui.md`

**Interfaces:**
- Consumes: 全部前置任务。
- Produces: 版本常量 + 两份 CHANGELOG 顶部条目（英文行+中文行连续，条目间空行）。core 条目涵盖：两段式管线、原文落库、批量整理、RefineTask、整理端点配置、response_format 分流；webui 条目涵盖：refine/raw 端点、任务新字段、审核页按钮与模态、配置组。

- [ ] **Step 1: 更新版本常量与 CHANGELOG**（格式照文件既有条目）。
- [ ] **Step 2: 全量测试** `make test` → 全绿。
- [ ] **Step 3: 构建** `make build && make build-webui` → 两个产物生成。
- [ ] **Step 4: 端到端冒烟**（有真实/fake 端点时）：off 模式识别一个文件 → 行为照旧；开整理 → 建议来自整理输出；失败后「重新整理」不重跑识图（识图调用计数不增）。
- [ ] **Step 5: Commit** `chore: bump core 0.13.0 / webui 5.16.0 with changelogs`