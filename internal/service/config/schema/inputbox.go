package schema

import (
	"errors"
	"fmt"
	"regexp"
)

// ValueType 是输入框取值的基本类型：存储形态的最小维度，约束与控件都按它匹配。
// 不设 Float 值类型：小数用 Int + Shift 定点表示，存储与传输一律整数。
type ValueType string

const (
	ValueInt    ValueType = "Int"
	ValueString ValueType = "String"
	ValueBool   ValueType = "Bool"
)

// ContainerType 是输入框的容器形态。
type ContainerType string

const (
	ContainerSingle       ContainerType = "Single"
	ContainerArray        ContainerType = "Array"        // 定长：长度 = 约束列表长度
	ContainerDynamicArray ContainerType = "DynamicArray" // 变长：单值约束作用于每个元素
)

// ConfigBlockDescriptor 是设置页的一个配置块：块直接拥有输入框描述符列表，
// 注册以块为单位。
type ConfigBlockDescriptor struct {
	ID                     string
	Title                  string
	Order                  int
	InputBoxDescriptorList []*InputBoxDescriptor
}

// InputBoxDescriptor 是一个配置键的元数据。ValueType × ContainerType × Constraint
// 三个维度正交：控件由维度推导，约束按类型匹配（validateDescriptor 强制）。
// 默认值与边界常量住在消费方，这里只做声明。
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
	// DynamicListConstraint 仅 DynamicArray 可设（validateDescriptor 强制）。
	DynamicListConstraint *DynamicListConstraintDescriptor
}

// InputBoxConstraintDescriptor 汇集按 ValueType × ContainerType 匹配的约束描述符。
// 形状由 validator 强制：Single/DynamicArray → 对应单例（nil = 无附加约束）、
// 其余单例 nil、三个列表为空；Array → 对应列表非空（定长 = 列表长度）、
// 其余列表空、三个单例 nil。
type InputBoxConstraintDescriptor struct {
	IntConstraint        *IntConstraintDescriptor
	StringConstraint     *StringConstraintDescriptor
	BoolConstraint       *BoolConstraintDescriptor
	IntConstraintList    []*IntConstraintDescriptor
	StringConstraintList []*StringConstraintDescriptor
	BoolConstraintList   []*BoolConstraintDescriptor
}

// IntConstraintDescriptor 是 Int 值的边界约束。Min/Max 用显示单位
// （= 存储整数 / 10^Shift）。
type IntConstraintDescriptor struct {
	Min, Max                   *float64
	MinExclusive, MaxExclusive bool
	Shift                      int // 小数位偏移，≥0；0 = 纯整数
}

// StringFormat 是 string 值的语法格式约束。
type StringFormat string

const (
	FormatNone  StringFormat = ""
	FormatURL   StringFormat = "url"
	FormatRegex StringFormat = "regex" // 仅 Array/DynamicArray 的项约束可用
)

// StringConstraintDescriptor 是 string 值的约束。
// Required/Options/Secret/Multiline 仅 Single 可声明（validateDescriptor 强制）。
type StringConstraintDescriptor struct {
	Required  bool // 空串非法（仅 Single）
	Format    StringFormat
	Pattern   string   // 只约束非空值（沿用现状）
	Options   []Option // 非空 → 枚举（仅 Single）
	Secret    bool     // 掩码 + 留空不覆盖（仅 Single）
	Multiline bool     // 分解为渲染指令：textarea（仅 Single）
}

// BoolConstraintDescriptor 校验内建于 ValueType，占位保持联合一致。
type BoolConstraintDescriptor struct{}

// DynamicListConstraintDescriptor 是 DynamicArray 的容器级项数约束。
type DynamicListConstraintDescriptor struct {
	MinItems *int
	MaxItems *int
}

// errConstraintMismatch 是约束形状与值类型/容器类型不匹配的统一错误。
var errConstraintMismatch = errors.New("约束与值类型/容器类型不匹配")

// IsSecret 报告该输入框是否为 secret：掩码与留空不覆盖行为由这一声明驱动。
// 只认三重条件的合取，不依赖描述符整体是否合法。
func (d *InputBoxDescriptor) IsSecret() bool {
	return d.Container == ContainerSingle &&
		d.ValueType == ValueString &&
		d.Constraint.StringConstraint != nil &&
		d.Constraint.StringConstraint.Secret
}

// validateDescriptor 检查描述符自身是否合法（与取值无关，先于值校验）。
// RegisterBlock 用它做注册自检，非法声明在启动期 panic，绝不留到用户改配置时
// 才以「取值非法」的形式出现。
func validateDescriptor(d InputBoxDescriptor) error {
	// 规则 1：值类型与容器类型都是封闭集合。
	switch d.ValueType {
	case ValueInt, ValueString, ValueBool:
	default:
		return fmt.Errorf("未知的值类型 '%s'", d.ValueType)
	}
	switch d.Container {
	case ContainerSingle, ContainerArray, ContainerDynamicArray:
	default:
		return fmt.Errorf("未知的容器类型 '%s'", d.Container)
	}

	// 规则 2：约束形状与 ValueType × ContainerType 匹配，并收集待深检的约束。
	var intConstraints []*IntConstraintDescriptor
	var stringConstraints []*StringConstraintDescriptor
	if d.Container == ContainerArray {
		if d.Constraint.IntConstraint != nil || d.Constraint.StringConstraint != nil || d.Constraint.BoolConstraint != nil {
			return errConstraintMismatch
		}
		switch d.ValueType {
		case ValueInt:
			intConstraints = d.Constraint.IntConstraintList
			if len(intConstraints) == 0 || len(d.Constraint.StringConstraintList) > 0 || len(d.Constraint.BoolConstraintList) > 0 {
				return errConstraintMismatch
			}
		case ValueString:
			stringConstraints = d.Constraint.StringConstraintList
			if len(stringConstraints) == 0 || len(d.Constraint.IntConstraintList) > 0 || len(d.Constraint.BoolConstraintList) > 0 {
				return errConstraintMismatch
			}
		case ValueBool:
			if len(d.Constraint.BoolConstraintList) == 0 || len(d.Constraint.IntConstraintList) > 0 || len(d.Constraint.StringConstraintList) > 0 {
				return errConstraintMismatch
			}
		}
	} else {
		// Single/DynamicArray：单值约束（DynamicArray 作用于每个元素）。
		if len(d.Constraint.IntConstraintList) > 0 || len(d.Constraint.StringConstraintList) > 0 || len(d.Constraint.BoolConstraintList) > 0 {
			return errConstraintMismatch
		}
		switch d.ValueType {
		case ValueInt:
			if d.Constraint.StringConstraint != nil || d.Constraint.BoolConstraint != nil {
				return errConstraintMismatch
			}
			if d.Constraint.IntConstraint != nil {
				intConstraints = []*IntConstraintDescriptor{d.Constraint.IntConstraint}
			}
		case ValueString:
			if d.Constraint.IntConstraint != nil || d.Constraint.BoolConstraint != nil {
				return errConstraintMismatch
			}
			if d.Constraint.StringConstraint != nil {
				stringConstraints = []*StringConstraintDescriptor{d.Constraint.StringConstraint}
			}
		case ValueBool:
			if d.Constraint.IntConstraint != nil || d.Constraint.StringConstraint != nil {
				return errConstraintMismatch
			}
		}
	}

	// 规则 3：DynamicListConstraint 仅 DynamicArray 可设，项数界限自洽。
	if err := validateDynamicListConstraint(d); err != nil {
		return err
	}

	// 规则 4：Int 约束自洽（Shift ≥ 0；Min/Max 同设时 Min ≤ Max）。
	for _, c := range intConstraints {
		if err := validateIntConstraint(c); err != nil {
			return err
		}
	}

	// 规则 5：String 约束自洽（格式、Single 专属字段、枚举去重、Pattern 预编译）。
	for _, c := range stringConstraints {
		if err := validateStringConstraint(c, d.Container); err != nil {
			return err
		}
	}
	return nil
}

// validateDynamicListConstraint 检查容器级项数约束的归属与自洽。
func validateDynamicListConstraint(d InputBoxDescriptor) error {
	c := d.DynamicListConstraint
	if c == nil {
		return nil
	}
	if d.Container != ContainerDynamicArray {
		return errors.New("DynamicListConstraint 仅 DynamicArray 可用")
	}
	if c.MinItems != nil && *c.MinItems < 0 {
		return fmt.Errorf("最小项数不能为负: %d", *c.MinItems)
	}
	if c.MaxItems != nil && *c.MaxItems < 0 {
		return fmt.Errorf("最大项数不能为负: %d", *c.MaxItems)
	}
	if c.MinItems != nil && c.MaxItems != nil && *c.MinItems > *c.MaxItems {
		return fmt.Errorf("最小项数 %d 大于最大项数 %d", *c.MinItems, *c.MaxItems)
	}
	return nil
}

// validateIntConstraint 检查 Int 边界约束的自洽。nil 表示该位置无附加约束，放行。
func validateIntConstraint(c *IntConstraintDescriptor) error {
	if c == nil {
		return nil
	}
	if c.Shift < 0 {
		return fmt.Errorf("小数位偏移 Shift 不能为负: %d", c.Shift)
	}
	if c.Min != nil && c.Max != nil && *c.Min > *c.Max {
		return fmt.Errorf("取值下界 %v 大于上界 %v", *c.Min, *c.Max)
	}
	return nil
}

// validateStringConstraint 检查 string 约束的自洽。container 参与判定：
// regex 格式仅容器项约束可用；Required/Options/Secret/Multiline 仅 Single 可声明。
// nil 表示该位置无附加约束，放行。
func validateStringConstraint(c *StringConstraintDescriptor, container ContainerType) error {
	if c == nil {
		return nil
	}
	switch c.Format {
	case FormatNone, FormatURL, FormatRegex:
	default:
		return fmt.Errorf("未知的字符串格式 '%s'", c.Format)
	}
	if c.Format == FormatRegex && container == ContainerSingle {
		return errors.New("regex 格式仅 Array/DynamicArray 的项约束可用")
	}
	if container != ContainerSingle && (c.Required || len(c.Options) > 0 || c.Secret || c.Multiline) {
		return errors.New("Required/Options/Secret/Multiline 仅 Single 可用")
	}
	if len(c.Options) > 0 && (c.Secret || c.Multiline) {
		return errors.New("枚举约束不能与 Secret/Multiline 同时使用")
	}
	if len(c.Options) > 1 {
		seen := make(map[string]bool, len(c.Options))
		for _, o := range c.Options {
			if seen[o.Value] {
				return fmt.Errorf("枚举选项值 %q 重复", o.Value)
			}
			seen[o.Value] = true
		}
	}
	// Pattern 在描述符检查期即编译：非法模式在启动期失败，而不是等到用户提交值。
	if c.Pattern != "" {
		if _, err := regexp.Compile(c.Pattern); err != nil {
			return fmt.Errorf("匹配模式无效: %v", err)
		}
	}
	return nil
}
