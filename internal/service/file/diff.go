package file

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"
)

func (s *Service) saveDiffState(result *DiffResult) {
	os.MkdirAll(".tagfile", 0755)
	entries := make([]fileEntry, 0, len(result.AllFiles))
	for _, f := range result.AllFiles {
		entries = append(entries, fileEntry{
			Path:    f.Path,
			Indexed: f.Indexed,
		})
	}
	state := diffState{
		Time:       time.Now().Unix(),
		Root:       result.Root,
		FilesFound: result.FilesFound,
		Unindexed:  result.FilesUnindexed,
		Files:      entries,
		Errors:     result.Errors,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		log.Printf("[file] saveDiffState: marshal: %v", err)
		return
	}
	if err := os.WriteFile(".tagfile/diff_state", data, 0644); err != nil {
		log.Printf("[file] saveDiffState: write: %v", err)
	}
}

func (s *Service) DiffUnindexed(dir string, recursive bool) (*DiffResult, error) {
	root, base, err := walkRoot(dir)
	if err != nil {
		log.Printf("[file] DiffUnindexed: %v", err)
		return nil, err
	}

	indexed, err := s.fileRepo.GetIndexedPaths()
	if err != nil {
		log.Printf("[file] DiffUnindexed: load indexed: %v", err)
		return nil, fmt.Errorf("加载索引失败: %w", err)
	}

	result := &DiffResult{Root: root}

	excludePatterns, _ := s.loadExcludePatterns()

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if recursive {
				if matchExclude(excludePatterns, d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			return nil
		}
		if matchExclude(excludePatterns, d.Name()) {
			return nil
		}
		result.FilesFound++
		relPath, relErr := filepath.Rel(base, path)
		if relErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, relErr))
			return nil
		}
		isIndexed := indexed[relPath]
		result.AllFiles = append(result.AllFiles, DiffFileEntry{Path: relPath, Indexed: isIndexed})
		if !isIndexed {
			result.Unindexed = append(result.Unindexed, relPath)
			result.FilesUnindexed++
		}
		return nil
	}

	if recursive {
		filepath.WalkDir(root, walkFn)
	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			log.Printf("[file] DiffUnindexed: read dir: %v", err)
			return nil, fmt.Errorf("读取目录失败: %w", err)
		}
		for _, e := range entries {
			walkFn(filepath.Join(root, e.Name()), e, nil)
		}
	}

	s.saveDiffState(result)
	return result, nil
}
