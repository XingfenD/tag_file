package ai

import "fmt"

// Task statuses stored in ai_tasks.status.
const (
	TaskQueued  = "queued"
	TaskRunning = "running"
	TaskDone    = "done"
	TaskFailed  = "failed"
)

// Review statuses stored in ai_suggestions.review_status. Nothing reaches the
// tag library before a human picks one of the two terminal states.
const (
	ReviewPending  = "pending"
	ReviewApproved = "approved"
	ReviewRejected = "rejected"
)

// File level AI tagging states derived from the two tables above (see Task 10).
// They are display labels only: File.TagDone stays manual and untouched.
const (
	AIStatusRunning  = "running"
	AIStatusFailed   = "failed"
	AIStatusPending  = "pending"
	AIStatusApproved = "approved"
)

// FileStatuses derives the display label of every requested file from the two
// AI tables. The rule looks at the file's newest task, i.e. the one with the
// largest task id (that column is the autoincrement primary key, so it orders
// the whole history of a file):
//
//	no task at all            -> absent from the map (the UI reads 未识别)
//	queued or running         -> AIStatusRunning
//	failed                    -> AIStatusFailed
//	done, pending proposals   -> AIStatusPending
//	done, nothing left to review -> AIStatusApproved
//
// The cost does not depend on how many ids are asked for: the task table is
// read once through latestTasks and the pending counts once through
// CountPendingByFile, and the reduction happens in memory — the same
// "load once, no per-node DB calls" trade the tag tree and the enqueue batch
// already make. Callers that render a whole tree therefore pay two reads, not
// one per file.
func (s *Service) FileStatuses(fileIDs []uint) (map[uint]string, error) {
	statuses := make(map[uint]string, len(fileIDs))
	if len(fileIDs) == 0 {
		// Never nil, so a JSON caller marshals {} instead of null.
		return statuses, nil
	}

	latest, err := s.latestTasks()
	if err != nil {
		return nil, err
	}
	pending, err := s.aiRepo.CountPendingByFile()
	if err != nil {
		return nil, fmt.Errorf("统计待审建议失败: %w", err)
	}

	for _, fileID := range fileIDs {
		task := latest[fileID]
		if task == nil {
			continue
		}
		// Only task.Status decides, and the cases are exhaustive over it: an
		// unexpected value leaves the file out of the map rather than guessing.
		//
		// Deliberately NOT `task.Error != ""` — a successful run can carry text
		// here. When a model answer exceeded maxTaskSuggestions, Task 8 kept the
		// first batch of proposals, marked the task done and recorded the
		// discarded remainder in Error, so failure has exactly one signal:
		// TaskFailed.
		switch task.Status {
		case TaskQueued, TaskRunning:
			statuses[fileID] = AIStatusRunning
		case TaskFailed:
			statuses[fileID] = AIStatusFailed
		case TaskDone:
			// The pending count is per file while the rule above is about the
			// newest task, so a proposal left behind by an older run would be
			// attributed to the current one. Unreachable through this service:
			// both force enqueue and RetryTask clear a file's pending proposals
			// before creating the task that replaces them (see enqueue.go), and
			// the in-flight guard refuses to queue over an unfinished run, so an
			// in-flight newest task never has another run's proposals next to it.
			if pending[fileID] > 0 {
				statuses[fileID] = AIStatusPending
			} else {
				statuses[fileID] = AIStatusApproved
			}
		}
	}
	return statuses, nil
}

// FileStatus is the one-file form of FileStatuses, for a detail panel. A file
// with no recognition history has no status: the empty string and no error.
func (s *Service) FileStatus(fileID uint) (string, error) {
	statuses, err := s.FileStatuses([]uint{fileID})
	if err != nil {
		return "", err
	}
	return statuses[fileID], nil
}
