package model

// TagSchema 是打标场景定义：独立实体，字段显式绑定标签树里的标签，
// 值经物化落进 TagGroupTag（默认组），schema 本身不存任何文件值。
type TagSchema struct {
	SchemaID    uint             `gorm:"primaryKey;column:schema_id" json:"schema_id"`
	SchemaName  string           `gorm:"column:schema_name;size:100;not null;uniqueIndex" json:"schema_name"`
	Description string           `gorm:"column:description;type:text" json:"description"`
	SortOrder   int              `gorm:"column:sort_order;default:0" json:"sort_order"`
	Fields      []TagSchemaField `gorm:"foreignKey:SchemaID" json:"fields,omitempty"`
}

func (TagSchema) TableName() string {
	return "tag_schemas"
}

// TagSchemaField 是场景里的一个字段。FieldType ∈ single|bool|multi；
// Widget 仅对 single 有意义：''|select|stars。BindTagID 全局唯一占用
// （服务层校验），multi 绑定父标签、候选取其子标签。
type TagSchemaField struct {
	FieldID   uint   `gorm:"primaryKey;column:field_id" json:"field_id"`
	SchemaID  uint   `gorm:"column:schema_id;not null;index" json:"schema_id"`
	Label     string `gorm:"column:label;size:100;not null" json:"label"`
	FieldType string `gorm:"column:field_type;size:20;not null" json:"field_type"`
	Widget    string `gorm:"column:widget;size:20;default:''" json:"widget"`
	BindTagID uint   `gorm:"column:bind_tag_id;not null;index" json:"bind_tag_id"`
	SortOrder int    `gorm:"column:sort_order;default:0" json:"sort_order"`
}

func (TagSchemaField) TableName() string {
	return "tag_schema_fields"
}
