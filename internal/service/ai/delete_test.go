package ai_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	repoAI "github.com/XingfenD/tag_file/internal/repository/ai"
	"github.com/XingfenD/tag_file/internal/service/ai"
)

func TestDeleteTask_RemovesTaskAndItsPendingOnly(t *testing.T) {
	defer setupTestDB(t)()
	file := seedFile(t, "a.mp4")
	failed := seedTask(t, file.FileID, ai.TaskFailed)
	seedSuggestions(t, failed.TaskID, file.FileID, ai.ReviewPending, 2)
	seedSuggestions(t, failed.TaskID, file.FileID, ai.ReviewApproved, 1)
	other := seedTask(t, file.FileID, ai.TaskDone)
	seedSuggestions(t, other.TaskID, file.FileID, ai.ReviewPending, 1)

	svc := ai.NewService()
	changes, cancel := svc.SubscribeChange()
	defer cancel()
	if err := svc.DeleteTask(failed.TaskID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	tasks := tasksOf(t, file.FileID)
	if len(tasks) != 1 || tasks[0].TaskID != other.TaskID {
		t.Fatalf("剩余任务 = %+v, want 只剩 %d", tasks, other.TaskID)
	}
	if got := countSuggestions(t, file.FileID, ai.ReviewPending); got != 1 {
		t.Errorf("待审建议 = %d, want 1 (别的任务的待审不该被牵连)", got)
	}
	if got := countSuggestions(t, file.FileID, ai.ReviewApproved); got != 1 {
		t.Errorf("已通过建议 = %d, want 1 (审核历史要保留)", got)
	}
	select {
	case <-changes:
	case <-time.After(2 * time.Second):
		t.Error("删除后没有收到 change 推送")
	}
}

func TestDeleteTask_QueuedAndStaleRunning(t *testing.T) {
	defer setupTestDB(t)()
	file := seedFile(t, "a.mp4")
	queued := seedTask(t, file.FileID, ai.TaskQueued)
	stalled := seedTask(t, file.FileID, ai.TaskRunning)

	svc := ai.NewService()
	if err := svc.DeleteTask(queued.TaskID); err != nil {
		t.Fatalf("删除排队任务: %v", err)
	}
	if err := svc.DeleteTask(stalled.TaskID); err != nil {
		t.Fatalf("删除无人认领的 running 任务: %v", err)
	}
	if got := tasksOf(t, file.FileID); len(got) != 0 {
		t.Errorf("剩余任务 = %d, want 0", len(got))
	}
}

func TestDeleteTask_Unknown(t *testing.T) {
	defer setupTestDB(t)()
	err := ai.NewService().DeleteTask(999)
	if err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("DeleteTask(999) = %v, want 不存在", err)
	}
}

// 条件删除的并发闸：读到的状态在落删前被 worker 抢走（queued→running），
// 删除必须落空而不是把在飞任务的行抽走。
func TestDeleteTask_StatusChangedRefused(t *testing.T) {
	defer setupTestDB(t)()
	file := seedFile(t, "a.mp4")
	task := seedTask(t, file.FileID, ai.TaskQueued)

	repo := repoAI.NewRepository()
	claimed, err := repo.ClaimNextQueued()
	if err != nil || claimed == nil || claimed.TaskID != task.TaskID {
		t.Fatalf("claim = %+v, %v", claimed, err)
	}
	affected, err := repo.DeleteTask(task.TaskID, ai.TaskQueued)
	if err != nil {
		t.Fatalf("条件删除: %v", err)
	}
	if affected != 0 {
		t.Fatalf("按过期状态删除影响 %d 行, want 0", affected)
	}
	if got := tasksOf(t, file.FileID); len(got) != 1 {
		t.Errorf("任务数 = %d, want 1 (条件落空不该删行)", len(got))
	}
}

func TestDeleteTask_RefusesLiveRunningTask(t *testing.T) {
	workerFixture(t)
	captureLogs(t)
	seedTagLibrary(t)

	svc := ai.NewService()
	defer svc.StopWorker()
	svc.Extractor = fakeExtractor{}
	hold := make(chan struct{})
	release := sync.OnceFunc(func() { close(hold) })
	defer release()
	client := &intervalClient{out: matchedJSON, hold: hold, holdFrom: 0}
	svc.Client = client

	picture := registerMedia(t, "a.jpg", encodeFrame(t, tintRed, bandBrightness(0b0000_1111)))
	if _, err := svc.EnqueuePaths([]string{"a.jpg"}, false); err != nil {
		t.Fatalf("EnqueuePaths: %v", err)
	}
	svc.StartWorker()
	waitFor(t, "任务进到识别里", func() bool { return len(client.recorded()) == 1 })

	task := newestTask(t, picture.FileID)
	err := svc.DeleteTask(task.TaskID)
	if err == nil || !strings.Contains(err.Error(), "正在执行中") {
		t.Fatalf("删除在飞任务 = %v, want 拒绝并说明正在执行", err)
	}
	if got := tasksOf(t, picture.FileID); len(got) != 1 {
		t.Errorf("任务数 = %d, want 1 (拒绝不该删行)", len(got))
	}

	release()
	waitFor(t, "在飞任务自己跑完", func() bool { return statusOf(t, picture.FileID) == ai.TaskDone })
	svc.StopWorker()
	if err := svc.DeleteTask(task.TaskID); err != nil {
		t.Fatalf("跑完后删除: %v", err)
	}
	if got := countSuggestions(t, picture.FileID, ai.ReviewPending); got != 0 {
		t.Errorf("删除后待审建议 = %d, want 0 (随任务一并清理)", got)
	}
}