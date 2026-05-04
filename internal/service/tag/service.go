package tag

import (
	"fmt"
	"log"

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

func (s *Service) AddTag(categoryName string, tagName string, description string, parentTagName string) (*model.Tag, error) {
	category, err := s.categoryRepo.FindByName(categoryName)
	if err != nil {
		log.Printf("[tag] AddTag: find category: %v", err)
		return nil, fmt.Errorf("查询分类项失败: %w", err)
	}
	if category == nil {
		return nil, fmt.Errorf("分类项 '%s' 不存在，请先创建分类项", categoryName)
	}

	var parentID *uint
	if parentTagName != "" {
		parents, err := s.tagRepo.FindByNameAndCategory(parentTagName, category.CategoryID)
		if err != nil {
			log.Printf("[tag] AddTag: find parent: %v", err)
			return nil, fmt.Errorf("查询父标签失败: %w", err)
		}
		if len(parents) == 0 {
			return nil, fmt.Errorf("父标签 '%s' 不存在", parentTagName)
		}
		parentID = &parents[0].TagID
	}

	existing, err := s.tagRepo.FindByParentNameAndCategory(parentID, tagName, category.CategoryID)
	if err != nil {
		log.Printf("[tag] AddTag: check dup: %v", err)
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf("标签 '%s' 在此层级已存在", tagName)
	}

	tag := &model.Tag{
		CategoryID:  category.CategoryID,
		TagName:     tagName,
		Description: description,
		ParentID:    parentID,
	}
	if err := s.tagRepo.Create(tag); err != nil {
		log.Printf("[tag] AddTag: create: %v", err)
		return nil, fmt.Errorf("创建标签失败: %w", err)
	}
	return tag, nil
}

func (s *Service) RemoveTag(tagName string) (int64, error) {
	count, err := s.tagRepo.DeleteByName(tagName)
	if err != nil {
		log.Printf("[tag] RemoveTag: %v", err)
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
			log.Printf("[tag] ListTags: find category: %v", err)
			return nil, 0, fmt.Errorf("查询分类项失败: %w", err)
		}
		if category == nil {
			return nil, 0, fmt.Errorf("分类项 '%s' 不存在", categoryName)
		}
		list, total, err := s.tagRepo.ListByCategory(category.CategoryID, offset, limit)
		if err != nil {
			log.Printf("[tag] ListTags: list by category: %v", err)
			return nil, 0, fmt.Errorf("查询标签失败: %w", err)
		}
		return list, total, nil
	}
	list, total, err := s.tagRepo.List(offset, limit)
	if err != nil {
		log.Printf("[tag] ListTags: list: %v", err)
		return nil, 0, fmt.Errorf("查询标签失败: %w", err)
	}
	return list, total, nil
}

func (s *Service) UpdateTag(tagID uint, newName string, newDescription string, newParentTagName string) (*model.Tag, error) {
	tag, err := s.tagRepo.FindByID(tagID)
	if err != nil {
		log.Printf("[tag] UpdateTag: find: %v", err)
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	if tag == nil {
		return nil, fmt.Errorf("标签 ID %d 不存在", tagID)
	}

	if newName != "" && newName != tag.TagName {
		tag.TagName = newName
	}

	if newParentTagName != "" {
		if tag.ParentID != nil {
			oldParent, _ := s.tagRepo.FindByID(*tag.ParentID)
			if oldParent != nil && oldParent.TagName == newParentTagName {
				newParentTagName = ""
			}
		}
		if newParentTagName != "" {
			parents, err := s.tagRepo.FindByNameAndCategory(newParentTagName, tag.CategoryID)
			if err != nil {
				log.Printf("[tag] UpdateTag: find parent: %v", err)
				return nil, fmt.Errorf("查询父标签失败: %w", err)
			}
			if len(parents) == 0 {
				return nil, fmt.Errorf("父标签 '%s' 不存在", newParentTagName)
			}
			parent := &parents[0]
			if parent.TagID == tagID {
				return nil, fmt.Errorf("不能将自己设为父标签")
			}
			descIDs, err := s.tagRepo.FindDescendantIDs(tagID)
			if err != nil {
				log.Printf("[tag] UpdateTag: find descendants: %v", err)
				return nil, fmt.Errorf("查询子孙标签失败: %w", err)
			}
			for _, id := range descIDs {
				if id == parent.TagID {
					return nil, fmt.Errorf("不能将子孙标签设为父标签（循环引用）")
				}
			}
			tag.ParentID = &parent.TagID
		}
	}

	if newDescription != "" {
		tag.Description = newDescription
	}

	if newName != "" || newParentTagName != "" {
		checkName := tag.TagName
		if newName != "" {
			checkName = newName
		}
		checkParentID := tag.ParentID
		if newParentTagName != "" && tag.ParentID != nil {
			checkParentID = tag.ParentID
		}
		existing, err := s.tagRepo.FindByParentNameAndCategory(checkParentID, checkName, tag.CategoryID)
		if err != nil {
			log.Printf("[tag] UpdateTag: check dup: %v", err)
			return nil, fmt.Errorf("查询标签失败: %w", err)
		}
		for _, t := range existing {
			if t.TagID != tagID {
				return nil, fmt.Errorf("标签 '%s' 在此层级已存在", checkName)
			}
		}
	}

	if err := s.tagRepo.Update(tag); err != nil {
		log.Printf("[tag] UpdateTag: save: %v", err)
		return nil, fmt.Errorf("更新标签失败: %w", err)
	}
	return tag, nil
}

func (s *Service) ListTagTree(categoryName string) ([]TagTreeNode, error) {
	var roots []model.Tag
	if categoryName != "" {
		category, err := s.categoryRepo.FindByName(categoryName)
		if err != nil {
			log.Printf("[tag] ListTagTree: find category: %v", err)
			return nil, fmt.Errorf("查询分类项失败: %w", err)
		}
		if category == nil {
			return nil, fmt.Errorf("分类项 '%s' 不存在", categoryName)
		}
		roots, err = s.tagRepo.FindRootsByCategory(category.CategoryID)
		if err != nil {
			log.Printf("[tag] ListTagTree: find roots: %v", err)
			return nil, fmt.Errorf("查询根标签失败: %w", err)
		}
	} else {
		categories, _, err := s.categoryRepo.List(0, 0)
		if err != nil {
			log.Printf("[tag] ListTagTree: list categories: %v", err)
			return nil, fmt.Errorf("查询分类项失败: %w", err)
		}
		for _, cat := range categories {
			r, e := s.tagRepo.FindRootsByCategory(cat.CategoryID)
			if e != nil {
				return nil, e
			}
			roots = append(roots, r...)
		}
	}

	nodes := make([]TagTreeNode, 0, len(roots))
	for _, root := range roots {
		node, err := s.buildTagTreeNode(&root)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *node)
	}
	return nodes, nil
}

type TagTreeNode struct {
	Tag      model.Tag
	Children []TagTreeNode
}

func (s *Service) buildTagTreeNode(tag *model.Tag) (*TagTreeNode, error) {
	children, err := s.tagRepo.FindChildren(tag.TagID)
	if err != nil {
		log.Printf("[tag] buildTagTreeNode: %v", err)
		return nil, fmt.Errorf("查询子标签失败: %w", err)
	}
	node := &TagTreeNode{Tag: *tag}
	for _, child := range children {
		childNode, err := s.buildTagTreeNode(&child)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, *childNode)
	}
	return node, nil
}

func (s *Service) GetDescendantIDs(tagName string) ([]uint, error) {
	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	var allIDs []uint
	for _, t := range tags {
		descIDs, err := s.tagRepo.FindDescendantIDs(t.TagID)
		if err != nil {
			return nil, err
		}
		allIDs = append(allIDs, descIDs...)
	}
	return allIDs, nil
}

func (s *Service) SearchTags(keyword string, offset, limit int) ([]model.Tag, int64, error) {
	if keyword == "" {
		return nil, 0, fmt.Errorf("搜索关键词不能为空")
	}
	return s.tagRepo.SearchByName(keyword, offset, limit)
}

func (s *Service) AddTagValue(tagName string, value string) (*model.TagValue, error) {
	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		log.Printf("[tag] AddTagValue: find tag: %v", err)
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("标签 '%s' 不存在", tagName)
	}
	tag := tags[0]

	existing, err := s.tagRepo.FindValueByName(tag.TagID, value)
	if err != nil {
		log.Printf("[tag] AddTagValue: check dup: %v", err)
		return nil, fmt.Errorf("查询标签值失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("标签值 '%s' 已存在", value)
	}

	val := &model.TagValue{TagID: tag.TagID, Value: value}
	if err := s.tagRepo.CreateValue(val); err != nil {
		log.Printf("[tag] AddTagValue: create: %v", err)
		return nil, fmt.Errorf("创建标签值失败: %w", err)
	}
	return val, nil
}

func (s *Service) RemoveTagValue(tagName string, value string) error {
	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		log.Printf("[tag] RemoveTagValue: find tag: %v", err)
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	val, err := s.tagRepo.FindValueByName(tags[0].TagID, value)
	if err != nil {
		log.Printf("[tag] RemoveTagValue: find value: %v", err)
		return fmt.Errorf("查询标签值失败: %w", err)
	}
	if val == nil {
		return fmt.Errorf("标签值 '%s' 不存在", value)
	}
	return s.tagRepo.DeleteValue(val.ValueID)
}

func (s *Service) ListTagValues(tagName string) ([]model.TagValue, error) {
	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("标签 '%s' 不存在", tagName)
	}
	return s.tagRepo.ListValues(tags[0].TagID)
}
