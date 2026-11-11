package file

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	repoFile "github.com/XingfenD/tag_file/internal/repository/file"
	repoCategory "github.com/XingfenD/tag_file/internal/repository/category"
	repoConfig "github.com/XingfenD/tag_file/internal/repository/config"
	repoTag "github.com/XingfenD/tag_file/internal/repository/tag"
	repoTagGroup "github.com/XingfenD/tag_file/internal/repository/taggroup"
)

type Service struct {
	fileRepo      *repoFile.Repository
	tagRepo       *repoTag.Repository
	categoryRepo  *repoCategory.Repository
	configRepo    *repoConfig.Repository
	tagGroupRepo  *repoTagGroup.Repository
}

func NewService() *Service {
	return &Service{
		fileRepo:      repoFile.NewRepository(),
		tagRepo:       repoTag.NewRepository(),
		categoryRepo:  repoCategory.NewRepository(),
		configRepo:    repoConfig.NewRepository(),
		tagGroupRepo:  repoTagGroup.NewRepository(),
	}
}

func (s *Service) loadExcludePatterns() ([]*regexp.Regexp, error) {
	cfg, err := s.configRepo.Get("scan_exclude")
	if err != nil {
		log.Printf("[file] loadExcludePatterns: get config: %v", err)
		return nil, nil
	}
	if cfg == nil {
		return nil, nil
	}
	var patterns []string
	if err := json.Unmarshal([]byte(cfg.ConfigValue), &patterns); err != nil {
		log.Printf("[file] loadExcludePatterns: unmarshal: %v", err)
		return nil, nil
	}
	var res []*regexp.Regexp
	for _, p := range patterns {
		r, err := regexp.Compile(p)
		if err != nil {
			log.Printf("[file] loadExcludePatterns: invalid regex %q: %v", p, err)
			continue
		}
		res = append(res, r)
	}
	return res, nil
}

func matchExclude(patterns []*regexp.Regexp, name string) bool {
	for _, r := range patterns {
		if r.MatchString(name) {
			return true
		}
	}
	return false
}

// walkRoot resolves a scan/diff argument into the directory to walk plus the
// workspace root that indexed paths are relative to. Stored paths only mean
// anything against the workspace root — the tree, detail and raw endpoints all
// resolve them there — so a walk may not start outside it.
func walkRoot(dir string) (root, base string, err error) {
	if dir == "" {
		dir = "."
	}
	base, err = os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("解析工作目录失败: %w", err)
	}
	root, err = filepath.Abs(dir)
	if err != nil {
		return "", "", fmt.Errorf("解析目录路径失败: %w", err)
	}
	rel, err := filepath.Rel(base, root)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("目录 %s 不在工作目录 %s 内", dir, base)
	}
	return root, base, nil
}
