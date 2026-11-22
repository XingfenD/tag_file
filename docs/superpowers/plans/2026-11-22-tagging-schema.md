# 打标场景 Schema 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 可配置的打标场景（schema）：独立实体 + 三种字段（单选/开关/多选，显式绑定标签），工作台按场景面板打标，值物化进 `TagGroupTag`（默认组），接管式覆盖。

**Architecture:** 新增 `internal/model/tag_schema.go` 两模型 + `internal/repository/schema` 仓储 + `internal/service/schema` 服务（CRUD 校验、面板读视图、ApplyField 物化）；物化 diff 落在 `internal/repository/taggroup` 的事务方法；WebUI 新增 `/schemas` 管理页与 `/api/schemas*` 端点，工作台在媒体区与标签组表格之间渲染「打标场景」卡片区。

**Tech Stack:** Go 1.25 / Gin / GORM（AutoMigrate only）/ Alpine.js / 原生 CSS。

**Spec:** `docs/superpowers/specs/2026-11-22-tagging-schema-design.md`

## Global Constraints

- Go 在 `/usr/local/go/bin`：每条命令前 `export PATH=$PATH:/usr/local/go/bin`。测试用 `go test ./<pkg>/ -v`（全量用 `make test`，需 `temp/test_workspace/`）。
- 分支纪律：commit 前 `git branch --show-current`，在 `master` 上不 commit。当前工作分支 `feat/webui-6.0`。
- `internal/` 改动 → `docs/CHANGELOG.md`；`cmd/webui/` 改动 → `docs/CHANGELOG_webui.md`；条目英文行+中文行相邻、短句、高版本在上。本特性全部并入既有 `[Unreleased→6.0.0]` 段落（用户已裁决「全都在 6.0.0」），`WebuiVersion` 常量保持 `6.0.0` 不再 bump。
- schema 定位是便捷写入口，**无排他**：自由打标下拉不过滤被绑定标签；schema 保存对绑定标签做接管式覆盖（删旧行全集、写当前值）。
- 新表只进 `bootstrap.InitDB` 的 AutoMigrate 清单；各测试包自己的 `setupTestDB` AutoMigrate 清单同步补（`internal/service/schema` 测试自带新清单即可，旧包测试不强制）。
- API 约定：响应走 `respond(c, msg, msgType, extra...)`；表单 `x-www-form-urlencoded`；DELETE 端点参数只读 `c.Query`/`c.Param`。新页面三注册：`cmd/webui/main.go` 页名 slice、`handlers.Server.RegisterRoutes`、`base.html` 导航。

## Review Focus

规格隐含但按任务顺序易漏、最可能伤到使用者的输入类（每条已钉到 owning task 的测试，验证者逐条对照）：

1. **no-op 必须静默幂等**——把没打过的开关关掉、多选取消到空集，不得像 `TagFile/UntagFile` 那样抛错弹红条（Task 1 `TestReplaceRows`、Task 3 `TestApplyBoolNoop`）。
2. **接管式覆盖真实发生**——自由打标/其他途径落进绑定标签的行（含字典外值、多行并存），schema 一次保存后被清成单行（Task 3 `TestApplySingleTakesOverFreeRows`）。
3. **绑定标签被删后**：面板返回 `bind_ok:false` 而非 500，apply 拒绝并给「绑定已失效」；删 schema 本身不删标签（Task 3 `TestApplyStaleBinding`、Task 2 `TestRemoveSchemaLeavesTags`）。
4. **字典外取值被拒**——apply single 传不在 `TagValue` 字典的值返回 400，不落脏行（Task 3 `TestApplySingleRejectsUnknownValue`）。
5. **畸形入参出信封**——未知 field_id、未注册 file_path 一律 `{ok:false}`+400，不 panic 不 500（Task 4 `TestSchemaAPIErrorEnvelope`）。

---

### Task 1: 模型、仓储与物化事务

**Files:**
- Create: `internal/model/tag_schema.go`
- Modify: `internal/bootstrap/init.go`（AutoMigrate 清单加两模型）
- Create: `internal/repository/schema/repo.go|schema_crud.go|field_crud.go|find.go`
- Create: `internal/repository/schema/schema_repo_test.go`
- Modify: `internal/repository/taggroup/tag_apply.go`（新建）
- Modify: `internal/repository/taggroup/tag_find.go`（加 `GetRowsByTags`）
- Create: `internal/repository/taggroup/tag_apply_test.go`

**Interfaces:**
- Consumes: 既有 `model.TagGroupTag`、`pkg/storage`；`taggroup.Repository.FindDefaultGroup(fileID)`。
- Produces:
  - `model.TagSchema{SchemaID uint; SchemaName string(uniqueIndex); Description string; SortOrder int}`，`TableName() "tag_schemas"`；`model.TagSchemaField{FieldID uint; SchemaID uint(index); Label string; FieldType string('single'|'bool'|'multi'); Widget string(''|'select'|'stars'); BindTagID uint(index); SortOrder int}`，`TableName() "tag_schema_fields"`。
  - `schema.Repository`：`NewRepository()`、`ListSchemas() ([]model.TagSchema, error)`（Preload Fields，按 sort_order,schema_id）、`GetSchema(id uint) (*model.TagSchema, error)`（无则 nil,nil）、`CreateSchema(*model.TagSchema) error`、`UpdateSchema(*model.TagSchema) error`、`DeleteSchema(id uint) error`（事务连带 fields）、`ListFields(schemaID uint) ([]model.TagSchemaField, error)`、`CreateField(*model.TagSchemaField) error`、`UpdateField(*model.TagSchemaField) error`、`DeleteField(id uint) error`、`FieldByBindTagID(tagID uint) (*model.TagSchemaField, error)`（无则 nil,nil）。
  - `taggroup.Repository.ReplaceRows(groupID, fileID uint, tagIDs []uint, rows []ApplyRow) (int, error)`，`ApplyRow{TagID uint; Value string}`——单事务：删 `tag_id IN tagIDs` 的全部行 → 插 rows → `tag_cnt = MAX(tag_cnt + delta, 0)`（delta=len(rows)-删除数）；`tagIDs` 与 `rows` 均空时返回 0,nil 不碰库。
  - `taggroup.Repository.GetRowsByTags(fileID uint, tagIDs []uint) ([]model.TagGroupTag, error)`。

- [ ] **Step 1: 写模型并挂进 AutoMigrate**（字段/gorm tag 按 Produces 逐字；`init.go` 清单插在 `&model.TagValue{}` 之后）。
- [ ] **Step 2: 写 `schema_crud.go`/`field_crud.go`/`find.go`**（一文件一 concern，链式风格从简即可，与 `taggroup` 包一致：方法直挂 `*Repository`）。
- [ ] **Step 3: 写失败测试 `schema_repo_test.go`**：`TestSchemaRepoCRUD`——建 schema+2 fields → `ListSchemas` 带出 Preload fields 且排序按 sort_order → `FieldByBindTagID` 命中/未命中返回 nil,nil → `DeleteSchema` 后 fields 行清空。setupTestDB 模板抄 `internal/service/file/file_service_test.go`（AutoMigrate 含全部既有模型 + 本任务两模型）。

Run: `go test ./internal/repository/schema/ -v` → Expected: FAIL（编译失败=未实现视为失败）。

- [ ] **Step 4: 实现至通过** Run 同上 → PASS。
- [ ] **Step 5: 写失败测试 `tag_apply_test.go`**：`TestReplaceRows`——seed 文件+默认组+`星级` 标签三行值(1,2,3) → `ReplaceRows(g,f,[星级],[])` 删全行且 tag_cnt 回落、返回负 delta；再写单行 → 组内该行唯一；**no-op**：空 tagIDs+空 rows 返回 0 且零 SQL 影响；`TestReplaceRowsMultiTagUnion`——tagIDs 两个子标签、rows 一个，旧两行删净、新行唯一、delta 正确。
- [ ] **Step 6: 实现 `tag_apply.go` 至通过** Run: `go test ./internal/repository/taggroup/ -v` → PASS，且 `go test ./... -count=1` 无既有用例回归。
- [ ] **Step 7: Commit** `git add internal/ && git commit -m "feat(core): schema models, repositories and replace-rows materialization"`（core changelog 留到 Task 3 一起写，避免碎片条目）。

### Task 2: schema 服务 CRUD 与校验

**Files:**
- Create: `internal/service/schema/service.go`（构造+CRUD）、`internal/service/schema/validate.go`
- Create: `internal/service/schema/schema_service_test.go`
- Modify: `internal/repository/tag/tree.go`（如需：`ChildIDs(parentID uint) ([]uint, error)`——复用 `ListAll` 全量内存遍历模式，无则新建 `tree_children.go`）

**Interfaces:**
- Consumes: Task 1 的 `schema.Repository` 全部方法、`taggroup.Repository`、`tag.Repository.Find().ByID()`、`tag` 包既有子树遍历（`GetDescendantIDs` 的内存模式）。
- Produces（`service/schema`，Task 3/4 依赖）:
  - `type Service struct{...}`、`NewService() *Service`
  - `AddSchema(name, description string) (*model.TagSchema, error)`——重名报 `场景 "X" 已存在`
  - `UpdateSchema(id uint, name, description string, sortOrder int) error`
  - `RemoveSchema(id uint) error`——只删定义，不碰标签与已物化行
  - `AddField(schemaID uint, label, fieldType, widget string, bindTagID uint) (*model.TagSchemaField, error)`
  - `UpdateField(fieldID uint, label, widget string, bindTagID uint) error`
  - `RemoveField(fieldID uint) error`
  - `ChildTagIDs(parentID uint) ([]uint, error)`（不含父自身，深度 ≥1）

- [ ] **Step 1: 写失败测试**（setupTestDB 同 Task 1 模板 + 直接 SQL seed 分类/标签/字典）：
  - `TestAddSchemaDuplicateName`：第二次建同名 → 错误含「已存在」。
  - `TestAddFieldValidation`：type 非三枚举 / bind 标签不存在 / widget='stars' 配到 bool → 各自报错；合法 single 通过。
  - `TestFieldBindUniqueness`：字段 A 绑 tag 5 后，字段 B 再绑 tag 5 → 报 `该标签已被字段 "A" 绑定`；UpdateField 改绑同值自身放行。
  - `TestRemoveSchemaLeavesTags`：schema+field 绑定既有标签 → RemoveSchema 后 tags/tag_values 行原样。
- [ ] **Step 2: Run → FAIL**：`go test ./internal/service/schema/ -v`。
- [ ] **Step 3: 实现** `validate.go` 收口三类校验（枚举集合、widget 仅 single 合法值 `''/select/stars`、bind 存在性 + `FieldByBindTagID` 占用），CRUD 方法薄。
- [ ] **Step 4: Run → PASS** + `go test ./internal/... -count=1` 干净。
- [ ] **Step 5: Commit** `git commit -am "feat(core): schema service CRUD with bind validation"`。

### Task 3: 面板读视图与 ApplyField 物化

**Files:**
- Create: `internal/service/schema/panels.go`、`internal/service/schema/apply.go`
- Modify: `internal/service/schema/schema_service_test.go`（追加）
- Create: `docs/CHANGELOG.md` 条目（随本任务 commit）

**Interfaces:**
- Consumes: Task 1 `ReplaceRows/GetRowsByTags/FindDefaultGroup/CreateGroup`、Task 2 `Service/ChildTagIDs`、`tag.Repository`（字典 `ListTagValues` 的 repo 级查询：`tagvalue` 行按 tag_id 直查）。
- Produces（JSON 形态即 WebUI 契约，Task 4/6 依赖，字段名逐字）:
  - `type OptionView struct{ Value string \`json:"value"\`; TagID uint \`json:"tag_id,omitempty"\`; Name string \`json:"name,omitempty"\` }`（single 填 Value；multi 填 TagID+Name；bool 空）
  - `type FieldView struct{ FieldID uint \`json:"field_id"\`; Label string \`json:"label"\`; Type string \`json:"type"\`; Widget string \`json:"widget"\`; BindOK bool \`json:"bind_ok"\`; Options []OptionView \`json:"options"\`; Value string \`json:"value"\`; Checked []uint \`json:"checked"\` }`
  - `type SchemaPanel struct{ SchemaID uint \`json:"schema_id"\`; Name string \`json:"name"\`; Fields []FieldView \`json:"fields"\` }`
  - `(s *Service) PanelsForFile(filePath string) ([]SchemaPanel, error)`——文件不存在/未注册返回错误；无默认组时按需创建（对齐 `file.Service.TagFile` 行为）；single 当前值取绑定标签行中 `id` 最大一行的 `tag_value`（多行并存的面板读规则）；绑定标签已删 → `BindOK:false`、Options 空、不 500。
  - `(s *Service) ApplyField(filePath string, fieldID uint, value string, checked []uint) error`——按类型物化：single 要求 `value ∈ 字典`（`''` = 清行）；bool `value=="true"` 写无值行否则清空；multi `checked ⊆ ChildTagIDs` 后整组接管；绑定失效报 `该字段的绑定标签已失效`。

- [ ] **Step 1: 写失败测试**：
  - `TestPanelsForFileShapes`：single(stars)+bool+multi 三字段一 schema → 面板结构、Options 来源（字典/子标签）、初值空态正确。
  - `TestPanelsStaleBinding`：绑定标签被物理删 → `BindOK:false`，同 schema 其余字段照常返回。
  - `TestPanelsSinglePicksLatestRow`：绑定标签自由落两行（id 小值 "3"、id 大值 "4"）→ Value=="4"。
  - `TestApplySingleTakesOverFreeRows`：同上两行自由值 → apply value "2" 后组内该标签仅剩一行值 "2"，`tag_cnt` 与行数一致。
  - `TestApplySingleRejectsUnknownValue`：apply value "9"（字典外）→ 报错且零落库。
  - `TestApplyBoolNoop`：从未打标 → 关（value "false"）不报错、tag_cnt 不动。
  - `TestApplyStaleBinding`：字段绑定标签被物理删后 apply → 报 `该字段的绑定标签已失效`，零落库。
  - `TestApplyMultiClearsAll`：勾两子标签后传空 checked → 子标签行清零，父标签行若有不受碰。
- [ ] **Step 2: Run → FAIL**：`go test ./internal/service/schema/ -run 'TestPanels|TestApply' -v`。
- [ ] **Step 3: 实现 `panels.go`/`apply.go`** 至全部 PASS。
- [ ] **Step 4: `go test ./... -count=1` 全绿**（`make test` 若 temp 工作区缺失先建）。
- [ ] **Step 5: Commit**（含 `docs/CHANGELOG.md` 一条：`Add configurable tagging schema (models, service, materialized writes)` + 同义中文行）。

### Task 4: WebUI API 端点

**Files:**
- Create: `cmd/webui/api/schema.go`
- Modify: `cmd/webui/api/api.go`（`NewRouter` 形参加 `*schemaSvc.Service`、`RegisterRoutes` 挂路由）、`cmd/webui/main.go`（构造 schemaSvc 传入）、`cmd/webui/api/routes_smoke_test.go`（补路由行）、`cmd/webui/api/*_test.go` 里手工 `NewRouter(...)` 的既有调用点同步补参
- Create: `cmd/webui/api/schema_test.go`

**Interfaces:**
- Consumes: Task 2/3 的 `schema.Service` 全部方法（JSON 契约即 FieldView/SchemaPanel 的 json tag）。
- Produces（路由表，Task 5/6 的前端按此调用）:
  - `GET /api/schemas` → `{ok, schemas:[{schema_id,schema_name,description,sort_order,fields:[...]}]}`（管理页：fields 里带 `bind_tag_name`，在 handler 层用 `tagRepo` 补名，找不到的置 `"（已删除）"`）
  - `POST /api/schemas`（`name`,`description`）；`PUT /api/schemas/:id`（`name`,`description`,`sort_order`）；`DELETE /api/schemas/:id`
  - `POST /api/schemas/:id/fields`（`label`,`type`,`widget`,`bind_tag_id`）；`PUT /api/schemas/fields/:id`（同 4 参+`sort_order`）；`DELETE /api/schemas/fields/:id`
  - `GET /api/schemas/panels?file_path=` → `{ok, panels:[SchemaPanel]}`
  - `POST /api/schemas/apply`（`file_path`,`field_id`,`value`；multi 用重复键 `tag_ids`）
- 全部走 `respond()`；service 错误 → `respond(c, err.Error(), "error")`（400）。

- [ ] **Step 1: 写失败测试 `schema_test.go`**（复用 api 包 `newTestEngine`/`seedFixture`；`NewRouter` 签名变更后编译失败即 RED）：
  - `TestSchemaAPIRoundTrip`：POST 建 schema → PUT 字段绑星级标签(stars) → GET /api/schemas 列出绑定名 → GET panels?file_path 见字段 → POST apply value=2 → 重取 panels `value=="2"` → DB 断言组内一行。
  - `TestSchemaAPIErrorEnvelope`：apply 未知 field_id、未注册 file_path、字典外 value → 三个 400 + `ok:false` + msg 非空。
  - `TestSchemaFieldFormArrays`：multi apply 传重复键 `tag_ids=7&tag_ids=8` → 两行落库（钉 URLSearchParams 重复键约定）。
- [ ] **Step 2: `go test ./cmd/webui/api/ -run TestSchema -v` → FAIL**。
- [ ] **Step 3: 实现 `schema.go` handler 组 + 路由挂载 + main.go 接线 + 既有 NewRouter 调用点补参**。
- [ ] **Step 4: Run → PASS**；`routes_smoke_test.go` 的 routes 表补 6 条（panels/apply/CRUD 各一），全套 `go test ./cmd/webui/... -count=1` 绿。
- [ ] **Step 5: Commit** `feat(webui): schema CRUD/panels/apply JSON API`（本任务只加 API 不出页面，changelog 与 Task 6 合并写）。

### Task 5: 场景管理页

**Files:**
- Create: `cmd/webui/templates/schema.html`、`cmd/webui/static/js/pages/schema/main.js`、`cmd/webui/static/css/pages/schema.css`
- Modify: `cmd/webui/main.go`（页名 slice 加 `"schema"`；静态 css/js 若有清单同步）、`cmd/webui/handlers/handlers.go`（`r.GET("/schemas", s.HandlePage("schema", "schema"))`——Active 键与 nav 一致）、`cmd/webui/templates/base.html`（标签管理之后加导航项，`title/aria` 中文「场景管理」；折叠态复用 `.nav-label` 既有规则）、`cmd/webui/static_test.go`
- Create: 如需新图标：`templates/partials/icons.html` 加 `i-shapes`（从简几何 path）

**Interfaces:**
- Consumes: Task 4 全部端点。
- Produces: 页面模板名 `"schema"`、`Alpine.data('schemaPage')`、管理页 DOM 契约（静态门钉）：`#schema-admin`、行内 `x-data="inlineEdit(...)"` 复用。

- [ ] **Step 1: 写失败静态门 `TestSchemaAdminPage`**（`cmd/webui/static_test.go`）：页名 slice 含 `"schema"`；handlers.go 含 `"/schemas"`；base.html 含 `场景管理`；schema.html 存在且含 `schemaPage`；main.js 含 `'/api/schemas'`。
- [ ] **Step 2: Run → FAIL**：`go test ./cmd/webui/ -run TestSchemaAdminPage -v`。
- [ ] **Step 3: 实现页面**：左列 schema 列表（新建/重命名/删除，inlineEdit + confirmDelete 复用），右列选中 schema 的字段表（label、类型徽章、绑定标签名、widget、排序上移下移、每行编辑弹层用现有 `Modal.show`：类型 select→three 枚举、绑定标签 select 用现有 tag 搜索 partial、stars 仅 single 出现）。空态用 `{{template "empty"}}`。spec 两条附属要求同页落地：字段编辑弹层的绑定标签下拉给「就地新建标签」输入框（调既有 `POST /api/tags`，成功后自动选中新标签，不开新端点）；列表侧对绑定字典为空（single/bool）或子标签为空（multi）的字段给黄条提示，stars 值数 ≠5 一并提示——均提示不拦截。
- [ ] **Step 4: Run 静态门 → PASS**；`go test ./cmd/webui/... -count=1` 全绿。
- [ ] **Step 5: 浏览器手验**（:8088 只读容器不可写 → 用 `go run ./cmd/webui --source <临时目录> --addr :8091` 起本地实例走一遍 CRUD，验完杀进程删临时目录）。
- [ ] **Step 6: Commit** `feat(webui): schema admin page`。

### Task 6: 工作台「打标场景」区

**Files:**
- Modify: `cmd/webui/templates/file_detail.html`（媒体区 stage 之后、标签组表格之前插 `#schema-panels` 区）、`cmd/webui/static/js/pages/file/detail.js`（panels 状态+加载+apply）、`cmd/webui/static/css/components/workbench.css`（`.schema-panel/.schema-field/.stars` 等）、`cmd/webui/static_test.go`
- Modify: `docs/CHANGELOG_webui.md`（[6.0.0] Added 追加两行英中：管理页 + 工作台场景区；与 Task 4/5 未写的 API 条目合并为一条亦可，保持一改动一条目）

**Interfaces:**
- Consumes: Task 4 `GET /api/schemas/panels` / `POST /api/schemas/apply` 的逐字 JSON；detail.js 既有 `applyDetail/reloadGroupsAndTags` 落点。
- Produces: detail.js 新增 `panels: []`、`loadPanels()`、`applyField(field, value)`、`applyMulti(field, tagID, on)`；DOM 契约 `#schema-panels`、`.schema-panel`、`button.star[aria-pressed]`。

- [ ] **Step 1: 写失败静态门 `TestSchemaWorkbench`**：file_detail.html 含 `id="schema-panels"` 且位于 `#tag-groups`（以现模板锚点实际名为准，执行时查）之前；detail.js 含 `'/api/schemas/panels'`、`'/api/schemas/apply'`、`loadPanels` 在 `applyDetail` 与 `navigateToFile` 路径被调；workbench.css 含 `.schema-panel` 且无内联 style 回流。
- [ ] **Step 2: Run → FAIL**：`go test ./cmd/webui/ -run TestSchemaWorkbench -v`。
- [ ] **Step 3: 实现**：
  - `loadPanels()`：GET panels（`filePath` 入 query），失败静默（工作台主功能不因 schema 区报错瘫痪，console.warn）。
  - 渲染：每 schema 一张 `.schema-panel`（面板头 = schema 名），字段一行：label + 控件。single(stars)=5 个 `button.star`（`aria-pressed`，Options 按序映射、不足 5 星置灰）、single(select)=原生 select 复用主题样式、bool=现有 checkbox 样式、multi=chip 按钮组（`.chip[aria-pressed]` 复用 tag 面板样式）；`bind_ok:false` 的字段显示「绑定已失效」灰字不留空控件。
  - `applyField`：POST apply（multi 组装重复键 `tag_ids`）→ 成功后本地更新 `panels` 内该字段 + `reloadGroupsAndTags()`（物化行出现在标签表格里）+ `refreshSidebarRow`（tag_cnt 变了）。失败 `showToast(msg,'error')`。
  - 文件切换路径（nextFile/navigateToFile/onPopState）随 `applyDetail` 重载 panels。
- [ ] **Step 4: Run 静态门 → PASS**；`go test ./... -count=1` 全绿。
- [ ] **Step 5: 浏览器手验**：本地 :8091 实例——建评级 schema(stars) → 工作台点星 → 标签表格出现 `星级:4`、重渲染回显正确；自由下拉再打 `星级:3` → schema 再保存 → 组内只剩一行。验完关进程。
- [ ] **Step 6: Commit + CHANGELOG_webui.md**（同 commit，`WebuiVersion` 保持 `6.0.0` 不动）。

---

## 完成定义

- `make test` 全绿；`go vet ./...` 干净。
- 手工冒烟：评级(stars)/已看(bool)/视角(multi) 三 schema 在工作台可用、覆盖语义与失效提示符合 spec 裁决。
- 两个 changelog 条目按 AGENTS 格式落位；无 `internal/` 对外泄露（schema 仅在 `internal/` 与 `cmd/webui` 引用）。
