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

func findBlock(blocks []any, id string) (map[string]any, bool) {
	for _, b := range blocks {
		m := mapOf(b)
		if m["id"] == id {
			return m, true
		}
	}
	return nil, false
}

func findInputBox(boxes []any, key string) (map[string]any, bool) {
	for _, f := range boxes {
		m := mapOf(f)
		if m["key"] == key {
			return m, true
		}
	}
	return nil, false
}

// findRule 在规则列表里按 kind 找一条指令。
func findRule(rules []any, kind string) (map[string]any, bool) {
	for _, r := range rules {
		m := mapOf(r)
		if m["kind"] == kind {
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

	blocks := listOf(out, "blocks")
	scan, ok := findBlock(blocks, "scan")
	if !ok {
		t.Fatalf("schema blocks missing scan: %v", blocks)
	}
	ai, ok := findBlock(blocks, "ai")
	if !ok {
		t.Fatalf("schema blocks missing ai: %v", blocks)
	}

	// scan Order(1) 必须在 ai(2) 之前。
	scanIdx, aiIdx := -1, -1
	for i, b := range blocks {
		switch mapOf(b)["id"] {
		case "scan":
			scanIdx = i
		case "ai":
			aiIdx = i
		}
	}
	if scanIdx < 0 || aiIdx < 0 || scanIdx >= aiIdx {
		t.Fatalf("blocks out of order: scan@%d ai@%d (%v)", scanIdx, aiIdx, blocks)
	}
	if scan["title"] == "" || ai["title"] == "" {
		t.Fatalf("blocks missing title: scan=%v ai=%v", scan, ai)
	}

	boxes := listOf(ai, "input_boxes")
	// 顺序即布局契约：条件字段按「地址 → 模型 → 密钥」同序声明，
	// 前端按序位分槽，切换 provider 时其余项位置不动。
	want := []string{
		"ai_provider", "ollama_url", "ollama_model",
		"ai_openai_url", "ai_openai_model", "ai_openai_api_key",
		"ai_anthropic_url", "ai_anthropic_model", "ai_anthropic_api_key",
		"ai_frame_interval", "ai_frame_diff_threshold", "ai_max_frames",
	}
	if len(boxes) != len(want) {
		t.Fatalf("ai input box count = %d, want %d", len(boxes), len(want))
	}
	for i, key := range want {
		if mapOf(boxes[i])["key"] != key {
			t.Fatalf("ai input box[%d] = %v, want %s", i, mapOf(boxes[i])["key"], key)
		}
	}

	// required = 空值非法：必填 URL/Int/枚举为 true，模型名与 secret 为 false。
	wantRequired := map[string]bool{
		"ai_provider": true, "ollama_url": true, "ollama_model": false,
		"ai_openai_url": true, "ai_openai_model": false, "ai_openai_api_key": false,
		"ai_anthropic_url": true, "ai_anthropic_model": false, "ai_anthropic_api_key": false,
		"ai_frame_interval": true, "ai_frame_diff_threshold": true, "ai_max_frames": true,
	}
	for key, req := range wantRequired {
		f, ok := findInputBox(boxes, key)
		if !ok {
			t.Fatalf("ai block missing input box %s", key)
		}
		if f["required"] != req {
			t.Fatalf("%s required = %v, want %v", key, f["required"], req)
		}
	}

	// 未种子时 ollama_url 取 default。
	url, ok := findInputBox(boxes, "ollama_url")
	if !ok {
		t.Fatal("ai block missing ollama_url")
	}
	if url["value"] != "http://127.0.0.1:11434" {
		t.Fatalf("ollama_url value = %v, want default", url["value"])
	}

	// 无参数规则（url）的 params 必须是 {} 而非 null（spec §9）。
	urlRule, ok := findRule(listOf(url, "rules"), "url")
	if !ok {
		t.Fatalf("ollama_url rules missing url: %v", url["rules"])
	}
	urlParams := mapOf(urlRule["params"])
	if urlParams == nil || len(urlParams) != 0 {
		t.Fatalf("url rule params = %#v, want empty object {}", urlRule["params"])
	}

	// int_range 规则携带定点偏移与显示单位边界：shift=1、min_exclusive=true。
	interval, ok := findInputBox(boxes, "ai_frame_interval")
	if !ok {
		t.Fatal("ai block missing ai_frame_interval")
	}
	rangeRule, ok := findRule(listOf(interval, "rules"), "int_range")
	if !ok {
		t.Fatalf("ai_frame_interval rules missing int_range: %v", interval["rules"])
	}
	params := mapOf(rangeRule["params"])
	if params["min_exclusive"] != "true" {
		t.Fatalf("int_range min_exclusive = %v, want true", params["min_exclusive"])
	}
	if params["shift"] != "1" {
		t.Fatalf("int_range shift = %v, want 1", params["shift"])
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
	ai, ok := findBlock(listOf(out, "blocks"), "ai")
	if !ok {
		t.Fatalf("missing ai block: %v", out)
	}
	url, ok := findInputBox(listOf(ai, "input_boxes"), "ollama_url")
	if !ok {
		t.Fatal("missing ollama_url input box")
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

func TestConfigSchemaRuleCopy(t *testing.T) {
	r := newTestEngine(t)

	_, out := doRequest(t, r, "GET", "/api/config/schema", "")
	scan, ok := findBlock(listOf(out, "blocks"), "scan")
	if !ok {
		t.Fatalf("missing scan block: %v", out)
	}
	exclude, ok := findInputBox(listOf(scan, "input_boxes"), "scan_exclude")
	if !ok {
		t.Fatalf("missing scan_exclude input box: %v", scan)
	}
	rules := listOf(exclude, "rules")
	if len(rules) != 1 {
		t.Fatalf("scan_exclude rules = %v, want exactly 1", rules)
	}
	rule := mapOf(rules[0])
	if rule["kind"] != "regex" || rule["role"] != "validate" ||
		rule["hint"] != "" || rule["message"] != "不是合法的正则表达式" {
		t.Fatalf("scan_exclude rule = %v", rule)
	}
	if exclude["description"] != "输入正则表达式" {
		t.Fatalf("scan_exclude description = %v, want 输入正则表达式", exclude["description"])
	}
	// 可选动态列表：空数组合法 → required=false（容器探 "[]" 而非 ""）。
	if exclude["required"] != false {
		t.Fatalf("scan_exclude required = %v, want false", exclude["required"])
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

	// 全合法重发 → 200。strict 组要求提交每个非 secret 字段。
	legalBody := "group=ai&ai_provider=ollama&ollama_url=http://127.0.0.1:11434&ollama_model=" +
		"&ai_openai_url=https://api.openai.com/v1&ai_openai_model=gpt-4o-mini" +
		"&ai_anthropic_url=https://api.anthropic.com&ai_anthropic_model=claude-sonnet-4" +
		"&ai_frame_interval=1&ai_frame_diff_threshold=10&ai_max_frames=10"
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
	schema.RegisterBlock(schema.ConfigBlockDescriptor{
		ID:    "testsecret_group",
		Title: "测试密钥",
		Order: 99,
		InputBoxDescriptorList: []*schema.InputBoxDescriptor{
			{
				Key:         "testsecret_key",
				Label:       "测试密钥",
				Description: "测试密钥字段",
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     "",
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{Secret: true},
				},
			},
		},
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
	blk, ok := findBlock(listOf(out, "blocks"), "testsecret_group")
	if !ok {
		t.Fatalf("schema missing testsecret_group: %v", out)
	}
	f, ok := findInputBox(listOf(blk, "input_boxes"), "testsecret_key")
	if !ok {
		t.Fatalf("schema missing testsecret_key: %v", blk)
	}
	if f["value"] != "" {
		t.Fatalf("schema leaked secret value: %v", f["value"])
	}
	if f["has_value"] != true {
		t.Fatalf("schema has_value = %v, want true", f["has_value"])
	}

	// secret 描述符投影含 kind=secret、widget=password、hint「已设置，留空不修改」。
	secretRule, ok := findRule(listOf(f, "rules"), "secret")
	if !ok {
		t.Fatalf("secret input box rules missing secret: %v", f["rules"])
	}
	if secretRule["role"] != "behavior" || secretRule["widget"] != "password" {
		t.Fatalf("secret rule role/widget = %v/%v, want behavior/password", secretRule["role"], secretRule["widget"])
	}
	if secretRule["hint"] != "已设置，留空不修改" {
		t.Fatalf("secret rule hint = %v, want 已设置，留空不修改", secretRule["hint"])
	}
}

// TestConfigSchemaMasksSecretDefault 钉住 secret 键的非空默认值同样不回传：
// Default 字段自身就可能是明文密钥，掩码 value 还不够。
func TestConfigSchemaMasksSecretDefault(t *testing.T) {
	// 独立于 TestListConfigMasksSecret 的组/键，避免与 testsecret_key 注册冲突。
	schema.RegisterBlock(schema.ConfigBlockDescriptor{
		ID:    "testsecretdef_group",
		Title: "测试密钥默认值",
		Order: 100,
		InputBoxDescriptorList: []*schema.InputBoxDescriptor{
			{
				Key:         "testsecretdef_key",
				Label:       "测试密钥默认值",
				Description: "带默认值的密钥",
				ValueType:   schema.ValueString,
				Container:   schema.ContainerSingle,
				Default:     "sk-default-secret",
				Constraint: schema.InputBoxConstraintDescriptor{
					StringConstraint: &schema.StringConstraintDescriptor{Secret: true},
				},
			},
		},
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
	blk, ok := findBlock(listOf(out, "blocks"), "testsecretdef_group")
	if !ok {
		t.Fatalf("schema missing testsecretdef_group: %v", out)
	}
	f, ok := findInputBox(listOf(blk, "input_boxes"), "testsecretdef_key")
	if !ok {
		t.Fatalf("schema missing testsecretdef_key: %v", blk)
	}
	if f["default"] != "" {
		t.Fatalf("schema leaked secret default: %v", f["default"])
	}
	if f["value"] != "" {
		t.Fatalf("schema leaked secret value: %v", f["value"])
	}

	secretRule, ok := findRule(listOf(f, "rules"), "secret")
	if !ok {
		t.Fatalf("secret input box rules missing secret: %v", f["rules"])
	}
	if secretRule["role"] != "behavior" || secretRule["widget"] != "password" ||
		secretRule["hint"] != "已设置，留空不修改" {
		t.Fatalf("secret rule role/widget/hint = %v/%v/%v, want behavior/password/已设置，留空不修改",
			secretRule["role"], secretRule["widget"], secretRule["hint"])
	}
}
