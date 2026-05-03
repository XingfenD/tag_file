package file

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/XingfenD/tag_file/internal/model"
)

func (s *Service) RenameFile(oldPath string, newName string) (*model.File, error) {
	file, err := s.fileRepo.FindByPath(oldPath)
	if err != nil {
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
		return nil, fmt.Errorf("重命名文件失败: %w", err)
	}

	newRelPath, _ := filepath.Rel(cwd, newAbs)
	file.FilePath = newRelPath
	file.FileName = newName

	if err := s.fileRepo.Update(file); err != nil {
		os.Rename(newAbs, oldAbs)
		return nil, fmt.Errorf("更新文件记录失败: %w", err)
	}

	return file, nil
}

func (s *Service) SearchFiles(keyword string, offset, limit int) ([]model.File, int64, error) {
	if keyword == "" {
		return nil, 0, fmt.Errorf("搜索关键词不能为空")
	}
	return s.fileRepo.SearchByName(keyword, offset, limit)
}
