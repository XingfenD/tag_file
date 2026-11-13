package ai

import (
	"errors"

	"github.com/XingfenD/tag_file/internal/model"
	"gorm.io/gorm"
)

// FindSuggestion loads a suggestion by primary key; returns (nil, nil) when absent.
func (r *Repository) FindSuggestion(id uint) (*model.AISuggestion, error) {
	var suggestion model.AISuggestion
	err := r.db.First(&suggestion, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &suggestion, nil
}
