package ai

import "fmt"

// DeleteTask removes one task and its still-pending suggestions; reviewed
// proposals stay as history. A queued row is deletable (that is a cancel), and
// so is a running row no live recognition owns — the same stalled-row escape
// hatch RetryTask offers. The repository delete is conditional on the status
// read here, so a worker claim landing in between refuses the delete instead of
// pulling the row out from under the run.
func (s *Service) DeleteTask(taskID uint) error {
	// Same critical section as enqueue/RetryTask: read-then-write on the task
	// table, so a concurrent enqueue of the same file sees a consistent latest.
	s.enqueueMu.Lock()
	defer s.enqueueMu.Unlock()

	task, err := s.aiRepo.FindTask(taskID)
	if err != nil {
		return fmt.Errorf("查询识别任务失败: %w", err)
	}
	if task == nil {
		return fmt.Errorf("识别任务 %d 不存在", taskID)
	}
	if task.Status == TaskRunning && s.liveTask() == task.TaskID {
		return fmt.Errorf("识别任务 %d 正在执行中, 请等待它结束", task.TaskID)
	}
	affected, err := s.aiRepo.DeleteTask(task.TaskID, task.Status)
	if err != nil {
		return fmt.Errorf("删除识别任务失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("任务状态已变化, 请刷新后重试")
	}
	s.notifyChange()
	return nil
}