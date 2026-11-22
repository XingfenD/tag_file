package schema

import "fmt"

var fieldTypes = map[string]bool{"single": true, "bool": true, "multi": true}

// checkField validates one field's type/widget combination, the existence of
// its bind target, and that no other field already occupies the tag
// (selfID != 0 when re-saving a field onto its own binding).
func (s *Service) checkField(label, fieldType, widget string, bindTagID, selfID uint) error {
	if label == "" {
		return fmt.Errorf("字段名不能为空")
	}
	if !fieldTypes[fieldType] {
		return fmt.Errorf("未知字段类型 %q", fieldType)
	}
	switch widget {
	case "", "select":
	case "stars":
		if fieldType != "single" {
			return fmt.Errorf("星级控件只适用于单选字段")
		}
	default:
		return fmt.Errorf("未知控件 %q", widget)
	}
	tag, err := s.tagRepo.Find().ByID(bindTagID).Exec()
	if err != nil {
		return fmt.Errorf("查询绑定标签失败: %w", err)
	}
	if tag == nil {
		return fmt.Errorf("绑定标签不存在")
	}
	occupier, err := s.repo.FieldByBindTagID(bindTagID)
	if err != nil {
		return fmt.Errorf("查询标签占用失败: %w", err)
	}
	if occupier != nil && occupier.FieldID != selfID {
		return fmt.Errorf("该标签已被字段 %q 绑定", occupier.Label)
	}
	return nil
}
