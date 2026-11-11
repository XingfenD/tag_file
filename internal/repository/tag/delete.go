package tag

import (
	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

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
	_, err := op.exec()
	return err
}

func (op *DeleteOperation) ExecCount() (int64, error) {
	return op.exec()
}

// exec removes the selected tags together with everything that references them.
// Nothing here is enforced by the database (foreign keys stay off, and
// AutoMigrate emits no useful constraints), so skipping any step leaves rows
// that keep presenting a deleted tag as still applied to a file.
func (op *DeleteOperation) exec() (int64, error) {
	var deleted int64
	err := op.repo.db.Transaction(func(tx *gorm.DB) error {
		targets := tx.Model(&model.Tag{})
		if op.id != nil {
			targets = targets.Where("tag_id = ?", *op.id)
		} else {
			targets = targets.Where("tag_name = ?", *op.name)
		}
		var ids []uint
		if err := targets.Pluck("tag_id", &ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}

		type fileDelta struct {
			FileID uint `gorm:"column:file_id"`
			Rows   int  `gorm:"column:n"`
		}
		var deltas []fileDelta
		if err := tx.Model(&model.TagGroupTag{}).
			Select("file_id, COUNT(*) AS n").
			Where("tag_id IN ?", ids).
			Group("file_id").
			Scan(&deltas).Error; err != nil {
			return err
		}

		if err := tx.Where("tag_id IN ?", ids).Delete(&model.TagGroupTag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tag_id IN ?", ids).Delete(&model.TagValue{}).Error; err != nil {
			return err
		}
		// Children survive as roots; a dangling parent_id hides them from the
		// tree while they are still applied to files.
		if err := tx.Model(&model.Tag{}).Where("parent_id IN ?", ids).
			UpdateColumn("parent_id", gorm.Expr("NULL")).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.Tag{}, ids).Error; err != nil {
			return err
		}

		for _, d := range deltas {
			if err := tx.Model(&model.File{}).Where("file_id = ?", d.FileID).
				UpdateColumn("tag_cnt", gorm.Expr("MAX(tag_cnt - ?, 0)", d.Rows)).Error; err != nil {
				return err
			}
		}

		deleted = int64(len(ids))
		return nil
	})
	return deleted, err
}
