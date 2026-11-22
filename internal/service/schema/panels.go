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
// row with BindOK=false instead of an error. Everything the panels show is
// read in batched queries (tags, dicts, child sets, the file's rows), and a
// file with nothing to render is never touched — no schemas means no
// default-group creation on a read path.
func (s *Service) PanelsForFile(filePath string) ([]SchemaPanel, error) {
	schemas, err := s.repo.ListSchemas()
	if err != nil {
		return nil, fmt.Errorf("查询场景失败: %w", err)
	}
	var bindIDs, multiIDs, singleIDs []uint
	for _, sc := range schemas {
		for _, fld := range sc.Fields {
			bindIDs = append(bindIDs, fld.BindTagID)
			switch fld.FieldType {
			case "multi":
				multiIDs = append(multiIDs, fld.BindTagID)
			case "single":
				singleIDs = append(singleIDs, fld.BindTagID)
			}
		}
	}
	if len(bindIDs) == 0 {
		return []SchemaPanel{}, nil
	}
	f, err := s.registeredFile(filePath)
	if err != nil {
		return nil, err
	}
	groupID, err := s.defaultGroupID(f.FileID)
	if err != nil {
		return nil, err
	}

	tags, err := s.tagRepo.FindByIDs(bindIDs)
	if err != nil {
		return nil, fmt.Errorf("查询绑定标签失败: %w", err)
	}
	tagOK := make(map[uint]bool, len(bindIDs))
	for _, t := range tags {
		tagOK[t.TagID] = true
	}
	kidsByParent, err := s.tagRepo.ChildrenMap(multiIDs)
	if err != nil {
		return nil, fmt.Errorf("查询子标签失败: %w", err)
	}
	var kidIDs []uint
	for _, kids := range kidsByParent {
		kidIDs = append(kidIDs, kids...)
	}
	kidName := map[uint]string{}
	if len(kidIDs) > 0 {
		kidTags, err := s.tagRepo.FindByIDs(kidIDs)
		if err != nil {
			return nil, err
		}
		for _, kt := range kidTags {
			kidName[kt.TagID] = kt.TagName
		}
	}
	dict, err := s.tagRepo.ListValuesByTagIDs(singleIDs)
	if err != nil {
		return nil, fmt.Errorf("查询标签字典失败: %w", err)
	}
	union := append(append([]uint{}, bindIDs...), kidIDs...)
	rows, err := s.tagGroupRepo.GetRowsByTags(groupID, union)
	if err != nil {
		return nil, err
	}
	rowsByTag := make(map[uint][]model.TagGroupTag, len(rows))
	for _, r := range rows {
		rowsByTag[r.TagID] = append(rowsByTag[r.TagID], r)
	}

	panels := make([]SchemaPanel, 0, len(schemas))
	for _, sc := range schemas {
		if len(sc.Fields) == 0 {
			continue
		}
		p := SchemaPanel{SchemaID: sc.SchemaID, Name: sc.SchemaName}
		for _, fld := range sc.Fields {
			p.Fields = append(p.Fields, viewOfField(fld, tagOK, kidsByParent, kidName, dict, rowsByTag))
		}
		panels = append(panels, p)
	}
	return panels, nil
}

// viewOfField assembles one FieldView from the batched reads; it never
// touches the database.
func viewOfField(fld model.TagSchemaField, tagOK map[uint]bool, kidsByParent map[uint][]uint,
	kidName map[uint]string, dict map[uint][]model.TagValue, rowsByTag map[uint][]model.TagGroupTag) FieldView {
	v := FieldView{FieldID: fld.FieldID, Label: fld.Label, Type: fld.FieldType, Widget: fld.Widget}
	if !tagOK[fld.BindTagID] {
		return v // BindOK=false: stale binding, no options and no value to show
	}
	v.BindOK = true
	switch fld.FieldType {
	case "single":
		for _, tv := range dict[fld.BindTagID] {
			v.Options = append(v.Options, OptionView{Value: tv.Value})
		}
		if rows := rowsByTag[fld.BindTagID]; len(rows) > 0 {
			v.Value = rows[len(rows)-1].TagValue // rows ordered by id: newest is canonical
		}
	case "bool":
		v.Value = "false"
		if len(rowsByTag[fld.BindTagID]) > 0 {
			v.Value = "true"
		}
	case "multi":
		var checked []uint
		for _, id := range kidsByParent[fld.BindTagID] {
			name, ok := kidName[id]
			if !ok {
				continue
			}
			v.Options = append(v.Options, OptionView{TagID: id, Name: name})
			if len(rowsByTag[id]) > 0 {
				checked = append(checked, id)
			}
		}
		v.Checked = checked
	}
	return v
}
