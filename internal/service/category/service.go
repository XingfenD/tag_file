package category

import (
	"fmt"

	"github.com/XingfenD/tag_file/internal/model"
	repoCategory "github.com/XingfenD/tag_file/internal/repository/category"
	repoTag "github.com/XingfenD/tag_file/internal/repository/tag"
)

type Service struct {
	categoryRepo *repoCategory.Repository
	tagRepo      *repoTag.Repository
}

func NewService() *Service {
	return &Service{
		categoryRepo: repoCategory.NewRepository(),
		tagRepo:      repoTag.NewRepository(),
	}
}

func (s *Service) AddCategory(name string, description string) (*model.TagCategory, error) {
	existing, err := s.categoryRepo.FindByName(name)
	if err != nil {
		return nil, fmt.Errorf("查询分类项失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("分类项 '%s' 已存在", name)
	}

	category := &model.TagCategory{
		CategoryName: name,
		Description:  description,
	}
	if err := s.categoryRepo.Create(category); err != nil {
		return nil, fmt.Errorf("创建分类项失败: %w", err)
	}
	return category, nil
}

func (s *Service) RemoveCategory(name string) error {
	category, err := s.categoryRepo.FindByName(name)
	if err != nil {
		return fmt.Errorf("查询分类项失败: %w", err)
	}
	if category == nil {
		return fmt.Errorf("分类项 '%s' 不存在", name)
	}

	hasTags, err := s.categoryRepo.HasTags(category.CategoryID)
	if err != nil {
		return fmt.Errorf("查询标签关联失败: %w", err)
	}
	if hasTags {
		return fmt.Errorf("分类项 '%s' 下存在标签，请先删除标签", name)
	}

	if err := s.categoryRepo.DeleteByID(category.CategoryID); err != nil {
		return fmt.Errorf("删除分类项失败: %w", err)
	}
	return nil
}

func (s *Service) ListCategories(offset, limit int) ([]model.TagCategory, int64, error) {
	list, total, err := s.categoryRepo.List(offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("查询分类项列表失败: %w", err)
	}
	return list, total, nil
}

func (s *Service) UpdateCategory(categoryID uint, newName string, newDescription string) (*model.TagCategory, error) {
	category, err := s.categoryRepo.FindByID(categoryID)
	if err != nil {
		return nil, fmt.Errorf("查询分类项失败: %w", err)
	}
	if category == nil {
		return nil, fmt.Errorf("分类项 ID %d 不存在", categoryID)
	}

	if newName != "" && newName != category.CategoryName {
		existing, err := s.categoryRepo.FindByName(newName)
		if err != nil {
			return nil, fmt.Errorf("查询分类项失败: %w", err)
		}
		if existing != nil && existing.CategoryID != categoryID {
			return nil, fmt.Errorf("分类项 '%s' 已存在", newName)
		}
		category.CategoryName = newName
	}
	if newDescription != "" {
		category.Description = newDescription
	}

	if err := s.categoryRepo.Update(category); err != nil {
		return nil, fmt.Errorf("更新分类项失败: %w", err)
	}
	return category, nil
}

func (s *Service) SearchCategories(keyword string, offset, limit int) ([]model.TagCategory, int64, error) {
	if keyword == "" {
		return nil, 0, fmt.Errorf("搜索关键词不能为空")
	}
	return s.categoryRepo.SearchByName(keyword, offset, limit)
}
