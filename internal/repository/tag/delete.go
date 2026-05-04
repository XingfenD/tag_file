package tag

import "github.com/XingfenD/tag_file/internal/model"

type DeleteOperation struct {
	repo *Repository
	id   *uint
	name *string
}

func (r *Repository) Delete() *DeleteOperation {
	return &DeleteOperation{repo: r}
}

func (op *DeleteOperation) ByID(id uint) *DeleteOperation {
	op.id = &id
	return op
}

func (op *DeleteOperation) ByName(name string) *DeleteOperation {
	op.name = &name
	return op
}

func (op *DeleteOperation) Exec() error {
	if op.id != nil {
		return op.repo.db.Delete(&model.Tag{}, *op.id).Error
	}
	_, err := op.execByName()
	return err
}

func (op *DeleteOperation) ExecCount() (int64, error) {
	if op.name != nil {
		return op.execByName()
	}
	return 0, op.Exec()
}

func (op *DeleteOperation) execByName() (int64, error) {
	result := op.repo.db.Where("tag_name = ?", *op.name).Delete(&model.Tag{})
	return result.RowsAffected, result.Error
}
