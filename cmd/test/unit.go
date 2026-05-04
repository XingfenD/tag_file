package main

import (
	"fmt"
	"os"

	"github.com/XingfenD/tag_file/internal/model"
	categoryRepoPkg "github.com/XingfenD/tag_file/internal/repository/category"
	tagRepoPkg "github.com/XingfenD/tag_file/internal/repository/tag"
	categorySvcPkg "github.com/XingfenD/tag_file/internal/service/category"
	tagSvcPkg "github.com/XingfenD/tag_file/internal/service/tag"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func testUnitTests(t *testRunner) {
	os.Remove(testDB)
	if err := storage.InitDB(testDB); err != nil {
		fmt.Fprintln(os.Stderr, "数据库初始化失败:", err)
		os.Exit(1)
	}
	storage.GetDB().AutoMigrate(
		&model.File{}, &model.FileTag{}, &model.TagCategory{}, &model.Tag{}, &model.TagValue{}, &model.Config{},
	)

	categoryRepo := categoryRepoPkg.NewRepository()
	tagRepo := tagRepoPkg.NewRepository()

	t.run("Create category", func() error {
		return categoryRepo.Create(&model.TagCategory{CategoryName: "test_cat"})
	})
	t.run("Create category (dup allowed at repo level)", func() error {
		return categoryRepo.Create(&model.TagCategory{CategoryName: "test_cat"})
	})
	t.run("FindByName existing", func() error {
		c, err := categoryRepo.FindByName("test_cat")
		if err != nil || c == nil {
			return fmt.Errorf("expected category, got err=%v, nil=%v", err, c == nil)
		}
		return nil
	})
	t.run("FindByName missing", func() error {
		c, err := categoryRepo.FindByName("no_exist")
		if err != nil || c != nil {
			return fmt.Errorf("expected nil, got %v", c)
		}
		return nil
	})
	t.run("List categories", func() error {
		list, _, err := categoryRepo.List(0, 0)
		if err != nil {
			return err
		}
		if len(list) != 2 {
			return fmt.Errorf("expected 2, got %d", len(list))
		}
		return nil
	})

	var catID uint
	if c, _ := categoryRepo.FindByName("test_cat"); c != nil {
		catID = c.CategoryID
	}
	t.run("Create tag", func() error {
		return tagRepo.Create(&model.Tag{CategoryID: catID, TagName: "tag1"})
	})
	t.run("Create tag (dup allowed at repo level)", func() error {
		return tagRepo.Create(&model.Tag{CategoryID: catID, TagName: "tag1"})
	})
	t.run("FindByNameAndCategory", func() error {
		tags, err := tagRepo.FindByNameAndCategory("tag1", catID)
		if err != nil || len(tags) == 0 {
			return fmt.Errorf("expected tag, got nil")
		}
		return nil
	})
	t.run("List tags", func() error {
		list, _, err := tagRepo.List(0, 0)
		if err != nil {
			return err
		}
		if len(list) != 2 {
			return fmt.Errorf("expected 2, got %d", len(list))
		}
		return nil
	})
	t.run("DeleteByName", func() error {
		if _, err := tagRepo.DeleteByName("tag1"); err != nil {
			return err
		}
		list, _, _ := tagRepo.List(0, 0)
		if len(list) != 0 {
			return fmt.Errorf("expected 0 after delete, got %d", len(list))
		}
		return nil
	})

	catSvc := categorySvcPkg.NewService()
	t.run("AddCategory service", func() error {
		_, err := catSvc.AddCategory("service_cat", "")
		return err
	})
	t.run("AddCategory dup (service rejects)", func() error {
		_, err := catSvc.AddCategory("service_cat", "")
		if err == nil {
			return fmt.Errorf("expected error for duplicate")
		}
		return nil
	})

	tagSvc := tagSvcPkg.NewService()
	t.run("AddTag service", func() error {
		_, err := tagSvc.AddTag("service_cat", "svc_tag", "", "")
		return err
	})
	t.run("AddTag dup (allowed)", func() error {
		_, err := tagSvc.AddTag("service_cat", "svc_tag", "", "")
		return err
	})
	t.run("AddTag missing category", func() error {
		_, err := tagSvc.AddTag("no_exist", "tag", "", "")
		if err == nil {
			return fmt.Errorf("expected error for missing category")
		}
		return nil
	})
	t.run("RemoveTag", func() error {
		_, err := tagSvc.RemoveTag("svc_tag")
		return err
	})
	t.run("RemoveCategory with no tags", func() error {
		return catSvc.RemoveCategory("service_cat")
	})
}
