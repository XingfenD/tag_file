package ai

import (
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/XingfenD/tag_file/internal/model"
)

// restartInterruptReason is the Error text of every task the previous process left
// "running". Its goroutine died with the process, so the row claims work nobody is
// doing; the sweep below turns the lie into a retryable failure. The string is the
// reason the review UI shows, hence fixed rather than per-task detail.
const restartInterruptReason = "重启中断"

// claimRetryDelay is how long the worker waits before claiming again after a
// claim came back with an error. The rows are already in the database, and an
// erroring claim does not remove them or schedule another wake-up, so a purely
// signal-driven wait would leave an already-queued task sitting there until some
// future, unrelated enqueue poked it. The timer re-arms the claim itself. It is
// deliberately slow (a persistently failing claim costs at most one logged line
// per delay, not one per loop) and deliberately only ever armed on an actual
// error, so an empty queue still waits on stop/wake alone.
const claimRetryDelay = 5 * time.Second

// workerRun is the lifecycle state of one start→stop cycle. stop is closed by
// StopWorker and done by the worker goroutine when it returns, and the sync.Once
// belongs to the run so a repeated StopWorker within one cycle cannot close stop
// twice while a start→stop→start sequence stops each cycle's own channel with a
// fresh Once. A Service points at exactly one run: StartWorker creates it, and
// StopWorker reads the pointer under Service.workerMu and then waits on done
// outside the lock.
type workerRun struct {
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

// StartWorker runs the restart sweep and launches the single recognition goroutine.
//
// One Service runs one worker at a time: the WebUI calls this once after the
// database is open and StopWorker on shutdown (Task 11), and every test builds a
// fresh Service. A start→stop→start sequence on one Service is supported — each
// start creates its own run state — but starting a second time without stopping
// the first still just replaces the run pointer and leaks the older goroutine;
// keeping one live run per Service remains the caller's job. The swap itself is
// mutex-guarded, so a StartWorker concurrent with a (documented, caller-ordered)
// StopWorker cannot tear the pointer access; what it does not do is serialize
// two overlapping starts.
//
// The sweep happens before the goroutine starts so a claimed task can never meet
// a row that a dead process still owns.
func (s *Service) StartWorker() {
	if _, err := s.aiRepo.FailRunning(restartInterruptReason); err != nil {
		// A failed sweep must not cost this process its queue: the stale rows stay
		// "running" (visible, still retryable by hand) and consumption continues.
		log.Printf("[ai] 启动时清扫中断任务失败: %v", err)
	}

	run := &workerRun{stop: make(chan struct{}), done: make(chan struct{})}
	s.workerMu.Lock()
	s.run = run
	s.workerMu.Unlock()
	go s.runWorker(run)
}

// StopWorker signals the worker and waits until its goroutine has returned, so a
// caller (a test above all) knows nothing touches the database after it returns.
//
// The wait targets the run that is live at call time, so the contract is:
// idempotent per cycle (the run's Once keeps a repeated call from closing an
// already closed channel, and every caller still blocks for the exit), harmless
// before any worker ever started (there is nothing to stop and nothing is spent),
// and correct again after a start→stop→start pair (the new run has its own Once).
// Two overlapping starts are not supported — see StartWorker.
//
// Shutdown stops at the task boundary: no new task is claimed once stop is
// closed, and rows still queued at that moment stay queued for the next
// StartWorker (the spec's 重启语义). A task already in flight is not cancelled:
// it runs to its own end (the transport timeouts of Task 5 and Task 6 bound it)
// and its suggestions are kept, because the state this pipeline refuses to leave
// behind is exactly the half-written one.
func (s *Service) StopWorker() {
	s.workerMu.Lock()
	run := s.run
	s.workerMu.Unlock()
	if run == nil {
		return
	}
	run.stopOnce.Do(func() { close(run.stop) })
	<-run.done
}

// runWorker is the whole concurrency story of this feature: one goroutine, one
// task at a time. The vision model on the user's GPU is the scarcest resource
// here and sharing it is one of the spec's stated anti-goals ("并发打爆显存"), so
// there is no pool, no priority queue and no per-task goroutine; the order is
// simply whatever ClaimNextQueued hands out, which is task_id order.
//
// The run arrives as a parameter because stop and done belong to this cycle,
// while wake is read off the Service: it exists from construction, since an
// enqueue has to be able to poke it whether or not a worker is running.
func (s *Service) runWorker(run *workerRun) {
	defer close(run.done)

	for {
		// Check the stop signal before claiming, so closing stop at shutdown
		// stops the worker at the task boundary instead of draining the rest of
		// the queue first: whatever is still "queued" here is picked up by the
		// next StartWorker, and only a task already inside processTask runs on.
		select {
		case <-run.stop:
			return
		default:
		}

		var retry <-chan time.Time
		task, err := s.aiRepo.ClaimNextQueued()
		if err != nil {
			log.Printf("[ai] 认领下一个识别任务失败: %v", err)
			// The rows did not go anywhere; try again on the timer instead of
			// waiting for a wake-up that may never come.
			retry = time.After(claimRetryDelay)
		}
		if task != nil {
			s.runTask(task)
			// Straight back to the claim: a batch of fifty files waits on fifty
			// recognitions, not on fifty wake-ups.
			continue
		}

		// Queue empty (or the claim failed, which the armed timer turns into a
		// bounded retry rather than a stall). Wait until an enqueue pokes us, the
		// timer fires, or the process shuts down. A nil retry never fires, so the
		// plain idle case is still just the two channels, and neither branch
		// spins: every wake-up is caused by new work, a real shutdown, or the
		// once-per-claimRetryDelay error retry.
		select {
		case <-run.stop:
			return
		case <-s.wake:
		case <-retry:
		}
	}
}

// runTask recognizes one claimed task and owns the three things that must hold
// around it whatever happens inside: the in-process marker by which RetryTask tells
// a live "running" row from a stale one (cleared by defer, so neither an error nor a
// panic can leave it behind), the operator's log line for a task that did not end
// "done", and a last-resort net so that nothing this loop does can take the process
// down.
func (s *Service) runTask(task *model.AITask) {
	s.setLiveTask(task.TaskID)
	defer s.setLiveTask(0)

	// processTask already converts a panic into that task's failure, so anything
	// reaching here escaped the failure write itself — repeating it would only panic
	// again. Keep the worker alive, say so loudly, and leave the row as it is: the
	// marker above is gone by the time the next claim runs, so RetryTask accepts that
	// "running" row again and the user has a button for it.
	defer func() {
		if p := recover(); p != nil {
			log.Printf("[ai] 任务 %d 的失败处理自身 panic, 状态未能写入: %v\n%s", task.TaskID, p, debug.Stack())
		}
	}()

	if err := s.processTask(task); err != nil {
		// The reason is already in the task's Error column for the UI; this
		// line is the operator's copy of the same fact.
		log.Printf("[ai] 识别任务 %d 未完成: %v", task.TaskID, err)
	}
}

// setLiveTask records (or clears, with 0) the task this process is recognising.
func (s *Service) setLiveTask(taskID uint) {
	s.liveTaskMu.Lock()
	s.liveTaskID = taskID
	s.liveTaskMu.Unlock()
}

// liveTask is the id setLiveTask last recorded, 0 when the worker is idle.
func (s *Service) liveTask() uint {
	s.liveTaskMu.Lock()
	defer s.liveTaskMu.Unlock()
	return s.liveTaskID
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
