package attr

import (
	"fmt"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Service) AddValue(attrID uint, valueName string) (*model.AttributeValue, error) {
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

func (s *Service) RemoveValue(valueID uint) error {
	val, err := s.attrRepo.FindValueByID(valueID)
	if err != nil {
		return fmt.Errorf("查询属性值失败: %w", err)
	}
	if val == nil {
		return fmt.Errorf("属性值 ID %d 不存在", valueID)
	}
	return s.attrRepo.DeleteValue(valueID)
}

func (s *Service) ListValues(attrID uint, offset, limit int) ([]model.AttributeValue, int64, error) {
	list, total, err := s.attrRepo.ListValues(attrID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("查询属性值失败: %w", err)
	}
	return list, total, nil
}

func (s *Service) UpdateValue(valueID uint, newName string) (*model.AttributeValue, error) {
	val, err := s.attrRepo.FindValueByID(valueID)
	if err != nil {
		return nil, fmt.Errorf("查询属性值失败: %w", err)
	}
	if val == nil {
		return nil, fmt.Errorf("属性值 ID %d 不存在", valueID)
	}

	if newName != "" && newName != val.ValueName {
		existing, err := s.attrRepo.FindValueByName(newName, val.AttributeID)
		if err != nil {
			return nil, fmt.Errorf("查询属性值失败: %w", err)
		}
		if existing != nil && existing.ValueID != valueID {
			return nil, fmt.Errorf("属性值 '%s' 已存在", newName)
		}
		val.ValueName = newName
	}

	if err := s.attrRepo.UpdateValue(val); err != nil {
		return nil, fmt.Errorf("更新属性值失败: %w", err)
	}
	return val, nil
}

func (s *Service) SearchValues(keyword string, attrID uint, offset, limit int) ([]model.AttributeValue, int64, error) {
	if keyword == "" {
		return nil, 0, fmt.Errorf("搜索关键词不能为空")
	}
	return s.attrRepo.SearchValuesByName(keyword, attrID, offset, limit)
}
