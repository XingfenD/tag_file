package attr

import (
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository() *Repository {
	return &Repository{db: storage.GetDB()}
}
