package ai

import "github.com/XingfenD/tag_file/internal/model"

type TaskListOperation struct {
	repo   *Repository
	status string
	fileID uint
	offset int
	limit  int
}

func (r *Repository) ListTasks() *TaskListOperation {
	return &TaskListOperation{repo: r}
}

func (op *TaskListOperation) ByStatus(status string) *TaskListOperation {
	op.status = status
	return op
}

func (op *TaskListOperation) ByFileID(fileID uint) *TaskListOperation {
	op.fileID = fileID
	return op
}

func (op *TaskListOperation) Offset(n int) *TaskListOperation {
	op.offset = n
	return op
}

func (op *TaskListOperation) Limit(n int) *TaskListOperation {
	op.limit = n
	return op
}

// Exec returns one page of tasks ordered by task_id plus the filtered total.
// Limit(0) means unlimited.
func (op *TaskListOperation) Exec() ([]model.AITask, int64, error) {
	cq := op.repo.db.Model(&model.AITask{})
	if op.status != "" {
		cq = cq.Where("status = ?", op.status)
	}
	if op.fileID > 0 {
		cq = cq.Where("file_id = ?", op.fileID)
	}
	var total int64
	cq.Count(&total)

	q := op.repo.db.Model(&model.AITask{})
	if op.status != "" {
		q = q.Where("status = ?", op.status)
	}
	if op.fileID > 0 {
		q = q.Where("file_id = ?", op.fileID)
	}
	q = q.Order("task_id")
	if op.limit > 0 {
		q = q.Offset(op.offset).Limit(op.limit)
	}
	var tasks []model.AITask
	err := q.Find(&tasks).Error
	return tasks, total, err
}
