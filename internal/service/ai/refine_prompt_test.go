package ai

import (
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
)

// Task 5 的提示词拆分测试。与 parse_test.go 同包 (ai) 而不是 ai_test:
// TestBuildPrompt_UnchangedAfterExtract 要用未导出的 renderPromptLibrary 与
// promptHead/promptCategories/promptTree/promptRules 常量拼期望串,
// TestVisionFreePrompt 要直接引用未导出常量 visionFreePrompt。

// seedRefineLibrary 建一份小标签库: 两个分类、一条两级路径带预设值、一个孤立根标签。
// 沿用 parse_test.go 里 prompt 测试的 seed 手法 (setupTagLibraryDB + AddCategory +
// AddTag + AddTagValue), 让提取前后的 BuildPrompt 输出与整理提示词都落在同一份数据上。
func seedRefineLibrary(t *testing.T) {
	t.Helper()
	catSvc := categorySvc.NewService()
	tagService := tagSvc.NewService()
	if _, err := catSvc.AddCategory("场景内容", ""); err != nil {
		t.Fatalf("add category 场景内容: %v", err)
	}
	if _, err := catSvc.AddCategory("主体", ""); err != nil {
		t.Fatalf("add category 主体: %v", err)
	}
	for _, s := range []struct{ category, name, parent string }{
		{"场景内容", "户外", ""},
		{"场景内容", "海滩", "户外"},
		{"主体", "人像", ""},
	} {
		if _, err := tagService.AddTag(s.category, s.name, "", s.parent); err != nil {
			t.Fatalf("add tag %s/%s (parent %q): %v", s.category, s.name, s.parent, err)
		}
	}
	if _, err := tagService.AddTagValue("海滩", "日落"); err != nil {
		t.Fatalf("add tag value: %v", err)
	}
}

// TestBuildPrompt_UnchangedAfterExtract 是库渲染抽取 (renderPromptLibrary) 的回归门:
// 期望串完全用现有常量手工拼出, 逐字节对照抽取前的真实输出 (任务报告里附了
// 抽取前后同一 seed 下提示词的逐字节 diff)。任何空白、小节顺序或段落归属的移动
// 都会在这里失败, 而不是靠肉眼比对。
func TestBuildPrompt_UnchangedAfterExtract(t *testing.T) {
	cleanup := setupTagLibraryDB(t)
	defer cleanup()
	seedRefineLibrary(t)

	prompt, err := NewService().BuildPrompt()
	if err != nil {
		t.Fatalf("BuildPrompt: %v", err)
	}
	t.Logf("prompt:\n----\n%s----", prompt)

	// 手工拼出的期望串: 与重构前捕获的 1403 字节输出逐字一致 (标签树缩进两层空格,
	// 各小节之间恰好一个空行, 结尾带换行)。
	want := promptHead + "\n\n" + promptCategories + "\n" +
		"- 场景内容\n- 主体\n" +
		"\n" + promptTree + "\n" +
		"场景内容\n  - 户外\n    - 户外 / 海滩(预设值: 日落)\n主体\n  - 人像\n" +
		"\n" + promptRules + "\n"
	if prompt != want {
		t.Errorf("抽取后 BuildPrompt 输出变了, 必须逐字节不变:\ngot:\n----\n%s----\nwant:\n----\n%s----", prompt, want)
	}

	// 抽取的接缝本身: head + "\n\n" + 库段落 + "\n" + rules + "\n" 拼回完整提示词,
	// 钉住 renderPromptLibrary 的边界 (从 promptCategories 起、到最后一条标签路径止)。
	library, err := NewService().renderPromptLibrary()
	if err != nil {
		t.Fatalf("renderPromptLibrary: %v", err)
	}
	if rebuilt := promptHead + "\n\n" + library + "\n" + promptRules + "\n"; rebuilt != prompt {
		t.Errorf("renderPromptLibrary 段落与 BuildPrompt 对不上:\n----\n%s----", rebuilt)
	}

	// 抽取不能把整理提示词的字样漏进视觉提示词里。
	for _, forbidden := range []string{"帧:", "整理"} {
		if strings.Contains(prompt, forbidden) {
			t.Errorf("BuildPrompt 混入了整理字样 %q:\n----\n%s", forbidden, prompt)
		}
	}
}

// TestBuildRefinePrompt 钉住整理提示词的语义: 标签库照常在场、逐帧原文带
// 「第 %d 帧:」标签且 FrameIndex 用原值、契约原样引用, 以及归一/合并/不凭空添加。
func TestBuildRefinePrompt(t *testing.T) {
	cleanup := setupTagLibraryDB(t)
	defer cleanup()
	seedRefineLibrary(t)

	frames := []model.AIFrameOutput{
		{FrameIndex: 2, RawText: "我看到一片沙滩, 好像有人在游泳。"},
		{FrameIndex: 5, RawText: "日落时分的海滩, 远处有一艘船。"},
	}

	prompt, err := NewService().BuildRefinePrompt(frames)
	if err != nil {
		t.Fatalf("BuildRefinePrompt: %v", err)
	}
	t.Logf("refine prompt:\n----\n%s----", prompt)

	for _, want := range []string{
		// 库段落: 分类名、标签路径、预设值。
		"场景内容", "主体", "户外 / 海滩", "预设值",
		// 逐帧原文: 帧标签用 FrameIndex 原值, 原文逐字。
		"第 2 帧:", "第 5 帧:",
		"我看到一片沙滩, 好像有人在游泳。",
		"日落时分的海滩, 远处有一艘船。",
		// 固定契约。
		wantContract,
		// 语义钉死的规则要点 (同义词归一含「沙滩」→「海滩」示例)。
		"只输出 JSON", "逐字", "同义词", "「沙滩」", "「海滩」", "合并", "留空字符串", "不得凭空添加", "不要换个说法",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("整理提示词缺少 %q:\n----\n%s", want, prompt)
		}
	}

	// 帧标签顺序即传入顺序, 且必须出现在库段落之后、整理要求之前。
	libraryStart := strings.Index(prompt, promptTree)
	frame2 := strings.Index(prompt, "第 2 帧:")
	frame5 := strings.Index(prompt, "第 5 帧:")
	rules := strings.Index(prompt, "整理要求")
	if !(libraryStart >= 0 && libraryStart < frame2 && frame2 < frame5 && frame5 < rules) {
		t.Errorf("提示词小节顺序不对 (库=%d, 第2帧=%d, 第5帧=%d, 整理要求=%d):\n----\n%s",
			libraryStart, frame2, frame5, rules, prompt)
	}
}

// TestVisionFreePrompt: 自由描述提示词不给格式, 所以契约和 JSON 字样一个字都不许出现。
func TestVisionFreePrompt(t *testing.T) {
	if !strings.Contains(visionFreePrompt, "观察图片, 描述画面内容并列出你看到的标签, 自由文本即可, 不需要任何格式") {
		t.Errorf("visionFreePrompt 缺少钉死的措辞要点:\n%s", visionFreePrompt)
	}
	if strings.Contains(visionFreePrompt, promptContract) || strings.Contains(visionFreePrompt, wantContract) {
		t.Errorf("visionFreePrompt 不该带输出契约:\n%s", visionFreePrompt)
	}
	if strings.Contains(visionFreePrompt, "JSON") {
		t.Errorf("visionFreePrompt 不该出现 JSON 字样:\n%s", visionFreePrompt)
	}
}
