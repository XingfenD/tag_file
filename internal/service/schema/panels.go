package schema

import (
	"fmt"

	"github.com/XingfenD/tag_file/internal/model"
)

// OptionView is one candidate of a field: Value for single (dict value),
// TagID+Name for multi (child tag).
type OptionView struct {
	Value string `json:"value"`
	TagID uint   `json:"tag_id,omitempty"`
	Name  string `json:"name,omitempty"`
}

// FieldView is one panel row. Value carries single's current dict value
// ('' = unset) and bool's 'true'/'false'; Checked carries multi's current
// child tag ids. BindOK=false means the bound tag no longer exists.
type FieldView struct {
	FieldID uint         `json:"field_id"`
	Label   string       `json:"label"`
	Type    string       `json:"type"`
	Widget  string       `json:"widget"`
	BindOK  bool         `json:"bind_ok"`
	Options []OptionView `json:"options"`
	Value   string       `json:"value"`
	Checked []uint       `json:"checked"`
}

type SchemaPanel struct {
	SchemaID uint        `json:"schema_id"`
	Name     string      `json:"name"`
	Fields   []FieldView `json:"fields"`
}

// registeredFile resolves the path to a row, failing like file.TagFile does.
func (s *Service) registeredFile(filePath string) (*model.File, error) {
	f, err := s.fileRepo.Find().ByPath(filePath).Exec()
	if err != nil {
		return nil, fmt.Errorf("查询文件失败: %w", err)
	}
	if f == nil {
		return nil, fmt.Errorf("文件 '%s' 未注册，请先 scan", filePath)
	}
	return f, nil
}

// defaultGroupID returns the file's default group, creating it on demand —
// same behaviour as file.Service.TagFile.
func (s *Service) defaultGroupID(fileID uint) (uint, error) {
	g, err := s.tagGroupRepo.FindDefaultGroup(fileID)
	if err == nil && g != nil {
		return g.GroupID, nil
	}
	ng := &model.TagGroup{FileID: fileID, GroupName: "default", IsDefault: true}
	if err := s.tagGroupRepo.CreateGroup(ng); err != nil {
		return 0, fmt.Errorf("获取默认标签组失败: %w", err)
	}
	return ng.GroupID, nil
}

// PanelsForFile renders every schema as a workbench panel for one file.
// A schema with no fields yields no panel; a stale binding yields a panel
// row with BindOK=false instead of an error.
func (s *Service) PanelsForFile(filePath string) ([]SchemaPanel, error) {
	f, err := s.registeredFile(filePath)
	if err != nil {
		return nil, err
	}
	groupID, err := s.defaultGroupID(f.FileID)
	if err != nil {
		return nil, err
	}
	schemas, err := s.repo.ListSchemas()
	if err != nil {
		return nil, fmt.Errorf("查询场景失败: %w", err)
	}
	panels := make([]SchemaPanel, 0, len(schemas))
	for _, sc := range schemas {
		if len(sc.Fields) == 0 {
			continue
		}
		p := SchemaPanel{SchemaID: sc.SchemaID, Name: sc.SchemaName}
		for _, fld := range sc.Fields {
			v, err := s.fieldView(fld, groupID)
			if err != nil {
				return nil, err
			}
			p.Fields = append(p.Fields, v)
		}
		panels = append(panels, p)
	}
	return panels, nil
}

func (s *Service) fieldView(fld model.TagSchemaField, groupID uint) (FieldView, error) {
	v := FieldView{FieldID: fld.FieldID, Label: fld.Label, Type: fld.FieldType, Widget: fld.Widget}
	tag, err := s.tagRepo.Find().ByID(fld.BindTagID).Exec()
	if err != nil {
		return v, fmt.Errorf("查询绑定标签失败: %w", err)
	}
	if tag == nil {
		v.BindOK = false
		return v, nil
	}
	v.BindOK = true

	switch fld.FieldType {
	case "single":
		vals, err := s.tagRepo.ListValues(fld.BindTagID)
		if err != nil {
			return v, fmt.Errorf("查询标签字典失败: %w", err)
		}
		for _, tv := range vals {
			v.Options = append(v.Options, OptionView{Value: tv.Value})
		}
		rows, err := s.tagGroupRepo.GetRowsByTags(groupID, []uint{fld.BindTagID})
		if err != nil {
			return v, err
		}
		if len(rows) > 0 {
			v.Value = rows[len(rows)-1].TagValue // rows ordered by id: newest is canonical
		}
	case "bool":
		rows, err := s.tagGroupRepo.GetRowsByTags(groupID, []uint{fld.BindTagID})
		if err != nil {
			return v, err
		}
		v.Value = "false"
		if len(rows) > 0 {
			v.Value = "true"
		}
	case "multi":
		kids, err := s.tagRepo.ChildIDs(fld.BindTagID)
		if err != nil {
			return v, fmt.Errorf("查询子标签失败: %w", err)
		}
		if len(kids) > 0 {
			tags, err := s.tagRepo.FindByIDs(kids)
			if err != nil {
				return v, err
			}
			byID := make(map[uint]string, len(tags))
			for _, kt := range tags {
				byID[kt.TagID] = kt.TagName
			}
			for _, id := range kids {
				if name, ok := byID[id]; ok {
					v.Options = append(v.Options, OptionView{TagID: id, Name: name})
				}
			}
			rows, err := s.tagGroupRepo.GetRowsByTags(groupID, kids)
			if err != nil {
				return v, err
			}
			seen := make(map[uint]bool, len(rows))
			for _, r := range rows {
				if !seen[r.TagID] {
					seen[r.TagID] = true
					v.Checked = append(v.Checked, r.TagID)
				}
			}
		}
	}
	return v, nil
}
