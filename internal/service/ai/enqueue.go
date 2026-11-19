package ai

import (
	"fmt"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
)

// SkippedFile is one rejected enqueue request and the user-facing reason for it.
//
// The json tags are the wire contract of `skipped` on POST /api/ai/tasks: the
// WebUI handler marshals this struct as it is, and every other key that endpoint
// emits is snake_case, so the tags live here rather than in a webui-local mirror
// type (which Task 12/14 would then drift from). Renaming a field here renames
// the JSON key.
type SkippedFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// EnqueueResult reports one batch. Queued holds the ids of the newly created
// tasks, in request order, so a caller can poll them right away.
type EnqueueResult struct {
	Queued  []uint
	Skipped []SkippedFile
}

// candidate is one enqueue request: the path shown to the user plus the id of
// the file row it resolves to (0 when the path is not registered).
type candidate struct {
	path   string
	fileID uint
}

// EnqueuePaths submits an explicit list of paths.
func (s *Service) EnqueuePaths(paths []string, force bool) (*EnqueueResult, error) {
	index, err := s.pathIndex()
	if err != nil {
		return nil, err
	}
	items := make([]candidate, 0, len(paths))
	for _, path := range paths {
		items = append(items, candidate{path: path, fileID: index[path]})
	}
	return s.enqueue(items, force)
}

// EnqueueDir submits the registered media files under one directory. It filters
// the whole file table in memory instead of adding a query to the file repo, so
// a directory costs the same three reads as an explicit list. recursive=false
// takes the direct children only, matching the file tree expansion view.
func (s *Service) EnqueueDir(dir string, recursive bool, force bool) (*EnqueueResult, error) {
	files, _, err := s.fileSvc.ListFiles(0, 0)
	if err != nil {
		return nil, err
	}
	prefix := dirPrefix(dir)
	items := make([]candidate, 0, len(files))
	for _, f := range files {
		if !inDir(f.FilePath, prefix, recursive) {
			continue
		}
		items = append(items, candidate{path: f.FilePath, fileID: f.FileID})
	}
	return s.enqueue(items, force)
}

// RetryTask re-queues a failed task. A "running" task is retryable too, but only
// when no live recognition owns it, which the database cannot show and Service's
// in-process marker can. The old row is never rewritten: a new queued task is
// created so the error stays visible in the history. A file that already has a
// newer queued or running task gets nothing at all. Its pending suggestions are
// cleared first, because a fresh run would otherwise pile a second copy of the
// proposals on top of the ones nobody has reviewed yet.
func (s *Service) RetryTask(taskID uint) error {
	// Same critical section as enqueue: this method reads the file's newest
	// in-flight task and then inserts, so a concurrent enqueue of the same file
	// would otherwise land a second queued row behind it.
	s.enqueueMu.Lock()
	defer s.enqueueMu.Unlock()

	task, err := s.aiRepo.FindTask(taskID)
	if err != nil {
		return fmt.Errorf("查询识别任务失败: %w", err)
	}
	if task == nil {
		return fmt.Errorf("识别任务 %d 不存在", taskID)
	}
	switch task.Status {
	case TaskFailed:
	case TaskRunning:
		// A running row says two contradictory things and only one of them can be
		// told apart here: the worker is recognising this file right now, or a
		// previous process's goroutine died with the process (or the failure write
		// that was going to say so failed too). In the first case re-queuing costs a
		// second model run for one file and deletes the proposals the live run is
		// about to write; in the second the row is a lie with no control left for the
		// reviewer, because a row nobody owns is never claimed again — claims take
		// "queued" and the restart sweep only runs at StartWorker.
		if s.liveTask() == task.TaskID {
			return fmt.Errorf("识别任务 %d 正在执行中, 请等待它结束", task.TaskID)
		}
	default:
		return fmt.Errorf("仅失败或中断的任务可重试,当前状态 '%s'", task.Status)
	}
	// The old row is never rewritten, so without this guard a second click
	// would queue the same file again while the first retry is still waiting
	// for the worker.
	inFlight, err := s.inFlightTask(task.FileID, task.TaskID)
	if err != nil {
		return err
	}
	if inFlight != nil {
		return fmt.Errorf("该文件已有进行中的识别任务 %d,请勿重复提交", inFlight.TaskID)
	}
	if _, err := s.aiRepo.DeletePendingByFile(task.FileID); err != nil {
		return fmt.Errorf("清理待审建议失败: %w", err)
	}
	retry := &model.AITask{FileID: task.FileID, Status: TaskQueued}
	if err := s.aiRepo.CreateTask(retry); err != nil {
		return fmt.Errorf("创建识别任务失败: %w", err)
	}
	s.notifyWake()
	s.notifyChange()
	return nil
}

// enqueue applies the gate to a batch of candidates. It only creates queued
// task rows and, when force re-queues a finished or failed file, clears that
// file's stale pending suggestions first: no tag is created or attached and
// File.TagDone is never touched here.
//
// The two snapshots below are read once for the whole batch, so N paths cost
// two queries rather than 2N. Both snapshots and every insert happen under
// s.enqueueMu (see service.go): the gate is a read-then-write with no database
// guard behind it, so without the lock two overlapping batches could each
// snapshot "no previous task" for the same never-recognised file and both queue
// it.
func (s *Service) enqueue(items []candidate, force bool) (*EnqueueResult, error) {
	s.enqueueMu.Lock()
	defer s.enqueueMu.Unlock()

	pending, err := s.aiRepo.CountPendingByFile()
	if err != nil {
		return nil, fmt.Errorf("统计待审建议失败: %w", err)
	}
	latest, err := s.latestTasks()
	if err != nil {
		return nil, err
	}

	result := &EnqueueResult{Queued: []uint{}, Skipped: []SkippedFile{}}
	for _, item := range items {
		if _, ok := IsMediaFile(item.path); !ok {
			result.Skipped = append(result.Skipped, SkippedFile{item.path, "不支持的文件类型"})
			continue
		}
		if item.fileID == 0 {
			result.Skipped = append(result.Skipped, SkippedFile{item.path, "未注册,请先 scan"})
			continue
		}

		if force {
			// Force means "recognise this finished or failed file again", so an
			// in-flight newest task is answered with the same skip as in the
			// normal mode and nothing at all is mutated: a second queued row
			// would cost a second model call and a second set of pending
			// suggestions (spec §8 item 6), and clearing the existing ones
			// without replacing them would only lose work.
			if task := latest[item.fileID]; task != nil && (task.Status == TaskQueued || task.Status == TaskRunning) {
				result.Skipped = append(result.Skipped, SkippedFile{item.path, "已在队列中"})
				continue
			}
			if _, err := s.aiRepo.DeletePendingByFile(item.fileID); err != nil {
				return nil, fmt.Errorf("清理待审建议失败: %w", err)
			}
		} else {
			// A pending proposal is the more specific complaint, so it answers
			// before the status of the task that produced it.
			if pending[item.fileID] > 0 {
				result.Skipped = append(result.Skipped, SkippedFile{item.path, "已有待审建议"})
				continue
			}
			switch task := latest[item.fileID]; {
			case task == nil:
			case task.Status == TaskDone:
				result.Skipped = append(result.Skipped, SkippedFile{item.path, "已完成识别"})
				continue
			case task.Status == TaskQueued || task.Status == TaskRunning:
				result.Skipped = append(result.Skipped, SkippedFile{item.path, "已在队列中"})
				continue
			}
		}

		task := &model.AITask{FileID: item.fileID, Status: TaskQueued}
		if err := s.aiRepo.CreateTask(task); err != nil {
			return nil, fmt.Errorf("创建识别任务失败: %w", err)
		}
		result.Queued = append(result.Queued, task.TaskID)

		// Keep the snapshot honest about what was just written, so a path
		// listed twice in one batch is reported as already queued instead of
		// getting a second task, with force as well as without it.
		latest[item.fileID] = task
	}
	if len(result.Queued) > 0 {
		// One poke for the whole batch: the worker re-reads the queue and finds
		// every row this call just created.
		s.notifyWake()
		s.notifyChange()
	}
	return result, nil
}

// pathIndex maps every registered path to its file id in one pass. The id is
// what a task row needs, which is why this goes through ListFiles rather than
// the path-set helper of the file repo.
func (s *Service) pathIndex() (map[string]uint, error) {
	files, _, err := s.fileSvc.ListFiles(0, 0)
	if err != nil {
		return nil, err
	}
	index := make(map[string]uint, len(files))
	for _, f := range files {
		index[f.FilePath] = f.FileID
	}
	return index, nil
}

// latestTasks reduces the task table into the newest task per file, where
// newest means the largest task id; a file whose recognition failed is allowed
// to be queued again.
func (s *Service) latestTasks() (map[uint]*model.AITask, error) {
	tasks, _, err := s.aiRepo.ListTasks().Exec()
	if err != nil {
		return nil, fmt.Errorf("查询识别任务失败: %w", err)
	}
	latest := make(map[uint]*model.AITask, len(tasks))
	for i := range tasks {
		current := latest[tasks[i].FileID]
		if current == nil || tasks[i].TaskID > current.TaskID {
			latest[tasks[i].FileID] = &tasks[i]
		}
	}
	return latest, nil
}

// inFlightTask returns the newest queued or running task of one file that is
// newer than sinceID, or nil when the file has nothing in flight. RetryTask
// needs this per file, so it goes through the by-file builder instead of the
// whole-table read the batch path uses.
func (s *Service) inFlightTask(fileID, sinceID uint) (*model.AITask, error) {
	tasks, _, err := s.aiRepo.ListTasks().ByFileID(fileID).Exec()
	if err != nil {
		return nil, fmt.Errorf("查询识别任务失败: %w", err)
	}
	var newest *model.AITask
	for i := range tasks {
		// Rows come back ordered by task id, so the last match is the newest.
		if task := &tasks[i]; task.TaskID > sinceID && (task.Status == TaskQueued || task.Status == TaskRunning) {
			newest = task
		}
	}
	return newest, nil
}

// dirPrefix normalises a directory to the "a/b/" form every file under it
// starts with; the scan root itself has an empty prefix.
func dirPrefix(dir string) string {
	prefix := strings.Trim(dir, "/")
	if prefix == "." {
		prefix = ""
	}
	if prefix == "" {
		return ""
	}
	return prefix + "/"
}

// inDir reports whether filePath belongs to the directory behind prefix.
func inDir(filePath, prefix string, recursive bool) bool {
	if !strings.HasPrefix(filePath, prefix) {
		return false
	}
	if recursive {
		return true
	}
	return !strings.Contains(filePath[len(prefix):], "/")
}
