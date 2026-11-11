package file

import (
	"fmt"
	"log"

	"github.com/XingfenD/tag_file/internal/model"
)

// Recommendation scopes.
const (
	ScopeTodo     = "todo"
	ScopeUntagged = "untagged"
	ScopeAll      = "all"
)

// RandomPick is a recommended file plus the pool it was drawn from.
type RandomPick struct {
	File      *model.File
	Total     int64
	Remaining int64
	Exhausted bool
}

// RandomFile draws one indexed file for tagging. exclude carries the ids the
// caller has already shown; a nil pick means the scope holds no files.
func (s *Service) RandomFile(scope string, exclude []uint) (*RandomPick, error) {
	if scope == "" {
		scope = ScopeTodo
	}
	switch scope {
	case ScopeTodo, ScopeUntagged, ScopeAll:
	default:
		return nil, fmt.Errorf("无效的推荐范围: %s", scope)
	}

	file, pool, err := s.fileRepo.Random().Scope(scope).Exclude(exclude).Exec()
	if err != nil {
		log.Printf("[file] RandomFile: %v", err)
		return nil, fmt.Errorf("随机选取文件失败: %w", err)
	}
	if file == nil {
		return nil, nil
	}
	return &RandomPick{
		File:      file,
		Total:     pool.Total,
		Remaining: pool.Remaining,
		Exhausted: pool.Exhausted,
	}, nil
}
