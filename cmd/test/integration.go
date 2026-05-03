package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	fileSvcPkg "github.com/XingfenD/tag_file/internal/service/file"
	searchSvcPkg "github.com/XingfenD/tag_file/internal/service/search"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func testIntegration(t *testRunner) {
	dbPath := workspace + "/.tagfile/tagfile.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "workspace DB not found, run setup first")
		os.Exit(1)
	}

	origDir, _ := os.Getwd()
	os.Chdir(workspace)
	defer os.Chdir(origDir)

	if err := storage.InitDB(".tagfile/tagfile.db"); err != nil {
		fmt.Fprintln(os.Stderr, "无法连接 workspace DB:", err)
		os.Exit(1)
	}

	fileSvc := fileSvcPkg.NewService()

	t.run("List files (workspace)", func() error {
		files, _, err := fileSvc.ListFiles(0, 0)
		if err != nil {
			return err
		}
		if len(files) < 9 {
			return fmt.Errorf("expected >=9 files from setup, got %d", len(files))
		}
		return nil
	})

	t.run("File info (cat_british_playing)", func() error {
		_, tags, err := fileSvc.GetFileInfo("cat_british_playing.mp4")
		if err != nil {
			return err
		}
		if len(tags) < 3 {
			return fmt.Errorf("expected >=3 tags, got %d", len(tags))
		}
		return nil
	})

	t.run("Tag unscanned file (reject)", func() error {
		err := fileSvc.TagFile("ghost.mp4", "猫")
		if err == nil {
			return fmt.Errorf("expected error for unregistered file")
		}
		return nil
	})

	t.run("Tag file (dup rejects)", func() error {
		err := fileSvc.TagFile("cat_british_playing.mp4", "猫")
		if err == nil {
			return fmt.Errorf("expected error for duplicate tag")
		}
		return nil
	})

	t.run("Untag and re-tag", func() error {
		if err := fileSvc.UntagFile("cat_british_playing.mp4", "玩耍"); err != nil {
			return err
		}
		fileSvc.TagFile("cat_british_playing.mp4", "玩耍")
		return nil
	})

	searchSvc := searchSvcPkg.NewService()
	t.run("Search OR: 猫 狗", func() error {
		results, err := searchSvc.SearchByTags([]string{"猫", "狗"}, nil, "or")
		if err != nil {
			return err
		}
		if len(results) < 3 {
			return fmt.Errorf("expected >=3 files, got %d", len(results))
		}
		return nil
	})
	t.run("Search AND: 猫 玩耍", func() error {
		results, err := searchSvc.SearchByTags([]string{"猫", "玩耍"}, nil, "and")
		if err != nil {
			return err
		}
		if len(results) != 2 {
			return fmt.Errorf("expected 2 (cat_british_playing + cat_ragdoll_playing), got %d", len(results))
		}
		return nil
	})
	t.run("Search AND: 英短 户外", func() error {
		results, err := searchSvc.SearchByTags([]string{"英短", "户外"}, nil, "and")
		if err != nil {
			return err
		}
		if len(results) != 1 {
			return fmt.Errorf("expected 1 (cat_british_outdoor), got %d", len(results))
		}
		return nil
	})
	t.run("Search NOT: exclude 风景", func() error {
		results, err := searchSvc.SearchByTags(nil, []string{"风景"}, "or")
		if err != nil {
			return err
		}
		for _, r := range results {
			for _, t := range r.Tags {
				if t.TagName == "风景" {
					return fmt.Errorf("expected no 风景 files")
				}
			}
		}
		return nil
	})

	t.run("Search include + exclude: 狗 --not 金毛", func() error {
		results, err := searchSvc.SearchByTags([]string{"狗"}, []string{"金毛"}, "or")
		if err != nil {
			return err
		}
		for _, r := range results {
			for _, t := range r.Tags {
				if t.TagName == "金毛" {
					return fmt.Errorf("expected no 金毛 files")
				}
			}
		}
		return nil
	})

	t.run("Imported file exists", func() error {
		_, tags, err := fileSvc.GetFileInfo("external_cat_video.mp4")
		if err != nil {
			return err
		}
		hasCat := false
		for _, t := range tags {
			if t.TagName == "猫" {
				hasCat = true
			}
		}
		if !hasCat {
			return fmt.Errorf("imported file should have 猫 tag")
		}
		return nil
	})

	t.run("Re-scan (no duplicates)", func() error {
		result, err := fileSvc.ScanDirectory(".")
		if err != nil {
			return err
		}
		if result.FilesAdded != 0 {
			return fmt.Errorf("expected 0 new files on re-scan, got %d", result.FilesAdded)
		}
		return nil
	})

	t.run("Import from external path", func() error {
		extPath, _ := filepath.Abs(filepath.Join("..", "test_imports", "external_cat_video.mp4"))
		if _, err := os.Stat(extPath); err != nil {
			return fmt.Errorf("external file not found: %s", extPath)
		}
		_, err := fileSvc.ImportFile(extPath)
		if err != nil {
			if !strings.Contains(err.Error(), "未注册") && !strings.Contains(err.Error(), "已存在") {
				return err
			}
		}
		return nil
	})
}
