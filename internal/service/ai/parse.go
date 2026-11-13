package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// RawTag is one suggestion exactly as the model wrote it: the three strings are
// trimmed, nothing else. They are not checked against the tag library, not
// resolved to ids and not stored — that is matchTag below for lookups and Task 9
// for writes, behind a human decision.
type RawTag struct {
	Category string `json:"category"`
	Tag      string `json:"tag"`
	Value    string `json:"value"`
}

// parseOutputLimit is how much of the model's text an error keeps, mirroring
// ollamaBodySnippet on the transport side: enough to see what went wrong without
// pasting a runaway answer into the task log.
const parseOutputLimit = 500

// ParseSuggestions turns whatever the model answered into suggestions.
//
// A vision model wraps its JSON in prose and code fences as often as it answers
// with a bare object, so two tolerances are applied, in this order: standalone markdown
// fence lines are dropped, then the first '{' to the last '}' is taken as the object.
// That is the whole repair budget — there is no retry, no bracket balancing and no
// schema library. Anything still left is a failure the task records, because a run
// that quietly yields zero suggestions looks identical to a run that found nothing.
func ParseSuggestions(output string) ([]RawTag, error) {
	text := stripFences(output)
	start, end := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("模型输出里没有 JSON 对象: %s", outputSnippet(output))
	}

	var payload struct {
		Tags []RawTag `json:"tags"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &payload); err != nil {
		return nil, fmt.Errorf("解析模型输出失败: %v, 原文: %s", err, outputSnippet(output))
	}

	items := make([]RawTag, 0, len(payload.Tags))
	for _, item := range payload.Tags {
		tagName := strings.TrimSpace(item.Tag)
		if tagName == "" {
			// 没有标签名的条目入不了库, 也不该占审核队列的位置。
			continue
		}
		item.Tag = tagName
		item.Category = strings.TrimSpace(item.Category)
		item.Value = strings.TrimSpace(item.Value)
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("模型没有给出任何有效标签建议: %s", outputSnippet(output))
	}
	return items, nil
}

// stripFences drops the lines that open or close a markdown code block: a line whose
// trimmed text starts with ``` and carries no brace at all, with or without a language
// tag. A line that has braces on it is kept even when it opens with a fence — small
// models write the whole answer inline (```json {"tags":[…]} ```), and dropping such a
// line would throw the JSON away instead of the fence, which the following brace slice
// could then never recover. Bare fence lines are still dropped so they cannot leak into
// the error text of an answer that is prose after all.
func stripFences(text string) string {
	lines := strings.Split(text, "\n")
	kept := lines[:0]
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") && !strings.ContainsAny(trimmed, "{}") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// outputSnippet is the model's text as it goes into an error message, cut on a rune
// boundary so Chinese output cannot end in half a character.
func outputSnippet(output string) string {
	text := strings.TrimSpace(output)
	runes := []rune(text)
	if len(runes) <= parseOutputLimit {
		return text
	}
	return string(runes[:parseOutputLimit]) + "…"
}

// matchTag looks one suggestion up in the library: the id of the tag with this name
// inside this category, or (nil, nil) when there is no such tag. The pointer is what
// lets Task 9 tell "no match" apart from a tag whose id would be a zero value, and
// matching is exact because the prompt demands the library's spelling — the library
// is a few hundred tags at most, so it is listed in memory (spec §5).
func (s *Service) matchTag(categoryName, tagName string) (*uint, error) {
	// ListTags("") means "no category filter", which would let a suggestion without a
	// category match a tag of the same name anywhere. No category is no match.
	if categoryName == "" || tagName == "" {
		return nil, nil
	}

	tags, _, err := s.tagSvc.ListTags(categoryName, 0, 0)
	if err != nil {
		// 分类不存在时 ListTags 报「分类项 '%s' 不存在」。模型编出分类名是审核期由
		// 人来纠正的事(spec §5/§6: 新词与错分类都进待审), 不该让整条识别失败,
		// 所以按未命中处理; 分类确实存在时剩下的只会是数据库故障, 那种错误必须上抛,
		// 否则一批建议会全部静默变成「新标签」。
		exists, cErr := s.categoryExists(categoryName)
		if cErr != nil {
			return nil, cErr
		}
		if !exists {
			log.Printf("[ai] matchTag: 分类 %q 不存在, 标签 %q 按未命中处理: %v", categoryName, tagName, err)
			return nil, nil
		}
		return nil, fmt.Errorf("匹配标签 %q 失败: %w", tagName, err)
	}

	for i := range tags {
		if tags[i].TagName == tagName {
			return &tags[i].TagID, nil
		}
	}
	return nil, nil
}

// categoryExists answers the one question matchTag needs on its error path. The
// category service exposes no find-by-name, so the small full list is scanned.
func (s *Service) categoryExists(categoryName string) (bool, error) {
	categories, _, err := s.categorySvc.ListCategories(0, 0)
	if err != nil {
		return false, err
	}
	for _, category := range categories {
		if category.CategoryName == categoryName {
			return true, nil
		}
	}
	return false, nil
}
