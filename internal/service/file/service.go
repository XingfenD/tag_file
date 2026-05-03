package file

import (
	repoFile "github.com/XingfenD/tag_file/internal/repository/file"
	repoCategory "github.com/XingfenD/tag_file/internal/repository/category"
	repoTag "github.com/XingfenD/tag_file/internal/repository/tag"
)

type Service struct {
	fileRepo     *repoFile.Repository
	tagRepo      *repoTag.Repository
	categoryRepo *repoCategory.Repository
}

func NewService() *Service {
	return &Service{
		fileRepo:     repoFile.NewRepository(),
		tagRepo:      repoTag.NewRepository(),
		categoryRepo: repoCategory.NewRepository(),
	}
}
