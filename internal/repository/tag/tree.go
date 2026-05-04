package tag

import "github.com/XingfenD/tag_file/internal/model"

func (r *Repository) FindChildren(parentID uint) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Where("parent_id = ?", parentID).Order("tag_id").Find(&tags).Error
	return tags, err
}

func (r *Repository) FindRootsByCategory(categoryID uint) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Where("category_id = ? AND parent_id IS NULL", categoryID).Order("tag_id").Find(&tags).Error
	return tags, err
}

func (r *Repository) FindDescendantIDs(tagID uint) ([]uint, error) {
	var ids []uint
	ids = append(ids, tagID)
	queue := []uint{tagID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		children, err := r.FindChildren(current)
		if err != nil {
			return nil, err
		}
		for _, child := range children {
			ids = append(ids, child.TagID)
			queue = append(queue, child.TagID)
		}
	}
	return ids, nil
}
