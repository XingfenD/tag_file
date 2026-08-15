package file

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Service) RenameFile(oldPath string, newName string) (*model.File, error) {
	file, err := s.fileRepo.Find().ByPath(oldPath).Exec()
	if err != nil {
		log.Printf("[file] RenameFile: find file: %v", err)
		return nil, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, fmt.Errorf("文件 '%s' 未注册", oldPath)
	}

	cwd, _ := os.Getwd()
	oldAbs := filepath.Join(cwd, oldPath)
	newAbs := filepath.Join(filepath.Dir(oldAbs), newName)

	if _, err := os.Stat(newAbs); err == nil {
		return nil, fmt.Errorf("目标文件名 '%s' 已存在", newName)
	}

	if err := os.Rename(oldAbs, newAbs); err != nil {
		log.Printf("[file] RenameFile: rename: %v", err)
		return nil, fmt.Errorf("重命名文件失败: %w", err)
	}

	newRelPath, _ := filepath.Rel(cwd, newAbs)

	if err := s.fileRepo.Update(file).SetFilePath(newRelPath).SetFileName(newName).Exec(); err != nil {
		log.Printf("[file] RenameFile: update record: %v", err)
		os.Rename(newAbs, oldAbs)
		return nil, fmt.Errorf("更新文件记录失败: %w", err)
	}

	return file, nil
}

func (s *Service) SetFileTagDone(path string, done bool) (*model.File, error) {
	file, err := s.fileRepo.Find().ByPath(path).Exec()
	if err != nil {
		log.Printf("[file] SetFileTagDone: find file: %v", err)
		return nil, fmt.Errorf("查询文件失败: %w", err)
	}
	if file == nil {
		return nil, fmt.Errorf("文件 '%s' 未注册", path)
	}
	if file.TagDone == done {
		return file, nil
	}
	if err := s.fileRepo.Update(file).SetTagDone(done).Exec(); err != nil {
		log.Printf("[file] SetFileTagDone: update record: %v", err)
		return nil, fmt.Errorf("更新文件记录失败: %w", err)
	}
	return file, nil
}

func (s *Service) SearchFiles(keyword string, offset, limit int) ([]model.File, int64, error) {
	if keyword == "" {
		return nil, 0, fmt.Errorf("搜索关键词不能为空")
	}
	return s.fileRepo.Search().ByName(keyword).Offset(offset).Limit(limit).Exec()
}
