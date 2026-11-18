package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

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
	RuleMultiline RuleKind = "multiline" // 渲染效果：textarea
	// behavior —— 行为指令（服务端强制，前端仅驱动 UX）
	RuleSecret RuleKind = "secret" // 掩码 + 留空不覆盖；渲染效果：password
)

// RuleRole 是指令的三种角色：校验、渲染、行为。三者共用注册表、下发格式
// 与降级机制，渲染/行为不是第二套机制。
type RuleRole string

const (
	RoleValidate RuleRole = "validate"
	RoleRender   RuleRole = "render"
	RoleBehavior RuleRole = "behavior"
)

// Rule 是下发给前端的一条指令：语义由 Kind+Params 定义，文案与渲染效果
// 随 payload 下发。
type Rule struct {
	Kind    RuleKind
	Role    RuleRole
	Params  map[string]string // 复杂值 JSON 编码（如 oneof 的 options）
	Hint    string            // 专属 UX 文案，来自 ruleDefs（现仅 secret 的 password 占位）
	Message string            // 校验失败文案（已按 Params 渲染，不含键名/项序号）；非 validate 为空
	Widget  string            // 渲染效果（password/textarea/select…），来自 ruleDefs；无则空
}

// RuleSet 是一个输入框分解出的全部指令：Rules 作用于值（Single）或每一项
// （DynamicArray）；PositionalRules 仅 Array，与定长同长、逐位一组；
// ContainerRules 作用于容器本身（item_count / fixed_arity）。
type RuleSet struct {
	Rules           []Rule
	PositionalRules [][]Rule
	ContainerRules  []Rule
}

// ruleDef 是一条原语的定义：校验动作、UI 文案、渲染效果同处一地，全项目唯一。
type ruleDef struct {
	Role    RuleRole
	Hint    string                                    // 专属 UX 文案（secret 的 password 占位），静态
	Message string                                    // 失败文案模板，按 Params 渲染（仅 validate）
	Widget  string                                    // 渲染效果（""=无）
	Check   func(p map[string]string, v string) error // 仅 validate 必有；render/behavior 为 nil
}

// ruleDefs 是指令注册表：语义、文案、渲染效果的唯一出处。服务端校验错误
// 与 API 下发的 message 用同一模板渲染，逐字一致；Check 的通用失败文案经
// ruleError 回读本注册表（不在 Check 内重写字面量）；Check 可返回细分文案
// （如 url 的各分支），细分文案也住在条目内，仍是单源。字面量在 init()
// 而非 var 初始化器中装配：注册表持有 Check 函数值、Check 体内又回读
// ruleDefs，var 初始化器会构成初始化环（Go 依赖分析把函数值引用也当
// 依赖边）；init() 在全部包级变量之后运行，环自然断开。
var ruleDefs map[RuleKind]ruleDef

func init() {
	ruleDefs = map[RuleKind]ruleDef{
		RuleRequired:   {Role: RoleValidate, Message: "取值不能为空", Check: checkRequired},
		RuleInt:        {Role: RoleValidate, Message: "取值必须是整数", Check: checkInt},
		RuleIntRange:   {Role: RoleValidate, Message: "取值超出合法范围，应为 {{range}}", Check: checkIntRange},
		RuleURL:        {Role: RoleValidate, Message: "取值必须是合法的 http 或 https 地址", Check: checkURL},
		RuleRegex:      {Role: RoleValidate, Message: "不是合法的正则表达式", Check: checkRegex},
		RulePattern:    {Role: RoleValidate, Message: "取值不符合匹配模式 {{pattern}}", Check: checkPattern},
		RuleOneOf:      {Role: RoleValidate, Message: "取值必须是以下之一: {{candidates}}", Widget: "select", Check: checkOneOf},
		RuleBool:       {Role: RoleValidate, Message: "取值只能是 true 或 false", Check: checkBool},
		RuleItemCount:  {Role: RoleValidate, Message: "项数超出合法范围", Check: checkItemCount},
		RuleFixedArity: {Role: RoleValidate, Message: "需要恰好 {{n}} 项", Check: checkFixedArity},
		RuleMultiline:  {Role: RoleRender, Widget: "textarea"},
		RuleSecret: {
			Role:   RoleBehavior,
			Hint:   "已设置，留空不修改", // 现状写死在 config_field.html 的占位文案，收编进注册表
			Widget: "password",
			// Check nil：secret 不产生校验语义；掩码与留空不覆盖由服务端强制（spec §8）。
		},
	}
}

// RulesOf 把类型化约束描述符分解为指令（spec §4.3）。声明层是 ergonomics，
// 指令才是引擎与协议。输出顺序稳定：先值类型内建指令与无渲染效果的校验
// 指令，再按注册表优先级（secret > oneof > multiline）输出带 Widget 的指令
// ——互斥组合已由 validateDescriptor 排除，优先级是防御性兜底。
func RulesOf(d InputBoxDescriptor) RuleSet {
	var rs RuleSet
	switch d.Container {
	case ContainerArray:
		n := arityOf(d)
		rs.ContainerRules = append(rs.ContainerRules, makeRule(RuleFixedArity, map[string]string{
			"n": strconv.Itoa(n),
		}))
		for i := 0; i < n; i++ {
			rs.PositionalRules = append(rs.PositionalRules, positionRulesOf(d, i))
		}
	case ContainerDynamicArray:
		if c := d.DynamicListConstraint; c != nil {
			params := map[string]string{}
			if c.MinItems != nil {
				params["min_items"] = strconv.Itoa(*c.MinItems)
			}
			if c.MaxItems != nil {
				params["max_items"] = strconv.Itoa(*c.MaxItems)
			}
			rs.ContainerRules = append(rs.ContainerRules, makeRule(RuleItemCount, params))
		}
		rs.Rules = valueRulesOf(d)
	case ContainerSingle:
		rs.Rules = valueRulesOf(d)
	}
	return rs
}

// arityOf 返回定长数组的长度（= 对应约束列表长度）。
func arityOf(d InputBoxDescriptor) int {
	switch d.ValueType {
	case ValueInt:
		return len(d.Constraint.IntConstraintList)
	case ValueString:
		return len(d.Constraint.StringConstraintList)
	case ValueBool:
		return len(d.Constraint.BoolConstraintList)
	}
	return 0
}

// positionRulesOf 返回 Array 第 i 位的指令组。
func positionRulesOf(d InputBoxDescriptor, i int) []Rule {
	switch d.ValueType {
	case ValueInt:
		return intRulesOf(d.Constraint.IntConstraintList[i])
	case ValueString:
		return stringRulesOf(d.Constraint.StringConstraintList[i])
	case ValueBool:
		return boolRulesOf()
	}
	return nil
}

// valueRulesOf 返回值级（Single）/项级（DynamicArray）的指令组。
// 空约束 = 只有 ValueType 内建指令（Int→int，Bool→bool，String→无）。
func valueRulesOf(d InputBoxDescriptor) []Rule {
	switch d.ValueType {
	case ValueInt:
		return intRulesOf(d.Constraint.IntConstraint)
	case ValueString:
		return stringRulesOf(d.Constraint.StringConstraint)
	case ValueBool:
		return boolRulesOf()
	}
	return nil
}

func intRulesOf(c *IntConstraintDescriptor) []Rule {
	rules := []Rule{makeRule(RuleInt, nil)}
	if c != nil {
		rules = append(rules, makeRule(RuleIntRange, intRangeParams(c)))
	}
	return rules
}

func boolRulesOf() []Rule {
	return []Rule{makeRule(RuleBool, nil)}
}

func stringRulesOf(c *StringConstraintDescriptor) []Rule {
	if c == nil {
		return nil
	}
	var rules []Rule
	if c.Required {
		rules = append(rules, makeRule(RuleRequired, nil))
	}
	switch c.Format {
	case FormatURL:
		rules = append(rules, makeRule(RuleURL, nil))
	case FormatRegex:
		rules = append(rules, makeRule(RuleRegex, nil))
	}
	if c.Pattern != "" {
		rules = append(rules, makeRule(RulePattern, map[string]string{"pattern": c.Pattern}))
	}
	if c.Secret {
		rules = append(rules, makeRule(RuleSecret, nil))
	}
	if len(c.Options) > 0 {
		rules = append(rules, makeRule(RuleOneOf, oneOfParams(c.Options)))
	}
	if c.Multiline {
		rules = append(rules, makeRule(RuleMultiline, nil))
	}
	return rules
}

// intRangeParams 序列化边界参数：Min/Max 用显示单位（formatNum 无尾零），
// 布尔/数值参数一律序列化为字符串（"true"/"1"），未声明的边界不输出键。
// shift 恒输出：前端换算小数位时直接从 Params 读取，无需默认值约定。
func intRangeParams(c *IntConstraintDescriptor) map[string]string {
	params := map[string]string{"shift": strconv.Itoa(c.Shift)}
	if c.Min != nil {
		params["min"] = formatNum(*c.Min)
		if c.MinExclusive {
			params["min_exclusive"] = "true"
		}
	}
	if c.Max != nil {
		params["max"] = formatNum(*c.Max)
		if c.MaxExclusive {
			params["max_exclusive"] = "true"
		}
	}
	return params
}

// optionParam 是 oneof 指令 options 参数的 JSON 投影（小写键，spec §4.3）。
// Option 本体沿用声明层结构，不为此加 json tag。
type optionParam struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func oneOfParams(options []Option) map[string]string {
	items := make([]optionParam, 0, len(options))
	for _, o := range options {
		items = append(items, optionParam{Value: o.Value, Label: o.Label})
	}
	// 字段全为 string，序列化失败只可能是编程错误。
	data, err := json.Marshal(items)
	if err != nil {
		panic(fmt.Sprintf("schema: 枚举选项序列化失败: %v", err))
	}
	return map[string]string{"options": string(data)}
}

// makeRule 按注册表定义与参数实例化一条指令：Hint/Widget 取静态定义，
// Message 按模板渲染。
func makeRule(kind RuleKind, params map[string]string) Rule {
	def := ruleDefs[kind]
	return Rule{
		Kind:    kind,
		Role:    def.Role,
		Params:  params,
		Hint:    def.Hint,
		Message: renderMessage(def, params),
		Widget:  def.Widget,
	}
}

// renderMessage 按参数渲染失败文案模板：{{range}} 为区间写法（显示单位）、
// {{candidates}} 为顿号连接的候选值，其余 {{key}} 直接取同名参数
// （n、pattern 等）。
func renderMessage(def ruleDef, params map[string]string) string {
	msg := def.Message
	if msg == "" {
		return ""
	}
	if strings.Contains(msg, "{{range}}") {
		msg = strings.ReplaceAll(msg, "{{range}}", describeRuleRange(params))
	}
	if strings.Contains(msg, "{{candidates}}") {
		msg = strings.ReplaceAll(msg, "{{candidates}}", joinCandidates(params["options"]))
	}
	for k, v := range params {
		msg = strings.ReplaceAll(msg, "{{"+k+"}}", v)
	}
	return msg
}

// ruleError 用某原语自身的注册表条目按其参数渲染通用失败文案：Check 内不
// 重写文案字面量，ruleDefs 仍是唯一出处。细分文案（url 各分支、item_count
// 项数界限等 spec 认可的特例）不在模板里，留在各 Check 内。
func ruleError(kind RuleKind, p map[string]string) error {
	return errors.New(renderMessage(ruleDefs[kind], p))
}

// describeRuleRange 用区间写法描述 int_range 的合法范围（显示单位），排他
// 边界用圆括号表示——语义与旧 describeRange（validate.go）一致，只是边界
// 来源从 Field 换成指令 Params（字符串形式，序列化时已 formatNum）。
func describeRuleRange(p map[string]string) string {
	min, hasMin := p["min"]
	max, hasMax := p["max"]
	left, right := "[", "]"
	if !hasMin || p["min_exclusive"] == "true" {
		left = "("
	}
	if !hasMax || p["max_exclusive"] == "true" {
		right = ")"
	}
	switch {
	case hasMin && hasMax:
		return left + min + ", " + max + right
	case hasMin:
		return left + min + ", +∞" + right
	case hasMax:
		return "(-∞, " + max + right
	default:
		return "任意值"
	}
}

// joinCandidates 把 oneof 的 options 参数投影为顿号连接的候选值串。
func joinCandidates(optionsJSON string) string {
	opts, err := parseOptions(optionsJSON)
	if err != nil {
		return ""
	}
	values := make([]string, 0, len(opts))
	for _, o := range opts {
		values = append(values, o.Value)
	}
	return strings.Join(values, "、")
}

func parseOptions(optionsJSON string) ([]optionParam, error) {
	if optionsJSON == "" {
		return nil, nil
	}
	var opts []optionParam
	if err := json.Unmarshal([]byte(optionsJSON), &opts); err != nil {
		return nil, err
	}
	return opts, nil
}

// checkRuleKindsKnown 是 spec §5.1 规则 6 的防御性检查：分解出的每个 kind
// 必须在 ruleDefs 中。类型化声明下由构造保证成立，此处兜底「新增原语忘写
// 注册表」的编程错误——引擎绝不静默放行未知 kind。
func checkRuleKindsKnown(rs RuleSet) error {
	groups := append([][]Rule{rs.Rules, rs.ContainerRules}, rs.PositionalRules...)
	for _, group := range groups {
		for _, r := range group {
			if _, ok := ruleDefs[r.Kind]; !ok {
				return fmt.Errorf("未知的校验规则 '%s'", r.Kind)
			}
		}
	}
	return nil
}

// ---- 各原语的 Check 实现（文案与细分文案均为单源，住注册表条目内）----

func checkRequired(p map[string]string, v string) error {
	if v == "" {
		return ruleError(RuleRequired, p)
	}
	return nil
}

func checkInt(p map[string]string, v string) error {
	if _, err := strconv.ParseInt(v, 10, 64); err != nil {
		return ruleError(RuleInt, p)
	}
	return nil
}

// checkIntRange 在显示单位上判界：存储整数按 shift 换算（/ 10^shift）后与
// min/max 比较。非整数 wire（如 "0.5"）按 RuleInt 的通用文案拒绝——传输
// 一律整数，小数只存在于显示层。
func checkIntRange(p map[string]string, v string) error {
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return ruleError(RuleInt, p)
	}
	shift := 0
	if s, ok := p["shift"]; ok {
		if parsed, err := strconv.Atoi(s); err == nil {
			shift = parsed
		}
	}
	display := float64(n) / math.Pow(10, float64(shift))
	if min, ok := parseFloatParam(p, "min"); ok {
		if p["min_exclusive"] == "true" {
			if display <= min {
				return ruleError(RuleIntRange, p)
			}
		} else if display < min {
			return ruleError(RuleIntRange, p)
		}
	}
	if max, ok := parseFloatParam(p, "max"); ok {
		if p["max_exclusive"] == "true" {
			if display >= max {
				return ruleError(RuleIntRange, p)
			}
		} else if display > max {
			return ruleError(RuleIntRange, p)
		}
	}
	return nil
}

// checkURL 的细分文案（空串/前缀/主机名）沿用旧 validateURL 逐字形态，
// 仍是条目内单源；解析失败的通用文案走注册表模板。
func checkURL(p map[string]string, v string) error {
	if v == "" {
		return fmt.Errorf("取值必须是 http 或 https 地址，不能为空")
	}
	u, err := url.Parse(v)
	if err != nil {
		return ruleError(RuleURL, p)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("取值必须以 http:// 或 https:// 开头")
	}
	if u.Host == "" {
		return fmt.Errorf("取值必须包含主机名")
	}
	return nil
}

func checkRegex(p map[string]string, v string) error {
	if _, err := regexp.Compile(v); err != nil {
		return ruleError(RuleRegex, p)
	}
	return nil
}

// checkPattern 只约束非空值（沿用现状）。模式在描述符检查期已预编译，
// 编译失败分支只是防御（文案为细分特例，不在模板里）。
func checkPattern(p map[string]string, v string) error {
	if v == "" {
		return nil
	}
	re, err := regexp.Compile(p["pattern"])
	if err != nil {
		return fmt.Errorf("匹配模式无效: %v", err)
	}
	if !re.MatchString(v) {
		return ruleError(RulePattern, p)
	}
	return nil
}

func checkOneOf(p map[string]string, v string) error {
	opts, _ := parseOptions(p["options"])
	for _, o := range opts {
		if o.Value == v {
			return nil
		}
	}
	return ruleError(RuleOneOf, p)
}

func checkBool(p map[string]string, v string) error {
	if v != "true" && v != "false" {
		return ruleError(RuleBool, p)
	}
	return nil
}

// parseStringArray 解析 JSON 字符串数组（存储形态，元素一律是字符串，
// Int/Bool 元素也不例外——与现状 scan_exclude 行编辑器传输一致）。
// JSON null 会 Unmarshal 成 nil 切片且不报错，必须显式拒绝：spec §5.2
// 要求容器是字符串数组，放行 null 会静默清空排除列表等存量配置。
func parseStringArray(value string) ([]string, error) {
	var items []string
	if err := json.Unmarshal([]byte(value), &items); err != nil || items == nil {
		return nil, fmt.Errorf(`取值必须是 JSON 字符串数组，例如 ["a","b"]`)
	}
	return items, nil
}

// checkItemCount 收到原始存储形态（JSON 字符串数组），项数界限按细分文案
// 报告，通用文案（项数超出合法范围）留给前端本地预校验。
func checkItemCount(p map[string]string, v string) error {
	items, err := parseStringArray(v)
	if err != nil {
		return err
	}
	if min, ok := parseIntParam(p, "min_items"); ok && len(items) < min {
		return fmt.Errorf("项数不能少于 %d", min)
	}
	if max, ok := parseIntParam(p, "max_items"); ok && len(items) > max {
		return fmt.Errorf("项数不能多于 %d", max)
	}
	return nil
}

func checkFixedArity(p map[string]string, v string) error {
	items, err := parseStringArray(v)
	if err != nil {
		return err
	}
	n, _ := parseIntParam(p, "n")
	if len(items) != n {
		return ruleError(RuleFixedArity, p)
	}
	return nil
}

func parseFloatParam(p map[string]string, key string) (float64, bool) {
	s, ok := p[key]
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func parseIntParam(p map[string]string, key string) (int, bool) {
	s, ok := p[key]
	if !ok {
		return 0, false
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return v, true
}

// formatNum 用最短无尾零写法序列化一个显示单位的浮点边界（如 1 → "1"、
// 0.5 → "0.5"），供 int_range 指令的 min/max 参数使用。原住旧 validate.go，
// 旧 Field 模型删除后这里成为唯一出处。
func formatNum(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
