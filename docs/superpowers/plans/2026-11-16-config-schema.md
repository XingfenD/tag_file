# 设置页 schema 化改造 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把设置页从自由 KV 表单改为 schema 驱动的分组结构化表单：写入校验、严格模式整组保存、info 提示、可折叠分组、未知键高级区。

**Architecture:** 新增叶子包 `internal/service/config/schema`（类型 + 校验器 + 注册表），消费方（ai、bootstrap）在 `init()` 注册字段以绕开 ai→config 的 import 环；config 服务按注册表做写入校验与禁删，bootstrap 种子改为遍历注册表；WebUI 新增 schema/批量保存接口，前端通用渲染器按类型出控件。

**Tech Stack:** Go 1.25 + Gin + GORM/SQLite（后端），Alpine.js + 原生 fetch（前端，无新依赖）。

**Spec:** `docs/superpowers/specs/2026-11-16-config-schema-design.md`（执行者必须同时读 spec；本计划与 spec 冲突时以 spec 为准并上报）

## Global Constraints

- 分支 `feat/config-schema`（已存在，直接在上面提交；禁止提交到 master）。
- 模块路径 `github.com/XingfenD/tag_file`（大写 D）。
- 不改 KV 表结构、不加迁移；读取端回落逻辑（`ai.Settings()` 等）一行不动。
- 写入校验范围 = 读取端接受范围，精确镜像：interval 为 Min=0 排他（任意 >0），threshold 1..64 闭区间，maxFrames 1..200 闭区间。
- `ollama_model` 空串合法（空 = 未配置）；string/text/secret 类型允许空串，url 不允许。
- 严格模式：组内任一字段（含未改动的存量值）非法 → 整组拒存，逐键报错。
- secret 键的值永不出现在任何读取接口响应中（schema 接口与列表接口都只给 `has_value`）。
- 错误文案中文、带键名与合法范围；HTTP 响应走 `respond(c, msg, msgType, extra...)`，`msgType="error"` 即 HTTP 400。
- DELETE 处理器参数只读 `c.Query`/`c.Param`；表单读 `c.PostForm`/`c.GetPostForm`/`c.PostFormArray`。
- 测试只走 `go test`（单包调试 `go test ./path -run Name -v`，全量 `make test`）；storage 是进程级单例，测试不得并行。
- 模板 `{{define}}` 不嵌套；新组件放 `cmd/webui/static/js/core/components.js` 的 `alpine:init` 内。
- CHANGELOG 中英双行紧邻、条目间空行、新版本在上；bump CHANGELOG 的同一 commit 里更新对应版本常量。
- 注册表是进程级全局：测试注册自定义字段必须用独一无二的键名/组名，避免与 init() 注册冲突。

## Review Focus

1. **空 `ollama_model` 被新校验误杀**（空=未配置是产品语义）→ Task 3 测试 `TestRegisterAIFields` 断言 `Validate(modelField, "") == nil`；Task 4 测试空串 Set 成功。
2. **存量非法值把用户锁死却无处可见**：schema 接口必须标记非法存量值，折叠组标题必须带可见标记 → Task 7 测试 `TestConfigSchemaMarksInvalidStored`；Task 9 浏览器清单第 4 项。
3. **secret 值从列表接口泄漏**（`HandleListConfig` 不带 unknown=1 时返回全部键）→ Task 7 测试 `TestListConfigMasksSecret`。
4. **JS 正则预览与 Go RE2 不一致**（如 lookahead 在 JS 合法、Go 非法）：服务端拒绝必须逐键 surfaced，前端不得吞错 → Task 7 测试 `TestSetGroupRejectsRE2InvalidRegex`；Task 8 保存失败必须展示 `d.failures`。
5. **旧测试用服务层塞垃圾值**：写入校验上线后 `ai/config_test.go` 的回落测试会挂 → Task 4 Step 5 显式改为直接写库，且全量 `make test` 必须绿。

---

### Task 1: schema 包 — 类型与校验器

**Files:**
- Create: `internal/service/config/schema/schema.go`（类型定义）
- Create: `internal/service/config/schema/validate.go`
- Test: `internal/service/config/schema/validate_test.go`

**Interfaces:**
- Consumes: 无（叶子包，不 import 任何 service/model 包，只用标准库）。
- Produces（后续所有任务依赖这些名字）:

```go
package schema

type FieldType string
const (
    TypeString FieldType = "string"; TypeText FieldType = "text"
    TypeURL FieldType = "url";       TypeInt FieldType = "int"
    TypeFloat FieldType = "float";   TypeDuration FieldType = "duration"
    TypeBool FieldType = "bool";     TypeEnum FieldType = "enum"
    TypeList FieldType = "list";     TypeSecret FieldType = "secret"
    TypeRegex FieldType = "regex" // 仅作 list 的 ItemType
)
type Option struct{ Value, Label string }
type Condition struct{ Key, Equals string }
type Field struct {
    Key, Group, Label, Description string
    Type     FieldType
    Default  string
    Min, Max *float64
    MinExclusive, MaxExclusive bool
    Pattern  string
    Options  []Option
    ItemType FieldType
    Unit     string
    ShowIf   *Condition
}
type Group struct{ ID, Title string; Order int }
func Ptr(v float64) *float64
func Validate(f Field, value string) error // 错误文案中文、含合法范围，不含键名（调用方包装键名）
```

- [ ] **Step 1: 写失败测试** `validate_test.go`（package `schema_test`），表驱动，每行一个用例：

```go
func TestValidate(t *testing.T) {
    cases := []struct{ name string; f schema.Field; value string; ok bool }{
        {"string 空串合法", schema.Field{Type: schema.TypeString}, "", true},
        {"string pattern 命中", schema.Field{Type: schema.TypeString, Pattern: `^[a-z]+$`}, "abc", true},
        {"string pattern 不命中", schema.Field{Type: schema.TypeString, Pattern: `^[a-z]+$`}, "A1", false},
        {"url 合法", schema.Field{Type: schema.TypeURL}, "http://127.0.0.1:11434", true},
        {"url 空串非法", schema.Field{Type: schema.TypeURL}, "", false},
        {"url 无 scheme 非法", schema.Field{Type: schema.TypeURL}, "127.0.0.1:11434", false},
        {"url ftp 非法", schema.Field{Type: schema.TypeURL}, "ftp://x/y", false},
        {"int 边界闭区间", schema.Field{Type: schema.TypeInt, Min: schema.Ptr(1), Max: schema.Ptr(64)}, "64", true},
        {"int 越上界", schema.Field{Type: schema.TypeInt, Min: schema.Ptr(1), Max: schema.Ptr(64)}, "65", false},
        {"int 非数字", schema.Field{Type: schema.TypeInt}, "not-a-number", false},
        {"float 排他下界拒 0", schema.Field{Type: schema.TypeFloat, Min: schema.Ptr(0), MinExclusive: true}, "0", false},
        {"float 排他下界收小正数", schema.Field{Type: schema.TypeFloat, Min: schema.Ptr(0), MinExclusive: true}, "0.05", true},
        {"float 拒负数", schema.Field{Type: schema.TypeFloat, Min: schema.Ptr(0), MinExclusive: true}, "-1", false},
        {"bool 只认 true/false", schema.Field{Type: schema.TypeBool}, "TRUE", false},
        {"enum 命中选项", schema.Field{Type: schema.TypeEnum, Options: []schema.Option{{Value: "a"}, {Value: "b"}}}, "b", true},
        {"enum 未命中", schema.Field{Type: schema.TypeEnum, Options: []schema.Option{{Value: "a"}}}, "c", false},
        {"list 合法 JSON 正则数组", schema.Field{Type: schema.TypeList, ItemType: schema.TypeRegex}, `["^\\.","^x$"]`, true},
        {"list 空数组合法", schema.Field{Type: schema.TypeList, ItemType: schema.TypeRegex}, `[]`, true},
        {"list 非法 JSON", schema.Field{Type: schema.TypeList, ItemType: schema.TypeRegex}, `not json`, false},
        {"list 非法正则项", schema.Field{Type: schema.TypeList, ItemType: schema.TypeRegex}, `["("]`, false},
        {"list 非数组 JSON", schema.Field{Type: schema.TypeList, ItemType: schema.TypeRegex}, `{"a":1}`, false},
        {"secret 任意串合法", schema.Field{Type: schema.TypeSecret}, "sk-xxx", true},
        {"duration 正秒数", schema.Field{Type: schema.TypeDuration, Min: schema.Ptr(0), MinExclusive: true}, "180", true},
    }
    // 逐行 t.Run：ok=true 断言 err==nil；ok=false 断言 err!=nil 且文案非空
}
```

- [ ] **Step 2: 跑测试确认失败**：`go test ./internal/service/config/schema/ -run TestValidate -v`，预期编译失败（包不存在）。
- [ ] **Step 3: 实现** `schema.go`（上述类型 + `Ptr`）与 `validate.go`。`Validate` 按 `f.Type` switch 分发，一个类型一个分支：url 用 `net/url.Parse` 后校验 scheme ∈ {http, https} 且 Host 非空；int/float/duration 解析后按 Min/Max（含排他标记）判定；list 用 `json.Unmarshal` 到 `[]string`，ItemType 为 regex 时逐项 `regexp.Compile`；未知类型返回错误（不静默放行）。spec 6.1 提到的 MaxLength 在 6.2 结构体中不存在且本期无使用者，不实现（以结构体为准）。
- [ ] **Step 4: 跑测试确认通过**：同 Step 2 命令，预期 PASS。
- [ ] **Step 5: Commit**：`git add internal/service/config/schema/ && git commit -m "feat: add config schema field types and per-type validators"`

### Task 2: schema 包 — 注册表与注册自检

**Files:**
- Create: `internal/service/config/schema/registry.go`
- Test: `internal/service/config/schema/registry_test.go`

**Interfaces:**
- Consumes: Task 1 的 `Field`/`Group`/`Validate`。
- Produces:

```go
func RegisterGroup(g Group)            // panic：重复组 ID
func Register(f Field)                 // panic：重复键 / 组未注册 / Validate(f, f.Default) != nil
func Lookup(key string) (Field, bool)
func Groups() []Group                  // 按 Order 升序
func FieldsOf(groupID string) []Field  // 按 Key 升序；未知组返回 nil
func All() []Field                     // 按 Key 升序
```

- [ ] **Step 1: 写失败测试**（package `schema_test`；自定义组/键用 `testreg_` 前缀避免与 init 注册冲突）：

```go
func TestRegisterSelfCheck(t *testing.T)
// 三个子测试，均用 defer recover 断言 panic：
// "坏默认值"：Register(Field{Key:"testreg_bad", Group:"testreg_g", Type:TypeInt, Default:"abc"}) → panic
// "重复键"：先 Register 合法 Field{Key:"testreg_dup",...}，再注册同键 → panic
// "未注册组"：Register(Field{Key:"testreg_nogroup", Group:"testreg_absent",...}) → panic
func TestRegistryOrdering(t *testing.T)
// RegisterGroup{ID:"testreg_b", Order:2} 与 {ID:"testreg_a", Order:1}；
// Groups() 中 testreg_a 在 testreg_b 前；FieldsOf/All 按 Key 升序；
// Lookup("testreg_x") 返回注册时的 Field 且 ok=true；Lookup("no_such_key") ok=false。
```

- [ ] **Step 2: 跑测试确认失败**：`go test ./internal/service/config/schema/ -run 'TestRegister|TestRegistry' -v`，预期编译失败。
- [ ] **Step 3: 实现** `registry.go`：包级 `map[string]Field` + `map[string]Group` + `sync.Mutex`（注册发生在 init 期，读多写少，锁保测试安全）；`Register` 先查重组、再查重键、再 `Validate(f, f.Default)`，任一不过即 panic（文案含键名与原因，风格对齐 `ai.mustParseInt`）。
- [ ] **Step 4: 跑测试确认通过**：同 Step 2，预期 PASS；再跑 `go test ./internal/service/config/schema/ -v` 确认 Task 1 未回归。
- [ ] **Step 5: Commit**：`git add internal/service/config/schema/ && git commit -m "feat: add config schema registry with fail-fast self check"`

### Task 3: 消费方注册 — ai 五键 + scan_exclude

**Files:**
- Create: `internal/service/ai/config_schema.go`
- Create: `internal/bootstrap/config_schema.go`
- Test: `internal/service/ai/config_schema_test.go`

**Interfaces:**
- Consumes: Task 2 的 `RegisterGroup`/`Register`/`Lookup`/`Validate`；ai 包现有常量 `KeyOllamaURL`、`DefaultOllamaURL`、`KeyOllamaModel`、`DefaultOllamaModel`、`KeyFrameInterval`、`DefaultFrameInterval`、`KeyDiffThreshold`、`DefaultDiffThreshold`、`KeyMaxFrames`、`DefaultMaxFrames`、`maxFramesLimit`(200)、`maxDiffThreshold`(64)（同包可见，不导出、不复制）；bootstrap 的 `defaultExcludePatterns`。
- Produces: 进程启动后注册表含组 `scan`(Order 1, Title 扫描设置)、`ai`(Order 2, Title AI 识别) 与 6 个字段。后续任务经 `schema.Lookup`/`FieldsOf` 消费，不直接引用这两个新文件。

- [ ] **Step 1: 写失败测试** `config_schema_test.go`（package `ai_test`）：

```go
func TestRegisterAIFields(t *testing.T)
// schema.Lookup("ollama_url")：ok，Type==TypeURL，Default==ai.DefaultOllamaURL，Group=="ai"
// schema.Lookup("ollama_model")：ok，Type==TypeString，Default==""，
//   且 schema.Validate(f, "") == nil（空=未配置，Review Focus 1）
// schema.Lookup("ai_frame_interval")：Min!=nil 且 *Min==0 且 MinExclusive==true，Max==nil
// schema.Lookup("ai_frame_diff_threshold")：*Min==1，*Max==64，均闭区间
// schema.Lookup("ai_max_frames")：*Min==1，*Max==200
// 每个字段断言 Validate(f, f.Default)==nil 且 Label/Description 非空
```

- [ ] **Step 2: 跑测试确认失败**：`go test ./internal/service/ai/ -run TestRegisterAIFields -v`，预期 FAIL（Lookup ok=false）。
- [ ] **Step 3: 实现注册**。`ai/config_schema.go` 的 `init()`：`RegisterGroup(schema.Group{ID:"ai", Title:"AI 识别", Order:2})` 后注册 5 个 Field，Default 一律引用现有常量；Label/Description 沿用 bootstrap 现有文案（「Ollama 服务地址」「Ollama 视觉模型名称（空表示未配置，识别时报错）」「视频抽帧间隔（秒）」「近似帧过滤阈值（与上一保留帧的 dHash 汉明距离小于该值的帧被丢弃）」「单个视频最多抽取的帧数」）；interval Unit「秒」。`bootstrap/config_schema.go` 的 `init()`：`RegisterGroup(schema.Group{ID:"scan", Title:"扫描设置", Order:1})`，注册 `scan_exclude`：TypeList、ItemType TypeRegex、Default 为 `json.Marshal(defaultExcludePatterns)`（与现种子同源）、Description「扫描排除正则列表」。
- [ ] **Step 4: 跑测试确认通过**：同 Step 2，预期 PASS。注意本测试进程不 import bootstrap，`scan` 组不在注册表属正常，不要在此断言它。
- [ ] **Step 5: Commit**：`git add internal/service/ai/config_schema.go internal/service/ai/config_schema_test.go internal/bootstrap/config_schema.go && git commit -m "feat: register ai and scan config fields into the schema registry"`

### Task 4: config 服务 — 写入校验与已知键禁删

**Files:**
- Modify: `internal/service/config/service.go`（Set/Update/Delete）
- Modify: `internal/service/ai/config_test.go`（垃圾值改直接写库，见 Step 5）
- Test: `internal/service/config/service_test.go`（新建）

**Interfaces:**
- Consumes: `schema.Lookup`/`Validate`；现有 `configRepo.Set/Update/Delete/Get`。
- Produces: `Set`/`Update`/`Delete` 签名不变，行为收紧：

```go
// Set/Update：key 命中注册表 → schema.Validate 不过则返回
//   fmt.Errorf("配置 %s 不合法: %w", key, err)，拒存；未命中 → 仅键名非空检查（现状）。
// Delete：key 命中注册表 → 返回错误「配置 'X' 是系统配置，不能删除；如需还原请使用恢复默认」；
//   未命中 → 照旧。
```

- [ ] **Step 1: 写失败测试** `service_test.go`（package `config_test`；import `_ "github.com/XingfenD/tag_file/internal/service/ai"` 触发注册；自建 DB helper：`storage.InitDB(filepath.Join(t.TempDir(),"test.db"))` + `AutoMigrate(&model.Config{})`，参照 `cmd/webui/api/router_test.go:24-38`）：

```go
func TestSetValidatesKnownKey(t *testing.T)
// Set("ai_max_frames","201") → err 非 nil 且含 "ai_max_frames"；库中无此行
// Set("ai_max_frames","200") → nil；Set("ai_frame_interval","0") → err；Set("ai_frame_interval","0.05") → nil
// Set("ollama_url","no-scheme") → err；Set("ollama_model","") → nil（Review Focus 1）
func TestSetUnknownKeyPassthrough(t *testing.T)
// Set("my_custom_key","任意值") → nil；Set("","x") → err
func TestUpdateValidatesKnownKey(t *testing.T)
// 先 Set("ai_max_frames","10")，Update("ai_max_frames","abc") → err 且库值仍 "10"
func TestDeleteGuardsKnownKey(t *testing.T)
// Set("ollama_url", 合法值) 后 Delete("ollama_url") → err 含「恢复默认」，行仍在；
// Set("my_custom_key","v") 后 Delete("my_custom_key") → nil，行消失
```

- [ ] **Step 2: 跑测试确认失败**：`go test ./internal/service/config/ -v`，预期 FAIL（校验/禁删未实现）。
- [ ] **Step 3: 实现** service.go 三处收紧（签名见上）。校验辅助抽一个私有函数 `validateKnown(key, value string) error` 供 Set/Update 共用。
- [ ] **Step 4: 跑测试确认通过**：同 Step 2，预期 PASS。
- [ ] **Step 5: 修复被新校验打破的存量测试**（Review Focus 5）：`internal/service/ai/config_test.go` 中所有经 `cfgSvc.Set` 写入**非法值**的调用（`"not-a-number"`、`"0"`、`"-1"`、`"201"`、`"100000"`、`"65"`、`"1000"` 等）改为直接写库：新增 helper `seedRaw(t, key, value)` 用 `storage.GetDB().Clauses(同 repo.Set 的 upsert).Create(&model.Config{...})`（或先 Delete 再 Create），替换这些调用；合法值调用不动。跑 `go test ./internal/service/ai/ -v` 确认全绿——回落测试的语义（读取端容忍垃圾）保持不变。
- [ ] **Step 6: 全量回归**：`make test`，预期全绿（worker_test 的 Set 调用都是合法值，smoke 测试用未知键 smoke，不受影响；若有其他挂点，同法改直接写库，不改校验语义）。
- [ ] **Step 7: Commit**：`git add internal/service/config/ internal/service/ai/config_test.go && git commit -m "feat: validate known config keys on write and guard them against deletion"`

### Task 5: bootstrap 种子改为遍历注册表

**Files:**
- Modify: `internal/bootstrap/init.go:25-75`（seedDefaultConfig）
- Test: `internal/bootstrap/init_test.go`（扩展）

**Interfaces:**
- Consumes: `schema.All()`；Task 3 的注册（bootstrap import ai，注册链可达）。
- Produces: `seedDefaultConfig(db *gorm.DB)` 签名不变；行为 = 对 `schema.All()` 逐键「缺才插入」，值取 `Field.Default`、描述取 `Field.Description`。手写 defaults 清单删除；`defaultExcludePatterns` 保留（Task 3 的注册引用它）。

- [ ] **Step 1: 写测试**（追加到 init_test.go；现有字面量钉值测试全部保留不动）。本任务是行为保持的重构，测试分两类：

```go
// A 类：钉住现有行为，重构前后都必须绿
func TestSeedCoversRegistry(t *testing.T)
// useTempWorkspace + bootstrap.InitDB() 后：
// schema.All() 的每个键都存在于 configRows(t)，值==Field.Default，Description==Field.Description
func TestSeedFillsMissingKeyOnExistingDB(t *testing.T)
// 先 InitDB；手工 DELETE 一行（如 ollama_model）；再 InitDB；该行回来且值==Default；
// 手工把 scan_exclude 改成 `["^custom$"]` 再 InitDB，值保持 `["^custom$"]`（只补缺不覆盖）

// B 类：新行为，重构前必红——手写清单不认识测试注册的键
func TestSeedCoversNewlyRegisteredField(t *testing.T)
// schema.Register(schema.Field{Key:"testseed_new", Group:"scan", Type:schema.TypeString,
//   Default:"v", Label:"测试", Description:"测试键"})（组 scan 已由 bootstrap init 注册）；
// InitDB() 后 configRows(t) 含 testseed_new 且值=="v"
```

- [ ] **Step 2: 跑测试确认红绿分布**：`go test ./internal/bootstrap/ -run TestSeed -v`，预期 A 类 PASS（钉住现状）、B 类 FAIL（手写清单不含 testseed_new）。
- [ ] **Step 3: 实现**：seedDefaultConfig 改为 `for _, f := range schema.All()` 循环，逐键 Count→缺则 Create（保持现有逐键语义与日志风格）。
- [ ] **Step 4: 跑测试确认全绿**：`go test ./internal/bootstrap/ -v`，预期全绿（含既有字面量钉值测试——它们现在同时钉住注册表与种子两条路径）。
- [ ] **Step 5: Commit**：`git add internal/bootstrap/ && git commit -m "refactor: seed default config from the schema registry"`

### Task 6: 批量保存 — repo 事务 + 服务严格模式 + 高级区数据源

**Files:**
- Create: `internal/repository/config/set_many.go`
- Modify: `internal/service/config/service.go`
- Test: `internal/service/config/service_test.go`（追加）

**Interfaces:**
- Consumes: `schema.FieldsOf`/`Lookup`/`Validate`；repo 现有 upsert 写法（`set.go` 的 `clause.OnConflict`）。
- Produces:

```go
// repo（新文件，一操作一文件约定）
func (r *Repository) SetMany(values map[string]string) error
// r.db.Transaction 内逐键执行与 Set 相同的 upsert；任一键失败整体回滚。

// service
type ValidationErrors map[string]string // Error() 按键排序拼接 "key: msg"，分号分隔
func (s *Service) SetGroup(group string, values map[string]string) error
// 未知组（FieldsOf 为空）→ 普通 error「未知配置分组 'X'」。
// 逐字段：values 缺该键 → ValidationErrors[key]="缺少字段"；
//         Validate 不过 → ValidationErrors[key]=err.Error()；
// values 中不属于该组的键 → ValidationErrors[key]="不属于该分组"。
// 有任何错误 → 返回 ValidationErrors，一个键都不写；全过 → repo.SetMany。
func (s *Service) ListUnknown(offset, limit int) ([]model.Config, int64, error)
func (s *Service) SearchUnknown(keyword string, offset, limit int) ([]model.Config, int64, error)
// 经 repo.List()/Search() 以 limit 0 取全量，过滤掉 schema.Lookup 命中的键，
// 返回过滤后总数与 [offset, offset+limit) 切片（配置表规模小，全量过滤可接受）。
```

- [ ] **Step 1: 写失败测试**（追加到 service_test.go）：

```go
func TestSetGroupStrictAllOrNothing(t *testing.T)
// 先 SetGroup("ai", 全 5 键合法值) 成功；
// 再提交全 5 键但 ai_max_frames="201"：err 是 ValidationErrors 且含 "ai_max_frames"；
// 库中 ollama_url 等其余 4 键保持第一次的值（一个都没写，Review Focus 2 的服务端半边）
func TestSetGroupMissingAndForeignKeys(t *testing.T)
// 缺 ai_max_frames → ValidationErrors["ai_max_frames"]=="缺少字段"
// 多塞 "scan_exclude" → ValidationErrors["scan_exclude"]=="不属于该分组"
// SetGroup("no_such_group", ...) → 普通 error
func TestSetGroupAcceptsEmptyModel(t *testing.T)
// 全 5 键、ollama_model="" → nil
func TestListUnknownFiltersKnownKeys(t *testing.T)
// Set("ollama_url",合法值)、Set("custom_a","1")、Set("custom_b","2")；
// ListUnknown(0,10) 只含 custom_a/custom_b，total==2；SearchUnknown("custom",0,10) 同；
// ListUnknown(1,1) 返回 1 条、total 仍 2
```

- [ ] **Step 2: 跑测试确认失败**：`go test ./internal/service/config/ -run 'TestSetGroup|TestListUnknown' -v`，预期编译失败。
- [ ] **Step 3: 实现** repo `set_many.go` 与 service 三个新方法 + `ValidationErrors`（签名见上）。
- [ ] **Step 4: 跑测试确认通过**：同 Step 2，预期 PASS；再跑 `go test ./internal/service/config/ ./internal/repository/config/ -v`。
- [ ] **Step 5: Commit**：`git add internal/repository/config/set_many.go internal/service/config/ && git commit -m "feat: add strict group config save and unknown-key listing"`

### Task 7: WebUI API — schema 接口、批量保存、列表补描述与 secret 掩码

**Files:**
- Modify: `cmd/webui/api/config.go`
- Modify: `cmd/webui/api/api.go:113-119`（Config 路由组）
- Modify: `cmd/webui/api/routes_smoke_test.go`（补两条 smoke）
- Test: `cmd/webui/api/config_schema_test.go`（新建）

**Interfaces:**
- Consumes: Task 6 的 `SetGroup`/`ValidationErrors`/`ListUnknown`/`SearchUnknown`；`schema.Groups`/`FieldsOf`/`Lookup`/`Validate`；现有 `newTestEngine`/`doRequest`（router_test.go）。
- Produces（Task 8 前端依赖的 JSON 契约）:

```
GET /api/config/schema →
{ok, msg, groups: [{id, title, order, fields: [{
  key, label, description, type, default,
  min, max, min_exclusive, max_exclusive,   // 数值边界，null 表示无
  pattern, options: [{value,label}], item_type, unit, show_if,
  value,            // 当前值；键缺失时 = default；secret 键恒为 ""
  has_value,        // 仅 secret 键有意义：库中是否存在非空值
  invalid, invalid_reason   // 存量值按 Validate 判定（secret 不判，恒 false）
}]}]}
// 任一键读取失败 → 整个接口 respond error（前端加载守卫据此禁用全页）。

POST /api/config/group  form: group=<组id>&<键>=<值>... →
成功 {ok:true, msg:"分组 «X» 已保存"}；
校验失败 HTTP 400 {ok:false, msg:"配置校验失败", failures:[{key, msg}...]}；
未知组 → respond error。
// handler 用 c.GetPostForm(f.Key) 逐组字段收集（区分缺失与空串），只转发该组字段。

GET /api/config[?unknown=1][&q=&page=] →
ConfigItem 增加 description 字段；unknown=1 时走 ListUnknown/SearchUnknown；
secret 键（Lookup 命中且 Type==secret）value 置 ""、附 has_value（Review Focus 3）。
```

路由注册：`cfg.GET("/schema", rt.HandleConfigSchema)`、`cfg.POST("/group", rt.HandleSetConfigGroup)`（与现有 `PUT /:key` 无冲突：不同方法/静态段）。

- [ ] **Step 1: 写失败测试** `config_schema_test.go`（package `api`，复用 newTestEngine/doRequest；文件顶部 import `_ "github.com/XingfenD/tag_file/internal/bootstrap"` 使 scan 组在本测试进程注册——newTestEngine 只 import ai，不带 bootstrap）：

```go
func TestConfigSchemaShapeAndValues(t *testing.T)
// GET /api/config/schema：ok；groups 含 scan 与 ai 且 scan 在前（Order）；
// ai 组 fields 含 5 键；ollama_url 的 value=="http://127.0.0.1:11434"（未种子时取 default）；
// interval 字段 min_exclusive==true
func TestConfigSchemaMarksInvalidStored(t *testing.T)
// 直接写库 ollama_url="no-scheme"（storage.GetDB().Create，绕过服务校验）；
// GET schema：该字段 invalid==true、invalid_reason 非空、value=="no-scheme"
func TestSetGroupEndpointStrict(t *testing.T)
// POST /api/config/group "group=ai&ollama_url=http://127.0.0.1:11434&ollama_model=&ai_frame_interval=1&ai_frame_diff_threshold=10&ai_max_frames=201"
// → 400，ok=false，failures 含 key=="ai_max_frames"；库中 ollama_url 未被写入
// 全合法重发 → 200 ok=true；缺键重发 → failures 含「缺少字段」
func TestSetGroupRejectsRE2InvalidRegex(t *testing.T)
// POST group=scan，scan_exclude=`["(?=a)"]`（JS 合法、Go RE2 非法）→ 400 failures 含 scan_exclude
func TestListConfigDescriptionAndUnknown(t *testing.T)
// 直接写库带 Description 的未知键 → GET /api/config items 含 description；
// GET /api/config?unknown=1 不含 ollama_url 等已知键
func TestListConfigMasksSecret(t *testing.T)
// 测试内注册唯一 secret 字段（schema.RegisterGroup+Register，键 "testsecret_key"）并直接写库值 "sk-1"；
// GET /api/config：该键 value=="" 且 has_value==true；GET schema：同样不泄漏
```

- [ ] **Step 2: 跑测试确认失败**：`go test ./cmd/webui/api/ -run 'TestConfigSchema|TestSetGroup|TestListConfig' -v`，预期 FAIL/404。
- [ ] **Step 3: 实现** config.go 三个 handler 与 ConfigItem 扩展、api.go 两条路由。schema 响应用局部 JSON struct（`json:"...,omitempty"` 处理可空边界），不直接把 schema.Field 序列化（避免泄漏内部字段名风格）。
- [ ] **Step 4: 跑测试确认通过**：同 Step 2，预期 PASS。
- [ ] **Step 5: 补 smoke**：routes_smoke_test.go 表内加 `{"GET", "/api/config/schema", "", false}` 与 `{"POST", "/api/config/group", "group=ai", true}`（缺字段必拒）。跑 `go test ./cmd/webui/api/ -v` 全绿。
- [ ] **Step 6: Commit**：`git add cmd/webui/api/ && git commit -m "feat: add config schema and strict group-save endpoints"`

### Task 8: 前端 — schema 驱动渲染器 + info 提示 + 折叠分组 + 高级区

**Files:**
- Modify: `cmd/webui/static/js/core/components.js`（新增 infoTip 组件）
- Modify: `cmd/webui/templates/config.html`（content 块重写）
- Modify: `cmd/webui/static/js/pages/config/main.js`（重写）

**Interfaces:**
- Consumes: Task 7 的三个 JSON 契约；现有 `API`/`showToast`/`showConfirm`/`Modal.show`/`escHtml`；`alpine:init` 注册模式。
- Produces: 页面行为（Task 9 浏览器验收对照此清单）：
  1. 加载守卫：`GET /api/config/schema` 完成前全部输入与保存禁用，警告条可重试（沿用 aiLoaded 语义，整页级）。
  2. 组 → 卡片按 order 渲染；字段按 type 出控件：string/url → input text；text → textarea；secret → password 输入（占位「已设置，留空不修改」当 has_value）；int/float/duration → input number，**不设 min/max/step HTML 属性**（浏览器步进网格会误杀合法存量值，旧模板 config.html:86-88 注释记录过此坑），边界由 JS 按 schema 判定（含排他下界）并即时提示，服务端兜底；bool → checkbox（值 "true"/"false"）；enum → select；list → 行编辑器（一行一条，增删行，失焦即 `new RegExp` 预览标红，保存时 `JSON.stringify(去空行)`）。
  3. infoTip：Label 旁 ⓘ，hover/聚焦弹出纯文本简介（x-text，防注入），触屏点按切换；高级区键名旁复用（description 非空才显示）。
  4. 折叠：点组标题切换，状态存 localStorage 键 `tagfile_config_collapsed`（JSON 对象），默认展开；组内有 invalid 字段时标题显示可见标记（如红色计数徽标），折叠不藏问题。
  5. 每字段「恢复默认」= 值置 default（list 解析 default JSON 为行）；每组保存 POST /api/config/group 提交整组字段；失败时按 `d.failures` 逐键标红并 toast 首条 msg。
  6. 高级区：折叠卡片，`/api/config?unknown=1` 数据源，保留搜索/分页/增/改（Modal）/删（confirm）；新增表单只有键+值（描述输入框移除）；提示「未知键不会被程序读取」。
  7. 数据导出卡片原样保留；旧 AI 卡片与自由 KV 表单（已知键部分）删除。

- [ ] **Step 1: 实现 infoTip 组件**（components.js `alpine:init` 内，`Alpine.data('infoTip', ...)`）：状态 `{open:false}`，mouseenter/mouseleave/focusin/focusout/click 切换；模板由使用处写 `<span class="info-tip" x-data="infoTip" ...>ⓘ<span class="info-tip-pop" x-show="open" x-text="..."></span></span>`；CSS 追加到现有样式文件（绝对定位气泡，不引新依赖）。
- [ ] **Step 2: 重写 config.html**：加载守卫条 + `<template x-for="g in groups">` 组卡片（标题栏含折叠箭头与 invalid 徽标）+ 字段按 type 的 `x-if` 分支控件 + 高级区卡片 + 导出卡片。所有 `{{define}}` 保持顶层。
- [ ] **Step 3: 重写 main.js**：`configPage()` 按 Produces 清单实现；schema 读取失败 → `schemaLoaded=false` + 错误条 + 重试按钮；保存期间按钮禁用；secret 留空不提交该键。
- [ ] **Step 4: 构建验证**：`make build-webui`，预期编译通过；`go vet ./cmd/webui/`。模板解析错误会在启动时暴露，留待 Task 9 浏览器确认。
- [ ] **Step 5: Commit**：`git add cmd/webui/templates/config.html cmd/webui/static/js/ && git commit -m "feat: render settings page from config schema with tooltips and collapsible groups"`

### Task 9: 浏览器端到端验收

**Files:** 无代码改动（发现问题回到对应任务修复并重跑）。

**Interfaces:**
- Consumes: Task 8 页面；`scripts/setup_test_workspace.sh`（make test 的 setup 同源）。
- Produces: 验收记录（通过/失败逐项），作为整分支 review 的输入。

- [ ] **Step 1: 准备环境**：`bash scripts/setup_test_workspace.sh`；后台启动 `go run ./cmd/webui --source temp/test_workspace --addr :8088`。
- [ ] **Step 2: 造一条存量非法值**：`sqlite3 temp/test_workspace/.tagfile/tagfile.db "UPDATE config SET config_value='no-scheme' WHERE config_name='ollama_url'"`（测试造数，非迁移；sqlite3 缺失时用等价的临时 Go test 写库）。
- [ ] **Step 3: 浏览器逐项验收**（用 playwright-cli 技能或 browser-use MCP，打开 http://127.0.0.1:8088/config）：
  1. 扫描/AI 两组卡片渲染，字段控件类型正确，ⓘ hover 出简介；
  2. ollama_url 标红且 AI 组标题有可见标记；折叠 AI 组后标记仍在；刷新后折叠状态保持；
  3. 只改 ollama_model 点保存 → 被拒（严格模式），failures 指向 ollama_url；把 ollama_url 改回合法值后整组保存成功；
  4. scan_exclude 行编辑器：增行、删行、填 `(` 即时标红、填合法正则保存成功；填 `(?=a)` 保存被服务端拒绝且错误可见（Review Focus 4）；
  5. 「恢复默认」把 ai_max_frames 置回 10；高级区新增未知键 `browser_test`=1 成功、列表可见、编辑/删除可用；已知键不出现在高级区；
  6. 高级区新增表单无描述输入框；未知键 ⓘ 显示库中描述；
  7. 导出按钮仍可下载 SQL；
  8. 断网模拟（DevTools offline 或停服务）刷新页面 → 加载守卫禁用输入并给重试。
- [ ] **Step 4: 清理与回归**：停服务；`make test` 全绿；删除验收造的临时数据无需特别处理（temp workspace 为一次性）。
- [ ] **Step 5: Commit**（若验收触发了修复）：修复随对应任务提交；纯验收无改动则跳过。

### Task 10: CHANGELOG 与版本号

**Files:**
- Modify: `docs/CHANGELOG.md`、`internal/constant/version.go`
- Modify: `docs/CHANGELOG_webui.md`、`cmd/webui/constant/version.go`

**Interfaces:**
- Consumes: 全部已完成任务的实际行为（条目必须描述已实现的事实，含严格模式、secret 掩码、折叠徽标等决策）。
- Produces: core `0.6.0 → 0.7.0`，webui `4.3.0 → 4.4.0`。

- [ ] **Step 1: 写 core CHANGELOG**：`## [0.7.0] - 2026-11-16` 置顶，Added/Changed 分节，中英双行紧邻、条目间空行；覆盖：schema 叶子包与注册自检、消费方 init 注册、写入校验与禁删、SetGroup 严格模式与事务、bootstrap 种子收敛、ListUnknown/SearchUnknown。同 commit 更新 `CoreVersion = "0.7.0"`。
- [ ] **Step 2: Commit**：`git add docs/CHANGELOG.md internal/constant/version.go && git commit -m "docs: record core config schema changes in changelog (0.7.0)"`
- [ ] **Step 3: 写 webui CHANGELOG**：`## [4.4.0] - 2026-11-16` 置顶，同格式；覆盖：GET /api/config/schema、POST /api/config/group、列表 description/unknown=1/secret 掩码、schema 驱动页面、infoTip、折叠分组与 localStorage、正则行编辑器、高级区、描述输入框移除。同 commit 更新 `WebuiVersion = "4.4.0"`。
- [ ] **Step 4: Commit**：`git add docs/CHANGELOG_webui.md cmd/webui/constant/version.go && git commit -m "docs: record webui config schema changes in changelog (4.4.0)"`
- [ ] **Step 5: 终验**：`make test` 全绿 + `make build build-webui` 成功。