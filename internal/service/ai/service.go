package ai

import (
	"sync"

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

	// Worker lifecycle, see worker.go: wake is the buffered "the queue grew"
	// signal, stop is closed by StopWorker and done by the worker goroutine when it
	// returns. wake exists from construction so an enqueue can always poke it;
	// stop and done only exist while a worker runs, and stopOnce keeps a repeated
	// StopWorker harmless.
	wake     chan struct{}
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

func NewService() *Service {
	return &Service{
		aiRepo:      repoAI.NewRepository(),
		fileSvc:     fileSvc.NewService(),
		tagSvc:      tagSvc.NewService(),
		categorySvc: categorySvc.NewService(),
		configSvc:   configSvc.NewService(),

		Extractor: NewFFmpegExtractor(),

		wake: make(chan struct{}, 1),
	}
}
