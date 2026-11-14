package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/XingfenD/tag_file/internal/model"
	repoAI "github.com/XingfenD/tag_file/internal/repository/ai"
	aiSvc "github.com/XingfenD/tag_file/internal/service/ai"
	"github.com/gin-gonic/gin"
)

// aiPageSize is the page size of both AI list endpoints; the two-stage pattern
// is the one HandleListConfig uses (parsePagination with total 0 → query →
// computePagination with the real total).
const aiPageSize = 20

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
type AITaskItem struct {
	TaskID       uint   `json:"task_id"`
	FileID       uint   `json:"file_id"`
	FilePath     string `json:"file_path"`
	FileName     string `json:"file_name"`
	Status       string `json:"status"`
	Error        string `json:"error"`
	FrameCount   int    `json:"frame_count"`
	PendingCount int64  `json:"pending_count"`
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
// A file that the gate refuses is reported in `skipped` with the reason, so a
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
	pg := parsePagination(c, 0, aiPageSize)
	offset := (pg.Page - 1) * pg.PageSize

	tasks, total, err := repoAI.NewRepository().ListTasks().
		ByStatus(c.Query("status")).Offset(offset).Limit(pg.PageSize).Exec()
	if err != nil {
		respond(c, err.Error(), "error")
		return
	}
	pg = computePagination(total, pg.Page, pg.PageSize)

	pending, err := repoAI.NewRepository().CountPendingByFile()
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
			CreatedAt:    task.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    task.UpdatedAt.Format(time.RFC3339),
		})
	}

	respond(c, "ok", "success", map[string]any{
		"items":      items,
		"pagination": pg,
	})
}

// HandleRetryAITask responds to POST /api/ai/tasks/retry.
// Only a failed task is retryable; the reason (unknown id, not failed, the file
// already has a newer run in flight) comes back as the error message.
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

	pg := parsePagination(c, 0, aiPageSize)
	offset := (pg.Page - 1) * pg.PageSize

	op := repoAI.NewRepository().ListSuggestions().ByStatus(status).Offset(offset).Limit(pg.PageSize)
	if fileID > 0 {
		op = op.ByFileID(fileID)
	}
	suggestions, total, err := op.Exec()
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
// back in `failures`.
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
// Closes the selected proposals without touching the tag library.
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
