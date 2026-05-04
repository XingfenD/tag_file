package file

import "github.com/XingfenD/tag_file/internal/model"

type CreateOperation struct {
	repo *Repository
	file model.File
	cols []string
}

func (r *Repository) Create() *CreateOperation {
	return &CreateOperation{repo: r}
}

func (op *CreateOperation) SetPath(v string) *CreateOperation {
	op.cols = append(op.cols, "file_path")
	op.file.FilePath = v
	return op
}

func (op *CreateOperation) SetName(v string) *CreateOperation {
	op.cols = append(op.cols, "file_name")
	op.file.FileName = v
	return op
}

func (op *CreateOperation) SetSize(v int64) *CreateOperation {
	op.cols = append(op.cols, "file_size")
	op.file.FileSize = v
	return op
}

func (op *CreateOperation) SetType(v string) *CreateOperation {
	op.cols = append(op.cols, "file_type")
	op.file.FileType = v
	return op
}

func (op *CreateOperation) Exec() (*model.File, error) {
	if len(op.cols) > 0 {
		err := op.repo.db.Select(op.cols).Create(&op.file).Error
		return &op.file, err
	}
	err := op.repo.db.Create(&op.file).Error
	return &op.file, err
}
