package ai

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
)

// refineBatchSize is how many stored frames go into one refine call. The
// budget is the text model's context, not the GPU: 「每帧原文 × 批帧数 + 标签库
// 提示词」at 20 frames per batch keeps a run of verbose free-text answers well
// inside what common endpoints accept, while one call per batch keeps the
// cross-frame normalisation (synonyms, dedupe) doing its job across most of a
// video at once. Batching 25 frames costs two calls (20 + 5).
const refineBatchSize = 20

// Refine statuses stored in ai_tasks.refine_status. The empty string — the
// absence of any value, hence no constant — means stage two never ran: the
// refine endpoint is off (单段管线不碰这列), or the task failed before stage
// two started. pending is written when stage two starts on the refine path,
// and done/failed is its terminal value, whichever path ended it.
const (
	RefinePending = "pending"
	RefineDone    = "done"
	RefineFailed  = "failed"
)

// allFramesFailedFormat is the terminal error when no frame's answer survives
// to produce anything — stage one's「没有一行入库」and off-mode stage two's
// 「有行但全部解析失败」are the same message with the same denominator and the
// same %w wrap of the earliest reason, so they share one constant, verbatim.
const allFramesFailedFormat = "全部 %d 帧识别失败, 首条原因: %w"

// frameLoss is stage one's in-memory ledger of frames whose raw answer never
// reached the frame table — transport failures and the rare raw-write failure.
// 「阶段二输入统一从库读」constrains the RawText only; this bookkeeping is
// carried to stage two in memory because without it the off-mode notes could
// not name the lost frames at all (they have no rows). Keys are the same
// 1-based frame indexes today's recognize loop recorded, and the map answers
// both of its queries in frame order: indexes() is the sorted list, firstErr()
// the reason of the earliest lost frame — which is exactly the error today's
// in-order loop kept as firstErr.
type frameLoss map[int]error

// add records one lost frame and never overwrites: a frame already lost at
// stage one cannot also earn a parse failure (its text is not in the table),
// and if it somehow did the earlier reason is the one in frame order.
func (l frameLoss) add(index int, err error) {
	if _, seen := l[index]; !seen {
		l[index] = err
	}
}

func (l frameLoss) indexes() []int {
	indexes := make([]int, 0, len(l))
	for index := range l {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	return indexes
}

func (l frameLoss) firstErr() error {
	if len(l) == 0 {
		return nil
	}
	return l[l.indexes()[0]]
}

// batchLoss is one refine batch that produced nothing — call or parse —
// carrying the FrameIndex span of the batch (first and last frame it covered),
// the failed/total batch counts for the note ratio, and the reason.
type batchLoss struct {
	first, last   int
	failed, total int
	err           error
}

// recognizeRefineTask is stage two's second entry: the 只重新整理 task runs
// over the source vision task's stored raw answers. No file is read, no frame
// extracted, no vision call made — so the vision endpoint's config is not even
// checked (checkProviderModel is skipped along with the calls it guards).
// A source row that is gone, or was never able to store a raw answer, leaves
// nothing to refine; the refine endpoint's own checks (开启与否、模型名) happen
// where a normal run's stage two meets them too, and an endpoint switched off
// after enqueue fails the task instead of falling back to single-stage —
// the per-task client build is the precedent for config read at execution.
func (s *Service) recognizeRefineTask(task *model.AITask) error {
	source, err := s.aiRepo.FindTask(task.SourceTaskID)
	if err != nil {
		return fmt.Errorf("查询源任务 %d 失败: %w", task.SourceTaskID, err)
	}
	if source == nil {
		// Deleted after this task was enqueued (spec 5.2): its raws went with it.
		return errors.New("源任务没有识图原文")
	}
	outputs, err := s.aiRepo.ListFrameOutputs(source.TaskID)
	if err != nil {
		return fmt.Errorf("读取任务 %d 的识图原文失败: %w", source.TaskID, err)
	}
	if len(outputs) == 0 {
		return errors.New("源任务没有识图原文")
	}

	settings := s.Settings()
	if !refineEnabled(settings) {
		return errors.New("整理端点未开启")
	}
	// 阶段二从库读到什么就认什么: 源任务当年的传输失败帧没有行, 本趟无从得知,
	// 簿记从零开始; FrameCount 沿用源值(本任务没有喂过任何帧)。
	return s.refineStage(task, outputs, settings, make(frameLoss), source.FrameCount)
}

// refineStage is stage two for both entries, over the raw answers read back
// from the frame table. The mode is the settings the run is executing under
// (one Settings() read per task keeps stage one and stage two agreeing): off
// parses every frame's text itself, reproducing today's single-stage result
// byte-for-byte including the notes; on hands the batches to the refine
// endpoint. frameCount is the denominator of the frame notes and the value
// written to FrameCount — today's len(frames) for a vision run, the source's
// for a re-refine.
func (s *Service) refineStage(task *model.AITask, outputs []model.AIFrameOutput, settings Settings, lost frameLoss, frameCount int) error {
	if refineEnabled(settings) {
		return s.refineBatched(task, outputs, settings, lost, frameCount)
	}
	return s.refineDirect(task, outputs, lost, frameCount)
}

// refineDirect is the off-mode stage two: per-frame ParseSuggestions over the
// stored texts, then the same merge/cap/write tail. The note ledger is the
// union, in frame order, of stage one's in-memory transport failures and the
// rows whose parse failed, which is what makes the partial note and the
// all-fail error the byte-for-byte descendants of today's — same wording,
// same denominator, same first reason wrapped with %w.
func (s *Service) refineDirect(task *model.AITask, outputs []model.AIFrameOutput, lost frameLoss, frameCount int) error {
	merged := make([]RawTag, 0, len(outputs))
	for _, output := range outputs {
		raws, err := ParseSuggestions(output.RawText)
		if err != nil {
			lost.add(output.FrameIndex, err)
			continue
		}
		merged = append(merged, raws...)
	}
	if len(merged) == 0 {
		// 今天这条分支同样覆盖「有行成功但全部解析失败」: 此刻 lost 里每一帧
		// 都在(传输败或解析败), firstErr 是帧序最早那条。
		return fmt.Errorf(allFramesFailedFormat, frameCount, lost.firstErr())
	}
	return s.finalizeSuggestions(task, merged, lost, nil, frameCount, false)
}

// refineBatched is the on-mode stage two: the stored frames are cut into
// batches of refineBatchSize, each batch becomes one text-only call
// (Recognize with nil images) whose answer goes through the same
// ParseSuggestions. Partial success moves from frame level to batch level —
// one bad batch drops that batch's tags and is named in the Error with its
// frame span; every batch bad is a failed task whose raw answers stay intact
// in the table, cheaply recoverable through 重新整理. The refine endpoint's
// model name is checked here, at execution time, mirroring checkProviderModel
// one stage earlier.
func (s *Service) refineBatched(task *model.AITask, outputs []model.AIFrameOutput, settings Settings, lost frameLoss, frameCount int) error {
	if err := checkRefineModel(settings); err != nil {
		return err
	}
	// pending 落在阶段二真正开始处(校验之后): 它是「整理在跑」的提示,
	// 配置就缺模型的失败还没进这一段, 不该谎称跑过。
	s.markRefinePending(task)

	client := s.RefineClient
	if client == nil {
		// Built per task, the mirror of the vision client: editing the refine
		// endpoint takes effect on the next run without a restart.
		client = newRefineClient(settings)
	}
	// 与阶段一同款理由: seam 自带超时, 中途放手会赌上建议的整套性。
	ctx := context.Background()

	merged := make([]RawTag, 0, len(outputs))
	totalBatches := (len(outputs) + refineBatchSize - 1) / refineBatchSize
	var lostBatches []batchLoss
	for start := 0; start < len(outputs); start += refineBatchSize {
		end := min(start+refineBatchSize, len(outputs))
		batch := outputs[start:end]
		out, err := s.refineBatch(ctx, client, batch)
		if err == nil {
			var raws []RawTag
			raws, err = ParseSuggestions(out)
			if err == nil {
				merged = append(merged, raws...)
				continue
			}
		}
		lostBatches = append(lostBatches, batchLoss{
			first: batch[0].FrameIndex,
			last:  batch[len(batch)-1].FrameIndex,
			total: totalBatches,
			err:   err,
		})
	}
	if len(lostBatches) == totalBatches {
		// 全批皆败无事可写(整理端一点可用输出都没给), 整任务失败; 原因里点名
		// 廉价恢复路径 —— 原文在库里, 重新整理不花 GPU。%w 保住首败归因链。
		return s.failRefine(task, fmt.Errorf("全部 %d 批整理失败, 可重新整理, 首条原因: %w", totalBatches, lostBatches[0].err))
	}
	for i := range lostBatches {
		lostBatches[i].failed = len(lostBatches)
	}
	return s.finalizeSuggestions(task, merged, lost, lostBatches, frameCount, true)
}

// refineBatch builds and sends one batch's text-only refine call.
func (s *Service) refineBatch(ctx context.Context, client VisionClient, batch []model.AIFrameOutput) (string, error) {
	prompt, err := s.BuildRefinePrompt(batch)
	if err != nil {
		return "", err
	}
	return client.Recognize(ctx, prompt, nil)
}

// markRefinePending records that stage two has started. The column is a UI
// hint whose terminal value is written later; a failed hint write costs only
// the hint — log it and refine on, the alternative is failing a task over a
// bookkeeping column.
func (s *Service) markRefinePending(task *model.AITask) {
	if err := s.aiRepo.UpdateTask(task).SetRefineStatus(RefinePending).Exec(); err != nil {
		log.Printf("[ai] 任务 %d 整理状态置 pending 时写库出错: %v", task.TaskID, err)
	}
}

// failRefine writes stage two's own terminal RefineStatus before handing the
// reason up to processTask's deferred write. The handoff is safe because the
// two writes touch disjoint column whitelists — the defer owns status/error,
// this owns refine_status — so no column is lost between them.
func (s *Service) failRefine(task *model.AITask, err error) error {
	if updateErr := s.aiRepo.UpdateTask(task).SetRefineStatus(RefineFailed).Exec(); updateErr != nil {
		log.Printf("[ai] 任务 %d 整理状态置 failed 时写库出错: %v", task.TaskID, updateErr)
	}
	return err
}

// finalizeSuggestions is the pipeline's common tail — both modes, both
// entries: merge the raw terms, cap, match, write the suggestions, then one
// terminal update carrying the merged notes. It replaces what the single-stage
// recognize did after its loop, and the off-mode path keeps that tail's every
// word and column: notes joined with 「；」, frame note first and truncation
// note last, no RefineStatus write at all (the ” the row already holds is the
// off-mode value).
func (s *Service) finalizeSuggestions(task *model.AITask, merged []RawTag, lost frameLoss, lostBatches []batchLoss, frameCount int, refined bool) error {
	merged = mergeRawTags(merged)

	var notes []string
	if len(lost) > 0 {
		// 今天的逐字同形: 帧序列表、失败/喂入计数、帧序最早一条原因。
		notes = append(notes, fmt.Sprintf("第 %s 帧识别失败(%d/%d), 首条原因: %v",
			frameIndexList(lost.indexes()), len(lost), frameCount, lost.firstErr()))
	}
	for _, batch := range lostBatches {
		span := []int{batch.first, batch.last}
		if batch.first == batch.last {
			span = []int{batch.first}
		}
		notes = append(notes, fmt.Sprintf("第 %s 帧整理失败(%d/%d), 首条原因: %v",
			frameIndexList(span), batch.failed, batch.total, batch.err))
	}

	// Bound the set before the expensive work, not after it: every row that reaches
	// buildSuggestions is looked up in the library at two statements apiece
	// (matchTag → tagSvc.ListTags), a model answer is bounded only by the client's
	// body budget, and this runs on the one worker whose shutdown everyone waits for.
	// The rows past the cap are never matched, never written, and named in the task's
	// Error below. The cap sees the cross-frame merged set, so one verbose frame
	// cannot spend it before the others are counted.
	if len(merged) > maxTaskSuggestions {
		total := len(merged)
		notes = append(notes, fmt.Sprintf("模型各帧合并后给出了 %d 条建议, 超过单任务上限 %d 条: 仅写入前 %d 条, 丢弃其余 %d 条",
			total, maxTaskSuggestions, maxTaskSuggestions, total-maxTaskSuggestions))
		merged = merged[:maxTaskSuggestions]
	}

	suggestions, err := s.buildSuggestions(task, merged)
	if err != nil {
		return s.endFailure(task, refined, err)
	}
	if err := s.aiRepo.CreateSuggestions(suggestions); err != nil {
		return s.endFailure(task, refined, fmt.Errorf("写入 %d 条待审建议失败: %w", len(suggestions), err))
	}

	update := s.aiRepo.UpdateTask(task).SetStatus(TaskDone).SetFrameCount(frameCount)
	if refined {
		// off 模式不碰这列: 它已经停在该模式唯一的取值 '' 上。
		update = update.SetRefineStatus(RefineDone)
	}
	if len(notes) > 0 {
		update = update.SetError(strings.Join(notes, "；"))
	}
	if err := update.Exec(); err != nil {
		return s.endFailure(task, refined, err)
	}
	return nil
}

// endFailure routes a tail failure through failRefine when refine had started,
// so the hint column never stays "pending" on a task that reached a terminal
// state. Single-stage runs have no such column to settle.
func (s *Service) endFailure(task *model.AITask, refined bool, err error) error {
	if refined {
		return s.failRefine(task, err)
	}
	return err
}
