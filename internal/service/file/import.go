package file

import (
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Service) ImportFile(extPath string) (*model.File, error) {
	absPath, err := filepath.Abs(extPath)
	if err != nil {
		log.Printf("[file] ImportFile: resolve path: %v", err)
		return nil, fmt.Errorf("解析外部文件路径失败: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		log.Printf("[file] ImportFile: stat: %v", err)
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
			log.Printf("[file] ImportFile: create record: %v", err)
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
		log.Printf("[file] ImportFile: open src: %v", err)
		return nil, fmt.Errorf("打开源文件失败: %w", err)
	}
	defer src.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		log.Printf("[file] ImportFile: create dest: %v", err)
		return nil, fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dest.Close()

	if _, err := dest.ReadFrom(src); err != nil {
		log.Printf("[file] ImportFile: copy: %v", err)
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
		log.Printf("[file] ImportFile: create record: %v", err)
		os.Remove(destPath)
		return nil, fmt.Errorf("记录文件失败: %w", err)
	}
	return file, nil
}
