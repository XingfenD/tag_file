package model

import "time"

type AITask struct {
	TaskID     uint   `gorm:"primaryKey;column:task_id" json:"task_id"`
	FileID     uint   `gorm:"column:file_id;not null;index" json:"file_id"`
	Status     string `gorm:"column:status;size:20;not null;default:'queued';index" json:"status"`
	Error      string `gorm:"column:error;type:text" json:"error"`
	FrameCount int    `gorm:"column:frame_count;default:0" json:"frame_count"`
	// SourceTaskID points at the first-pass task a refine task was derived from
	// (0 for a normal task), so a re-run can be traced back to its original.
	SourceTaskID uint      `gorm:"column:source_task_id;default:0" json:"source_task_id"`
	RefineStatus string    `gorm:"column:refine_status;size:20;default:''" json:"refine_status"`
	CreatedAt    time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	File         File      `gorm:"foreignKey:FileID;references:FileID" json:"file"`
}

func (AITask) TableName() string {
	return "ai_tasks"
}

// AIFrameOutput keeps the raw answer of one frame so a later refine pass can
// read the first-pass text without calling the model again.
type AIFrameOutput struct {
	OutputID   uint      `gorm:"primaryKey;column:output_id" json:"output_id"`
	TaskID     uint      `gorm:"column:task_id;index" json:"task_id"`
	FileID     uint      `gorm:"column:file_id;index" json:"file_id"`
	FrameIndex int       `gorm:"column:frame_index;default:0" json:"frame_index"`
	RawText    string    `gorm:"column:raw_text;type:text" json:"raw_text"`
	CreatedAt  time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (AIFrameOutput) TableName() string {
	return "ai_frame_outputs"
}

type AISuggestion struct {
	SuggestionID      uint       `gorm:"primaryKey;column:suggestion_id" json:"suggestion_id"`
	TaskID            uint       `gorm:"column:task_id;not null;index" json:"task_id"`
	FileID            uint       `gorm:"column:file_id;not null;index" json:"file_id"`
	CategoryName      string     `gorm:"column:category_name;size:100;not null" json:"category_name"`
	TagName           string     `gorm:"column:tag_name;size:100;not null" json:"tag_name"`
	TagValue          string     `gorm:"column:tag_value;size:200;default:''" json:"tag_value"`
	FinalCategoryName string     `gorm:"column:final_category_name;size:100;not null;default:''" json:"final_category_name"`
	FinalTagName      string     `gorm:"column:final_tag_name;size:100;not null;default:''" json:"final_tag_name"`
	FinalValue        string     `gorm:"column:final_value;size:200;default:''" json:"final_value"`
	MatchedTagID      *uint      `gorm:"column:matched_tag_id;index" json:"matched_tag_id,omitempty"`
	ApprovedTagID     *uint      `gorm:"column:approved_tag_id" json:"approved_tag_id,omitempty"`
	ReviewStatus      string     `gorm:"column:review_status;size:20;not null;default:'pending';index" json:"review_status"`
	ReviewedAt        *time.Time `gorm:"column:reviewed_at" json:"reviewed_at,omitempty"`
	CreatedAt         time.Time  `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (AISuggestion) TableName() string {
	return "ai_suggestions"
}
