package attr

import (
	"fmt"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Service) AddAttribute(tagName string, attrName string, description string) (*model.Attribute, error) {
	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("标签 '%s' 不存在", tagName)
	}
	tag := tags[0]

	existing, err := s.attrRepo.FindAttributeByName(attrName, tag.TagID)
	if err != nil {
		return nil, fmt.Errorf("查询属性失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("属性 '%s' 在标签 '%s' 下已存在", attrName, tagName)
	}

	attr := &model.Attribute{
		TagID:         tag.TagID,
		AttributeName: attrName,
		Description:   description,
	}
	if err := s.attrRepo.CreateAttribute(attr); err != nil {
		return nil, fmt.Errorf("创建属性失败: %w", err)
	}
	return attr, nil
}

func (s *Service) RemoveAttribute(attrID uint) error {
	attr, err := s.attrRepo.FindAttributeByID(attrID)
	if err != nil {
		return fmt.Errorf("查询属性失败: %w", err)
	}
	if attr == nil {
		return fmt.Errorf("属性 ID %d 不存在", attrID)
	}

	hasVals, err := s.attrRepo.HasValues(attrID)
	if err != nil {
		return fmt.Errorf("查询属性值失败: %w", err)
	}
	if hasVals {
		return fmt.Errorf("属性下存在属性值，请先删除属性值")
	}

	return s.attrRepo.DeleteAttribute(attrID)
}

func (s *Service) ListAttributes(tagName string, offset, limit int) ([]model.Attribute, int64, error) {
	var tagID uint
	if tagName != "" {
		tags, err := s.tagRepo.FindByName(tagName)
		if err != nil {
			return nil, 0, fmt.Errorf("查询标签失败: %w", err)
		}
		if len(tags) == 0 {
			return nil, 0, fmt.Errorf("标签 '%s' 不存在", tagName)
		}
		tagID = tags[0].TagID
	}
	list, total, err := s.attrRepo.ListAttributes(tagID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("查询属性失败: %w", err)
	}
	return list, total, nil
}
