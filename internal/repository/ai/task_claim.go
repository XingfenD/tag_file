package ai

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

// ClaimNextQueued picks the queued task with the smallest task_id and marks it
// running. It returns (nil, nil) when nothing is queued or when the candidate
// was claimed by someone else between the two steps.
func (r *Repository) ClaimNextQueued() (*model.AITask, error) {
	var task model.AITask
	err := r.db.Where("status = ?", "queued").Order("task_id").First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	res := r.db.Model(&model.AITask{}).
		Where("task_id = ? AND status = ?", task.TaskID, "queued").
		Updates(map[string]any{"status": "running", "error": ""})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil
	}

	task.Status = "running"
	task.Error = ""
	return &task, nil
}

// FailRunning marks every running task failed with the given reason (restart
// sweep) and returns the number of affected rows.
func (r *Repository) FailRunning(reason string) (int64, error) {
	res := r.db.Model(&model.AITask{}).
		Where("status = ?", "running").
		Updates(map[string]any{"status": "failed", "error": reason})
	return res.RowsAffected, res.Error
}
