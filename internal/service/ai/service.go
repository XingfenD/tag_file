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
	// signal and exists from construction, so an enqueue can always poke it. The
	// rest of the lifecycle lives in the run that is currently live: run is nil
	// until the first StartWorker and is replaced by every start, and workerMu
	// guards only that pointer swap (StopWorker copies the pointer under the lock
	// and waits for the goroutine outside it).
	wake     chan struct{}
	workerMu sync.Mutex
	run      *workerRun

	// enqueueMu serialises every read-decide-write of a queued task, i.e. the two
	// Enqueue* entry points (through enqueue) and RetryTask. Their gate is a
	// snapshot-then-insert with no database-level guard behind it: two callers
	// that both snapshot before either inserts each conclude "this file has no
	// previous task" and both create a task, which costs two model runs and two
	// sets of duplicate pending proposals for one file. Nothing else in the
	// process used to reach these methods concurrently — before the WebUI API
	// (Task 11) only tests and the CLI did — but two browser tabs, one double
	// click or one script now can. The worker needs no part of this lock: its
	// ClaimNextQueued is an atomic update in the database, and an enqueue that
	// reads a task mid queued→running transition sees an in-flight task either
	// way, which is a skip.
	//
	// Held across a whole batch on purpose: the batch is a handful of writes on
	// a local SQLite file, and a finer-grained lock per file would let a second
	// caller's snapshot land between two rows of the first one's batch.
	enqueueMu sync.Mutex
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
