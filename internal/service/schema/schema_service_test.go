package schema_test

import (
	"os"
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/schema"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()
	dbPath := ".schema_svc_test.db"
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

func seedTag(t *testing.T, name string, parentID *uint) uint {
	t.Helper()
	cat := model.TagCategory{CategoryName: "c-" + name}
	if err := storage.GetDB().Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	tag := model.Tag{CategoryID: cat.CategoryID, TagName: name, ParentID: parentID}
	if err := storage.GetDB().Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	return tag.TagID
}

func TestAddSchemaDuplicateName(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()

	if _, err := svc.AddSchema("评级", ""); err != nil {
		t.Fatalf("first add: %v", err)
	}
	_, err := svc.AddSchema("评级", "")
	if err == nil || !strings.Contains(err.Error(), "已存在") {
		t.Fatalf("duplicate name want 已存在 error, got %v", err)
	}
}

func TestAddFieldValidation(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	s, err := svc.AddSchema("评级", "")
	if err != nil {
		t.Fatal(err)
	}
	star := seedTag(t, "星级", nil)

	if _, err := svc.AddField(s.SchemaID, "x", "range", "", star); err == nil {
		t.Error("unknown field type must be rejected")
	}
	if _, err := svc.AddField(s.SchemaID, "x", "single", "", 999999); err == nil {
		t.Error("bind to nonexistent tag must be rejected")
	}
	boolTag := seedTag(t, "收藏", nil)
	if _, err := svc.AddField(s.SchemaID, "x", "bool", "stars", boolTag); err == nil {
		t.Error("widget stars on bool field must be rejected")
	}
	f, err := svc.AddField(s.SchemaID, "星级", "single", "stars", star)
	if err != nil || f.FieldID == 0 {
		t.Fatalf("valid single field must pass: %v", err)
	}
}

func TestFieldBindUniqueness(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	s, _ := svc.AddSchema("评级", "")
	star := seedTag(t, "星级", nil)
	other := seedTag(t, "其他", nil)

	fa, err := svc.AddField(s.SchemaID, "A", "single", "", star)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.AddField(s.SchemaID, "B", "bool", "", star)
	if err == nil || !strings.Contains(err.Error(), `字段 "A"`) {
		t.Fatalf("second bind of same tag must name the occupying field, got %v", err)
	}

	// UpdateField 改绑占用中自己的标签放行；改绑他人在用的拒绝
	if err := svc.UpdateField(fa.FieldID, "A2", "stars", star, 0); err != nil {
		t.Errorf("re-save own binding must pass: %v", err)
	}
	fb, err := svc.AddField(s.SchemaID, "B", "bool", "", other)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateField(fb.FieldID, "B", "", star, 1); err == nil {
		t.Error("rebind onto occupied tag must fail")
	}
}

func TestRemoveSchemaLeavesTags(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	s, _ := svc.AddSchema("评级", "")
	star := seedTag(t, "星级", nil)
	if err := storage.GetDB().Create(&model.TagValue{TagID: star, Value: "4"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddField(s.SchemaID, "星级", "single", "stars", star); err != nil {
		t.Fatal(err)
	}

	if err := svc.RemoveSchema(s.SchemaID); err != nil {
		t.Fatalf("remove schema: %v", err)
	}
	var tags, vals int64
	storage.GetDB().Model(&model.Tag{}).Where("tag_id = ?", star).Count(&tags)
	storage.GetDB().Model(&model.TagValue{}).Where("tag_id = ?", star).Count(&vals)
	if tags != 1 || vals != 1 {
		t.Errorf("RemoveSchema touched tag tree: tags=%d values=%d, want 1/1", tags, vals)
	}
}

func TestSchemaNameBlankRejected(t *testing.T) {
	defer setupTestDB(t)()
	svc := schema.NewService()
	if _, err := svc.AddSchema("   ", ""); err == nil {
		t.Error("blank create name must be rejected")
	}
	s, err := svc.AddSchema("评级", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateSchema(s.SchemaID, "  ", "", 0); err == nil {
		t.Error("blank update name must be rejected")
	}
	// 合法改名首尾空白被修剪
	if err := svc.UpdateSchema(s.SchemaID, " 评级 ", "", 0); err != nil {
		t.Fatalf("padded valid name: %v", err)
	}
	if got, _ := svc.ListSchemas(); len(got) != 1 || got[0].SchemaName != "评级" {
		t.Errorf("name not trimmed: %+v", got)
	}
}
