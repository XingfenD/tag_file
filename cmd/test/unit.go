package main

import (
	"fmt"
	"os"

	"github.com/XingfenD/tag_file/internal/model"
	categoryRepoPkg "github.com/XingfenD/tag_file/internal/repository/category"
	fileRepoPkg "github.com/XingfenD/tag_file/internal/repository/file"
	tagRepoPkg "github.com/XingfenD/tag_file/internal/repository/tag"
	categorySvcPkg "github.com/XingfenD/tag_file/internal/service/category"
	fileSvcPkg "github.com/XingfenD/tag_file/internal/service/file"
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
		&model.File{}, &model.TagGroup{}, &model.TagGroupTag{}, &model.TagCategory{}, &model.Tag{}, &model.TagValue{}, &model.Config{},
	)

	categoryRepo := categoryRepoPkg.NewRepository()
	tagRepo := tagRepoPkg.NewRepository()

	t.run("Create category", func() error {
		_, err := categoryRepo.Create().SetName("test_cat").Exec()
		return err
	})
	t.run("Create category (dup allowed at repo level)", func() error {
		_, err := categoryRepo.Create().SetName("test_cat").Exec()
		return err
	})
	t.run("FindByName existing", func() error {
		c, err := categoryRepo.Find().ByName("test_cat").Exec()
		if err != nil || c == nil {
			return fmt.Errorf("expected category, got err=%v, nil=%v", err, c == nil)
		}
		return nil
	})
	t.run("FindByName missing", func() error {
		c, err := categoryRepo.Find().ByName("no_exist").Exec()
		if err != nil || c != nil {
			return fmt.Errorf("expected nil, got %v", c)
		}
		return nil
	})
	t.run("List categories", func() error {
		list, _, err := categoryRepo.List().Offset(0).Limit(0).Exec()
		if err != nil {
			return err
		}
		if len(list) != 2 {
			return fmt.Errorf("expected 2, got %d", len(list))
		}
		return nil
	})

	var catID uint
	if c, _ := categoryRepo.Find().ByName("test_cat").Exec(); c != nil {
		catID = c.CategoryID
	}
	t.run("Create tag", func() error {
		_, err := tagRepo.Create().SetCategoryID(catID).SetName("tag1").Exec()
		return err
	})
	t.run("Create tag (dup allowed at repo level)", func() error {
		_, err := tagRepo.Create().SetCategoryID(catID).SetName("tag1").Exec()
		return err
	})
	t.run("FindByNameAndCategory", func() error {
		tags, err := tagRepo.Find().ByNameAndCategory("tag1", catID).All()
		if err != nil || len(tags) == 0 {
			return fmt.Errorf("expected tag, got nil")
		}
		return nil
	})
	t.run("List tags", func() error {
		list, _, err := tagRepo.List().Offset(0).Limit(0).Exec()
		if err != nil {
			return err
		}
		if len(list) != 2 {
			return fmt.Errorf("expected 2, got %d", len(list))
		}
		return nil
	})
	t.run("DeleteByName", func() error {
		if _, err := tagRepo.Delete().ByName("tag1").ExecCount(); err != nil {
			return err
		}
		list, _, _ := tagRepo.List().Offset(0).Limit(0).Exec()
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
	t.run("AddTag dup (allowed under different parent)", func() error {
		_, err := tagSvc.AddTag("service_cat", "svc_tag", "", "svc_tag")
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

	t.run("Tag file by ID (same name in different categories)", func() error {
		cat1, err := catSvc.AddCategory("cat_a", "")
		if err != nil {
			return err
		}
		cat2, err := catSvc.AddCategory("cat_b", "")
		if err != nil {
			return err
		}
		t1, err := tagRepo.Create().SetCategoryID(cat1.CategoryID).SetName("same").Exec()
		if err != nil {
			return err
		}
		t2, err := tagRepo.Create().SetCategoryID(cat2.CategoryID).SetName("same").Exec()
		if err != nil {
			return err
		}

		fileRepo := fileRepoPkg.NewRepository()
		_, err = fileRepo.Create().SetName("test.jpg").SetPath("test.jpg").SetSize(100).SetType("image/jpeg").Exec()
		if err != nil {
			return err
		}

		fileSvc := fileSvcPkg.NewService()
		if err := fileSvc.TagFile("test.jpg", t2.TagID, ""); err != nil {
			return err
		}

		_, tags, _, err := fileSvc.GetFileInfo("test.jpg")
		if err != nil {
			return err
		}

		for _, t := range tags {
			if t.TagID == t1.TagID {
				return fmt.Errorf("wrong tag associated: tag_id=%d (category %s) instead of tag_id=%d (category %s)", t1.TagID, cat1.CategoryName, t2.TagID, cat2.CategoryName)
			}
		}
		for _, t := range tags {
			if t.TagID == t2.TagID {
				return nil
			}
		}
		return fmt.Errorf("expected tag_id=%d not found in file tags", t2.TagID)
	})
}
