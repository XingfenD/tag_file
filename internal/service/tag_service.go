package service

import (
	"fmt"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/repository"
)

type TagService struct {
	tagRepo      *repository.TagRepository
	categoryRepo *repository.CategoryRepository
}

func NewTagService() *TagService {
	return &TagService{
		tagRepo:      repository.NewTagRepository(),
		categoryRepo: repository.NewCategoryRepository(),
	}
}

func (s *TagService) AddTag(categoryName string, tagName string, description string) (*model.Tag, error) {
	category, err := s.categoryRepo.FindByName(categoryName)
	if err != nil {
		return nil, fmt.Errorf("查询分类项失败: %w", err)
	}
	if category == nil {
		return nil, fmt.Errorf("分类项 '%s' 不存在，请先创建分类项", categoryName)
	}

	existing, err := s.tagRepo.FindByNameAndCategory(tagName, category.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("标签 '%s' 在分类项 '%s' 下已存在", tagName, categoryName)
	}

	tag := &model.Tag{
		CategoryID:  category.CategoryID,
		TagName:     tagName,
		Description: description,
	}
	if err := s.tagRepo.Create(tag); err != nil {
		return nil, fmt.Errorf("创建标签失败: %w", err)
	}
	return tag, nil
}

func (s *TagService) RemoveTag(tagName string) (int64, error) {
	count, err := s.tagRepo.DeleteByName(tagName)
	if err != nil {
		return 0, fmt.Errorf("删除标签失败: %w", err)
	}
	if count == 0 {
		return 0, fmt.Errorf("标签 '%s' 不存在", tagName)
	}
	return count, nil
}

func (s *TagService) ListTags(categoryName string) ([]model.Tag, error) {
	if categoryName != "" {
		category, err := s.categoryRepo.FindByName(categoryName)
		if err != nil {
			return nil, fmt.Errorf("查询分类项失败: %w", err)
		}
		if category == nil {
			return nil, fmt.Errorf("分类项 '%s' 不存在", categoryName)
		}
		return s.tagRepo.ListByCategory(category.CategoryID)
	}
	return s.tagRepo.List()
}
