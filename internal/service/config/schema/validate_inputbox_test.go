package schema_test

import (
	"testing"

	"github.com/XingfenD/tag_file/internal/service/config/schema"
)

// TestValidate 迁移旧 validate_test.go 的全部语义到 InputBoxDescriptor
// 模型（旧文件已删除）。errMsg 非空时断言逐字相等；为空只断言
// 通过/失败——错误文案的逐字形态由 rules_test.go 的 Check 级测试钉死。
func TestValidate(t *testing.T) {
	cases := []struct {
		name   string
		d      schema.InputBoxDescriptor
		value  string
		ok     bool
		errMsg string
	}{
		// string
		{"string 空串合法", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle}, "", true, ""},
		{"string pattern 命中", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Pattern: `^[a-z]+$`}}}, "abc", true, ""},
		{"string pattern 不命中", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Pattern: `^[a-z]+$`}}}, "A1", false, ""},
		{"string 空串跳过 pattern", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Pattern: `^[a-z]+$`}}}, "", true, ""},
		{"required 空串拒绝", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Required: true}}}, "", false, "取值不能为空"},

		// url：细分文案与旧 validateURL 逐字一致。
		{"url 合法", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Format: schema.FormatURL}}}, "http://127.0.0.1:11434", true, ""},
		{"url 空串非法", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Format: schema.FormatURL}}}, "", false, "取值必须是 http 或 https 地址，不能为空"},
		{"url 无 scheme 非法", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Format: schema.FormatURL}}}, "127.0.0.1:11434", false, "取值必须是合法的 http 或 https 地址"},
		{"url ftp 非法", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Format: schema.FormatURL}}}, "ftp://x/y", false, "取值必须以 http:// 或 https:// 开头"},

		// int：闭区间边界。
		{"int 闭区间收 64", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{IntConstraint: &schema.IntConstraintDescriptor{Min: schema.Ptr(1.0), Max: schema.Ptr(64.0)}}}, "64", true, ""},
		{"int 越上界拒 65", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{IntConstraint: &schema.IntConstraintDescriptor{Min: schema.Ptr(1.0), Max: schema.Ptr(64.0)}}}, "65", false, "取值超出合法范围，应为 [1, 64]"},
		{"int 非数字拒", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerSingle}, "not-a-number", false, "取值必须是整数"},
		{"int 拒 NaN", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerSingle}, "NaN", false, "取值必须是整数"},
		{"int 拒 +Inf", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerSingle}, "+Inf", false, "取值必须是整数"},

		// 定点数（Shift=1：存储整数 1 = 显示 0.1）。边界在显示单位上判定，
		// 非整数 wire（"0.5"）按「取值必须是整数」拒绝。
		{"定点数存储 1 即 0.1 通过", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{IntConstraint: &schema.IntConstraintDescriptor{Min: schema.Ptr(0.0), MinExclusive: true, Shift: 1}}}, "1", true, ""},
		{"定点数拒边界 0", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{IntConstraint: &schema.IntConstraintDescriptor{Min: schema.Ptr(0.0), MinExclusive: true, Shift: 1}}}, "0", false, "取值超出合法范围，应为 (0, +∞)"},
		{"定点数拒非整数 wire", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{IntConstraint: &schema.IntConstraintDescriptor{Min: schema.Ptr(0.0), MinExclusive: true, Shift: 1}}}, "0.5", false, "取值必须是整数"},
		{"定点数拒负数", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{IntConstraint: &schema.IntConstraintDescriptor{Min: schema.Ptr(0.0), MinExclusive: true, Shift: 1}}}, "-1", false, "取值超出合法范围，应为 (0, +∞)"},

		// bool
		{"bool 只认 true", schema.InputBoxDescriptor{ValueType: schema.ValueBool, Container: schema.ContainerSingle}, "true", true, ""},
		{"bool 只认 false", schema.InputBoxDescriptor{ValueType: schema.ValueBool, Container: schema.ContainerSingle}, "false", true, ""},
		{"bool 拒大写 TRUE", schema.InputBoxDescriptor{ValueType: schema.ValueBool, Container: schema.ContainerSingle}, "TRUE", false, "取值只能是 true 或 false"},

		// oneof
		{"oneof 命中选项", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{
				Options: []schema.Option{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}},
			}}}, "b", true, ""},
		{"oneof 未命中", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{
				Options: []schema.Option{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}},
			}}}, "c", false, "取值必须是以下之一: a、b"},

		// DynamicArray + regex：项级失败拼「第 %d 项%s」。
		{"动态数组合法正则", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerDynamicArray,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Format: schema.FormatRegex}}}, `["^\\.","^x$"]`, true, ""},
		{"动态数组空数组合法", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerDynamicArray,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Format: schema.FormatRegex}}}, `[]`, true, ""},
		{"动态数组非 JSON 拒", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerDynamicArray,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Format: schema.FormatRegex}}}, `not json`, false, `取值必须是 JSON 字符串数组，例如 ["a","b"]`},
		{"动态数组非数组 JSON 拒", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerDynamicArray,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Format: schema.FormatRegex}}}, `{"a":1}`, false, `取值必须是 JSON 字符串数组，例如 ["a","b"]`},
		{"动态数组非法正则项拒", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerDynamicArray,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Format: schema.FormatRegex}}}, `["("]`, false, "第 1 项不是合法的正则表达式"},
		{"动态数组第二项非法正则", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerDynamicArray,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Format: schema.FormatRegex}}}, `["^x$","("]`, false, "第 2 项不是合法的正则表达式"},

		// DynamicArray + item_count：容器级项数界限。
		{"项数超上限拒", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerDynamicArray,
			DynamicListConstraint: &schema.DynamicListConstraintDescriptor{MaxItems: schema.Ptr(1)}}, `["a","b"]`, false, "项数不能多于 1"},
		{"项数合规通过", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerDynamicArray,
			DynamicListConstraint: &schema.DynamicListConstraintDescriptor{MaxItems: schema.Ptr(1)}}, `["a"]`, true, ""},
		{"项数低于下限拒", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerDynamicArray,
			DynamicListConstraint: &schema.DynamicListConstraintDescriptor{MinItems: schema.Ptr(1)}}, `[]`, false, "项数不能少于 1"},

		// Array 定长：fixed_arity + 逐位约束。
		{"定长数组长度符合通过", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerArray,
			Constraint: schema.InputBoxConstraintDescriptor{IntConstraintList: []*schema.IntConstraintDescriptor{{}, {}}}}, `["1","2"]`, true, ""},
		{"定长数组长度不足拒", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerArray,
			Constraint: schema.InputBoxConstraintDescriptor{IntConstraintList: []*schema.IntConstraintDescriptor{{}, {}}}}, `["1"]`, false, "需要恰好 2 项"},
		{"定长逐位约束通过", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerArray,
			Constraint: schema.InputBoxConstraintDescriptor{IntConstraintList: []*schema.IntConstraintDescriptor{
				{Min: schema.Ptr(1.0)}, {Max: schema.Ptr(9.0)},
			}}}, `["5","5"]`, true, ""},
		{"定长逐位约束位 0 拒", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerArray,
			Constraint: schema.InputBoxConstraintDescriptor{IntConstraintList: []*schema.IntConstraintDescriptor{
				{Min: schema.Ptr(1.0)}, {Max: schema.Ptr(9.0)},
			}}}, `["0","5"]`, false, "第 1 项取值超出合法范围，应为 [1, +∞)"},
		{"定长逐位约束位 1 拒", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerArray,
			Constraint: schema.InputBoxConstraintDescriptor{IntConstraintList: []*schema.IntConstraintDescriptor{
				{Min: schema.Ptr(1.0)}, {Max: schema.Ptr(9.0)},
			}}}, `["1","10"]`, false, "第 2 项取值超出合法范围，应为 (-∞, 9]"},

		// secret / render+behavior：不产生校验语义。
		{"secret 任意串合法", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Secret: true}}}, "sk-xxx", true, ""},
		{"secret 空串合法", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Secret: true}}}, "", true, ""},
		{"multiline 加 secret 任意串通过", schema.InputBoxDescriptor{ValueType: schema.ValueString, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{Multiline: true, Secret: true}}}, "任意\n多行内容", true, ""},

		// 坏描述符：Validate 先于取值报错，与取值无关。
		{"坏描述符合法取值仍报错", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{}}}, "1", false, "约束与值类型/容器类型不匹配"},
		{"坏描述符任意取值报错", schema.InputBoxDescriptor{ValueType: schema.ValueInt, Container: schema.ContainerSingle,
			Constraint: schema.InputBoxConstraintDescriptor{StringConstraint: &schema.StringConstraintDescriptor{}}}, "whatever", false, "约束与值类型/容器类型不匹配"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := schema.Validate(c.d, c.value)
			if c.ok {
				if err != nil {
					t.Fatalf("期望校验通过，实际报错: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("期望校验失败，实际通过")
			}
			if err.Error() == "" {
				t.Fatalf("校验错误文案不能为空")
			}
			if c.errMsg != "" && err.Error() != c.errMsg {
				t.Fatalf("错误文案应为 %q，实际为 %q", c.errMsg, err.Error())
			}
		})
	}
}
