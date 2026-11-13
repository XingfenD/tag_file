package ai

import (
	repoAI "github.com/XingfenD/tag_file/internal/repository/ai"
	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	fileSvc "github.com/XingfenD/tag_file/internal/service/file"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
)

// Service is the Ollama vision auto-tagging service.
//
// Extractor and Client are the exported seams to the outside world (video frame
// extraction and the vision model), so tests can substitute fakes and `make
// test` runs without ffmpeg or a model present: Extractor is the real ffmpeg one,
// but its test skips when the binary is missing, and Client is deliberately left
// nil because the recognition path builds one per task out of Settings().
type Service struct {
	aiRepo      *repoAI.Repository
	fileSvc     *fileSvc.Service
	tagSvc      *tagSvc.Service
	categorySvc *categorySvc.Service
	configSvc   *configSvc.Service

	Extractor FrameExtractor
	Client    VisionClient

	wake, stop, done chan struct{}
}

func NewService() *Service {
	return &Service{
		aiRepo:      repoAI.NewRepository(),
		fileSvc:     fileSvc.NewService(),
		tagSvc:      tagSvc.NewService(),
		categorySvc: categorySvc.NewService(),
		configSvc:   configSvc.NewService(),

		Extractor: NewFFmpegExtractor(),
	}
}
