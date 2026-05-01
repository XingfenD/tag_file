package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/repository"
	"github.com/XingfenD/tag_file/internal/service"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func main() {
	os.Remove(".tag_file_test.db")

	if err := storage.InitDB(".tag_file_test.db"); err != nil {
		fmt.Fprintln(os.Stderr, "数据库初始化失败:", err)
		os.Exit(1)
	}

	if err := storage.GetDB().AutoMigrate(
		&model.File{},
		&model.FileTag{},
		&model.TagCategory{},
		&model.Tag{},
		&model.Attribute{},
		&model.AttributeValue{},
		&model.FileAttribute{},
		&model.Config{},
	); err != nil {
		fmt.Fprintln(os.Stderr, "AutoMigrate 失败:", err)
		os.Exit(1)
	}

	failed := 0
	passed := 0

	run := func(name string, fn func() error) {
		if err := fn(); err != nil {
			fmt.Printf("  FAIL %s: %v\n", name, err)
			failed++
		} else {
			fmt.Printf("  PASS %s\n", name)
			passed++
		}
	}

	section := func(title string) {
		fmt.Printf("\n%s\n%s\n", title, strings.Repeat("=", len(title)))
	}

	section("CategoryRepository")
	categoryRepo := repository.NewCategoryRepository()

	run("Create category", func() error {
		return categoryRepo.Create(&model.TagCategory{CategoryName: "着装类型"})
	})
	run("Create category (dup name)", func() error {
		return categoryRepo.Create(&model.TagCategory{CategoryName: "着装类型"})
	})
	run("FindByName existing", func() error {
		c, err := categoryRepo.FindByName("着装类型")
		if err != nil {
			return err
		}
		if c == nil {
			return fmt.Errorf("expected category, got nil")
		}
		if c.CategoryName != "着装类型" {
			return fmt.Errorf("expected 着装类型, got %s", c.CategoryName)
		}
		return nil
	})
	run("FindByName missing", func() error {
		c, err := categoryRepo.FindByName("不存在")
		if err != nil {
			return err
		}
		if c != nil {
			return fmt.Errorf("expected nil, got %v", c)
		}
		return nil
	})
	run("List", func() error {
		list, err := categoryRepo.List()
		if err != nil {
			return err
		}
		if len(list) != 2 {
			return fmt.Errorf("expected 2 categories (dup allowed at repo level), got %d", len(list))
		}
		return nil
	})

	section("TagRepository")
	tagRepo := repository.NewTagRepository()

	var catID uint
	if c, _ := categoryRepo.FindByName("着装类型"); c != nil {
		catID = c.CategoryID
	}

	run("Create tag", func() error {
		return tagRepo.Create(&model.Tag{CategoryID: catID, TagName: "cos"})
	})
	run("Create tag (dup in same category)", func() error {
		return tagRepo.Create(&model.Tag{CategoryID: catID, TagName: "cos"})
	})
	run("FindByNameAndCategory existing", func() error {
		t, err := tagRepo.FindByNameAndCategory("cos", catID)
		if err != nil {
			return err
		}
		if t == nil {
			return fmt.Errorf("expected tag, got nil")
		}
		if t.TagName != "cos" {
			return fmt.Errorf("expected cos, got %s", t.TagName)
		}
		return nil
	})
	run("FindByNameAndCategory missing", func() error {
		t, err := tagRepo.FindByNameAndCategory("notag", catID)
		if err != nil {
			return err
		}
		if t != nil {
			return fmt.Errorf("expected nil, got %v", t)
		}
		return nil
	})
	run("List all", func() error {
		list, err := tagRepo.List()
		if err != nil {
			return err
		}
		if len(list) != 2 {
			return fmt.Errorf("expected 2 tags (dup allowed at repo level), got %d", len(list))
		}
		return nil
	})
	run("DeleteByName", func() error {
		if _, err := tagRepo.DeleteByName("cos"); err != nil {
			return err
		}
		list, _ := tagRepo.List()
		if len(list) != 0 {
			return fmt.Errorf("expected 0 tags after delete, got %d", len(list))
		}
		return nil
	})

	section("CategoryService")
	catSvc := service.NewCategoryService()

	run("AddCategory", func() error {
		_, err := catSvc.AddCategory("视频内容", "")
		return err
	})
	run("AddCategory (dup)", func() error {
		_, err := catSvc.AddCategory("视频内容", "")
		if err == nil {
			return fmt.Errorf("expected error for duplicate")
		}
		return nil
	})
	run("ListCategories", func() error {
		list, err := catSvc.ListCategories()
		if err != nil {
			return err
		}
		if len(list) != 3 {
			return fmt.Errorf("expected 3 categories (2 from repo test + 1 from service), got %d", len(list))
		}
		return nil
	})

	section("TagService")
	tagSvc := service.NewTagService()

	run("AddTag", func() error {
		_, err := tagSvc.AddTag("视频内容", "跳舞", "")
		return err
	})
	run("AddTag to non-existent category", func() error {
		_, err := tagSvc.AddTag("不存在", "测试", "")
		if err == nil {
			return fmt.Errorf("expected error for missing category")
		}
		return nil
	})
	run("AddTag (dup in same category)", func() error {
		_, err := tagSvc.AddTag("视频内容", "跳舞", "")
		if err == nil {
			return fmt.Errorf("expected error for duplicate tag")
		}
		return nil
	})
	run("ListTags by category", func() error {
		list, err := tagSvc.ListTags("视频内容")
		if err != nil {
			return err
		}
		if len(list) != 1 {
			return fmt.Errorf("expected 1 tag, got %d", len(list))
		}
		return nil
	})
	run("ListTags all", func() error {
		list, err := tagSvc.ListTags("")
		if err != nil {
			return err
		}
		if len(list) != 1 {
			return fmt.Errorf("expected 1 tag, got %d", len(list))
		}
		return nil
	})
	run("RemoveTag", func() error {
		_, err := tagSvc.RemoveTag("跳舞")
		return err
	})
	run("RemoveTag missing", func() error {
		_, err := tagSvc.RemoveTag("跳舞")
		if err == nil {
			return fmt.Errorf("expected error for missing tag")
		}
		return nil
	})
	run("RemoveCategory with no tags", func() error {
		return catSvc.RemoveCategory("视频内容")
	})
	run("RemoveCategory with tags", func() error {
		if err := categoryRepo.Create(&model.TagCategory{CategoryName: "测试分类"}); err != nil {
			return err
		}
		c, _ := categoryRepo.FindByName("测试分类")
		tagRepo.Create(&model.Tag{CategoryID: c.CategoryID, TagName: "test"})
		err := catSvc.RemoveCategory("测试分类")
		if err == nil {
			return fmt.Errorf("expected error for category with tags")
		}
		return nil
	})

	os.Remove(".tag_file_test.db")

	fmt.Printf("\n")
	fmt.Printf("结果: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
