package ai_test

import (
	"os"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	repoAI "github.com/XingfenD/tag_file/internal/repository/ai"
	"github.com/XingfenD/tag_file/pkg/storage"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()
	dbPath := ".tag_file_test.db"
	os.Remove(dbPath)
	if err := storage.InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := storage.GetDB().AutoMigrate(
		&model.File{}, &model.TagGroup{}, &model.TagGroupTag{}, &model.TagCategory{}, &model.Tag{}, &model.TagValue{},
		&model.Config{}, &model.AITask{}, &model.AISuggestion{}, &model.AIFrameOutput{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return func() { os.Remove(dbPath) }
}

// The refine pipeline keeps every raw frame answer and marks a re-run task via
// ai_tasks, so both the table and the two columns have to exist before a worker
// starts writing them.
func TestAutoMigrate_FrameOutputTable(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	m := storage.GetDB().Migrator()
	if !m.HasTable(&model.AIFrameOutput{}) {
		t.Fatal("expected ai_frame_outputs table")
	}
	for _, column := range []string{"source_task_id", "refine_status"} {
		if !m.HasColumn(&model.AITask{}, column) {
			t.Fatalf("expected ai_tasks.%s column", column)
		}
	}
}

func TestAITaskRepo_ClaimNextQueued(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	repo := repoAI.NewRepository()
	first := &model.AITask{FileID: 1, Status: "queued"}
	second := &model.AITask{FileID: 2, Status: "queued"}
	if err := repo.CreateTask(first); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := repo.CreateTask(second); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if first.TaskID == 0 || second.TaskID == 0 {
		t.Fatalf("expected CreateTask to backfill TaskID, got %d and %d", first.TaskID, second.TaskID)
	}

	claimed, err := repo.ClaimNextQueued()
	if err != nil {
		t.Fatalf("claim first: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected a claimed task, got nil")
	}
	if claimed.TaskID != first.TaskID || claimed.FileID != 1 {
		t.Fatalf("expected smallest task_id (file_id 1), got task_id %d file_id %d", claimed.TaskID, claimed.FileID)
	}
	if claimed.Status != "running" {
		t.Fatalf("expected claimed task reported as running, got %q", claimed.Status)
	}
	stored, err := repo.FindTask(first.TaskID)
	if err != nil {
		t.Fatalf("find task: %v", err)
	}
	if stored == nil || stored.Status != "running" {
		t.Fatalf("expected running persisted, got %+v", stored)
	}

	claimedAgain, err := repo.ClaimNextQueued()
	if err != nil {
		t.Fatalf("claim second: %v", err)
	}
	if claimedAgain == nil {
		t.Fatal("expected the second queued task, got nil")
	}
	if claimedAgain.TaskID != second.TaskID || claimedAgain.Status != "running" {
		t.Fatalf("expected task_id %d running, got task_id %d status %q", second.TaskID, claimedAgain.TaskID, claimedAgain.Status)
	}

	empty, err := repo.ClaimNextQueued()
	if empty != nil || err != nil {
		t.Fatalf("expected (nil, nil) with nothing queued, got %+v, %v", empty, err)
	}

	missing, err := repo.FindTask(9999)
	if missing != nil || err != nil {
		t.Fatalf("expected (nil, nil) for missing task, got %+v, %v", missing, err)
	}
}

func TestAITaskRepo_FailRunning(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	repo := repoAI.NewRepository()
	running := &model.AITask{FileID: 1, Status: "running", Error: "半截错误"}
	queued := &model.AITask{FileID: 2, Status: "queued"}
	if err := repo.CreateTask(running); err != nil {
		t.Fatalf("create running task: %v", err)
	}
	if err := repo.CreateTask(queued); err != nil {
		t.Fatalf("create queued task: %v", err)
	}

	affected, err := repo.FailRunning("重启中断")
	if err != nil {
		t.Fatalf("fail running: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected 1 affected row, got %d", affected)
	}

	failed, err := repo.FindTask(running.TaskID)
	if err != nil {
		t.Fatalf("find failed task: %v", err)
	}
	if failed == nil || failed.Status != "failed" || failed.Error != "重启中断" {
		t.Fatalf("expected failed/重启中断, got %+v", failed)
	}

	untouched, err := repo.FindTask(queued.TaskID)
	if err != nil {
		t.Fatalf("find queued task: %v", err)
	}
	if untouched == nil || untouched.Status != "queued" || untouched.Error != "" {
		t.Fatalf("expected queued task untouched, got %+v", untouched)
	}

	again, err := repo.FailRunning("重启中断")
	if again != 0 || err != nil {
		t.Fatalf("expected 0 affected rows on second sweep, got %d, %v", again, err)
	}
}

func TestAITaskRepo_UpdateTask(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	repo := repoAI.NewRepository()
	task := &model.AITask{FileID: 1, Status: "queued", Error: "旧错误"}
	if err := repo.CreateTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := repo.UpdateTask(task).SetStatus("failed").SetError("模型不可达").SetFrameCount(3).Exec(); err != nil {
		t.Fatalf("update task: %v", err)
	}
	got, err := repo.FindTask(task.TaskID)
	if err != nil {
		t.Fatalf("find task: %v", err)
	}
	if got == nil || got.Status != "failed" || got.Error != "模型不可达" || got.FrameCount != 3 {
		t.Fatalf("expected failed/模型不可达/3, got %+v", got)
	}
	if got.FileID != 1 {
		t.Fatalf("expected unselected file_id to stay 1, got %d", got.FileID)
	}

	// an explicitly selected empty string must still be written (retry clears the old error)
	if err := repo.UpdateTask(got).SetError("").Exec(); err != nil {
		t.Fatalf("clear error: %v", err)
	}
	cleared, err := repo.FindTask(task.TaskID)
	if err != nil {
		t.Fatalf("find cleared task: %v", err)
	}
	if cleared == nil || cleared.Error != "" || cleared.Status != "failed" {
		t.Fatalf("expected empty error with status kept, got %+v", cleared)
	}
}

func TestAISuggestionRepo_ListAndCountPending(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	repo := repoAI.NewRepository()
	const fileA, fileB, taskID uint = 1, 2, 7
	items := []model.AISuggestion{
		{
			TaskID: taskID, FileID: fileA, CategoryName: "服饰", TagName: "裤子", TagValue: "黑色",
			FinalCategoryName: "服饰", FinalTagName: "裤子", FinalValue: "黑色", ReviewStatus: "pending",
		},
		{
			TaskID: taskID, FileID: fileA, CategoryName: "服饰", TagName: "上衣",
			FinalCategoryName: "服饰", FinalTagName: "上衣", ReviewStatus: "pending",
		},
		{
			TaskID: taskID, FileID: fileA, CategoryName: "服饰", TagName: "袜子",
			FinalCategoryName: "服饰", FinalTagName: "袜子", ReviewStatus: "rejected",
		},
		{
			TaskID: taskID, FileID: fileB, CategoryName: "数码", TagName: "手机",
			FinalCategoryName: "数码", FinalTagName: "手机", ReviewStatus: "pending",
		},
	}
	if err := repo.CreateSuggestions(items); err != nil {
		t.Fatalf("create suggestions: %v", err)
	}
	for i, item := range items {
		if item.SuggestionID == 0 {
			t.Fatalf("expected CreateSuggestions to backfill SuggestionID of item %d", i)
		}
	}

	pending, total, err := repo.ListSuggestions().ByStatus("pending").ByFileID(fileA).Exec()
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 2 || total != 2 {
		t.Fatalf("expected 2 pending rows and total 2, got %d rows, total %d", len(pending), total)
	}

	byTask, taskTotal, err := repo.ListSuggestions().ByTaskID(taskID).Exec()
	if err != nil {
		t.Fatalf("list by task: %v", err)
	}
	if len(byTask) != 4 || taskTotal != 4 {
		t.Fatalf("expected 4 rows for task %d, got %d rows, total %d", taskID, len(byTask), taskTotal)
	}

	page, pageTotal, err := repo.ListSuggestions().ByStatus("pending").ByFileID(fileA).Offset(1).Limit(1).Exec()
	if err != nil {
		t.Fatalf("list page: %v", err)
	}
	if len(page) != 1 || pageTotal != 2 {
		t.Fatalf("expected 1 row with total 2, got %d rows, total %d", len(page), pageTotal)
	}
	if page[0].SuggestionID != items[1].SuggestionID {
		t.Fatalf("expected offset 1 to skip suggestion %d, got %d", items[0].SuggestionID, page[0].SuggestionID)
	}

	all, allTotal, err := repo.ListSuggestions().Limit(0).Exec()
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 4 || allTotal != 4 {
		t.Fatalf("expected Limit(0) to be unlimited, got %d rows, total %d", len(all), allTotal)
	}

	counts, err := repo.CountPendingByFile()
	if err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if counts[fileA] != 2 {
		t.Fatalf("expected 2 pending for file %d, got %d", fileA, counts[fileA])
	}
	if counts[fileB] != 1 {
		t.Fatalf("expected 1 pending for file %d, got %d", fileB, counts[fileB])
	}
	if len(counts) != 2 {
		t.Fatalf("expected counts for 2 files, got %d entries", len(counts))
	}

	// Task 9 builds tags from Final* on approval, so these three columns must
	// round-trip exactly as written and never hold an empty name
	written := items[0]
	got, err := repo.FindSuggestion(written.SuggestionID)
	if err != nil {
		t.Fatalf("find suggestion: %v", err)
	}
	if got == nil {
		t.Fatal("expected to find the created suggestion")
	}
	if got.FinalCategoryName != written.FinalCategoryName || got.FinalTagName != written.FinalTagName || got.FinalValue != written.FinalValue {
		t.Fatalf("Final* must round-trip as written: got %q/%q/%q, want %q/%q/%q",
			got.FinalCategoryName, got.FinalTagName, got.FinalValue,
			written.FinalCategoryName, written.FinalTagName, written.FinalValue)
	}
	if got.FinalCategoryName == "" || got.FinalTagName == "" {
		t.Fatalf("Final* must hold effective values, got %q/%q", got.FinalCategoryName, got.FinalTagName)
	}
	if got.ReviewStatus != "pending" || got.ReviewedAt != nil || got.ApprovedTagID != nil {
		t.Fatalf("expected pending/unreviewed row, got %+v", got)
	}

	missing, err := repo.FindSuggestion(9999)
	if missing != nil || err != nil {
		t.Fatalf("expected (nil, nil) for missing suggestion, got %+v, %v", missing, err)
	}
}

func TestAISuggestionRepo_UpdateSuggestion(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	repo := repoAI.NewRepository()
	items := []model.AISuggestion{{
		TaskID: 1, FileID: 1, CategoryName: "服饰", TagName: "裤子", TagValue: "黑色",
		FinalCategoryName: "服饰", FinalTagName: "裤子", FinalValue: "黑色", ReviewStatus: "pending",
	}}
	if err := repo.CreateSuggestions(items); err != nil {
		t.Fatalf("create suggestions: %v", err)
	}
	suggestion := items[0]

	if err := repo.UpdateSuggestion(&suggestion).SetFinal("服饰", "长裤", "藏青").Exec(); err != nil {
		t.Fatalf("set final: %v", err)
	}
	renamed, err := repo.FindSuggestion(items[0].SuggestionID)
	if err != nil {
		t.Fatalf("find after rename: %v", err)
	}
	if renamed == nil {
		t.Fatal("expected to find the renamed suggestion")
	}
	if renamed.FinalCategoryName != "服饰" || renamed.FinalTagName != "长裤" || renamed.FinalValue != "藏青" {
		t.Fatalf("expected Final* 服饰/长裤/藏青, got %q/%q/%q", renamed.FinalCategoryName, renamed.FinalTagName, renamed.FinalValue)
	}
	// the original model output must stay untouched so review can compare it
	if renamed.CategoryName != "服饰" || renamed.TagName != "裤子" || renamed.TagValue != "黑色" {
		t.Fatalf("expected original CategoryName/TagName/TagValue untouched, got %q/%q/%q",
			renamed.CategoryName, renamed.TagName, renamed.TagValue)
	}
	if renamed.ReviewStatus != "pending" || renamed.ReviewedAt != nil {
		t.Fatalf("expected still pending and unreviewed, got %+v", renamed)
	}

	tagID := uint(42)
	if err := repo.UpdateSuggestion(renamed).SetReview("approved", &tagID).Exec(); err != nil {
		t.Fatalf("set review: %v", err)
	}
	approved, err := repo.FindSuggestion(items[0].SuggestionID)
	if err != nil {
		t.Fatalf("find after approve: %v", err)
	}
	if approved.ReviewStatus != "approved" || approved.ApprovedTagID == nil || *approved.ApprovedTagID != tagID {
		t.Fatalf("expected approved with tag id 42, got %+v", approved)
	}
	if approved.ReviewedAt == nil || approved.ReviewedAt.IsZero() {
		t.Fatalf("expected reviewed_at written by SetReview, got %v", approved.ReviewedAt)
	}
	if approved.FinalTagName != "长裤" {
		t.Fatalf("expected Final* kept after review, got %q", approved.FinalTagName)
	}

	if err := repo.UpdateSuggestion(approved).SetReview("rejected", nil).Exec(); err != nil {
		t.Fatalf("set review to rejected: %v", err)
	}
	rejected, err := repo.FindSuggestion(items[0].SuggestionID)
	if err != nil {
		t.Fatalf("find after reject: %v", err)
	}
	if rejected.ReviewStatus != "rejected" || rejected.ApprovedTagID != nil {
		t.Fatalf("expected rejected with cleared approved_tag_id, got %+v", rejected)
	}
}

func TestFrameOutput_CreateAndList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	repo := repoAI.NewRepository()
	task := &model.AITask{FileID: 1, Status: "completed"}
	if err := repo.CreateTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// insert out of frame order: the refine pass reads frames back in order
	outputs := []*model.AIFrameOutput{
		{TaskID: task.TaskID, FileID: 1, FrameIndex: 3, RawText: "第三帧"},
		{TaskID: task.TaskID, FileID: 1, FrameIndex: 1, RawText: "第一帧"},
		{TaskID: task.TaskID, FileID: 1, FrameIndex: 2, RawText: "第二帧"},
	}
	for i, o := range outputs {
		if err := repo.CreateFrameOutput(o); err != nil {
			t.Fatalf("create frame output %d: %v", i, err)
		}
		if o.OutputID == 0 {
			t.Fatalf("expected CreateFrameOutput to backfill OutputID of row %d", i)
		}
	}

	got, err := repo.ListFrameOutputs(task.TaskID)
	if err != nil {
		t.Fatalf("list frame outputs: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(got))
	}
	for i, want := range []int{1, 2, 3} {
		if got[i].FrameIndex != want {
			t.Fatalf("expected row %d ordered by frame_index %d, got %d", i, want, got[i].FrameIndex)
		}
	}
	if got[0].RawText != "第一帧" || got[1].RawText != "第二帧" || got[2].RawText != "第三帧" {
		t.Fatalf("expected RawText to round-trip in order, got %q/%q/%q", got[0].RawText, got[1].RawText, got[2].RawText)
	}

	// a task with no stored frames must answer a non-nil empty slice
	empty, err := repo.ListFrameOutputs(9999)
	if err != nil {
		t.Fatalf("list empty task: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("expected non-nil empty slice for unknown task, got %#v", empty)
	}
}

func TestRawCountsBySource(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	repo := repoAI.NewRepository()
	first := &model.AITask{FileID: 1, Status: "completed"}
	second := &model.AITask{FileID: 2, Status: "completed"}
	if err := repo.CreateTask(first); err != nil {
		t.Fatalf("create first task: %v", err)
	}
	if err := repo.CreateTask(second); err != nil {
		t.Fatalf("create second task: %v", err)
	}
	rows := []*model.AIFrameOutput{
		{TaskID: first.TaskID, FileID: 1, FrameIndex: 1, RawText: "a"},
		{TaskID: first.TaskID, FileID: 1, FrameIndex: 2, RawText: "b"},
		{TaskID: second.TaskID, FileID: 2, FrameIndex: 1, RawText: "c"},
	}
	for i, o := range rows {
		if err := repo.CreateFrameOutput(o); err != nil {
			t.Fatalf("create frame output %d: %v", i, err)
		}
	}

	counts, err := repo.RawCountsBySource([]uint{first.TaskID, second.TaskID, 9999})
	if err != nil {
		t.Fatalf("raw counts: %v", err)
	}
	if counts[first.TaskID] != 2 {
		t.Fatalf("expected 2 outputs for task %d, got %d", first.TaskID, counts[first.TaskID])
	}
	if counts[second.TaskID] != 1 {
		t.Fatalf("expected 1 output for task %d, got %d", second.TaskID, counts[second.TaskID])
	}
	if _, ok := counts[9999]; ok {
		t.Fatalf("expected task without outputs to be absent from map, got %#v", counts)
	}
	if len(counts) != 2 {
		t.Fatalf("expected counts for 2 tasks, got %d entries", len(counts))
	}

	empty, err := repo.RawCountsBySource(nil)
	if err != nil {
		t.Fatalf("raw counts with no sources: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty map for empty input, got %#v", empty)
	}
}

func TestDeleteTask_CascadesFrameOutputs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	repo := repoAI.NewRepository()
	task := &model.AITask{FileID: 1, Status: "completed"}
	if err := repo.CreateTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	outputs := []*model.AIFrameOutput{
		{TaskID: task.TaskID, FileID: 1, FrameIndex: 1, RawText: "第一帧"},
		{TaskID: task.TaskID, FileID: 1, FrameIndex: 2, RawText: "第二帧"},
	}
	for i, o := range outputs {
		if err := repo.CreateFrameOutput(o); err != nil {
			t.Fatalf("create frame output %d: %v", i, err)
		}
	}
	suggestions := []model.AISuggestion{
		{TaskID: task.TaskID, FileID: 1, CategoryName: "服饰", TagName: "裤子", FinalCategoryName: "服饰", FinalTagName: "裤子", ReviewStatus: "pending"},
		{TaskID: task.TaskID, FileID: 1, CategoryName: "服饰", TagName: "上衣", FinalCategoryName: "服饰", FinalTagName: "上衣", ReviewStatus: "approved"},
	}
	if err := repo.CreateSuggestions(suggestions); err != nil {
		t.Fatalf("create suggestions: %v", err)
	}

	affected, err := repo.DeleteTask(task.TaskID, "completed")
	if err != nil {
		t.Fatalf("delete task: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected 1 deleted task row, got %d", affected)
	}
	left, err := repo.ListFrameOutputs(task.TaskID)
	if err != nil {
		t.Fatalf("list frame outputs after delete: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("expected frame outputs cascaded away, got %d rows", len(left))
	}
	remaining, total, err := repo.ListSuggestions().ByTaskID(task.TaskID).Exec()
	if err != nil {
		t.Fatalf("list suggestions after delete: %v", err)
	}
	if len(remaining) != 1 || total != 1 || remaining[0].ReviewStatus != "approved" {
		t.Fatalf("expected only the approved suggestion to survive, got %+v (total %d)", remaining, total)
	}

	// a status-mismatched conditional delete answers 0 rows and must leave
	// the raw outputs alone: the worker may still be writing them
	other := &model.AITask{FileID: 2, Status: "completed"}
	if err := repo.CreateTask(other); err != nil {
		t.Fatalf("create other task: %v", err)
	}
	output := &model.AIFrameOutput{TaskID: other.TaskID, FileID: 2, FrameIndex: 1, RawText: "在途帧"}
	if err := repo.CreateFrameOutput(output); err != nil {
		t.Fatalf("create in-flight output: %v", err)
	}
	missed, err := repo.DeleteTask(other.TaskID, "queued")
	if err != nil {
		t.Fatalf("conditional delete: %v", err)
	}
	if missed != 0 {
		t.Fatalf("expected 0 rows deleted on status mismatch, got %d", missed)
	}
	survivors, err := repo.ListFrameOutputs(other.TaskID)
	if err != nil {
		t.Fatalf("list survivors: %v", err)
	}
	if len(survivors) != 1 || survivors[0].RawText != "在途帧" {
		t.Fatalf("expected the status-mismatched task's outputs untouched, got %#v", survivors)
	}
}

func TestUpdateTask_SetRefineStatus(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	repo := repoAI.NewRepository()
	task := &model.AITask{FileID: 1, Status: "completed", Error: "旧错误"}
	if err := repo.CreateTask(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := repo.UpdateTask(task).SetRefineStatus("refining").Exec(); err != nil {
		t.Fatalf("set refine status: %v", err)
	}
	got, err := repo.FindTask(task.TaskID)
	if err != nil {
		t.Fatalf("find task: %v", err)
	}
	if got == nil || got.RefineStatus != "refining" {
		t.Fatalf("expected refine_status refining, got %+v", got)
	}
	if got.Status != "completed" || got.Error != "旧错误" {
		t.Fatalf("expected status/error not overwritten, got status %q error %q", got.Status, got.Error)
	}
}

func TestAISuggestionRepo_DeletePendingByFile(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	repo := repoAI.NewRepository()
	const fileA, fileB uint = 1, 2
	items := []model.AISuggestion{
		{TaskID: 1, FileID: fileA, CategoryName: "服饰", TagName: "裤子", FinalCategoryName: "服饰", FinalTagName: "裤子", ReviewStatus: "pending"},
		{TaskID: 1, FileID: fileA, CategoryName: "服饰", TagName: "上衣", FinalCategoryName: "服饰", FinalTagName: "上衣", ReviewStatus: "pending"},
		{TaskID: 1, FileID: fileA, CategoryName: "服饰", TagName: "袜子", FinalCategoryName: "服饰", FinalTagName: "袜子", ReviewStatus: "rejected"},
		{TaskID: 1, FileID: fileB, CategoryName: "数码", TagName: "手机", FinalCategoryName: "数码", FinalTagName: "手机", ReviewStatus: "pending"},
	}
	if err := repo.CreateSuggestions(items); err != nil {
		t.Fatalf("create suggestions: %v", err)
	}

	deleted, err := repo.DeletePendingByFile(fileA)
	if err != nil {
		t.Fatalf("delete pending: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 deleted rows, got %d", deleted)
	}

	left, total, err := repo.ListSuggestions().ByFileID(fileA).Exec()
	if err != nil {
		t.Fatalf("list remaining: %v", err)
	}
	if len(left) != 1 || total != 1 || left[0].ReviewStatus != "rejected" {
		t.Fatalf("expected only the rejected row left for file %d, got %+v (total %d)", fileA, left, total)
	}

	counts, err := repo.CountPendingByFile()
	if err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if counts[fileA] != 0 {
		t.Fatalf("expected no pending left for file %d, got %d", fileA, counts[fileA])
	}
	if counts[fileB] != 1 {
		t.Fatalf("expected file %d pending untouched, got %d", fileB, counts[fileB])
	}
}
