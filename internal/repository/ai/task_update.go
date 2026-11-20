package ai

import "github.com/XingfenD/tag_file/internal/model"

type TaskUpdateOperation struct {
	repo *Repository
	task *model.AITask
	cols []string
}

func (r *Repository) UpdateTask(task *model.AITask) *TaskUpdateOperation {
	return &TaskUpdateOperation{repo: r, task: task}
}

func (op *TaskUpdateOperation) SetStatus(v string) *TaskUpdateOperation {
	op.cols = append(op.cols, "status")
	op.task.Status = v
	return op
}

func (op *TaskUpdateOperation) SetError(v string) *TaskUpdateOperation {
	op.cols = append(op.cols, "error")
	op.task.Error = v
	return op
}

func (op *TaskUpdateOperation) SetFrameCount(v int) *TaskUpdateOperation {
	op.cols = append(op.cols, "frame_count")
	op.task.FrameCount = v
	return op
}

func (op *TaskUpdateOperation) SetRefineStatus(v string) *TaskUpdateOperation {
	op.cols = append(op.cols, "refine_status")
	op.task.RefineStatus = v
	return op
}

// Exec writes only the columns whose setters were called (cols whitelist,
// same shape as internal/repository/tag/update.go).
func (op *TaskUpdateOperation) Exec() error {
	if len(op.cols) == 0 {
		return op.repo.db.Save(op.task).Error
	}
	return op.repo.db.Model(op.task).Select(op.cols).Updates(*op.task).Error
}
