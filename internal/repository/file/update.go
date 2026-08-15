package file

import (
	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

type UpdateOperation struct {
	repo *Repository
	file *model.File
	cols []string
}

func (r *Repository) Update(file *model.File) *UpdateOperation {
	return &UpdateOperation{repo: r, file: file}
}

func (op *UpdateOperation) SetFilePath(v string) *UpdateOperation {
	op.cols = append(op.cols, "file_path")
	op.file.FilePath = v
	return op
}

func (op *UpdateOperation) SetFileName(v string) *UpdateOperation {
	op.cols = append(op.cols, "file_name")
	op.file.FileName = v
	return op
}

func (op *UpdateOperation) SetFileSize(v int64) *UpdateOperation {
	op.cols = append(op.cols, "file_size")
	op.file.FileSize = v
	return op
}

func (op *UpdateOperation) Exec() error {
	if len(op.cols) == 0 {
		return op.repo.db.Save(op.file).Error
	}
	return op.repo.db.Model(op.file).Select(op.cols).Updates(*op.file).Error
}

func (r *Repository) IncrementTagCnt(fileID uint, delta int) error {
	return r.db.Model(&model.File{}).Where("file_id = ?", fileID).
		UpdateColumn("tag_cnt", gorm.Expr("MAX(tag_cnt + ?, 0)", delta)).Error
}
