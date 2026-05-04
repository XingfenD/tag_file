package tag

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) CreateValue(val *model.TagValue) error {
	return r.db.Create(val).Error
}
