package ai_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	repoAI "github.com/XingfenD/tag_file/internal/repository/ai"
	"github.com/XingfenD/tag_file/internal/service/ai"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
)

// scriptedClient is the refine-stage twin of perFrameClient with one thing the
// worker_test.go fakes do not record: the prompt of every call. The two-stage
// tests need to see what the vision model and the refine model were each handed
// (free prompt vs library + frame texts), and chunking tests need per-call
// answers AND per-call prompts at once, so this fake lives here rather than
// changing the shared fakes' behavior.
type scriptedClient struct {
	mu          sync.Mutex
	outs        map[int]string // 1-based 调用序号 → 答案
	errs        map[int]error  // 1-based 调用序号 → 错误, 优先于 outs
	fallback    string
	fallbackErr error // 未逐调用指定时: 每次调用的默认错误
	calls       int
	prompts     []string
	imgCounts   []int
}

func (c *scriptedClient) Recognize(ctx context.Context, prompt string, imgs [][]byte) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	c.prompts = append(c.prompts, prompt)
	c.imgCounts = append(c.imgCounts, len(imgs))
	n := c.calls
	if err, ok := c.errs[n]; ok {
		return "", err
	}
	if out, ok := c.outs[n]; ok {
		return out, nil
	}
	return c.fallback, c.fallbackErr
}

func (c *scriptedClient) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func (c *scriptedClient) recordedPrompts() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.prompts...)
}

func (c *scriptedClient) recordedImageCounts() []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]int(nil), c.imgCounts...)
}

// enableRefine switches the second stage on through the config table itself,
// exactly the way the WebUI config page does: an endpoint with a model name.
// The URL is left at its default because every test injects RefineClient.
func enableRefine(t *testing.T) {
	t.Helper()
	config := configSvc.NewService()
	if err := config.Set(ai.KeyRefineProvider, ai.ProviderOpenAI); err != nil {
		t.Fatalf("set %s: %v", ai.KeyRefineProvider, err)
	}
	if err := config.Set(ai.KeyRefineOpenAIModel, "gpt-4o"); err != nil {
		t.Fatalf("set %s: %v", ai.KeyRefineOpenAIModel, err)
	}
}

// frameCodes returns n band codes that survive the dedupe filter pairwise: two
// codes of equal parity differ in at least two of the eight bands (their XOR
// has even popcount), and two bands apart is 16 of the 64 hash bits, above the
// default threshold of 10 — the mechanism pinned by TestFilterSimilarFrames.
func frameCodes(t *testing.T, n int) []uint8 {
	t.Helper()
	codes := make([]uint8, 0, n)
	for code := 0; code < 256 && len(codes) < n; code++ {
		if strings.Count(fmt.Sprintf("%08b", code), "1")%2 == 0 {
			codes = append(codes, uint8(code))
		}
	}
	if len(codes) < n {
		t.Fatalf("只有 %d 个互异帧码, want %d", len(codes), n)
	}
	return codes
}

// nFrameVideo registers a video whose fake extraction is n mutually distinct
// frames, and raises ai_max_frames past its default of 10 so the filter really
// keeps all of them.
func nFrameVideo(t *testing.T, svc *ai.Service, n int) *model.File {
	t.Helper()
	if err := configSvc.NewService().Set(ai.KeyMaxFrames, "200"); err != nil {
		t.Fatalf("set %s: %v", ai.KeyMaxFrames, err)
	}
	frames := make([][]byte, 0, n)
	for _, code := range frameCodes(t, n) {
		frames = append(frames, encodeFrame(t, tintGray, bandBrightness(code)))
	}
	svc.Extractor = fakeExtractor{dir: t.TempDir(), frames: frames}
	return registerMedia(t, "v.mp4", []byte("dummy video content"))
}

// seedSourceFrames writes the raw answers a finished vision task left behind,
// the state a re-refine task reads.
func seedSourceFrames(t *testing.T, taskID, fileID uint, raws ...string) {
	t.Helper()
	repo := repoAI.NewRepository()
	for i, raw := range raws {
		output := &model.AIFrameOutput{TaskID: taskID, FileID: fileID, FrameIndex: i + 1, RawText: raw}
		if err := repo.CreateFrameOutput(output); err != nil {
			t.Fatalf("seed frame output: %v", err)
		}
	}
}

// seedRefineTask queues a re-refine task over sourceTaskID's stored raws.
func seedRefineTask(t *testing.T, fileID, sourceTaskID uint) *model.AITask {
	t.Helper()
	task := &model.AITask{FileID: fileID, Status: ai.TaskQueued, SourceTaskID: sourceTaskID}
	if err := repoAI.NewRepository().CreateTask(task); err != nil {
		t.Fatalf("seed refine task: %v", err)
	}
	return task
}

func frameOutputsOf(t *testing.T, taskID uint) []model.AIFrameOutput {
	t.Helper()
	outputs, err := repoAI.NewRepository().ListFrameOutputs(taskID)
	if err != nil {
		t.Fatalf("list frame outputs: %v", err)
	}
	return outputs
}

// TestRecognize_OffModeRegression is the byte-level contract of the split: with
// the refine endpoint off, a run where frame 1 answers valid JSON and frame 2
// answers prose must still land on done with frame 1's suggestion alone, the
// partial-failure note word-for-word as the single-stage pipeline wrote it, and
// BOTH raw answers in the frame table — the malformed one included, because
// storing it verbatim is the point of the split.
func TestRecognize_OffModeRegression(t *testing.T) {
	workerFixture(t)
	captureLogs(t)
	seedTagLibrary(t)

	const malformed = "这不是 JSON"

	svc := ai.NewService()
	defer svc.StopWorker()
	svc.Extractor = fakeExtractor{dir: t.TempDir(), frames: [][]byte{
		encodeFrame(t, tintRed, bandBrightness(0b0000_1111)),
		encodeFrame(t, tintBlue, bandBrightness(0b1111_0000)),
	}}
	svc.Client = &perFrameClient{outs: map[int]string{1: matchedJSON, 2: malformed}}

	movie := registerMedia(t, "v.mp4", []byte("dummy video content"))
	if _, err := svc.EnqueuePaths([]string{"v.mp4"}, false); err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}
	svc.StartWorker()
	waitFor(t, "单段任务变成 done", func() bool { return statusOf(t, movie.FileID) == ai.TaskDone })
	svc.StopWorker()

	task := newestTask(t, movie.FileID)
	// 首条原因用 ParseSuggestions 对畸形串的实际错误断言: 注记要逐字等于今天的形状。
	_, parseErr := ai.ParseSuggestions(malformed)
	if parseErr == nil {
		t.Fatal("畸形串竟然解析成功了, 这条回归断言失去了对象")
	}
	if want := fmt.Sprintf("第 2 帧识别失败(1/2), 首条原因: %v", parseErr); task.Error != want {
		t.Errorf("Error = %q, want 逐字等于 %q", task.Error, want)
	}
	if got := pendingOf(t, movie.FileID); len(got) != 1 || got[0].TagName != "猫" {
		t.Errorf("待审建议 = %+v, want 1 条「主体/猫」(只有帧 1 的答案)", got)
	}
	if got := task.FrameCount; got != 2 {
		t.Errorf("FrameCount = %d, want 2", got)
	}
	if task.RefineStatus != "" {
		t.Errorf("RefineStatus = %q, want 单段任务恒 ''", task.RefineStatus)
	}

	outputs := frameOutputsOf(t, task.TaskID)
	if len(outputs) != 2 {
		t.Fatalf("原文 %d 行, want 2 (畸形帧也要落库)", len(outputs))
	}
	if outputs[0].FrameIndex != 1 || outputs[0].RawText != matchedJSON {
		t.Errorf("第 1 行 = %+v, want FrameIndex=1 且 RawText 逐字为帧 1 答案", outputs[0])
	}
	if outputs[1].FrameIndex != 2 || outputs[1].RawText != malformed {
		t.Errorf("第 2 行 = %+v, want FrameIndex=2 且 RawText 逐字为畸形答案", outputs[1])
	}
}

// TestRecognize_OnModeFreePromptAndRefine pins the two-stage shape: the vision
// model gets the free-text prompt (no output contract anywhere in it), the
// refine model gets one text-only call carrying the library and both frames'
// stored raws, and the suggestions come from the refine answer alone.
func TestRecognize_OnModeFreePromptAndRefine(t *testing.T) {
	workerFixture(t)
	captureLogs(t)
	seedTagLibrary(t)
	enableRefine(t)

	const freeText = "画面里有一只橘色的猫躺在沙滩上, 背景是海浪。"

	svc := ai.NewService()
	defer svc.StopWorker()
	svc.Extractor = fakeExtractor{dir: t.TempDir(), frames: [][]byte{
		encodeFrame(t, tintRed, bandBrightness(0b0000_1111)),
		encodeFrame(t, tintBlue, bandBrightness(0b1111_0000)),
	}}
	vision := &scriptedClient{fallback: freeText}
	refine := &scriptedClient{fallback: suggestionsJSON}
	svc.Client = vision
	svc.RefineClient = refine

	movie := registerMedia(t, "v.mp4", []byte("dummy video content"))
	if _, err := svc.EnqueuePaths([]string{"v.mp4"}, false); err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}
	svc.StartWorker()
	waitFor(t, "两段任务变成 done", func() bool { return statusOf(t, movie.FileID) == ai.TaskDone })
	svc.StopWorker()

	if got := vision.callCount(); got != 2 {
		t.Errorf("识图被调用 %d 次, want 2 (逐帧各一次)", got)
	}
	if got := vision.recordedImageCounts(); !sameCounts(got, []int{1, 1}) {
		t.Errorf("识图各次调用的图片数 = %v, want 每帧各 1 张", got)
	}
	// promptContract 是包内常量, 这里钉它的字面量: 自由描述提示词不许夹带输出契约。
	for i, prompt := range vision.recordedPrompts() {
		if strings.Contains(prompt, `{"tags":[{"category"`) {
			t.Errorf("识图第 %d 次调用的提示词带着 JSON 契约, want 自由文本提示词", i+1)
		}
		if strings.Contains(prompt, "现有分类") {
			t.Errorf("识图第 %d 次调用的提示词带着标签库, want 自由文本提示词", i+1)
		}
	}

	if got := refine.callCount(); got != 1 {
		t.Fatalf("整理被调用 %d 次, want 1 (两帧一批)", got)
	}
	if got := refine.recordedImageCounts(); !sameCounts(got, []int{0}) {
		t.Errorf("整理调用的图片数 = %v, want 纯文本调用 (0 张)", got)
	}
	prompt := refine.recordedPrompts()[0]
	for _, want := range []string{
		`{"tags":[{"category"`, // 契约随整理端走
		"主体",                   // 标签库分类
		"猫",                    // 标签库标签
		"第 1 帧:", "第 2 帧:",     // 逐帧原文标注
		freeText, // 原文逐字附上
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("整理提示词缺少 %q", want)
		}
	}

	// 建议来自整理输出(主体/猫 + 陌生的材质/皮革), 而不是识图自由文本。
	suggestions := pendingOf(t, movie.FileID)
	if len(suggestions) != 2 {
		t.Fatalf("待审建议 %d 条, want 2 (整理输出两条)", len(suggestions))
	}
	if suggestions[0].CategoryName != "主体" || suggestions[0].TagName != "猫" || suggestions[1].CategoryName != "材质" {
		t.Errorf("建议来自识图原文而不是整理输出: %+v", suggestions)
	}

	task := newestTask(t, movie.FileID)
	if task.RefineStatus != ai.RefineDone {
		t.Errorf("RefineStatus = %q, want %q", task.RefineStatus, ai.RefineDone)
	}
	if task.Error != "" {
		t.Errorf("Error = %q, want 两段全成功无注记", task.Error)
	}
	outputs := frameOutputsOf(t, task.TaskID)
	if len(outputs) != 2 {
		t.Fatalf("原文 %d 行, want 2", len(outputs))
	}
	for _, output := range outputs {
		if output.RawText != freeText {
			t.Errorf("帧 %d 原文 = %q, want 识图自由文本逐字入库", output.FrameIndex, output.RawText)
		}
	}
}

// sameCounts compares a recorded per-call image-count sequence positionally:
// the worker is serial, so the fake's record order is exact.
func sameCounts(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// TestRecognize_OnModeChunking pins the batching: 25 stored frames become two
// refine calls (20 + 5), each prompt carries exactly its own batch's frame
// numbers, and the cross-batch merge dedupes the shared tag into one suggestion.
func TestRecognize_OnModeChunking(t *testing.T) {
	workerFixture(t)
	captureLogs(t)
	seedTagLibrary(t)
	enableRefine(t)

	svc := ai.NewService()
	defer svc.StopWorker()
	movie := nFrameVideo(t, svc, 25)
	vision := &scriptedClient{fallback: "一批互异帧的自由观察记录。"}
	refine := &scriptedClient{outs: map[int]string{
		1: tagsPayload("A", 2),
		2: `{"tags":[{"category":"主体","tag":"A1","value":"值1"},{"category":"主体","tag":"A2","value":"值2"}]}`,
	}}
	svc.Client = vision
	svc.RefineClient = refine

	if _, err := svc.EnqueuePaths([]string{"v.mp4"}, false); err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}
	svc.StartWorker()
	waitFor(t, "25 帧任务变成 done", func() bool { return statusOf(t, movie.FileID) == ai.TaskDone })
	svc.StopWorker()

	if got := vision.callCount(); got != 25 {
		t.Errorf("识图被调用 %d 次, want 25 (每帧一次)", got)
	}
	if got := refine.callCount(); got != 2 {
		t.Fatalf("整理被调用 %d 次, want 2 (20 + 5 两批)", got)
	}
	prompts := refine.recordedPrompts()
	for _, index := range []int{1, 20} {
		if !strings.Contains(prompts[0], fmt.Sprintf("第 %d 帧:", index)) {
			t.Errorf("第 1 批提示词缺少帧 %d", index)
		}
	}
	if strings.Contains(prompts[0], "第 21 帧:") {
		t.Error("第 1 批提示词带着第 2 批的帧号")
	}
	for _, index := range []int{21, 25} {
		if !strings.Contains(prompts[1], fmt.Sprintf("第 %d 帧:", index)) {
			t.Errorf("第 2 批提示词缺少帧 %d", index)
		}
	}
	if strings.Contains(prompts[1], "第 20 帧:") {
		t.Error("第 2 批提示词带着第 1 批的帧号")
	}

	// A0+A1(批 1) 与 A1+A2(批 2) 跨批合并去重 → 恰好 3 条。
	suggestions := pendingOf(t, movie.FileID)
	if len(suggestions) != 3 {
		t.Fatalf("待审建议 %d 条, want 3 (A0/A1/A2, 共享的 A1 去重)", len(suggestions))
	}
	names := make([]string, 0, len(suggestions))
	for _, item := range suggestions {
		names = append(names, item.TagName)
	}
	want := []string{"A0", "A1", "A2"}
	sortStrings(names)
	if !equalNames(names, want) {
		t.Errorf("建议标签 = %v, want %v", names, want)
	}

	task := newestTask(t, movie.FileID)
	if task.FrameCount != 25 {
		t.Errorf("FrameCount = %d, want 25", task.FrameCount)
	}
	if task.RefineStatus != ai.RefineDone {
		t.Errorf("RefineStatus = %q, want %q", task.RefineStatus, ai.RefineDone)
	}
}

// sortStrings is the tiny helper the multiset-free ordering assertion needs.
func sortStrings(names []string) {
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
}

// TestRecognize_OnModeChunkPartialFailure pins the batch-level translation of
// partial success: the second batch's refine call errors, the first batch's
// suggestions still land, the task lands on done, the Error names the lost
// batch's frame span (21、30) with the batch-level ratio, and RefineStatus
// still says done — the lost raws stay in the table for a cheap re-refine.
func TestRecognize_OnModeChunkPartialFailure(t *testing.T) {
	workerFixture(t)
	captureLogs(t)
	seedTagLibrary(t)
	enableRefine(t)

	svc := ai.NewService()
	defer svc.StopWorker()
	movie := nFrameVideo(t, svc, 30)
	refine := &scriptedClient{
		outs: map[int]string{1: tagsPayload("A", 2)},
		errs: map[int]error{2: errors.New("整理端点超时")},
	}
	svc.Client = &scriptedClient{fallback: "自由观察记录。"}
	svc.RefineClient = refine

	if _, err := svc.EnqueuePaths([]string{"v.mp4"}, false); err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}
	svc.StartWorker()
	waitFor(t, "坏一批的任务变成 done", func() bool { return statusOf(t, movie.FileID) == ai.TaskDone })
	svc.StopWorker()

	suggestions := pendingOf(t, movie.FileID)
	if len(suggestions) != 2 {
		t.Fatalf("待审建议 %d 条, want 2 (第 1 批照常入库)", len(suggestions))
	}
	task := newestTask(t, movie.FileID)
	if !strings.Contains(task.Error, "第 21、30 帧整理失败(1/2), 首条原因: 整理端点超时") {
		t.Errorf("Error = %q, want 写明第 2 批(帧 21、30)整理失败(1/2)及原因", task.Error)
	}
	if task.RefineStatus != ai.RefineDone {
		t.Errorf("RefineStatus = %q, want %q (部分成功仍是完成)", task.RefineStatus, ai.RefineDone)
	}
	if got := len(frameOutputsOf(t, task.TaskID)); got != 30 {
		t.Errorf("原文 %d 行, want 30 (丢的只是批, 不是原文)", got)
	}
}

// TestRecognize_OnModeAllChunksFail pins the other half: every batch failing
// is a failed task — no suggestions, the「可重新整理」recovery pointer in the
// Error, the raws intact, and RefineStatus=failed to mark the cheap retry.
func TestRecognize_OnModeAllChunksFail(t *testing.T) {
	workerFixture(t)
	captureLogs(t)
	seedTagLibrary(t)
	enableRefine(t)

	svc := ai.NewService()
	defer svc.StopWorker()
	movie := nFrameVideo(t, svc, 25)
	svc.Client = &scriptedClient{fallback: "自由观察记录。"}
	svc.RefineClient = &scriptedClient{fallbackErr: errors.New("整理端点不可达")}

	if _, err := svc.EnqueuePaths([]string{"v.mp4"}, false); err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}
	svc.StartWorker()
	waitFor(t, "全批皆败的任务变成 failed", func() bool { return statusOf(t, movie.FileID) == ai.TaskFailed })
	svc.StopWorker()

	task := newestTask(t, movie.FileID)
	if !strings.Contains(task.Error, "整理失败, 可重新整理") {
		t.Errorf("Error = %q, want 含「整理失败, 可重新整理」", task.Error)
	}
	if got := countSuggestions(t, movie.FileID, ai.ReviewPending); got != 0 {
		t.Errorf("全败任务留下 %d 条待审建议, want 0 (不写半套)", got)
	}
	if task.RefineStatus != ai.RefineFailed {
		t.Errorf("RefineStatus = %q, want %q", task.RefineStatus, ai.RefineFailed)
	}
	outputs := frameOutputsOf(t, task.TaskID)
	if len(outputs) != 25 {
		t.Fatalf("原文 %d 行, want 25 行完好 (整理失败不毁原文)", len(outputs))
	}
}

// TestRecognize_RefineTaskSkipsVision pins the second entry: a queued task
// with SourceTaskID runs stage two over the source's stored raws without ever
// touching the vision endpoint, keeps the source's FrameCount, and writes its
// own suggestions.
func TestRecognize_RefineTaskSkipsVision(t *testing.T) {
	workerFixture(t)
	captureLogs(t)
	seedTagLibrary(t)
	enableRefine(t)

	file := seedFile(t, "a.jpg")
	source := seedTask(t, file.FileID, ai.TaskDone)
	if err := repoAI.NewRepository().UpdateTask(source).SetFrameCount(7).Exec(); err != nil {
		t.Fatalf("seed source frame count: %v", err)
	}
	seedSourceFrames(t, source.TaskID, file.FileID, "第 1 帧的自由原文", "第 2 帧的自由原文")

	svc := ai.NewService()
	defer svc.StopWorker()
	vision := &scriptedClient{fallback: matchedJSON}
	svc.Client = vision
	svc.RefineClient = &scriptedClient{fallback: suggestionsJSON}

	refine := seedRefineTask(t, file.FileID, source.TaskID)
	svc.StartWorker()
	waitFor(t, "重整理任务变成 done", func() bool { return statusOf(t, file.FileID) == ai.TaskDone })
	svc.StopWorker()

	if got := vision.callCount(); got != 0 {
		t.Errorf("识图端点被调用 %d 次, want 0 (重整理不重跑 GPU)", got)
	}

	task := newestTask(t, file.FileID)
	if task.TaskID != refine.TaskID {
		t.Fatalf("完成的是任务 %d, want 重整理任务 %d", task.TaskID, refine.TaskID)
	}
	if got := pendingOf(t, file.FileID); len(got) != 2 {
		t.Errorf("待审建议 %d 条, want 2 (整理输出两条, 属于重整理任务自己)", len(got))
		for _, item := range got {
			if item.TaskID != refine.TaskID {
				t.Errorf("建议 %s/%s 的 TaskID = %d, want %d", item.CategoryName, item.TagName, item.TaskID, refine.TaskID)
			}
		}
	}
	if task.FrameCount != 7 {
		t.Errorf("FrameCount = %d, want 沿用源任务的 7", task.FrameCount)
	}
	if task.RefineStatus != ai.RefineDone {
		t.Errorf("RefineStatus = %q, want %q", task.RefineStatus, ai.RefineDone)
	}
	if task.Error != "" {
		t.Errorf("Error = %q, want 全成功无注记", task.Error)
	}
	// 原文属于源任务, 重整理任务不复制行。
	if got := len(frameOutputsOf(t, refine.TaskID)); got != 0 {
		t.Errorf("重整理任务自己名下有 %d 行原文, want 0", got)
	}
}

// TestRecognize_RefineTaskSourceGone: a re-refine task whose source row is
// gone (deleted, or never existed) fails naming the missing raws — the state
// spec 5.2 promises when the source was deleted out from under the queue.
func TestRecognize_RefineTaskSourceGone(t *testing.T) {
	workerFixture(t)
	captureLogs(t)
	seedTagLibrary(t)
	enableRefine(t)

	file := seedFile(t, "a.jpg")

	svc := ai.NewService()
	defer svc.StopWorker()
	svc.Client = &scriptedClient{fallback: matchedJSON}
	svc.RefineClient = &scriptedClient{fallback: suggestionsJSON}

	task := seedRefineTask(t, file.FileID, 999999)
	svc.StartWorker()
	waitFor(t, "源不存在的重整理任务变成 failed", func() bool { return statusOf(t, file.FileID) == ai.TaskFailed })
	svc.StopWorker()

	got := newestTask(t, file.FileID)
	if got.TaskID != task.TaskID {
		t.Fatalf("失败的是任务 %d, want %d", got.TaskID, task.TaskID)
	}
	if !strings.Contains(got.Error, "源任务没有识图原文") {
		t.Errorf("Error = %q, want 含「源任务没有识图原文」", got.Error)
	}
	if count := countSuggestions(t, file.FileID, ai.ReviewPending); count != 0 {
		t.Errorf("失败任务留下 %d 条待审建议, want 0", count)
	}
}

// TestRecognize_RefineTaskEndpointOffAtExecution pins the per-task client
// precedent for the refine endpoint: switched to off after the task was
// enqueued, the run must fail saying so rather than quietly falling back to
// the single-stage parse.
func TestRecognize_RefineTaskEndpointOffAtExecution(t *testing.T) {
	workerFixture(t)
	captureLogs(t)
	seedTagLibrary(t)
	enableRefine(t)

	file := seedFile(t, "a.jpg")
	source := seedTask(t, file.FileID, ai.TaskDone)
	seedSourceFrames(t, source.TaskID, file.FileID, "第 1 帧的自由原文")

	svc := ai.NewService()
	defer svc.StopWorker()
	svc.Client = &scriptedClient{fallback: matchedJSON}
	svc.RefineClient = &scriptedClient{fallback: suggestionsJSON}

	task := seedRefineTask(t, file.FileID, source.TaskID)

	// 建任务之后、执行之前把整理端点切回 off。
	if err := configSvc.NewService().Set(ai.KeyRefineProvider, ai.RefineOff); err != nil {
		t.Fatalf("set %s off: %v", ai.KeyRefineProvider, err)
	}

	svc.StartWorker()
	waitFor(t, "整理端点被关后的重整理任务变成 failed", func() bool { return statusOf(t, file.FileID) == ai.TaskFailed })
	svc.StopWorker()

	got := newestTask(t, file.FileID)
	if got.TaskID != task.TaskID {
		t.Fatalf("失败的是任务 %d, want %d", got.TaskID, task.TaskID)
	}
	if !strings.Contains(got.Error, "整理端点未开启") {
		t.Errorf("Error = %q, want 含「整理端点未开启」", got.Error)
	}
	if count := countSuggestions(t, file.FileID, ai.ReviewPending); count != 0 {
		t.Errorf("失败任务留下 %d 条待审建议, want 0", count)
	}
	if got := len(frameOutputsOf(t, source.TaskID)); got != 1 {
		t.Errorf("源任务原文 %d 行, want 1 行完好", got)
	}
}
