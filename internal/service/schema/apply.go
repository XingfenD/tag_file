package schema

import (
	"fmt"

	repoTagGroup "github.com/XingfenD/tag_file/internal/repository/taggroup"
)

// ApplyField materializes one control change onto the file's default group
// with takeover semantics: every existing row of the field's bind scope is
// replaced by what the panel holds now. Free-tagged or AI-added rows of the
// same tag are folded in — a schema save is an explicit statement of the
// field's single current value. No-op transitions stay silent.
func (s *Service) ApplyField(filePath string, fieldID uint, value string, checked []uint) error {
	fld, err := s.repo.GetField(fieldID)
	if err != nil {
		return err
	}
	if fld == nil {
		return fmt.Errorf("字段 %d 不存在", fieldID)
	}
	f, err := s.registeredFile(filePath)
	if err != nil {
		return err
	}
	groupID, err := s.defaultGroupID(f.FileID)
	if err != nil {
		return err
	}
	tag, err := s.tagRepo.Find().ByID(fld.BindTagID).Exec()
	if err != nil {
		return fmt.Errorf("查询绑定标签失败: %w", err)
	}
	if tag == nil {
		return fmt.Errorf("该字段的绑定标签已失效")
	}

	var tagIDs []uint
	var rows []repoTagGroup.ApplyRow
	switch fld.FieldType {
	case "single":
		tagIDs = []uint{fld.BindTagID}
		if value != "" {
			dv, err := s.tagRepo.FindValueByName(fld.BindTagID, value)
			if err != nil {
				return fmt.Errorf("查询标签字典失败: %w", err)
			}
			if dv == nil {
				return fmt.Errorf("取值 %q 不在标签 %q 的字典中", value, tag.TagName)
			}
			rows = []repoTagGroup.ApplyRow{{TagID: fld.BindTagID, Value: value}}
		}
	case "bool":
		tagIDs = []uint{fld.BindTagID}
		if value == "true" {
			rows = []repoTagGroup.ApplyRow{{TagID: fld.BindTagID}}
		}
	case "multi":
		kids, err := s.tagRepo.ChildIDs(fld.BindTagID)
		if err != nil {
			return fmt.Errorf("查询子标签失败: %w", err)
		}
		allowed := make(map[uint]bool, len(kids))
		for _, id := range kids {
			allowed[id] = true
		}
		seen := make(map[uint]bool, len(checked))
		for _, id := range checked {
			if !allowed[id] {
				return fmt.Errorf("取值 %d 不在该字段的候选之内", id)
			}
			if !seen[id] {
				seen[id] = true
				rows = append(rows, repoTagGroup.ApplyRow{TagID: id})
			}
		}
		tagIDs = kids
	default:
		return fmt.Errorf("未知字段类型 %q", fld.FieldType)
	}

	if _, err := s.tagGroupRepo.ReplaceRows(groupID, f.FileID, tagIDs, rows); err != nil {
		return fmt.Errorf("保存失败: %w", err)
	}
	return nil
}
