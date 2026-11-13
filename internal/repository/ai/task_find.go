package ai

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

// FindTask loads a task by primary key; returns (nil, nil) when absent.
func (r *Repository) FindTask(id uint) (*model.AITask, error) {
	var task model.AITask
	err := r.db.First(&task, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}
