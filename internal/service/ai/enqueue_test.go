package ai_test

import (
	"strings"
	"sync"
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

// Force re-recognises finished or failed work. A file whose newest task is
// still in flight must be left completely untouched: a second queued row would
// mean a second model call and a second set of pending proposals (spec §8
// item 6), and clearing the existing ones without producing anything would just
// destroy them.
func TestEnqueuePaths_ForceLeavesInFlightUntouched(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	queuedFile := seedFile(t, "q.jpg")
	queuedTask := seedTask(t, queuedFile.FileID, ai.TaskQueued)
	seedSuggestions(t, queuedTask.TaskID, queuedFile.FileID, ai.ReviewPending, 2)

	runningFile := seedFile(t, "r.jpg")
	runningTask := seedTask(t, runningFile.FileID, ai.TaskRunning)
	seedSuggestions(t, runningTask.TaskID, runningFile.FileID, ai.ReviewPending, 1)

	svc := ai.NewService()
	res, err := svc.EnqueuePaths([]string{"q.jpg", "r.jpg", "ghost.jpg", "note.txt"}, true)
	if err != nil {
		t.Fatalf("force enqueue: %v", err)
	}
	if len(res.Queued) != 0 {
		t.Fatalf("Queued = %v, want no task for a file that is already in flight", res.Queued)
	}
	if len(res.Skipped) != 4 {
		t.Fatalf("Skipped = %+v, want four entries", res.Skipped)
	}
	if reason := skipReason(res, "q.jpg"); !strings.Contains(reason, "已在队列中") {
		t.Errorf("reason for q.jpg = %q, want 已在队列中", reason)
	}
	if reason := skipReason(res, "r.jpg"); !strings.Contains(reason, "已在队列中") {
		t.Errorf("reason for r.jpg = %q, want 已在队列中", reason)
	}
	// The eligibility gates still answer first, so force cannot push a
	// non-media or unregistered path through either.
	if reason := skipReason(res, "ghost.jpg"); !strings.Contains(reason, "未注册") {
		t.Errorf("reason for ghost.jpg = %q, want it to mention 未注册", reason)
	}
	if reason := skipReason(res, "note.txt"); !strings.Contains(reason, "不支持") {
		t.Errorf("reason for note.txt = %q, want it to mention 不支持", reason)
	}

	if got := tasksOf(t, queuedFile.FileID); len(got) != 1 {
		t.Errorf("q.jpg tasks = %d, want the single queued row only", len(got))
	}
	if got := tasksOf(t, runningFile.FileID); len(got) != 1 {
		t.Errorf("r.jpg tasks = %d, want the single running row only", len(got))
	}
	if pending := countSuggestions(t, queuedFile.FileID, ai.ReviewPending); pending != 2 {
		t.Errorf("q.jpg pending = %d, want the proposals kept when force skips the file", pending)
	}
	if pending := countSuggestions(t, runningFile.FileID, ai.ReviewPending); pending != 1 {
		t.Errorf("r.jpg pending = %d, want the proposals kept when force skips the file", pending)
	}
}

// The six brief-named tests never reach three rows of the rule table: a newest
// task that failed, a newest task that is running, and the same path listed
// twice in one batch. This pins all three.
func TestEnqueuePaths_NonForceFailedRunningAndDuplicatePath(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// A failed task is the only newest state that stays enqueueable; f1 carries
	// no pending suggestion, because 已有待审建议 would answer before it.
	failedLatest := seedFile(t, "f1.jpg")
	seedTask(t, failedLatest.FileID, ai.TaskFailed)

	runningLatest := seedFile(t, "f2.jpg")
	seedTask(t, runningLatest.FileID, ai.TaskRunning)

	twice := seedFile(t, "f3.jpg")

	svc := ai.NewService()
	res, err := svc.EnqueuePaths([]string{"f1.jpg", "f2.jpg", "f3.jpg", "f3.jpg"}, false)
	if err != nil {
		t.Fatalf("enqueue paths: %v", err)
	}

	if len(res.Queued) != 2 {
		t.Fatalf("Queued = %v, want f1.jpg and one task for f3.jpg", res.Queued)
	}
	if len(res.Skipped) != 2 {
		t.Fatalf("Skipped = %+v, want f2.jpg plus the repeated f3.jpg", res.Skipped)
	}
	if reason := skipReason(res, "f2.jpg"); !strings.Contains(reason, "已在队列中") {
		t.Errorf("reason for f2.jpg = %q, want 已在队列中", reason)
	}
	// The second f3.jpg request sees the task this same batch just wrote.
	if reason := skipReason(res, "f3.jpg"); !strings.Contains(reason, "已在队列中") {
		t.Errorf("reason for the repeated f3.jpg = %q, want 已在队列中", reason)
	}

	if got := tasksOf(t, failedLatest.FileID); len(got) != 2 {
		t.Errorf("f1.jpg tasks = %d, want the failed row plus one new queued row", len(got))
	}
	if got := tasksOf(t, runningLatest.FileID); len(got) != 1 {
		t.Errorf("f2.jpg tasks = %d, want the running row only", len(got))
	}
	if got := tasksOf(t, twice.FileID); len(got) != 1 {
		t.Errorf("f3.jpg tasks = %d, want one row for a path listed twice", len(got))
	}
}

// The two filter modes cannot share one database: every accepted EnqueueDir
// call leaves a live queued row behind, so a second call on the same files is
// answered by the in-flight rule instead of the path filter, and stops saying
// anything about recursion at all. setupTestDB swaps a package-global sqlite
// handle and never closes the one it replaces, so each mode gets its own test
// function with its own fresh DB rather than two setups inside one. Both calls
// use force=false, as the brief has them; the expected outcome is queueing,
// never a skip.
func TestEnqueueDir_NonRecursiveFilter(t *testing.T) {
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
	task, err := repoAI.NewRepository().FindTask(res.Queued[0])
	if err != nil || task == nil {
		t.Fatalf("find task %d: %v", res.Queued[0], err)
	}
	if task.FileID != shallow.FileID {
		t.Fatalf("non-recursive enqueued file id %d, want d/a.jpg id %d", task.FileID, shallow.FileID)
	}
	// The deeper child is filtered out of the batch, so it is neither queued
	// nor skipped: it never reaches the rule table at all.
	if got := tasksOf(t, deep.FileID); len(got) != 0 {
		t.Errorf("d/sub/b.jpg tasks = %d, want 0", len(got))
	}
	if got := tasksOf(t, outside.FileID); len(got) != 0 {
		t.Errorf("e/c.jpg tasks = %d, want 0", len(got))
	}
}

func TestEnqueueDir_RecursiveFilter(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	shallow := seedFile(t, "d/a.jpg")
	deep := seedFile(t, "d/sub/b.jpg")
	outside := seedFile(t, "e/c.jpg")
	svc := ai.NewService()

	// The trailing slash is part of the fixture: dirPrefix normalises it.
	res, err := svc.EnqueueDir("d/", true, false)
	if err != nil {
		t.Fatalf("enqueue dir recursive: %v", err)
	}
	if len(res.Queued) != 2 || len(res.Skipped) != 0 {
		t.Fatalf("recursive = queued %v skipped %+v, want d/a.jpg and d/sub/b.jpg", res.Queued, res.Skipped)
	}
	queuedIDs := map[uint]bool{}
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

// The retry button needs the same in-flight guard as the enqueue paths: the old
// failed row is never rewritten, so a second click would otherwise stack
// another queued task on the one the first click created.
func TestRetryTask_SkipsWhenFileAlreadyInFlight(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	target := seedFile(t, "a.jpg")
	failed := seedTask(t, target.FileID, ai.TaskFailed)

	svc := ai.NewService()
	if err := svc.RetryTask(failed.TaskID); err != nil {
		t.Fatalf("first retry: %v", err)
	}
	tasks := tasksOf(t, target.FileID)
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want the failed row plus one queued row", len(tasks))
	}
	inFlight := tasks[len(tasks)-1]
	if inFlight.TaskID <= failed.TaskID {
		t.Fatalf("newest task id = %d, want it newer than the retried %d", inFlight.TaskID, failed.TaskID)
	}
	if inFlight.Status != ai.TaskQueued {
		t.Fatalf("newest task status = %q, want %q", inFlight.Status, ai.TaskQueued)
	}
	// Proposals produced by the run that is now in flight must survive.
	seedSuggestions(t, inFlight.TaskID, target.FileID, ai.ReviewPending, 1)

	err := svc.RetryTask(failed.TaskID)
	if err == nil {
		t.Fatal("retrying while a newer queued task exists must return an error")
	}
	if !strings.Contains(err.Error(), "已有进行中的识别任务") {
		t.Errorf("retry error = %q, want it to name the in-flight task", err)
	}
	if got := tasksOf(t, target.FileID); len(got) != 2 {
		t.Errorf("tasks = %d, want no second queued row from the repeated click", len(got))
	}
	if pending := countSuggestions(t, target.FileID, ai.ReviewPending); pending != 1 {
		t.Errorf("pending suggestions = %d, want the in-flight run's proposals kept", pending)
	}

	// A newer task that is not in flight does not block the retry.
	finished := seedFile(t, "b.jpg")
	finishedFailed := seedTask(t, finished.FileID, ai.TaskFailed)
	seedTask(t, finished.FileID, ai.TaskDone)
	if err := svc.RetryTask(finishedFailed.TaskID); err != nil {
		t.Fatalf("retry with a newer finished task: %v", err)
	}
	if got := tasksOf(t, finished.FileID); len(got) != 3 {
		t.Errorf("b.jpg tasks = %d, want the failed, done and new queued rows", len(got))
	}
}

// The gate is a snapshot-then-insert with no database-level guard behind it, so
// before Task 11 serialized it two concurrent batches for one never-recognised
// file each read "no previous task" and both queued it: two model runs and two
// sets of duplicate pending proposals for one file. Unreachable while nothing
// called this over HTTP, reachable from two browser tabs the moment the WebUI
// API exists, hence a permanent guard rather than a one-off check.
func TestEnqueuePaths_ConcurrentCallersQueueOnce(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	target := seedFile(t, "race.jpg")
	svc := ai.NewService()

	const callers = 8
	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make([]*ai.EnqueueResult, callers)
	errs := make([]error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = svc.EnqueuePaths([]string{"race.jpg"}, false)
		}(i)
	}
	close(start)
	wg.Wait()

	queued := 0
	for i := 0; i < callers; i++ {
		if errs[i] != nil {
			t.Fatalf("caller %d: %v", i, errs[i])
		}
		queued += len(results[i].Queued)
	}
	if queued != 1 {
		t.Errorf("queued tasks = %d across %d concurrent callers, want exactly 1", queued, callers)
	}
	if got := tasksOf(t, target.FileID); len(got) != 1 {
		t.Errorf("task rows = %d, want 1", len(got))
	}
	// The losers are reported, not silently dropped: every other caller sees the
	// row the winner just wrote.
	skipped := 0
	for i := 0; i < callers; i++ {
		skipped += len(results[i].Skipped)
	}
	if skipped != callers-1 {
		t.Errorf("skipped = %d, want %d", skipped, callers-1)
	}
	for i := 0; i < callers; i++ {
		for _, s := range results[i].Skipped {
			if s.Reason != "已在队列中" {
				t.Errorf("skip reason = %q, want 已在队列中", s.Reason)
			}
		}
	}
}

// EnqueueDir shares the same critical section, and so does the retry button: a
// retry of the file a concurrent enqueue just queued must not stack a second
// row on top of it.
func TestEnqueueDir_ConcurrentWithRetryQueuesOnce(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	target := seedFile(t, "race2.jpg")
	failed := seedTask(t, target.FileID, ai.TaskFailed)
	svc := ai.NewService()

	var wg sync.WaitGroup
	dirErr := make([]error, 4)
	retryErr := error(nil)
	wg.Add(1)
	go func() {
		defer wg.Done()
		retryErr = svc.RetryTask(failed.TaskID)
	}()
	for i := 0; i < len(dirErr); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, dirErr[i] = svc.EnqueueDir(".", true, false)
		}(i)
	}
	wg.Wait()

	for i := range dirErr {
		if dirErr[i] != nil {
			t.Fatalf("enqueue dir %d: %v", i, dirErr[i])
		}
	}
	if retryErr != nil {
		t.Fatalf("retry: %v", retryErr)
	}
	// RetryTask queued the file, so every directory pass over it is a skip. The
	// one queued row is the point: not five.
	if got := tasksOf(t, target.FileID); len(got) != 2 {
		t.Errorf("task rows = %d, want the failed row plus exactly one queued row", len(got))
	}
}
