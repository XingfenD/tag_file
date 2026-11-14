package ai_test

import (
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	repoAI "github.com/XingfenD/tag_file/internal/repository/ai"
	"github.com/XingfenD/tag_file/internal/service/ai"
	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	fileSvc "github.com/XingfenD/tag_file/internal/service/file"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
)

// seedSuggestion inserts one pending suggestion in exactly the shape Task 8's
// pipeline writes it: the model's raw output and the effective Final* values hold
// the same text, review_status is pending, nothing has been created in the tag
// library. The task id stays 0 on purpose — the review gate reads the suggestion
// and the file, never the task, so a review test does not have to stage a
// recognition run to have something to review.
func seedSuggestion(t *testing.T, fileID uint, category, tag, value string) uint {
	t.Helper()
	items := []model.AISuggestion{{
		FileID:            fileID,
		CategoryName:      category,
		TagName:           tag,
		TagValue:          value,
		FinalCategoryName: category,
		FinalTagName:      tag,
		FinalValue:        value,
		ReviewStatus:      ai.ReviewPending,
	}}
	if err := repoAI.NewRepository().CreateSuggestions(items); err != nil {
		t.Fatalf("seed suggestion: %v", err)
	}
	return items[0].SuggestionID
}

// suggestionOf reads one suggestion back, failing the test when it cannot.
func suggestionOf(t *testing.T, id uint) *model.AISuggestion {
	t.Helper()
	got, err := repoAI.NewRepository().FindSuggestion(id)
	if err != nil {
		t.Fatalf("find suggestion %d: %v", id, err)
	}
	if got == nil {
		t.Fatalf("suggestion %d vanished", id)
	}
	return got
}

// setFinal rewrites a suggestion's effective values through the repository instead
// of through Service.UpdateSuggestion, so the tests that pin what approval does do
// not also depend on the edit path being right.
func setFinal(t *testing.T, id uint, category, tag, value string) {
	t.Helper()
	suggestion := suggestionOf(t, id)
	if err := repoAI.NewRepository().UpdateSuggestion(suggestion).SetFinal(category, tag, value).Exec(); err != nil {
		t.Fatalf("set final: %v", err)
	}
}

// fileTagState returns the file's associations and additionally fails when
// File.TagCnt disagrees with their count: TagCnt is the redundant counter the
// review gate is required to maintain through the file service, so every approval
// path is checked against both numbers, not just one of them.
func fileTagState(t *testing.T, filePath string, fileID uint) []model.TagGroupTag {
	t.Helper()
	fs := fileSvc.NewService()
	assoc, _, err := fs.ListFileTags(filePath, 0, 0)
	if err != nil {
		t.Fatalf("list tags of %s: %v", filePath, err)
	}
	got, err := fs.FindByID(fileID)
	if err != nil {
		t.Fatalf("find file %d: %v", fileID, err)
	}
	if got == nil {
		t.Fatalf("file %d vanished", fileID)
	}
	if got.TagCnt != len(assoc) {
		t.Fatalf("file %s: TagCnt = %d but associations = %d", filePath, got.TagCnt, len(assoc))
	}
	return assoc
}

// libraryNames is every tag name in one category, for the "nothing else was
// created" half of an assertion.
func libraryNames(t *testing.T, categoryName string) []string {
	t.Helper()
	tags, _, err := tagSvc.NewService().ListTags(categoryName, 0, 0)
	if err != nil {
		t.Fatalf("list tags of %s: %v", categoryName, err)
	}
	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		names = append(names, tag.TagName)
	}
	return names
}

// failureText joins the messages of a batch result so an assertion on its wording
// reads as one string.
func failureText(failures []ai.Failure) string {
	parts := make([]string, 0, len(failures))
	for _, failure := range failures {
		parts = append(parts, failure.Msg)
	}
	return strings.Join(parts, "; ")
}

// The approvals below drive the real file service, which logs 「find default group:
// record not found」 for a file that has no default tag group yet, right before it
// creates one (internal/service/file/tag.go:29-37) — the first tag of any
// never-tagged file, manual tagging included. The review gate itself logs nothing:
// every outcome it has is a return value. So each test that approves captures the
// standard logger with the package's existing captureLogs helper rather than letting
// that line into `go test` output, and
// TestApprove_CreatesTagAndAssociatesValue asserts the captured buffer holds nothing
// of the gate's own.

func TestApprove_CreatesTagAndAssociatesValue(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	logs := captureLogs(t)
	f := seedFile(t, "clothes/pants.jpg")
	if _, err := categorySvc.NewService().AddCategory("服饰", ""); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	id := seedSuggestion(t, f.FileID, "服饰", "裤子", "黑色")

	n, failures, err := ai.NewService().Approve([]uint{id})
	if err != nil || n != 1 || len(failures) != 0 {
		t.Fatalf("Approve = (%d, %v, %v), want (1, [], nil)", n, failures, err)
	}
	tags, _, err := tagSvc.NewService().ListTags("服饰", 0, 0)
	if err != nil || len(tags) != 1 || tags[0].TagName != "裤子" {
		t.Fatalf("expected new tag 裤子, got %v (%v)", tags, err)
	}
	vals, err := tagSvc.NewService().ListTagValues("裤子")
	if err != nil || len(vals) != 0 {
		t.Fatalf("approved value must not enter predefined dictionary, got %v", vals)
	}
	fs := fileSvc.NewService()
	assoc, _, err := fs.ListFileTags(f.FilePath, 0, 0)
	if err != nil || len(assoc) != 1 || assoc[0].TagValue != "黑色" {
		t.Fatalf("expected association value 黑色, got %v (%v)", assoc, err)
	}
	got, _ := fs.FindByID(f.FileID)
	if got.TagCnt != 1 {
		t.Fatalf("TagCnt = %d, want 1", got.TagCnt)
	}

	// The gate also owns the review record itself: approved, with the tag it landed
	// on and a review timestamp, while the model's raw words stay on the row.
	reviewed := suggestionOf(t, id)
	if reviewed.ReviewStatus != ai.ReviewApproved {
		t.Fatalf("review_status = %q, want %q", reviewed.ReviewStatus, ai.ReviewApproved)
	}
	if reviewed.ApprovedTagID == nil || *reviewed.ApprovedTagID != tags[0].TagID {
		t.Fatalf("approved_tag_id = %v, want %d", reviewed.ApprovedTagID, tags[0].TagID)
	}
	if reviewed.ReviewedAt == nil {
		t.Fatal("expected reviewed_at written on approve")
	}
	if reviewed.CategoryName != "服饰" || reviewed.TagName != "裤子" || reviewed.TagValue != "黑色" {
		t.Fatalf("raw output must stay as the model wrote it, got %q/%q/%q",
			reviewed.CategoryName, reviewed.TagName, reviewed.TagValue)
	}
	// A successful approval says everything by returning it: nothing of the review
	// gate's own is left in the log.
	if hits := logs.containing("[ai]"); len(hits) != 0 {
		t.Fatalf("the review gate must not log, got %v", hits)
	}
}

func TestApprove_ReusesExistingTagAndDedupesAcrossFiles(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	captureLogs(t)
	left := seedFile(t, "clothes/socks_left.jpg")
	right := seedFile(t, "clothes/socks_right.jpg")
	if _, err := categorySvc.NewService().AddCategory("服饰", ""); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	leftID := seedSuggestion(t, left.FileID, "服饰", "袜子", "黑色")
	rightID := seedSuggestion(t, right.FileID, "服饰", "袜子", "黑色")

	// Two separate batches for two files with the same brand new term: the second
	// approval must find the tag the first one created and reuse it (spec §8 第 5 条).
	for _, id := range []uint{leftID, rightID} {
		n, failures, err := ai.NewService().Approve([]uint{id})
		if err != nil || n != 1 || len(failures) != 0 {
			t.Fatalf("Approve(%d) = (%d, %v, %v), want (1, [], nil)", id, n, failures, err)
		}
	}

	tags, _, err := tagSvc.NewService().ListTags("服饰", 0, 0)
	if err != nil || len(tags) != 1 || tags[0].TagName != "袜子" {
		t.Fatalf("expected exactly one tag 袜子, got %v (%v)", tags, err)
	}
	for _, suggestionID := range []uint{leftID, rightID} {
		reviewed := suggestionOf(t, suggestionID)
		if reviewed.ApprovedTagID == nil || *reviewed.ApprovedTagID != tags[0].TagID {
			t.Fatalf("suggestion %d: approved_tag_id = %v, want the one shared tag %d",
				suggestionID, reviewed.ApprovedTagID, tags[0].TagID)
		}
	}

	for _, f := range []*model.File{left, right} {
		assoc := fileTagState(t, f.FilePath, f.FileID)
		if len(assoc) != 1 || assoc[0].TagID != tags[0].TagID || assoc[0].TagValue != "黑色" {
			t.Fatalf("file %s: expected one association to tag %d with value 黑色, got %v", f.FilePath, tags[0].TagID, assoc)
		}
	}
}

// TestApprove_DuplicatePendingRowsShareOneAssociation pins the alreadyTagged
// pre-check's reason for existing: two pending rows for the SAME file resolving to
// the same (tag, value) — what buildSuggestions produces for a repeated model term,
// and what re-recognising an approved file produces for an already-filed pair. The
// first approval files the association; the second must skip TagFile and STILL close
// its own review row. A SetReview moved inside the !associated branch, or an early
// exit there, leaves such rows pending forever behind 「文件已存在标签」 while every
// other test stays green — this is the one that would not.
func TestApprove_DuplicatePendingRowsShareOneAssociation(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	captureLogs(t)
	f := seedFile(t, "clothes/pants.jpg")
	if _, err := categorySvc.NewService().AddCategory("服饰", ""); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	first := seedSuggestion(t, f.FileID, "服饰", "裤子", "黑色")
	second := seedSuggestion(t, f.FileID, "服饰", "裤子", "黑色")

	// One batch, both rows: the second item reaches the dedupe-skip branch inside
	// the same walk, and the batch must still count it as a success.
	n, failures, err := ai.NewService().Approve([]uint{first, second})
	if err != nil || n != 2 || len(failures) != 0 {
		t.Fatalf("Approve = (%d, %v, %v), want (2, [], nil)", n, failures, err)
	}

	// Exactly one association, and TagCnt agrees with it (fileTagState pins both).
	assoc := fileTagState(t, f.FilePath, f.FileID)
	if len(assoc) != 1 || assoc[0].TagValue != "黑色" {
		t.Fatalf("file %s: expected one association with value 黑色, got %v", f.FilePath, assoc)
	}

	// That one association lives in the default group — the group TagFile writes
	// into and the group alreadyTagged reads, so skip and write agree on it.
	groups, err := fileSvc.NewService().ListGroups(f.FilePath)
	if err != nil {
		t.Fatalf("list groups of %s: %v", f.FilePath, err)
	}
	var defaultGroupID uint
	for _, group := range groups {
		if group.IsDefault {
			defaultGroupID = group.GroupID
		}
	}
	if defaultGroupID == 0 {
		t.Fatalf("approval must have left a default group on %s, got %v", f.FilePath, groups)
	}
	if assoc[0].GroupID != defaultGroupID {
		t.Fatalf("association sits in group %d, want the default group %d", assoc[0].GroupID, defaultGroupID)
	}

	// Both rows end approved and point at the one shared tag: skipping the
	// association is never skipping the review record.
	tags, _, err := tagSvc.NewService().ListTags("服饰", 0, 0)
	if err != nil || len(tags) != 1 || tags[0].TagName != "裤子" {
		t.Fatalf("expected exactly one tag 裤子, got %v (%v)", tags, err)
	}
	if assoc[0].TagID != tags[0].TagID {
		t.Fatalf("association points at tag %d, want the one library tag %d", assoc[0].TagID, tags[0].TagID)
	}
	for _, suggestionID := range []uint{first, second} {
		reviewed := suggestionOf(t, suggestionID)
		if reviewed.ReviewStatus != ai.ReviewApproved {
			t.Fatalf("suggestion %d: review_status = %q, want approved — the dedupe-skip path must still write the review record",
				suggestionID, reviewed.ReviewStatus)
		}
		if reviewed.ApprovedTagID == nil || *reviewed.ApprovedTagID != tags[0].TagID {
			t.Fatalf("suggestion %d: approved_tag_id = %v, want the one shared tag %d",
				suggestionID, reviewed.ApprovedTagID, tags[0].TagID)
		}
	}
}

func TestApprove_TwiceIsNoOpAndKeepsTagCnt(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	captureLogs(t)
	first := seedFile(t, "clothes/pants.jpg")
	second := seedFile(t, "clothes/shirt.jpg")
	if _, err := categorySvc.NewService().AddCategory("服饰", ""); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	id := seedSuggestion(t, first.FileID, "服饰", "裤子", "黑色")
	freshID := seedSuggestion(t, second.FileID, "服饰", "衬衫", "白色")

	if n, failures, err := ai.NewService().Approve([]uint{id}); err != nil || n != 1 || len(failures) != 0 {
		t.Fatalf("first Approve = (%d, %v, %v), want (1, [], nil)", n, failures, err)
	}

	// A terminal review status is terminal: approving the same id again names it as
	// a failure and changes nothing — no second association, no second TagCnt step,
	// no second tag (Review Focus #2).
	n, failures, err := ai.NewService().Approve([]uint{id})
	if err != nil {
		t.Fatalf("second Approve returned error: %v", err)
	}
	if n != 0 {
		t.Fatalf("second Approve counted %d successes, want 0", n)
	}
	if len(failures) != 1 || failures[0].ID != id || !strings.Contains(failures[0].Msg, "已审核") {
		t.Fatalf("second Approve failures = %v, want one 已审核 for id %d", failures, id)
	}

	// The batch keeps going past that failure: the untouched id next to it still
	// lands, and so does the walk over an id that has no row at all — both are
	// failures about one item, and the batch-level error stays nil.
	missingID := uint(100000)
	n, failures, err = ai.NewService().Approve([]uint{id, missingID, freshID})
	if err != nil {
		t.Fatalf("mixed Approve returned error: %v", err)
	}
	if n != 1 {
		t.Fatalf("mixed Approve counted %d successes, want 1 (one bad id must not drag the batch down)", n)
	}
	if len(failures) != 2 {
		t.Fatalf("mixed Approve failures = %v, want the 已审核 one and the missing-row one", failures)
	}
	if failures[0].ID != id || !strings.Contains(failures[0].Msg, "已审核") {
		t.Fatalf("mixed Approve failures[0] = %v, want 已审核 for id %d", failures[0], id)
	}
	if failures[1].ID != missingID || !strings.Contains(failures[1].Msg, "不存在") {
		t.Fatalf("mixed Approve failures[1] = %v, want a missing-row note for id %d", failures[1], missingID)
	}

	assoc := fileTagState(t, first.FilePath, first.FileID)
	if len(assoc) != 1 || assoc[0].TagValue != "黑色" {
		t.Fatalf("file %s: associations = %v, want the single 黑色 one", first.FilePath, assoc)
	}
	if names := libraryNames(t, "服饰"); len(names) != 2 {
		t.Fatalf("服饰 must hold exactly 裤子 and 衬衫, got %v", names)
	}
}

func TestApprove_MissingCategoryFailsWithoutSideEffects(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	f := seedFile(t, "clothes/pants.jpg")
	if _, err := categorySvc.NewService().AddCategory("服饰", ""); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	id := seedSuggestion(t, f.FileID, "服饰", "裤子", "黑色")
	setFinal(t, id, "服饰备用", "裤子", "黑色")

	n, failures, err := ai.NewService().Approve([]uint{id})
	if err != nil {
		t.Fatalf("Approve returned error for one unknown category: %v", err)
	}
	if n != 0 || len(failures) != 1 || failures[0].ID != id {
		t.Fatalf("Approve = (%d, %v, nil), want (0, [one failure for %d], nil)", n, failures, id)
	}
	if !strings.Contains(failures[0].Msg, "不存在") {
		t.Fatalf("failure text = %q, want it to name the missing category", failureText(failures))
	}

	// Nothing at all happened on the side (Review Focus #3): the suggestion is still
	// pending and editable, no category or tag was invented, the file is unchanged.
	reviewed := suggestionOf(t, id)
	if reviewed.ReviewStatus != ai.ReviewPending || reviewed.ApprovedTagID != nil || reviewed.ReviewedAt != nil {
		t.Fatalf("failed approval must leave the suggestion pending, got %+v", reviewed)
	}
	categories, _, err := categorySvc.NewService().ListCategories(0, 0)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	for _, category := range categories {
		if category.CategoryName == "服饰备用" {
			t.Fatal("a missing category must not be created by approval")
		}
	}
	if names := libraryNames(t, "服饰"); len(names) != 0 {
		t.Fatalf("no tag may be created for a failed approval, got %v", names)
	}
	if assoc := fileTagState(t, f.FilePath, f.FileID); len(assoc) != 0 {
		t.Fatalf("file must stay untagged, got %v", assoc)
	}
}

func TestApprove_SameTagNameInOtherCategoryCreatesSeparateTag(t *testing.T) {
	// This name is pinned by the plan; what the pinned reality now asserts is the
	// refusal — a same-named tag under a second category cannot be created at all,
	// so what this test proves is that another category's tag is never borrowed
	// (the plan-vs-reality reconciliation lives in Task 15's acceptance doc).
	cleanup := setupTestDB(t)
	defer cleanup()
	f := seedFile(t, "clothes/cat.jpg")
	if _, err := categorySvc.NewService().AddCategory("动物", ""); err != nil {
		t.Fatalf("seed category 动物: %v", err)
	}
	if _, err := categorySvc.NewService().AddCategory("植物", ""); err != nil {
		t.Fatalf("seed category 植物: %v", err)
	}
	existing, err := tagSvc.NewService().AddTag("动物", "猫", "", "")
	if err != nil {
		t.Fatalf("seed tag: %v", err)
	}
	id := seedSuggestion(t, f.FileID, "动物", "猫", "橘色")
	setFinal(t, id, "植物", "猫", "橘色")

	// The brief pins this as "在 植物 下新建「猫」, 两条 TagID 不同". That outcome is
	// not reachable through the sanctioned creation path, and the assertion is
	// therefore NOT pinned here — see the report. tagSvc.AddTag checks duplicates with
	// the repository's ByParentAndName (internal/repository/tag/find.go), which filters
	// on (parent_id IS NULL, tag_name) and not on category_id, so a root tag name is
	// unique across the whole library and AddTag("植物", "猫") answers
	// 「标签 '猫' 在此层级已存在」. Creating the tag around AddTag instead is forbidden
	// by spec §7 (新标签创建必须走现有 tag service). What this test does pin is the
	// half of Review Focus #3 that is real: the lookup is scoped to the reviewer's
	// category, so approving under 植物 never silently borrows 动物's tag, and the
	// refusal is reported as a failure that leaves everything where it was.
	n, failures, err := ai.NewService().Approve([]uint{id})
	if err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}
	if n != 0 || len(failures) != 1 || failures[0].ID != id {
		t.Fatalf("Approve = (%d, %v, nil), want (0, [one failure for %d], nil)", n, failures, id)
	}
	if !strings.Contains(failures[0].Msg, "已存在") {
		t.Fatalf("failure text = %q, want the tag service's duplicate-name reason", failureText(failures))
	}

	if names := libraryNames(t, "植物"); len(names) != 0 {
		t.Fatalf("植物 must hold no new tag, got %v", names)
	}
	// The tag of the other category is exactly as it was: same id, same name, and
	// nothing associated to it.
	tagsAnimal, _, err := tagSvc.NewService().ListTags("动物", 0, 0)
	if err != nil || len(tagsAnimal) != 1 || tagsAnimal[0].TagID != existing.TagID || tagsAnimal[0].TagName != "猫" {
		t.Fatalf("动物's 猫 must be untouched, got %v (%v)", tagsAnimal, err)
	}
	reviewed := suggestionOf(t, id)
	if reviewed.ReviewStatus != ai.ReviewPending || reviewed.ApprovedTagID != nil {
		t.Fatalf("the refused suggestion must stay pending and unlinked, got %+v", reviewed)
	}
	// fileTagState pins both halves: no association at all, and TagCnt agreeing with it.
	if assoc := fileTagState(t, f.FilePath, f.FileID); len(assoc) != 0 {
		t.Fatalf("file must stay untagged, got %v", assoc)
	}
}

func TestUpdateSuggestion_OnlyFinalFieldsAndPendingOnly(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	captureLogs(t)
	f := seedFile(t, "clothes/pants.jpg")
	if _, err := categorySvc.NewService().AddCategory("服饰", ""); err != nil {
		t.Fatalf("seed category 服饰: %v", err)
	}
	if _, err := categorySvc.NewService().AddCategory("配饰", ""); err != nil {
		t.Fatalf("seed category 配饰: %v", err)
	}
	id := seedSuggestion(t, f.FileID, "服饰", "裤子", "黑色")

	if err := ai.NewService().UpdateSuggestion(id, "配饰", "手链", "银色"); err != nil {
		t.Fatalf("UpdateSuggestion: %v", err)
	}
	edited := suggestionOf(t, id)
	// 模型的原始输出永远留着, 审核页要能对比说过什么、改成了什么。
	if edited.CategoryName != "服饰" || edited.TagName != "裤子" || edited.TagValue != "黑色" {
		t.Fatalf("raw output must stay untouched, got %q/%q/%q",
			edited.CategoryName, edited.TagName, edited.TagValue)
	}
	if edited.FinalCategoryName != "配饰" || edited.FinalTagName != "手链" || edited.FinalValue != "银色" {
		t.Fatalf("Final* = %q/%q/%q, want 配饰/手链/银色",
			edited.FinalCategoryName, edited.FinalTagName, edited.FinalValue)
	}
	if edited.ReviewStatus != ai.ReviewPending {
		t.Fatalf("editing must not review anything, review_status = %q", edited.ReviewStatus)
	}

	// The edit is what approval acts on, and only the edit: 手链 lands under 配饰.
	if n, failures, err := ai.NewService().Approve([]uint{id}); err != nil || n != 1 || len(failures) != 0 {
		t.Fatalf("Approve = (%d, %v, %v), want (1, [], nil)", n, failures, err)
	}
	if names := libraryNames(t, "配饰"); len(names) != 1 || names[0] != "手链" {
		t.Fatalf("配饰 must hold exactly the edited 手链, got %v", names)
	}
	if names := libraryNames(t, "服饰"); len(names) != 0 {
		t.Fatalf("the original category must gain nothing, got %v", names)
	}

	// An already reviewed row is frozen: editing it is refused, loudly.
	if err := ai.NewService().UpdateSuggestion(id, "服饰", "裤子", "黑色"); err == nil {
		t.Fatal("expected updating an approved suggestion to fail")
	}
	frozen := suggestionOf(t, id)
	if frozen.FinalTagName != "手链" || frozen.ReviewStatus != ai.ReviewApproved {
		t.Fatalf("the refused edit must not have landed, got %+v", frozen)
	}
}

func TestReject_MarksRejectedWithoutTagging(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	f := seedFile(t, "clothes/pants.jpg")
	if _, err := categorySvc.NewService().AddCategory("服饰", ""); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	id := seedSuggestion(t, f.FileID, "服饰", "裤子", "黑色")

	n, failures, err := ai.NewService().Reject([]uint{id})
	if err != nil || n != 1 || len(failures) != 0 {
		t.Fatalf("Reject = (%d, %v, %v), want (1, [], nil)", n, failures, err)
	}

	reviewed := suggestionOf(t, id)
	if reviewed.ReviewStatus != ai.ReviewRejected {
		t.Fatalf("review_status = %q, want %q", reviewed.ReviewStatus, ai.ReviewRejected)
	}
	if reviewed.ApprovedTagID != nil {
		t.Fatalf("a rejection must not point at a tag, got %v", *reviewed.ApprovedTagID)
	}
	if reviewed.ReviewedAt == nil {
		t.Fatal("expected reviewed_at written on reject")
	}
	// Nothing was tagged, anywhere: no tag in the library at all, no association,
	// no preset value and TagCnt still 0.
	all, total, err := tagSvc.NewService().ListTags("", 0, 0)
	if err != nil || total != 0 || len(all) != 0 {
		t.Fatalf("rejecting must create no tag, got %v (%v)", all, err)
	}
	if assoc := fileTagState(t, f.FilePath, f.FileID); len(assoc) != 0 {
		t.Fatalf("file must stay untagged, got %v", assoc)
	}
	// Rejection is terminal in both directions: approving afterwards is the same
	// failure, and the row stays rejected.
	if n, failures, err := ai.NewService().Approve([]uint{id}); err != nil || n != 0 || len(failures) != 1 ||
		!strings.Contains(failures[0].Msg, "已审核") {
		t.Fatalf("Approve after Reject = (%d, %v, %v), want (0, [one 已审核], nil)", n, failures, err)
	}
	if suggestionOf(t, id).ReviewStatus != ai.ReviewRejected {
		t.Fatal("a rejected suggestion must not be flip-able to approved")
	}
}

func TestApprove_ValueWithColonRoundTrips(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	captureLogs(t)
	f := seedFile(t, "clothes/socks.jpg")
	if _, err := categorySvc.NewService().AddCategory("服饰", ""); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	id := seedSuggestion(t, f.FileID, "服饰", "袜子", "黑色:长袜")

	if n, failures, err := ai.NewService().Approve([]uint{id}); err != nil || n != 1 || len(failures) != 0 {
		t.Fatalf("Approve = (%d, %v, %v), want (1, [], nil)", n, failures, err)
	}

	// The value is one string end to end: the colon is data, not a separator, so
	// 黑色:长袜 must arrive whole (spec §5 标签值).
	assoc := fileTagState(t, f.FilePath, f.FileID)
	if len(assoc) != 1 {
		t.Fatalf("expected one association, got %v", assoc)
	}
	if assoc[0].TagValue != "黑色:长袜" {
		t.Fatalf("associated value = %q, want 黑色:长袜", assoc[0].TagValue)
	}
	if got := suggestionOf(t, id); got.FinalValue != "黑色:长袜" || got.TagValue != "黑色:长袜" {
		t.Fatalf("stored suggestion lost the value: %+v", got)
	}
	vals, err := tagSvc.NewService().ListTagValues("袜子")
	if err != nil || len(vals) != 0 {
		t.Fatalf("approved value must not enter predefined dictionary, got %v (%v)", vals, err)
	}
}
