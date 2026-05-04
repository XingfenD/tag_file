package model

import "time"

type TagGroup struct {
	GroupID     uint      `gorm:"primaryKey;column:group_id" json:"group_id"`
	FileID      uint      `gorm:"column:file_id;not null;uniqueIndex:idx_file_group_name" json:"file_id"`
	GroupName   string    `gorm:"column:group_name;size:100;not null;default:'default';uniqueIndex:idx_file_group_name" json:"group_name"`
	IsDefault   bool      `gorm:"column:is_default;default:false" json:"is_default"`
	Description string    `gorm:"column:description;type:text" json:"description"`
	CreatedAt   time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	File        File      `gorm:"foreignKey:FileID;references:FileID" json:"file"`
}

func (TagGroup) TableName() string {
	return "tag_groups"
}

type TagGroupTag struct {
	ID       uint     `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	GroupID  uint     `gorm:"column:group_id;not null;index;uniqueIndex:idx_group_tag_val" json:"group_id"`
	FileID   uint     `gorm:"column:file_id;not null;index" json:"file_id"`
	TagID    uint     `gorm:"column:tag_id;not null;uniqueIndex:idx_group_tag_val" json:"tag_id"`
	TagValue string   `gorm:"column:tag_value;size:200;default:'';uniqueIndex:idx_group_tag_val" json:"tag_value"`
	Group    TagGroup `gorm:"foreignKey:GroupID;references:GroupID" json:"group"`
	Tag      Tag      `gorm:"foreignKey:TagID;references:TagID" json:"tag"`
}

func (TagGroupTag) TableName() string {
	return "tag_group_tags"
}
