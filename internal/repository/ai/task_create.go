package ai

import "github.com/XingfenD/tag_file/internal/model"

// CreateTask inserts one recognition task and backfills TaskID.
func (r *Repository) CreateTask(t *model.AITask) error {
	return r.db.Create(t).Error
}
