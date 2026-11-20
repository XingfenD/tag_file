package ai

import (
	"errors"
	"fmt"

	"github.com/XingfenD/tag_file/internal/model"
)

// RefineTask queues a 只重新整理 run for one finished task: a stage-two rerun
// over the raw answers its vision source already stored, costing no GPU and no
// frame extraction. It mirrors RetryTask's enqueue discipline (see there for
// why the old row is never rewritten and why pending suggestions are cleared
// first) with three entries' own guards in front of the action: only done and
// failed rows can be re-refined — a running row this process is inside is
// refused with RetryTask's live wording, and unlike a retry, a stale running
// row is refused by status too, because stage one dying mid-frame leaves an
// unknown raw set that a retry, not a re-refine, is the way out of; the
// normalized source must still hold at least one stored raw answer; and the
// refine endpoint must be on. These are the early, friendly faces of the two
// conditions recognizeRefineTask re-checks at execution time (a source deleted
// after the click, an endpoint switched off before the claim) — same facts,
// different layers, the enqueue texts being the ones the UI quotes.
func (s *Service) RefineTask(taskID uint) error {
	// Same critical section as enqueue and RetryTask: this method reads the
	// task, its source and the file's newest in-flight task and then inserts,
	// so two double-clicks that both snapshotted before either inserted would
	// each conclude "nothing in flight" and both queue the file.
	s.enqueueMu.Lock()
	defer s.enqueueMu.Unlock()

	task, err := s.aiRepo.FindTask(taskID)
	if err != nil {
		return fmt.Errorf("查询识别任务失败: %w", err)
	}
	if task == nil {
		return fmt.Errorf("识别任务 %d 不存在", taskID)
	}
	// The live check answers before the status set, the same split RetryTask
	// makes: a row the worker is inside right now is not a stale one, and the
	// user gets "wait for it" instead of a status verdict that would send them
	// at the task that is running fine.
	if task.Status == TaskRunning && s.liveTask() == task.TaskID {
		return fmt.Errorf("识别任务 %d 正在执行中, 请等待它结束", task.TaskID)
	}
	if task.Status != TaskDone && task.Status != TaskFailed {
		return fmt.Errorf("仅已完成或失败的任务可重新整理,当前状态 '%s'", task.Status)
	}

	// 归一化: a refine task points at the ORIGINAL vision task, never at the
	// refine row in between, so the raws chain stays one hop deep however often
	// the button is clicked. When this task is itself the vision task its row is
	// the source — already loaded, no second read.
	sourceID := task.TaskID
	source := task
	if task.SourceTaskID != 0 {
		sourceID = task.SourceTaskID
		source, err = s.aiRepo.FindTask(sourceID)
		if err != nil {
			return fmt.Errorf("查询源任务 %d 失败: %w", sourceID, err)
		}
	}
	hasRaws := false
	if source != nil {
		outputs, err := s.aiRepo.ListFrameOutputs(sourceID)
		if err != nil {
			return fmt.Errorf("读取任务 %d 的识图原文失败: %w", sourceID, err)
		}
		hasRaws = len(outputs) > 0
	}
	if !hasRaws {
		// The source row is gone (deleted after the first click) or never
		// stored an answer: 阶段二从库读, 库里没有就没有可整理的内容。
		return errors.New("该任务没有识图原文, 无法重新整理")
	}
	if !refineEnabled(s.Settings()) {
		return errors.New("未开启整理端点, 请先在配置页配置")
	}

	// The old row is never rewritten, so without this guard a second click
	// would queue the same file again while the first re-refine is still
	// waiting for the worker — word for word RetryTask's refusal.
	inFlight, err := s.inFlightTask(task.FileID, task.TaskID)
	if err != nil {
		return err
	}
	if inFlight != nil {
		return fmt.Errorf("该文件已有进行中的识别任务 %d,请勿重复提交", inFlight.TaskID)
	}
	// 「最新任务拥有待审集合」: the fresh run would otherwise pile a second
	// copy of the proposals on top of the ones nobody has reviewed yet. This
	// must stay behind every refusal above — a rejected click deletes nothing.
	if _, err := s.aiRepo.DeletePendingByFile(task.FileID); err != nil {
		return fmt.Errorf("清理待审建议失败: %w", err)
	}
	refine := &model.AITask{FileID: task.FileID, Status: TaskQueued, SourceTaskID: sourceID}
	if err := s.aiRepo.CreateTask(refine); err != nil {
		return fmt.Errorf("创建识别任务失败: %w", err)
	}
	s.notifyWake()
	s.notifyChange()
	return nil
}
