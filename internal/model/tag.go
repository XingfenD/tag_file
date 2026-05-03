package model

type Tag struct {
	TagID       uint        `gorm:"primaryKey;column:tag_id" json:"tag_id"`
	CategoryID  uint        `gorm:"column:category_id;not null" json:"category_id"`
	ParentID    *uint       `gorm:"column:parent_id" json:"parent_id,omitempty"`
	TagName     string      `gorm:"column:tag_name;size:100;not null" json:"tag_name"`
	Description string      `gorm:"column:description;type:text" json:"description"`
	Category    TagCategory `gorm:"foreignKey:CategoryID;references:CategoryID" json:"category"`
	Parent      *Tag        `gorm:"foreignKey:ParentID;references:TagID" json:"parent,omitempty"`
	Children    []Tag       `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (Tag) TableName() string {
	return "tags"
}

type TagCategory struct {
	CategoryID   uint   `gorm:"primaryKey;column:category_id" json:"category_id"`
	CategoryName string `gorm:"column:category_name;size:100;not null" json:"category_name"`
	Description  string `gorm:"column:description;type:text" json:"description"`
}

func (TagCategory) TableName() string {
	return "tag_categories"
}
