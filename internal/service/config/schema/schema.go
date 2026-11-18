package schema

// Option 是 enum 的一个候选项，Value 为存储值，Label 为显示名。
type Option struct {
	Value string
	Label string
}

// Condition 描述字段的条件显隐：当 Key 的值等于 Equals 时显示。
type Condition struct {
	Key    string
	Equals string
}

// Ptr 返回 v 的指针，便于声明可选的边界。泛型化后由实参类型决定指针类型：
// 浮点边界写浮点字面量 Ptr(0.0)，整型写 Ptr(1)。
func Ptr[T any](v T) *T {
	return &v
}
