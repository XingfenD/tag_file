# Ollama 视觉识别自动打标 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 WebUI 上对已注册的图片/视频批量发起 Ollama 视觉识别,产出的标签建议全部进入人工待审队列,审核通过后才建标签并带值关联文件。

**Architecture:** 新增 `ai_tasks` / `ai_suggestions` 两张表(AutoMigrate)。`internal/service/ai` 承载全部业务:入队与跳过规则、单 worker 顺序消费、ffmpeg 抽帧 + dHash 去近似帧、Ollama HTTP 调用、提示词构建与输出解析、审核入库、文件级 AI 状态推导。ffmpeg 与 Ollama 各自藏在 `FrameExtractor` / `VisionClient` 接口后,测试用假实现替换,因此 `make test` 不依赖真实媒体或模型服务。`cmd/webui` 只做薄壳:新增 `/api/ai/*` 端点、`/review` 页面、配置卡片与文件树/详情徽章。

**Tech Stack:** Go 1.25.0、GORM + glebarez/sqlite、Gin v1.12.0、Alpine.js(内嵌模板 + `static/js`)、stdlib `net/http` / `os/exec` / `image` / `encoding/base64`、外部 ffmpeg。**不新增任何 go.mod 依赖。**

**Spec:** `docs/superpowers/specs/2026-11-13-ollama-vision-tagging-design.md`

## Global Constraints

以下为 spec 与仓库的硬约束,每个任务的要求都隐含包含本节。

- 分支 `feat/ollama-vision-tagging`;**禁止在 `master` 提交**。提交前 `git branch --show-current`。
- schema 只走 `AutoMigrate`,禁止裸 SQL 迁移。DB 路径 `.tagfile/tagfile.db`。
- 模型只提建议,**任何模型输出在人工通过前不得出现在标签库或文件标签中**;不自动应用,哪怕置信度高。
- 审核通过的新值只写 `TagGroupTag.TagValue`(字段名是 `TagValue`,**不是** `Value`);**绝不写预设值字典 `TagValue` 表**。
- 新标签创建必须走 `tagSvc.AddTag(categoryName, tagName, description, parentTagName)`;打标必须走 `fileSvc.TagFile(filePath, tagID, tagValue)`(它内部维护 `TagCnt`)。不得绕过 service 层唯一性校验。
- `File.TagDone` 语义不变、纯手动;AI 状态是独立维度。
- 描述文字丢弃,只留标签。不做 CLI 入口、不做向量检索、不做音频/人脸。
- 单 worker 顺序处理,默认并发 1;长耗时识别不得阻塞 WebUI 请求。
- 视频:抽帧默认 **1 帧/秒**,最大帧数默认 **10**,差异过滤后截断。抽帧间隔/差异阈值/最大帧数/Ollama 地址/模型名全部走现有 config 服务。
- 重启语义:`running` → `failed`(原因「重启中断」,可手动重试);`queued` 在 worker 启动后自动继续。
- 重复发起:默认跳过已有待审建议或已完成识别的文件;单文件显式「重新识别」时先清掉该文件旧的 pending 建议再入队。
- 核心改动(`internal/`、`pkg/`)记 `docs/CHANGELOG.md` 并 bump `internal/constant.CoreVersion`;WebUI 改动(`cmd/webui/`)记 `docs/CHANGELOG_webui.md` 并 bump `cmd/webui/constant.WebuiVersion`,**同 commit 完成**。同条目英文行紧接中文行(中间无空行),不同条目之间空行,高版本在上。
- 测试:`make test`(= build CLI → `scripts/setup_test_workspace.sh` → `go test ./... -v`)。stdlib `testing`,外部测试包 `package xxx_test`,无 testify、无 `t.Parallel()`。**禁止构建临时测试二进制。**
- 环境事实(与 spec/AGENTS.md 的陈述不符,以此为准):`go.mod:3` = `go 1.25.0`;本机 `ffmpeg 4.4.2`(spec §6 写的 8.0.1 不成立)——只用 4.x 就有的参数(`-vf fps=`、`-frames:v`、`-y`、`-f image2`)。
- 现有测试的 `setupTestDB` 在 3 个文件里各有一份且**都不含 `&model.Config{}`**:`internal/service/{file,tag,category}/*_service_test.go:20-22`。新增表必须同步补进去,否则 DB 测试报 no such table。
- 同一测试包内共用 `.tag_file_test.db` 与全局 `storage.db`,**测试不得并行**,worker 测试必须在 cleanup 里 `StopWorker()` 并等待 goroutine 退出。
- Gin 路由:`/api/files` 组内已有 `/*path` 通配,**不得**再加 `/api/files/:id` 或 `/api/files/*path` 兄弟路由(见 commit `34bf78c`)。AI 端点一律放新组 `/api/ai`。
- API 约定:响应统一 `respond(c, msg, "success"|"error", map[string]any{...})`,错误一律 HTTP 400;**请求体一律 `application/x-www-form-urlencoded`,用 `c.PostForm` / `c.PostFormArray`,全仓库没有 JSON 请求体**。
- 前端:`{{define}}` 不能嵌套,页面只定义 `content`;树/列表用 JS 字符串 + `x-html` 渲染,插值必须 `escHtml(...)`;新页面必须同时加进 `cmd/webui/main.go:64` 的页面名切片、`handlers.RegisterRoutes`、`base.html` 导航。

## Review Focus

spec 隐含、但常规路径测试不会覆盖的五类输入/失败模式,按最可能咬人排序。每条都在归属任务里补了对应测试。

1. **模型输出不是干净 JSON**(散文、```json 围栏、空 `tags`、截断)——用户期望任务显式失败并看到原因,而不是静默产出 0 条建议还标成「完成」。→ Task 7 解析测试 + Task 8 worker 失败落库测试。
2. **同一建议被通过两次 / 批量通过里 ID 重叠 / 同一文件重复入队**——用户期望不产生重复关联、`TagCnt` 不漂移,第二次是明确的无操作或报错。→ Task 4 入队跳过测试 + Task 9 重复通过测试。
3. **模型编造不存在的分类,或标签名与另一分类下的同名标签撞车**——用户期望审核时能改分类并成功入库,不得跨分类建重复标签、不得 panic。→ Task 9 分类缺失/同名跨分类测试。
4. **视频抽不出帧**:ffmpeg 缺失、文件损坏、注册后文件被删、帧数少于最大帧数——用户期望任务失败且错误可操作、可重试,不留下半截建议。→ Task 5 抽帧错误测试 + Task 8 失败可重试测试。
5. **进程被杀后重启,以及 Ollama 不可达/挂死**——用户期望 running 变「重启中断」可重试、queued 自动续跑,且 HTTP 超时兜住 worker,队列不会永久卡住。→ Task 6 超时测试 + Task 8 重启语义测试。

---

## File Structure

**新增 — 核心层**

| 文件 | 职责 |
|---|---|
| `internal/model/ai.go` | `AITask` / `AISuggestion` 两个 GORM 模型 + TableName |
| `internal/repository/ai/repo.go` | `Repository` + `NewRepository()` |
| `internal/repository/ai/task_create.go` / `task_find.go` / `task_list.go` / `task_update.go` | 任务 CRUD(链式 builder 用于 list/update) |
| `internal/repository/ai/task_claim.go` | `ClaimNextQueued()` 抢占式置 running、`FailRunning(reason)` 重启清扫 |
| `internal/repository/ai/suggestion_create.go` / `suggestion_find.go` / `suggestion_list.go` / `suggestion_update.go` / `suggestion_delete.go` / `suggestion_count.go` | 建议 CRUD + 按文件统计 pending |
| `internal/service/ai/service.go` | `Service` 结构、`NewService()`、依赖装配 |
| `internal/service/ai/ports.go` | `FrameExtractor` / `VisionClient` 接口 |
| `internal/service/ai/config.go` | config key 常量、`Settings` 与带默认值的读取 |
| `internal/service/ai/media.go` | 扩展名判定图片/视频 |
| `internal/service/ai/enqueue.go` | 入队、跳过规则、重新识别、重试 |
| `internal/service/ai/frames.go` | ffmpeg 抽帧适配器 |
| `internal/service/ai/diff.go` | dHash + 近似帧过滤(纯函数) |
| `internal/service/ai/ollama.go` | Ollama `/api/chat` 客户端 |
| `internal/service/ai/prompt.go` | 提示词构建(分类 + 标签树 + 预设值) |
| `internal/service/ai/parse.go` | 模型输出解析 + 命中已有标签 |
| `internal/service/ai/recognize.go` | 单文件识别流水线编排 |
| `internal/service/ai/worker.go` | 单 worker 生命周期 |
| `internal/service/ai/review.go` | 改/通过/拒绝/批量、建标签去重 |
| `internal/service/ai/status.go` | 状态常量 + 文件级 AI 状态推导 |

**新增 — WebUI 层**:`cmd/webui/api/ai.go`、`cmd/webui/templates/review.html`、`cmd/webui/static/js/pages/review/main.js`。

**修改**:`internal/bootstrap/init.go`(AutoMigrate + 配置种子)、3 份既有 `setupTestDB`、`cmd/webui/main.go`、`cmd/webui/api/api.go`、`cmd/webui/api/file.go`、`cmd/webui/handlers/handlers.go`、`cmd/webui/templates/base.html`、`config.html`、`file_detail.html`、`static/js/pages/config/main.js`、`static/js/pages/file/{tree,detail}.js`、`static/css/components/{tags,tree}.css`、两个 version.go、两份 CHANGELOG、`AGENTS.md`(纠正过期陈述)。

---

### Task 1: 数据模型与迁移

**Files:**
- Create: `internal/model/ai.go`
- Create: `internal/service/ai/ai_service_test.go`(本包共用的 `setupTestDB` 与种子helper,后续任务都往这里加)
- Modify: `internal/bootstrap/init.go:89-99`(AutoMigrate 列表)
- Modify: `internal/service/file/file_service_test.go:20-22`、`internal/service/tag/tag_service_test.go:20-22`、`internal/service/category/category_service_test.go:20-22`

**Interfaces:**
- Consumes: 现有 `model.File`(`FileID uint`)、`storage.GetDB()`。
- Produces: `model.AITask{TaskID, FileID, Status, Error, FrameCount, CreatedAt, UpdatedAt, File}`、`model.AISuggestion{SuggestionID, TaskID, FileID, CategoryName, TagName, TagValue, FinalCategoryName, FinalTagName, FinalValue, MatchedTagID *uint, ApprovedTagID *uint, ReviewStatus, ReviewedAt *time.Time, CreatedAt}`。表名 `ai_tasks` / `ai_suggestions`。`CategoryName/TagName/TagValue` 是模型原始输出(永不改写),`Final*` 是生效值(插入时等于原始输出,审核编辑只改 `Final*`)。

- [ ] **Step 1: 写失败测试** — `internal/service/ai/ai_service_test.go`,`package ai_test`。先写 `setupTestDB`(照抄 `internal/service/file/file_service_test.go:13-26`,但迁移列表补上 `&model.Config{}`、`&model.AITask{}`、`&model.AISuggestion{}`)与 `seedFile(t, path)`(用 `repoFile.NewRepository().Create().SetPath(path).SetName(filepath.Base(path)).SetType(mime.TypeByExtension(filepath.Ext(path))).Exec()`)。测试:

```go
func TestAIModels_Migrate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	m := storage.GetDB().Migrator()
	if !m.HasTable(&model.AITask{}) || !m.HasTable(&model.AISuggestion{}) {
		t.Fatal("expected ai_tasks and ai_suggestions tables")
	}
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run TestAIModels_Migrate -v`。预期 FAIL:`undefined: model.AITask`。

- [ ] **Step 3: 写模型** — `internal/model/ai.go`,照 `internal/model/file.go` 的 tag 风格(`gorm:"primaryKey;column:..."` + `json:"..."` + `TableName()`)。字段与类型:

```go
type AITask struct {
	TaskID     uint      `gorm:"primaryKey;column:task_id" json:"task_id"`
	FileID     uint      `gorm:"column:file_id;not null;index" json:"file_id"`
	Status     string    `gorm:"column:status;size:20;not null;default:'queued';index" json:"status"`
	Error      string    `gorm:"column:error;type:text" json:"error"`
	FrameCount int       `gorm:"column:frame_count;default:0" json:"frame_count"`
	CreatedAt  time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	File       File      `gorm:"foreignKey:FileID;references:FileID" json:"file"`
}

type AISuggestion struct {
	SuggestionID      uint       `gorm:"primaryKey;column:suggestion_id" json:"suggestion_id"`
	TaskID            uint       `gorm:"column:task_id;not null;index" json:"task_id"`
	FileID            uint       `gorm:"column:file_id;not null;index" json:"file_id"`
	CategoryName      string     `gorm:"column:category_name;size:100;not null" json:"category_name"`
	TagName           string     `gorm:"column:tag_name;size:100;not null" json:"tag_name"`
	TagValue          string     `gorm:"column:tag_value;size:200;default:''" json:"tag_value"`
	FinalCategoryName string     `gorm:"column:final_category_name;size:100;not null;default:''" json:"final_category_name"`
	FinalTagName      string     `gorm:"column:final_tag_name;size:100;not null;default:''" json:"final_tag_name"`
	FinalValue        string     `gorm:"column:final_value;size:200;default:''" json:"final_value"`
	MatchedTagID      *uint      `gorm:"column:matched_tag_id;index" json:"matched_tag_id,omitempty"`
	ApprovedTagID     *uint      `gorm:"column:approved_tag_id" json:"approved_tag_id,omitempty"`
	ReviewStatus      string     `gorm:"column:review_status;size:20;not null;default:'pending';index" json:"review_status"`
	ReviewedAt        *time.Time `gorm:"column:reviewed_at" json:"reviewed_at,omitempty"`
	CreatedAt         time.Time  `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
}
```

`TableName()` 分别返回 `"ai_tasks"`、`"ai_suggestions"`。

- [ ] **Step 4: 接进迁移** — `internal/bootstrap/init.go` 的 AutoMigrate 列表在 `&model.Config{}` 后追加 `&model.AITask{}, &model.AISuggestion{}`;同时给 3 份既有 `setupTestDB` 的列表补 `&model.Config{}, &model.AITask{}, &model.AISuggestion{}`。

- [ ] **Step 5: 跑测试确认通过** — `go test ./internal/... -v`。预期 PASS,且既有 7 个测试全绿。

- [ ] **Step 6: 提交**

```bash
git add internal/model/ai.go internal/service/ai/ai_service_test.go internal/bootstrap/init.go internal/service/file/file_service_test.go internal/service/tag/tag_service_test.go internal/service/category/category_service_test.go
git commit -m "feat: add AI task and suggestion models"
```

---

### Task 2: AI Repository 层

**Files:**
- Create: `internal/repository/ai/{repo,task_create,task_find,task_list,task_update,task_claim,suggestion_create,suggestion_find,suggestion_list,suggestion_update,suggestion_delete,suggestion_count}.go`
- Test: `internal/repository/ai/ai_repo_test.go`

**Interfaces:**
- Consumes: Task 1 的两个模型;`storage.GetDB()`。
- Produces(全部在 `package ai`,`NewRepository() *Repository`):
  - `CreateTask(t *model.AITask) error`(回填 `TaskID`)
  - `FindTask(id uint) (*model.AITask, error)` — 未找到返回 `(nil, nil)`
  - `ListTasks() *TaskListOperation`,链式 `.ByStatus(string) .ByFileID(uint) .Offset(int) .Limit(int)`,终态 `.Exec() ([]model.AITask, int64, error)`(按 `task_id` 排序,`Limit(0)` 为不限)
  - `UpdateTask(t *model.AITask) *TaskUpdateOperation`,`.SetStatus(string) .SetError(string) .SetFrameCount(int)`,`.Exec() error`(cols 白名单,照 `internal/repository/tag/update.go`)
  - `ClaimNextQueued() (*model.AITask, error)` — 取最小 `task_id` 的 queued,条件更新 `WHERE task_id=? AND status='queued'` 置 running,`RowsAffected==0` 或无候选返回 `(nil, nil)`
  - `FailRunning(reason string) (int64, error)` — 把所有 running 置 failed 并写 error,返回受影响行数
  - `CreateSuggestions(items []model.AISuggestion) error`(批量,照 `repository/file/create_batch.go`)
  - `FindSuggestion(id uint) (*model.AISuggestion, error)` — `(nil, nil)` 未找到
  - `ListSuggestions() *SuggestionListOperation`,`.ByStatus(string) .ByFileID(uint) .ByTaskID(uint) .Offset(int) .Limit(int)`,`.Exec() ([]model.AISuggestion, int64, error)`
  - `UpdateSuggestion(s *model.AISuggestion) *SuggestionUpdateOperation`,`.SetFinal(category, tag, value string) .SetReview(status string, approvedTagID *uint)`,`.Exec() error`(`SetReview` 同时写 `reviewed_at`)
  - `DeletePendingByFile(fileID uint) (int64, error)`
  - `CountPendingByFile() (map[uint]int64, error)`

- [ ] **Step 1: 写失败测试** — `internal/repository/ai/ai_repo_test.go`,`package ai_test`,自带一份 `setupTestDB`(同 Task 1 的九表列表)。覆盖:

```go
func TestAITaskRepo_ClaimNextQueued(t *testing.T) {
	// 建 2 个 queued 任务(file_id 1、2),ClaimNextQueued 两次:
	// 第一次返回 file_id 较小者且 Status=="running";第二次返回另一个;
	// 第三次返回 (nil, nil)。
}
func TestAITaskRepo_FailRunning(t *testing.T) {
	// 1 running + 1 queued → FailRunning("重启中断") 返回 1,
	// 且只有原 running 那条变 failed、Error=="重启中断",queued 不变。
}
func TestAISuggestionRepo_ListAndCountPending(t *testing.T) {
	// 同一 file 插 3 条(2 pending、1 rejected),另一 file 插 1 pending:
	// ListSuggestions().ByStatus("pending").ByFileID(f1).Exec() → 2 条、total==2;
	// CountPendingByFile()[f1]==2、[f2]==1。
}
func TestAISuggestionRepo_DeletePendingByFile(t *testing.T) {
	// 同上数据 → DeletePendingByFile(f1) 返回 2,rejected 那条仍在。
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/repository/ai/ -v`。预期 FAIL:`undefined: NewRepository`。

- [ ] **Step 3: 实现** — 照 `internal/repository/tag/list.go`(builder + 独立 count 查询 + `Order("task_id")`)、`tag/update.go`(cols 白名单)、`tag/find.go:68-82`(`errors.Is(err, gorm.ErrRecordNotFound)` → `(nil, nil)`)、`taggroup/group_create.go`(plain func)的既有写法。`ClaimNextQueued` 用两步:先 `Where("status = ?", "queued").Order("task_id").First(&t)`,再 `Model(&model.AITask{}).Where("task_id = ? AND status = ?", t.TaskID, "queued").Updates(map[string]any{"status": "running", "error": ""})` 并检查 `RowsAffected`。`CountPendingByFile` 用 `Model(&model.AISuggestion{}).Select("file_id, COUNT(*) as cnt").Where("review_status = ?", "pending").Group("file_id").Scan(&rows)`。

- [ ] **Step 4: 跑测试确认通过** — `go test ./internal/repository/ai/ -v`。预期 4 个测试 PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/repository/ai
git commit -m "feat: add AI task and suggestion repositories"
```

---

### Task 3: 配置项与默认值

**Files:**
- Create: `internal/service/ai/config.go`、`internal/service/ai/service.go`、`internal/service/ai/ports.go`
- Create: `internal/service/ai/config_test.go`
- Modify: `internal/bootstrap/init.go:24-36`(`seedDefaultConfig`)

**Interfaces:**
- Consumes: `configSvc.NewService()`、`(*config.Service).Get(key) (*model.Config, error)`(**key 缺失时返回 error,不是默认值**)、`Set(key, value) error`(upsert)。
- Produces:
  - key 常量:`KeyOllamaURL = "ollama_url"`、`KeyOllamaModel = "ollama_model"`、`KeyFrameInterval = "ai_frame_interval"`、`KeyDiffThreshold = "ai_frame_diff_threshold"`、`KeyMaxFrames = "ai_max_frames"`
  - `type Settings struct { OllamaURL string; OllamaModel string; FrameInterval float64; DiffThreshold int; MaxFrames int }`
  - `func (s *Service) Settings() Settings`
  - `type Service struct { aiRepo *repoAI.Repository; fileSvc *fileSvc.Service; tagSvc *tagSvc.Service; categorySvc *categorySvc.Service; configSvc *configSvc.Service; Extractor FrameExtractor; Client VisionClient; wake, stop, done chan struct{} }`、`func NewService() *Service`
  - `type FrameExtractor interface { Extract(ctx context.Context, videoPath string, intervalSeconds float64, maxFrames int) ([][]byte, error) }`
  - `type VisionClient interface { Recognize(ctx context.Context, prompt string, images [][]byte) (string, error) }`

默认值(写死在 `config.go`):`ollama_url` = `http://127.0.0.1:11434`;`ollama_model` = `""`(空表示未配置,识别时报错);`ai_frame_interval` = `1`;`ai_frame_diff_threshold` = `10`;`ai_max_frames` = `10`。

- [ ] **Step 1: 写失败测试** — `internal/service/ai/config_test.go`:

```go
func TestAISettings_DefaultsWhenUnset(t *testing.T) {
	// 不写任何 config → Settings() 得 URL=="http://127.0.0.1:11434"、
	// Model==""、FrameInterval==1、DiffThreshold==10、MaxFrames==10。
}
func TestAISettings_ReadsConfigAndFallsBackOnGarbage(t *testing.T) {
	// Set("ai_max_frames","25")、Set("ai_frame_interval","0.5")、
	// Set("ai_frame_diff_threshold","not-a-number")、Set("ollama_model","qwen2.5vl:7b")
	// → MaxFrames==25、FrameInterval==0.5、DiffThreshold==10(回落)、Model=="qwen2.5vl:7b"。
}
func TestAISettings_RejectsNonPositiveNumbers(t *testing.T) {
	// Set("ai_max_frames","0")、Set("ai_frame_interval","-1") → 回落 10 与 1。
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run TestAISettings -v`。预期 FAIL:`undefined: NewService`。

- [ ] **Step 3: 实现** — `ports.go` 放两个接口;`service.go` 的 `NewService()` 装配 5 个依赖(`Extractor`/`Client` 先留 nil,Task 5/6 完成后在此接上默认实现,导出字段便于测试替换);`config.go` 写 `Settings()`:逐 key `Get`,**忽略 error 回落默认值**,数值用 `strconv` 解析,解析失败或 ≤0 一律回落(阈值 0 也回落,因为 0 意味着「全部保留」失去过滤意义)。`seedDefaultConfig` 改为逐 key 存在性检查后 `db.Create`(现有实现遇到 `scan_exclude` 就整体 return,必须改成 per-key,否则新 key 永远种不进去),`Description` 用中文说明。

- [ ] **Step 4: 跑测试确认通过** — `go test ./internal/service/ai/ -run TestAISettings -v`。预期 3 个 PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/service/ai/config.go internal/service/ai/service.go internal/service/ai/ports.go internal/service/ai/config_test.go internal/bootstrap/init.go
git commit -m "feat: add Ollama and frame-extraction config with defaults"
```

---

### Task 4: 媒体判定与入队规则

**Files:**
- Create: `internal/service/ai/media.go`、`internal/service/ai/enqueue.go`、`internal/service/ai/status.go`(仅状态常量)、`internal/service/ai/enqueue_test.go`

**Interfaces:**
- Consumes: Task 2 repo;`fileSvc.FindByPath(path) (*model.File, error)`、`fileSvc.ListFiles(0, 0) ([]model.File, int64, error)`。
- Produces:
  - `func IsMediaFile(name string) (kind string, ok bool)` — kind 为 `"image"` / `"video"`;图片扩展名 `.jpg .jpeg .png .webp`,视频 `.mp4 .mkv .mov .avi .webm .m4v`(小写比较)
  - 状态常量:`TaskQueued/TaskRunning/TaskDone/TaskFailed = "queued"/"running"/"done"/"failed"`;`ReviewPending/ReviewApproved/ReviewRejected = "pending"/"approved"/"rejected"`;`AIStatusRunning/AIStatusFailed/AIStatusPending/AIStatusApproved = "running"/"failed"/"pending"/"approved"`
  - `type SkippedFile struct { Path string; Reason string }`、`type EnqueueResult struct { Queued []uint; Skipped []SkippedFile }`
  - `func (s *Service) EnqueuePaths(paths []string, force bool) (*EnqueueResult, error)`
  - `func (s *Service) EnqueueDir(dir string, recursive bool, force bool) (*EnqueueResult, error)`
  - `func (s *Service) RetryTask(taskID uint) error`

入队规则(逐条实现):未注册 → skip「未注册,请先 scan」;非图片/视频 → skip「不支持的文件类型」;`force==false` 且(该文件有 pending 建议 **或** 最新任务为 done)→ skip「已有待审建议」/「已完成识别」;`force==false` 且最新任务为 queued/running → skip「已在队列中」;`force==true` → 先 `DeletePendingByFile(fileID)` 再建新 queued 任务。`EnqueueDir` 用 `ListFiles(0,0)` 全量取回后按路径前缀在内存里筛(`recursive==false` 时只要直接子文件),避免动 file repo。`RetryTask` 只允许 failed → **先 `DeletePendingByFile(fileID)` 清掉旧 pending(spec §8 第 6 条要求避免重复建议堆积)**,再新建 queued 任务(不改旧行,保留失败历史)。

- [ ] **Step 1: 写失败测试** — `internal/service/ai/enqueue_test.go`:

```go
func TestIsMediaFile(t *testing.T) {
	// "a.JPG"→image,true;"b.mp4"→video,true;"c.txt"→"",false;"d"→"",false
}
func TestEnqueuePaths_SkipsUnregisteredAndNonMedia(t *testing.T) {
	// 只注册 a.jpg;EnqueuePaths(["a.jpg","ghost.jpg","note.txt"], false)
	// → Queued 长度 1,Skipped 长度 2 且 Reason 分别含「未注册」「不支持」。
}
func TestEnqueuePaths_SkipsAlreadyPendingOrDone(t *testing.T) {
	// f1 有 pending 建议、f2 最新任务 done、f3 最新任务 queued、f4 无任务
	// → 只有 f4 入队;三条 skip。
}
func TestEnqueuePaths_ForceClearsPendingAndRequeues(t *testing.T) {
	// f1 有 2 条 pending + 1 条 rejected,force=true
	// → 新任务 queued,pending 剩 0,rejected 仍 1(Review Focus #2)。
}
func TestEnqueueDir_RecursiveFilter(t *testing.T) {
	// 注册 "d/a.jpg"、"d/sub/b.jpg"、"e/c.jpg"
	// → EnqueueDir("d", false) 只入队 a.jpg;EnqueueDir("d", true) 入队 a.jpg 与 b.jpg。
}
func TestRetryTask_OnlyFailed(t *testing.T) {
	// failed 任务(该文件另有 2 条 pending)→ RetryTask 成功、新增 queued、pending 清为 0;
	// done 任务 → 返回 error 且不新增任务。
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run 'TestIsMediaFile|TestEnqueue|TestRetryTask' -v`。预期 FAIL:`undefined: IsMediaFile`。

- [ ] **Step 3: 实现** — `media.go` 用 `strings.ToLower(filepath.Ext(name))` 查两个 map。`enqueue.go` 先一次性取 `CountPendingByFile()` 与 `ListTasks().Exec()` 并在内存里归约出「每文件最新任务」(按最大 `TaskID`),再逐路径判定;这样 N 个文件只有 2 次查询。`status.go` 本任务只放常量。

- [ ] **Step 4: 跑测试确认通过** — 同 Step 2 命令。预期 6 个 PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/service/ai/media.go internal/service/ai/enqueue.go internal/service/ai/status.go internal/service/ai/enqueue_test.go
git commit -m "feat: add media eligibility and AI task enqueue rules"
```

---

### Task 5: 抽帧与近似帧过滤

**Files:**
- Create: `internal/service/ai/frames.go`、`internal/service/ai/diff.go`、`internal/service/ai/diff_test.go`
- Modify: `internal/service/ai/service.go`(`NewService()` 里 `Extractor: NewFFmpegExtractor()`)

**Interfaces:**
- Consumes: Task 3 的 `FrameExtractor`。
- Produces:
  - `func NewFFmpegExtractor() FrameExtractor`(内部 `ffmpegExtractor struct{}`)
  - `func FilterSimilarFrames(frames [][]byte, threshold int, maxFrames int) ([][]byte, error)`
  - 未导出:`dHash(img image.Image) uint64`、`hamming(a, b uint64) int`

`Extract` 行为:`exec.LookPath("ffmpeg")` 失败 → 明确 error「未找到 ffmpeg,请先安装」;`os.MkdirTemp("", "tagfile-ai-")` 建临时目录并 `defer os.RemoveAll`;命令 `ffmpeg -hide_banner -loglevel error -y -i <videoPath> -vf fps=1/<interval> -frames:v <maxFrames*10> -f image2 <tmp>/frame_%04d.jpg`(interval 用 `strconv.FormatFloat(v,'f',-1,64)`);`exec.CommandContext` 带 120s 超时;按文件名排序读回 `[][]byte`;**0 帧 → error「未能从视频抽取任何帧」**;stderr 非空时并入 error。`FilterSimilarFrames`:逐帧 `image.Decode`(匿名导入 `image/jpeg`、`image/png`),解码失败 → error;首帧必留;与「上一个保留帧」的 dHash 汉距 `< threshold` 则丢弃;保留数达 `maxFrames` 即停。dHash 用 9×8 灰度最近邻采样(不引入 `x/image`),相邻像素比较得 64 位。

- [ ] **Step 1: 写失败测试** — `internal/service/ai/diff_test.go`,用 `image.NewRGBA` + `jpeg.Encode` 现造帧,不依赖任何 fixture 文件:

```go
func TestFilterSimilarFrames_DropsNearDuplicates(t *testing.T) {
	// 造 5 帧:纯红、纯红(微噪 1 像素)、纯蓝、纯蓝、纯绿
	// → FilterSimilarFrames(frames, 10, 10) 保留 3 帧(红/蓝/绿)。
}
func TestFilterSimilarFrames_AlwaysKeepsFirstAndCapsAtMax(t *testing.T) {
	// 8 帧互不相同 → threshold=10, maxFrames=3 → 恰好 3 帧,且第 1 帧等于输入第 1 帧。
	// 单帧输入 → 原样返回 1 帧。空输入 → 返回空、无 error。
}
func TestFilterSimilarFrames_RejectsUndecodableFrame(t *testing.T) {
	// [][]byte{[]byte("not an image")} → 返回 error(Review Focus #4)。
}
func TestFFmpegExtractor_MissingBinaryOrZeroFrames(t *testing.T) {
	// 对不存在的路径调用 Extract → 必须返回 error,不得返回 (nil, nil)。
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run 'TestFilterSimilarFrames|TestFFmpeg' -v`。预期 FAIL:`undefined: FilterSimilarFrames`。

- [ ] **Step 3: 实现** — `diff.go` 写纯函数(采样、灰度 `0.299R+0.587G+0.114B`、位比较、`bits.OnesCount64(a^b)`);`frames.go` 写 ffmpeg 适配器。`NewService()` 接上 `Extractor`。

- [ ] **Step 4: 跑测试确认通过** — 同 Step 2 命令。预期 4 个 PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/service/ai/frames.go internal/service/ai/diff.go internal/service/ai/diff_test.go internal/service/ai/service.go
git commit -m "feat: add ffmpeg frame extraction and dHash dedup"
```

---

### Task 6: Ollama 视觉客户端

**Files:**
- Create: `internal/service/ai/ollama.go`、`internal/service/ai/ollama_test.go`
- Modify: `internal/service/ai/ollama_test.go` 之外无需改 `service.go`:`Client` 字段保持 nil,由 `processTask` 在运行时按 `Settings()` 构造(见 Task 8),测试直接给 `svc.Client` 赋假实现即可。

**Interfaces:**
- Consumes: Task 3 的 `VisionClient`、`Settings()`。
- Produces: `func NewOllamaClient(baseURL, model string) VisionClient`;`Recognize(ctx, prompt, images)` 返回模型 `message.content` 原文。

请求:`POST <baseURL>/api/chat`,JSON body `{"model":..., "messages":[{"role":"user","content":<prompt>,"images":[<base64>...]}], "stream":false, "format":"json"}`。base64 用 `base64.StdEncoding`,**不带 data-URI 前缀**。`http.Client{Timeout: 180 * time.Second}`(常量 `ollamaTimeout`)。响应解析 `{"message":{"content":"..."}}`。非 2xx → error 含状态码与 body 前 200 字符;`content` 为空 → error「模型返回空内容」;`model == ""` → error「未配置 Ollama 模型名」。

- [ ] **Step 1: 写失败测试** — `internal/service/ai/ollama_test.go`,用 `net/http/httptest`(stdlib,无新依赖):

```go
func TestOllamaClient_SendsPromptAndBase64Images(t *testing.T) {
	// httptest server 解码请求体,断言:model 正确、stream==false、format=="json"、
	// messages[0].content==prompt、images[0]==base64.StdEncoding.EncodeToString([]byte{1,2,3})、
	// URL path=="/api/chat";返回 {"message":{"content":"{\"tags\":[]}"}} → Recognize 得到该字符串。
}
func TestOllamaClient_HTTPErrorAndEmptyContent(t *testing.T) {
	// 500 + body "boom" → error 含 "500" 与 "boom";
	// 200 + {"message":{"content":""}} → error 含「空内容」。
}
func TestOllamaClient_MissingModelAndTimeout(t *testing.T) {
	// model=="" → error 含「未配置」;
	// server  sleep 超过客户端超时(测试里把超时改小或用一个立即关闭连接的 server)→ 返回 error 而非永久阻塞(Review Focus #5)。
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run TestOllamaClient -v`。预期 FAIL:`undefined: NewOllamaClient`。

- [ ] **Step 3: 实现** — 请求/响应用未导出 struct + `encoding/json`;`baseURL` 末尾 `/` 先 `strings.TrimRight`;超时测试通过把 `ollamaTimeout` 做成 `ollamaClient` 的字段(默认 180s)以便测试注入小值。

- [ ] **Step 4: 跑测试确认通过** — 同 Step 2 命令。预期 3 个 PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/service/ai/ollama.go internal/service/ai/ollama_test.go
git commit -m "feat: add Ollama vision chat client"
```

---

### Task 7: 提示词构建与输出解析

**Files:**
- Create: `internal/service/ai/prompt.go`、`internal/service/ai/parse.go`、`internal/service/ai/parse_test.go`

**Interfaces:**
- Consumes: `categorySvc.ListCategories(0, 0)`、`tagSvc.ListTagTree("") ([]tag.TagTreeNode, error)`(`TagTreeNode{Tag model.Tag; Children []TagTreeNode}`)、`tagSvc.ListTagValuesBatch(names []string) (map[string][]model.TagValue, error)`。
- Produces:
  - `func (s *Service) BuildPrompt() (string, error)`
  - `type RawTag struct { Category string \`json:"category"\`; Tag string \`json:"tag"\`; Value string \`json:"value"\` }`
  - `func ParseSuggestions(output string) ([]RawTag, error)`
  - `func (s *Service) matchTag(categoryName, tagName string) (*uint, error)` — 命中返回 tag_id,未命中返回 `(nil, nil)`

`BuildPrompt` 内容:中文指令 + 现有分类清单 + 标签树(用 `Parent / Child` 路径,附各标签预设值)+ 输出契约。契约固定为 `{"tags":[{"category":"<现有分类名>","tag":"<标签名>","value":"<可选,自由文本>"}]}`,明确要求:优先复用已有分类/标签/预设值;允许新词;不要输出描述、置信度、父标签;只输出 JSON。`ParseSuggestions` 容错:剥 ```/```json 围栏 → 取首个 `{` 到末个 `}` → `json.Unmarshal` 到 `struct{ Tags []RawTag }`;`Tag` 去空白后为空的条目丢弃;解析失败或 0 条有效 → error,error 文本附原文前 500 字符。`matchTag` 用 `tagSvc.ListTags(categoryName, 0, 0)` 后在内存里按 `TagName` 精确匹配(标签库规模小,spec §5 已确认)。

- [ ] **Step 1: 写失败测试** — `internal/service/ai/parse_test.go`:

```go
func TestBuildPrompt_ContainsCategoriesTreeAndValues(t *testing.T) {
	// 建分类「场景内容」+ 标签「海滩」(父「户外」)+ 预设值「日落」
	// → prompt 含「场景内容」「户外 / 海滩」「日落」与 `"tags"` 契约字样。
}
func TestParseSuggestions_PlainFencedAndProse(t *testing.T) {
	// 三种输入都解析出 1 条 {Category:"场景内容", Tag:"海滩", Value:"日落"}:
	// 裸 JSON、```json 围栏、前后带散文的 JSON(Review Focus #1)。
}
func TestParseSuggestions_RejectsGarbageAndEmpty(t *testing.T) {
	// "我觉得这是海滩" → error 且含原文片段;
	// `{"tags":[]}` → error;`{"tags":[{"category":"x","tag":"  "}]}` → error。
}
func TestMatchTag_HitsOnlyWithinCategory(t *testing.T) {
	// 分类 A 有「猫」,分类 B 无 → matchTag("A","猫") 返回非 nil;
	// matchTag("B","猫") 返回 (nil,nil);matchTag("不存在","猫") 返回 (nil,nil)(Review Focus #3)。
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run 'TestBuildPrompt|TestParseSuggestions|TestMatchTag' -v`。预期 FAIL:`undefined: ParseSuggestions`。

- [ ] **Step 3: 实现** — 树遍历用递归 + `strings.Builder`,深度上限 20(照 `api.buildTagPath` 的做法);预设值批量一次取回,不在循环里查库。

- [ ] **Step 4: 跑测试确认通过** — 同 Step 2 命令。预期 4 个 PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/service/ai/prompt.go internal/service/ai/parse.go internal/service/ai/parse_test.go
git commit -m "feat: add vision prompt builder and suggestion parser"
```

---

### Task 8: 识别流水线与单 worker

**Files:**
- Create: `internal/service/ai/recognize.go`、`internal/service/ai/worker.go`、`internal/service/ai/worker_test.go`

**Interfaces:**
- Consumes: Task 2–7 全部;`fileSvc.FindByID(id)`。
- Produces:
  - `func (s *Service) StartWorker()` — 先 `FailRunning("重启中断")`,再起 goroutine
  - `func (s *Service) StopWorker()` — 发停止信号并**等待 goroutine 退出**(幂等)
  - `func (s *Service) processTask(task *model.AITask) error`(未导出编排, worker 调用)

`processTask` 顺序:取文件 → `st := s.Settings()`,`st.OllamaModel == ""` 直接失败 → 解析客户端 `client := s.Client; if client == nil { client = NewOllamaClient(st.OllamaURL, st.OllamaModel) }`(测试赋假实现即生效)→ 按 `IsMediaFile` 决定图片(直接 `os.ReadFile`,路径 = `filepath.Join(cwd, file.FilePath)`,因为 `File.FilePath` 是相对 scan 根)或视频(`Extractor.Extract` → `FilterSimilarFrames`)→ `BuildPrompt` → `client.Recognize` → `ParseSuggestions` → 逐条 `matchTag` → `CreateSuggestions`(全部 `ReviewPending`,`Final*` = 原始值)→ `UpdateTask` 置 `TaskDone` + `FrameCount`。**任何一步出错都置 `TaskFailed` 并把 error 文本写进 `Error`,不写半截建议**。worker 循环:`ClaimNextQueued` → 有则 `processTask` → 无则等 `wake`/`stop`;入队成功后非阻塞投递 `wake`。

- [ ] **Step 1: 写失败测试** — `internal/service/ai/worker_test.go`。假实现:

```go
type fakeExtractor struct{ frames [][]byte; err error }
func (f fakeExtractor) Extract(ctx context.Context, p string, iv float64, mx int) ([][]byte, error) { return f.frames, f.err }
type fakeClient struct{ out string; err error; calls int }
func (f *fakeClient) Recognize(ctx context.Context, p string, imgs [][]byte) (string, error) { f.calls++; return f.out, f.err }
```

测试(cleanup 里必须 `svc.StopWorker()`):

```go
func TestWorker_ProcessesQueuedTaskAndStoresPendingSuggestions(t *testing.T) {
	// 注册 a.jpg,入队,fakeClient 返回 2 条建议 JSON
	// → 轮询等待任务 Status==done、FrameCount==1、2 条建议均 pending 且 Final*==原始值。
}
func TestWorker_RecordsFailureAndAllowsRetry(t *testing.T) {
	// fakeClient.err = errors.New("模型不可达") → 任务 failed、Error 含「模型不可达」、建议 0 条;
	// RetryTask 后新任务 queued(Review Focus #4)。
	// 另一例:fakeExtractor.err → 同样 failed 且 0 条建议。
}
func TestWorker_RestartMarksRunningFailedAndResumesQueued(t *testing.T) {
	// 手工插 1 条 running + 1 条 queued → StartWorker()
	// → 原 running 变 failed 且 Error=="重启中断";queued 被处理成 done(Review Focus #5)。
}
func TestWorker_SequentialNotConcurrent(t *testing.T) {
	// fakeClient 每次调用记录进入/离开时间戳并 sleep 20ms,入队 3 个文件
	// → 3 次调用区间互不重叠(单 worker,spec 反目标「并发打爆显存」)。
}
func TestWorker_EmptyModelConfigFails(t *testing.T) {
	// 不设 ollama_model → 任务 failed 且 Error 含「未配置」。
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run TestWorker -v`。预期 FAIL:`undefined: StartWorker`。

- [ ] **Step 3: 实现** — `worker.go` 用 `wake/stop/done` 三个 channel(`wake` 缓冲 1,投递用 `select { case s.wake <- struct{}{}: default: }`);`StopWorker` 用 `sync.Once` 保证幂等。`recognize.go` 里错误统一 `fmt.Errorf("...: %w", err)` 中文前缀,与既有 service 风格一致。等待任务完成的测试辅助用 50ms 间隔、2s 上限的轮询,超时就 `t.Fatal`。

- [ ] **Step 4: 跑测试确认通过** — `go test ./internal/service/ai/ -v`。预期全部 PASS,**且无 goroutine 泄漏导致的后续包失败**:再跑 `go test ./internal/... `确认。

- [ ] **Step 5: 提交**

```bash
git add internal/service/ai/recognize.go internal/service/ai/worker.go internal/service/ai/worker_test.go
git commit -m "feat: add single-worker AI recognition pipeline"
```

---

### Task 9: 审核入库

**Files:**
- Create: `internal/service/ai/review.go`、`internal/service/ai/review_test.go`

**Interfaces:**
- Consumes: `tagSvc.AddTag(categoryName, tagName, description, parentTagName string) (*model.Tag, error)`、`tagSvc.ListTags(categoryName, 0, 0)`、`fileSvc.TagFile(filePath, tagID, tagValue string) error`、`fileSvc.ListFileTags(path, 0, 0) ([]model.TagGroupTag, int64, error)`、`fileSvc.FindByID`。
- Produces:
  - `type Failure struct { ID uint; Msg string }`
  - `func (s *Service) UpdateSuggestion(id uint, categoryName, tagName, tagValue string) error`(只改 `Final*`,仅 pending 可改)
  - `func (s *Service) Approve(ids []uint) (int, []Failure, error)`
  - `func (s *Service) Reject(ids []uint) (int, []Failure, error)`
  - `func (s *Service) resolveOrCreateTag(categoryName, tagName string) (*model.Tag, error)`

`Approve` 单条流程:取建议 → 非 pending 记 Failure「建议已审核」→ `resolveOrCreateTag(FinalCategoryName, FinalTagName)` → 若该文件已存在同 `(tag_id, tag_value)` 关联(用 `ListFileTags` 比对)则跳过 `TagFile`,否则 `TagFile(file.FilePath, tag.TagID, FinalValue)` → `SetReview(ReviewApproved, &tag.TagID)`。`resolveOrCreateTag`:先在目标分类内按名找,找到即复用;找不到调 `AddTag(category, name, "", "")`;若 `AddTag` 报「已存在」则**再查一次并复用**(覆盖并发/跨文件同新词,spec §8 第 5 条);分类不存在则原样返回 error(不自动建分类)。`Reject` 只置 `ReviewRejected`。**任何路径都不得写 `TagValue` 预设值表。**

- [ ] **Step 1: 写失败测试** — `internal/service/ai/review_test.go`。先加共用 helper `seedSuggestion(t, fileID, category, tag, value) uint`(用 `repoAI.CreateSuggestions` 插一条 pending,`Final*` 等于原始值,返回 `SuggestionID`);导入别名 `ai`、`repoAI`、`categorySvc`、`tagSvc`、`fileSvc`。核心不变量用完整断言写死:

```go
func TestApprove_CreatesTagAndAssociatesValue(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	f := seedFile(t, "clothes/pants.jpg")
	if _, err := categorySvc.NewService().AddCategory("服饰", ""); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	id := seedSuggestion(t, f.FileID, "服饰", "裤子", "黑色")

	n, failures, err := ai.NewService().Approve([]uint{id})
	if err != nil || n != 1 || len(failures) != 0 {
		t.Fatalf("Approve = (%d, %v, %v), want (1, [], nil)", n, failures, err)
	}
	tags, _, err := tagSvc.NewService().ListTags("服饰", 0, 0)
	if err != nil || len(tags) != 1 || tags[0].TagName != "裤子" {
		t.Fatalf("expected new tag 裤子, got %v (%v)", tags, err)
	}
	vals, err := tagSvc.NewService().ListTagValues("裤子")
	if err != nil || len(vals) != 0 {
		t.Fatalf("approved value must not enter predefined dictionary, got %v", vals)
	}
	fs := fileSvc.NewService()
	assoc, _, err := fs.ListFileTags(f.FilePath, 0, 0)
	if err != nil || len(assoc) != 1 || assoc[0].TagValue != "黑色" {
		t.Fatalf("expected association value 黑色, got %v (%v)", assoc, err)
	}
	got, _ := fs.FindByID(f.FileID)
	if got.TagCnt != 1 {
		t.Fatalf("TagCnt = %d, want 1", got.TagCnt)
	}
}
```

其余测试(断言要点已写死,照上面的风格补全):

```go
func TestApprove_ReusesExistingTagAndDedupesAcrossFiles(t *testing.T) {
	// 两个文件的建议都是新词「袜子」→ 两次 Approve 后标签库里「袜子」只有 1 条,
	// 两个文件各自 TagCnt==1(spec §8 第 5 条)。
}
func TestApprove_TwiceIsNoOpAndKeepsTagCnt(t *testing.T) {
	// 同一 id 连续 Approve 两次 → 第二次 failures 含「已审核」,
	// 关联仍 1 条、TagCnt 仍 1(Review Focus #2)。
}
func TestApprove_MissingCategoryFailsWithoutSideEffects(t *testing.T) {
	// FinalCategoryName 改成不存在的分类 → failure 含「不存在」,
	// 建议仍 pending、无新标签、TagCnt==0(Review Focus #3)。
}
func TestApprove_SameTagNameInOtherCategoryCreatesSeparateTag(t *testing.T) {
	// 分类 A 已有「猫」,建议 FinalCategoryName=B → 在 B 下新建「猫」,
	// A 的「猫」不受影响,两条 TagID 不同(Review Focus #3)。
}
func TestUpdateSuggestion_OnlyFinalFieldsAndPendingOnly(t *testing.T) {
	// Update 后 CategoryName/TagName/TagValue 原值不变、Final* 已改;
	// 对 approved 建议 Update → error。
}
func TestReject_MarksRejectedWithoutTagging(t *testing.T) {
	// Reject → review_status==rejected、标签库无新词、TagCnt==0。
}
func TestApprove_ValueWithColonRoundTrips(t *testing.T) {
	// value = "黑色:长袜" → 关联 TagValue 完全一致(不被冒号截断)。
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run 'TestApprove|TestReject|TestUpdateSuggestion' -v`。预期 FAIL:`undefined: Approve`。

- [ ] **Step 3: 实现** — `Approve`/`Reject` 遍历 ids,单条失败收集进 `[]Failure` 继续下一条(批量里一条坏的不拖垮整批),返回值第一位是成功数。关联预检查用 `ListFileTags` 遍历比对 `TagID` 与 `TagValue`。

- [ ] **Step 4: 跑测试确认通过** — 同 Step 2 命令。预期 8 个 PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/service/ai/review.go internal/service/ai/review_test.go
git commit -m "feat: add human review gate for AI tag suggestions"
```

---

### Task 10: 文件级 AI 状态推导

**Files:**
- Modify: `internal/service/ai/status.go`
- Create: `internal/service/ai/status_test.go`

**Interfaces:**
- Consumes: Task 2 的 `ListTasks()`、`CountPendingByFile()`;Task 4 的状态常量。
- Produces:
  - `func (s *Service) FileStatuses(fileIDs []uint) (map[uint]string, error)`
  - `func (s *Service) FileStatus(fileID uint) (string, error)`

推导规则(取该文件**最新**任务,即最大 `TaskID`):无任务 → 不入 map(前端视为未识别);queued/running → `AIStatusRunning`;failed → `AIStatusFailed`;done 且有 pending 建议 → `AIStatusPending`;done 且无 pending → `AIStatusApproved`。实现只发 2 次查询(全量任务 + pending 计数),在内存归约,与仓库「一次性载入、无逐节点查库」的既有做法一致。

- [ ] **Step 1: 写失败测试** — `internal/service/ai/status_test.go`:

```go
func TestFileStatuses_AllTransitions(t *testing.T) {
	// f1 无任务 → map 里没有 f1;
	// f2 queued、f3 running → 都是 "running";
	// f4 failed → "failed";
	// f5 done + 1 pending → "pending";
	// f6 done + 全部 rejected → "approved"。
}
func TestFileStatuses_LatestTaskWins(t *testing.T) {
	// f1 先 done(有 pending)后又新建 queued → 状态回到 "running";
	// 旧 done 任务的 pending 建议不影响判定。
}
func TestFileStatus_EmptyFileID(t *testing.T) {
	// FileStatus(999) → ("", nil),不报错。
}
```

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/service/ai/ -run TestFileStatus -v`。预期 FAIL:`undefined: FileStatuses`。

- [ ] **Step 3: 实现** — 归约 `map[uint]model.AITask` 保留最大 `TaskID`;`FileStatus` 复用 `FileStatuses([]uint{id})`。

- [ ] **Step 4: 跑测试确认通过** — `go test ./internal/... `。预期全绿。

- [ ] **Step 5: 提交**

```bash
git add internal/service/ai/status.go internal/service/ai/status_test.go
git commit -m "feat: derive per-file AI tagging status"
```

---

### Task 11: WebUI API 端点与接线

**Files:**
- Create: `cmd/webui/api/ai.go`
- Modify: `cmd/webui/api/api.go:35-61`(Router 字段 + `NewRouter` 第 7 个参数)、`:108-120`(新增 `api.Group("/ai")`)
- Modify: `cmd/webui/api/file.go`(`FileItem:20-28`、`FileTreeNode:31-42` 加 `AIStatus string \`json:"ai_status,omitempty"\``;`convertTreeNodes:195-213`、`HandleExpandDir:238`、`HandleFileDetail:347,389` 填充)
- Modify: `cmd/webui/main.go:73-84`(实例化 `aiService := aiSvc.NewService()`、传入 `NewRouter`)、`:95-98`(`aiService.StartWorker()` + `defer aiService.StopWorker()`)

**Interfaces:**
- Consumes: Task 4/8/9/10 的 service 方法;`respond`、`parsePagination`、`computePagination`、`rt.loadTagPathMap`、`buildTagPath`。
- Produces:7 个 handler + 2 个 DTO。路由(全部 form-urlencoded):

| 方法 | 路径 | 入参 | 返回 extra |
|---|---|---|---|
| POST | `/api/ai/tasks` | `paths`(可重复)或 `dir` + `recursive` + `force`(`"true"`) | `queued`(int)、`skipped`([]SkippedFile) |
| GET | `/api/ai/tasks` | `page`、`status` | `items`([]AITaskItem)、`pagination` |
| POST | `/api/ai/tasks/retry` | `task_id` | — |
| GET | `/api/ai/suggestions` | `page`、`status`(默认 `pending`)、`file_id` | `items`([]AISuggestionItem)、`pagination` |
| PUT | `/api/ai/suggestions/:id` | `category_name`、`tag_name`、`tag_value` | — |
| POST | `/api/ai/suggestions/approve` | `ids`(可重复) | `approved`(int)、`failures`([]Failure) |
| POST | `/api/ai/suggestions/reject` | `ids`(可重复) | `rejected`(int)、`failures`([]Failure) |

DTO 字段(json key):`AITaskItem{task_id, file_id, file_path, file_name, status, error, frame_count, pending_count, created_at, updated_at}`(时间用 `time.RFC3339` 字符串);`AISuggestionItem{suggestion_id, file_id, file_path, category_name, tag_name, tag_value, final_category_name, final_tag_name, final_value, matched_tag_id, matched_tag_path, review_status}`。`matched_tag_path` 用 `loadTagPathMap` + `buildTagPath`。列表端点照 `HandleListConfig` 的两段式分页(`parsePagination(c,0,20)` → 查询 → `computePagination`)。

- [ ] **Step 1: 写端点与 DTO** — `cmd/webui/api/ai.go`。`c.PostFormArray("paths")` / `("ids")` 取重复键;`force`/`recursive` 判定用 `c.PostForm("force") == "true"`;`task_id`/`file_id` 用 `strconv.ParseUint` 且 0 视为无效(照 `HandleTagFile:516-549`)。`/api/ai` 是新顶层组,不碰 `/api/files` 的通配。

- [ ] **Step 2: 接线** — 改 `api.go` 的 struct/`NewRouter`/`RegisterRoutes`、`main.go` 的实例化与 worker 启停、`file.go` 的 AI 状态填充(`HandleFileTree`/`HandleExpandDir`/`HandleFileDetail` 各调一次 `FileStatuses` 批量取,不在循环里查)。

- [ ] **Step 3: 编译验证** — `go build ./... && make build-webui`。预期无错误、产出 `tagfile-webui.out`。

- [ ] **Step 4: 接口冒烟** — `./tagfile-webui.out --source temp/test_workspace --addr :8088`,另开终端:

```bash
curl -s 'localhost:8088/api/ai/tasks?page=1'                      # → {"ok":true,...,"items":[],"pagination":{...}}
curl -s -X POST localhost:8088/api/ai/tasks -d 'dir=.'            # → ok:true,skipped 里是 dummy 媒体或未注册原因
curl -s 'localhost:8088/api/ai/suggestions?status=pending'        # → ok:true, items:[]
curl -s 'localhost:8088/api/files' | head -c 400                  # → 树节点含 ai_status 字段(有任务时)
curl -s -X POST localhost:8088/api/ai/suggestions/approve -d 'ids=999'  # → ok:true, approved:0, failures 含「不存在」
```

预期:全部 `ok:true`(业务失败走 `failures`,不是 HTTP 500),无 panic。

- [ ] **Step 5: 提交**

```bash
git add cmd/webui/api/ai.go cmd/webui/api/api.go cmd/webui/api/file.go cmd/webui/main.go
git commit -m "feat: expose AI task and review API endpoints"
```

---

### Task 12: 待审队列页面

**Files:**
- Create: `cmd/webui/templates/review.html`、`cmd/webui/static/js/pages/review/main.js`
- Modify: `cmd/webui/main.go:64`(页面名切片加 `"review"`)、`cmd/webui/handlers/handlers.go:46-55`(`r.GET("/review", s.HandlePage("review", "review"))`)、`cmd/webui/templates/base.html:36-41`(导航项,插在「文件管理」与「标签搜索」之间)

**Interfaces:**
- Consumes: Task 11 的 7 个端点;`API.get/post/put`、`showToast`、`showConfirm`、`Modal.show`、`escHtml`、`inlineEdit`。
- Produces:Alpine 组件 `reviewPage()`,含三块卡片:发起识别(目录输入 + 递归/强制 checkbox + 按钮)、任务状态(轮询)、待审建议(多选 + 批量通过/拒绝 + 逐条编辑)。

实现要点:页面只 `{{define "content"}}`,数据全部客户端拉取(`HandlePage` 只注入 `Active`);`init()` 里 `this.refresh()` 并 `this.timer = setInterval(..., 3000)` 轮询任务,**加 `destroy()` 清 interval**(仓库现有页面没有 destroy 先例,这里必须自己写);轮询用 `try/catch` 包住 `API.get`,因为 `respond(...,"error")` 返回 400 而 `API.get` 会对非 2xx throw,不能让一次抖动杀掉轮询;`x-html` 生成的行内所有插值走 `escHtml`,点击用事件委托 + `data-action`/`data-id`(照 `tag/main.js`);多选用 `selected: []` + 全选 checkbox(仓库无先例,自己写);逐条编辑用 `Modal.show` 三个字段(分类/标签名/值),标签名字段带 `autocomplete:{tags}`(`Modal.show` 已支持,标签源用 `API.loadAllTags()`)——**选中已有标签即 spec §6 要求的「合并到已有标签」**,因为 `Approve` 会按名复用已有标签而不新建;提交走 `API.put('/api/ai/suggestions/'+id, ...)`;批量通过 → `API.post(..., params)` 用 `URLSearchParams.append('ids', id)` 重复键,返回后按 `failures` 逐条 toast。

- [ ] **Step 1: 写页面与 JS** — 照 `templates/config.html` + `static/js/pages/category/main.js` 的骨架(`alpine:init` 包裹、`loading` 标志、`refresh/doSearch/goToPage` 命名)。

- [ ] **Step 2: 注册三处** — 页面名切片、路由、导航。漏掉切片会 500 `template not found: review`。

- [ ] **Step 3: 编译 + 浏览器验证** — `make build-webui` 后启动,浏览器打开 `/review`:导航高亮正确;空队列显示空态;填目录 `.` 点发起 → toast 显示 queued/skipped 数;任务卡片 3 秒内自动刷新状态;**切到别的页面后确认轮询已停**(控制台无持续请求)。

- [ ] **Step 4: 审核闭环验证** — 产生 pending 建议必须有可用的 Ollama 视觉模型(**本机当前没有运行 Ollama**,`127.0.0.1:11434` 无响应)。在配置页填好 `ollama_url`/`ollama_model` 后,对一张真实图片发起识别,然后在 `/review` 逐条验证:改值 → 通过 → 到 `/files/detail/<path>` 确认标签与值出现、`TagCnt` +1;拒绝另一条 → 确认标签库无该新词;把标签名改成已有标签 → 通过后确认复用了原标签而非新建。**若没有可用模型,本步只验证空态与失败态,并把「有建议时的审核闭环」明确记为未验证项写进提交说明,不得声称已完成。**

- [ ] **Step 5: 提交**

```bash
git add cmd/webui/templates/review.html cmd/webui/static/js/pages/review/main.js cmd/webui/main.go cmd/webui/handlers/handlers.go cmd/webui/templates/base.html
git commit -m "feat: add AI review queue page"
```

---

### Task 13: 配置页 AI 卡片

**Files:**
- Modify: `cmd/webui/templates/config.html:67-69`(在导出卡片前插入「AI 识别」卡片)、`cmd/webui/static/js/pages/config/main.js`

**Interfaces:**
- Consumes: 现有 `POST /api/config`(`key`/`value`)、`GET /api/config`;Task 3 的 5 个 key 与默认值。
- Produces:`configPage()` 新增 `ai` 状态对象与 `loadAiConfig()` / `submitAiConfig()`。**不新增任何后端端点。**

卡片含 5 个带标签输入:`ollama_url`(text)、`ollama_model`(text,placeholder `qwen2.5vl:7b`)、`ai_frame_interval`(number,step 0.5,min 0.1)、`ai_frame_diff_threshold`(number,min 1,max 64)、`ai_max_frames`(number,min 1)。`loadAiConfig()` 用 `API.getAllPages('/api/config')` 取全量后按 key 填,缺失填默认值;`submitAiConfig()` 逐个 `API.post('/api/config', {key, value})`,任一失败 toast 报错并中止,全部成功 toast「AI 配置已保存」。**不修改现有通用配置表格的行为。**

- [ ] **Step 1: 写卡片与 JS** — 照 `config.html` 既有 `.card` 结构与 `x-model` 绑定风格。
- [ ] **Step 2: 编译 + 浏览器验证** — `/config` 出现 AI 卡片;改 `ai_max_frames` 为 3 保存 → 刷新页面仍为 3;通用配置表格里能看到 `ai_max_frames` 行且值为 3;把 `ai_frame_diff_threshold` 填 `abc` 保存后,后端 `Settings()` 回落 10(用 Task 3 的测试已覆盖,这里只需确认保存不报错)。
- [ ] **Step 3: 提交**

```bash
git add cmd/webui/templates/config.html cmd/webui/static/js/pages/config/main.js
git commit -m "feat: add AI recognition settings card to config page"
```

---

### Task 14: 文件树/详情的 AI 状态与发起入口

**Files:**
- Modify: `cmd/webui/static/js/pages/file/tree.js:103,109`(每文件 AI 徽章)、`cmd/webui/templates/file.html`(工具栏多选按钮)、`cmd/webui/static/js/pages/file/detail.js:170-186` 附近、`cmd/webui/templates/file_detail.html:31`(侧栏徽章)与 `:70-73`(头部按钮区)
- Modify: `cmd/webui/static/css/components/tags.css:42` 后、`cmd/webui/static/css/components/tree.css:53` 后(新增状态色)

**Interfaces:**
- Consumes: Task 11 在 `FileTreeNode`/`FileItem`/详情里填好的 `ai_status`;`POST /api/ai/tasks`(`paths` 重复键 / `force=true`)。
- Produces:树行 AI 徽章、树工具栏「识别选中(N)」、详情页 AI 状态 pill + 「AI 识别」/「重新识别」按钮。

徽章映射:`running`→「识别中」(`--accent`)、`pending`→「待审」(`--warning`)、`approved`→「AI 通过」(`--success`)、`failed`→「识别失败」(`--danger`)、空→不渲染。新增 `.badge-ai-running/.badge-ai-pending/.badge-ai-approved/.badge-ai-failed` 与 `.tree-status-ai-*`,复用既有 `--accent/--warning/--success/--danger` 变量(**没有 `--info`**)。`TagDone` 徽章与按钮**保持原样不动**。多选:树行渲染时给已索引媒体文件加 checkbox(`data-path`),`selectedPaths: []` 状态,工具栏按钮 `:disabled="selectedPaths.length===0"`,点击后 `URLSearchParams.append('paths', p)` 逐个追加 → `POST /api/ai/tasks` → toast + `refresh()`。详情页按钮:`ai_status` 为空或 failed 时显示「AI 识别」,否则显示「重新识别」并 `showConfirm` 后带 `force=true`;成功后乐观更新本地 `ai_status='running'`(照 `toggleTagDone` 的乐观更新写法),不要 `window.location.reload()`(详情页是 pushState 导航)。

- [ ] **Step 1: 加 CSS 与徽章渲染** — `tree.js` 在 `:103` 旁加 `var ai = ...`,并在 `:109` 的拼接里 `done` 之后插 `+ ai`。
- [ ] **Step 2: 加多选与详情页入口** — 按上述要点实现,所有插值 `escHtml`。
- [ ] **Step 3: 编译 + 浏览器验证** — `/files`:媒体文件出现 checkbox,选中 2 个 → 按钮显示「识别选中(2)」→ 点击后 toast 报告 queued/skipped,树刷新后对应文件出现「识别中」或「待审」徽章,且原「已完成」徽章不受影响;`/files/detail/<path>`:头部出现 AI pill 与按钮,点「重新识别」弹确认 → 状态变「识别中」;侧栏同目录文件的 AI 徽章正确。
- [ ] **Step 4: 回归检查** — 手动打标/取消打标、标记完成、标签组切换、搜索页均正常(这些是本次改动最容易碰坏的既有功能)。
- [ ] **Step 5: 提交**

```bash
git add cmd/webui/static/js/pages/file/tree.js cmd/webui/static/js/pages/file/detail.js cmd/webui/templates/file.html cmd/webui/templates/file_detail.html cmd/webui/static/css/components/tags.css cmd/webui/static/css/components/tree.css
git commit -m "feat: surface AI status and batch recognition in file UI"
```

---

### Task 15: 验收、版本与文档

**Files:**
- Modify: `internal/constant/version.go:3`(`0.5.6` → `0.6.0`)、`cmd/webui/constant/version.go:3`(`4.2.1` → `4.3.0`)
- Modify: `docs/CHANGELOG.md:5` 前插入新版本节、`docs/CHANGELOG_webui.md:5` 前插入新版本节
- Modify: `AGENTS.md:10,23,24,27,28`(纠正过期陈述)

**Interfaces:**
- Consumes: 前 14 个任务的全部产出。
- Produces: 可发布的版本记录与准确的仓库指南。

- [ ] **Step 1: 全量测试** — `make test`。预期:CLI 构建成功、fixture 脚本跑完、`go test ./... -v` 全绿(含新增约 30 个测试)。失败就修,不得跳过。
- [ ] **Step 2: 逐条核对 spec §10 验收口径** — ① 批量发起不阻塞页面、状态可见、失败可见可重试(Task 12/14 浏览器验证);② 人工通过前标签库与文件标签无模型输出(Task 9 测试 + 浏览器确认);③ 通过后 `./tagfile.out search tag --and <新标签>` 能命中该文件(**用 Makefile 构建的 `tagfile.out`,不得另建二进制**);④ 改 `ai_max_frames` 后对视频重新识别,`frame_count` 随之变化;⑤ `make test` 通过。把每条的实际命令与输出记进提交说明。
- [ ] **Step 3: 写 CHANGELOG 与 bump 版本** — 核心节 `## [0.6.0] - 2026-11-13` + `### Added / 新增`,WebUI 节 `## [4.3.0] - 2026-11-13` + `### Added / 新增`;每条目英文行紧接中文行、条目间空行;两个 version.go 同 commit 改。注意 bump `CoreVersion` 会让 `bootstrap.backupDBIfNeeded` 在下次启动时自动备份 DB,属预期行为。
- [ ] **Step 4: 纠正 AGENTS.md** — Go 版本改 1.25.0;`TagGroupTag.Value` 改 `TagGroupTag.TagValue`;`IncrementTagCnt/DecrementTagCnt` 改为「repo 的 `IncrementTagCnt(fileID, delta)`,减计数传 -1」;WebUI 路由改为 Gin(`api.Router.RegisterRoutes` + `handlers.Server.RegisterRoutes`);删掉 `_return` 与 `cells[]` 两条不存在的约定,换成「form-urlencoded + `respond` 信封」「Alpine `inlineEdit` 组件」「新页面需同时注册 main.go 页面名切片/路由/导航」。
- [ ] **Step 5: 提交**

```bash
git add internal/constant/version.go cmd/webui/constant/version.go docs/CHANGELOG.md docs/CHANGELOG_webui.md AGENTS.md
git commit -m "docs: record AI vision tagging release and correct stale guide"
```

---

## 留给实现者的自由裁量

spec §9 与本计划刻意不锁死的点:提示词具体措辞(契约与必含信息已定)、dHash 阈值调优、临时目录清理时机、审核页是按文件分组还是平铺、轮询间隔(建议 3s)、`GET /api/ai/tasks` 的 pageSize(建议 20)。这些改动不影响任何接口签名,实现者自行决定即可。