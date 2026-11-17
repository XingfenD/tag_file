# 配置 Schema 重构：InputBoxDescriptor 正交模型 + 规则协议

日期：2026-11-17
状态：待评审（v3：渲染/行为指令并入规则协议，三角色统一注册表/下发/降级）
分支（建议）：`feat/config-inputbox-descriptor`

## 1. 背景与目标

现有 `internal/service/config/schema` 用封闭枚举 `FieldType`（11 个类型）同时驱动三件事：服务端校验、JS 镜像校验、控件渲染。问题：

- 枚举把「存储形态 / 校验约束 / 控件 / 行为」揉在一个维度里，`url`/`regex` 本质是 string + 语法约束，`duration` 与 float 校验逐行相同。
- `Field` 已是半组合体：`Pattern` 只对 string 生效、`Options` 只对 enum、`Min/Max` 只对数值、`ItemType` 只对 list——枚举没有封装干净。
- **前端硬编码后端语义与文案**：`main.js` 写死「每行一条，正则表达式」、`item_type === 'regex'` 字面量、整段中文校验文案。后端加约束，前端必须同步改 JS，否则漏校验、错渲染。

目标（两条，缺一不可）：

1. **正交模型**：按用户给定框架拆成 ValueType × ContainerType × Constraint，约束按类型匹配、validator 强制，控件由维度推导。
2. **规则协议**：校验语义 + UI 文案（hint/message）+ **渲染/行为指令**全部收敛到后端一处注册表，作为单一事实源随 API 下发；前端是通用指令执行器，只认识原语 `kind`，零文案副本，未知 kind 优雅降级。

第 2 条是本次改造的核心目的，不是附属品：规则协议泛化到一切约束（不止列表项），渲染/行为指令一并收敛，在正交模型内实现。

## 2. 已确认的框架决策（用户拍板）

1. **不设 Float 值类型**：小数用 `Int + Shift`（定点数）。存储与传输一律整数 `n`，语义值 = `n / 10^Shift`。前端显示小数，保存乘回整数。
2. **加 Bool 值类型**。
3. **InputBoxType 拆为 ValueType × ContainerType**：ValueType ∈ {Int, String, Bool}；ContainerType ∈ {Single, Array（定长，逐位约束）, DynamicArray（单值约束作用于每个元素 + DynamicListConstraintDescriptor 容器级约束）}。
4. **约束匹配校验在 validator 里做**。
5. **元数据由实现方设计**——约束描述符承载校验/渲染/行为指令的语义与文案，支撑目标 2。

用户草稿在 stash（`stash@{0}`，schema.go 工作区修改）。演进约定：`InputBoxType` 按决策 3 替换；联合体字段名缩短（`IntConstraintDescriptor` → `IntConstraint`）；schema 类型不带 json tag，沿用 `api/config.go` 局部投影模式。

## 3. 数据模型（package schema，保持叶子包）

```go
type ValueType string
const (
    ValueInt    ValueType = "Int"
    ValueString ValueType = "String"
    ValueBool   ValueType = "Bool"
)

type ContainerType string
const (
    ContainerSingle       ContainerType = "Single"
    ContainerArray        ContainerType = "Array"        // 定长：长度 = 约束列表长度
    ContainerDynamicArray ContainerType = "DynamicArray" // 变长：单值约束作用于每个元素
)

type ConfigBlockDescriptor struct {
    ID    string
    Title string
    Order int
    InputBoxDescriptorList []*InputBoxDescriptor
}

type InputBoxDescriptor struct {
    Key         string
    Label       string
    Description string
    Default     string // 存储形态
    Unit        string // 展示用，不参与校验
    ShowIf      *Condition
    ValueType   ValueType
    Container   ContainerType
    Constraint  InputBoxConstraintDescriptor
    DynamicListConstraint *DynamicListConstraintDescriptor // 仅 DynamicArray 可设
}

type InputBoxConstraintDescriptor struct {
    // 按 ValueType × ContainerType 匹配，validator 强制：
    // Single/DynamicArray → 对应单例（nil = 无附加约束）；Array → 对应列表（逐位）。
    IntConstraint        *IntConstraintDescriptor
    StringConstraint     *StringConstraintDescriptor
    BoolConstraint       *BoolConstraintDescriptor
    IntConstraintList    []*IntConstraintDescriptor
    StringConstraintList []*StringConstraintDescriptor
    BoolConstraintList   []*BoolConstraintDescriptor
}

type IntConstraintDescriptor struct {
    Min, Max                 *float64 // 显示单位（= 存储整数 / 10^Shift）
    MinExclusive, MaxExclusive bool
    Shift                    int // 小数位偏移，≥0；0 = 纯整数
}

type StringFormat string
const (
    FormatNone  StringFormat = ""
    FormatURL   StringFormat = "url"
    FormatRegex StringFormat = "regex" // 仅 Array/DynamicArray 的项约束可用
)

type StringConstraintDescriptor struct {
    Required  bool         // 空串非法（仅 Single）
    Format    StringFormat
    Pattern   string       // 只约束非空值（沿用现状）
    Options   []Option     // 非空 → 枚举（仅 Single）
    Secret    bool         // 分解为 behavior 指令：掩码 + 留空不覆盖（仅 Single）
    Multiline bool         // 分解为 render 指令：textarea（仅 Single）
}

type BoolConstraintDescriptor struct{} // 校验内建于 ValueType，占位保持联合一致

type DynamicListConstraintDescriptor struct {
    MinItems *int
    MaxItems *int
}

type Option struct{ Value, Label string }   // 沿用
type Condition struct{ Key, Equals string } // 沿用
func Ptr[T any](v T) *T                     // 泛型化；浮点边界写浮点字面量 Ptr(0.0)，整型写 Ptr(1)
```

删除：`FieldType` 及全部 `Type*` 常量、旧 `Field`、旧 `Group`、`ItemType`。

**Secret/Multiline 在声明层是类型化标志，在协议层分解为指令原语**（§4）：与校验规则共用同一套注册表、下发格式、执行与降级机制，不再作为 payload 上的独立布尔字段。渲染/行为与校验是指令的三种**角色**，不是两套机制。

## 4. 规则协议：校验 / 渲染 / 行为指令（核心）

### 4.1 指令原语

协议词汇统一为**指令原语**（沿用 Rule 命名），按角色分三类：`validate`（校验）、`render`（渲染）、`behavior`（行为）。三者共用注册表、下发格式与降级机制。

```go
// RuleKind 是指令原语的封闭集合，独立于 ValueType/ContainerType 命名空间。
// 新增原语同样先讨论再加；它不伪装成字段类型。
type RuleKind string
const (
    // validate —— 校验语义
    RuleRequired   RuleKind = "required"    // 非空
    RuleInt        RuleKind = "int"         // 整数形态（ValueType 内建）
    RuleIntRange   RuleKind = "int_range"   // 显示单位边界 + Shift 换算
    RuleURL        RuleKind = "url"         // http/https + 主机名
    RuleRegex      RuleKind = "regex"       // 可编译为正则（项级）
    RulePattern    RuleKind = "pattern"     // 匹配给定模式（非空值）
    RuleOneOf      RuleKind = "oneof"       // 枚举成员（渲染效果：select）
    RuleBool       RuleKind = "bool"        // true/false（ValueType 内建）
    RuleItemCount  RuleKind = "item_count"  // 动态数组项数界限
    RuleFixedArity RuleKind = "fixed_arity" // 定长数组项数
    // render —— 渲染指令
    RuleMultiline  RuleKind = "multiline"   // 渲染效果：textarea
    // behavior —— 行为指令（服务端强制，前端仅驱动 UX）
    RuleSecret     RuleKind = "secret"      // 掩码 + 留空不覆盖；渲染效果：password
)

type RuleRole string
const (
    RoleValidate RuleRole = "validate"
    RoleRender   RuleRole = "render"
    RoleBehavior RuleRole = "behavior"
)

// Rule 是下发给前端的一条指令：语义由 Kind+Params 定义，文案与渲染效果随 payload 下发。
type Rule struct {
    Kind    RuleKind
    Role    RuleRole
    Params  map[string]string // 复杂值 JSON 编码（如 oneof 的 options）
    Hint    string            // 用法提示，来自 ruleDefs
    Message string            // 校验失败文案（已按 Params 渲染，不含键名/项序号）；非 validate 为空
    Widget  string            // 渲染效果（password/textarea/select…），来自 ruleDefs；无则空
}
```

### 4.2 指令注册表：语义 + 文案 + 渲染效果的唯一出处

```go
// ruleDef 是一条原语的定义：校验动作、UI 文案、渲染效果同处一地，全项目唯一。
type ruleDef struct {
    Role    RuleRole
    Hint    string                          // 用法提示（静态）
    Message string                          // 失败文案模板，按 Params 渲染（仅 validate）
    Widget  string                          // 渲染效果（""=无）
    Check   func(p map[string]string, v string) error // 仅 validate 必有；render/behavior 为 nil
}

var ruleDefs = map[RuleKind]ruleDef{
    RuleRegex: {
        Role:    RoleValidate,
        Hint:    "每行一条，正则表达式",
        Message: "不是合法的正则表达式",
        Check:   checkRegex,
    },
    RuleIntRange: {
        Role:    RoleValidate,
        Message: "取值超出合法范围，应为 {{range}}", // 渲染为 [1, 200] / (0, +∞) 等
        Check:   checkIntRange,
    },
    RuleOneOf: {
        Role:    RoleValidate,
        Message: "取值必须是以下之一: {{candidates}}",
        Widget:  "select", // 枚举即下拉——映射住在后端，不在 JS
        Check:   checkOneOf,
    },
    RuleMultiline: {Role: RoleRender, Widget: "textarea"},
    RuleSecret: {
        Role:   RoleBehavior,
        Hint:   "已设置，留空不修改", // 现状写死在 config_field.html 的占位文案，收编进注册表
        Widget: "password",
        // Check nil：secret 不产生校验语义；掩码与留空不覆盖由服务端强制（§8）
    },
    // required / int / url / pattern / bool / item_count / fixed_arity 同构（RoleValidate）
}
```

要点：

- **校验就是跑 validate 指令**。`Validate` 不再有一组 per-type 校验函数；描述符经 `RulesOf` 分解为指令，校验器逐条执行 `ruleDefs[kind].Check`（render/behavior 无 Check，引擎跳过）。语义与文案不可能分居两处。
- **文案单源**：hint 静态取自 ruleDefs；message 模板按 Params 渲染（区间、候选值、模式串）。服务端校验错误与 payload 下发的 message 用同一模板渲染，逐字一致。secret 占位文案 likewise 只此一份。
- **控件映射单源**：oneof→select、secret→password、multiline→textarea 住在 ruleDefs.Widget，JS 不再维护「哪种约束用哪种控件」的后端语义表。
- `Check` 可返回细分文案（如 url 的「必须以 http:// 开头」「必须包含主机名」）——细分文案也住在 ruleDefs 条目内，仍是单源。前端本地预校验失败时显示下发的通用 message，服务端 `invalid_reason` 给细分文案。
- 项序号由校验器拼接：`第 %d 项%s`（下标从 1），与现状逐字对齐；键名由调用方包装（沿用）。

### 4.3 约束 → 指令分解（RulesOf）

类型化约束描述符是**声明层 ergonomics**；指令是**引擎与协议**。分解关系：

| 声明 | 分解出的指令 | 角色 | Params / Widget |
|---|---|---|---|
| ValueType=Int（内建） | `int` | validate | — |
| IntConstraint{Min,Max,开闭,Shift} | `int_range` | validate | min, max, min_exclusive, max_exclusive, shift |
| ValueType=Bool（内建） | `bool` | validate | — |
| StringConstraint.Required | `required` | validate | — |
| StringConstraint.Format=url | `url` | validate | — |
| StringConstraint.Format=regex | `regex`（仅项级） | validate | — |
| StringConstraint.Pattern | `pattern` | validate | pattern |
| StringConstraint.Options | `oneof` | validate | options（JSON [{value,label}]）/ select |
| StringConstraint.Multiline | `multiline` | render | textarea |
| StringConstraint.Secret | `secret` | behavior | password |
| DynamicListConstraint | `item_count` | validate | min_items, max_items |
| Container=Array（内建） | `fixed_arity` | validate | n |

- Single：指令作用于值。
- DynamicArray：同一组指令作用于**每一项**；`item_count` 作用于容器。render/behavior 仅 Single 可声明（§5.1），不下发到项级。
- Array：`fixed_arity` 作用于容器；**逐位**各有一组指令（来自对应位置的约束描述符）。
- 空约束 = 只有 ValueType 内建指令（Int→int，Bool→bool，String→无）。
- 指令按稳定顺序输出，带 Widget 的指令按注册表优先级排序（secret > oneof > multiline；互斥组合已由 §5.1 矩阵排除，优先级是防御性兜底）。

### 4.4 前端：通用指令执行器

```js
// 本地可执行校验注册表：只认识 kind 原语，零后端文案副本。
var ruleCheckers = {
  required:  function (v, p) { return v !== ''; },
  int:       function (v, p) { return /^-?\d+$/.test(v); },
  int_range: function (v, p) { /* 解析整数、按 p.shift 换算、按 p.min/max 判界 */ },
  url:       function (v, p) { /* http/https + 主机名 */ },
  regex:     function (v, p) { try { new RegExp(v); return true; } catch (e) { return false; } },
  pattern:   function (v, p) { /* 非空值匹配 p.pattern */ },
  oneof:     function (v, p) { /* 命中 p.options */ },
  bool:      function (v, p) { return v === 'true' || v === 'false'; },
  item_count:  function (rows, p) { /* 项数界限 */ },
  fixed_arity: function (rows, p) { return rows.length === Number(p.n); }
};
```

前端对一条指令的处理按角色分派，**未知 kind 一律优雅降级**：

- **validate**：`ruleCheckers[kind]` 存在则执行，失败显示**下发的** `rule.message`（项级拼 `第 N 项`）；不存在则本地放行，信任服务端。
- **render / behavior**：认识则应用其 UX 效果（widget、secret 保存时留空不提交、占位文案用下发的 hint）；不认识则忽略——**行为强制在服务端独立成立（§8），不依赖前端认识 kind**，故降级安全：secret 值从未下发（服务端掩码），未知 secret 指令的前端最多渲染普通输入框，留空保存仍被服务端视为「保持不变」。
- **控件推导**：取指令中第一个非空 `widget`（后端已按优先级排序）；无则按结构默认——Int→number、Bool→checkbox、Array/DynamicArray→行编辑器、String→input。这些是 HTML 原语级映射，属前端结构知识，不含后端语义。
- 提示：`fieldHint` = 下发 hint 的拼装（`join('；')`）；secret 占位文案 = 该指令 hint（has_value 时显示）。**「每行一条，正则表达式」「已设置，留空不修改」及所有校验文案在 JS/模板中零副本**。
- shift/options 等参数**从指令 Params 读取**，不另投影一份——无第二份拷贝。
- 加新字段（复用已有 kind）：零 JS 改动。加新 kind：前端执行器/渲染器可选实现，不实现则降级，校验写入仍被服务端正确拒绝并回填原因，行为仍被服务端强制。

### 4.5 正则方言差异

JS `new RegExp`（ECMA）与 Go `regexp`（RE2）方言不一致，客户端预校验是 best-effort，服务端 `regexp.Compile` 才是权威。现状已如此，本次不新增风险，也不在客户端复刻 RE2。

## 5. 描述符合法性 + 值校验管线

### 5.1 描述符合法性（Validate 内，与取值无关，先于值校验）

任何一条不满足即返回错误；`RegisterBlock` 经默认值自检升级为启动 panic（fail-fast 沿用）。

1. ValueType/ContainerType 属于封闭集合，未知报「未知的值类型/容器类型」。
2. 约束匹配：
   - Single/DynamicArray：仅对应单例可非 nil（nil=无附加约束）；其余单例 nil；三个列表为空。
   - Array：仅对应列表非空（len≥1，定长=len）；其余列表空；三个单例 nil。
   - 错配报「约束与值类型/容器类型不匹配」。
3. DynamicListConstraint 仅 DynamicArray；MinItems/MaxItems≥0，同设时 Min≤Max。
4. IntConstraint：Shift≥0；Min/Max 同设时 Min≤Max。
5. StringConstraint：Format ∈ {"",url,regex}；regex 仅 Container≠Single；Required/Options/Secret/Multiline 仅 Single；Options 非空时 Secret/Multiline 必须 false，Value 互不重复；Pattern 在描述符检查期即编译（非法启动即失败）。
6. 规则分解后每个 kind 必须在 ruleDefs 中（类型化分解下由构造保证；仍做防御性检查）。

### 5.2 值校验管线（Validate(D, value)）

1. 描述符合法性（§5.1）。
2. 容器解析：Single 原样；Array/DynamicArray 必须是 **JSON 字符串数组**（元素是字符串，按 ValueType 校验——与现状 scan_exclude 存储/行编辑器传输一致，Int/Bool 元素也不例外）。
3. 跑 validate 指令（唯一校验引擎；render/behavior 无 Check，引擎跳过）：
   - 容器级：Array→fixed_arity（「需要恰好 N 项」）；DynamicArray→item_count。
   - 值级/项级：Single 跑一组；Array 逐位跑对应组；DynamicArray 每项跑同一组。逐条全跑（AND，无短路组合逻辑），失败文案 `第 %d 项%s` / 值级直接 message。
4. 文案约定：中文、含合法范围、不含键名。旧 duration「必须是秒数」措辞取消（传输是整数，单位由 Unit 展示）。

## 6. 定点数 Shift 与存量迁移

### 声明与消费

- 偏移常量住消费方：`ai.FrameIntervalShift = 1`（十分之一秒）。
- `ai.DefaultFrameInterval` `"1"` → `"10"`（存储形态 = 1.0 秒 × 10）。
- 声明：Int / Single / IntConstraint{Min: Ptr(0.0), MinExclusive: true, Shift: ai.FrameIntervalShift}，Unit「秒」。
- `ai.Settings()`：`floatSetting` → `shiftedIntSetting(key, def, shift)`——解析存储整数、除以 10^shift、≤0 或解析失败走既有 logged fallback。`defaultFrameInterval` 由 mustParseInt(Default) 除以 10^shift 派生（=1.0）。

### 前端

- 加载：显示值 = 存储整数 / 10^Shift（shift 从 int_range 规则 Params 读）。
- 编辑：number 框显示小数（step=10^-Shift，仍不设 HTML min/max，沿用「步进网格误杀存量值」理由）。
- 保存：先拒 NaN/Inf，再 `String(Math.round(v * 10^Shift))`——后端永远收整数串。
- 边界判定在显示单位上，与服务端一致；失败文案用下发的 message。

### 存量迁移（一次性）

旧库 `ai_frame_interval` 存秒数字符串（"1"=1秒），新制 "1"=0.1秒——**必须迁移，否则语义静默漂移 10 倍**。

- 位置：`bootstrap.InitDB`，AutoMigrate 之后、seedDefaultConfig 之前，新增 `migrateConfigStorage(db, dbPath)`。
- 幂等守卫：账本文件 `.tagfile/config_migrations`（每行一个已应用迁移 ID），与 core_version 同目录同模式。ID：`ai_frame_interval_fixedpoint_v1`。
- 动作：读行；存在则按旧形态 ParseFloat，`n = round(v × 10^Shift)` 写回整数串。新库无行→空操作，seed 补新 Default "10"。
- 旧值解析失败：保留原值 + 日志；设置页经 Validate 标 invalid 给原因，由运维修正。不静默改写无法解析的值。
- 安全网：bump CoreVersion 触发 `backupDBIfNeeded` 既有整库备份（`.bak.<时间戳>`），迁移前自动生效。
- 不改 configs 表结构，纯数据迁移。

## 7. 注册表 API

块拥有描述符列表，注册以块为单位：

```go
func RegisterBlock(b ConfigBlockDescriptor)      // 重复块 ID / 重复键 / 描述符非法 / 默认值非法 → panic
func Lookup(key string) (*InputBoxDescriptor, bool)
func Blocks() []ConfigBlockDescriptor            // Order 升序，同 Order 按 ID
func BlockOf(id string) *ConfigBlockDescriptor   // 未知 → nil
func All() []*InputBoxDescriptor                 // 跨块按 Key 升序，供 seed
```

内部：块 map + 键索引 map，互斥锁沿用。删除 RegisterGroup/Register/FieldsOf/Groups。

## 8. Secret 行为

`StringConstraint.Secret` 分解为 `secret` behavior 指令，经辅助方法读取：

```go
func (d *InputBoxDescriptor) IsSecret() bool // Single + String + 含 secret 指令
```

三处行为沿用，仅改判断来源：service.SetGroup 缺失=保持不变（现 service.go:66）；api 列表/schema 投影值与默认值掩码只回 has_value；JS saveGroup 空值 secret 不提交。secret 恒不判 invalid（沿用）。

**行为强制在服务端独立成立，不依赖前端认识 kind**：掩码发生在投影时（值从未下发），留空不覆盖发生在 SetGroup（按描述符判定）。前端的 secret 指令只驱动 UX（password 框、占位文案、空值不提交）；未知该指令的前端降级为普通输入框，仍不可能泄露或抹掉已存密钥。

## 9. API 契约与前端

### GET /api/config/schema

外层 `{blocks: [{id, title, order, input_boxes: [...]}]}`。投影结构局部住 api/config.go。单个 input box：

```json
{
  "key": "...", "label": "...", "description": "...", "default": "...",
  "unit": "...", "show_if": {"key": "...", "equals": "..."} | null,
  "value_type": "Int|String|Bool",
  "container": "Single|Array|DynamicArray",
  "rules": [ {"kind": "url", "role": "validate", "params": {}, "hint": "...", "message": "...", "widget": ""} ],
  "positional_rules": [ [ {"kind": "...", ...} ], [ ... ] ],
  "container_rules": [ {"kind": "item_count", "role": "validate", "params": {...}, "hint": "", "message": "...", "widget": ""} ],
  "value": "...", "has_value": true, "invalid": false, "invalid_reason": ""
}
```

- `rules`：Single 作用于值；DynamicArray 作用于每项（统一，仅 validate）。
- `positional_rules`：仅 Array，与定长同长，逐位指令组。
- `container_rules`：item_count / fixed_arity。
- render/behavior 指令（multiline/secret）只出现在 Single 的 `rules` 中；**不再有独立的 secret/multiline 布尔字段**。
- hint/message/widget 由后端 ruleDefs 渲染填充。
- value/has_value/invalid/invalid_reason 语义沿用。POST /api/config/group 与高级区 CRUD 不变。

### 控件推导（指令渲染效果优先，结构默认兜底）

1. 取 `rules` 中第一个非空 `widget`（后端按优先级排序：secret→password、oneof→select、multiline→textarea）。
2. 无渲染效果时按结构默认：value_type=Int→number（shift 从 int_range Params 读）、Bool→checkbox、container=Array→rows-fixed（定长无增删）、DynamicArray→rows（可增删）、String→input。

结构默认是 HTML 原语级映射（前端结构知识）；「哪种约束用哪种控件」的后端语义映射住在 ruleDefs.Widget，JS 零副本。行编辑器每行控件按去掉容器后的同一规则推导（Int 项→number 含 Shift 换算，Bool 项→checkbox，String 项→input）。今天只有 String 动态数组在线上，其余组合测试覆盖。

模板 config_field.html 从类型交叉判断改为单一 f.widget 分派；secret 占位文案用该指令下发的 hint（has_value 时显示），不再写死「已设置，留空不修改」；config_group.html/config.html 随 groups→blocks、fields→input_boxes 改名。

### 保存

DynamicArray 行 trim+过滤空行→JSON 字符串数组（沿用）；Array 原样发恰好 N 行（不过滤，空串合法）；Int 走 Shift 换算；secret 空值不提交。

## 10. 现有键映射

| 键 | 新声明 | 下发指令 |
|---|---|---|
| ollama_model | String / Single / 无约束 | 无（空串合法=未配置） |
| ollama_url | String / Single / {Required, Format:url} | required, url |
| ai_frame_interval | Int / Single / {Min:0 开, Shift:1}，Unit 秒，Default "10" | int, int_range |
| ai_frame_diff_threshold | Int / Single / {Min:1, Max:64} | int, int_range |
| ai_max_frames | Int / Single / {Min:1, Max:200} | int, int_range |
| scan_exclude | String / DynamicArray / 项约束 {Format:regex} | 项级 regex |

旧闲置词汇落点（机制保留，无线上键）：text=String+Multiline；secret=String+Secret；enum=String+Options；duration=Int+Shift+Unit。

## 11. 影响面、测试与验收

改动文件：

- internal/service/config/schema/：schema.go / validate.go / registry.go 重写，新增 rules.go（RuleKind/ruleDefs/RulesOf/规则引擎）+ 测试。
- internal/service/config/service.go（+test）：FieldsOf→BlockOf；secret 判断改 IsSecret()。
- internal/service/ai/config_schema.go（+test）：块注册、新声明。
- internal/service/ai/config.go：FrameIntervalShift、DefaultFrameInterval、shiftedIntSetting。
- internal/bootstrap/config_schema.go / init.go（+test）：块注册、migrateConfigStorage、seed 走 All()。
- cmd/webui/api/config.go（+config_schema_test.go）：投影（规则+文案渲染）、掩码、BlockOf/Blocks。
- cmd/webui/static/js/pages/config/main.js：ruleCheckers 通用指令执行器、widget 效果应用、保存换算；删除全部硬编码文案（含 secret 占位）与 item_type 字面量。
- cmd/webui/templates/partials/config_field.html / config_group.html、config.html。
- docs/CHANGELOG.md + docs/CHANGELOG_webui.md（英文行+中文行成对，条目间空行）。
- 版本：Core 0.7.1→0.8.0；WebUI 次版本+1（基线随最终分支定：teal-glass 4.5.0→4.6.0 / 扁平主题 5.0.0→5.1.0），与 CHANGELOG 同 commit。

测试（TDD，先红后绿）：

- schema：描述符合法性矩阵；RulesOf 分解快照（含 render/behavior 指令与 widget 效果）；各 validate 原语 Check；Shift 边界换算；Array 定长/逐位、DynamicArray 项数/逐项；文案渲染（区间/候选/模式）；secret 辅助。
- registry：块注册、重复 panic、默认值自检 panic、排序。
- bootstrap：迁移（"1"→"10"、"0.5"→"5"、账本幂等、新库空操作、损坏值保留）。
- ai：声明快照；shiftedIntSetting fallback。
- config service / api：secret 缺失=保持、掩码、规则投影（hint/message 来自 ruleDefs）、整组保存。
- 收尾 make test；WebUI 手工验证设置页。

验收不变量（全部满足才算成功）：

1. `grep -rn "TypeRegex\|ItemType\|item_type\|FieldType" internal cmd/webui` 无命中。
2. 后端语义文案（「每行一条，正则表达式」、「已设置，留空不修改」、各校验失败文案）在 cmd/webui/static/js 与 templates 中零副本，只存在于后端 ruleDefs；控件语义映射（oneof→select 等） likewise 只在 ruleDefs.Widget。
3. scan_exclude 下发 `rules:[{kind:"regex",role:"validate",hint:"每行一条，正则表达式",message:"不是合法的正则表达式"}]`；非法项错误文案 `第 1 项不是合法的正则表达式` 与改造前逐字一致。
4. 页面加载、逐行编辑、非法标红、整组保存严格模式——行为与改造前对用户不可分辨（Shift 显示换算除外，那是决策 1 的预期变化）。
5. **降级演练**：删掉 ruleCheckers.regex（模拟未知 validate kind）后，页面仍加载、非法正则仍被服务端兜底拒存并回填 invalid_reason；前端不认识 secret 指令时，值仍被服务端掩码、留空保存仍「保持不变」。
6. 默认值/边界/文案/控件映射无第二份拷贝（shift/options 从指令 Params 读，widget 从指令效果读，不另投影）。

## 12. 非目标

- 不加显式 Widget 维度（控件可推导，YAGNI）。
- 不新增线上配置键；不动高级 KV 区与 configs 表结构。
- 不做规则组合逻辑（AND/OR 短路、按项条件启用）；本期规则逐条全跑。
- 不支持字段级文案覆盖（保持单源）；确有特例再加 Rule 上的 override。
- CLI 不受影响（cmd/cli 不引用 schema 包）。
- Array（定长）今天无线上键，按用户要求实现机制并测试覆盖。