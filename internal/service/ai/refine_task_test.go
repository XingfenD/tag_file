package ai_test

import (
	"fmt"
	"sync"
	"testing"

	repoAI "github.com/XingfenD/tag_file/internal/repository/ai"
	"github.com/XingfenD/tag_file/internal/service/ai"
)

// The guard matrix of the enqueue-time entry, pinned to the brief's exact
// sentences. The order of the guards is itself under test: the status check
// answers before the raws check (case 2's queued task has no frame outputs at
// all, yet is refused on its status), and the raws check before the endpoint
// check (cases 4/5 run with refine off and still answer 「没有识图原文」, not
// 「未开启整理端点」). Cases 6 and 7 additionally pin the two texts that must
// NOT fire early: with raws present the off endpoint finally answers (6), and
// only an endpoint that is on lets the refusal reach the in-flight row (7).
func TestRefineTask_Guards(t *testing.T) {
	workerFixture(t)
	captureLogs(t)
	seedTagLibrary(t)

	svc := ai.NewService()
	defer svc.StopWorker()

	// 1. 不存在的任务 id。
	if err := svc.RefineTask(999999); err == nil || err.Error() != "识别任务 999999 不存在" {
		t.Errorf("RefineTask(999999) = %v, want 「识别任务 999999 不存在」", err)
	}

	// 2. queued 任务 → 按状态拒绝(即使它名下也没有任何原文)。
	queuedFile := seedFile(t, "q.jpg")
	queued := seedTask(t, queuedFile.FileID, ai.TaskQueued)
	if err := svc.RefineTask(queued.TaskID); err == nil ||
		err.Error() != "仅已完成或失败的任务可重新整理,当前状态 'queued'" {
		t.Errorf("queued refusal = %v, want 「仅已完成或失败的任务可重新整理,当前状态 'queued'」", err)
	}

	// 3. running 但没有 worker 拥有它(陈旧行)→ 同样按状态拒绝。RetryTask
	// 接受这种行, RefineTask 不接受: 阶段一死在中间的 run, 其原文集合不可知。
	staleFile := seedFile(t, "s.jpg")
	stale := seedTask(t, staleFile.FileID, ai.TaskRunning)
	if err := svc.RefineTask(stale.TaskID); err == nil ||
		err.Error() != "仅已完成或失败的任务可重新整理,当前状态 'running'" {
		t.Errorf("stale running refusal = %v, want 「仅已完成或失败的任务可重新整理,当前状态 'running'」", err)
	}

	// 4. done 但一条识图原文都没存下 → 缺原文。
	dryFile := seedFile(t, "d.jpg")
	dry := seedTask(t, dryFile.FileID, ai.TaskDone)
	if err := svc.RefineTask(dry.TaskID); err == nil ||
		err.Error() != "该任务没有识图原文, 无法重新整理" {
		t.Errorf("no-raws refusal = %v, want 「该任务没有识图原文, 无法重新整理」", err)
	}

	// 5. 重整理任务的源行已被删除 → 同款缺原文(源解析走的是归一化后的 id)。
	goneFile := seedFile(t, "g.jpg")
	gone := seedRefineTask(t, goneFile.FileID, 999998)
	if err := repoAI.NewRepository().UpdateTask(gone).SetStatus(ai.TaskDone).Exec(); err != nil {
		t.Fatalf("mark refine task done: %v", err)
	}
	if err := svc.RefineTask(gone.TaskID); err == nil ||
		err.Error() != "该任务没有识图原文, 无法重新整理" {
		t.Errorf("gone-source refusal = %v, want 「该任务没有识图原文, 无法重新整理」", err)
	}

	// 6. 原文齐了, 整理端点还关着 → 端点文案。
	offFile := seedFile(t, "off.jpg")
	off := seedTask(t, offFile.FileID, ai.TaskDone)
	seedSourceFrames(t, off.TaskID, offFile.FileID, "第 1 帧的自由原文")
	if err := svc.RefineTask(off.TaskID); err == nil ||
		err.Error() != "未开启整理端点, 请先在配置页配置" {
		t.Errorf("endpoint-off refusal = %v, want 「未开启整理端点, 请先在配置页配置」", err)
	}

	enableRefine(t)

	// 7. 同文件已有更新的在途任务 → 重复提交文案, 点名在途行的 id(RetryTask
	// 同款)。守卫必须赶在 DeletePendingByFile 之前: 在途 run 的产物不能被
	// 一次被拒的点击顺手删掉。
	busyFile := seedFile(t, "w.jpg")
	older := seedTask(t, busyFile.FileID, ai.TaskDone)
	seedSourceFrames(t, older.TaskID, busyFile.FileID, "第 1 帧的自由原文")
	inFlight := seedTask(t, busyFile.FileID, ai.TaskQueued)
	want := fmt.Sprintf("该文件已有进行中的识别任务 %d,请勿重复提交", inFlight.TaskID)
	if err := svc.RefineTask(older.TaskID); err == nil || err.Error() != want {
		t.Errorf("in-flight refusal = %v, want %q", err, want)
	}
	if got := tasksOf(t, busyFile.FileID); len(got) != 2 {
		t.Errorf("w.jpg tasks = %d, want 2 (被拒的点击不得另起一行)", len(got))
	}

	// 8. running 且本进程 worker 正卡在它里面 → 「正在执行中」(同 RetryTask
	// 文案)。镜像 TestRetryTask_RefusesLiveRunningTask: 进到被 hold 的
	// Recognize 里, 就是 liveTask 已置位的可观察证据。
	svc.Extractor = fakeExtractor{}
	// 端点此时已开, 注入 RefineClient 让放行之后的阶段二不连真实端点。
	svc.RefineClient = &scriptedClient{fallback: matchedJSON}
	hold := make(chan struct{})
	release := sync.OnceFunc(func() { close(hold) })
	defer release()
	client := &intervalClient{out: matchedJSON, hold: hold, holdFrom: 0}
	svc.Client = client

	picture := registerMedia(t, "live.jpg", encodeFrame(t, tintRed, bandBrightness(0b0000_1111)))
	live := seedTask(t, picture.FileID, ai.TaskQueued)
	svc.StartWorker()
	waitFor(t, "任务进到识别里", func() bool { return len(client.recorded()) == 1 })

	if task := newestTask(t, picture.FileID); task.TaskID != live.TaskID || task.Status != ai.TaskRunning {
		t.Fatalf("在飞任务 = %+v, want %d 且 status=%q", task, live.TaskID, ai.TaskRunning)
	}
	want = fmt.Sprintf("识别任务 %d 正在执行中, 请等待它结束", live.TaskID)
	if err := svc.RefineTask(live.TaskID); err == nil || err.Error() != want {
		t.Errorf("live refusal = %v, want %q", err, want)
	}
	if got := tasksOf(t, picture.FileID); len(got) != 1 {
		t.Errorf("live.jpg tasks = %d, want 1 (拒绝在飞任务不该另起一行)", len(got))
	}

	// 拒绝没有伤到在飞的 run: 放行后它自己跑到 done。
	release()
	waitFor(t, "在飞任务自己跑完", func() bool { return statusOf(t, picture.FileID) == ai.TaskDone })
	svc.StopWorker()
	if got := countSuggestions(t, picture.FileID, ai.ReviewPending); got != 1 {
		t.Errorf("在飞任务的建议 = %d 条, want 1 (没被被拒的重整理删掉)", got)
	}
}

// The action half of the entry, mirroring TestRetryTask_OnlyFailed's shape:
// pending is cleared before the new row lands (「最新任务拥有待审集合」), the
// new row is queued and carries the normalized SourceTaskID, and the old row
// plus every reviewed suggestion stay exactly as they were.
func TestRefineTask_ReplacesPendingAndKeepsHistory(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	enableRefine(t)

	file := seedFile(t, "a.jpg")
	source := seedTask(t, file.FileID, ai.TaskDone)
	seedSourceFrames(t, source.TaskID, file.FileID, "第 1 帧的自由原文", "第 2 帧的自由原文")
	seedSuggestions(t, source.TaskID, file.FileID, ai.ReviewPending, 2)
	seedSuggestions(t, source.TaskID, file.FileID, ai.ReviewApproved, 1)

	svc := ai.NewService()
	if err := svc.RefineTask(source.TaskID); err != nil {
		t.Fatalf("RefineTask: %v", err)
	}

	tasks := tasksOf(t, file.FileID)
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want 旧 done 行 + 1 条新 queued 行", len(tasks))
	}
	if tasks[0].TaskID != source.TaskID || tasks[0].Status != ai.TaskDone || tasks[0].SourceTaskID != 0 {
		t.Errorf("旧行被改写了: %+v, want 原样保留 (status=%q, SourceTaskID=0)", tasks[0], ai.TaskDone)
	}
	if tasks[1].Status != ai.TaskQueued {
		t.Errorf("新任务 status = %q, want %q", tasks[1].Status, ai.TaskQueued)
	}
	if tasks[1].FileID != file.FileID {
		t.Errorf("新任务 FileID = %d, want %d", tasks[1].FileID, file.FileID)
	}
	if tasks[1].SourceTaskID != source.TaskID {
		t.Errorf("新任务 SourceTaskID = %d, want 源任务 %d", tasks[1].SourceTaskID, source.TaskID)
	}

	if pending := countSuggestions(t, file.FileID, ai.ReviewPending); pending != 0 {
		t.Errorf("pending suggestions = %d, want 0 so refine runs do not pile up", pending)
	}
	if approved := countSuggestions(t, file.FileID, ai.ReviewApproved); approved != 1 {
		t.Errorf("approved suggestions = %d, want the reviewed history kept", approved)
	}
	// 原文属于源任务, 新行名下不复制任何行。
	if got := len(frameOutputsOf(t, tasks[1].TaskID)); got != 0 {
		t.Errorf("新任务名下有 %d 行原文, want 0", got)
	}
}

// R3 binding: re-refining a refine task points the new row at the ORIGINAL
// vision task, so the raws chain is never indirected through an intermediate.
func TestRefineTask_NormalizesSourceChain(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	enableRefine(t)

	file := seedFile(t, "a.jpg")
	vision := seedTask(t, file.FileID, ai.TaskDone)
	seedSourceFrames(t, vision.TaskID, file.FileID, "第 1 帧的自由原文")

	first := seedRefineTask(t, file.FileID, vision.TaskID)
	if err := repoAI.NewRepository().UpdateTask(first).SetStatus(ai.TaskDone).Exec(); err != nil {
		t.Fatalf("mark first refine task done: %v", err)
	}

	svc := ai.NewService()
	if err := svc.RefineTask(first.TaskID); err != nil {
		t.Fatalf("RefineTask on a refine task: %v", err)
	}

	tasks := tasksOf(t, file.FileID)
	if len(tasks) != 3 {
		t.Fatalf("tasks = %d, want vision + first refine + new queued", len(tasks))
	}
	newest := tasks[len(tasks)-1]
	if newest.Status != ai.TaskQueued {
		t.Errorf("newest status = %q, want %q", newest.Status, ai.TaskQueued)
	}
	if newest.SourceTaskID != vision.TaskID {
		t.Errorf("newest SourceTaskID = %d, want 归一化到最初识图任务 %d (不是中间行 %d)",
			newest.SourceTaskID, vision.TaskID, first.TaskID)
	}
	// 被再次整理的那一行同样不许被改写。
	if tasks[1].TaskID != first.TaskID || tasks[1].Status != ai.TaskDone || tasks[1].SourceTaskID != vision.TaskID {
		t.Errorf("中间行被改写了: %+v, want 原样保留 (status=%q, SourceTaskID=%d)",
			tasks[1], ai.TaskDone, vision.TaskID)
	}
}

// The read-decide-write runs under enqueueMu, so a double click cannot land two
// queued rows: the loser waits on the mutex, then its in-flight guard sees the
// winner's row. Without the lock both callers would pass the guard on the same
// snapshot and both insert — the same failure TestEnqueuePaths_ConcurrentCallers
// pins for the batch path.
func TestRefineTask_ConcurrentClicks(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	enableRefine(t)

	file := seedFile(t, "a.jpg")
	done := seedTask(t, file.FileID, ai.TaskDone)
	seedSourceFrames(t, done.TaskID, file.FileID, "第 1 帧的自由原文")

	svc := ai.NewService()
	const clicks = 2
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, clicks)
	for i := 0; i < clicks; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = svc.RefineTask(done.TaskID)
		}(i)
	}
	close(start)
	wg.Wait()

	succeeded := 0
	for i, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		t.Logf("click %d refused: %v", i, err)
	}
	if succeeded != 1 {
		t.Fatalf("succeeded clicks = %d, want exactly 1 (errs: %v)", succeeded, errs)
	}

	tasks := tasksOf(t, file.FileID)
	if len(tasks) != 2 {
		t.Fatalf("task rows = %d, want the done row plus exactly one queued row", len(tasks))
	}
	queued := 0
	for _, task := range tasks {
		if task.Status == ai.TaskQueued {
			queued++
		}
	}
	if queued != 1 {
		t.Errorf("queued rows = %d, want 1", queued)
	}
	// The loser is refused by the in-flight guard, naming the winner's row.
	loser := errs[0]
	if loser == nil {
		loser = errs[1]
	}
	want := fmt.Sprintf("该文件已有进行中的识别任务 %d,请勿重复提交", tasks[1].TaskID)
	if loser == nil || loser.Error() != want {
		t.Errorf("loser error = %v, want %q", loser, want)
	}
}
