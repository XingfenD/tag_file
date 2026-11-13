package ai_test

import (
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	repoAI "github.com/XingfenD/tag_file/internal/repository/ai"
	"github.com/XingfenD/tag_file/internal/service/ai"
)

func seedTask(t *testing.T, fileID uint, status string) *model.AITask {
	t.Helper()
	task := &model.AITask{FileID: fileID, Status: status}
	if status == ai.TaskFailed {
		task.Error = "模型不可用"
	}
	if err := repoAI.NewRepository().CreateTask(task); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	return task
}

func seedSuggestions(t *testing.T, taskID, fileID uint, reviewStatus string, count int) {
	t.Helper()
	items := make([]model.AISuggestion, 0, count)
	for i := 0; i < count; i++ {
		items = append(items, model.AISuggestion{
			TaskID:       taskID,
			FileID:       fileID,
			CategoryName: "主体",
			TagName:      "猫",
			ReviewStatus: reviewStatus,
		})
	}
	if err := repoAI.NewRepository().CreateSuggestions(items); err != nil {
		t.Fatalf("seed suggestions: %v", err)
	}
}

// tasksOf loads every task of one file ordered by task_id.
func tasksOf(t *testing.T, fileID uint) []model.AITask {
	t.Helper()
	tasks, _, err := repoAI.NewRepository().ListTasks().ByFileID(fileID).Exec()
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	return tasks
}

func countSuggestions(t *testing.T, fileID uint, reviewStatus string) int64 {
	t.Helper()
	_, total, err := repoAI.NewRepository().ListSuggestions().ByFileID(fileID).ByStatus(reviewStatus).Exec()
	if err != nil {
		t.Fatalf("count suggestions: %v", err)
	}
	return total
}

// skipReason returns the reason recorded for path, or "" when path was not skipped.
func skipReason(res *ai.EnqueueResult, path string) string {
	for _, skipped := range res.Skipped {
		if skipped.Path == path {
			return skipped.Reason
		}
	}
	return ""
}

func TestIsMediaFile(t *testing.T) {
	cases := []struct {
		name     string
		wantKind string
		wantOK   bool
	}{
		{"a.JPG", "image", true},
		{"a.jpeg", "image", true},
		{"a.PNG", "image", true},
		{"a.webp", "image", true},
		{"b.mp4", "video", true},
		{"b.MKV", "video", true},
		{"b.mov", "video", true},
		{"b.avi", "video", true},
		{"b.webm", "video", true},
		{"b.m4V", "video", true},
		{"c.txt", "", false},
		{"d", "", false},
		// A nested path still resolves by its extension.
		{"dir/sub/e.JPeG", "image", true},
		// Only the final suffix counts, so a doubled extension is unknown.
		{"archive.tar.gz", "", false},
		// filepath.Ext cuts at the final dot, which for a dot file such as
		// ".jpg" is its leading character: it does classify as an image.
		{".jpg", "image", true},
	}

	for _, c := range cases {
		kind, ok := ai.IsMediaFile(c.name)
		if kind != c.wantKind || ok != c.wantOK {
			t.Errorf("IsMediaFile(%q) = (%q, %v), want (%q, %v)", c.name, kind, ok, c.wantKind, c.wantOK)
		}
	}
}

func TestEnqueuePaths_SkipsUnregisteredAndNonMedia(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	registered := seedFile(t, "a.jpg")
	svc := ai.NewService()

	res, err := svc.EnqueuePaths([]string{"a.jpg", "ghost.jpg", "note.txt"}, false)
	if err != nil {
		t.Fatalf("enqueue paths: %v", err)
	}

	if len(res.Queued) != 1 {
		t.Fatalf("Queued = %v, want exactly one task", res.Queued)
	}
	task, err := repoAI.NewRepository().FindTask(res.Queued[0])
	if err != nil {
		t.Fatalf("find queued task: %v", err)
	}
	if task == nil {
		t.Fatal("Queued must hold the new task id, but that task is absent")
	}
	if task.FileID != registered.FileID || task.Status != ai.TaskQueued {
		t.Fatalf("new task = file %d status %q, want file %d status %q",
			task.FileID, task.Status, registered.FileID, ai.TaskQueued)
	}

	if len(res.Skipped) != 2 {
		t.Fatalf("Skipped = %+v, want two entries", res.Skipped)
	}
	if reason := skipReason(res, "ghost.jpg"); !strings.Contains(reason, "未注册") {
		t.Errorf("reason for ghost.jpg = %q, want it to mention 未注册", reason)
	}
	// note.txt is unregistered too, but the extension gate answers first.
	if reason := skipReason(res, "note.txt"); !strings.Contains(reason, "不支持") {
		t.Errorf("reason for note.txt = %q, want it to mention 不支持", reason)
	}

	if total := tasksOf(t, registered.FileID); len(total) != 1 {
		t.Fatalf("a.jpg tasks = %d, want 1", len(total))
	}
}

func TestEnqueuePaths_SkipsAlreadyPendingOrDone(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	withPending := seedFile(t, "f1.jpg")
	doneTask := seedTask(t, withPending.FileID, ai.TaskDone)
	seedSuggestions(t, doneTask.TaskID, withPending.FileID, ai.ReviewPending, 1)

	justDone := seedFile(t, "f2.jpg")
	seedTask(t, justDone.FileID, ai.TaskDone)

	inQueue := seedFile(t, "f3.jpg")
	seedTask(t, inQueue.FileID, ai.TaskQueued)

	fresh := seedFile(t, "f4.jpg")

	svc := ai.NewService()
	res, err := svc.EnqueuePaths([]string{"f1.jpg", "f2.jpg", "f3.jpg", "f4.jpg"}, false)
	if err != nil {
		t.Fatalf("enqueue paths: %v", err)
	}

	if len(res.Queued) != 1 {
		t.Fatalf("Queued = %v, want only f4.jpg enqueued", res.Queued)
	}
	task, err := repoAI.NewRepository().FindTask(res.Queued[0])
	if err != nil || task == nil {
		t.Fatalf("find queued task: %v", err)
	}
	if task.FileID != fresh.FileID {
		t.Fatalf("queued task file = %d, want f4.jpg id %d", task.FileID, fresh.FileID)
	}

	if len(res.Skipped) != 3 {
		t.Fatalf("Skipped = %+v, want three entries", res.Skipped)
	}
	// f1 has both a done task and a pending suggestion: the pending suggestion wins.
	if reason := skipReason(res, "f1.jpg"); !strings.Contains(reason, "已有待审建议") {
		t.Errorf("reason for f1.jpg = %q, want 已有待审建议", reason)
	}
	if reason := skipReason(res, "f2.jpg"); !strings.Contains(reason, "已完成识别") {
		t.Errorf("reason for f2.jpg = %q, want 已完成识别", reason)
	}
	if reason := skipReason(res, "f3.jpg"); !strings.Contains(reason, "已在队列中") {
		t.Errorf("reason for f3.jpg = %q, want 已在队列中", reason)
	}

	for _, f := range []*model.File{withPending, justDone, inQueue} {
		if got := tasksOf(t, f.FileID); len(got) != 1 {
			t.Errorf("file %s tasks = %d, want the single pre-existing task", f.FilePath, len(got))
		}
	}
}

func TestEnqueuePaths_ForceClearsPendingAndRequeues(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	target := seedFile(t, "f1.jpg")
	original := seedTask(t, target.FileID, ai.TaskDone)
	seedSuggestions(t, original.TaskID, target.FileID, ai.ReviewPending, 2)
	seedSuggestions(t, original.TaskID, target.FileID, ai.ReviewRejected, 1)

	svc := ai.NewService()
	res, err := svc.EnqueuePaths([]string{"f1.jpg"}, true)
	if err != nil {
		t.Fatalf("force enqueue: %v", err)
	}

	if len(res.Queued) != 1 {
		t.Fatalf("Queued = %v, want one new task", res.Queued)
	}
	if len(res.Skipped) != 0 {
		t.Fatalf("Skipped = %+v, want none when force", res.Skipped)
	}
	newTask, err := repoAI.NewRepository().FindTask(res.Queued[0])
	if err != nil || newTask == nil {
		t.Fatalf("find new task: %v", err)
	}
	if newTask.Status != ai.TaskQueued {
		t.Fatalf("new task status = %q, want %q", newTask.Status, ai.TaskQueued)
	}
	if newTask.TaskID == original.TaskID {
		t.Fatal("force must create a new task instead of rewriting the old row")
	}

	// The failed/done history and the reviewed rows survive; only pending is dropped.
	tasks := tasksOf(t, target.FileID)
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want the old done task plus the new queued one", len(tasks))
	}
	if tasks[0].TaskID != original.TaskID || tasks[0].Status != ai.TaskDone {
		t.Fatalf("old task = %d status %q, want %d still %q",
			tasks[0].TaskID, tasks[0].Status, original.TaskID, ai.TaskDone)
	}
	if pending := countSuggestions(t, target.FileID, ai.ReviewPending); pending != 0 {
		t.Errorf("pending suggestions = %d, want 0 after force", pending)
	}
	if rejected := countSuggestions(t, target.FileID, ai.ReviewRejected); rejected != 1 {
		t.Errorf("rejected suggestions = %d, want the reviewed history kept", rejected)
	}
}

func TestEnqueueDir_RecursiveFilter(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	shallow := seedFile(t, "d/a.jpg")
	deep := seedFile(t, "d/sub/b.jpg")
	outside := seedFile(t, "e/c.jpg")
	svc := ai.NewService()

	res, err := svc.EnqueueDir("d", false, false)
	if err != nil {
		t.Fatalf("enqueue dir non-recursive: %v", err)
	}
	if len(res.Queued) != 1 || len(res.Skipped) != 0 {
		t.Fatalf("non-recursive = queued %v skipped %+v, want only d/a.jpg", res.Queued, res.Skipped)
	}
	queuedIDs := map[uint]bool{}
	for _, id := range res.Queued {
		task, err := repoAI.NewRepository().FindTask(id)
		if err != nil || task == nil {
			t.Fatalf("find task %d: %v", id, err)
		}
		queuedIDs[task.FileID] = true
	}
	if !queuedIDs[shallow.FileID] {
		t.Fatalf("non-recursive enqueued file ids %v, want d/a.jpg id %d", queuedIDs, shallow.FileID)
	}

	res, err = svc.EnqueueDir("d/", true, true)
	if err != nil {
		t.Fatalf("enqueue dir recursive: %v", err)
	}
	if len(res.Queued) != 2 || len(res.Skipped) != 0 {
		t.Fatalf("recursive = queued %v skipped %+v, want d/a.jpg and d/sub/b.jpg", res.Queued, res.Skipped)
	}
	queuedIDs = map[uint]bool{}
	for _, id := range res.Queued {
		task, err := repoAI.NewRepository().FindTask(id)
		if err != nil || task == nil {
			t.Fatalf("find task %d: %v", id, err)
		}
		queuedIDs[task.FileID] = true
	}
	if !queuedIDs[shallow.FileID] || !queuedIDs[deep.FileID] {
		t.Fatalf("recursive enqueued file ids %v, want %d and %d", queuedIDs, shallow.FileID, deep.FileID)
	}
	if queuedIDs[outside.FileID] {
		t.Errorf("e/c.jpg must not be enqueued by dir d")
	}
	if got := tasksOf(t, outside.FileID); len(got) != 0 {
		t.Errorf("e/c.jpg tasks = %d, want 0", len(got))
	}
}

func TestRetryTask_OnlyFailed(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	failedFile := seedFile(t, "a.jpg")
	failed := seedTask(t, failedFile.FileID, ai.TaskFailed)
	seedSuggestions(t, failed.TaskID, failedFile.FileID, ai.ReviewPending, 2)

	doneFile := seedFile(t, "b.jpg")
	done := seedTask(t, doneFile.FileID, ai.TaskDone)

	svc := ai.NewService()

	if err := svc.RetryTask(failed.TaskID); err != nil {
		t.Fatalf("retry failed task: %v", err)
	}
	tasks := tasksOf(t, failedFile.FileID)
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want the failed row plus one new queued row", len(tasks))
	}
	if tasks[0].TaskID != failed.TaskID || tasks[0].Status != ai.TaskFailed {
		t.Fatalf("failed row = %d status %q, want it kept for history", tasks[0].TaskID, tasks[0].Status)
	}
	if tasks[1].Status != ai.TaskQueued {
		t.Fatalf("new task status = %q, want %q", tasks[1].Status, ai.TaskQueued)
	}
	if pending := countSuggestions(t, failedFile.FileID, ai.ReviewPending); pending != 0 {
		t.Errorf("pending suggestions = %d, want 0 so retried runs do not pile up", pending)
	}

	if err := svc.RetryTask(done.TaskID); err == nil {
		t.Fatal("retrying a done task must return an error")
	}
	if got := tasksOf(t, doneFile.FileID); len(got) != 1 {
		t.Errorf("done file tasks = %d, want no new task", len(got))
	}

	if err := svc.RetryTask(9999); err == nil {
		t.Error("retrying an absent task must return an error")
	}
}
