package service

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/repository"
)

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

func (s *FileService) scanFile(fullPath string, root string) (*model.File, error) {
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, nil
	}

	relPath, err := filepath.Rel(root, fullPath)
	if err != nil {
		return nil, err
	}

	existing, err := s.fileRepo.FindByPath(relPath)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, nil
	}

	ext := strings.ToLower(filepath.Ext(fullPath))
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

func (s *FileService) ScanDirectory(dir string) (*ScanResult, error) {
	if dir == "" {
		dir = "."
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("解析目录路径失败: %w", err)
	}

	result := &ScanResult{Root: root}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			result.DirsScanned++
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		result.FilesFound++
		file, scanErr := s.scanFile(path, root)
		if scanErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, scanErr))
			return nil
		}
		if file != nil {
			result.FilesAdded++
		}
		return nil
	})

	return result, err
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
		return s.scanFile(absPath, cwd)
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

func (s *FileService) ListFiles() ([]model.File, error) {
	return s.fileRepo.List()
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
	Root       string
	FilesFound int
	FilesAdded int
	DirsScanned int
	Errors     []string
}
