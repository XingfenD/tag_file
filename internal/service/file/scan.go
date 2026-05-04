package file

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/XingfenD/tag_file/internal/model"
)

type fileEntry struct {
	Path    string `json:"path"`
	Indexed bool   `json:"indexed"`
}

type scanState struct {
	Time       int64    `json:"time"`
	Root       string   `json:"root"`
	FilesFound int      `json:"files_found"`
	FilesAdded int      `json:"files_added"`
	Errors     []string `json:"errors,omitempty"`
}

type diffState struct {
	Time       int64       `json:"time"`
	Root       string      `json:"root"`
	FilesFound int         `json:"files_found"`
	Unindexed  int         `json:"unindexed"`
	Files      []fileEntry `json:"files"`
	Errors     []string    `json:"errors,omitempty"`
}

type ScanResult struct {
	Root        string
	FilesFound  int
	FilesAdded  int
	DirsScanned int
	Errors      []string
}

type DiffResult struct {
	Root           string
	FilesFound     int
	FilesUnindexed int
	Unindexed      []string
	AllFiles       []DiffFileEntry
	Errors         []string
}

type DiffFileEntry struct {
	Path    string
	Indexed bool
}

func (s *Service) buildFileModel(fullPath string, root string, info fs.FileInfo) (*model.File, string, error) {
	relPath, err := filepath.Rel(root, fullPath)
	if err != nil {
		return nil, "", err
	}
	ext := strings.ToLower(filepath.Ext(fullPath))
	fileType := mime.TypeByExtension(ext)
	if fileType == "" {
		fileType = ext
	}
	return &model.File{
		FilePath: relPath,
		FileName: info.Name(),
		FileSize: info.Size(),
		FileType: fileType,
	}, relPath, nil
}

func (s *Service) loadScanState() scanState {
	data, err := os.ReadFile(".tagfile/scan_state")
	if err != nil {
		return scanState{}
	}
	var state scanState
	if err := json.Unmarshal(data, &state); err != nil {
		return scanState{}
	}
	return state
}

func (s *Service) saveScanState(result *ScanResult) {
	os.MkdirAll(".tagfile", 0755)
	state := scanState{
		Time:       time.Now().Unix(),
		Root:       result.Root,
		FilesFound: result.FilesFound,
		FilesAdded: result.FilesAdded,
		Errors:     result.Errors,
	}
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(".tagfile/scan_state", data, 0644)
}

func (s *Service) ScanDirectory(dir string) (*ScanResult, error) {
	return s.ScanDirectoryOptions(dir, false)
}

func (s *Service) ScanDirectoryOptions(dir string, full bool) (*ScanResult, error) {
	if dir == "" {
		dir = "."
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		log.Printf("[file] ScanDirectoryOptions: resolve path: %v", err)
		return nil, fmt.Errorf("解析目录路径失败: %w", err)
	}

	indexed, err := s.fileRepo.GetIndexedPaths()
	if err != nil {
		log.Printf("[file] ScanDirectoryOptions: load indexed: %v", err)
		return nil, fmt.Errorf("加载索引失败: %w", err)
	}

	lastScan := s.loadScanState()
	if full {
		lastScan.Time = 0
	} else if rootInfo, statErr := os.Stat(root); statErr == nil {
		if rootInfo.ModTime().Unix() > lastScan.Time {
			lastScan.Time = 0
		}
	}

	result := &ScanResult{Root: root}
	var newFiles []model.File

	excludePatterns, _ := s.loadExcludePatterns()

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if matchExclude(excludePatterns, d.Name()) {
				return filepath.SkipDir
			}
			result.DirsScanned++
			return nil
		}
		if matchExclude(excludePatterns, d.Name()) {
			return nil
		}
		result.FilesFound++

		if !full {
			fi, fiErr := d.Info()
			if fiErr == nil && fi.ModTime().Unix() <= lastScan.Time {
				return nil
			}
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, infoErr))
			return nil
		}

		file, relPath, buildErr := s.buildFileModel(path, root, info)
		if buildErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, buildErr))
			return nil
		}
		if indexed[relPath] {
			return nil
		}
		newFiles = append(newFiles, *file)
		indexed[relPath] = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(newFiles) > 0 {
		if err := s.fileRepo.CreateBatch(newFiles); err != nil {
			log.Printf("[file] ScanDirectoryOptions: create batch: %v", err)
			return nil, fmt.Errorf("批量写入失败: %w", err)
		}
		for i := range newFiles {
			s.createDefaultGroup(newFiles[i].FileID)
		}
		result.FilesAdded = len(newFiles)
	}

	s.saveScanState(result)
	return result, nil
}
