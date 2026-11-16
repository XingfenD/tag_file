package schema

// FieldType 是配置字段类型的封闭集合。新增类型必须先讨论，不允许给单个键开特例。
type FieldType string

const (
	TypeString   FieldType = "string"
	TypeText     FieldType = "text"
	TypeURL      FieldType = "url"
	TypeInt      FieldType = "int"
	TypeFloat    FieldType = "float"
	TypeDuration FieldType = "duration"
	TypeBool     FieldType = "bool"
	TypeEnum     FieldType = "enum"
	TypeList     FieldType = "list"
	TypeSecret   FieldType = "secret"
	TypeRegex    FieldType = "regex" // 仅作 list 的 ItemType
)

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

// Field 是一个配置键的元数据。默认值与边界常量住在消费方，这里只做声明。
type Field struct {
	Key          string
	Group        string
	Label        string
	Description  string
	Type         FieldType
	Default      string   // 存储形态
	Min          *float64 // int/float/duration
	Max          *float64
	MinExclusive bool
	MaxExclusive bool
	Pattern      string   // string
	Options      []Option // enum
	ItemType     FieldType
	Unit         string
	ShowIf       *Condition
}

// Group 是设置页的一个分组。Order 决定展示顺序。
type Group struct {
	ID    string
	Title string
	Order int
}

// Ptr 返回 v 的指针，便于声明可选的数值边界。
func Ptr(v float64) *float64 {
	return &v
}
