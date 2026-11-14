package ai_test

import (
	"testing"

	repoAI "github.com/XingfenD/tag_file/internal/repository/ai"
	"github.com/XingfenD/tag_file/internal/service/ai"
)

// One file per row of the derivation table, plus the no-task case: the map is
// what the file tree, the detail panel and the queue page read, so the labels
// and the absences are both part of the contract.
func TestFileStatuses_AllTransitions(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	neverRun := seedFile(t, "a.jpg")

	queued := seedFile(t, "b.jpg")
	seedTask(t, queued.FileID, ai.TaskQueued)

	running := seedFile(t, "c.jpg")
	seedTask(t, running.FileID, ai.TaskRunning)

	failed := seedFile(t, "d.jpg")
	seedTask(t, failed.FileID, ai.TaskFailed)

	pendingReview := seedFile(t, "e.jpg")
	pendingTask := seedTask(t, pendingReview.FileID, ai.TaskDone)
	seedSuggestions(t, pendingTask.TaskID, pendingReview.FileID, ai.ReviewPending, 1)

	approved := seedFile(t, "f.jpg")
	approvedTask := seedTask(t, approved.FileID, ai.TaskDone)
	// Nothing of this finished run is left to review: every proposal got a
	// terminal review status.
	seedSuggestions(t, approvedTask.TaskID, approved.FileID, ai.ReviewRejected, 2)

	got, err := ai.NewService().FileStatuses([]uint{
		neverRun.FileID, queued.FileID, running.FileID,
		failed.FileID, pendingReview.FileID, approved.FileID,
	})
	if err != nil {
		t.Fatalf("FileStatuses: %v", err)
	}

	want := map[uint]string{
		queued.FileID:        ai.AIStatusRunning,
		running.FileID:       ai.AIStatusRunning,
		failed.FileID:        ai.AIStatusFailed,
		pendingReview.FileID: ai.AIStatusPending,
		approved.FileID:      ai.AIStatusApproved,
	}
	if len(got) != len(want) {
		t.Fatalf("FileStatuses returned %+v, want exactly the %d entries %v", got, len(want), want)
	}
	for fileID, wantStatus := range want {
		if got[fileID] != wantStatus {
			t.Errorf("file %d status = %q, want %q", fileID, got[fileID], wantStatus)
		}
	}
	// A file that was never submitted stays out of the map; the front end reads
	// that absence as 未识别.
	if _, ok := got[neverRun.FileID]; ok {
		t.Errorf("file %d with no task must be absent from the map, got %q", neverRun.FileID, got[neverRun.FileID])
	}
}

// The reduction keeps the task with the largest task id, and only that row's
// Status decides the label. Both halves are easy to get wrong: counting pending
// proposals before looking at the newest task answers a stale row, and treating
// a non-empty error column as a failure answers Task 8's successful truncation.
func TestFileStatuses_LatestTaskWins(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// 1. done with proposals left to review, then a newer queued task: the run
	// in flight wins, so the status goes back to running. The enqueue rules
	// delete pending proposals before re-queueing, so this row combination is
	// unreachable through them — it is staged directly here precisely to pin
	// that such a stale proposal cannot answer for the file.
	requeued := seedFile(t, "a.jpg")
	oldDone := seedTask(t, requeued.FileID, ai.TaskDone)
	seedSuggestions(t, oldDone.TaskID, requeued.FileID, ai.ReviewPending, 2)
	seedTask(t, requeued.FileID, ai.TaskQueued)

	// 2. a queued task that has since finished: keying on the oldest row of the
	// history would still report it as in flight.
	finished := seedFile(t, "b.jpg")
	seedTask(t, finished.FileID, ai.TaskQueued)
	seedTask(t, finished.FileID, ai.TaskDone)

	// 3. a file that was recognised and then re-recognised into a failure.
	broke := seedFile(t, "c.jpg")
	seedTask(t, broke.FileID, ai.TaskDone)
	seedTask(t, broke.FileID, ai.TaskFailed)

	// 4. a successful run whose answer exceeded the per-task cap: Task 8 keeps
	// the task done and records the loss in the error column, so a non-empty
	// error says nothing about the status.
	truncated := seedFile(t, "d.jpg")
	capped := seedTask(t, truncated.FileID, ai.TaskDone)
	if err := repoAI.NewRepository().UpdateTask(capped).
		SetError("模型给出了 64 条建议, 超过单任务上限 50 条: 仅写入前 50 条, 丢弃其余 14 条").Exec(); err != nil {
		t.Fatalf("record truncation: %v", err)
	}
	seedSuggestions(t, capped.TaskID, truncated.FileID, ai.ReviewPending, 3)

	got, err := ai.NewService().FileStatuses([]uint{
		requeued.FileID, finished.FileID, broke.FileID, truncated.FileID,
	})
	if err != nil {
		t.Fatalf("FileStatuses: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("FileStatuses returned %+v, want one entry per file", got)
	}

	want := map[uint]string{
		requeued.FileID:  ai.AIStatusRunning,
		finished.FileID:  ai.AIStatusApproved,
		broke.FileID:     ai.AIStatusFailed,
		truncated.FileID: ai.AIStatusPending,
	}
	for fileID, wantStatus := range want {
		if got[fileID] != wantStatus {
			t.Errorf("file %d status = %q, want %q", fileID, got[fileID], wantStatus)
		}
	}
}

// Nothing about an unknown file id is an error: absence is the answer, and it is
// the empty string the caller shows as 未识别.
func TestFileStatus_EmptyFileID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	known := seedFile(t, "a.jpg")
	task := seedTask(t, known.FileID, ai.TaskDone)
	seedSuggestions(t, task.TaskID, known.FileID, ai.ReviewPending, 1)

	status, err := ai.NewService().FileStatus(999)
	if err != nil {
		t.Fatalf("FileStatus for an unregistered file id: %v", err)
	}
	if status != "" {
		t.Fatalf("FileStatus(999) = %q, want the empty string", status)
	}

	// The same answer through the batch entry point, including for an empty
	// batch: an id that has no task is not in the map.
	got, err := ai.NewService().FileStatuses([]uint{999})
	if err != nil {
		t.Fatalf("FileStatuses for an unregistered file id: %v", err)
	}
	if _, ok := got[999]; ok {
		t.Errorf("FileStatuses(999) = %q, want no entry", got[999])
	}
	for _, ids := range [][]uint{nil, {}} {
		empty, err := ai.NewService().FileStatuses(ids)
		if err != nil {
			t.Fatalf("FileStatuses(%v): %v", ids, err)
		}
		// Non-nil and empty, so Task 11 marshals {} rather than null.
		if empty == nil {
			t.Errorf("FileStatuses(%v) = nil map, want an empty one", ids)
		}
		if len(empty) != 0 {
			t.Errorf("FileStatuses(%v) = %+v, want no entries", ids, empty)
		}
	}

	// A file that does have a task gets its label through the single-file
	// wrapper as well.
	single, err := ai.NewService().FileStatus(known.FileID)
	if err != nil {
		t.Fatalf("FileStatus: %v", err)
	}
	if single != ai.AIStatusPending {
		t.Fatalf("FileStatus(%d) = %q, want %q", known.FileID, single, ai.AIStatusPending)
	}
}
