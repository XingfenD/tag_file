package category_test

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
		&model.File{}, &model.TagGroup{}, &model.TagGroupTag{}, &model.TagCategory{}, &model.Tag{}, &model.TagValue{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return func() { os.Remove(dbPath) }
}

func TestCategoryService_AddCategory_Duplicate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	svc := category.NewService()
	_, err := svc.AddCategory("类型1", "")
	if err != nil {
		t.Fatalf("first add: %v", err)
	}
	_, err = svc.AddCategory("类型1", "")
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestCategoryService_ListCategories(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	svc := category.NewService()
	svc.AddCategory("A", "")
	svc.AddCategory("B", "")

	list, _, err := svc.ListCategories(0, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2, got %d", len(list))
	}
}

func TestCategoryService_RemoveCategory_RejectsWithTags(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	catSvc := category.NewService()
	catSvc.AddCategory("测试", "")

	tagSvc := tag.NewService()
	tagSvc.AddTag("测试", "标签1", "", "")

	err := catSvc.RemoveCategory("测试")
	if err == nil {
		t.Fatal("expected error removing category with tags")
	}
}
