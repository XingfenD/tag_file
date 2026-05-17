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
	category, err := s.categoryRepo.Find().ByName(categoryName).Exec()
	if err != nil {
		log.Printf("[tag] AddTag: find category: %v", err)
		return nil, fmt.Errorf("查询分类项失败: %w", err)
	}
	if category == nil {
		return nil, fmt.Errorf("分类项 '%s' 不存在，请先创建分类项", categoryName)
	}

	var parentID *uint
	if parentTagName != "" {
		parents, err := s.tagRepo.Find().ByNameAndCategory(parentTagName, category.CategoryID).All()
		if err != nil {
			log.Printf("[tag] AddTag: find parent: %v", err)
			return nil, fmt.Errorf("查询父标签失败: %w", err)
		}
		if len(parents) == 0 {
			return nil, fmt.Errorf("父标签 '%s' 不存在", parentTagName)
		}
		parentID = &parents[0].TagID
	}

	existing, err := s.tagRepo.Find().ByParentAndName(parentID, tagName).All()
	if err != nil {
		log.Printf("[tag] AddTag: check dup: %v", err)
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf("标签 '%s' 在此层级已存在", tagName)
	}

	tag, err := s.tagRepo.Create().SetName(tagName).SetCategoryID(category.CategoryID).SetParentID(parentID).SetDescription(description).Exec()
	if err != nil {
		log.Printf("[tag] AddTag: create: %v", err)
		return nil, fmt.Errorf("创建标签失败: %w", err)
	}
	return tag, nil
}

func (s *Service) RemoveTag(tagName string) (int64, error) {
	count, err := s.tagRepo.Delete().ByName(tagName).ExecCount()
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
		category, err := s.categoryRepo.Find().ByName(categoryName).Exec()
		if err != nil {
			log.Printf("[tag] ListTags: find category: %v", err)
			return nil, 0, fmt.Errorf("查询分类项失败: %w", err)
		}
		if category == nil {
			return nil, 0, fmt.Errorf("分类项 '%s' 不存在", categoryName)
		}
		list, total, err := s.tagRepo.List().ByCategory(category.CategoryID).Offset(offset).Limit(limit).Exec()
		if err != nil {
			log.Printf("[tag] ListTags: list by category: %v", err)
			return nil, 0, fmt.Errorf("查询标签失败: %w", err)
		}
		return list, total, nil
	}
	list, total, err := s.tagRepo.List().Offset(offset).Limit(limit).Exec()
	if err != nil {
		log.Printf("[tag] ListTags: list: %v", err)
		return nil, 0, fmt.Errorf("查询标签失败: %w", err)
	}
	return list, total, nil
}

func (s *Service) UpdateTag(tagID uint, newName string, newDescription string, newParentTagName string) (*model.Tag, error) {
	tag, err := s.tagRepo.Find().ByID(tagID).Exec()
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
			oldParent, _ := s.tagRepo.Find().ByID(*tag.ParentID).Exec()
			if oldParent != nil && oldParent.TagName == newParentTagName {
				newParentTagName = ""
			}
		}
		if newParentTagName != "" {
			parents, err := s.tagRepo.Find().ByNameAndCategory(newParentTagName, tag.CategoryID).All()
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
		existing, err := s.tagRepo.Find().ByParentAndName(checkParentID, checkName).All()
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

	if err := s.tagRepo.Update(tag).SetName(tag.TagName).SetDescription(tag.Description).SetParentID(tag.ParentID).Exec(); err != nil {
		log.Printf("[tag] UpdateTag: save: %v", err)
		return nil, fmt.Errorf("更新标签失败: %w", err)
	}
	return tag, nil
}

func (s *Service) ListTagTree(categoryName string) ([]TagTreeNode, error) {
	allTags, _, err := s.ListTags(categoryName, 0, 0)
	if err != nil {
		return nil, err
	}

	// Build parent→children map in one pass, no more DB calls
	childrenMap := make(map[uint][]model.Tag)
	var roots []model.Tag
	for i := range allTags {
		t := allTags[i]
		if t.ParentID != nil {
			childrenMap[*t.ParentID] = append(childrenMap[*t.ParentID], t)
		} else {
			roots = append(roots, t)
		}
	}

	nodes := make([]TagTreeNode, 0, len(roots))
	for _, root := range roots {
		nodes = append(nodes, s.buildTreeNodeFromMap(root, childrenMap))
	}
	return nodes, nil
}

type TagTreeNode struct {
	Tag      model.Tag
	Children []TagTreeNode
}

func (s *Service) buildTreeNodeFromMap(tag model.Tag, childrenMap map[uint][]model.Tag) TagTreeNode {
	node := TagTreeNode{Tag: tag}
	children := childrenMap[tag.TagID]
	for _, child := range children {
		node.Children = append(node.Children, s.buildTreeNodeFromMap(child, childrenMap))
	}
	return node
}

func (s *Service) GetDescendantIDs(tagName string) ([]uint, error) {
	tags, err := s.tagRepo.Find().ByName(tagName).All()
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	// Load all tags once and build in-memory adjacency for descendant traversal
	allTags, _, err := s.tagRepo.List().Offset(0).Limit(0).Exec()
	if err != nil {
		return nil, fmt.Errorf("查询标签列表失败: %w", err)
	}
	childrenMap := make(map[uint][]uint)
	for _, t := range allTags {
		if t.ParentID != nil {
			childrenMap[*t.ParentID] = append(childrenMap[*t.ParentID], t.TagID)
		}
	}

	var allIDs []uint
	for _, t := range tags {
		allIDs = append(allIDs, t.TagID)
		allIDs = append(allIDs, s.collectDescendantIDsFromMap(t.TagID, childrenMap)...)
	}
	return allIDs, nil
}

func (s *Service) collectDescendantIDsFromMap(tagID uint, childrenMap map[uint][]uint) []uint {
	var ids []uint
	queue := childrenMap[tagID]
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ids = append(ids, id)
		queue = append(queue, childrenMap[id]...)
	}
	return ids
}

func (s *Service) FindByID(id uint) (*model.Tag, error) {
	return s.tagRepo.Find().ByID(id).Exec()
}

func (s *Service) SearchTags(keyword string, offset, limit int) ([]model.Tag, int64, error) {
	if keyword == "" {
		return nil, 0, fmt.Errorf("搜索关键词不能为空")
	}
	return s.tagRepo.Search().ByName(keyword).Offset(offset).Limit(limit).Exec()
}

func (s *Service) AddTagValue(tagName string, value string) (*model.TagValue, error) {
	tags, err := s.tagRepo.Find().ByName(tagName).All()
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
	tags, err := s.tagRepo.Find().ByName(tagName).All()
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
	tags, err := s.tagRepo.Find().ByName(tagName).All()
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("标签 '%s' 不存在", tagName)
	}
	return s.tagRepo.ListValues(tags[0].TagID)
}
