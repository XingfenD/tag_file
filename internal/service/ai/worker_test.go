package ai_test

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XingfenD/tag_file/internal/model"
	repoAI "github.com/XingfenD/tag_file/internal/repository/ai"
	"github.com/XingfenD/tag_file/internal/service/ai"
	categorySvc "github.com/XingfenD/tag_file/internal/service/category"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	tagSvc "github.com/XingfenD/tag_file/internal/service/tag"
)

// The two seams of ports.go, faked: everything below drives the real pipeline
// against the real database, and only the frame extraction and the vision model
// are substitutes, so `go test` needs neither ffmpeg nor a running Ollama.
type fakeExtractor struct {
	frames [][]byte
	err    error
}

func (f fakeExtractor) Extract(ctx context.Context, p string, iv float64, mx int) ([][]byte, error) {
	return f.frames, f.err
}

type fakeClient struct {
	out   string
	err   error
	calls int
}

// Recognize counts its calls without a mutex on purpose: calls is only ever read
// by a test after StopWorker has returned, and close(done)/<-done is what orders
// the worker's write before that read.
func (f *fakeClient) Recognize(ctx context.Context, p string, imgs [][]byte) (string, error) {
	f.calls++
	return f.out, f.err
}

// suggestionsJSON names one tag of the seeded library and one term of a category
// the library has never heard of, so a single batch walks both the matched path
// and the invented-category path.
const suggestionsJSON = `{"tags":[{"category":"主体","tag":"猫","value":"橘色"},{"category":"材质","tag":"皮革","value":""}]}`

// matchedJSON stays inside the seeded library, so the tests that use it produce no
// review-facing noise at all.
const matchedJSON = `{"tags":[{"category":"主体","tag":"猫","value":"橘色"}]}`

// workerFixture gives one test its own scan root, a fresh database and a configured
// model name.
//
// The chdir is what makes the pipeline testable at all: File.FilePath is relative to
// the working directory (which is what processTask resolves it against, because the
// WebUI chdirs into the scan root), and setupTestDB opens a relative path too, so
// moving first keeps both the media files and the sqlite file inside a directory Go
// deletes again. t.TempDir comes before the chdir and the chdir before setupTestDB
// because cleanups run LIFO: the database is removed inside the temporary directory,
// then the working directory is restored, then the directory goes away.
func workerFixture(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	t.Cleanup(setupTestDB(t))
	if err := configSvc.NewService().Set(ai.KeyOllamaModel, "qwen2.5vl:7b"); err != nil {
		t.Fatalf("seed ollama_model: %v", err)
	}
}

// seedTagLibrary creates the pre-existing library the model is shown and allowed to
// match against: one category with one tag. Anything the suggestions below name
// besides 「主体/猫」 is a term the model invented, which is a review decision and not
// this pipeline's business. Tag names stay unique across categories on purpose
// (Task 7 carry: ListTagValuesBatch keys by name, so a duplicate would make the
// prompt nondeterministic).
func seedTagLibrary(t *testing.T) *model.Tag {
	t.Helper()
	if _, err := categorySvc.NewService().AddCategory("主体", ""); err != nil {
		t.Fatalf("add category: %v", err)
	}
	tag, err := tagSvc.NewService().AddTag("主体", "猫", "", "")
	if err != nil {
		t.Fatalf("add tag: %v", err)
	}
	return tag
}

// registerMedia writes one file into the scan root and indexes it, which is the
// state both EnqueuePaths and processTask need: a row whose FilePath resolves to
// bytes on disk.
func registerMedia(t *testing.T, relPath string, content []byte) *model.File {
	t.Helper()
	if err := os.WriteFile(relPath, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
	return seedFile(t, relPath)
}

// newestTask returns the highest-id task of one file; nil when it has none.
func newestTask(t *testing.T, fileID uint) *model.AITask {
	t.Helper()
	tasks := tasksOf(t, fileID)
	if len(tasks) == 0 {
		return nil
	}
	return &tasks[len(tasks)-1]
}

// statusOf is the polling variant of newestTask: an unreadable row (a read that
// races the worker's write reports SQLITE_BUSY) is reported as "no task yet"
// instead of failing the test, because the next poll resolves it.
func statusOf(t *testing.T, fileID uint) string {
	t.Helper()
	tasks, _, err := repoAI.NewRepository().ListTasks().ByFileID(fileID).Exec()
	if err != nil || len(tasks) == 0 {
		return ""
	}
	return tasks[len(tasks)-1].Status
}

// pendingOf loads the unreviewed suggestions of one file, oldest first.
func pendingOf(t *testing.T, fileID uint) []model.AISuggestion {
	t.Helper()
	items, _, err := repoAI.NewRepository().ListSuggestions().ByFileID(fileID).ByStatus(ai.ReviewPending).Exec()
	if err != nil {
		t.Fatalf("list suggestions: %v", err)
	}
	return items
}

// waitFor polls check every 50ms for at most 2s. The worker is asynchronous, so
// every assertion about what it produced goes through here instead of a fixed sleep.
func waitFor(t *testing.T, what string, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if check() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待超时(2s): %s", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// lineBuffer collects what the standard logger wrote during one test. The worker
// goroutine writes while the test goroutine reads, hence the mutex.
type lineBuffer struct {
	mu    sync.Mutex
	lines []string
}

func (b *lineBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append(b.lines, strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

// containing returns the recorded lines that hold substr.
func (b *lineBuffer) containing(substr string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	var hits []string
	for _, line := range b.lines {
		if strings.Contains(line, substr) {
			hits = append(hits, line)
		}
	}
	return hits
}

// captureLogs redirects the standard logger for the rest of one test. The worker
// logs what it writes into a task's Error column, and that line belongs in the
// operator's log, not in `go test` output: capturing it keeps the output pristine
// and turns it into something a test can assert on.
func captureLogs(t *testing.T) *lineBuffer {
	t.Helper()
	buf := &lineBuffer{}
	previous := log.Writer()
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(previous) })
	return buf
}

// callWindow is one Recognize call: when it started, when it ended, and the prompt
// it was handed.
type callWindow struct {
	start  time.Time
	end    time.Time
	prompt string
}

// intervalClient is the fake of the sequential test: it keeps the worker inside
// Recognize for a measurable while and records the window, so a second concurrent
// worker would leave overlapping windows behind. It needs its own mutex (the plain
// fakeClient above is only read after the worker stopped), and it records the
// prompt, which is how that test sees the prompt builder was actually called.
//
// hold gates one chosen call: from the holdFrom-th invocation on, Recognize blocks
// until the channel is closed, which is how the test gets a task provably in flight
// at the moment it stops the worker.
type intervalClient struct {
	mu       sync.Mutex
	out      string
	dwell    time.Duration
	hold     chan struct{}
	holdFrom int
	calls    []callWindow
}

func (c *intervalClient) Recognize(ctx context.Context, prompt string, imgs [][]byte) (string, error) {
	c.mu.Lock()
	// The slot is remembered by index rather than as "the last one", so a call that
	// overlaps another still closes its own window: an interleaved run has to show up
	// as overlapping windows, which is the whole point of this fake.
	index := len(c.calls)
	c.calls = append(c.calls, callWindow{start: time.Now(), prompt: prompt})
	c.mu.Unlock()

	time.Sleep(c.dwell)

	if c.hold != nil && index >= c.holdFrom {
		<-c.hold
	}

	c.mu.Lock()
	c.calls[index].end = time.Now()
	c.mu.Unlock()
	return c.out, nil
}

// recorded copies the windows out of the mutex so the caller can inspect them
// while the worker is still running.
func (c *intervalClient) recorded() []callWindow {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]callWindow(nil), c.calls...)
}

func TestWorker_ProcessesQueuedTaskAndStoresPendingSuggestions(t *testing.T) {
	workerFixture(t)
	logs := captureLogs(t)
	matched := seedTagLibrary(t)

	frameRed := encodeFrame(t, tintRed, bandBrightness(0b0000_1111))
	frameBlue := encodeFrame(t, tintBlue, bandBrightness(0b1111_0000))

	svc := ai.NewService()
	defer svc.StopWorker()
	// Three frames, two of them byte-identical: the task's frame count is what the
	// dedupe filter kept, so it proves both Extract and FilterSimilarFrames ran.
	svc.Extractor = fakeExtractor{frames: [][]byte{frameRed, frameRed, frameBlue}}
	client := &fakeClient{out: suggestionsJSON}
	svc.Client = client

	picture := registerMedia(t, "a.jpg", frameRed)
	movie := registerMedia(t, "b.mp4", []byte("dummy video content"))

	result, err := svc.EnqueuePaths([]string{"a.jpg", "b.mp4"}, false)
	if err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}
	if len(result.Queued) != 2 {
		t.Fatalf("queued %d tasks, want 2 (skipped: %+v)", len(result.Queued), result.Skipped)
	}

	svc.StartWorker()
	waitFor(t, "图片与视频任务都变成 done", func() bool {
		return statusOf(t, picture.FileID) == ai.TaskDone && statusOf(t, movie.FileID) == ai.TaskDone
	})
	svc.StopWorker()

	if client.calls != 2 {
		t.Errorf("模型被调用 %d 次, want 2 (每个任务一次)", client.calls)
	}
	// 图片只有一帧(直接读文件), 视频三帧去重后两帧(走抽帧 + 过滤)。
	if got := newestTask(t, picture.FileID).FrameCount; got != 1 {
		t.Errorf("图片任务 FrameCount = %d, want 1", got)
	}
	if got := newestTask(t, movie.FileID).FrameCount; got != 2 {
		t.Errorf("视频任务 FrameCount = %d, want 2 (抽 3 帧, 近似帧被过滤掉 1 帧)", got)
	}

	for _, file := range []*model.File{picture, movie} {
		suggestions := pendingOf(t, file.FileID)
		if len(suggestions) != 2 {
			t.Fatalf("%s 有 %d 条待审建议, want 2", file.FilePath, len(suggestions))
		}
		hit, invented := suggestions[0], suggestions[1]
		if hit.ReviewStatus != ai.ReviewPending || invented.ReviewStatus != ai.ReviewPending {
			t.Errorf("%s 建议状态 = %q/%q, want 全部 %q", file.FilePath, hit.ReviewStatus, invented.ReviewStatus, ai.ReviewPending)
		}
		// 原始三列与生效三列同值入库: 审核之前 Final* 就是模型写的那套。
		for _, item := range suggestions {
			if item.FinalCategoryName != item.CategoryName || item.FinalTagName != item.TagName || item.FinalValue != item.TagValue {
				t.Errorf("建议 %s/%s 的 Final* 与原始值不一致: %s/%s(%q vs %q)",
					item.CategoryName, item.TagName, item.FinalCategoryName, item.FinalTagName, item.FinalValue, item.TagValue)
			}
			if item.FileID != file.FileID || item.TaskID != newestTask(t, file.FileID).TaskID {
				t.Errorf("建议 %s 的 TaskID/FileID = %d/%d, want %d/%d",
					item.TagName, item.TaskID, item.FileID, newestTask(t, file.FileID).TaskID, file.FileID)
			}
		}
		if hit.CategoryName != "主体" || hit.TagName != "猫" || hit.TagValue != "橘色" {
			t.Errorf("命中原库的建议被改写了: %+v", hit)
		}
		if hit.MatchedTagID == nil || *hit.MatchedTagID != matched.TagID {
			t.Errorf("「主体/猫」MatchedTagID = %v, want %d", derefAny(hit.MatchedTagID), matched.TagID)
		}
		// 模型编出来的分类原样入库, 不建分类、不建标签, 只留一个待审新词。
		if invented.CategoryName != "材质" || invented.TagName != "皮革" || invented.TagValue != "" {
			t.Errorf("陌生分类的建议被改写了: %+v", invented)
		}
		if invented.MatchedTagID != nil {
			t.Errorf("「材质/皮革」不该匹配到标签: %d", *invented.MatchedTagID)
		}
	}

	// Task 7 移交的降噪责任: 陌生分类每个任务汇总成一行, 而不是每条建议一行。
	if got := len(logs.containing("未知分类")); got != 2 {
		t.Errorf("陌生分类日志 %d 行, want 2 (每任务一行汇总)", got)
	}
	if got := logs.containing("按未命中处理"); len(got) != 0 {
		t.Errorf("出现了逐条建议的 matchTag 日志, want 0: %v", got)
	}
}

func TestWorker_RecordsFailureAndAllowsRetry(t *testing.T) {
	workerFixture(t)
	logs := captureLogs(t)
	seedTagLibrary(t)

	svc := ai.NewService()
	defer svc.StopWorker()
	// Stopping before starting is a no-op, and it must stay one that leaves the
	// sync.Once unspent: otherwise the real shutdown further down waits for a
	// goroutine that never ran, which surfaces as a test binary that times out.
	svc.StopWorker()
	client := &fakeClient{err: errors.New("模型不可达")}
	svc.Client = client
	svc.Extractor = fakeExtractor{}

	picture := registerMedia(t, "a.jpg", encodeFrame(t, tintRed, bandBrightness(0b0000_1111)))
	result, err := svc.EnqueuePaths([]string{"a.jpg"}, false)
	if err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}
	failed := result.Queued[0]

	svc.StartWorker()
	waitFor(t, "图片任务变成 failed", func() bool { return statusOf(t, picture.FileID) == ai.TaskFailed })

	task := newestTask(t, picture.FileID)
	if task.TaskID != failed {
		t.Fatalf("失败的是任务 %d, want %d", task.TaskID, failed)
	}
	if !strings.Contains(task.Error, "模型不可达") {
		t.Errorf("Error = %q, want 含「模型不可达」", task.Error)
	}
	if got := countSuggestions(t, picture.FileID, ai.ReviewPending); got != 0 {
		t.Errorf("失败任务留下 %d 条待审建议, want 0 (不写半截)", got)
	}
	if got := logs.containing("识别任务"); len(got) != 1 || !strings.Contains(got[0], "模型不可达") {
		t.Errorf("失败只该有一行日志并带上原因, got %v", got)
	}

	// 抽帧失败同样只留下失败原因: 视频分支的另一半(Review Focus #4)。
	movie := registerMedia(t, "b.mp4", []byte("dummy video content"))
	client.err = nil
	client.out = matchedJSON
	svc.Extractor = fakeExtractor{err: errors.New("抽不出帧")}
	if _, err := svc.EnqueuePaths([]string{"b.mp4"}, false); err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}
	waitFor(t, "视频任务变成 failed", func() bool { return statusOf(t, movie.FileID) == ai.TaskFailed })

	if movieTask := newestTask(t, movie.FileID); !strings.Contains(movieTask.Error, "抽不出帧") {
		t.Errorf("视频任务 Error = %q, want 含「抽不出帧」", movieTask.Error)
	}
	if got := countSuggestions(t, movie.FileID, ai.ReviewPending); got != 0 {
		t.Errorf("抽帧失败留下 %d 条待审建议, want 0", got)
	}
	if client.calls != 1 {
		t.Errorf("模型被调用 %d 次, want 1 (抽帧失败的任务不该问模型)", client.calls)
	}

	// 失败历史不改写: 重试是一条新任务。停掉 worker 才能看到它停在 queued 的那一瞬间
	// —— 这也是 StopWorker 幂等性的用武之地(下面 defer 里那次调用是空操作)。
	svc.StopWorker()
	if err := svc.RetryTask(failed); err != nil {
		t.Fatalf("RetryTask: %v", err)
	}
	tasks := tasksOf(t, picture.FileID)
	if len(tasks) != 2 {
		t.Fatalf("重试后该文件有 %d 条任务, want 2", len(tasks))
	}
	if tasks[0].TaskID != failed || tasks[0].Status != ai.TaskFailed || !strings.Contains(tasks[0].Error, "模型不可达") {
		t.Errorf("旧任务被改写了: %+v", tasks[0])
	}
	if tasks[1].TaskID == failed || tasks[1].Status != ai.TaskQueued {
		t.Errorf("重试任务是 %+v, want 新行且 status=%q", tasks[1], ai.TaskQueued)
	}
	if got := countSuggestions(t, picture.FileID, ai.ReviewPending); got != 0 {
		t.Errorf("重试前该文件有 %d 条待审建议, want 0", got)
	}
}

func TestWorker_RestartMarksRunningFailedAndResumesQueued(t *testing.T) {
	workerFixture(t)
	seedTagLibrary(t)

	svc := ai.NewService()
	defer svc.StopWorker()
	svc.Extractor = fakeExtractor{}
	client := &fakeClient{out: matchedJSON}
	svc.Client = client

	// 上一次进程被杀时留下的现场: 一条卡在 running 的、一条还没轮到的。
	interrupted := registerMedia(t, "old.jpg", encodeFrame(t, tintRed, bandBrightness(0b0000_1111)))
	resumed := registerMedia(t, "new.jpg", encodeFrame(t, tintBlue, bandBrightness(0b1111_0000)))
	stuck := seedTask(t, interrupted.FileID, ai.TaskRunning)
	queued := seedTask(t, resumed.FileID, ai.TaskQueued)

	svc.StartWorker()
	waitFor(t, "queued 任务被接着跑完", func() bool { return statusOf(t, resumed.FileID) == ai.TaskDone })

	if got := newestTask(t, resumed.FileID); got.TaskID != queued.TaskID {
		t.Errorf("处理掉的是任务 %d, want 入队那条 %d", got.TaskID, queued.TaskID)
	}
	svc.StopWorker()

	if client.calls != 1 {
		t.Errorf("模型被调用 %d 次, want 1 (被中断的任务不该重跑)", client.calls)
	}
	if task := newestTask(t, interrupted.FileID); task.TaskID != stuck.TaskID || task.Status != ai.TaskFailed || task.Error != "重启中断" {
		t.Errorf("重启后旧 running 任务是 %+v, want failed 且 Error==「重启中断」", task)
	}
	if got := pendingOf(t, resumed.FileID); len(got) != 1 || got[0].TagName != "猫" || got[0].FinalValue != "橘色" {
		t.Errorf("续跑任务的建议 = %+v, want 1 条「主体/猫=橘色」pending", got)
	}
}

func TestWorker_SequentialNotConcurrent(t *testing.T) {
	workerFixture(t)
	seedTagLibrary(t)

	const dwell = 20 * time.Millisecond

	svc := ai.NewService()
	defer svc.StopWorker()
	svc.Extractor = fakeExtractor{}
	// The fourth call (index 3) is the one the shutdown phase parks inside
	// Recognize, so the first three run at their normal length.
	hold := make(chan struct{})
	client := &intervalClient{out: matchedJSON, dwell: dwell, hold: hold, holdFrom: 3}
	svc.Client = client

	files := make([]*model.File, 0, 3)
	paths := []string{"a.jpg", "b.jpg", "c.jpg"}
	for _, path := range paths {
		files = append(files, registerMedia(t, path, encodeFrame(t, tintGreen, bandBrightness(0b0011_0011))))
	}
	result, err := svc.EnqueuePaths(paths, false)
	if err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}
	if len(result.Queued) != 3 {
		t.Fatalf("queued %d tasks, want 3 (skipped: %+v)", len(result.Queued), result.Skipped)
	}

	begun := time.Now()
	svc.StartWorker()
	waitFor(t, "3 个任务全部完成", func() bool {
		for _, file := range files {
			if statusOf(t, file.FileID) != ai.TaskDone {
				return false
			}
		}
		return true
	})
	elapsed := time.Since(begun)

	calls := client.recorded()
	if len(calls) != 3 {
		t.Fatalf("模型被调用 %d 次, want 3", len(calls))
	}
	// 调用区间互不重叠 = 同一时刻只有一个识别在跑(spec 反目标「并发打爆显存」)。
	for i := 1; i < len(calls); i++ {
		if calls[i].start.Before(calls[i-1].end) {
			t.Errorf("第 %d 次调用在第 %d 次结束前就开始了(%s → %s 撞上 %s → %s): 识别并发跑了",
				i+1, i, calls[i-1].start.Format("15:04:05.000"), calls[i-1].end.Format("15:04:05.000"),
				calls[i].start.Format("15:04:05.000"), calls[i].end.Format("15:04:05.000"))
		}
	}
	if elapsed < 3*dwell {
		t.Errorf("3 次调用只花了 %v, want 至少 %v (顺序执行)", elapsed, 3*dwell)
	}
	for i, call := range calls {
		if !strings.Contains(call.prompt, "主体") {
			t.Errorf("第 %d 次调用的提示词里没有标签库里的分类「主体」, 提示词构建没生效", i+1)
		}
	}

	// 关停语义: StopWorker 必须等到 goroutine 真的退出。第四个文件的识别被 hold 卡在
	// Recognize 里, 于是「还没退出」是可以确定性观察的 —— worker 只在 hold 上阻塞,
	// 别处都不会。
	last := registerMedia(t, "d.jpg", encodeFrame(t, tintGray, bandBrightness(0b1010_1010)))
	if _, err := svc.EnqueuePaths([]string{"d.jpg"}, false); err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}
	waitFor(t, "第四个任务进到 Recognize 里", func() bool { return len(client.recorded()) == 4 })

	stopped := make(chan struct{})
	go func() {
		svc.StopWorker()
		close(stopped)
	}()
	select {
	case <-stopped:
		t.Fatal("StopWorker 在识别还在进行的时候就返回了 (没有等 goroutine 退出)")
	case <-time.After(100 * time.Millisecond):
		// 还在跑, 正是想要的。
	}

	// 放开在飞的任务: 它得跑完, 不能被关停打断 —— 半截识别比慢一点的退出糟糕得多。
	close(hold)
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("放开在飞的任务后 StopWorker 仍没返回")
	}
	if got := newestTask(t, last.FileID); got.Status != ai.TaskDone {
		t.Errorf("关停时在飞的任务 %+v, want 跑完变成 done", got)
	}
	if got := countSuggestions(t, last.FileID, ai.ReviewPending); got != 1 {
		t.Errorf("在飞的任务留下 %d 条待审建议, want 1", got)
	}
}

func TestWorker_EmptyModelConfigFails(t *testing.T) {
	workerFixture(t)
	logs := captureLogs(t)
	seedTagLibrary(t)
	// 用户把模型名清空了(配置页可以做到的事)。
	if err := configSvc.NewService().Set(ai.KeyOllamaModel, ""); err != nil {
		t.Fatalf("clear ollama_model: %v", err)
	}

	svc := ai.NewService()
	defer svc.StopWorker()
	svc.Extractor = fakeExtractor{}
	client := &fakeClient{out: matchedJSON}
	svc.Client = client

	picture := registerMedia(t, "a.jpg", encodeFrame(t, tintRed, bandBrightness(0b0000_1111)))
	if _, err := svc.EnqueuePaths([]string{"a.jpg"}, false); err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}

	svc.StartWorker()
	waitFor(t, "缺模型名的任务变成 failed", func() bool { return statusOf(t, picture.FileID) == ai.TaskFailed })

	task := newestTask(t, picture.FileID)
	if !strings.Contains(task.Error, "未配置") {
		t.Errorf("Error = %q, want 含「未配置」", task.Error)
	}
	if got := countSuggestions(t, picture.FileID, ai.ReviewPending); got != 0 {
		t.Errorf("失败任务留下 %d 条待审建议, want 0", got)
	}
	svc.StopWorker()
	if client.calls != 0 {
		t.Errorf("模型被调用 %d 次, want 0 (配置缺失要在建客户端之前就拦下)", client.calls)
	}
	if got := logs.containing("识别任务"); len(got) != 1 || !strings.Contains(got[0], "未配置") {
		t.Errorf("缺配置该有一行带原因的日志, got %v", got)
	}
}

// derefAny prints a matched tag pointer for a readable failure message.
func derefAny(id *uint) any {
	if id == nil {
		return nil
	}
	return *id
}
