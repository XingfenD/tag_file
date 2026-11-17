package schema

import (
	"strings"
	"testing"
)

// 本文件在包内部（package schema）测试：validateDescriptor 是未导出的
// 合法性检查，描述符直接构造、不经注册表。

func TestValidateDescriptor(t *testing.T) {
	cases := []struct {
		name   string
		d      InputBoxDescriptor
		ok     bool
		errMsg string // 非空时额外断言错误文案包含该片段
	}{
		// 规则 1：值类型与容器类型是封闭集合。
		{"未知值类型", InputBoxDescriptor{ValueType: "Float", Container: ContainerSingle}, false, ""},
		{"未知容器类型", InputBoxDescriptor{ValueType: ValueInt, Container: "Matrix"}, false, ""},

		// 规则 2：约束形状与值类型/容器类型匹配。
		{"Int 配 StringConstraint", InputBoxDescriptor{
			ValueType:  ValueInt,
			Container:  ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{}},
		}, false, "约束与值类型/容器类型不匹配"},
		{"Single 配 IntConstraintList", InputBoxDescriptor{
			ValueType:  ValueInt,
			Container:  ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{IntConstraintList: []*IntConstraintDescriptor{{}}},
		}, false, ""},
		{"Array 配单例 IntConstraint", InputBoxDescriptor{
			ValueType:  ValueInt,
			Container:  ContainerArray,
			Constraint: InputBoxConstraintDescriptor{IntConstraint: &IntConstraintDescriptor{}},
		}, false, ""},
		{"Array 空约束列表", InputBoxDescriptor{
			ValueType: ValueInt,
			Container: ContainerArray,
		}, false, ""},

		// 规则 3：DynamicListConstraint 仅 DynamicArray，项数界限自洽。
		{"DynamicArray 项数 Min 大于 Max", InputBoxDescriptor{
			ValueType:             ValueString,
			Container:             ContainerDynamicArray,
			DynamicListConstraint: &DynamicListConstraintDescriptor{MinItems: Ptr(5), MaxItems: Ptr(2)},
		}, false, ""},
		{"最小项数为负", InputBoxDescriptor{
			ValueType:             ValueString,
			Container:             ContainerDynamicArray,
			DynamicListConstraint: &DynamicListConstraintDescriptor{MinItems: Ptr(-1)},
		}, false, ""},
		{"Single 配 DynamicListConstraint", InputBoxDescriptor{
			ValueType:             ValueString,
			Container:             ContainerSingle,
			DynamicListConstraint: &DynamicListConstraintDescriptor{MinItems: Ptr(1)},
		}, false, ""},

		// 规则 4：IntConstraint 自洽。
		{"Shift 为负", InputBoxDescriptor{
			ValueType:  ValueInt,
			Container:  ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{IntConstraint: &IntConstraintDescriptor{Shift: -1}},
		}, false, ""},
		{"Min 大于 Max", InputBoxDescriptor{
			ValueType:  ValueInt,
			Container:  ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{IntConstraint: &IntConstraintDescriptor{Min: Ptr(9.0), Max: Ptr(1.0)}},
		}, false, ""},

		// 规则 5：StringConstraint 自洽。
		{"未知字符串格式", InputBoxDescriptor{
			ValueType:  ValueString,
			Container:  ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{Format: "yaml"}},
		}, false, ""},
		{"Single 配 Format regex", InputBoxDescriptor{
			ValueType:  ValueString,
			Container:  ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{Format: FormatRegex}},
		}, false, ""},
		{"DynamicArray 配 Format regex", InputBoxDescriptor{
			ValueType:  ValueString,
			Container:  ContainerDynamicArray,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{Format: FormatRegex}},
		}, true, ""},
		{"DynamicArray 配 Required", InputBoxDescriptor{
			ValueType:  ValueString,
			Container:  ContainerDynamicArray,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{Required: true}},
		}, false, ""},
		{"DynamicArray 配 Options", InputBoxDescriptor{
			ValueType:  ValueString,
			Container:  ContainerDynamicArray,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{Options: []Option{{Value: "a", Label: "A"}}}},
		}, false, ""},
		{"DynamicArray 配 Secret", InputBoxDescriptor{
			ValueType:  ValueString,
			Container:  ContainerDynamicArray,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{Secret: true}},
		}, false, ""},
		{"DynamicArray 配 Multiline", InputBoxDescriptor{
			ValueType:  ValueString,
			Container:  ContainerDynamicArray,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{Multiline: true}},
		}, false, ""},
		{"Options 配 Secret", InputBoxDescriptor{
			ValueType: ValueString,
			Container: ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{
				Options: []Option{{Value: "a", Label: "A"}},
				Secret:  true,
			}},
		}, false, ""},
		{"Options 配 Multiline", InputBoxDescriptor{
			ValueType: ValueString,
			Container: ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{
				Options:   []Option{{Value: "a", Label: "A"}},
				Multiline: true,
			}},
		}, false, ""},
		{"Options 值重复", InputBoxDescriptor{
			ValueType: ValueString,
			Container: ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{
				Options: []Option{{Value: "a", Label: "A"}, {Value: "a", Label: "又 A"}},
			}},
		}, false, ""},
		{"Pattern 非法正则", InputBoxDescriptor{
			ValueType:  ValueString,
			Container:  ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{Pattern: "("}},
		}, false, ""},

		// 合法声明。
		{"合法 Int 边界与 Shift", InputBoxDescriptor{
			ValueType: ValueInt,
			Container: ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{IntConstraint: &IntConstraintDescriptor{
				Min: Ptr(0.0), MinExclusive: true, Shift: 1,
			}},
		}, true, ""},
		{"合法 Secret", InputBoxDescriptor{
			ValueType:  ValueString,
			Container:  ContainerSingle,
			Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{Secret: true}},
		}, true, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateDescriptor(c.d)
			if c.ok {
				if err != nil {
					t.Fatalf("期望描述符合法，实际报错: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("期望描述符非法，实际放行")
			}
			if err.Error() == "" {
				t.Fatalf("错误文案不能为空")
			}
			if c.errMsg != "" && !strings.Contains(err.Error(), c.errMsg) {
				t.Fatalf("错误文案应包含 %q，实际为 %q", c.errMsg, err.Error())
			}
		})
	}
}

func TestIsSecret(t *testing.T) {
	secret := &InputBoxDescriptor{
		ValueType:  ValueString,
		Container:  ContainerSingle,
		Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{Secret: true}},
	}
	if !secret.IsSecret() {
		t.Fatalf("Single+String+Secret 应识别为 secret")
	}

	// 非法声明直接构造（不经注册表）：IsSecret 只认三重条件的合取，
	// 不依赖描述符是否合法。
	dynSecret := &InputBoxDescriptor{
		ValueType:  ValueString,
		Container:  ContainerDynamicArray,
		Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{Secret: true}},
	}
	if dynSecret.IsSecret() {
		t.Fatalf("DynamicArray+Secret（非法声明）不应识别为 secret")
	}

	intSecret := &InputBoxDescriptor{
		ValueType:  ValueInt,
		Container:  ContainerSingle,
		Constraint: InputBoxConstraintDescriptor{StringConstraint: &StringConstraintDescriptor{Secret: true}},
	}
	if intSecret.IsSecret() {
		t.Fatalf("Int+Secret 不应识别为 secret")
	}
}
