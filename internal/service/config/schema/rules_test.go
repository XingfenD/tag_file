package schema

import (
	"reflect"
	"testing"
)

// 本文件在包内部（package schema）测试：ruleDefs、renderMessage 与各 Check
// 是未导出的协议实现，需直接构造参数表驱动。

// findRule 返回规则组中指定 kind 的第一条指令；缺失即失败。
func findRule(t *testing.T, rules []Rule, kind RuleKind) Rule {
	t.Helper()
	for _, r := range rules {
		if r.Kind == kind {
			return r
		}
	}
	t.Fatalf("规则组中缺少 kind %q：%+v", kind, rules)
	return Rule{}
}

// kindsOf 按顺序取出指令组的 kind 列表，用于断言分解的稳定输出顺序。
func kindsOf(rules []Rule) []string {
	kinds := make([]string, 0, len(rules))
	for _, r := range rules {
		kinds = append(kinds, string(r.Kind))
	}
	return kinds
}

func TestRulesOfDecomposition(t *testing.T) {
	t.Run("Int 单值带排他下界与 Shift", func(t *testing.T) {
		rs := RulesOf(InputBoxDescriptor{
			ValueType: ValueInt,
			Container: ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{IntConstraint: &IntConstraintDescriptor{
				Min: Ptr(0.0), MinExclusive: true, Shift: 1,
			}},
		})
		if got := kindsOf(rs.Rules); !reflect.DeepEqual(got, []string{"int", "int_range"}) {
			t.Fatalf("Rules 应为 [int int_range]，实际为 %v", got)
		}
		r := findRule(t, rs.Rules, RuleIntRange)
		if r.Params["shift"] != "1" || r.Params["min"] != "0" || r.Params["min_exclusive"] != "true" {
			t.Fatalf("int_range 参数不符：%+v", r.Params)
		}
		if _, hasMax := r.Params["max"]; hasMax {
			t.Fatalf("未声明上界不应输出 max 参数：%+v", r.Params)
		}
		if r.Role != RoleValidate {
			t.Fatalf("int_range 角色应为 validate，实际为 %q", r.Role)
		}
		if len(rs.PositionalRules) != 0 || len(rs.ContainerRules) != 0 {
			t.Fatalf("Single 不应有逐位/容器级指令：%+v", rs)
		}
	})

	t.Run("String 单值 Required 加 url", func(t *testing.T) {
		rs := RulesOf(InputBoxDescriptor{
			ValueType: ValueString,
			Container: ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{
				Required: true, Format: FormatURL,
			}},
		})
		if got := kindsOf(rs.Rules); !reflect.DeepEqual(got, []string{"required", "url"}) {
			t.Fatalf("Rules 应为 [required url]，实际为 %v", got)
		}
	})

	t.Run("String 单值枚举分解 oneof", func(t *testing.T) {
		rs := RulesOf(InputBoxDescriptor{
			ValueType: ValueString,
			Container: ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{
				Options: []Option{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}},
			}},
		})
		if got := kindsOf(rs.Rules); !reflect.DeepEqual(got, []string{"oneof"}) {
			t.Fatalf("Rules 应为 [oneof]，实际为 %v", got)
		}
		r := findRule(t, rs.Rules, RuleOneOf)
		if r.Widget != "select" {
			t.Fatalf("oneof 渲染效果应为 select，实际为 %q", r.Widget)
		}
		want := `[{"value":"a","label":"A"},{"value":"b","label":"B"}]`
		if r.Params["options"] != want {
			t.Fatalf("options 参数应为 %s，实际为 %s", want, r.Params["options"])
		}
	})

	t.Run("String 单值 Multiline 分解渲染指令", func(t *testing.T) {
		rs := RulesOf(InputBoxDescriptor{
			ValueType: ValueString,
			Container: ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{
				Multiline: true,
			}},
		})
		if got := kindsOf(rs.Rules); !reflect.DeepEqual(got, []string{"multiline"}) {
			t.Fatalf("Rules 应为 [multiline]，实际为 %v", got)
		}
		r := findRule(t, rs.Rules, RuleMultiline)
		if r.Role != RoleRender {
			t.Fatalf("multiline 角色应为 render，实际为 %q", r.Role)
		}
		if r.Widget != "textarea" {
			t.Fatalf("multiline 渲染效果应为 textarea，实际为 %q", r.Widget)
		}
		if ruleDefs[RuleMultiline].Check != nil {
			t.Fatalf("multiline 是渲染指令，不应带 Check 语义")
		}
	})

	t.Run("String 单值 Secret 分解行为指令", func(t *testing.T) {
		rs := RulesOf(InputBoxDescriptor{
			ValueType: ValueString,
			Container: ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{
				Secret: true,
			}},
		})
		if got := kindsOf(rs.Rules); !reflect.DeepEqual(got, []string{"secret"}) {
			t.Fatalf("Rules 应为 [secret]，实际为 %v", got)
		}
		r := findRule(t, rs.Rules, RuleSecret)
		if r.Role != RoleBehavior {
			t.Fatalf("secret 角色应为 behavior，实际为 %q", r.Role)
		}
		if r.Widget != "password" {
			t.Fatalf("secret 渲染效果应为 password，实际为 %q", r.Widget)
		}
		if r.Hint != "已设置，留空不修改" {
			t.Fatalf("secret 用法提示应为「已设置，留空不修改」，实际为 %q", r.Hint)
		}
		if ruleDefs[RuleSecret].Check != nil {
			t.Fatalf("secret 是行为指令，不应带 Check 语义")
		}
	})

	t.Run("带 Widget 指令按 secret 优先于 multiline 输出", func(t *testing.T) {
		rs := RulesOf(InputBoxDescriptor{
			ValueType: ValueString,
			Container: ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{
				Secret: true, Multiline: true,
			}},
		})
		if got := kindsOf(rs.Rules); !reflect.DeepEqual(got, []string{"secret", "multiline"}) {
			t.Fatalf("Rules 应为 [secret multiline]（secret 优先），实际为 %v", got)
		}
	})

	t.Run("DynamicArray 项级 regex 加容器级 item_count", func(t *testing.T) {
		rs := RulesOf(InputBoxDescriptor{
			ValueType:             ValueString,
			Container:             ContainerDynamicArray,
			DynamicListConstraint: &DynamicListConstraintDescriptor{MaxItems: Ptr(5)},
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{
				Format: FormatRegex,
			}},
		})
		if got := kindsOf(rs.Rules); !reflect.DeepEqual(got, []string{"regex"}) {
			t.Fatalf("Rules 应为 [regex]，实际为 %v", got)
		}
		if got := kindsOf(rs.ContainerRules); !reflect.DeepEqual(got, []string{"item_count"}) {
			t.Fatalf("ContainerRules 应为 [item_count]，实际为 %v", got)
		}
		r := findRule(t, rs.ContainerRules, RuleItemCount)
		if r.Params["max_items"] != "5" {
			t.Fatalf("max_items 参数应为 5，实际为 %+v", r.Params)
		}
		if _, hasMin := r.Params["min_items"]; hasMin {
			t.Fatalf("未声明下界不应输出 min_items 参数：%+v", r.Params)
		}
	})

	t.Run("定长数组分解逐位指令与 fixed_arity", func(t *testing.T) {
		rs := RulesOf(InputBoxDescriptor{
			ValueType: ValueInt,
			Container: ContainerArray,
			Constraint: InputBoxConstraintDescriptor{IntConstraintList: []*IntConstraintDescriptor{
				{Min: Ptr(1.0)}, nil, // nil = 该位置无附加约束
			}},
		})
		if len(rs.Rules) != 0 {
			t.Fatalf("Array 的值级指令应为空（逐位指令住在 PositionalRules）：%+v", rs.Rules)
		}
		if len(rs.PositionalRules) != 2 {
			t.Fatalf("PositionalRules 长度应为 2，实际为 %d", len(rs.PositionalRules))
		}
		if got := kindsOf(rs.PositionalRules[0]); !reflect.DeepEqual(got, []string{"int", "int_range"}) {
			t.Fatalf("位 0 指令应为 [int int_range]，实际为 %v", got)
		}
		if got := kindsOf(rs.PositionalRules[1]); !reflect.DeepEqual(got, []string{"int"}) {
			t.Fatalf("位 1 无附加约束应只有 [int]，实际为 %v", got)
		}
		if got := kindsOf(rs.ContainerRules); !reflect.DeepEqual(got, []string{"fixed_arity"}) {
			t.Fatalf("ContainerRules 应为 [fixed_arity]，实际为 %v", got)
		}
		r := findRule(t, rs.ContainerRules, RuleFixedArity)
		if r.Params["n"] != "2" {
			t.Fatalf("n 参数应为 2，实际为 %+v", r.Params)
		}
	})

	t.Run("空约束只有值类型内建指令", func(t *testing.T) {
		for _, c := range []struct {
			name string
			d    InputBoxDescriptor
			want []string
		}{
			{"Int", InputBoxDescriptor{ValueType: ValueInt, Container: ContainerSingle}, []string{"int"}},
			{"Bool", InputBoxDescriptor{ValueType: ValueBool, Container: ContainerSingle}, []string{"bool"}},
			{"String 无内建", InputBoxDescriptor{ValueType: ValueString, Container: ContainerSingle}, []string{}},
		} {
			t.Run(c.name, func(t *testing.T) {
				rs := RulesOf(c.d)
				if got := kindsOf(rs.Rules); !reflect.DeepEqual(got, c.want) {
					t.Fatalf("Rules 应为 %v，实际为 %v", c.want, got)
				}
			})
		}
	})
}

func TestRuleMessageRendering(t *testing.T) {
	cases := []struct {
		name   string
		def    ruleDef
		params map[string]string
		want   string
	}{
		{"int_range 闭区间", ruleDefs[RuleIntRange], map[string]string{"min": "1", "max": "64"}, "取值超出合法范围，应为 [1, 64]"},
		{"int_range 排他下界", ruleDefs[RuleIntRange], map[string]string{"min": "0", "min_exclusive": "true"}, "取值超出合法范围，应为 (0, +∞)"},
		{"int_range 仅上界", ruleDefs[RuleIntRange], map[string]string{"max": "64"}, "取值超出合法范围，应为 (-∞, 64]"},
		{"oneof 候选顿号连接", ruleDefs[RuleOneOf], map[string]string{"options": `[{"value":"a","label":"A"},{"value":"b","label":"B"}]`}, "取值必须是以下之一: a、b"},
		{"pattern 模式串", ruleDefs[RulePattern], map[string]string{"pattern": "^x$"}, "取值不符合匹配模式 ^x$"},
		{"regex 静态文案", ruleDefs[RuleRegex], nil, "不是合法的正则表达式"},
		{"fixed_arity 项数", ruleDefs[RuleFixedArity], map[string]string{"n": "2"}, "需要恰好 2 项"},
		{"render 无失败文案", ruleDefs[RuleMultiline], nil, ""},
		{"behavior 无失败文案", ruleDefs[RuleSecret], nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := renderMessage(c.def, c.params); got != c.want {
				t.Fatalf("渲染文案应为 %q，实际为 %q", c.want, got)
			}
		})
	}
}

func TestRuleChecks(t *testing.T) {
	cases := []struct {
		name   string
		check  func(map[string]string, string) error
		params map[string]string
		value  string
		ok     bool
		errMsg string
	}{
		// url：细分文案沿用旧 validateURL 逐字形态。
		{"url 合法", checkURL, nil, "http://127.0.0.1:11434", true, ""},
		{"url 空串", checkURL, nil, "", false, "取值必须是 http 或 https 地址，不能为空"},
		{"url 解析失败", checkURL, nil, "127.0.0.1:11434", false, "取值必须是合法的 http 或 https 地址"},
		{"url ftp", checkURL, nil, "ftp://x/y", false, "取值必须以 http:// 或 https:// 开头"},
		{"url 无主机名", checkURL, nil, "http://", false, "取值必须包含主机名"},
		// regex
		{"regex 合法", checkRegex, nil, "^x$", true, ""},
		{"regex 非法", checkRegex, nil, "(", false, "不是合法的正则表达式"},
		// int
		{"int 合法", checkInt, nil, "42", true, ""},
		{"int 非整数", checkInt, nil, "1.5", false, "取值必须是整数"},
		// int_range：边界在显示单位上判定。
		{"int_range 收边界值", checkIntRange, map[string]string{"min": "1", "max": "64"}, "64", true, ""},
		{"int_range 越上界", checkIntRange, map[string]string{"min": "1", "max": "64"}, "65", false, "取值超出合法范围，应为 [1, 64]"},
		{"int_range 排他端拒边界值", checkIntRange, map[string]string{"min": "0", "min_exclusive": "true", "shift": "1"}, "0", false, "取值超出合法范围，应为 (0, +∞)"},
		{"int_range shift 换算收 1", checkIntRange, map[string]string{"min": "0", "min_exclusive": "true", "shift": "1"}, "1", true, ""},
		{"int_range 拒非整数 wire", checkIntRange, map[string]string{"shift": "1"}, "0.5", false, "取值必须是整数"},
		// oneof
		{"oneof 命中", checkOneOf, map[string]string{"options": `[{"value":"a"},{"value":"b"}]`}, "a", true, ""},
		{"oneof 未命中", checkOneOf, map[string]string{"options": `[{"value":"a"},{"value":"b"}]`}, "c", false, "取值必须是以下之一: a、b"},
		// required
		{"required 非空通过", checkRequired, nil, "x", true, ""},
		{"required 空串错", checkRequired, nil, "", false, "取值不能为空"},
		// bool
		{"bool 合法", checkBool, nil, "true", true, ""},
		{"bool 大写拒绝", checkBool, nil, "TRUE", false, "取值只能是 true 或 false"},
		// pattern：只约束非空值。
		{"pattern 命中", checkPattern, map[string]string{"pattern": "^x$"}, "x", true, ""},
		{"pattern 不命中", checkPattern, map[string]string{"pattern": "^x$"}, "y", false, "取值不符合匹配模式 ^x$"},
		{"pattern 空串跳过", checkPattern, map[string]string{"pattern": "^x$"}, "", true, ""},
		// item_count
		{"item_count 合规", checkItemCount, map[string]string{"max_items": "1"}, `["a"]`, true, ""},
		{"item_count 超上限", checkItemCount, map[string]string{"max_items": "1"}, `["a","b"]`, false, "项数不能多于 1"},
		{"item_count 低于下限", checkItemCount, map[string]string{"min_items": "1"}, `[]`, false, "项数不能少于 1"},
		// fixed_arity
		{"fixed_arity 长度符合", checkFixedArity, map[string]string{"n": "2"}, `["1","2"]`, true, ""},
		{"fixed_arity 长度不足", checkFixedArity, map[string]string{"n": "2"}, `["1"]`, false, "需要恰好 2 项"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.check(c.params, c.value)
			if c.ok {
				if err != nil {
					t.Fatalf("期望校验通过，实际报错: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("期望校验失败，实际通过")
			}
			if err.Error() != c.errMsg {
				t.Fatalf("错误文案应为 %q，实际为 %q", c.errMsg, err.Error())
			}
		})
	}
}

// TestCheckRuleKindsKnown 覆盖 spec §5.1 规则 6 的防御性检查：类型化分解下
// 不可达，但「新增原语忘写注册表」时引擎绝不静默放行。
func TestCheckRuleKindsKnown(t *testing.T) {
	t.Run("值级未知 kind", func(t *testing.T) {
		err := checkRuleKindsKnown(RuleSet{Rules: []Rule{{Kind: RuleInt}, {Kind: "bogus"}}})
		if err == nil || err.Error() != "未知的校验规则 'bogus'" {
			t.Fatalf("应报未知校验规则，实际为 %v", err)
		}
	})
	t.Run("逐位未知 kind", func(t *testing.T) {
		err := checkRuleKindsKnown(RuleSet{PositionalRules: [][]Rule{{{Kind: "bogus"}}}})
		if err == nil || err.Error() != "未知的校验规则 'bogus'" {
			t.Fatalf("应报未知校验规则，实际为 %v", err)
		}
	})
	t.Run("容器级未知 kind", func(t *testing.T) {
		err := checkRuleKindsKnown(RuleSet{ContainerRules: []Rule{{Kind: "bogus"}}})
		if err == nil || err.Error() != "未知的校验规则 'bogus'" {
			t.Fatalf("应报未知校验规则，实际为 %v", err)
		}
	})
	t.Run("已知 kind 放行", func(t *testing.T) {
		rs := RulesOf(InputBoxDescriptor{
			ValueType: ValueInt,
			Container: ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{IntConstraint: &IntConstraintDescriptor{
				Min: Ptr(0.0), MinExclusive: true, Shift: 1,
			}},
		})
		if err := checkRuleKindsKnown(rs); err != nil {
			t.Fatalf("合法分解不应报错: %v", err)
		}
	})
}
