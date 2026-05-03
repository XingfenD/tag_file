package main

import (
	"fmt"
	"os"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func testAttrAndConfig(t *testRunner) {
	os.Remove(testDB)
	if err := storage.InitDB(testDB); err != nil {
		fmt.Fprintln(os.Stderr, "DB init failed:", err)
		os.Exit(1)
	}
	storage.GetDB().AutoMigrate(
		&model.File{}, &model.FileTag{}, &model.TagCategory{}, &model.Tag{},
		&model.Attribute{}, &model.AttributeValue{}, &model.FileAttribute{}, &model.Config{},
	)

	tagSvc := service.NewTagService()
	catSvc := service.NewCategoryService()

	catSvc.AddCategory("test_cat", "")
	tagSvc.AddTag("test_cat", "test_tag", "")

	attrSvc := service.NewAttrService()
	t.run("Add attribute to tag", func() error {
		_, err := attrSvc.AddAttribute("test_tag", "品种", "")
		return err
	})
	t.run("Add duplicate attribute (reject)", func() error {
		_, err := attrSvc.AddAttribute("test_tag", "品种", "")
		if err == nil {
			return fmt.Errorf("expected duplicate error")
		}
		return nil
	})
	t.run("Add attribute to missing tag", func() error {
		_, err := attrSvc.AddAttribute("no_tag", "颜色", "")
		if err == nil {
			return fmt.Errorf("expected missing tag error")
		}
		return nil
	})
	t.run("List attributes by tag", func() error {
		attrs, err := attrSvc.ListAttributes("test_tag")
		if err != nil {
			return err
		}
		if len(attrs) != 1 {
			return fmt.Errorf("expected 1 attribute, got %d", len(attrs))
		}
		return nil
	})
	t.run("List all attributes", func() error {
		attrs, err := attrSvc.ListAttributes("")
		if err != nil {
			return err
		}
		if len(attrs) != 1 {
			return fmt.Errorf("expected 1 attribute, got %d", len(attrs))
		}
		return nil
	})

	var attrID uint
	if attrs, _ := attrSvc.ListAttributes(""); len(attrs) > 0 {
		attrID = attrs[0].AttributeID
	}

	t.run("Add attribute value", func() error {
		_, err := attrSvc.AddValue(attrID, "英短")
		return err
	})
	t.run("Add duplicate value (reject)", func() error {
		_, err := attrSvc.AddValue(attrID, "英短")
		if err == nil {
			return fmt.Errorf("expected duplicate error")
		}
		return nil
	})
	t.run("List values", func() error {
		vals, err := attrSvc.ListValues(attrID)
		if err != nil {
			return err
		}
		if len(vals) != 1 {
			return fmt.Errorf("expected 1 value, got %d", len(vals))
		}
		return nil
	})
	t.run("Remove attribute value", func() error {
		var valID uint
		if vals, _ := attrSvc.ListValues(attrID); len(vals) > 0 {
			valID = vals[0].ValueID
		}
		err := attrSvc.RemoveValue(valID)
		return err
	})
	t.run("Remove attribute (no values)", func() error {
		return attrSvc.RemoveAttribute(attrID)
	})

	cfgSvc := service.NewConfigService()
	t.run("Set config", func() error {
		return cfgSvc.Set("import_dir", "./media")
	})
	t.run("Set config (overwrite)", func() error {
		return cfgSvc.Set("import_dir", "./custom_dir")
	})
	t.run("Get config", func() error {
		cfg, err := cfgSvc.Get("import_dir")
		if err != nil {
			return err
		}
		if cfg.ConfigValue != "./custom_dir" {
			return fmt.Errorf("expected ./custom_dir, got %s", cfg.ConfigValue)
		}
		return nil
	})
	t.run("Get missing config", func() error {
		_, err := cfgSvc.Get("no_such_key")
		if err == nil {
			return fmt.Errorf("expected error for missing config")
		}
		return nil
	})
	t.run("List configs", func() error {
		cfgs, err := cfgSvc.List()
		if err != nil {
			return err
		}
		if len(cfgs) != 1 {
			return fmt.Errorf("expected 1 config, got %d", len(cfgs))
		}
		return nil
	})
	t.run("Delete config", func() error {
		return cfgSvc.Delete("import_dir")
	})
	t.run("Set empty key (reject)", func() error {
		err := cfgSvc.Set("", "val")
		if err == nil {
			return fmt.Errorf("expected error for empty key")
		}
		return nil
	})
}
