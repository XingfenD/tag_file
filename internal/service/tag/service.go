package tag

import (
	"fmt"

	"github.com/XingfenD/tag_file/internal/model"
	repoCategory "github.com/XingfenD/tag_file/internal/repository/category"
	repoTag "github.com/XingfenD/tag_file/internal/repository/tag"
)

type Service struct {
	tagRepo      *repoTag.Repository
	categoryRepo *repoCategory.Repository
}

func NewService() *Service {
	return &Service{
		tagRepo:      repoTag.NewRepository(),
		categoryRepo: repoCategory.NewRepository(),
	}
}

func (s *Service) AddTag(categoryName string, tagName string, description string) (*model.Tag, error) {
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

func (s *Service) RemoveTag(tagName string) (int64, error) {
	count, err := s.tagRepo.DeleteByName(tagName)
	if err != nil {
		return 0, fmt.Errorf("删除标签失败: %w", err)
	}
	if count == 0 {
		return 0, fmt.Errorf("标签 '%s' 不存在", tagName)
	}
	return count, nil
}

func (s *Service) ListTags(categoryName string, offset, limit int) ([]model.Tag, int64, error) {
	if categoryName != "" {
		category, err := s.categoryRepo.FindByName(categoryName)
		if err != nil {
			return nil, 0, fmt.Errorf("查询分类项失败: %w", err)
		}
		if category == nil {
			return nil, 0, fmt.Errorf("分类项 '%s' 不存在", categoryName)
		}
		list, total, err := s.tagRepo.ListByCategory(category.CategoryID, offset, limit)
		if err != nil {
			return nil, 0, fmt.Errorf("查询标签失败: %w", err)
		}
		return list, total, nil
	}
	list, total, err := s.tagRepo.List(offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("查询标签失败: %w", err)
	}
	return list, total, nil
}

func (s *Service) UpdateTag(tagID uint, newName string, newDescription string) (*model.Tag, error) {
	tag, err := s.tagRepo.FindByID(tagID)
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	if tag == nil {
		return nil, fmt.Errorf("标签 ID %d 不存在", tagID)
	}

	if newName != "" && newName != tag.TagName {
		existing, err := s.tagRepo.FindByNameAndCategory(newName, tag.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("查询标签失败: %w", err)
		}
		if existing != nil && existing.TagID != tagID {
			return nil, fmt.Errorf("标签 '%s' 已存在", newName)
		}
		tag.TagName = newName
	}
	if newDescription != "" {
		tag.Description = newDescription
	}

	if err := s.tagRepo.Update(tag); err != nil {
		return nil, fmt.Errorf("更新标签失败: %w", err)
	}
	return tag, nil
}

func (s *Service) SearchTags(keyword string, offset, limit int) ([]model.Tag, int64, error) {
	if keyword == "" {
		return nil, 0, fmt.Errorf("搜索关键词不能为空")
	}
	return s.tagRepo.SearchByName(keyword, offset, limit)
}
