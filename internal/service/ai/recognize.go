package ai

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"

	"github.com/XingfenD/tag_file/internal/model"
)

// maxTaskSuggestions caps how many proposals one task persists. The guarantee
// this pipeline states — a task never leaves half a suggestion set behind —
// rests on the write being one statement: the repository's CreateSuggestions
// starts a fresh INSERT every 100 rows without a transaction between them
// (internal/repository/ai/suggestion_create.go), and a failure in a second
// batch would leave a partial pending set under a "failed" task. Staying
// safely below that batch boundary keeps every set this task can write a
// single-statement (atomic) INSERT, and the cap is applied to the parsed answer
// rather than to the rows built from it: ParseSuggestions bounds nothing (the
// prompt asks for 1–6 tags and a verbose model obeys nobody), and a row that is
// going to be thrown away must not first be paid for with the library lookups
// buildSuggestions runs over it.
//
// What the cap does to the remainder is decided and stated here rather than
// silently dropped: the first maxTaskSuggestions rows are kept, the rest are
// discarded, and the task's Error column records what was thrown away while
// the status still becomes "done". Reviewers see a complete (truncated) set
// plus a note about the loss — never a split write. 自逐帧识别起, 上限作用在
// 跨帧合并后的集合上（mergeRawTags 之后）, 而不是任何单帧的答案上。
const maxTaskSuggestions = 50

// processTask turns one claimed task into pending suggestions, or into a failed task
// carrying the reason. It owns the status column: every step of the pipeline below
// reports through it, so there is exactly one place where a task stops being
// "running".
//
// Suggestions are written once, at the end, and never split across batches:
// the set is capped at maxTaskSuggestions — below the repository's
// un-transactional 100-row batch boundary — so the proposal write is a single
// statement. The review queue therefore never holds half a set: the worst
// case is a whole set plus a task that still reads "failed" because the
// status write after the insert went wrong, never partial rows on a failed
// task. An answer larger than the cap is truncated, and the truncation is
// reported in the task's Error rather than dropped in silence. A human
// reading a half answer would be worse off than one reading an error.
func (s *Service) processTask(task *model.AITask) (err error) {
	// The deferred block is the one place this task stops being "running", and the
	// recover sits inside it on purpose: a panic in the recognition has to reach the
	// same terminal write an ordinary error does, because a row left at "running"
	// claims work nobody is doing — and an unrecovered panic would take the whole
	// WebUI down with it. The reason is what the reviewer sees, the stack is the
	// operator's.
	defer func() {
		if p := recover(); p != nil {
			log.Printf("[ai] 任务 %d 识别过程 panic: %v\n%s", task.TaskID, p, debug.Stack())
			err = fmt.Errorf("识别过程发生 panic: %v", p)
		}
		if err == nil {
			return
		}
		if updateErr := s.aiRepo.UpdateTask(task).SetStatus(TaskFailed).SetError(err.Error()).Exec(); updateErr != nil {
			// Nothing else can report this, and the log is all the operator gets. The
			// row stays "running", which RetryTask now accepts for a running row no
			// live worker owns — that is the user's way out of this state.
			log.Printf("[ai] 任务 %d 置为失败时写库出错: %v", task.TaskID, updateErr)
		}
	}()
	return s.recognize(task)
}

// recognize dispatches the two entries of the split pipeline: a normal task
// runs stage one (vision, every successful frame's raw answer stored verbatim)
// and hands the stored texts to stage two; a task carrying SourceTaskID is the
// 只重新整理 entry and skips vision entirely (recognizeRefineTask, refine.go).
// A panic anywhere still reaches processTask's deferred failure write.
func (s *Service) recognize(task *model.AITask) error {
	if task.SourceTaskID != 0 {
		return s.recognizeRefineTask(task)
	}
	return s.recognizeVisionTask(task)
}

// recognizeVisionTask is the pinned step order of stage one, in one piece:
// file → settings → client → frames → prompt(按整理开关分流) → 逐帧 model 一次
// → 成功即落原文. Parsing, merging, capping and the terminal write belong to
// stage two (refine.go), which reads the raw answers back from the table —
// 「阶段二输入统一从库读」. What stays in memory across the handoff is only the
// frameLoss ledger of frames whose answer never reached the table, because
// the off-mode notes must name exactly the frames today's single loop named.
func (s *Service) recognizeVisionTask(task *model.AITask) error {
	file, err := s.fileSvc.FindByID(task.FileID)
	if err != nil {
		return fmt.Errorf("查询文件 %d 失败: %w", task.FileID, err)
	}
	if file == nil {
		// Enqueued and then deleted or unindexed; the row outlived the file.
		return fmt.Errorf("文件 %d 已不在库里, 无法识别", task.FileID)
	}

	settings := s.Settings()
	if err := checkProviderModel(settings); err != nil {
		return err
	}
	client := s.Client
	if client == nil {
		// Built per task (NewService deliberately leaves Client nil), so editing the
		// provider, address or model takes effect on the next file without a restart.
		client = newVisionClient(settings)
	}

	// The seams below carry their own deadlines (120s of ffmpeg, 180s of HTTP), and
	// stopping mid-flight would risk the all-or-nothing property above.
	ctx := context.Background()

	frames, err := s.loadFrames(ctx, file, settings)
	if err != nil {
		return err
	}
	// 提示词分流: 整理开启时识图只管描述(格式与库对齐随整理提示词走), 关闭时
	// 还是今天那份 BuildPrompt —— 一字未动是 off 回归的一部分。
	prompt := visionFreePrompt
	if !refineEnabled(settings) {
		if prompt, err = s.BuildPrompt(); err != nil {
			return err
		}
	}

	// 逐帧识别: 每帧一次 Recognize(单图切片)。传输失败(或原文写库失败)的帧
	// 没有原文行, 只记进 failedFrames/firstErr 这套内存簿记, 注记文案由阶段二
	// 按今天的形状重建; 成功的帧无论答案是不是合法 JSON 都先逐字落库 —— 这正是
	// 落库的价值(畸形答案日后可靠重整理救回)。一帧坏掉只丢那一帧(部分成功沿用
	// 「done + Error 说明」的先例, 截断即其原型); 全帧皆败无事可写, 整任务失败,
	// 首条原因以 %w 包裹保住归因链。
	lost := make(frameLoss)
	stored := false
	for i, frame := range frames {
		output, err := client.Recognize(ctx, prompt, [][]byte{frame})
		if err == nil {
			record := model.AIFrameOutput{
				TaskID:     task.TaskID,
				FileID:     task.FileID,
				FrameIndex: i + 1,
				RawText:    output,
			}
			err = s.aiRepo.CreateFrameOutput(&record)
		}
		if err != nil {
			lost.add(i+1, err)
			continue
		}
		stored = true
	}
	if !stored {
		return fmt.Errorf(allFramesFailedFormat, len(frames), lost.firstErr())
	}
	outputs, err := s.aiRepo.ListFrameOutputs(task.TaskID)
	if err != nil {
		return fmt.Errorf("读取任务 %d 的识图原文失败: %w", task.TaskID, err)
	}
	return s.refineStage(task, outputs, settings, lost, len(frames))
}

// loadFrames produces the image half of the model request: the file itself for an
// image, deduplicated frames for a video.
func (s *Service) loadFrames(ctx context.Context, file *model.File, settings Settings) ([][]byte, error) {
	kind, ok := IsMediaFile(file.FilePath)
	if !ok {
		// Enqueue gates on the same predicate, so this only fires for a row whose
		// extension changed under us — worth naming instead of sending to the model.
		return nil, fmt.Errorf("文件 %s 不是可识别的图片或视频", file.FilePath)
	}
	// File.FilePath 是相对 scan 根存的路径, 而 WebUI 启动时已经 os.Chdir 进了那个
	// 根目录(cmd/webui/main.go:43), 所以当前工作目录就是拼回真实路径的基准。
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("取当前工作目录失败: %w", err)
	}
	path := filepath.Join(cwd, file.FilePath)

	if kind == kindImage {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("读取图片 %s 失败: %w", file.FilePath, err)
		}
		return [][]byte{data}, nil
	}

	extracted, err := s.Extractor.Extract(ctx, path, settings.FrameInterval, settings.MaxFrames)
	if err != nil {
		return nil, fmt.Errorf("视频 %s 抽帧失败: %w", file.FilePath, err)
	}
	// The extractor handed its scratch directory over with the paths; from
	// here it is this function's job to delete it, whatever happens below.
	defer extracted.Cleanup()
	kept, err := FilterSimilarFrames(extracted.Paths, settings.DiffThreshold, settings.MaxFrames)
	if err != nil {
		return nil, fmt.Errorf("视频 %s 过滤近似帧失败: %w", file.FilePath, err)
	}
	if len(kept) == 0 {
		// 真实的抽帧实现在返回 0 帧时自己就先报错了, 到这里只会是过滤把帧全丢了。
		// 拿空图集去问模型只会换回一条没头没尾的失败, 不如在这里说清楚。
		return nil, fmt.Errorf("视频 %s 过滤后没有可用帧", file.FilePath)
	}
	return kept, nil
}

// buildSuggestions turns the model's raw terms into review-queue rows: every field
// the model wrote is stored verbatim, the effective values start as a copy of it,
// and nothing is created in the tag library — the human decides that later (Task 9).
func (s *Service) buildSuggestions(task *model.AITask, raws []RawTag) ([]model.AISuggestion, error) {
	known, err := s.knownCategories()
	if err != nil {
		return nil, err
	}

	var invented []string
	items := make([]model.AISuggestion, 0, len(raws))
	for _, raw := range raws {
		var matchedID *uint
		if known[raw.Category] {
			// matchTag 只对存在的分类调用。它对陌生分类的答案本来就是「未命中」, 但
			// 每问一次会 log 一行, 一批新词能把 stderr 刷满 —— 所以在这里分流, 陌生
			// 分类统一汇总成下面的一行(Task 7 移交的降噪责任)。
			matchedID, err = s.matchTag(raw.Category, raw.Tag)
			if err != nil {
				return nil, fmt.Errorf("匹配建议 %s/%s 失败: %w", raw.Category, raw.Tag, err)
			}
		} else {
			invented = append(invented, raw.Category)
		}

		items = append(items, model.AISuggestion{
			TaskID: task.TaskID,
			FileID: task.FileID,
			// 原始输出与生效值同值入库: 审核只改 Final*, 模型说过什么永远留着。
			CategoryName:      raw.Category,
			TagName:           raw.Tag,
			TagValue:          raw.Value,
			FinalCategoryName: raw.Category,
			FinalTagName:      raw.Tag,
			FinalValue:        raw.Value,
			MatchedTagID:      matchedID,
			ReviewStatus:      ReviewPending,
		})
	}
	if len(invented) > 0 {
		log.Printf("[ai] 任务 %d: 模型提出了 %d 条未知分类的建议(%s), 原样进待审队列",
			task.TaskID, len(invented), categoryList(invented))
	}
	return items, nil
}

// knownCategories is the category set the prompt showed the model, read once per
// task: anything outside it is a name the model made up rather than a lookup miss.
func (s *Service) knownCategories() (map[string]bool, error) {
	categories, _, err := s.categorySvc.ListCategories(0, 0)
	if err != nil {
		return nil, fmt.Errorf("读取分类列表失败: %w", err)
	}
	known := make(map[string]bool, len(categories))
	for _, category := range categories {
		known[category.CategoryName] = true
	}
	return known, nil
}

// categoryList names the distinct invented categories of one task, ordered so the
// summary line is reproducible. 空分类也要有个名字, 否则它在日志里是隐形的。
func categoryList(names []string) string {
	seen := make(map[string]bool, len(names))
	list := make([]string, 0, len(names))
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		if name == "" {
			name = "(空)"
		}
		list = append(list, name)
	}
	sort.Strings(list)
	return strings.Join(list, "、")
}

// frameIndexList names the failed frames of one task in the partial-failure note:
// the indexes are 1-based (they are what a reviewer counts when they count frames)
// and joined by 「、」, the same separator categoryList uses for readable lists.
func frameIndexList(indexes []int) string {
	parts := make([]string, 0, len(indexes))
	for _, index := range indexes {
		parts = append(parts, strconv.Itoa(index))
	}
	return strings.Join(parts, "、")
}
