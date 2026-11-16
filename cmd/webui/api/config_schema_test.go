package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	// 空导入只为让 scan 组在本测试进程注册：newTestEngine 只 import ai，不带 bootstrap，
	// 不引入这条 import 链，TestConfigSchemaShapeAndValues 就看不到 scan_exclude。
	_ "github.com/XingfenD/tag_file/internal/bootstrap"
	"github.com/XingfenD/tag_file/internal/model"
	"github.com/XingfenD/tag_file/internal/service/config/schema"
	"github.com/XingfenD/tag_file/pkg/storage"
	"gorm.io/gorm"
)

// listOf 从 respond 信封里取出一个 JSON 数组字段（缺省返回 nil）。
func listOf(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	l, _ := m[key].([]any)
	return l
}

// mapOf 把一个 any 断言成 map[string]any（失败返回 nil）。
func mapOf(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func findGroup(groups []any, id string) (map[string]any, bool) {
	for _, g := range groups {
		m := mapOf(g)
		if m["id"] == id {
			return m, true
		}
	}
	return nil, false
}

func findField(fields []any, key string) (map[string]any, bool) {
	for _, f := range fields {
		m := mapOf(f)
		if m["key"] == key {
			return m, true
		}
	}
	return nil, false
}

// failureMap 把 failures 列表压成 key→msg。
func failureMap(out map[string]any) map[string]string {
	fails := listOf(out, "failures")
	res := make(map[string]string, len(fails))
	for _, f := range fails {
		m := mapOf(f)
		key, _ := m["key"].(string)
		msg, _ := m["msg"].(string)
		res[key] = msg
	}
	return res
}

// dbConfigValue 直接读库，返回 (value, found)。
func dbConfigValue(t *testing.T, key string) (string, bool) {
	t.Helper()
	var cfg model.Config
	err := storage.GetDB().Where("config_name = ?", key).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false
	}
	if err != nil {
		t.Fatalf("read config %s: %v", key, err)
	}
	return cfg.ConfigValue, true
}

func TestConfigSchemaShapeAndValues(t *testing.T) {
	r := newTestEngine(t)

	code, out := doRequest(t, r, "GET", "/api/config/schema", "")
	if code != 200 || out["ok"] != true {
		t.Fatalf("schema: got %d %v, want 200 ok=true", code, out)
	}

	groups := listOf(out, "groups")
	scan, ok := findGroup(groups, "scan")
	if !ok {
		t.Fatalf("schema groups missing scan: %v", groups)
	}
	ai, ok := findGroup(groups, "ai")
	if !ok {
		t.Fatalf("schema groups missing ai: %v", groups)
	}

	// scan Order(1) 必须在 ai(2) 之前。
	scanIdx, aiIdx := -1, -1
	for i, g := range groups {
		switch mapOf(g)["id"] {
		case "scan":
			scanIdx = i
		case "ai":
			aiIdx = i
		}
	}
	if scanIdx < 0 || aiIdx < 0 || scanIdx >= aiIdx {
		t.Fatalf("groups out of order: scan@%d ai@%d (%v)", scanIdx, aiIdx, groups)
	}
	if scan["title"] == "" || ai["title"] == "" {
		t.Fatalf("groups missing title: scan=%v ai=%v", scan, ai)
	}

	fields := listOf(ai, "fields")
	want := []string{
		"ollama_model", "ollama_url", "ai_frame_interval",
		"ai_frame_diff_threshold", "ai_max_frames",
	}
	for _, key := range want {
		if _, ok := findField(fields, key); !ok {
			t.Fatalf("ai group missing field %s: %v", key, fields)
		}
	}

	// 未种子时 ollama_url 取 default。
	url, ok := findField(fields, "ollama_url")
	if !ok {
		t.Fatal("ai group missing ollama_url")
	}
	if url["value"] != "http://127.0.0.1:11434" {
		t.Fatalf("ollama_url value = %v, want default", url["value"])
	}

	interval, ok := findField(fields, "ai_frame_interval")
	if !ok {
		t.Fatal("ai group missing ai_frame_interval")
	}
	if interval["min_exclusive"] != true {
		t.Fatalf("ai_frame_interval min_exclusive = %v, want true", interval["min_exclusive"])
	}
}

func TestConfigSchemaMarksInvalidStored(t *testing.T) {
	r := newTestEngine(t)

	// 绕过服务校验，直接落一条非法存量值。
	if err := storage.GetDB().Create(&model.Config{
		ConfigName:  "ollama_url",
		ConfigValue: "no-scheme",
	}).Error; err != nil {
		t.Fatalf("write invalid ollama_url: %v", err)
	}

	_, out := doRequest(t, r, "GET", "/api/config/schema", "")
	ai, ok := findGroup(listOf(out, "groups"), "ai")
	if !ok {
		t.Fatalf("missing ai group: %v", out)
	}
	url, ok := findField(listOf(ai, "fields"), "ollama_url")
	if !ok {
		t.Fatal("missing ollama_url field")
	}
	if url["invalid"] != true {
		t.Fatalf("ollama_url invalid = %v, want true", url["invalid"])
	}
	if url["invalid_reason"] == "" || url["invalid_reason"] == nil {
		t.Fatalf("ollama_url invalid_reason empty: %v", url)
	}
	if url["value"] != "no-scheme" {
		t.Fatalf("ollama_url value = %v, want no-scheme", url["value"])
	}
}

func TestSetGroupEndpointStrict(t *testing.T) {
	r := newTestEngine(t)

	invalidBody := "group=ai&ollama_url=http://127.0.0.1:11434&ollama_model=&ai_frame_interval=1&ai_frame_diff_threshold=10&ai_max_frames=201"
	code, out := doRequest(t, r, "POST", "/api/config/group", invalidBody)
	if code != 400 || out["ok"] != false {
		t.Fatalf("invalid group save: got %d %v, want 400 ok=false", code, out)
	}
	fails := failureMap(out)
	if _, ok := fails["ai_max_frames"]; !ok {
		t.Fatalf("failures missing ai_max_frames: %v", fails)
	}
	if _, found := dbConfigValue(t, "ollama_url"); found {
		t.Fatal("ollama_url was written despite validation failure")
	}

	// 全合法重发 → 200。
	legalBody := "group=ai&ollama_url=http://127.0.0.1:11434&ollama_model=&ai_frame_interval=1&ai_frame_diff_threshold=10&ai_max_frames=10"
	code, out = doRequest(t, r, "POST", "/api/config/group", legalBody)
	if code != 200 || out["ok"] != true {
		t.Fatalf("legal group save: got %d %v, want 200 ok=true", code, out)
	}

	// 缺键重发 → failures 含「缺少字段」。
	missingBody := "group=ai&ollama_url=http://127.0.0.1:11434&ollama_model=&ai_frame_interval=1&ai_max_frames=10"
	code, out = doRequest(t, r, "POST", "/api/config/group", missingBody)
	if code != 400 || out["ok"] != false {
		t.Fatalf("missing-key group save: got %d %v, want 400 ok=false", code, out)
	}
	fails = failureMap(out)
	found := false
	for _, msg := range fails {
		if msg == "缺少字段" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("failures missing 缺少字段: %v", fails)
	}
}

func TestSetGroupRejectsRE2InvalidRegex(t *testing.T) {
	r := newTestEngine(t)

	// JS 合法、Go RE2 非法的前瞻断言。
	body := `group=scan&scan_exclude=["(?=a)"]`
	code, out := doRequest(t, r, "POST", "/api/config/group", body)
	if code != 400 || out["ok"] != false {
		t.Fatalf("invalid regex group save: got %d %v, want 400 ok=false", code, out)
	}
	fails := failureMap(out)
	if _, ok := fails["scan_exclude"]; !ok {
		t.Fatalf("failures missing scan_exclude: %v", fails)
	}
}

func TestListConfigDescription(t *testing.T) {
	r := newTestEngine(t)

	// 直接写一条带 Description 的未知键。
	if err := storage.GetDB().Create(&model.Config{
		ConfigName:  "custom_unknown",
		ConfigValue: "x",
		Description: "自定义说明",
	}).Error; err != nil {
		t.Fatalf("write unknown key: %v", err)
	}

	_, out := doRequest(t, r, "GET", "/api/config", "")
	found := false
	for _, item := range listOf(out, "items") {
		m := mapOf(item)
		if m["key"] == "custom_unknown" {
			found = true
			if m["description"] != "自定义说明" {
				t.Fatalf("custom_unknown description = %v, want 自定义说明", m["description"])
			}
		}
	}
	if !found {
		t.Fatalf("items missing custom_unknown: %v", out)
	}
}

func TestListConfigMasksSecret(t *testing.T) {
	r := newTestEngine(t)

	// 测试内注册一个唯一 secret 字段。必须放在本文件最后：注册表是进程级全局。
	schema.RegisterGroup(schema.Group{ID: "testsecret_group", Title: "测试密钥", Order: 99})
	schema.Register(schema.Field{
		Key:         "testsecret_key",
		Group:       "testsecret_group",
		Label:       "测试密钥",
		Description: "测试密钥字段",
		Type:        schema.TypeSecret,
		Default:     "",
	})

	if err := storage.GetDB().Create(&model.Config{
		ConfigName:  "testsecret_key",
		ConfigValue: "sk-1",
	}).Error; err != nil {
		t.Fatalf("write secret key: %v", err)
	}

	_, out := doRequest(t, r, "GET", "/api/config", "")
	var item map[string]any
	for _, it := range listOf(out, "items") {
		if mapOf(it)["key"] == "testsecret_key" {
			item = mapOf(it)
			break
		}
	}
	if item == nil {
		t.Fatalf("items missing testsecret_key: %v", out)
	}
	if item["value"] != "" {
		t.Fatalf("list leaked secret value: %v", item["value"])
	}
	if item["has_value"] != true {
		t.Fatalf("list has_value = %v, want true", item["has_value"])
	}

	_, out = doRequest(t, r, "GET", "/api/config/schema", "")
	grp, ok := findGroup(listOf(out, "groups"), "testsecret_group")
	if !ok {
		t.Fatalf("schema missing testsecret_group: %v", out)
	}
	f, ok := findField(listOf(grp, "fields"), "testsecret_key")
	if !ok {
		t.Fatalf("schema missing testsecret_key: %v", grp)
	}
	if f["value"] != "" {
		t.Fatalf("schema leaked secret value: %v", f["value"])
	}
	if f["has_value"] != true {
		t.Fatalf("schema has_value = %v, want true", f["has_value"])
	}
}

// TestConfigSchemaMasksSecretDefault 钉住 secret 键的非空默认值同样不回传：
// Default 字段自身就可能是明文密钥，掩码 value 还不够。
func TestConfigSchemaMasksSecretDefault(t *testing.T) {
	// 独立于 TestListConfigMasksSecret 的组/键，避免与 testsecret_key 注册冲突。
	schema.RegisterGroup(schema.Group{ID: "testsecretdef_group", Title: "测试密钥默认值", Order: 100})
	schema.Register(schema.Field{
		Key:         "testsecretdef_key",
		Group:       "testsecretdef_group",
		Label:       "测试密钥默认值",
		Description: "带默认值的密钥",
		Type:        schema.TypeSecret,
		Default:     "sk-default-secret",
	})

	r := newTestEngine(t)

	// 原始响应体任何位置都不得出现默认值字符串。
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/config/schema", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("schema: got %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "sk-default-secret") {
		t.Fatalf("schema body leaked secret default: %s", w.Body.String())
	}

	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode schema: %v (body=%s)", err, w.Body.String())
	}
	grp, ok := findGroup(listOf(out, "groups"), "testsecretdef_group")
	if !ok {
		t.Fatalf("schema missing testsecretdef_group: %v", out)
	}
	f, ok := findField(listOf(grp, "fields"), "testsecretdef_key")
	if !ok {
		t.Fatalf("schema missing testsecretdef_key: %v", grp)
	}
	if f["default"] != "" {
		t.Fatalf("schema leaked secret default: %v", f["default"])
	}
	if f["value"] != "" {
		t.Fatalf("schema leaked secret value: %v", f["value"])
	}
}
