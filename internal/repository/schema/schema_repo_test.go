package schema_test

import (
	"os"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/repository/schema"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()
	dbPath := ".schema_repo_test.db"
	os.Remove(dbPath)
	if err := storage.InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := storage.GetDB().AutoMigrate(
		&model.File{}, &model.TagGroup{}, &model.TagGroupTag{}, &model.TagCategory{},
		&model.Tag{}, &model.TagValue{}, &model.TagSchema{}, &model.TagSchemaField{},
		&model.Config{}, &model.AITask{}, &model.AISuggestion{}, &model.AIFrameOutput{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return func() { os.Remove(dbPath) }
}

func TestSchemaRepoCRUD(t *testing.T) {
	defer setupTestDB(t)()
	repo := schema.NewRepository()

	s := &model.TagSchema{SchemaName: "评级", SortOrder: 2}
	if err := repo.CreateSchema(s); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	f1 := &model.TagSchemaField{SchemaID: s.SchemaID, Label: "是否收藏", FieldType: "bool", BindTagID: 11, SortOrder: 1}
	f2 := &model.TagSchemaField{SchemaID: s.SchemaID, Label: "星级", FieldType: "single", Widget: "stars", BindTagID: 12, SortOrder: 0}
	if err := repo.CreateField(f1); err != nil {
		t.Fatalf("create field 1: %v", err)
	}
	if err := repo.CreateField(f2); err != nil {
		t.Fatalf("create field 2: %v", err)
	}

	list, err := repo.ListSchemas()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || len(list[0].Fields) != 2 {
		t.Fatalf("want 1 schema with 2 preloaded fields, got %d/%d", len(list), len(list[0].Fields))
	}
	// fields 按 sort_order 升序：f2(0) 在 f1(1) 前
	if list[0].Fields[0].FieldID != f2.FieldID || list[0].Fields[1].FieldID != f1.FieldID {
		t.Errorf("fields not sorted by sort_order: %d,%d", list[0].Fields[0].FieldID, list[0].Fields[1].FieldID)
	}

	got, err := repo.GetSchema(s.SchemaID)
	if err != nil || got == nil || got.SchemaName != "评级" {
		t.Fatalf("get schema: %v %+v", err, got)
	}

	// FieldByBindTagID 命中与未命中
	by, err := repo.FieldByBindTagID(12)
	if err != nil || by == nil || by.FieldID != f2.FieldID {
		t.Fatalf("FieldByBindTagID miss: %v %+v", err, by)
	}
	none, err := repo.FieldByBindTagID(999)
	if err != nil || none != nil {
		t.Fatalf("FieldByBindTagID should return nil,nil for unused tag, got %v %+v", err, none)
	}

	// UpdateField 改 label/widget
	f2.Label = "星级2"
	f2.Widget = "select"
	if err := repo.UpdateField(f2); err != nil {
		t.Fatalf("update field: %v", err)
	}
	fs, err := repo.ListFields(s.SchemaID)
	if err != nil || len(fs) != 2 || fs[0].Widget != "select" {
		t.Fatalf("ListFields after update: %v %+v", err, fs)
	}

	// DeleteSchema 连带删 fields（tag_schema_fields 表清空）
	if err := repo.DeleteSchema(s.SchemaID); err != nil {
		t.Fatalf("delete schema: %v", err)
	}
	var cnt int64
	if err := storage.GetDB().Model(&model.TagSchemaField{}).Count(&cnt).Error; err != nil {
		t.Fatal(err)
	}
	if cnt != 0 {
		t.Errorf("DeleteSchema left %d orphan fields", cnt)
	}
	if g, _ := repo.GetSchema(s.SchemaID); g != nil {
		t.Error("schema still present after delete")
	}
}
