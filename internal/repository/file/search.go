package file

import (
	"github.com/XingfenD/tag_file/internal/model"
)

func (r *Repository) SearchByTags(tagIDs []uint, logic string) ([]model.File, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}

	var files []model.File
	switch logic {
	case "and":
		err := r.db.Where("file_id IN (?)",
			r.db.Table("file_tags").
				Select("file_id").
				Where("tag_id IN (?)", tagIDs).
				Group("file_id").
				Having("COUNT(DISTINCT tag_id) = ?", len(tagIDs)),
		).Order("file_path").Find(&files).Error
		return files, err
	case "not":
		err := r.db.Where("file_id NOT IN (?)",
			r.db.Table("file_tags").Select("file_id").Where("tag_id IN (?)", tagIDs),
		).Order("file_path").Find(&files).Error
		return files, err
	default: // "or"
		err := r.db.Where("file_id IN (?)",
			r.db.Table("file_tags").Select("file_id").Where("tag_id IN (?)", tagIDs),
		).Order("file_path").Find(&files).Error
		return files, err
	}
}
