package file

import "github.com/XingfenD/tag_file/internal/model"

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
