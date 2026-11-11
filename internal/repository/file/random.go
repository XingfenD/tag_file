package file

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

// RandomPool describes the candidate set a random pick was drawn from.
type RandomPool struct {
	Total     int64
	Remaining int64
	Exhausted bool
}

type RandomOperation struct {
	repo    *Repository
	scope   string
	exclude []uint
}

func (r *Repository) Random() *RandomOperation {
	return &RandomOperation{repo: r}
}

// Scope narrows the pool: "todo" skips files already marked done,
// "untagged" skips files carrying at least one tag, "all" narrows nothing.
func (op *RandomOperation) Scope(scope string) *RandomOperation {
	op.scope = scope
	return op
}

// Exclude drops already-shown file ids from the pool.
func (op *RandomOperation) Exclude(ids []uint) *RandomOperation {
	op.exclude = ids
	return op
}

func (op *RandomOperation) base() *gorm.DB {
	q := op.repo.db.Model(&model.File{})
	switch op.scope {
	case "todo":
		q = q.Where("tag_done = ?", false)
	case "untagged":
		q = q.Where("tag_cnt = ?", 0)
	}
	return q
}

// Exec draws one file at random. When every candidate has already been shown,
// it draws again from the whole scope and marks the pool exhausted, so a
// recommendation loop never dead-ends. A nil file with no error means the
// scope itself is empty.
func (op *RandomOperation) Exec() (*model.File, *RandomPool, error) {
	var pool RandomPool
	if err := op.base().Count(&pool.Total).Error; err != nil {
		return nil, nil, err
	}

	q := op.base()
	if len(op.exclude) > 0 {
		q = q.Where("file_id NOT IN ?", op.exclude)
	}
	if err := q.Count(&pool.Remaining).Error; err != nil {
		return nil, nil, err
	}

	var file model.File
	err := q.Order("RANDOM()").Take(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		pool.Exhausted = pool.Total > 0
		err = op.base().Order("RANDOM()").Take(&file).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &pool, nil
	}
	if err != nil {
		return nil, &pool, err
	}
	return &file, &pool, nil
}
