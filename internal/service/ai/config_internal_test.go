package ai

import (
	"strings"
	"testing"
)

// refineEnabled 是阶段二的总开关：off 必须为 false（默认态即今天的单段
// 行为），其余枚举值都是某个端点。
func TestRefineEnabled(t *testing.T) {
	if refineEnabled(Settings{RefineProvider: RefineOff}) {
		t.Error("off 不应启用整理（默认状态必须与今天一致）")
	}
	for _, p := range []string{ProviderOllama, ProviderOpenAI, ProviderAnthropic} {
		if !refineEnabled(Settings{RefineProvider: p}) {
			t.Errorf("RefineProvider = %q, want refine enabled", p)
		}
	}
}

// checkRefineModel 镜像 checkProviderModel：off 无模型可要求；已选端点的
// 模型名为空是配置错误，绝不静默回落单段。
func TestCheckRefineModel(t *testing.T) {
	if err := checkRefineModel(Settings{RefineProvider: RefineOff}); err != nil {
		t.Errorf("off = %v, want nil", err)
	}

	err := checkRefineModel(Settings{RefineProvider: ProviderOpenAI})
	if err == nil || !strings.Contains(err.Error(), "未配置整理端点的 OpenAI 模型名, 请先在配置页填写") {
		t.Fatalf("openai 空模型 = %v, want 含「未配置整理端点的 OpenAI 模型名」的配置错误", err)
	}
	if err := checkRefineModel(Settings{RefineProvider: ProviderOpenAI, RefineOpenAIModel: "qwen3-max"}); err != nil {
		t.Errorf("openai 已填模型 = %v, want nil", err)
	}

	// 各端点只看自己的模型字段。
	if err := checkRefineModel(Settings{RefineProvider: ProviderOllama, RefineOpenAIModel: "x"}); err == nil {
		t.Error("ollama 空整理模型应报错, 哪怕 openai 模型已填")
	}
	if err := checkRefineModel(Settings{RefineProvider: ProviderOllama, RefineOllamaModel: "qwen2.5vl:7b"}); err != nil {
		t.Errorf("ollama 已填模型 = %v, want nil", err)
	}
	if err := checkRefineModel(Settings{RefineProvider: ProviderAnthropic, RefineAnthropicModel: "claude-sonnet-4"}); err != nil {
		t.Errorf("anthropic 已填模型 = %v, want nil", err)
	}
}
