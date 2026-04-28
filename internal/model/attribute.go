package model

type Attribute struct {
	AttributeID   uint   `gorm:"primaryKey;column:attribute_id" json:"attribute_id"`
	TagID         uint   `gorm:"column:tag_id;not null" json:"tag_id"`
	AttributeName string `gorm:"column:attribute_name;size:100;not null" json:"attribute_name"`
	Description   string `gorm:"column:description;type:text" json:"description"`
	Tag           Tag    `gorm:"foreignKey:TagID;references:TagID" json:"tag"`
}

func (Attribute) TableName() string {
	return "attributes"
}

type AttributeValue struct {
	ValueID     uint      `gorm:"primaryKey;column:value_id" json:"value_id"`
	AttributeID uint      `gorm:"column:attribute_id;not null" json:"attribute_id"`
	ValueName   string    `gorm:"column:value_name;size:100;not null" json:"value_name"`
	Description string    `gorm:"column:description;type:text" json:"description"`
	Attribute   Attribute `gorm:"foreignKey:AttributeID;references:AttributeID" json:"attribute"`
}

func (AttributeValue) TableName() string {
	return "attribute_values"
}
