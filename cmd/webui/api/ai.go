package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/XingfenD/tag_file/internal/model"
	aiSvc "github.com/XingfenD/tag_file/internal/service/ai"
	"github.com/gin-gonic/gin"
)

// Page sizes of the two AI list endpoints; the two-stage pattern is the one
// HandleListConfig uses (parsePagination with total 0 → query →
// computePagination with the real total).
const (
	aiTaskPageSize       = 5
	aiSuggestionPageSize = 20
)

// AITaskItem is the JSON representation of one recognition task.
//
// Error is the task row's Error column verbatim. It is NOT a failure flag: a
// run that produced more suggestions than the configured cap keeps the ones it
// stored, stays "done" and records the discarded remainder here. Only Status
// decides success or failure.
//
// PendingCount is the file's still-unreviewed suggestion count, which is the
// number the review queue needs: every path that re-queues a file clears its
// stale pending rows first, so in practice these are the proposals of the run
// this row belongs to.
//
// CreatedAt and UpdatedAt are RFC3339 in UTC — both always carry the "Z" suffix,
// so a consumer may sort or diff the strings without knowing the server's zone.
//
// SourceTaskID is 0 on a first-pass task and points at the original vision task
// on a refine run (always the first hop, never an in-between refine). RefineStatus
// is the row's stage-two marker verbatim ("" on a single-stage task). HasRaw
// reports whether the task has stored first-pass answers to re-refine or to show
// — for a refine run that is its source's answer, since a refine row never stores
// raws of its own.
type AITaskItem struct {
	TaskID       uint   `json:"task_id"`
	FileID       uint   `json:"file_id"`
	FilePath     string `json:"file_path"`
	FileName     string `json:"file_name"`
	Status       string `json:"status"`
	Error        string `json:"error"`
	FrameCount   int    `json:"frame_count"`
	PendingCount int64  `json:"pending_count"`
	SourceTaskID uint   `json:"source_task_id"`
	RefineStatus string `json:"refine_status"`
	HasRaw       bool   `json:"has_raw"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// AISuggestionItem is the JSON representation of one tag proposal. CategoryName
// / TagName / TagValue are what the model said and are never rewritten; the
// Final* values are the effective ones a reviewer edits, and they start out as
// a copy of what the model said (Task 8), so a row nobody touched has both
// halves equal. MatchedTagPath is the tag-tree path behind MatchedTagID, empty
// when the proposal matched no existing tag.
type AISuggestionItem struct {
	SuggestionID      uint   `json:"suggestion_id"`
	FileID            uint   `json:"file_id"`
	FilePath          string `json:"file_path"`
	CategoryName      string `json:"category_name"`
	TagName           string `json:"tag_name"`
	TagValue          string `json:"tag_value"`
	FinalCategoryName string `json:"final_category_name"`
	FinalTagName      string `json:"final_tag_name"`
	FinalValue        string `json:"final_value"`
	MatchedTagID      *uint  `json:"matched_tag_id"`
	MatchedTagPath    string `json:"matched_tag_path"`
	ReviewStatus      string `json:"review_status"`
}

// ---------------------------------------------------------------------------
// Tasks
// ---------------------------------------------------------------------------

// HandleEnqueueAITasks responds to POST /api/ai/tasks.
// Takes an explicit list of repeatable `paths`, or one `dir` (+ `recursive`),
// plus `force`. Both forms are relative to the scan root, exactly like
// File.FilePath and like `dir` on POST /api/files/scan; `.` means the root.
// `paths` wins when both are sent: `dir` and `recursive` are then ignored, not
// merged in, so a form that carries both must expect only its checked files.
// A file that the gate refuses is reported in `skipped` — a list of
// {path, reason} objects, serialized straight from ai.SkippedFile — so a
// partially accepted batch is a success, not an error.
func (rt *Router) HandleEnqueueAITasks(c *gin.Context) {
	paths := c.PostFormArray("paths")
	dir := c.PostForm("dir")
	force := c.PostForm("force") == "true"

	var (
		result *aiSvc.EnqueueResult
		err    error
	)
	switch {
	case len(paths) > 0:
		result, err = rt.AISvc.EnqueuePaths(paths, force)
	case dir != "":
		recursive := c.PostForm("recursive") == "true"
		result, err = rt.AISvc.EnqueueDir(dir, recursive, force)
	default:
		respond(c, "参数不完整: 需要 paths 或 dir", "error")
		return
	}
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}

	respond(c, fmt.Sprintf("已加入队列 %d 个文件, 跳过 %d 个", len(result.Queued), len(result.Skipped)), "success", map[string]any{
		"queued":  len(result.Queued),
		"skipped": result.Skipped,
	})
}

// HandleListAITasks responds to GET /api/ai/tasks.
// Supports ?status=X&page=N; an empty status lists every task. Ordered by
// task id, i.e. submission order.
func (rt *Router) HandleListAITasks(c *gin.Context) {
	pg := parsePagination(c, 0, aiTaskPageSize)
	offset := (pg.Page - 1) * pg.PageSize

	tasks, total, err := rt.AISvc.ListTasks(c.Query("status"), offset, pg.PageSize)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	pg = computePagination(total, pg.Page, pg.PageSize)

	pending, err := rt.AISvc.PendingCountByFile()
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}

	fileIDs := make([]uint, 0, len(tasks))
	for i := range tasks {
		fileIDs = append(fileIDs, tasks[i].FileID)
	}
	files, err := rt.filesOf(fileIDs)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}

	// has_raw follows the task that actually stores the answers — a refine run
	// reads its source's — and the whole page resolves in one aggregated read,
	// never one per row. Deduping the source ids keeps that IN list honest even
	// when a page stacks several refines onto the same vision task.
	sourceIDs := make([]uint, 0, len(tasks))
	seen := make(map[uint]bool, len(tasks))
	for i := range tasks {
		id := rawSourceID(&tasks[i])
		if !seen[id] {
			seen[id] = true
			sourceIDs = append(sourceIDs, id)
		}
	}
	rawCounts, err := rt.AISvc.RawCountsBySource(sourceIDs)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}

	items := make([]AITaskItem, 0, len(tasks))
	for i := range tasks {
		task := &tasks[i]
		file := files[task.FileID]
		items = append(items, AITaskItem{
			TaskID:       task.TaskID,
			FileID:       task.FileID,
			FilePath:     filePathOf(file),
			FileName:     fileNameOf(file),
			Status:       task.Status,
			Error:        task.Error,
			FrameCount:   task.FrameCount,
			PendingCount: pending[task.FileID],
			SourceTaskID: task.SourceTaskID,
			RefineStatus: task.RefineStatus,
			HasRaw:       rawCounts[rawSourceID(task)] > 0,
			// .UTC() is load-bearing, do not drop it as noise: created_at arrives
			// zone-naive UTC (the column's CURRENT_TIMESTAMP default; GORM omits
			// it from the INSERT) while GORM's auto-update writes updated_at as a
			// local offset-bearing time, so raw rendering spells one instant
			// "…Z" and the next "…+08:00" and breaks any string sort or duration
			// diff. UTC() respells the instant without moving it; the storage-side
			// half of the inconsistency predates this handler and is not its call.
			CreatedAt: task.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: task.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	respond(c, "ok", "success", map[string]any{
		"items":      items,
		"pagination": pg,
	})
}

// HandleRetryAITask responds to POST /api/ai/tasks/retry.
// A failed task is retryable, and so is a running one this process's worker is not
// executing; the reason (unknown id, a status that is neither, a live run, the file
// already having a newer run in flight) comes back as the error message.
func (rt *Router) HandleRetryAITask(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.PostForm("task_id"), 10, 64)
	if err != nil || taskID == 0 {
		respond(c, "无效的任务ID", "error")
		return
	}
	if err := rt.AISvc.RetryTask(uint(taskID)); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "任务已重新加入队列", "success")
}

// HandleRefineAITask responds to POST /api/ai/tasks/refine.
// Queues a 只重新整理 run over the raw answers the task's vision source already
// stored — no frames, no GPU. HandleRetryAITask's shape: every refusal (unknown
// id, a status that is neither done nor failed, no stored raws, the refine
// endpoint off, a live run, the file already having a newer run in flight) is
// the service's error message verbatim.
func (rt *Router) HandleRefineAITask(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.PostForm("task_id"), 10, 64)
	if err != nil || taskID == 0 {
		respond(c, "无效的任务ID", "error")
		return
	}
	if err := rt.AISvc.RefineTask(uint(taskID)); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "任务已加入重新整理队列", "success")
}

// AITaskRawItem is one stored first-pass answer, the row shape of the raw
// endpoint's payload. FrameIndex is the position in the video's frame sequence
// the items are ordered by.
type AITaskRawItem struct {
	FrameIndex int    `json:"frame_index"`
	RawText    string `json:"raw_text"`
}

// HandleAITaskRaw responds to GET /api/ai/tasks/:id/raw.
// Returns the first pass verbatim in frame order. A refine run stores no raws
// of its own, so the read follows rawSourceID and serves its vision source's —
// the same resolution has_raw and RefineTask use, so the list never advertises
// a raw an id-then-fetch would not deliver. An unknown id is the service-side
// wording, not an empty list.
func (rt *Router) HandleAITaskRaw(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respond(c, "无效的任务ID", "error")
		return
	}
	task, err := rt.AISvc.FindTask(uint(id))
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	if task == nil {
		respond(c, fmt.Sprintf("识别任务 %d 不存在", uint(id)), "error")
		return
	}
	outputs, err := rt.AISvc.ListFrameOutputs(rawSourceID(task))
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	items := make([]AITaskRawItem, 0, len(outputs))
	for i := range outputs {
		items = append(items, AITaskRawItem{
			FrameIndex: outputs[i].FrameIndex,
			RawText:    outputs[i].RawText,
		})
	}
	respond(c, "ok", "success", map[string]any{"items": items})
}

// HandleDeleteAITask responds to DELETE /api/ai/tasks/:id.
// Removes the task and its still-pending suggestions; reviewed proposals stay.
// A live running task is refused by the service, a stalled one is deletable.
func (rt *Router) HandleDeleteAITask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respond(c, "无效的任务ID", "error")
		return
	}
	if err := rt.AISvc.DeleteTask(uint(id)); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "任务已删除", "success")
}

// ---------------------------------------------------------------------------
// Suggestions
// ---------------------------------------------------------------------------

// HandleListAISuggestions responds to GET /api/ai/suggestions.
// Supports ?status=X&page=N&file_id=M; the status defaults to pending, which is
// the review queue. file_id, when present, must be a real id.
func (rt *Router) HandleListAISuggestions(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		status = aiSvc.ReviewPending
	}

	var fileID uint
	if raw := c.Query("file_id"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || parsed == 0 {
			respond(c, "无效的文件ID", "error")
			return
		}
		fileID = uint(parsed)
	}

	pg := parsePagination(c, 0, aiSuggestionPageSize)
	offset := (pg.Page - 1) * pg.PageSize

	suggestions, total, err := rt.AISvc.ListSuggestions(status, fileID, offset, pg.PageSize)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	pg = computePagination(total, pg.Page, pg.PageSize)

	fileIDs := make([]uint, 0, len(suggestions))
	tagIDs := make([]uint, 0, len(suggestions))
	for i := range suggestions {
		fileIDs = append(fileIDs, suggestions[i].FileID)
		if suggestions[i].MatchedTagID != nil {
			tagIDs = append(tagIDs, *suggestions[i].MatchedTagID)
		}
	}
	files, err := rt.filesOf(fileIDs)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	idToTag := rt.loadTagPathMap(tagIDs)

	items := make([]AISuggestionItem, 0, len(suggestions))
	for i := range suggestions {
		s := &suggestions[i]
		path := ""
		if s.MatchedTagID != nil {
			path = buildTagPath(*s.MatchedTagID, idToTag, 0)
		}
		items = append(items, AISuggestionItem{
			SuggestionID:      s.SuggestionID,
			FileID:            s.FileID,
			FilePath:          filePathOf(files[s.FileID]),
			CategoryName:      s.CategoryName,
			TagName:           s.TagName,
			TagValue:          s.TagValue,
			FinalCategoryName: s.FinalCategoryName,
			FinalTagName:      s.FinalTagName,
			FinalValue:        s.FinalValue,
			MatchedTagID:      s.MatchedTagID,
			MatchedTagPath:    path,
			ReviewStatus:      s.ReviewStatus,
		})
	}

	respond(c, "ok", "success", map[string]any{
		"items":      items,
		"pagination": pg,
	})
}

// HandleUpdateAISuggestion responds to PUT /api/ai/suggestions/:id.
// Rewrites the three effective values of a pending proposal. Only a pending row
// is editable; an approved or rejected one is refused by the service.
func (rt *Router) HandleUpdateAISuggestion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respond(c, "无效的建议ID", "error")
		return
	}
	if err := rt.AISvc.UpdateSuggestion(uint(id),
		c.PostForm("category_name"), c.PostForm("tag_name"), c.PostForm("tag_value")); err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, "建议已更新", "success")
}

// HandleApproveAISuggestions responds to POST /api/ai/suggestions/approve.
// Files the selected proposals: missing terms are created, the value is
// associated with the file. One bad id does not drag the batch down, it comes
// back in `failures` as a list of {id, msg} objects (ai.Failure as it is).
func (rt *Router) HandleApproveAISuggestions(c *gin.Context) {
	ids, ok := parseIDSlice(c, "ids")
	if !ok {
		return
	}
	approved, failures, err := rt.AISvc.Approve(ids)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, fmt.Sprintf("已通过 %d 条建议", approved), "success", map[string]any{
		"approved": approved,
		"failures": nonNilFailures(failures),
	})
}

// HandleRejectAISuggestions responds to POST /api/ai/suggestions/reject.
// Closes the selected proposals without touching the tag library; `failures`
// has the same {id, msg} shape as the approve response.
func (rt *Router) HandleRejectAISuggestions(c *gin.Context) {
	ids, ok := parseIDSlice(c, "ids")
	if !ok {
		return
	}
	rejected, failures, err := rt.AISvc.Reject(ids)
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	respond(c, fmt.Sprintf("已拒绝 %d 条建议", rejected), "success", map[string]any{
		"rejected": rejected,
		"failures": nonNilFailures(failures),
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseIDSlice reads a repeatable id field. Anything that is not a real id is
// malformed input rather than a business failure: the batch has no id to report
// in `failures` for a value that never was one, so the whole request is
// refused with the standard 400 shape. The second return reports that a
// response was already written.
func parseIDSlice(c *gin.Context, key string) ([]uint, bool) {
	raw := c.PostFormArray(key)
	if len(raw) == 0 {
		respond(c, "参数不完整: 未选择任何建议", "error")
		return nil, false
	}
	ids := make([]uint, 0, len(raw))
	for _, item := range raw {
		id, err := strconv.ParseUint(item, 10, 64)
		if err != nil || id == 0 {
			respond(c, "无效的建议ID: "+item, "error")
			return nil, false
		}
		ids = append(ids, uint(id))
	}
	return ids, true
}

// nonNilFailures keeps an all-good batch on `"failures":[]` instead of `null`:
// the service only allocates the slice when it appends, and a review page should
// not have to branch on two spellings of "nothing went wrong".
func nonNilFailures(failures []aiSvc.Failure) []aiSvc.Failure {
	if failures == nil {
		return []aiSvc.Failure{}
	}
	return failures
}

// rawSourceID resolves which task row actually stores the frame outputs behind
// this one: a refine run never has raws of its own and reads its first-pass
// source, every other task its own row. The source id is already normalized to
// the original vision task (RefineTask pins that), so one hop is the whole
// chain. The list's has_raw and the raw endpoint share this so they can never
// disagree about where the answers live.
func rawSourceID(task *model.AITask) uint {
	if task.SourceTaskID != 0 {
		return task.SourceTaskID
	}
	return task.TaskID
}

// filesOf loads the file rows behind a page of ids, deduplicated, because the
// AI tables only carry file ids and neither list is Preloaded. A file that has
// since left the index is absent from the result and renders as a blank path;
// its task and proposal rows stay visible so the review history is not lost.
func (rt *Router) filesOf(fileIDs []uint) (map[uint]*model.File, error) {
	files := make(map[uint]*model.File, len(fileIDs))
	seen := make(map[uint]bool, len(fileIDs))
	for _, id := range fileIDs {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		file, err := rt.FileSvc.FindByID(id)
		if err != nil {
			return nil, err
		}
		if file != nil {
			files[id] = file
		}
	}
	return files, nil
}

func filePathOf(file *model.File) string {
	if file == nil {
		return ""
	}
	return file.FilePath
}

func fileNameOf(file *model.File) string {
	if file == nil {
		return ""
	}
	return file.FileName
}
