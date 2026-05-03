package file

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository() *Repository {
	return &Repository{db: storage.GetDB()}
}

func (r *Repository) Create(file *model.File) error {
	return r.db.Create(file).Error
}

func (r *Repository) FindByPath(path string) (*model.File, error) {
	var file model.File
	err := r.db.Where("file_path = ?", path).First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &file, err
}

func (r *Repository) FindByID(id uint) (*model.File, error) {
	var file model.File
	err := r.db.First(&file, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &file, err
}

func (r *Repository) List(offset, limit int) ([]model.File, int64, error) {
	var total int64
	r.db.Model(&model.File{}).Count(&total)
	var files []model.File
	q := r.db.Order("file_path")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&files).Error
	return files, total, err
}

func (r *Repository) GetIndexedPaths() (map[string]bool, error) {
	var paths []string
	err := r.db.Model(&model.File{}).Select("file_path").Find(&paths).Error
	if err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(paths))
	for _, p := range paths {
		m[p] = true
	}
	return m, nil
}

const batchSize = 100

func (r *Repository) CreateBatch(files []model.File) error {
	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}
		if err := r.db.Create(files[i:end]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) Update(file *model.File) error {
	return r.db.Save(file).Error
}

func (r *Repository) SearchByName(keyword string, offset, limit int) ([]model.File, int64, error) {
	pattern := "%" + keyword + "%"
	var total int64
	r.db.Model(&model.File{}).Where("file_name LIKE ? OR file_path LIKE ?", pattern, pattern).Count(&total)
	var files []model.File
	q := r.db.Where("file_name LIKE ? OR file_path LIKE ?", pattern, pattern).Order("file_path")
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	err := q.Find(&files).Error
	return files, total, err
}

func (r *Repository) Delete(id uint) error {
	return r.db.Delete(&model.File{}, id).Error
}
