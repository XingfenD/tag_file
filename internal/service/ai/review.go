package ai

import (
	"fmt"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
)

// Failure is one id of a batch review call that did not land, plus the reason in
// words a reviewer can read. The id is a field rather than part of the text because
// the review page already knows which row it sent: what it has to show is why that
// row did not become a tag.
//
// The WebUI API serializes this struct as it is into the `failures` array of
// POST /api/ai/suggestions/approve and /reject, so the json tags below are that
// endpoint's wire contract (snake_case, like every other key it emits) and
// renaming a field here renames the JSON key.
//
// A Failure always describes one item. The error a batch call returns as its last
// value describes the operation instead: it is non-nil only when the review could
// not tell anything about the batch at all (a read that failed, not a read that
// found nothing), in which case the walk stops and everything already landed is
// still reported — there is no transaction here and no pretending otherwise.
type Failure struct {
	ID  uint   `json:"id"`
	Msg string `json:"msg"`
}

// The two reasons an id is refused before any work is done. Kept as constants
// because Approve and Reject must word the same gate the same way.
const (
	msgMissingSuggestion = "建议不存在"
	msgAlreadyReviewed   = "建议已审核"
)

// UpdateSuggestion rewrites the effective values of one pending suggestion: what a
// reviewer fixes on the review page (name, category, value) without changing what
// the model said. The raw CategoryName/TagName/TagValue columns are never written
// again after the pipeline inserted them (Task 8), so the comparison stays possible
// for as long as the row is unreviewed.
//
// Only a pending row is editable. Once a suggestion is approved its Final* values are
// the description of a real association, and editing them would make the row lie
// about what was filed; the same holds for a rejected row.
//
// The effective values are trimmed here, at write time — that is the deliberate
// single source of truth for whitespace: what is stored is what Approve reads, what
// the tag library is named, and what the reviewer sees again on reload. The other
// writer of Final* is Task 8's pipeline, whose input Task 7's parser already trims,
// so nothing in the database ends up padded through this service. resolveOrCreateTag
// correspondingly trims only inside its emptiness guards and never rewrites the
// string it looks a tag up or creates a tag under (see there).
func (s *Service) UpdateSuggestion(id uint, categoryName, tagName, tagValue string) error {
	suggestion, err := s.aiRepo.FindSuggestion(id)
	if err != nil {
		return fmt.Errorf("读取建议 %d 失败: %w", id, err)
	}
	if suggestion == nil {
		return fmt.Errorf("建议 %d 不存在", id)
	}
	if suggestion.ReviewStatus != ReviewPending {
		return fmt.Errorf("建议 %d 已审核(当前 %s), 不能再修改生效值", id, suggestion.ReviewStatus)
	}
	categoryName = strings.TrimSpace(categoryName)
	tagName = strings.TrimSpace(tagName)
	tagValue = strings.TrimSpace(tagValue)
	if err := s.aiRepo.UpdateSuggestion(suggestion).SetFinal(categoryName, tagName, tagValue).Exec(); err != nil {
		return fmt.Errorf("更新建议 %d 失败: %w", id, err)
	}
	s.notifyChange()
	return nil
}

// Approve files a batch of suggestions into the tag library: new terms are created,
// the value is associated with the file, and the review row is closed as approved.
//
// One bad id never drags the batch down. Every per-item problem — a row that is
// gone, already reviewed, a category that does not exist, a tag the library refuses
// to create, an association that failed — is collected as a Failure and the walk
// continues, because the reviewer selected these rows by hand and has to be told
// which ones did not land, row by row.
//
// The success count is safe to act on twice: approving a row that is already
// approved only reads it, and approving a row whose association already exists skips
// the association (see alreadyTagged), so a re-run of a half-finished batch repairs
// the row instead of double-counting File.TagCnt.
func (s *Service) Approve(ids []uint) (int, []Failure, error) {
	var (
		approved int
		failures []Failure
	)
	for _, id := range ids {
		suggestion, failure, err := s.pendingSuggestion(id)
		if err != nil {
			return approved, failures, err
		}
		if failure != nil {
			failures = append(failures, *failure)
			continue
		}
		if err := s.approveSuggestion(suggestion); err != nil {
			failures = append(failures, Failure{ID: id, Msg: err.Error()})
			continue
		}
		approved++
	}
	if approved > 0 {
		s.notifyChange()
	}
	return approved, failures, nil
}

// Reject closes a batch of suggestions as rejected. It touches the review row and
// nothing else: a rejected term does not enter the tag library, the file and the
// preset value dictionary alike.
func (s *Service) Reject(ids []uint) (int, []Failure, error) {
	var (
		rejected int
		failures []Failure
	)
	for _, id := range ids {
		suggestion, failure, err := s.pendingSuggestion(id)
		if err != nil {
			return rejected, failures, err
		}
		if failure != nil {
			failures = append(failures, *failure)
			continue
		}
		// approved_tag_id is written as NULL on purpose: the row records that a
		// human said no, not which tag they said no to.
		if err := s.aiRepo.UpdateSuggestion(suggestion).SetReview(ReviewRejected, nil).Exec(); err != nil {
			failures = append(failures, Failure{ID: id, Msg: fmt.Sprintf("写入审核结果失败: %v", err)})
			continue
		}
		rejected++
	}
	if rejected > 0 {
		s.notifyChange()
	}
	return rejected, failures, nil
}

// approveSuggestion is one row's way through the gate, in the order the effects
// depend on: file → term → existing association → association → review record.
//
// Every error it returns is about this one row and becomes a Failure; none of them
// leaves a half-filed suggestion behind except a failure of the last step, which
// leaves a real association on a still-pending row — and approving it again then
// skips the association (alreadyTagged) and only closes the row.
func (s *Service) approveSuggestion(suggestion *model.AISuggestion) error {
	file, err := s.fileSvc.FindByID(suggestion.FileID)
	if err != nil {
		return fmt.Errorf("读取文件 %d 失败: %w", suggestion.FileID, err)
	}
	if file == nil {
		// The row outlived its file (deleted or unindexed since recognition).
		return fmt.Errorf("文件 %d 已不在库里, 无法关联标签", suggestion.FileID)
	}

	tag, err := s.resolveOrCreateTag(suggestion.FinalCategoryName, suggestion.FinalTagName)
	if err != nil {
		return err
	}

	value := suggestion.FinalValue
	associated, err := s.alreadyTagged(file.FilePath, tag.TagID, value)
	if err != nil {
		return fmt.Errorf("检查 %s 的已有标签失败: %w", file.FilePath, err)
	}
	if !associated {
		// TagFile owns the association and the redundant File.TagCnt counter, so
		// neither is touched here. Its error is the reviewer's message.
		if err := s.fileSvc.TagFile(file.FilePath, tag.TagID, value); err != nil {
			return err
		}
	}

	if err := s.aiRepo.UpdateSuggestion(suggestion).SetReview(ReviewApproved, &tag.TagID).Exec(); err != nil {
		return fmt.Errorf("写入审核结果失败: %w", err)
	}
	return nil
}

// alreadyTagged reports whether the file's default tag group already holds this
// exact (tag, value) pair.
//
// The check exists because TagFile answers "already associated" with an error
// (internal/service/file/tag.go:44-49) and only increments File.TagCnt on its
// success path: without this pre-check, re-approving a suggestion that is still
// pending but already filed would surface as a spurious failure instead of closing
// the review row. It is deliberately not treated as the dedupe.
//
// It is scoped to the default group, which is the group TagFile writes into, rather
// than to all of the file's groups: an all-groups check would call a tag:value that
// exists only in some hand-made group "already associated", skip TagFile, and then
// close the suggestion as approved while the AI-approved tag never reached the
// default group and TagCnt never moved — a silent loss no other layer would notice.
// The cost is one extra ListGroups call per approved suggestion.
func (s *Service) alreadyTagged(filePath string, tagID uint, tagValue string) (bool, error) {
	groups, err := s.fileSvc.ListGroups(filePath)
	if err != nil {
		return false, err
	}
	var groupID uint
	for _, group := range groups {
		if group.IsDefault {
			groupID = group.GroupID
			break
		}
	}
	// groupID doubles as the "found" flag, on this explicit basis: TagGroup.GroupID
	// is an autoincrement primary key that starts at 1, so 0 is never a real group
	// id and can only mean "no default group in this list" — do not "simplify" the
	// loop into a range check against that sentinel.
	if groupID == 0 {
		// No default group yet means the file cannot have been tagged in one. It is
		// not an error: TagFile creates the group lazily (internal/service/file/tag.go:29-37).
		return false, nil
	}

	assoc, _, err := s.fileSvc.ListFileTagsByGroup(filePath, groupID, 0, 0)
	if err != nil {
		return false, err
	}
	for _, item := range assoc {
		if item.TagID == tagID && item.TagValue == tagValue {
			return true, nil
		}
	}
	return false, nil
}

// resolveOrCreateTag turns the reviewer's effective category and term into a real
// tag: reuse the tag of that name inside that category when the library has one,
// otherwise have the tag service create it.
//
// Nothing else is done with the term. The match is scoped to the category on purpose
// — borrowing a same-named tag of another category would file the association under
// a taxonomy the reviewer did not choose, while the review row would keep claiming a
// different category. Within that category the lookup matches at any hierarchy level
// (findTag scans ListTags(category, 0, 0), which returns roots and descendants), so a
// term can be filed against a nested same-named tag where creation would have made a
// root one, and because ListTags orders by id the lowest-id match wins — intended,
// deterministic, and recorded in approved_tag_id as it is filed.
//
// A tag is only ever created through tagSvc.AddTag, the one path that enforces the
// service-layer uniqueness rules (spec §7), with no description and no parent: the
// model is not asked for either (spec §5) and inventing a hierarchy from a review
// click is nobody's decision but the reviewer's. A missing category is reported as
// the error the category lookup produced and is never created automatically —
// categories are few, meaningful and hand-designed, and an approval that silently
// added one would bury that in a batch result.
//
// It writes no preset value either: nothing on this path touches the TagValue
// dictionary, which stays manually maintained (spec §5 新值入库).
func (s *Service) resolveOrCreateTag(categoryName, tagName string) (*model.Tag, error) {
	// Both guards are load-bearing, not tidiness: ListTags("") is "no category
	// filter", so an empty category would let a name match a tag of some category
	// nobody chose, and an empty name would ask the tag service for a blank tag.
	// The tests trim because a name that only trims to empty must never reach
	// findTag or AddTag either — AddTag does no validation of its own, so this is
	// the only barrier. The values themselves are used verbatim below: the writer
	// (UpdateSuggestion) normalises at write time and the pipeline's parser trims,
	// so the stored Final* string is the single source of truth, and quietly
	// creating a trimmed tag for an untrimmed record is the divergence this avoids.
	if strings.TrimSpace(categoryName) == "" {
		return nil, fmt.Errorf("建议 %s 的生效分类为空, 请先在审核时补上分类", tagName)
	}
	if strings.TrimSpace(tagName) == "" {
		return nil, fmt.Errorf("分类 %s 下的建议没有标签名, 无法入库", categoryName)
	}

	tag, err := s.findTag(categoryName, tagName)
	if err != nil {
		// 分类不存在时 ListTags 报「分类项 '%s' 不存在」, 原样上抛。
		return nil, err
	}
	if tag != nil {
		return tag, nil
	}

	tag, err = s.tagSvc.AddTag(categoryName, tagName, "", "")
	if err == nil {
		return tag, nil
	}
	// 「已存在」 — somebody else filed that term between the lookup and the insert,
	// the same new term approved from two batches at once, or a concurrent
	// reviewer — is the trigger the brief names (spec §8 第 5 条). The reuse below
	// is deliberately NOT narrowed to that message by error-string matching (a
	// comparison that would silently stop firing the day the tag service reworded
	// it): ANY AddTag error earns a category-scoped re-query first. That is safe
	// because findTag can only return a row that actually is (categoryName,
	// tagName) — so a transient 创建标签失败 whose row nevertheless landed becomes a
	// correct reuse, while every error that leaves no such row behind, including a
	// duplicate from a different category (which AddTag's root-level name check
	// also rejects) and a broken read itself, finds nothing and the ORIGINAL AddTag
	// error is restored, so the reviewer still gets the tag service's own answer.
	reused, findErr := s.findTag(categoryName, tagName)
	if findErr == nil && reused != nil {
		return reused, nil
	}
	return nil, err
}

// findTag is the category-scoped lookup resolveOrCreateTag needs twice: before
// creating a term and again when the creation was refused. It reports "no such tag"
// as (nil, nil) and only returns an error when the category itself cannot be read.
func (s *Service) findTag(categoryName, tagName string) (*model.Tag, error) {
	tags, _, err := s.tagSvc.ListTags(categoryName, 0, 0)
	if err != nil {
		return nil, err
	}
	for i := range tags {
		if tags[i].TagName == tagName {
			return &tags[i], nil
		}
	}
	return nil, nil
}

// pendingSuggestion is the gate both batch operations run through: the row has to
// exist and still be unreviewed. Exactly one of the suggestion and the Failure is
// non-nil unless the read itself broke, which is the third value and ends the batch
// — a database this service cannot read makes every remaining id equally unknown,
// while an id that simply has no row is one failure among the selected ones.
func (s *Service) pendingSuggestion(id uint) (*model.AISuggestion, *Failure, error) {
	suggestion, err := s.aiRepo.FindSuggestion(id)
	if err != nil {
		return nil, nil, fmt.Errorf("读取建议 %d 失败: %w", id, err)
	}
	if suggestion == nil {
		return nil, &Failure{ID: id, Msg: msgMissingSuggestion}, nil
	}
	if suggestion.ReviewStatus != ReviewPending {
		return nil, &Failure{ID: id, Msg: msgAlreadyReviewed}, nil
	}
	return suggestion, nil, nil
}
