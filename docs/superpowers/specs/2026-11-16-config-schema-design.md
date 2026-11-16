# 设置页 schema 化改造 — 设计文档

日期：2026-11-16
状态：已确认核心决策；建造者审读完成，遗留问题已由需求方拍板（存量非法值取严格模式）

## 1. 意图与背景

WebUI 设置页（`cmd/webui/templates/config.html`）目前以自由 KV 表单为主：用户手填键名和值，对键的含义、类型、合法范围零提示零校验。配错的根源是**页面不知道每个键是什么**：

- 键名拼错一个字母就静默新建一条无效配置，真正的配置没变，用户以为改了。
- `scan_exclude` 的值是 JSON 正则数组，手写极易配坏；配坏后扫描排除静默失效。
- 数值键越界或不可解析时，读取端静默回落默认值——「存进去了但没生效」是最隐蔽的配错。
- 5 个 AI 键已有结构化卡片，但同时出现在 KV 列表里，双重编辑入口容易改一处忘一处。
- 删除已知键无任何警告，功能悄悄退化。
- 配置列表的「描述」列写死显示 `—`（模板第 44 行），数据库里实际存有描述。

代码中真正被消费的键只有 6 个：`scan_exclude` + 5 个 AI 键（`ollama_url`、`ollama_model`、`ai_frame_interval`、`ai_frame_diff_threshold`、`ai_max_frames`）。键少而含义明确，完全可以在后端把每个键的元数据（类型、范围、默认值、说明）声明一次，前端照此渲染和校验。

同时，代码里已有一批写死但将来很可能开放的参数（`ollamaTimeout=180s`、`extractTimeout=120s`、`maxTaskSuggestions=50`、扫描排除正则等），本方案的类型系统必须能声明式地容纳它们，做到**加一个配置 = Go 里加一条注册 + 消费方读取代码，前端零改动**。

## 2. 目标

- 已知键全部通过按类型渲染的结构化表单编辑，写入时校验，不合法直接报错拒存。
- 存量非法值（老库里「今天能用、新校验下非法」的值）在页面加载时即被标记标红，严格模式下必须先修好才能保存该组。
- `scan_exclude` 用正则列表编辑器（一行一条，可增删行，逐行校验正则合法性），不再手写 JSON。
- 消灭已知键的双重编辑入口：KV 自由编辑只保留给 schema 之外的未知键。
- 已知键不可删除，提供「恢复默认」。
- schema 成为单一事实源：前端渲染、写入校验、bootstrap 种子三处共用。
- 类型系统封闭且可扩展，覆盖第 8 节候选清单中的全部配置种类。
- 修复描述列显示 bug。

## 3. 非目标

- 不改 KV 存储结构（`Config` 模型、表结构不动），不加迁移。
- 不实现第 8 节候选清单中的任何新配置键——本期只填已有 6 键。
- 不做配置版本历史/回滚、配置档案/导入导出、权限体系。
- 不做热更新订阅：读取方本来就是用时即读。
- 进程级参数（`--source`、`--addr`）不进 DB 配置。
- 不放宽读取端的回落逻辑——它是最后防线，写入校验是新增的第一道。
- 「测试连接」按钮与模型下拉选择列为第二期，本期不做（见第 9 节）。

## 4. 反目标（做成了也算失败）

- 用户保存了不合法的值却收到成功反馈（静默回落继续发生在写入路径上）。
- 加一个新配置键需要同时改前端模板/JS 才能出现在页面上。
- 默认值出现第三份拷贝（现有两份：消费方常量 + bootstrap 手写清单，本方案要收敛为一份）。
- 已知键仍能从某个入口被删除或绕过校验写入。
- AI 卡片现有的 aiLoaded 防覆盖守卫（读取未完成时禁用保存，防止默认值覆盖库中真实配置）在改造中丢失。

## 5. 已确认的决策（需求方拍板）

| 决策点 | 结论 |
|---|---|
| 技术取向 | 沿用 Gin + Alpine + KV 表，不动存储，加 schema 层 + 重做页面 |
| 页面结构 | 分组结构化表单取代自由 KV 区；未知键收进折叠的「高级 / 自定义配置」区 |
| 写入校验 | 已知键按类型校验，不合法报错拒存；已知键禁删，改为恢复默认 |
| 拓展能力 | 封闭类型系统 + 分组注册表 + 通用渲染器，候选清单仅预留不实现 |
| `Default` 类型 | string（存储形态），配注册时自检兜底类型安全 |
| AI 双重入口 | 已知 6 键不再出现在自由 KV 区 |
| 存量非法值 | 严格模式：组内存在任何非法值（含存量）即不允许保存，必须先修好；schema 接口标记非法存量值，页面加载即标红 |

## 6. 设计细节

### 6.1 类型系统（封闭集合）

校验器按类型编写，一个类型一个，键可以有无限个。遇到现有类型表达不了的需求，先讨论加类型，不允许给单键开特例渲染。

| 类型 | 控件 | 约束参数 | 存储形态 | 用途示例 |
|---|---|---|---|---|
| `string` | 单行输入 | Pattern、MaxLength | 原样 | `default_editor` |
| `text` | 多行文本 | MaxLength | 原样 | AI 追加提示词 |
| `url` | 单行输入 | scheme 白名单（http/https） | 原样 | `ollama_url` |
| `int` | 数字输入 | Min、Max、Step、Unit | 十进制整数字符串 | 抽帧阈值/上限、页大小 |
| `float` | 数字输入 | Min、Max、Step、Unit | 十进制小数字符串 | `ai_frame_interval` |
| `duration` | 数字+单位后缀 | Min、Max（秒） | 秒数字符串 | 识别/抽帧超时 |
| `bool` | 开关 | — | `"true"` / `"false"` | 跟随符号链接 |
| `enum` | 下拉 | Options（值+显示名） | 选项值原样 | 主题、ai_provider |
| `list` | 行编辑器 | ItemType（string/regex） | JSON 字符串数组 | `scan_exclude` = list\<regex\> |
| `secret` | 掩码输入 | — | 原样 | 云端 API key |

`secret` 的数据流规则现在就定进协议：任何读取接口永不回传值，只回 `has_value: true`；前端显示「已设置」+ 覆盖输入框，留空提交表示不修改。本期没有 secret 键，但渲染器与接口按此协议实现。

`list` 的 ItemType 为 `regex` 时逐条 `regexp.Compile` 校验；序列化固定为 JSON 数组，与现有 `scan_exclude` 存储格式一致。

数值边界支持排他标记（`MinExclusive`/`MaxExclusive`）。原则：**写入校验范围 = 读取端接受范围，精确镜像，不严不宽**。例如 `ai_frame_interval` 读取端接受任意 >0 的数（`floatSetting` 只拒 ≤0），声明为 Min=0 排他；现有前端表单手写的 min=0.1 比后端严，废弃。不允许出现「前端一套范围、后端一套范围」的新配错源。

### 6.2 schema 条目与分组注册表

```go
type Field struct {
    Key, Group, Label, Description string
    Type     FieldType
    Default  string        // 存储形态，见 6.3
    Min, Max *float64      // int/float/duration
    MinExclusive, MaxExclusive bool // 排他边界，镜像读取端（如 interval >0）
    Pattern  string        // string / list 项
    Options  []Option      // enum
    ItemType FieldType      // list
    Unit     string         // 显示后缀：秒、MB、帧
    ShowIf   *Condition     // 可选：{Key, Equals}，条件显隐
}

type Group struct {
    ID, Title string
    Order     int
}
```

- 分组注册表与字段注册表同处一个**叶子包**（建议 `internal/service/config/schema`，不 import 任何 service 包）。
- **注册模式（关键约束）**：`internal/service/ai` 已 import config 服务（`internal/service/ai/service.go:8`），config 侧不能反向 import ai，否则循环依赖。因此字段由各消费方在自己的 `init()` 里调用 `schema.Register(Field{...})` 注册，引用本包常量：
  - ai 包注册 5 个 AI 键，引用 `KeyOllamaURL`、`DefaultOllamaURL`、`maxFramesLimit` 等现有常量；
  - `scan_exclude` 的默认值现居 `internal/bootstrap/init.go`（`defaultExcludePatterns`），由 bootstrap 注册。
  - 两个入口（CLI、WebUI）都 import bootstrap，bootstrap import ai，注册链可达。新增消费方注册时必须保证同样的可达性。
- 默认值与边界常量**仍住在消费方**（如 `maxFramesLimit` 连同其 OOM 注释留在 ai 包），schema 引用不复制——边界的依据是消费方的实现约束，注释和代码不分家。
- `ShowIf` 本期只定义协议字段并随 schema 下发，渲染器的条件显隐实现推迟到第一个使用者出现（预期是 ai_provider）。

### 6.3 Default 为 string + 注册时自检

KV 表 `ConfigValue` 是 string，bootstrap 种子与消费方默认常量均为 string。`Default` 取存储形态使种子零转换、「恢复默认」就是一次 Set、与消费方常量同形。代价是丢编译期类型安全，补救：`schema.Register` 时用该字段自己的类型校验器校验 Default，不合法直接 panic（与现有 `mustParseInt` 的 fail-fast 风格一致），坏默认值活不过进程启动。

### 6.4 bootstrap 种子收敛

`seedDefaultConfig`（`internal/bootstrap/init.go:30`）的手写清单改为遍历注册表：对每个注册字段，缺键才插入（保持现有逐键检查语义，后加的键仍能到达老数据库），值取 `Field.Default`，描述取 `Field.Description`。已存在的行不覆盖。

### 6.5 写入校验（service 层）

- `Set` / `Update`：键在注册表中 → 按类型校验，不合法返回带键名和原因的错误，拒存；键不在注册表中 → 仅要求键名非空，原样通过（未知键是通用 KV 的合法用途，CLI 也会写）。
- `Delete`：已知键拒绝，错误信息提示用「恢复默认」；未知键照旧。
- 校验在 service 层实现，CLI 与 WebUI 同时受益。
- 读取端（`ai.Settings()` 等）的回落逻辑不动。

### 6.6 API

- 新增 `GET /api/config/schema`：返回分组 + 字段元数据 + **当前值**（secret 只回 `has_value`）+ **存量值合法性标记**（按该字段类型校验器判定，非法时附原因），一次请求拿到整页快照。具体 JSON 形状留给建造者，但必须满足：前端可据此通用渲染、能区分「读取失败」与「值为空」、能标红非法存量值。
- 新增按组批量保存端点，**严格模式**：前端提交该组全部字段，后端先校验全部字段（含未改动的存量值），任一不合法则整组拒存并返回逐键错误；全部合法才写入（事务保证原子性）。存量非法值会阻塞该组保存，直到用户修好——这是需求方拍板的行为，页面须通过加载时的标红让用户一眼看到要修什么。
- 现有单键 CRUD 端点保留（高级区与 CLI 使用），行为按 6.5 收紧。
- 批量保存请求沿用 `application/x-www-form-urlencoded`；重复键用真实 `URLSearchParams` 传递。
- 所有响应走 `respond(c, msg, msgType, extra...)` 约定；DELETE 类端点参数一律 `c.Query`/`c.Param`。

### 6.7 前端页面

- 页面由 schema 驱动通用渲染：组 → 卡片（按 Order 排序），字段 → 按类型选控件。字段旁展示 Label、Description、默认值、单位与范围提示；每字段提供「恢复默认」；每组独立保存按钮。非法存量值加载即标红并附原因，修好前该组保存按钮给出明确阻塞提示。
- **加载守卫**：整页沿用 AI 卡片现有的 aiLoaded 模式——schema+值读取完成前所有输入禁用、保存禁用，显示可重试的警告条，防止默认值覆盖库中真实配置。
- `scan_exclude` 渲染为正则行编辑器：一行一条，增删行，逐行即时校验并标红非法正则。
- **高级 / 自定义配置**：折叠区，仅展示注册表之外的键，保留自由增删改与搜索/分页，附「未知键不会被程序读取」提示；描述列显示真实 Description（修复写死 `—` 的 bug）。现有表单的描述输入框移除——它端到端被静默丢弃（handler 不读、repo upsert 不写），本期不打通写入，已有描述只读展示；已知键的说明来自 schema。
- AI 卡片模板中手写的键说明/范围文案删除，改由 schema 提供。
- 数据导出卡片不动。
- 模板遵守现有约定：`{{define}}` 不嵌套，块各自顶层；页面三处注册（main.go 模板解析、路由、导航）已有，不新增页面。

## 7. 约束（来自 AGENTS.md 与代码现状）

- Gin 路由，输入读 `c.Param`/`c.Query`/`c.PostForm`/`c.PostFormArray`；`API.del` 无 body，DELETE 处理器禁用 `c.PostForm`。
- 前端以 `data.ok` 判断成败，不依赖 try/catch。
- SQLite + GORM，schema 变更仅 AutoMigrate——本方案无表结构变更。
- 核心改动记 `docs/CHANGELOG.md` 并同步 `internal/constant.CoreVersion`；WebUI 改动记 `docs/CHANGELOG_webui.md` 并同步 `cmd/webui/constant.WebuiVersion`；中英双行条目格式。
- 测试走 `make test`（`go test ./...`），不造临时测试二进制。
- import 环约束见 6.2，注册链可达性是本方案最容易被忽视的坑。

## 8. 候选配置清单（预留示例，本期不实现）

类型系统的验收标准：以下全部能用现有类型声明，无需特例。

| 组 | 键（示意） | 类型 | 现状依据 |
|---|---|---|---|
| 扫描 | scan_exclude | list\<regex\> | 已有 |
| 扫描 | 跟随符号链接 | bool | 扫描器行为 |
| 扫描 | 单文件大小上限 | int（MB） | 扫描器行为 |
| 扫描 | 扫描并发 | int | 扫描器行为 |
| AI | 已有 5 键 | url/string/float/int/int | 已有 |
| AI | 追加提示词 | text | BuildPrompt 扩展点 |
| AI | 识别超时 | duration | `ollamaTimeout=180s`（ollama.go:19） |
| AI | 抽帧超时 | duration | `extractTimeout=120s`（frames.go:20） |
| AI | 单任务建议上限 | int | `maxTaskSuggestions=50`（recognize.go:35） |
| AI | 免审核直接应用 | bool | 产品决策，未定 |
| AI | ai_provider / base_url / api_key | enum+url+secret+ShowIf | 云端视觉服务接入时 |
| 界面 | 每页条数 | int | 分页写死 |
| 界面 | 主题 | enum | — |
| 界面 | default_editor | string | KV 表单占位符已暗示 |
| 维护 | 备份保留份数 | int | `backupDBIfNeeded` |

## 9. 第二期（本期不做，另行立项）

- AI 组「测试连接」按钮：后端代理请求 `ollama_url/api/tags`，连通性与模型名存在性一次看清。
- 连接成功后 `ollama_model` 改下拉选择（列已装模型），保留手动输入兜底。

## 10. 默认选择（设计者决定，需求方未反对，可否决）

- schema 叶子包位置：`internal/service/config/schema`；注册走消费方 `init()`。
- `GET /api/config/schema` 一次返回元数据+当前值（而非分离两个请求），保证整页快照原子性、简化加载守卫。
- 按组批量保存「前端提交整组、后端先全量校验、事务写入」，不做逐字段独立提交的半组生效。
- 数值边界加排他标记，写入校验精确镜像读取端接受范围（interval 为 Min=0 排他，废弃前端手写的 min=0.1）。
- 描述输入框本期移除（现状端到端静默丢弃），已有描述只读展示；打通描述写入留作后续需求。
- 未知键写入不校验内容（仅键名非空），保持通用 KV 的开放性。
- 已知键的「恢复默认」实现为 Set 回 `Field.Default`，不引入新语义。
- 本期分组：扫描设置、AI 识别、数据管理（导出）、高级/自定义。界面/维护组等候选键落地时再加。

## 11. 留给建造者

- schema JSON 的具体字段命名与形状（满足 6.6 的可渲染性与 secret 协议即可）。
- 批量保存端点的具体路径与参数编码细节（遵守 form-urlencoded 与重复键约定）。
- 类型校验器的错误文案（中文，带键名与合法范围）。
- 通用渲染器的 Alpine 组件拆分方式（遵守现有 `inlineEdit`/组件文件组织）。
- 测试用例的具体选取；至少覆盖：注册自检 panic、每类型校验边界（含排他边界）、已知键禁删、未知键直通、批量保存严格模式（存量非法值阻塞整组、逐键错误、事务原子性）、schema 接口的非法存量值标记、bootstrap 按注册表种子、secret 不回传值。