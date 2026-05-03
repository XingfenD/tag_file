package service

import (
	"fmt"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/repository"
)

type AttrService struct {
	attrRepo *repository.AttrRepository
	tagRepo  *repository.TagRepository
}

func NewAttrService() *AttrService {
	return &AttrService{
		attrRepo: repository.NewAttrRepository(),
		tagRepo:  repository.NewTagRepository(),
	}
}

func (s *AttrService) AddAttribute(tagName string, attrName string, description string) (*model.Attribute, error) {
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

func (s *AttrService) RemoveAttribute(attrID uint) error {
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

func (s *AttrService) ListAttributes(tagName string, offset, limit int) ([]model.Attribute, int64, error) {
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

func (s *AttrService) AddValue(attrID uint, valueName string) (*model.AttributeValue, error) {
	attr, err := s.attrRepo.FindAttributeByID(attrID)
	if err != nil {
		return nil, fmt.Errorf("查询属性失败: %w", err)
	}
	if attr == nil {
		return nil, fmt.Errorf("属性 ID %d 不存在", attrID)
	}

	existing, err := s.attrRepo.FindValueByName(valueName, attrID)
	if err != nil {
		return nil, fmt.Errorf("查询属性值失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("属性值 '%s' 已存在", valueName)
	}

	val := &model.AttributeValue{
		AttributeID: attrID,
		ValueName:   valueName,
	}
	if err := s.attrRepo.CreateValue(val); err != nil {
		return nil, fmt.Errorf("创建属性值失败: %w", err)
	}
	return val, nil
}

func (s *AttrService) RemoveValue(valueID uint) error {
	val, err := s.attrRepo.FindValueByID(valueID)
	if err != nil {
		return fmt.Errorf("查询属性值失败: %w", err)
	}
	if val == nil {
		return fmt.Errorf("属性值 ID %d 不存在", valueID)
	}
	return s.attrRepo.DeleteValue(valueID)
}

func (s *AttrService) ListValues(attrID uint, offset, limit int) ([]model.AttributeValue, int64, error) {
	list, total, err := s.attrRepo.ListValues(attrID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("查询属性值失败: %w", err)
	}
	return list, total, nil
}
