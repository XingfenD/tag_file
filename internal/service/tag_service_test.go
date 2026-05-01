package service_test

import (
	"testing"

	"github.com/XingfenD/tag_file/internal/service"
)

func TestTagService_AddTag_RejectsDuplicate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	catSvc := service.NewCategoryService()
	catSvc.AddCategory("角色", "")

	tagSvc := service.NewTagService()
	_, err := tagSvc.AddTag("角色", "cos", "")
	if err != nil {
		t.Fatalf("first add: %v", err)
	}
	_, err = tagSvc.AddTag("角色", "cos", "")
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestTagService_AddTag_MissingCategory(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	tagSvc := service.NewTagService()
	_, err := tagSvc.AddTag("不存在", "标签", "")
	if err == nil {
		t.Fatal("expected error for missing category")
	}
}

func TestTagService_RemoveTag(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	catSvc := service.NewCategoryService()
	catSvc.AddCategory("类型", "")

	tagSvc := service.NewTagService()
	tagSvc.AddTag("类型", "tmp", "")

	_, err := tagSvc.RemoveTag("tmp")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}

	_, err = tagSvc.RemoveTag("tmp")
	if err == nil {
		t.Fatal("expected error removing non-existing tag")
	}
}
