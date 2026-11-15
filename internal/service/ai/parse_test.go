package ai

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/XingfenD/tag_file/internal/model"
	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
	"github.com/XingfenD/tag_file/pkg/storage"
)

// These tests sit in the in-package test package (not ai_test like the six other
// files here) for one reason: TestMatchTag has to call the unexported method
// s.matchTag, and Go forbids reaching an unexported method from another package
// even through an interface the test declares itself
// ("*Service does not implement matcher (unexported method matchTag)"). Task 8
// calls matchTag from inside this package, so its name stays unexported.
//
// The cost is a fourth copy of the per-package DB bootstrap. That duplication is
// the established shape of this repo (ai_test, tag_test, category_test, file_test
// each carry their own); the shared-helper refactor is explicitly ruled out.

// setupTagLibraryDB opens a fresh DB for the prompt and the matcher. The AutoMigrate
// list is kept byte-identical to ai_test's setupTestDB on purpose: three tables would
// be enough for what this file touches, but a shorter copy drifts silently the day the
// tag or category service starts reading another one.
func setupTagLibraryDB(t *testing.T) func() {
	t.Helper()
	dbPath := ".tag_file_test.db"
	os.Remove(dbPath)
	if err := storage.InitDB(dbPath); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := storage.GetDB().AutoMigrate(
		&model.File{}, &model.TagGroup{}, &model.TagGroupTag{}, &model.TagCategory{}, &model.Tag{}, &model.TagValue{},
		&model.Config{}, &model.AITask{}, &model.AISuggestion{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return func() { os.Remove(dbPath) }
}

// wantContract is the fixed output contract the prompt has to hand the model.
const wantContract = `{"tags":[{"category":"<现有分类名>","tag":"<标签名>","value":"<可选,自由文本>"}]}`

func TestBuildPrompt_ContainsCategoriesTreeAndValues(t *testing.T) {
	cleanup := setupTagLibraryDB(t)
	defer cleanup()

	catSvc := categorySvc.NewService()
	tagService := tagSvc.NewService()
	seed := func(category, name, parent string) {
		t.Helper()
		if _, err := tagService.AddTag(category, name, "", parent); err != nil {
			t.Fatalf("add tag %s/%s (parent %q): %v", category, name, parent, err)
		}
	}
	if _, err := catSvc.AddCategory("场景内容", ""); err != nil {
		t.Fatalf("add category: %v", err)
	}
	if _, err := catSvc.AddCategory("主体", ""); err != nil {
		t.Fatalf("add category: %v", err)
	}
	seed("场景内容", "户外", "")
	seed("场景内容", "海滩", "户外")
	seed("主体", "人像", "")
	if _, err := tagService.AddTagValue("海滩", "日落"); err != nil {
		t.Fatalf("add tag value: %v", err)
	}

	prompt, err := NewService().BuildPrompt()
	if err != nil {
		t.Fatalf("BuildPrompt: %v", err)
	}
	if prompt == "" {
		t.Fatal("BuildPrompt returned an empty prompt")
	}
	// 措辞是要给人看的, 断言只钉得住关键字, 所以 -v 下把整段打出来。
	t.Logf("prompt:\n----\n%s----", prompt)

	// 分类清单、Parent / Child 路径、路径上那条预设值、固定契约, 少一段模型就会开始
	// 编同义词而不是复用已有名字。预设值要连着它所属的那条路径断言, 否则全 dump 在
	// 一起也能过。契约里的 "tags" 字样随整条契约一起被钉住。
	for _, want := range []string{"场景内容", "主体", "户外 / 海滩(预设值: 日落)", wantContract, `"tags"`} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt 缺少 %q:\n----\n%s", want, prompt)
		}
	}
	// 标签树必须按分类分组: 每条路径要落在自己分类名的区间里, 否则模型无从挑 category。
	// 只在标签树这一节里比位置: 分类清单一节和输出要求里都还会再出现分类名。
	treeStart := strings.Index(prompt, promptTree)
	if treeStart < 0 {
		t.Fatalf("提示词缺少标签树一节:\n----\n%s", prompt)
	}
	rulesOffset := strings.Index(prompt[treeStart:], "\n\n输出要求")
	if rulesOffset < 0 {
		t.Fatalf("提示词缺少输出要求一节:\n----\n%s", prompt)
	}
	tree := prompt[treeStart : treeStart+rulesOffset]
	sceneAt := strings.Index(tree, "场景内容")
	beachAt := strings.Index(tree, "户外 / 海滩")
	subjectAt := strings.Index(tree, "主体")
	portraitAt := strings.Index(tree, "人像")
	if !(0 <= sceneAt && sceneAt < beachAt && beachAt < subjectAt && subjectAt < portraitAt) {
		t.Errorf("标签树的分组顺序不对 (场景内容=%d, 户外 / 海滩=%d, 主体=%d, 人像=%d):\n----\n%s",
			sceneAt, beachAt, subjectAt, portraitAt, tree)
	}
	// spec §「不要求模型给父标签、置信度、描述」: 这些字样只能出现在禁止项里。
	for _, want := range []string{"只输出 JSON", "置信度", "父标签"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt 缺少 %q:\n----\n%s", want, prompt)
		}
	}
	for _, forbidden := range []string{`"description"`, `"confidence"`, `"parent"`} {
		if strings.Contains(prompt, forbidden) {
			t.Errorf("prompt 把 %q 当成输出字段要求了, 契约里只能有 category/tag/value:\n----\n%s", forbidden, prompt)
		}
	}
}

func TestBuildPrompt_TruncatesTreeBeyondDepthLimit(t *testing.T) {
	cleanup := setupTagLibraryDB(t)
	defer cleanup()

	catSvc := categorySvc.NewService()
	tagService := tagSvc.NewService()
	if _, err := catSvc.AddCategory("深度", ""); err != nil {
		t.Fatalf("add category: %v", err)
	}
	parent := ""
	for i := 1; i <= promptTagDepthLimit+5; i++ {
		name := fmt.Sprintf("层%d", i)
		if _, err := tagService.AddTag("深度", name, "", parent); err != nil {
			t.Fatalf("add tag %s under %q: %v", name, parent, err)
		}
		parent = name
	}

	prompt, err := NewService().BuildPrompt()
	if err != nil {
		t.Fatalf("BuildPrompt: %v", err)
	}
	// 「层 k」在第 k-1 层, 所以 层21 是最后一行可见路径, 更深一点只剩截断标记。
	// 这条上限是照 cmd/webui/api.buildTagPath 定的, 不是渲染巧合。
	lastShown := fmt.Sprintf("层%d", promptTagDepthLimit+1)
	firstDropped := fmt.Sprintf("层%d", promptTagDepthLimit+2)
	if !strings.Contains(prompt, lastShown) {
		t.Errorf("prompt 缺少 %q, 标签树应渲染到第 %d 层:\n----\n%s", lastShown, promptTagDepthLimit, prompt)
	}
	if strings.Contains(prompt, firstDropped) {
		t.Errorf("prompt 渲染了 %q, 超出深度上限的层级必须截断:\n----\n%s", firstDropped, prompt)
	}
	if !strings.Contains(prompt, "已截断") {
		t.Errorf("prompt 缺少截断标记:\n----\n%s", prompt)
	}
}

// 链条刚好停在深度上限上的那一层(叶子标签在第 promptTagDepthLimit+1 层)时, 什么也没被
// 截断, 所以「已截断」这行一个字都不该出现 —— 否则读提示词的人以为标签库比实际更大。
func TestBuildPrompt_ExactlyAtDepthLimitHasNoTruncationMarker(t *testing.T) {
	cleanup := setupTagLibraryDB(t)
	defer cleanup()

	catSvc := categorySvc.NewService()
	tagService := tagSvc.NewService()
	if _, err := catSvc.AddCategory("边界", ""); err != nil {
		t.Fatalf("add category: %v", err)
	}
	parent := ""
	for i := 1; i <= promptTagDepthLimit+1; i++ {
		name := fmt.Sprintf("层%d", i)
		if _, err := tagService.AddTag("边界", name, "", parent); err != nil {
			t.Fatalf("add tag %s under %q: %v", name, parent, err)
		}
		parent = name
	}

	prompt, err := NewService().BuildPrompt()
	if err != nil {
		t.Fatalf("BuildPrompt: %v", err)
	}
	deepest := fmt.Sprintf("层%d", promptTagDepthLimit+1)
	if !strings.Contains(prompt, deepest) {
		t.Errorf("prompt 缺少最深一层的 %q, 上限那一层仍要渲染:\n----\n%s", deepest, prompt)
	}
	if strings.Contains(prompt, "已截断") {
		t.Errorf("标签链正好停在深度上限内, 却出现了截断标记:\n----\n%s", prompt)
	}
}

// 空标签库(首次使用)也要给出可用的提示词: 两段清单写成「暂无」, 契约照常出现。
func TestBuildPrompt_EmptyLibraryStillCarriesContract(t *testing.T) {
	cleanup := setupTagLibraryDB(t)
	defer cleanup()

	prompt, err := NewService().BuildPrompt()
	if err != nil {
		t.Fatalf("BuildPrompt on an empty library: %v", err)
	}
	for _, want := range []string{"暂无分类", "暂无标签", wantContract} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt 缺少 %q:\n----\n%s", want, prompt)
		}
	}
}

func TestParseSuggestions_PlainFencedAndProse(t *testing.T) {
	body := `{"tags":[{"category":"场景内容","tag":"海滩","value":"日落"}]}`
	want := []RawTag{{Category: "场景内容", Tag: "海滩", Value: "日落"}}

	tests := []struct{ name, input string }{
		{"裸 JSON", body},
		{"json 围栏", "```json\n" + body + "\n```"},
		{"裸围栏", "```\n" + body + "\n```"},
		// 小模型常把整份答案连同围栏写在同一行: 围栏剥除不能把花括号里的 JSON 一起丢掉。
		{"围栏与 JSON 同行", "```json " + body + " ```"},
		{"前后带散文", "我看这张图想到的是:\n" + body + "\n以上, 希望有帮助。"},
		{"值带空白", `{"tags":[{"category":"  场景内容  ","tag":"  海滩  ","value":"  日落  "}]}`},
	}
	for _, tt := range tests {
		got, err := ParseSuggestions(tt.input)
		if err != nil {
			t.Errorf("%s: ParseSuggestions(%q) = nil, %v; want %+v", tt.name, tt.input, err, want[0])
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: ParseSuggestions = %+v, want %+v", tt.name, got, want)
		}
	}
}

// spec 只要 category/tag/value 三个字段: 描述与置信度一律丢弃。丢弃的方式是「看不见」
// 而不是「报错」—— 模型自己加戏是常态, 加字段不该让整次识别失败, 所以现在也没有开
// DisallowUnknownFields。这几条负向断言把该行为钉住: 将来谁改成严格解码, 这里先红。
func TestParseSuggestions_IgnoresUnknownFieldsAndMissingValue(t *testing.T) {
	want := []RawTag{{Category: "场景内容", Tag: "海滩", Value: "日落"}}
	noValue := []RawTag{{Category: "场景内容", Tag: "海滩"}}

	tests := []struct {
		name  string
		input string
		want  []RawTag
	}{
		{
			name:  "条目里多出描述与置信度",
			input: `{"tags":[{"category":"场景内容","tag":"海滩","value":"日落","description":"黄昏的海滩照片","confidence":0.9}]}`,
			want:  want,
		},
		{
			name:  "顶层多出说明字段",
			input: `{"tags":[{"category":"场景内容","tag":"海滩","value":"日落"}],"reasoning":"光线偏暖","count":1}`,
			want:  want,
		},
		{
			name:  "value 为 null",
			input: `{"tags":[{"category":"场景内容","tag":"海滩","value":null}]}`,
			want:  noValue,
		},
		{
			name:  "value 字段缺失",
			input: `{"tags":[{"category":"场景内容","tag":"海滩"}]}`,
			want:  noValue,
		},
	}
	for _, tt := range tests {
		got, err := ParseSuggestions(tt.input)
		if err != nil {
			t.Errorf("%s: ParseSuggestions(%q) = nil, %v; want %+v (多出来的字段必须被忽略, 不是致命错误)", tt.name, tt.input, err, tt.want[0])
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: ParseSuggestions = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestParseSuggestions_RejectsGarbageAndEmpty(t *testing.T) {
	long := strings.Repeat("细", 600) + "TAILMARKER"

	tests := []struct {
		name     string
		input    string
		contains string
		absent   string
	}{
		// 纯散文里没有 JSON 对象, 必须失败而不是当成「本次没找到标签」。
		{name: "散文无 JSON", input: "我觉得这是海滩", contains: "我觉得这是海滩"},
		{name: "围栏里仍是散文", input: "```json\n抱歉, 我无法识别这张图\n```", contains: "抱歉, 我无法识别这张图"},
		{name: "截断的 JSON", input: `{"tags":[{"category":"场景内容","tag":"海`, contains: `"tags"`},
		{name: "tags 为空数组", input: `{"tags":[]}`},
		{name: "唯一条目的 tag 全空白", input: `{"tags":[{"category":"x","tag":"  "}]}`},
		{name: "tags 不是数组", input: `{"tags":"海滩"}`},
		// 原文片段只带到前 500 个字符, 长输出不能整段粘进任务日志。
		{name: "超长散文截断", input: long, contains: strings.Repeat("细", 500), absent: "TAILMARKER"},
	}
	for _, tt := range tests {
		got, err := ParseSuggestions(tt.input)
		if err == nil {
			t.Errorf("%s: ParseSuggestions(%q) = %+v, want an error", tt.name, tt.input, got)
			continue
		}
		if len(got) != 0 {
			t.Errorf("%s: ParseSuggestions returned %d suggestions next to an error, want 0", tt.name, len(got))
		}
		if tt.contains != "" && !strings.Contains(err.Error(), tt.contains) {
			t.Errorf("%s: error %q 缺少原文片段 %q", tt.name, err, tt.contains)
		}
		if tt.absent != "" && strings.Contains(err.Error(), tt.absent) {
			t.Errorf("%s: error %q 带了不该出现的 %q", tt.name, err, tt.absent)
		}
	}
}

// The model's strings are the only input to this pipeline that nobody sizes: the
// columns declare size:100/200 (internal/model/ai.go) and SQLite ignores that, so the
// parser is where a runaway answer meets a limit. Counted in runes — a cut made on
// bytes would end a Chinese name in half a character, which the review page then
// renders as mojibake.
func TestParseSuggestions_ClampsFieldsToColumnBudget(t *testing.T) {
	// 130 runes of a 3-byte character each: the rune budget and the byte budget
	// cannot be confused by this input.
	oversizedName := strings.Repeat("猫", 130)
	oversizedValue := strings.Repeat("值", 260)
	wantName := strings.Repeat("猫", nameBudget)
	wantValue := strings.Repeat("值", valueBudget)

	input := fmt.Sprintf(`{"tags":[{"category":%q,"tag":%q,"value":%q}]}`,
		oversizedName, oversizedName, oversizedValue)
	got, err := ParseSuggestions(input)
	if err != nil {
		t.Fatalf("ParseSuggestions: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("解析出 %d 条建议, want 1", len(got))
	}
	item := got[0]
	for _, field := range []struct {
		label, text, want string
	}{
		{"category", item.Category, wantName},
		{"tag", item.Tag, wantName},
		{"value", item.Value, wantValue},
	} {
		if field.text != field.want {
			t.Errorf("%s 未被截到列宽: 前 %d 个 rune 不符 (长度 %d rune / %d byte)",
				field.label, len([]rune(field.want)), len([]rune(field.text)), len(field.text))
		}
		if n := len([]rune(field.text)); n != len([]rune(field.want)) {
			t.Errorf("%s 截出来是 %d 个 rune, want %d", field.label, n, len([]rune(field.want)))
		}
	}
	// A byte-budgeted cut would have landed mid-character; this is the same claim
	// stated as the property that actually matters.
	if !utf8.ValidString(item.Tag) || !utf8.ValidString(item.Value) {
		t.Errorf("截断切在了字符中间: %q / %q", item.Tag, item.Value)
	}

	// Trimming still happens first, and a blank name is still dropped rather than
	// clamped into an empty stored row.
	blank, err := ParseSuggestions(`{"tags":[{"category":"x","tag":"   ","value":"y"},{"category":"x","tag":` + fmt.Sprintf("%q", oversizedName) + `}]}`)
	if err != nil {
		t.Fatalf("ParseSuggestions with a blank name: %v", err)
	}
	if len(blank) != 1 || blank[0].Tag != wantName {
		t.Errorf("解析结果 = %+v, want 只剩 1 条且标签名被截到列宽", blank)
	}
}

func TestMatchTag_HitsOnlyWithinCategory(t *testing.T) {
	cleanup := setupTagLibraryDB(t)
	defer cleanup()

	catSvc := categorySvc.NewService()
	tagService := tagSvc.NewService()
	for _, name := range []string{"动物", "植物"} {
		if _, err := catSvc.AddCategory(name, ""); err != nil {
			t.Fatalf("add category %s: %v", name, err)
		}
	}
	cat, err := tagService.AddTag("动物", "猫", "", "")
	if err != nil {
		t.Fatalf("add tag 动物/猫: %v", err)
	}
	if _, err := tagService.AddTag("植物", "树", "", ""); err != nil {
		t.Fatalf("add tag 植物/树: %v", err)
	}

	svc := NewService()

	// 分类 A 有「猫」→ 命中, 返回的 id 就是 A 里那个标签的 id。
	id, err := svc.matchTag("动物", "猫")
	if err != nil {
		t.Errorf(`matchTag("动物", "猫") error = %v, want nil`, err)
	}
	if id == nil {
		t.Fatal(`matchTag("动物", "猫") = nil, want the tag id pointer`)
	}
	if *id != cat.TagID {
		t.Errorf("matchTag 返回 id %d, want %d", *id, cat.TagID)
	}

	// 分类 B 没有「猫」→ (nil, nil); 同分类里的「树」仍能命中, 证明查询确实落在 B 上。
	if id, err = svc.matchTag("植物", "猫"); id != nil || err != nil {
		t.Errorf(`matchTag("植物", "猫") = (%v, %v), want (nil, nil)`, derefID(id), err)
	}
	if id, err = svc.matchTag("植物", "树"); id == nil || err != nil {
		t.Errorf(`matchTag("植物", "树") = (%v, %v), want a hit`, derefID(id), err)
	}

	// 模型编出来的分类不是失败, 是审核期的人工纠正项 (Review Focus #3)。
	if id, err = svc.matchTag("不存在", "猫"); id != nil || err != nil {
		t.Errorf(`matchTag("不存在", "猫") = (%v, %v), want (nil, nil)`, derefID(id), err)
	}

	// ListTags("") 的语义是「所有分类」, 空分类名不能借它跨分类命中。
	if id, err = svc.matchTag("", "猫"); id != nil || err != nil {
		t.Errorf(`matchTag("", "猫") = (%v, %v), want (nil, nil)`, derefID(id), err)
	}
}

func derefID(id *uint) any {
	if id == nil {
		return nil
	}
	return *id
}
