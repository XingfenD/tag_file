package model

type TagValue struct {
	ValueID uint   `gorm:"primaryKey;column:value_id" json:"value_id"`
	TagID   uint   `gorm:"column:tag_id;not null;uniqueIndex:idx_tag_value" json:"tag_id"`
	Value   string `gorm:"column:value;size:200;not null;uniqueIndex:idx_tag_value" json:"value"`
	Tag     Tag    `gorm:"foreignKey:TagID;references:TagID" json:"tag"`
}

func (TagValue) TableName() string {
	return "tag_values"
}
