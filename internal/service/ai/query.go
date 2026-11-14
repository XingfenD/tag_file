package ai

import "github.com/XingfenD/tag_file/internal/model"

// This file holds the read-only pass-throughs the WebUI API layer needs, so that
// cmd/webui/ talks to this Service and never to internal/repository/ai directly —
// the layering every other file under cmd/webui/api/ follows (AGENTS.md: services
// are imported aliased, repositories are not imported outside internal/).
//
// All three are pure delegation and add no behavior on purpose: each builds
// exactly the repository chain its handler call site used to build, in the same
// order, and returns the same (rows, total, err) / (map, err) triple. The moment
// one of them grows a filter, a default or a re-order, it has stopped being a
// pass-through and belongs next to its writer instead.

// ListTasks returns one page of recognition tasks ordered by task id plus the
// filtered total; status "" lists every task, limit 0 means unlimited.
// Delegates to Repository.ListTasks.
func (s *Service) ListTasks(status string, offset, limit int) ([]model.AITask, int64, error) {
	return s.aiRepo.ListTasks().ByStatus(status).Offset(offset).Limit(limit).Exec()
}

// ListSuggestions returns one page of tag proposals ordered by suggestion id plus
// the filtered total; status "" lists every review state and fileID 0 adds no
// file filter (the builder ignores a zero id, exactly as the caller's conditional
// ByFileID did). Delegates to Repository.ListSuggestions.
func (s *Service) ListSuggestions(status string, fileID uint, offset, limit int) ([]model.AISuggestion, int64, error) {
	return s.aiRepo.ListSuggestions().ByStatus(status).ByFileID(fileID).Offset(offset).Limit(limit).Exec()
}

// PendingCountByFile returns the count of still-pending suggestions per file
// (a file with none is absent from the map). Delegates to
// Repository.CountPendingByFile.
//
// This is the read a caller rendering AITaskItem's pending_count needs.
// FileStatuses performs the same read internally to choose between 待审 and
// 已识别 and deliberately does not go through this method: one derives a status
// label, the other reports a number, and keeping them apart means neither caller
// has to accept the other's shape. A request that wants both (the task list)
// therefore pays two reads rather than one shared snapshot.
func (s *Service) PendingCountByFile() (map[uint]int64, error) {
	return s.aiRepo.CountPendingByFile()
}
