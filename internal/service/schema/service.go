package schema

import (
	"fmt"

	"github.com/XingfenD/tag_file/internal/model"
	repoFile "github.com/XingfenD/tag_file/internal/repository/file"
	repoSchema "github.com/XingfenD/tag_file/internal/repository/schema"
	repoTag "github.com/XingfenD/tag_file/internal/repository/tag"
	repoTagGroup "github.com/XingfenD/tag_file/internal/repository/taggroup"
)

type Service struct {
	repo         *repoSchema.Repository
	tagRepo      *repoTag.Repository
	fileRepo     *repoFile.Repository
	tagGroupRepo *repoTagGroup.Repository
}

func NewService() *Service {
	return &Service{
		repo:         repoSchema.NewRepository(),
		tagRepo:      repoTag.NewRepository(),
		fileRepo:     repoFile.NewRepository(),
		tagGroupRepo: repoTagGroup.NewRepository(),
	}
}

// ── schema CRUD ──

func (s *Service) ListSchemas() ([]model.TagSchema, error) {
	return s.repo.ListSchemas()
}

func (s *Service) AddSchema(name, description string) (*model.TagSchema, error) {
	list, err := s.repo.ListSchemas()
	if err != nil {
		return nil, fmt.Errorf("查询场景列表失败: %w", err)
	}
	for _, x := range list {
		if x.SchemaName == name {
			return nil, fmt.Errorf("场景 %q 已存在", name)
		}
	}
	next := 0
	for _, x := range list {
		if x.SortOrder >= next {
			next = x.SortOrder + 1
		}
	}
	sch := &model.TagSchema{SchemaName: name, Description: description, SortOrder: next}
	if err := s.repo.CreateSchema(sch); err != nil {
		return nil, fmt.Errorf("创建场景失败: %w", err)
	}
	return sch, nil
}

func (s *Service) UpdateSchema(id uint, name, description string, sortOrder int) error {
	cur, err := s.repo.GetSchema(id)
	if err != nil {
		return err
	}
	if cur == nil {
		return fmt.Errorf("场景 %d 不存在", id)
	}
	list, err := s.repo.ListSchemas()
	if err != nil {
		return fmt.Errorf("查询场景列表失败: %w", err)
	}
	for _, x := range list {
		if x.SchemaID != id && x.SchemaName == name {
			return fmt.Errorf("场景 %q 已存在", name)
		}
	}
	cur.SchemaName = name
	cur.Description = description
	cur.SortOrder = sortOrder
	return s.repo.UpdateSchema(cur)
}

// RemoveSchema deletes only the definition. Materialized tag rows and the
// tag tree are deliberately left alone.
func (s *Service) RemoveSchema(id uint) error {
	return s.repo.DeleteSchema(id)
}

// ── field CRUD（校验收口在 validate.go）──

func (s *Service) AddField(schemaID uint, label, fieldType, widget string, bindTagID uint) (*model.TagSchemaField, error) {
	sch, err := s.repo.GetSchema(schemaID)
	if err != nil {
		return nil, err
	}
	if sch == nil {
		return nil, fmt.Errorf("场景 %d 不存在", schemaID)
	}
	if err := s.checkField(label, fieldType, widget, bindTagID, 0); err != nil {
		return nil, err
	}
	next := 0
	for _, f := range sch.Fields {
		if f.SortOrder >= next {
			next = f.SortOrder + 1
		}
	}
	f := &model.TagSchemaField{
		SchemaID: schemaID, Label: label, FieldType: fieldType,
		Widget: widget, BindTagID: bindTagID, SortOrder: next,
	}
	if err := s.repo.CreateField(f); err != nil {
		return nil, fmt.Errorf("创建字段失败: %w", err)
	}
	return f, nil
}

func (s *Service) UpdateField(fieldID uint, label, widget string, bindTagID uint, sortOrder int) error {
	f, err := s.repo.GetField(fieldID)
	if err != nil {
		return err
	}
	if f == nil {
		return fmt.Errorf("字段 %d 不存在", fieldID)
	}
	if err := s.checkField(label, f.FieldType, widget, bindTagID, fieldID); err != nil {
		return err
	}
	f.Label = label
	f.Widget = widget
	f.BindTagID = bindTagID
	f.SortOrder = sortOrder
	return s.repo.UpdateField(f)
}

func (s *Service) RemoveField(fieldID uint) error {
	return s.repo.DeleteField(fieldID)
}

// ChildTagIDs returns the bound parent tag's direct children (the parent
// itself excluded) in tag_id order.
func (s *Service) ChildTagIDs(parentID uint) ([]uint, error) {
	return s.tagRepo.ChildIDs(parentID)
}
