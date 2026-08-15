package model

import "time"

type File struct {
	FileID    uint      `gorm:"primaryKey;column:file_id" json:"file_id"`
	FilePath  string    `gorm:"column:file_path;size:255;not null" json:"file_path"`
	FileName  string    `gorm:"column:file_name;size:255;not null" json:"file_name"`
	FileSize  int64     `gorm:"column:file_size" json:"file_size"`
	FileType  string    `gorm:"column:file_type;size:50" json:"file_type"`
	TagCnt    int       `gorm:"column:tag_cnt;default:0" json:"tag_cnt"`
	TagDone   bool      `gorm:"column:tag_done;default:false" json:"tag_done"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (File) TableName() string {
	return "files"
}
