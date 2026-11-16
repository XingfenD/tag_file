package config_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XingfenD/tag_file/internal/model"
	_ "github.com/XingfenD/tag_file/internal/service/ai"
	configSvc "github.com/XingfenD/tag_file/internal/service/config"
	"github.com/XingfenD/tag_file/pkg/storage"
)

// storage 是进程级单例，测试之间不得并行。
func newTestService(t *testing.T) *configSvc.Service {
	t.Helper()
	if err := storage.InitDB(filepath.Join(t.TempDir(), "test.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := storage.GetDB().AutoMigrate(&model.Config{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return configSvc.NewService()
}

func rowExists(t *testing.T, key string) bool {
	t.Helper()
	var count int64
	if err := storage.GetDB().Model(&model.Config{}).Where("config_name = ?", key).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", key, err)
	}
	return count > 0
}

func rowValue(t *testing.T, key string) string {
	t.Helper()
	var cfg model.Config
	if err := storage.GetDB().Where("config_name = ?", key).First(&cfg).Error; err != nil {
		t.Fatalf("read %s: %v", key, err)
	}
	return cfg.ConfigValue
}

func TestSetValidatesKnownKey(t *testing.T) {
	svc := newTestService(t)

	err := svc.Set("ai_max_frames", "201")
	if err == nil {
		t.Fatalf("Set(ai_max_frames, 201) 应返回错误")
	}
	if !strings.Contains(err.Error(), "ai_max_frames") {
		t.Fatalf("错误应包含键名，实际为 %q", err.Error())
	}
	if rowExists(t, "ai_max_frames") {
		t.Fatalf("非法值不应写入库")
	}

	if err := svc.Set("ai_max_frames", "200"); err != nil {
		t.Fatalf("Set(ai_max_frames, 200) 应通过: %v", err)
	}
	if err := svc.Set("ai_frame_interval", "0"); err == nil {
		t.Fatalf("Set(ai_frame_interval, 0) 应返回错误")
	}
	if err := svc.Set("ai_frame_interval", "0.05"); err != nil {
		t.Fatalf("Set(ai_frame_interval, 0.05) 应通过: %v", err)
	}
	if err := svc.Set("ollama_url", "no-scheme"); err == nil {
		t.Fatalf("Set(ollama_url, no-scheme) 应返回错误")
	}
	// 空模型名是「未配置」的合法取值（Review Focus 1）。
	if err := svc.Set("ollama_model", ""); err != nil {
		t.Fatalf("Set(ollama_model, \"\") 应通过: %v", err)
	}
}

func TestSetUnknownKeyPassthrough(t *testing.T) {
	svc := newTestService(t)

	if err := svc.Set("my_custom_key", "任意值"); err != nil {
		t.Fatalf("未知键应原样放行: %v", err)
	}
	if err := svc.Set("", "x"); err == nil {
		t.Fatalf("空键应返回错误")
	}
}

func TestUpdateValidatesKnownKey(t *testing.T) {
	svc := newTestService(t)

	if err := svc.Set("ai_max_frames", "10"); err != nil {
		t.Fatalf("seed ai_max_frames: %v", err)
	}
	if err := svc.Update("ai_max_frames", "abc"); err == nil {
		t.Fatalf("Update(ai_max_frames, abc) 应返回错误")
	}
	if got := rowValue(t, "ai_max_frames"); got != "10" {
		t.Fatalf("非法 Update 后库值应仍为 10，实际为 %q", got)
	}
}

func TestDeleteGuardsKnownKey(t *testing.T) {
	svc := newTestService(t)

	if err := svc.Set("ollama_url", "http://127.0.0.1:11434"); err != nil {
		t.Fatalf("seed ollama_url: %v", err)
	}
	err := svc.Delete("ollama_url")
	if err == nil {
		t.Fatalf("Delete(ollama_url) 应返回错误")
	}
	if !strings.Contains(err.Error(), "恢复默认") {
		t.Fatalf("错误应提示使用恢复默认，实际为 %q", err.Error())
	}
	if !rowExists(t, "ollama_url") {
		t.Fatalf("被禁删的已知键行应仍在库中")
	}

	if err := svc.Set("my_custom_key", "v"); err != nil {
		t.Fatalf("seed my_custom_key: %v", err)
	}
	if err := svc.Delete("my_custom_key"); err != nil {
		t.Fatalf("Delete(my_custom_key) 应通过: %v", err)
	}
	if rowExists(t, "my_custom_key") {
		t.Fatalf("未知键删除后行应消失")
	}
}

func aiGroupLegal() map[string]string {
	return map[string]string{
		"ollama_url":              "http://127.0.0.1:11434",
		"ollama_model":            "llava",
		"ai_frame_interval":       "1",
		"ai_frame_diff_threshold": "10",
		"ai_max_frames":           "10",
	}
}

func TestSetGroupStrictAllOrNothing(t *testing.T) {
	svc := newTestService(t)

	if err := svc.SetGroup("ai", aiGroupLegal()); err != nil {
		t.Fatalf("首次 SetGroup(ai) 应通过: %v", err)
	}

	bad := aiGroupLegal()
	bad["ai_max_frames"] = "201"
	bad["ollama_url"] = "http://127.0.0.1:9999"
	bad["ollama_model"] = "other"
	bad["ai_frame_interval"] = "2"
	bad["ai_frame_diff_threshold"] = "20"

	err := svc.SetGroup("ai", bad)
	if err == nil {
		t.Fatalf("含非法值的 SetGroup 应返回错误")
	}
	var verrs configSvc.ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("错误应为 ValidationErrors，实际为 %T: %v", err, err)
	}
	if _, ok := verrs["ai_max_frames"]; !ok {
		t.Fatalf("ValidationErrors 应含 ai_max_frames，实际为 %v", verrs)
	}
	if got := rowValue(t, "ollama_url"); got != "http://127.0.0.1:11434" {
		t.Fatalf("整组失败后 ollama_url 应保持首次值，实际为 %q", got)
	}
	if got := rowValue(t, "ai_frame_interval"); got != "1" {
		t.Fatalf("整组失败后 ai_frame_interval 应保持首次值，实际为 %q", got)
	}
}

func TestSetGroupMissingAndForeignKeys(t *testing.T) {
	svc := newTestService(t)

	missing := aiGroupLegal()
	delete(missing, "ai_max_frames")
	err := svc.SetGroup("ai", missing)
	var verrs configSvc.ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("缺字段应返回 ValidationErrors，实际为 %T: %v", err, err)
	}
	if verrs["ai_max_frames"] != "缺少字段" {
		t.Fatalf("ai_max_frames 应为「缺少字段」，实际为 %q", verrs["ai_max_frames"])
	}

	foreign := aiGroupLegal()
	foreign["scan_exclude"] = "[]"
	err = svc.SetGroup("ai", foreign)
	if !errors.As(err, &verrs) {
		t.Fatalf("多余键应返回 ValidationErrors，实际为 %T: %v", err, err)
	}
	if verrs["scan_exclude"] != "不属于该分组" {
		t.Fatalf("scan_exclude 应为「不属于该分组」，实际为 %q", verrs["scan_exclude"])
	}

	err = svc.SetGroup("no_such_group", aiGroupLegal())
	if err == nil {
		t.Fatalf("未知分组应返回错误")
	}
	if errors.As(err, &verrs) {
		t.Fatalf("未知分组应为普通错误，不应是 ValidationErrors")
	}
	if !strings.Contains(err.Error(), "no_such_group") {
		t.Fatalf("错误应包含分组名，实际为 %q", err.Error())
	}
}

func TestSetGroupAcceptsEmptyModel(t *testing.T) {
	svc := newTestService(t)

	values := aiGroupLegal()
	values["ollama_model"] = ""
	if err := svc.SetGroup("ai", values); err != nil {
		t.Fatalf("空模型名应通过: %v", err)
	}
}

func TestListUnknownFiltersKnownKeys(t *testing.T) {
	svc := newTestService(t)

	if err := svc.Set("ollama_url", "http://127.0.0.1:11434"); err != nil {
		t.Fatalf("seed ollama_url: %v", err)
	}
	if err := svc.Set("custom_a", "1"); err != nil {
		t.Fatalf("seed custom_a: %v", err)
	}
	if err := svc.Set("custom_b", "2"); err != nil {
		t.Fatalf("seed custom_b: %v", err)
	}

	list, total, err := svc.ListUnknown(0, 10)
	if err != nil {
		t.Fatalf("ListUnknown: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("ListUnknown 应返回 2 条，实际 total=%d len=%d", total, len(list))
	}
	if list[0].ConfigName != "custom_a" || list[1].ConfigName != "custom_b" {
		t.Fatalf("ListUnknown 应只含 custom_a/custom_b，实际为 %q %q", list[0].ConfigName, list[1].ConfigName)
	}

	list, total, err = svc.SearchUnknown("custom", 0, 10)
	if err != nil {
		t.Fatalf("SearchUnknown: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("SearchUnknown 应返回 2 条，实际 total=%d len=%d", total, len(list))
	}

	list, total, err = svc.ListUnknown(1, 1)
	if err != nil {
		t.Fatalf("ListUnknown(1,1): %v", err)
	}
	if total != 2 || len(list) != 1 {
		t.Fatalf("ListUnknown(1,1) 应返回 1 条且 total=2，实际 total=%d len=%d", total, len(list))
	}
}
