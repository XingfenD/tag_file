package file

import "github.com/XingfenD/tag_file/internal/model"

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
