package config_test

import (
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
