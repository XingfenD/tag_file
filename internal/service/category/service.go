package category

import (
	"fmt"
	"log"

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
	existing, err := s.categoryRepo.Find().ByName(name).Exec()
	if err != nil {
		log.Printf("[category] AddCategory: check dup: %v", err)
		return nil, fmt.Errorf("查询分类项失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("分类项 '%s' 已存在", name)
	}

	category, err := s.categoryRepo.Create().SetName(name).SetDescription(description).Exec()
	if err != nil {
		log.Printf("[category] AddCategory: create: %v", err)
		// Handle race condition: unique constraint violation
		existing, findErr := s.categoryRepo.Find().ByName(name).Exec()
		if findErr == nil && existing != nil {
			return nil, fmt.Errorf("分类项 '%s' 已存在", name)
		}
		return nil, fmt.Errorf("创建分类项失败: %w", err)
	}
	return category, nil
}

func (s *Service) RemoveCategory(name string) error {
	category, err := s.categoryRepo.Find().ByName(name).Exec()
	if err != nil {
		log.Printf("[category] RemoveCategory: find: %v", err)
		return fmt.Errorf("查询分类项失败: %w", err)
	}
	if category == nil {
		return fmt.Errorf("分类项 '%s' 不存在", name)
	}

	hasTags, err := s.categoryRepo.HasTags(category.CategoryID)
	if err != nil {
		log.Printf("[category] RemoveCategory: check tags: %v", err)
		return fmt.Errorf("查询标签关联失败: %w", err)
	}
	if hasTags {
		return fmt.Errorf("分类项 '%s' 下存在标签，请先删除标签", name)
	}

	if err := s.categoryRepo.Delete().ByID(category.CategoryID).Exec(); err != nil {
		log.Printf("[category] RemoveCategory: delete: %v", err)
		return fmt.Errorf("删除分类项失败: %w", err)
	}
	return nil
}

func (s *Service) ListCategories(offset, limit int) ([]model.TagCategory, int64, error) {
	list, total, err := s.categoryRepo.List().Offset(offset).Limit(limit).Exec()
	if err != nil {
		log.Printf("[category] ListCategories: %v", err)
		return nil, 0, fmt.Errorf("查询分类项列表失败: %w", err)
	}
	return list, total, nil
}

func (s *Service) UpdateCategory(categoryID uint, newName string, newDescription string) (*model.TagCategory, error) {
	category, err := s.categoryRepo.Find().ByID(categoryID).Exec()
	if err != nil {
		log.Printf("[category] UpdateCategory: find: %v", err)
		return nil, fmt.Errorf("查询分类项失败: %w", err)
	}
	if category == nil {
		return nil, fmt.Errorf("分类项 ID %d 不存在", categoryID)
	}

	if newName != "" && newName != category.CategoryName {
		existing, err := s.categoryRepo.Find().ByName(newName).Exec()
		if err != nil {
			log.Printf("[category] UpdateCategory: check name dup: %v", err)
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

	if err := s.categoryRepo.Update(category).SetName(category.CategoryName).SetDescription(category.Description).Exec(); err != nil {
		log.Printf("[category] UpdateCategory: save: %v", err)
		return nil, fmt.Errorf("更新分类项失败: %w", err)
	}
	return category, nil
}

func (s *Service) FindByID(id uint) (*model.TagCategory, error) {
	return s.categoryRepo.Find().ByID(id).Exec()
}

func (s *Service) SearchCategories(keyword string, offset, limit int) ([]model.TagCategory, int64, error) {
	if keyword == "" {
		return nil, 0, fmt.Errorf("搜索关键词不能为空")
	}
	return s.categoryRepo.Search().ByName(keyword).Offset(offset).Limit(limit).Exec()
}
