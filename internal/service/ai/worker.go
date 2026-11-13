package ai

import (
	"log"
)

// restartInterruptReason is the Error text of every task the previous process left
// "running". Its goroutine died with the process, so the row claims work nobody is
// doing; the sweep below turns the lie into a retryable failure. The string is the
// reason the review UI shows, hence fixed rather than per-task detail.
const restartInterruptReason = "重启中断"

// StartWorker runs the restart sweep and launches the single recognition goroutine.
//
// One Service runs one worker: the WebUI calls this once after the database is open
// and StopWorker on shutdown (Task 11), and every test builds a fresh Service. It is
// the caller's job to keep it that way — a second call would leave two goroutines
// claiming tasks from the same queue, which is the one thing this design rules out.
// The sweep happens before the goroutine starts so a claimed task can never meet a
// row that a dead process still owns.
func (s *Service) StartWorker() {
	if _, err := s.aiRepo.FailRunning(restartInterruptReason); err != nil {
		// A failed sweep must not cost this process its queue: the stale rows stay
		// "running" (visible, still retryable by hand) and consumption continues.
		log.Printf("[ai] 启动时清扫中断任务失败: %v", err)
	}

	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	go s.runWorker(s.stop, s.done)
}

// StopWorker signals the worker and waits until its goroutine has returned, so a
// caller (a test above all) knows nothing touches the database after it returns.
//
// The sync.Once keeps a repeated call from closing an already closed channel, and
// the wait lives outside it so every caller blocks for the exit, not just the one
// that got to close. Called before StartWorker there is nothing to stop: it returns
// without spending the Once, so the pair started later still stops cleanly — see
// StartWorker for why exactly one start per Service is the deal.
//
// A task in flight is not cancelled: it runs to its own end (the transport timeouts
// of Task 5 and Task 6 bound it) and its suggestions are kept, because the state
// this pipeline refuses to leave behind is exactly the half-written one.
func (s *Service) StopWorker() {
	if s.stop == nil {
		return
	}
	s.stopOnce.Do(func() { close(s.stop) })
	<-s.done
}

// runWorker is the whole concurrency story of this feature: one goroutine, one task
// at a time. The vision model on the user's GPU is the scarcest resource here and
// sharing it is one of the spec's stated anti-goals ("并发打爆显存"), so there is no
// pool, no priority queue and no per-task goroutine; the order is simply whatever
// ClaimNextQueued hands out, which is task_id order.
//
// stop and done arrive as parameters because they belong to this run, while wake is
// read off the Service: it exists from construction, since an enqueue has to be able
// to poke it whether or not a worker is running.
func (s *Service) runWorker(stop, done chan struct{}) {
	defer close(done)

	for {
		task, err := s.aiRepo.ClaimNextQueued()
		if err != nil {
			log.Printf("[ai] 认领下一个识别任务失败: %v", err)
		}
		if task != nil {
			if err := s.processTask(task); err != nil {
				// The reason is already in the task's Error column for the UI; this
				// line is the operator's copy of the same fact.
				log.Printf("[ai] 识别任务 %d 未完成: %v", task.TaskID, err)
			}
			// Straight back to the claim: a batch of fifty files waits on fifty
			// recognitions, not on fifty wake-ups.
			continue
		}

		// Queue empty (or the claim failed — one logged line costs a sleep, not a
		// spin loop). Wait until an enqueue pokes us or the process shuts down.
		select {
		case <-stop:
			return
		case <-s.wake:
		}
	}
}

// notifyWake tells the worker that the queue grew. The send never blocks and its
// result never matters: a full buffer already means "there is work you have not
// looked at", so no enqueue is ever delayed by (or deadlocked on) the recognition of
// an earlier file — and while no worker runs the token simply sits there, unread and
// harmless, until one starts.
func (s *Service) notifyWake() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
