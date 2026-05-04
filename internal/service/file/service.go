package file

import (
	"encoding/json"
	"regexp"

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
	if err != nil || cfg == nil {
		return nil, nil
	}
	var patterns []string
	if err := json.Unmarshal([]byte(cfg.ConfigValue), &patterns); err != nil {
		return nil, nil
	}
	var res []*regexp.Regexp
	for _, p := range patterns {
		r, err := regexp.Compile(p)
		if err != nil {
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
