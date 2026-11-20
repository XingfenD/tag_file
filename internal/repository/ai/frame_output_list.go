package ai

import "github.com/XingfenD/tag_file/internal/model"

// ListFrameOutputs returns every stored frame of one task ordered by
// frame_index, so a refine pass reads the first-pass answers in frame order.
// A task without stored frames yields a non-nil empty slice.
func (r *Repository) ListFrameOutputs(taskID uint) ([]model.AIFrameOutput, error) {
	outputs := make([]model.AIFrameOutput, 0)
	err := r.db.Where("task_id = ?", taskID).Order("frame_index").Find(&outputs).Error
	return outputs, err
}
