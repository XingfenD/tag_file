package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/XingfenD/tag_file/internal/service"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func testIntegration(t *testRunner) {
	workspaceDB := workspace + "/.tag_file.db"
	if _, err := os.Stat(workspaceDB); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "workspace DB not found, run setup first")
		os.Exit(1)
	}
	if err := storage.InitDB(workspaceDB); err != nil {
		fmt.Fprintln(os.Stderr, "无法连接 workspace DB:", err)
		os.Exit(1)
	}

	fileSvc := service.NewFileService()

	t.run("List files (workspace)", func() error {
		files, err := fileSvc.ListFiles()
		if err != nil {
			return err
		}
		if len(files) < 9 {
			return fmt.Errorf("expected >=9 files from setup, got %d", len(files))
		}
		return nil
	})

	t.run("File info (cos_remu_dance)", func() error {
		_, tags, err := fileSvc.GetFileInfo("cos_remu_dance.mp4")
		if err != nil {
			return err
		}
		if len(tags) < 3 {
			return fmt.Errorf("expected >=3 tags, got %d", len(tags))
		}
		return nil
	})

	t.run("Tag unscanned file (reject)", func() error {
		err := fileSvc.TagFile("ghost.mp4", "cos")
		if err == nil {
			return fmt.Errorf("expected error for unregistered file")
		}
		return nil
	})

	t.run("Tag file (dup rejects)", func() error {
		err := fileSvc.TagFile("cos_remu_dance.mp4", "cos")
		if err == nil {
			return fmt.Errorf("expected error for duplicate tag")
		}
		return nil
	})

	t.run("Untag and re-tag", func() error {
		if err := fileSvc.UntagFile("cos_remu_dance.mp4", "跳舞"); err != nil {
			return err
		}
		fileSvc.TagFile("cos_remu_dance.mp4", "跳舞")
		return nil
	})

	searchSvc := service.NewSearchService()
	t.run("Search OR: cos JK", func() error {
		results, err := searchSvc.SearchByTags([]string{"cos", "JK"}, nil, "or")
		if err != nil {
			return err
		}
		if len(results) < 3 {
			return fmt.Errorf("expected >=3 files, got %d", len(results))
		}
		return nil
	})
	t.run("Search AND: cos 跳舞", func() error {
		results, err := searchSvc.SearchByTags([]string{"cos", "跳舞"}, nil, "and")
		if err != nil {
			return err
		}
		if len(results) != 2 {
			return fmt.Errorf("expected 2 (cos_remu_dance + cos_miku_dance), got %d", len(results))
		}
		return nil
	})
	t.run("Search AND: 蕾姆cos服 采访", func() error {
		results, err := searchSvc.SearchByTags([]string{"蕾姆cos服", "采访"}, nil, "and")
		if err != nil {
			return err
		}
		if len(results) != 1 {
			return fmt.Errorf("expected 1 (cos_remu_interview), got %d", len(results))
		}
		return nil
	})
	t.run("Search NOT: exclude 常服", func() error {
		results, err := searchSvc.SearchByTags(nil, []string{"常服"}, "or")
		if err != nil {
			return err
		}
		for _, r := range results {
			for _, t := range r.Tags {
				if t.TagName == "常服" {
					return fmt.Errorf("expected no 常服 files")
				}
			}
		}
		return nil
	})

	t.run("Search include + exclude: JK --not JK格裙", func() error {
		results, err := searchSvc.SearchByTags([]string{"JK"}, []string{"JK格裙"}, "or")
		if err != nil {
			return err
		}
		for _, r := range results {
			for _, t := range r.Tags {
				if t.TagName == "JK格裙" {
					return fmt.Errorf("expected no JK格裙 files")
				}
			}
		}
		return nil
	})

	t.run("Imported file exists", func() error {
		_, tags, err := fileSvc.GetFileInfo("imports/external_cos_video.mp4")
		if err != nil {
			return err
		}
		hasCos := false
		for _, t := range tags {
			if t.TagName == "cos" {
				hasCos = true
			}
		}
		if !hasCos {
			return fmt.Errorf("imported file should have cos tag")
		}
		return nil
	})

	t.run("Re-scan (no duplicates)", func() error {
		result, err := fileSvc.ScanDirectory(workspace)
		if err != nil {
			return err
		}
		if result.FilesAdded != 0 {
			return fmt.Errorf("expected 0 new files on re-scan, got %d", result.FilesAdded)
		}
		return nil
	})

	t.run("Import from external path", func() error {
		extDir := filepath.Join(workspace, "..", "test_imports")
		absDir, _ := filepath.Abs(extDir)
		extFile := filepath.Join(absDir, "external_cos_video.mp4")
		if _, err := os.Stat(extFile); err != nil {
			return fmt.Errorf("external file not found: %s", extFile)
		}
		_, err := fileSvc.ImportFile(extFile)
		if err != nil {
			if !strings.Contains(err.Error(), "未注册") && !strings.Contains(err.Error(), "已存在") {
				return err
			}
		}
		return nil
	})
}
