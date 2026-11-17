# 配置 InputBoxDescriptor 正交模型 + 规则协议 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用 ValueType × ContainerType × Constraint 正交模型替换 FieldType 枚举，校验/渲染/行为统一为后端单源、随 payload 下发的指令协议，前端成为通用执行器。

**Architecture:** strangler 渐进切换——新模型/规则引擎/块注册表先以不冲突的名字（`ValidateInputBox`/`LookupInputBox`/`AllInputBoxes`）与旧 API 共存，旧消费者保持绿；再按 config service → webui api → bootstrap+ai 消费的顺序逐个切换；最后删旧 API、改回规范名（`Validate`/`Lookup`/`All`）、bump 版本。`ai_frame_interval` 改定点数存储（Int+Shift），bootstrap 一次性迁移存量值。

**Tech Stack:** Go 1.25（Gin/GORM/SQLite）、Alpine.js、Go html/template。

**Spec:** `docs/superpowers/specs/2026-11-17-config-inputbox-descriptor-design.md`

**基线:** 从 `0438fbb`（当前 `docs/config-item-rules-design` HEAD，含扁平主题 WebUI 5.0.0）切出 `feat/config-inputbox-descriptor`。

## Global Constraints

- Module `github.com/XingfenD/tag_file`（大写 D）；Go 1.25.0；`internal/` 不可被外部 import；schema 是叶子包，不 import 任何 service 包。
- 分支命名 `{feat|fix|docs|chore}/{name}`；提交前 `git branch --show-current`，master 上不提交。
-  schema 仅 AutoMigrate，无 raw SQL 迁移；`configs` 表结构不变（迁移是纯数据）。
- 校验文案中文、含合法范围、不含键名（调用方包装）；项级 `第 %d 项%s`，下标从 1，与现状逐字对齐。
- 文案/控件映射单源在后端 `ruleDefs`；JS 与模板零副本（含「每行一条，正则表达式」「已设置，留空不修改」）。
- Gin 路由；请求体 `application/x-www-form-urlencoded`；响应经 `respond(c, msg, msgType, extra...)` 扁平为 `{ok, msg, ...}`，error → HTTP 400；前端以 `data.ok` 判成败。
- 模板 `{{define}}` 顶层不可嵌套；递归结构走 JS。
- CHANGELOG：核心改动记 `docs/CHANGELOG.md`，WebUI 改动记 `docs/CHANGELOG_webui.md`；同条目英文行紧接中文行（中间无空行），条目间空行；高版本在上。bump CHANGELOG 的同一 commit 更新对应版本常量。
- 版本目标：`internal/constant.CoreVersion` 0.7.1 → 0.8.0；`cmd/webui/constant.WebuiVersion` 5.0.0 → 5.1.0。
- 测试用 `go test`（快速循环 `go test ./pkg/... -run TestX -v`）或 `make test`；不造临时测试二进制。
- 定点数：存储/传输整数 `n`，语义值 `n / 10^Shift`；`ai.FrameIntervalShift = 1`；`DefaultFrameInterval = "10"`。
- 迁移账本 `.tagfile/config_migrations`（每行一个已应用 ID）；迁移 ID `ai_frame_interval_fixedpoint_v1`。

## Review Focus

1. **存量小数升级往返**：旧库 `ai_frame_interval="0.5"` 升级后必须变 `"5"`，且 `Settings().FrameInterval` 读回 0.5 秒（不是 5 秒、不是回落）。→ Task 5 迁移测试 + ai 消费测试钉住。
2. **损坏存量值迁移鲁棒性**：`ai_frame_interval="abc"` 时 `InitDB` 不得 panic，保留原值，设置页标 invalid，`Settings()` 回落默认。→ Task 5 测试钉住。
3. **secret 留空不覆盖（端到端）**：整组保存缺失 secret 键、或显式提交空串，已存密钥都不得被抹掉。→ Task 3 服务测试钉住（空串=保持是比现状更强的硬化，与「留空不修改」文案一致）。
4. **定点数传输契约**：非整数 wire（`"0.5"`）必须被服务端拒绝；显示 0.55（Shift=1）前端取整为 6 → 0.6。→ Task 2 引擎测试 + Task 6 前端换算钉住。
5. **未知指令降级**：前端删掉 `ruleCheckers.regex` 后页面仍加载、非法正则仍被服务端拒存回填；引擎遇到 render/behavior（无 Check）不得崩溃。→ Task 2 引擎测试 + Task 7 降级演练钉住。

---

### Task 1: 描述符模型 + 块注册表 + 合法性检查

**Files:**
- Create: `internal/service/config/schema/inputbox.go`（新模型类型 + `IsSecret` + `validateDescriptor`）
- Create: `internal/service/config/schema/registry_block.go`（块注册表）
- Create: `internal/service/config/schema/inputbox_test.go`
- Create: `internal/service/config/schema/registry_block_test.go`
- Modify: `internal/service/config/schema/schema.go`（仅追加 `Ptr` 泛型化；旧类型原样保留）

**Interfaces:**
- Produces: 类型 `ValueType`/`ContainerType`/`ConfigBlockDescriptor`/`InputBoxDescriptor`/`InputBoxConstraintDescriptor`/`IntConstraintDescriptor`/`StringConstraintDescriptor`/`BoolConstraintDescriptor`/`DynamicListConstraintDescriptor`/`StringFormat`/`Option`/`Condition`（字段与常量名严格按 spec §3）；`func (d *InputBoxDescriptor) IsSecret() bool`；`func RegisterBlock(b ConfigBlockDescriptor)`；`func LookupInputBox(key string) (*InputBoxDescriptor, bool)`；`func Blocks() []ConfigBlockDescriptor`；`func BlockOf(id string) *ConfigBlockDescriptor`；`func AllInputBoxes() []*InputBoxDescriptor`；`func Ptr[T any](v T) *T`。
- 旧 `Ptr(v float64)` 改为泛型后，旧调用 `schema.Ptr(0)`（int 默认推断）会编译失败——旧测试里的浮点边界同步改 `Ptr(0.0)` 等浮点字面量（仅改字面量，不改断言）。

- [ ] **Step 1: 确认分支与环境**

```bash
git branch --show-current   # 控制器已建 feat/config-inputbox-descriptor 并提交 spec/plan，必须显示它
go build ./...              # 基线绿
```

用户框架草稿不恢复（Ruling：草稿类型会被 `inputbox.go` 全量取代，且与恢复后的重复声明冲突；草稿保留在 stash，Task 7 确认后 drop）。

- [ ] **Step 2: 写模型与合法性失败测试** `inputbox_test.go`

表驱动 `TestValidateDescriptor`，断言 `validateDescriptor` 报错/放行。必含用例（值按 spec §5.1）：
- 未知 ValueType `"Float"` / 未知 Container `"Matrix"` → 错。
- Int + StringConstraint 非 nil → 错「约束与值类型/容器类型不匹配」。
- Single + IntConstraintList 非空 → 错；Array + IntConstraint 单例非 nil → 错。
- Array + IntConstraintList 空 → 错（len≥1）。
- DynamicArray + DynamicListConstraint{MinItems: Ptr(5), MaxItems: Ptr(2)} → 错；Single + DynamicListConstraint 非 nil → 错。
- IntConstraint{Shift: -1} → 错；{Min: Ptr(9.0), Max: Ptr(1.0)} → 错。
- String Single + Format regex → 错；String DynamicArray + Format regex → 放行。
- String DynamicArray + Required/Options/Secret/Multiline 任一 → 错（仅 Single）。
- Options 非空 + Secret → 错；Options Value 重复 → 错。
- Pattern `"("` → 错（描述符期编译）。
- 合法：Int/Single/{Min: Ptr(0.0), MinExclusive: true, Shift: 1}；String/DynamicArray/项约束{Format: regex}；String/Single/{Secret: true}。

`TestIsSecret`：Single+String+Secret → true；DynamicArray+Secret（非法声明，直接构造）→ false；Int+Secret → false。

- [ ] **Step 3: 运行确认失败**

Run: `go test ./internal/service/config/schema/ -run 'TestValidateDescriptor|TestIsSecret' -v`
Expected: FAIL（类型/函数未定义）。

- [ ] **Step 4: 实现模型与合法性检查**

`inputbox.go`：按 spec §3 定义全部类型与常量；`IsSecret()` = `d.Container==ContainerSingle && d.ValueType==ValueString && d.Constraint.StringConstraint!=nil && d.Constraint.StringConstraint.Secret`；`validateDescriptor(d InputBoxDescriptor) error` 实现 spec §5.1 全部规则（Pattern 在此编译）。`schema.go` 的 `Ptr` 改泛型 `func Ptr[T any](v T) *T`，旧测试浮点字面量修正。

- [ ] **Step 5: 写块注册表失败测试** `registry_block_test.go`

复用 `registry_test.go` 的 `mustPanic`/`panicMessage`（同包测试可调用）。键/块用 `testblk_` 前缀隔离。
- `TestRegisterBlockSelfCheck`：坏描述符（Int+StringConstraint）→ panic 含键名；重复块 ID → panic 含「重复」；重复键（跨块）→ panic 含键名+「重复」。
- `TestBlockRegistryOrdering`：先注册 Order 2 再 Order 1 → `Blocks()` 升序；`BlockOf` 块内描述符按 Key 升序；`BlockOf("testblk_absent")` → nil；`AllInputBoxes()` 跨块按 Key 升序；`LookupInputBox` 命中/未命中。

- [ ] **Step 6: 运行确认失败 → 实现 → 通过**

Run: `go test ./internal/service/config/schema/ -run 'TestRegisterBlock|TestBlockRegistry' -v` → FAIL。
`registry_block.go`：独立 `map[string]ConfigBlockDescriptor` + `map[string]*InputBoxDescriptor` 键索引 + `sync.Mutex`（与旧注册表互不相干）。`RegisterBlock`：重复块/键 panic；对每个描述符 `validateDescriptor` 失败 panic（默认值自检 Task 2 加）。`Blocks` 按 Order/ID 排序；`BlockOf` 返回副本指针、内部按 Key 排序；`AllInputBoxes` 跨块 Key 升序。
Run: `go test ./internal/service/config/schema/ -v` → 全 PASS（新旧共存）。

- [ ] **Step 7: 提交**

```bash
git add internal/service/config/schema/
git commit -m "feat: add InputBoxDescriptor model and block registry alongside legacy schema"
```

---

### Task 2: 规则协议 + 校验引擎

**Files:**
- Create: `internal/service/config/schema/rules.go`（RuleKind/RuleRole/Rule/RuleSet/ruleDefs/RulesOf/renderMessage + 各 Check）
- Create: `internal/service/config/schema/validate_inputbox.go`（`ValidateInputBox`）
- Create: `internal/service/config/schema/rules_test.go`
- Create: `internal/service/config/schema/validate_inputbox_test.go`
- Modify: `internal/service/config/schema/registry_block.go`（RegisterBlock 加默认值自检）

**Interfaces:**
- Consumes: Task 1 全部类型 + `validateDescriptor`。
- Produces: `RuleKind` 常量 `RuleRequired/RuleInt/RuleIntRange/RuleURL/RuleRegex/RulePattern/RuleOneOf/RuleBool/RuleItemCount/RuleFixedArity/RuleMultiline/RuleSecret`；`RuleRole` 常量 `RoleValidate/RoleRender/RoleBehavior`；`Rule{Kind RuleKind; Role RuleRole; Params map[string]string; Hint, Message, Widget string}`；`RuleSet{Rules []Rule; PositionalRules [][]Rule; ContainerRules []Rule}`；`func RulesOf(d InputBoxDescriptor) RuleSet`；`func ValidateInputBox(d InputBoxDescriptor, value string) error`。
- Params key 约定：int_range → `min`,`max`,`min_exclusive`,`max_exclusive`,`shift`；pattern → `pattern`；oneof → `options`（JSON `[{value,label}]`）；item_count → `min_items`,`max_items`；fixed_arity → `n`。布尔/数值参数序列化为字符串（`"true"`/`"1"`）。

- [ ] **Step 1: 写规则分解与文案失败测试** `rules_test.go`

- `TestRulesOfDecomposition`：Int/Single/{Min:Ptr(0.0),MinExclusive:true,Shift:1} → Rules 含 kind `int` 与 `int_range`（params shift="1", min="0", min_exclusive="true"）；String/Single/{Required,Format:url} → `required`+`url`；String/Single/{Options:[{a,A},{b,B}]} → `oneof`（widget "select", params options JSON）；String/Single/{Multiline} → `multiline`（role render, widget textarea, 无 Check 语义）；String/Single/{Secret} → `secret`（role behavior, widget password, hint "已设置，留空不修改"）；String/DynamicArray/{Format:regex}+DynamicListConstraint{MaxItems:Ptr(5)} → Rules 含 `regex`，ContainerRules 含 `item_count`；Array/Int/IntConstraintList 长度 2 → PositionalRules 长度 2，ContainerRules 含 `fixed_arity`（n="2"）。
- `TestRuleMessageRendering`：int_range{min:1,max:64} → 「取值超出合法范围，应为 [1, 64]」；{min:0,min_exclusive:true} → 「(0, +∞)」；oneof{a,b} → 「取值必须是以下之一: a、b」；pattern{^x$} → 「取值不符合匹配模式 ^x$」；regex → 「不是合法的正则表达式」。
- `TestRuleChecks`：各 validate 原语 Check 的正/反例（url 空串/无 scheme/ftp/合法；regex `"("` 错、`"^x$"` 对；int_range 越界/排他端；oneof 命中/未命中；required 空串错；int `"1.5"` 错；bool `"TRUE"` 错）。

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/service/config/schema/ -run 'TestRulesOf|TestRuleMessage|TestRuleChecks' -v` → FAIL。

- [ ] **Step 3: 实现规则注册表与分解**

`rules.go`：`ruleDef{Role, Hint, Message, Widget string; Check func(map[string]string,string) error}`；`ruleDefs` 按 spec §4.2 填全部原语（文案逐字按 spec/Step 1）。`RulesOf` 按 spec §4.3 分解；带 Widget 的指令按 secret>oneof>multiline 排序输出。`renderMessage(def, params)` 替换 `{{range}}`（区间写法复用旧 `describeRange` 逻辑，显示单位）、`{{candidates}}`（顿号连接）、`{{pattern}}`。Check 细分文案：url 返回「取值必须以 http:// 或 https:// 开头」「取值必须包含主机名」等（住 ruleDefs 条目内）。

- [ ] **Step 4: 运行通过**

Run: `go test ./internal/service/config/schema/ -run 'TestRulesOf|TestRuleMessage|TestRuleChecks' -v` → PASS。

- [ ] **Step 5: 写引擎失败测试** `validate_inputbox_test.go`

迁移旧 `validate_test.go` 全部语义到新模型（旧文件保留不动，Task 7 删），必含：
- string 空串合法 / pattern 命中/不命中/空串跳过。
- url 合法/空串/无 scheme/ftp 非法。
- int 闭区间 [1,64] 收 64 拒 65；非数字拒。
- **定点数**：Int/Single/{Min:Ptr(0.0),MinExclusive:true,Shift:1}：`"1"`（0.1s）通过、`"0"` 拒、`"0.5"` 拒（非整数 wire，Review Focus 4）、`"-1"` 拒。
- bool 只认 true/false。
- oneof 命中/未命中。
- DynamicArray+regex：`["^\\.","^x$"]` 合法、`[]` 合法、`not json` 拒、`["("]` 拒且文案恰为 `第 1 项不是合法的正则表达式`、`{"a":1}` 拒。
- DynamicArray+item_count{MaxItems:1}：两项拒。
- Array/Int/定长 2：`["1","2"]` 合法、`["1"]` 拒「需要恰好 2 项」、逐位约束（位 0 {Min:Ptr(1.0)}、位 1 {Max:Ptr(9.0)}）`["0","5"]` 拒。
- secret 任意串合法；render/behavior 指令不产生校验（String/Single/{Multiline,Secret} 任意串通过，Review Focus 5）。
- 坏描述符（Int+StringConstraint）→ ValidateInputBox 直接报错（与取值无关）。

- [ ] **Step 6: 运行确认失败 → 实现引擎 → 通过**

Run: `go test ./internal/service/config/schema/ -run TestValidateInputBox -v` → FAIL。
`validate_inputbox.go`：`ValidateInputBox` = `validateDescriptor` → 容器解析（Single 原样；Array/DynamicArray 必须 JSON 字符串数组，否则「取值必须是 JSON 字符串数组，例如 ["a","b"]」）→ 跑 ContainerRules 的 Check（传原始 value）→ 值级/项级跑 Rules（Array 用 PositionalRules 逐位），项级失败拼 `第 %d 项%s`。只跑 RoleValidate（有 Check）；render/behavior 跳过。
`registry_block.go`：RegisterBlock 对每个描述符追加 `ValidateInputBox(d, d.Default)` 失败 panic（含键名+「默认值」）。
Run: `go test ./internal/service/config/schema/ -v` → 全 PASS。

- [ ] **Step 7: 提交**

```bash
git add internal/service/config/schema/
git commit -m "feat: add rule protocol and InputBox validation engine"
```

---

### Task 3: ai 新声明 + config service 切换

**Files:**
- Modify: `internal/service/ai/config_schema.go`（追加新块注册 init；旧 init 暂留）
- Modify: `internal/service/ai/config_schema_test.go`（切到新注册表断言）
- Modify: `internal/service/config/service.go`（SetGroup/validateKnown/Delete 切新 API）
- Modify: `internal/service/config/service_test.go`（fixture 与定点数用例）

**Interfaces:**
- Consumes: `RegisterBlock`/`LookupInputBox`/`BlockOf`/`ValidateInputBox`/`IsSecret`。
- Produces: ai 新块（5 描述符，spec §10 映射）；service 新 API 路径。ai 旧 init 注册暂留（旧 api/bootstrap 仍读旧注册表），Task 5 删。

- [ ] **Step 1: 写 ai 新声明失败测试**

`config_schema_test.go` 改用 `schema.LookupInputBox`，断言：ollama_url String/Single 含 required+url 指令；ollama_model String/Single 无指令、空串合法；ai_frame_interval Int/Single、Default `"10"`、IntConstraint{Min:Ptr(0.0),MinExclusive:true,Shift:1}、Unit「秒」；diff_threshold Int [1,64]；max_frames Int [1,200]。`TestRegisterAIFieldBounds` 从 `IntConstraint` 读边界。

- [ ] **Step 2: 运行确认失败 → 追加 ai 新块注册 → 通过**

Run: `go test ./internal/service/ai/ -run TestRegisterAI -v` → FAIL。
`config_schema.go` 追加 `func init() { schema.RegisterBlock(schema.ConfigBlockDescriptor{ID:"ai",Title:"AI 识别",Order:2, InputBoxDescriptorList: []*schema.InputBoxDescriptor{...5 个...}}) }`（旧 init 保留）。Default/边界引用 ai 常量；frame_interval Default 用 `"10"`（此任务先写死字面量 `"10"`，Task 5 改引用 `DefaultFrameInterval`）。
Run: `go test ./internal/service/ai/ -v` → PASS。

- [ ] **Step 3: 写 service 切换失败测试**

`service_test.go`：`registerSecretFixture` 改 `RegisterBlock`（secret 描述符 String/Single/{Secret:true} + label String/Single）；`TestSetValidatesKnownKey` 的 `Set("ai_frame_interval","0.05")` 改为断言**拒绝**（非整数 wire），新增 `Set("ai_frame_interval","1")` 通过；`TestSetGroupSkipsUntouchedSecret` 增子例：显式提交 `{"testsecret_key":"", "testsecret_label":"v"}` → 密钥仍 `sk-1`（空串=保持，Review Focus 3）。其余断言不变。

- [ ] **Step 4: 运行确认失败 → 切换 service → 通过**

Run: `go test ./internal/service/config/ -v` → FAIL。
`service.go`：`SetGroup` 用 `schema.BlockOf(group)`（nil → 「未知配置分组 '%s'」），遍历 `block.InputBoxDescriptorList`；缺失或空串的 secret（`f.IsSecret()`）视为保持（跳过写入，从 values 删除该键以免 SetMany 写空）；非 secret 缺失 → 「缺少字段」。`validateKnown`/`Delete` 用 `LookupInputBox`；校验用 `ValidateInputBox`。
Run: `go test ./internal/service/config/ ./internal/service/ai/ -v` → PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/service/ai/config_schema.go internal/service/ai/config_schema_test.go internal/service/config/service.go internal/service/config/service_test.go
git commit -m "feat: register ai block and switch config service to InputBox schema"
```

---

### Task 4: webui api 切换 + 规则投影

**Files:**
- Modify: `cmd/webui/api/config.go`（HandleConfigSchema/HandleListConfig/HandleSetConfigGroup + 投影结构）
- Modify: `cmd/webui/api/config_schema_test.go`（blocks/input_boxes 契约 + fixture）

**Interfaces:**
- Consumes: `Blocks`/`BlockOf`/`LookupInputBox`/`ValidateInputBox`/`IsSecret`/`RulesOf`/`RuleSet`。
- Produces: `GET /api/config/schema` 新契约 `{blocks:[{id,title,order,input_boxes:[...]}]}`（spec §9）；投影结构 `schemaBlockResponse`/`schemaInputBoxResponse`/`schemaRuleResponse`（json tag: kind/role/params/hint/message/widget；input_boxes/positional_rules/container_rules/value_type）。

- [ ] **Step 1: 写新契约失败测试**

`config_schema_test.go`：`findGroup`→`findBlock`（读 `blocks`），`fields`→`input_boxes`。`TestConfigSchemaShapeAndValues`：scan 在 ai 前；ai 含 5 键；ollama_url value 默认；ai_frame_interval 的 `rules` 含 int_range（params.min_exclusive="true", shift="1"）。`TestConfigSchemaMarksInvalidStored`：`no-scheme` → invalid=true、invalid_reason 非空、value 原样。secret fixture 改 `RegisterBlock`；`TestListConfigMasksSecret`/`TestConfigSchemaMasksSecretDefault` 断言不变（value/default 掩码、has_value），secret 描述符投影含 kind secret、widget password、hint「已设置，留空不修改」。`TestSetGroupEndpointStrict`/`TestSetGroupRejectsRE2InvalidRegex` body 不变（`ai_frame_interval=1` 在新制合法）。新增 `TestConfigSchemaRuleCopy`：scan_exclude 的 rules 恰含 `{kind:"regex",role:"validate",hint:"每行一条，正则表达式",message:"不是合法的正则表达式"}`。

- [ ] **Step 2: 运行确认失败**

Run: `go test ./cmd/webui/api/ -run 'TestConfigSchema|TestListConfigMasks|TestSetGroup' -v` → FAIL。

- [ ] **Step 3: 实现投影与切换**

`config.go`：`HandleConfigSchema` 遍历 `Blocks()`，每块 `buildSchemaInputBox(d, stored)`：`RulesOf(d)` → 投影 Rules/PositionalRules/ContainerRules（hint/message/widget 取自 ruleDefs 渲染，params 原样）；value/has_value 来自存量；`IsSecret()` → value/default 掩码、恒不判 invalid；否则 `ValidateInputBox` 判 invalid+reason。`HandleListConfig` 掩码用 `LookupInputBox`+`IsSecret`。`HandleSetConfigGroup` 用 `BlockOf`。删旧 `schemaGroupResponse`/`schemaFieldResponse`。

- [ ] **Step 4: 运行通过**

Run: `go test ./cmd/webui/api/ -v` → PASS。

- [ ] **Step 5: 提交**

```bash
git add cmd/webui/api/config.go cmd/webui/api/config_schema_test.go
git commit -m "feat: serve config schema as blocks with served rule directives"
```

---

### Task 5: bootstrap 切换 + 定点数迁移 + ai 消费

**Files:**
- Modify: `internal/bootstrap/config_schema.go`（RegisterBlock，删旧 Register）
- Modify: `internal/bootstrap/init.go`（seed 走 AllInputBoxes；migrateConfigStorage + 账本）
- Modify: `internal/bootstrap/init_test.go`（字面量 + 迁移测试 + 新注册 fixture）
- Modify: `internal/service/ai/config.go`（FrameIntervalShift/DefaultFrameInterval/shiftedIntSetting）
- Modify: `internal/service/ai/config_schema.go`（删旧 init；frame_interval Default 引用 DefaultFrameInterval）

**Interfaces:**
- Consumes: `RegisterBlock`/`AllInputBoxes`；ai `KeyFrameInterval`/`FrameIntervalShift`。
- Produces: `func migrateConfigStorage(db *gorm.DB, dbPath string) error`；`ai.FrameIntervalShift = 1`；`ai.DefaultFrameInterval = "10"`；`func (s *Service) shiftedIntSetting(key string, def float64, shift int) float64`。

- [ ] **Step 1: 写迁移与消费失败测试**

`init_test.go`：`defaultFrameIntervalValue` 改 `"10"`；`TestSeedCoversNewlyRegisteredField` 的 `schema.Register` 改 `RegisterBlock`（块 "scan" 已存在 → 用独立块 `testseed_blk`）。新增 `TestMigrateFrameIntervalFixedPoint`（`useTempWorkspace`）：先 `storage.InitDB(".tagfile/tagfile.db")`+`AutoMigrate(&model.Config{})`+写入旧值，再 `bootstrap.InitDB()`，断言：
- 旧 `"1"` → `"10"`；旧 `"0.5"` → `"5"`（Review Focus 1）。
- 账本 `.tagfile/config_migrations` 含 `ai_frame_interval_fixedpoint_v1`；再次 `InitDB()` 值不变（幂等）。
- 旧 `"abc"` → 保留 `"abc"`、InitDB 不 panic（Review Focus 2）。
- 新库（无预写）→ seed 出 `"10"`。
ai 消费测试（`config_test.go` 或新增）：直接写库 `ai_frame_interval="5"` → `Settings().FrameInterval==0.5`；`"abc"`/`"0"` → 回落 `defaultFrameInterval`（1.0）。

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/bootstrap/ ./internal/service/ai/ -v` → FAIL。

- [ ] **Step 3: 实现迁移 + ai 消费 + bootstrap 切换**

`ai/config.go`：`FrameIntervalShift = 1`；`DefaultFrameInterval = "10"`；`defaultFrameInterval = float64(mustParseInt(DefaultFrameInterval)) / 10`；`shiftedIntSetting` 解析整数、`/math.Pow(10,float64(shift))`、≤0 或解析失败 logged fallback；`Settings().FrameInterval` 用它。`config_schema.go`：删旧 init，frame_interval Default 改 `DefaultFrameInterval`。
`bootstrap/config_schema.go`：`RegisterBlock` scan 块（scan_exclude String/DynamicArray/项约束{Format:regex}，Default 由 defaultExcludePatterns 序列化）。
`init.go`：`seedDefaultConfig` 遍历 `AllInputBoxes()`；新增 `migrateConfigStorage`：读账本（不存在=空），已含 ID 则跳过；读 `ai_frame_interval` 行，存在则 `ParseFloat` 旧值、`int64(math.Round(v*math.Pow(10,ai.FrameIntervalShift)))` 写回整数串，解析失败保留+日志；写账本（追加 ID）。`InitDB` 在 AutoMigrate 后、seed 前调用。bootstrap 显式 import ai（取 KeyFrameInterval/FrameIntervalShift）。

- [ ] **Step 4: 运行通过**

Run: `go test ./internal/bootstrap/ ./internal/service/ai/ ./internal/service/config/ -v` → PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/bootstrap/ internal/service/ai/config.go internal/service/ai/config_schema.go
git commit -m "feat: migrate ai_frame_interval to fixed-point and switch bootstrap to block registry"
```

---

### Task 6: 前端通用指令执行器 + 模板

**Files:**
- Modify: `cmd/webui/static/js/pages/config/main.js`（重写 configPage）
- Modify: `cmd/webui/templates/partials/config_field.html`（widget 分派）
- Modify: `cmd/webui/templates/partials/config_group.html`（g.input_boxes）
- Modify: `cmd/webui/templates/config.html`（x-for blocks）

**Interfaces:**
- Consumes: Task 4 契约（blocks/input_boxes/rules/positional_rules/container_rules，rule 带 kind/role/params/hint/message/widget）。
- Produces: 前端零文案副本的通用执行器；`f.widget` 单一分派。

- [ ] **Step 1: 重写 main.js**

`buildField`：映射 input box → field；`widget` = rules 中第一个非空 widget，否则结构默认（Int→number、Bool→checkbox、Array→rows-fixed、DynamicArray→rows、String→input）；`shift`/`options` 从 int_range/oneof 的 params 读；Int 显示值 = 存储整数/10^shift；容器解析 rows。`ruleCheckers` 按 spec §4.4（required/int/int_range/url/regex/pattern/oneof/bool/item_count/fixed_arity）。`fieldError`：遍历 rules 跑 checker，失败显示下发的 message（项级拼 `第 N 项`）；未知 kind 跳过；容器项数校验。`fieldHint`：rules hint `join('；')`。`saveGroup`：Int `String(Math.round(v*10^shift))`（先拒 NaN/Inf）；DynamicArray rows trim+过滤空行→JSON；Array 原样 N 行；secret（含 secret 指令）空值不提交。`resetField`/`resetGroup`/`refreshGroup`/`applySchema` 随 blocks/input_boxes 改名。删除全部硬编码中文校验文案与 `item_type` 字面量。

- [ ] **Step 2: 改模板**

`config.html`：`x-for="g in blocks"`。`config_group.html`：`x-for="f in g.input_boxes"`。`config_field.html`：按 `f.widget` 分派（input/textarea/password/number/checkbox/select/rows/rows-fixed）；password 占位 `:placeholder="f.has_value ? secretHint(f) : ''"`（secretHint 取 secret 指令 hint，零硬编码）；number 不设 HTML min/max（沿用理由注释）；rows-fixed 无增删按钮；行级标红用通用项校验。

- [ ] **Step 3: 构建 + 手工验证**

```bash
make build-webui
go run ./cmd/webui --source /tmp/opencode/tagfile-demo --addr :8088
```
浏览器验证：设置页加载、scan_exclude 行编辑/非法正则标红/提示文案来自后端、ai_frame_interval 显示 1.0（存储 10）、改 0.5 保存后库为 5、恢复默认、整组保存拒绝非法值。

- [ ] **Step 4: 提交**

```bash
git add cmd/webui/static/js/pages/config/main.js cmd/webui/templates/
git commit -m "feat: render config page as generic directive executor with served copy"
```

---

### Task 7: 删旧 API + 规范改名 + 版本 + CHANGELOG + 验收

**Files:**
- Modify: `internal/service/config/schema/schema.go`（删 FieldType/Field/Group/Type* 常量）
- Delete: `internal/service/config/schema/validate.go`、`registry.go`、`validate_test.go`、`registry_test.go`（旧）
- Modify: 所有 `ValidateInputBox`/`LookupInputBox`/`AllInputBoxes` 调用方 → `Validate`/`Lookup`/`All`（schema 包内 + ai + config service + bootstrap + api + 各测试）
- Modify: `internal/constant/version.go`（0.8.0）、`cmd/webui/constant/version.go`（5.1.0）
- Modify: `docs/CHANGELOG.md`、`docs/CHANGELOG_webui.md`

**Interfaces:**
- Consumes: 全部前序任务。
- Produces: 规范名 `Validate`/`Lookup`/`All`/`RegisterBlock`/`Blocks`/`BlockOf`（spec §7）；无旧 API 残留。

- [ ] **Step 1: 删旧 API + 改名**

删旧类型/常量/validate.go/registry.go 及旧测试。`ValidateInputBox`→`Validate`、`LookupInputBox`→`Lookup`、`AllInputBoxes`→`All`（定义与全部调用方）。`mustPanic`/`panicMessage` 辅助移入保留的 `registry_block_test.go`。

- [ ] **Step 2: 全量测试**

Run: `make test`
Expected: 全 PASS。

- [ ] **Step 3: 验收 grep**

```bash
grep -rn "TypeRegex\|ItemType\|item_type\|FieldType" internal cmd/webui   # 期望无命中
grep -rn "每行一条，正则表达式\|已设置，留空不修改" cmd/webui/static/js cmd/webui/templates   # 期望无命中
```

- [ ] **Step 4: 降级演练（Review Focus 5）**

临时删 `ruleCheckers.regex` → 页面仍加载、非法正则仍被服务端拒存回填 invalid_reason；恢复。确认 secret 指令前端不认识时值仍掩码（服务端投影保证）。

- [ ] **Step 5: bump 版本 + CHANGELOG**

`CoreVersion="0.8.0"`、`WebuiVersion="5.1.0"`。CHANGELOG.md 记：正交模型替换 FieldType、规则协议（校验/渲染/行为单源下发）、定点数迁移、块注册表。CHANGELOG_webui.md 记：blocks/input_boxes 契约、通用指令执行器、widget 效果推导、零文案副本。英中成对、高版本在上。

- [ ] **Step 6: 提交 + 清理 stash**

```bash
git add -A
git commit -m "refactor: remove legacy FieldType schema, bump core 0.8.0 / webui 5.1.0"
git stash drop   # 确认草稿已被演进覆盖、无遗漏后
```

---

## Self-Review 记录

- **Spec 覆盖**：§3 模型→T1；§4 协议→T2；§5 合法性+管线→T1/T2；§6 定点+迁移→T2/T5；§7 注册表→T1/T7；§8 secret→T3/T4；§9 契约+前端→T4/T6；§10 映射→T3/T5；§11 验收→T7；§12 非目标（无 Widget 维度/无组合逻辑/无文案覆盖）全程遵守。
- **类型一致**：`ValidateInputBox`/`LookupInputBox`/`AllInputBoxes` 过渡名在 T7 统一改回 `Validate`/`Lookup`/`All`；`RuleSet{Rules,PositionalRules,ContainerRules}` 在 T2 定义、T4 投影消费一致。
- **Review Focus**：1/2→T5，3→T3，4→T2+T6，5→T2+T7，均有对应测试步骤。