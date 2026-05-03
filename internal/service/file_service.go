package service

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/repository"
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

type FileService struct {
	fileRepo     *repository.FileRepository
	tagRepo      *repository.TagRepository
	categoryRepo *repository.CategoryRepository
}

func NewFileService() *FileService {
	return &FileService{
		fileRepo:     repository.NewFileRepository(),
		tagRepo:      repository.NewTagRepository(),
		categoryRepo: repository.NewCategoryRepository(),
	}
}

func (s *FileService) buildFileModel(fullPath string, root string, info fs.FileInfo) (*model.File, string, error) {
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

func (s *FileService) ScanDirectory(dir string) (*ScanResult, error) {
	return s.ScanDirectoryOptions(dir, false)
}

func (s *FileService) ScanDirectoryOptions(dir string, full bool) (*ScanResult, error) {
	if dir == "" {
		dir = "."
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("解析目录路径失败: %w", err)
	}

	indexed, err := s.fileRepo.GetIndexedPaths()
	if err != nil {
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

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			result.DirsScanned++
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
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
			return nil, fmt.Errorf("批量写入失败: %w", err)
		}
		result.FilesAdded = len(newFiles)
	}

	s.saveScanState(result)
	return result, nil
}

func (s *FileService) loadScanState() scanState {
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

func (s *FileService) saveScanState(result *ScanResult) {
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

func (s *FileService) saveDiffState(result *DiffResult) {
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
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(".tagfile/diff_state", data, 0644)
}

func (s *FileService) ImportFile(extPath string) (*model.File, error) {
	absPath, err := filepath.Abs(extPath)
	if err != nil {
		return nil, fmt.Errorf("解析外部文件路径失败: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("无法访问外部文件: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("暂不支持导入目录")
	}

	cwd, _ := os.Getwd()
	cwd, _ = filepath.Abs(cwd)

	if strings.HasPrefix(absPath, cwd+string(filepath.Separator)) {
		relPath, _ := filepath.Rel(cwd, absPath)
		indexed, _ := s.fileRepo.GetIndexedPaths()
		if indexed[relPath] {
			return nil, fmt.Errorf("文件 '%s' 已在当前目录", relPath)
		}
		ext := strings.ToLower(filepath.Ext(absPath))
		fileType := mime.TypeByExtension(ext)
		if fileType == "" {
			fileType = ext
		}
		file := &model.File{
			FilePath: relPath,
			FileName: info.Name(),
			FileSize: info.Size(),
			FileType: fileType,
		}
		if err := s.fileRepo.Create(file); err != nil {
			return nil, err
		}
		return file, nil
	}

	destName := info.Name()
	destPath := filepath.Join(cwd, destName)
	counter := 1
	for {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		ext := filepath.Ext(destName)
		base := strings.TrimSuffix(destName, ext)
		destPath = filepath.Join(cwd, fmt.Sprintf("%s_%d%s", base, counter, ext))
		counter++
	}

	src, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("打开源文件失败: %w", err)
	}
	defer src.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dest.Close()

	if _, err := dest.ReadFrom(src); err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("复制文件失败: %w", err)
	}

	relPath, _ := filepath.Rel(cwd, destPath)
	ext := strings.ToLower(filepath.Ext(destPath))
	fileType := mime.TypeByExtension(ext)
	if fileType == "" {
		fileType = ext
	}

	file := &model.File{
		FilePath: relPath,
		FileName: filepath.Base(destPath),
		FileSize: info.Size(),
		FileType: fileType,
	}
	if err := s.fileRepo.Create(file); err != nil {
		os.Remove(destPath)
		return nil, fmt.Errorf("记录文件失败: %w", err)
	}
	return file, nil
}

func (s *FileService) ListFiles(offset, limit int) ([]model.File, int64, error) {
	list, total, err := s.fileRepo.List(offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("查询文件列表失败: %w", err)
	}
	return list, total, nil
}

func (s *FileService) IsIndexed(path string) (bool, error) {
	file, err := s.fileRepo.FindByPath(path)
	if err != nil {
		return false, fmt.Errorf("查询文件失败: %w", err)
	}
	return file != nil, nil
}

func (s *FileService) GetFileInfo(path string) (*model.File, []model.Tag, error) {
	file, err := s.fileRepo.FindByPath(path)
	if err != nil {
		return nil, nil, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, nil, fmt.Errorf("文件 '%s' 未注册", path)
	}
	tags, err := s.fileRepo.GetTags(file.FileID)
	if err != nil {
		return nil, nil, fmt.Errorf("查询标签失败: %w", err)
	}
	return file, tags, nil
}

func (s *FileService) TagFile(filePath string, tagName string) error {
	file, err := s.fileRepo.FindByPath(filePath)
	if err != nil {
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册，请先 scan", filePath)
	}

	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	for _, tag := range tags {
		has, err := s.fileRepo.HasTag(file.FileID, tag.TagID)
		if err != nil {
			return fmt.Errorf("查询关联失败: %w", err)
		}
		if has {
			return fmt.Errorf("文件已存在标签 '%s'", tagName)
		}
		return s.fileRepo.AddTag(file.FileID, tag.TagID)
	}
	return fmt.Errorf("标签 '%s' 不存在", tagName)
}

func (s *FileService) UntagFile(filePath string, tagName string) error {
	file, err := s.fileRepo.FindByPath(filePath)
	if err != nil {
		return fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return fmt.Errorf("文件 '%s' 未注册", filePath)
	}

	tags, err := s.tagRepo.FindByName(tagName)
	if err != nil {
		return fmt.Errorf("查询标签失败: %w", err)
	}
	if len(tags) == 0 {
		return fmt.Errorf("标签 '%s' 不存在", tagName)
	}

	for _, tag := range tags {
		has, err := s.fileRepo.HasTag(file.FileID, tag.TagID)
		if err != nil {
			return fmt.Errorf("查询关联失败: %w", err)
		}
		if has {
			return s.fileRepo.RemoveTag(file.FileID, tag.TagID)
		}
	}
	return fmt.Errorf("文件不存在标签 '%s'", tagName)
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
	AllFiles       []diffFileEntry
	Errors         []string
}

type diffFileEntry struct {
	Path    string
	Indexed bool
}

func (s *FileService) DiffUnindexed(dir string, recursive bool) (*DiffResult, error) {
	if dir == "" {
		dir = "."
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("解析目录路径失败: %w", err)
	}

	indexed, err := s.fileRepo.GetIndexedPaths()
	if err != nil {
		return nil, fmt.Errorf("加载索引失败: %w", err)
	}

	result := &DiffResult{Root: root}

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if recursive {
				if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
					return filepath.SkipDir
				}
				return nil
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		result.FilesFound++
		relPath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, relErr))
			return nil
		}
		isIndexed := indexed[relPath]
		result.AllFiles = append(result.AllFiles, diffFileEntry{Path: relPath, Indexed: isIndexed})
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
			return nil, fmt.Errorf("读取目录失败: %w", err)
		}
		for _, e := range entries {
			walkFn(filepath.Join(root, e.Name()), e, nil)
		}
	}

	s.saveDiffState(result)
	return result, nil
}
