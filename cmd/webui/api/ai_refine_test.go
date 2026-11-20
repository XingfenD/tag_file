package api

import (
	"fmt"
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	aiSvc "github.com/XingfenD/tag_file/internal/service/ai"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

// seedTask 直写一行识别任务，返回其主键。
func seedTask(t *testing.T, db *gorm.DB, fileID uint, status string, sourceTaskID uint, refineStatus string) uint {
	t.Helper()
	task := model.AITask{FileID: fileID, Status: status, SourceTaskID: sourceTaskID, RefineStatus: refineStatus}
	mustCreate(t, db, &task)
	return task.TaskID
}

// seedOutput 直写一帧识图原文，挂在 taskID 行下。
func seedOutput(t *testing.T, db *gorm.DB, taskID, fileID uint, frameIndex int, rawText string) {
	t.Helper()
	mustCreate(t, db, &model.AIFrameOutput{TaskID: taskID, FileID: fileID, FrameIndex: frameIndex, RawText: rawText})
}

// itemsByTaskID 把任务列表信封按 task_id 归档，让断言不依赖行序。
func itemsByTaskID(t *testing.T, out map[string]any) map[uint]map[string]any {
	t.Helper()
	items := listOf(out, "items")
	if items == nil {
		t.Fatalf("response has no items array: %v", out)
	}
	byID := make(map[uint]map[string]any, len(items))
	for _, raw := range items {
		m := mapOf(raw)
		if m == nil {
			t.Fatalf("item is not an object: %#v", raw)
		}
		id, ok := m["task_id"].(float64)
		if !ok {
			t.Fatalf("item task_id = %#v, want JSON number", m["task_id"])
		}
		byID[uint(id)] = m
	}
	return byID
}

func wantBool(t *testing.T, m map[string]any, key string, want bool) {
	t.Helper()
	got, ok := m[key].(bool)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %v (item=%v)", key, m[key], want, m)
	}
}

func wantNum(t *testing.T, m map[string]any, key string, want uint) {
	t.Helper()
	got, ok := m[key].(float64)
	if !ok || uint(got) != want {
		t.Fatalf("%s = %#v, want %d (item=%v)", key, m[key], want, m)
	}
}

func wantStr(t *testing.T, m map[string]any, key, want string) {
	t.Helper()
	got, ok := m[key].(string)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %q (item=%v)", key, m[key], want, m)
	}
}

// TestListAITasks_RefineFields 钉住列表信封的四个新字段：source_task_id/refine_status
// 逐行透传，has_raw 走源解析（重整理任务按源任务判定），can_refine 是 has_raw 与
// 服务器侧整理端点开关的合取——off 时必须为 false，配置端点+模型后重取才翻 true，
// 且照旧没有原文的行保持 false。并且整页只付一次 ai_frame_outputs 聚合读——
// 计数器就是无 N+1 的断言。
func TestListAITasks_RefineFields(t *testing.T) {
	r := newTestEngine(t)
	db := storage.GetDB()

	f := model.File{FilePath: "clip.mp4", FileName: "clip.mp4", FileType: "video/mp4"}
	mustCreate(t, db, &f)

	withRaw := seedTask(t, db, f.FileID, aiSvc.TaskDone, 0, "")
	seedOutput(t, db, withRaw, f.FileID, 0, "raw-a0")
	seedOutput(t, db, withRaw, f.FileID, 1, "raw-a1")
	noRaw := seedTask(t, db, f.FileID, aiSvc.TaskDone, 0, "")
	// 重整理任务自己不存原文，has_raw 必须由源任务解析出来。
	refine := seedTask(t, db, f.FileID, aiSvc.TaskDone, withRaw, aiSvc.RefineDone)

	// 计数落在查询回调里：页上有 3 行任务，逐行读会数出 3，一次聚合只会数出 1。
	// Scan 走 Row 链、Find 走 Query 链，两条都钉上，实现选哪条都数得到。
	var rawReads int
	countRaw := func(d *gorm.DB) {
		if strings.Contains(d.Statement.SQL.String(), "ai_frame_outputs") {
			rawReads++
		}
	}
	const counterName = "test:count_frame_output_reads"
	if err := db.Callback().Query().After("gorm:query").Register(counterName+"/query", countRaw); err != nil {
		t.Fatalf("register counter: %v", err)
	}
	if err := db.Callback().Row().After("gorm:row").Register(counterName+"/row", countRaw); err != nil {
		t.Fatalf("register counter: %v", err)
	}
	defer func() {
		if err := db.Callback().Query().Remove(counterName + "/query"); err != nil {
			t.Fatalf("remove counter: %v", err)
		}
		if err := db.Callback().Row().Remove(counterName + "/row"); err != nil {
			t.Fatalf("remove counter: %v", err)
		}
	}()

	code, out := doRequest(t, r, "GET", "/api/ai/tasks", "")
	if code != 200 || out["ok"] != true {
		t.Fatalf("list: got %d %v, want 200 ok=true", code, out)
	}
	if rawReads != 1 {
		t.Fatalf("ai_frame_outputs reads for one page = %d, want exactly 1 (一次聚合, 无 N+1)", rawReads)
	}

	byID := itemsByTaskID(t, out)
	if len(byID) != 3 {
		t.Fatalf("got %d items, want 3", len(byID))
	}

	item := byID[withRaw]
	if item == nil {
		t.Fatalf("task %d missing from items", withRaw)
	}
	wantNum(t, item, "source_task_id", 0)
	wantStr(t, item, "refine_status", "")
	wantBool(t, item, "has_raw", true)

	item = byID[noRaw]
	if item == nil {
		t.Fatalf("task %d missing from items", noRaw)
	}
	wantBool(t, item, "has_raw", false)

	item = byID[refine]
	if item == nil {
		t.Fatalf("task %d missing from items", refine)
	}
	wantNum(t, item, "source_task_id", withRaw)
	wantStr(t, item, "refine_status", aiSvc.RefineDone)
	wantBool(t, item, "has_raw", true)

	// 默认态整理端点 off：has_raw 只回答库里有没有原文，「重新整理」的门必须由
	// 服务器侧的 can_refine 把 off 也算进去——否则 off 模式的每一行都挂着一个
	// 点必被拒的按钮。
	wantBool(t, byID[withRaw], "can_refine", false)
	wantBool(t, byID[noRaw], "can_refine", false)
	wantBool(t, byID[refine], "can_refine", false)

	// 配置端点+模型后重取同一页：can_refine 随 has_raw 走，没有原文的行照旧 false。
	mustCreate(t, db, &model.Config{ConfigName: aiSvc.KeyRefineProvider, ConfigValue: aiSvc.ProviderOpenAI})
	mustCreate(t, db, &model.Config{ConfigName: aiSvc.KeyRefineOpenAIModel, ConfigValue: "gpt-4o"})

	code, out = doRequest(t, r, "GET", "/api/ai/tasks", "")
	if code != 200 || out["ok"] != true {
		t.Fatalf("list after refine config: got %d %v, want 200 ok=true", code, out)
	}
	byID = itemsByTaskID(t, out)
	wantBool(t, byID[withRaw], "can_refine", true)
	wantBool(t, byID[noRaw], "can_refine", false)
	wantBool(t, byID[refine], "can_refine", true)
}

// TestRefineEndpoint 镜像 retry 的端点形状：非法入参先拒，服务守卫文案逐字透传，
// 成功则落一行指向源的 queued 新任务。
func TestRefineEndpoint(t *testing.T) {
	r := newTestEngine(t)
	db := storage.GetDB()

	f := model.File{FilePath: "clip.mp4", FileName: "clip.mp4", FileType: "video/mp4"}
	mustCreate(t, db, &f)
	taskID := seedTask(t, db, f.FileID, aiSvc.TaskDone, 0, "")

	// 非法 task_id：解析失败或 0 都在触库之前拒绝。
	for _, bad := range []string{"", "0", "abc"} {
		code, out := doRequest(t, r, "POST", "/api/ai/tasks/refine", "task_id="+bad)
		if code != 400 || out["ok"] != false || out["msg"] != "无效的任务ID" {
			t.Fatalf("task_id=%q: got %d %v, want 400 无效的任务ID", bad, code, out)
		}
	}

	// 无原文的已完成任务 → 服务守卫文案逐字透传（此时端点 off 与否都轮不到发言）。
	code, out := doRequest(t, r, "POST", "/api/ai/tasks/refine", fmt.Sprintf("task_id=%d", taskID))
	if code != 400 || out["ok"] != false || out["msg"] != "该任务没有识图原文, 无法重新整理" {
		t.Fatalf("no-raws guard: got %d %v, want 400 该任务没有识图原文, 无法重新整理", code, out)
	}
	seedOutput(t, db, taskID, f.FileID, 0, "raw-0")

	// 未知 id → 与 RefineTask 同文的「不存在」。
	code, out = doRequest(t, r, "POST", "/api/ai/tasks/refine", "task_id=4242")
	if code != 400 || out["ok"] != false || out["msg"] != "识别任务 4242 不存在" {
		t.Fatalf("unknown id: got %d %v, want 400 识别任务 4242 不存在", code, out)
	}

	// 有原文但整理端点未开启（默认 off）→ 端点守卫文案。
	code, out = doRequest(t, r, "POST", "/api/ai/tasks/refine", fmt.Sprintf("task_id=%d", taskID))
	if code != 400 || out["ok"] != false || out["msg"] != "未开启整理端点, 请先在配置页配置" {
		t.Fatalf("endpoint-off guard: got %d %v, want 400 未开启整理端点, 请先在配置页配置", code, out)
	}

	mustCreate(t, db, &model.Config{ConfigName: aiSvc.KeyRefineProvider, ConfigValue: aiSvc.ProviderOpenAI})

	code, out = doRequest(t, r, "POST", "/api/ai/tasks/refine", fmt.Sprintf("task_id=%d", taskID))
	if code != 200 || out["ok"] != true || out["msg"] != "任务已加入重新整理队列" {
		t.Fatalf("success: got %d %v, want 200 任务已加入重新整理队列", code, out)
	}

	// 成功必须落成一行指向源的 queued 新任务，旧行不动。
	var created []model.AITask
	if err := db.Where("source_task_id = ?", taskID).Find(&created).Error; err != nil {
		t.Fatal(err)
	}
	if len(created) != 1 {
		t.Fatalf("queued refine rows for source %d = %d, want 1", taskID, len(created))
	}
	if created[0].Status != aiSvc.TaskQueued || created[0].FileID != f.FileID {
		t.Fatalf("new task = %+v, want queued on file %d", created[0], f.FileID)
	}

	// 在途的再点一次 → RetryTask 的同款拒绝。
	code, out = doRequest(t, r, "POST", "/api/ai/tasks/refine", fmt.Sprintf("task_id=%d", taskID))
	if code != 400 || out["ok"] != false || out["msg"] != fmt.Sprintf("该文件已有进行中的识别任务 %d,请勿重复提交", created[0].TaskID) {
		t.Fatalf("in-flight guard: got %d %v, want 400 该文件已有进行中的识别任务 %d,请勿重复提交", code, out, created[0].TaskID)
	}
}

// TestRawEndpoint 钉 GET /api/ai/tasks/:id/raw：帧序、源解析、items 非 nil、
// 未知 id 报「不存在」。
func TestRawEndpoint(t *testing.T) {
	r := newTestEngine(t)
	db := storage.GetDB()

	f := model.File{FilePath: "clip.mp4", FileName: "clip.mp4", FileType: "video/mp4"}
	mustCreate(t, db, &f)
	taskID := seedTask(t, db, f.FileID, aiSvc.TaskDone, 0, "")
	// 乱序落行，端点必须按 frame_index 回序。
	seedOutput(t, db, taskID, f.FileID, 2, "third")
	seedOutput(t, db, taskID, f.FileID, 0, "first")
	seedOutput(t, db, taskID, f.FileID, 1, "second")
	refineID := seedTask(t, db, f.FileID, aiSvc.TaskDone, taskID, aiSvc.RefineDone)
	noRawID := seedTask(t, db, f.FileID, aiSvc.TaskDone, 0, "")

	wantFrames := []struct {
		index float64
		text  string
	}{{0, "first"}, {1, "second"}, {2, "third"}}
	checkFrames := func(label string, out map[string]any) {
		items := listOf(out, "items")
		if len(items) != len(wantFrames) {
			t.Fatalf("%s: got %d items, want %d (%v)", label, len(items), len(wantFrames), out)
		}
		for i, raw := range items {
			m := mapOf(raw)
			if v, ok := m["frame_index"].(float64); !ok || uint(v) != uint(wantFrames[i].index) {
				t.Fatalf("%s item %d: frame_index = %#v, want %d", label, i, m["frame_index"], uint(wantFrames[i].index))
			}
			if m["raw_text"] != wantFrames[i].text {
				t.Fatalf("%s item %d: raw_text = %v, want %q", label, i, m["raw_text"], wantFrames[i].text)
			}
		}
	}

	code, out := doRequest(t, r, "GET", fmt.Sprintf("/api/ai/tasks/%d/raw", taskID), "")
	if code != 200 || out["ok"] != true || out["msg"] != "ok" {
		t.Fatalf("raw: got %d %v, want 200 ok=true msg=ok", code, out)
	}
	checkFrames("source task", out)

	// 重整理任务自己不存原文，GET 它必须拿到源的。
	code, out = doRequest(t, r, "GET", fmt.Sprintf("/api/ai/tasks/%d/raw", refineID), "")
	if code != 200 || out["ok"] != true {
		t.Fatalf("refine raw: got %d %v, want 200 ok=true", code, out)
	}
	checkFrames("refine task", out)

	// 无原文任务：200 + 空数组，绝不能是 null。
	code, out = doRequest(t, r, "GET", fmt.Sprintf("/api/ai/tasks/%d/raw", noRawID), "")
	if code != 200 || out["ok"] != true {
		t.Fatalf("empty raw: got %d %v, want 200 ok=true", code, out)
	}
	items, isList := out["items"].([]any)
	if !isList {
		t.Fatalf("empty items = %#v, want a non-nil array", out["items"])
	}
	if len(items) != 0 {
		t.Fatalf("empty items has %d entries, want 0", len(items))
	}

	// 未知 id → 400「识别任务 %d 不存在」；非法 id 与 DELETE 同形。
	code, out = doRequest(t, r, "GET", "/api/ai/tasks/4242/raw", "")
	if code != 400 || out["ok"] != false || out["msg"] != "识别任务 4242 不存在" {
		t.Fatalf("unknown id: got %d %v, want 400 识别任务 4242 不存在", code, out)
	}
	code, out = doRequest(t, r, "GET", "/api/ai/tasks/abc/raw", "")
	if code != 400 || out["ok"] != false || out["msg"] != "无效的任务ID" {
		t.Fatalf("bad id: got %d %v, want 400 无效的任务ID", code, out)
	}
}
