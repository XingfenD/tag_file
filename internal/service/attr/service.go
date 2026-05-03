package attr

import (
	repoAttr "github.com/XingfenD/tag_file/internal/repository/attr"
	repoTag "github.com/XingfenD/tag_file/internal/repository/tag"
)

type Service struct {
	attrRepo *repoAttr.Repository
	tagRepo  *repoTag.Repository
}

func NewService() *Service {
	return &Service{
		attrRepo: repoAttr.NewRepository(),
		tagRepo:  repoTag.NewRepository(),
	}
}
