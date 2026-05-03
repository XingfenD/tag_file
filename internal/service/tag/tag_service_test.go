package tag_test

import (
	"os"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/category"
	"github.com/XingfenD/tag_file/internal/service/tag"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()
	dbPath := ".tag_file_test.db"
	os.Remove(dbPath)
	if err := storage.InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := storage.GetDB().AutoMigrate(
		&model.File{}, &model.FileTag{}, &model.TagCategory{}, &model.Tag{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return func() { os.Remove(dbPath) }
}

func TestTagService_AddTag_RejectsDuplicate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	catSvc := category.NewService()
	catSvc.AddCategory("动物", "")

	tagSvc := tag.NewService()
	_, err := tagSvc.AddTag("动物", "猫", "", "")
	if err != nil {
		t.Fatalf("first add: %v", err)
	}
	_, err = tagSvc.AddTag("动物", "猫", "", "")
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestTagService_AddTag_MissingCategory(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tagSvc := tag.NewService()
	_, err := tagSvc.AddTag("不存在", "标签", "", "")
	if err == nil {
		t.Fatal("expected error for missing category")
	}
}

func TestTagService_RemoveTag(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	catSvc := category.NewService()
	catSvc.AddCategory("类型", "")

	tagSvc := tag.NewService()
	tagSvc.AddTag("类型", "tmp", "", "")

	_, err := tagSvc.RemoveTag("tmp")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}

	_, err = tagSvc.RemoveTag("tmp")
	if err == nil {
		t.Fatal("expected error removing non-existing tag")
	}
}
