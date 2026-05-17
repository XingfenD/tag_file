package file

import "github.com/XingfenD/tag_file/internal/model"

type ListOperation struct {
	repo   *Repository
	offset int
	limit  int
}

// ListByDir returns files in a specific directory (direct children only).
func (r *Repository) ListByDir(dir string) ([]model.File, error) {
	var files []model.File
	if dir == "." || dir == "" {
		err := r.db.Where("file_path NOT LIKE ?", "%/%").
			Order("file_path").Find(&files).Error
		return files, err
	}
	prefix := dir
	if len(prefix) > 0 && prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	err := r.db.Where("file_path LIKE ? AND file_path NOT LIKE ?", prefix+"%", prefix+"%/%").
		Order("file_path").Find(&files).Error
	return files, err
}

func (r *Repository) List() *ListOperation {
	return &ListOperation{repo: r}
}

func (op *ListOperation) Offset(n int) *ListOperation {
	op.offset = n
	return op
}

func (op *ListOperation) Limit(n int) *ListOperation {
	op.limit = n
	return op
}

func (op *ListOperation) Exec() ([]model.File, int64, error) {
	var total int64
	op.repo.db.Model(&model.File{}).Count(&total)
	var files []model.File
	q := op.repo.db.Order("file_path")
	if op.limit > 0 {
		q = q.Offset(op.offset).Limit(op.limit)
	}
	err := q.Find(&files).Error
	return files, total, err
}
