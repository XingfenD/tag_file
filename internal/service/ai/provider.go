package ai

import "fmt"

// providerLabel names the provider in user-facing config errors.
func providerLabel(provider string) string {
	switch provider {
	case ProviderOpenAI:
		return "OpenAI"
	case ProviderAnthropic:
		return "Anthropic"
	default:
		return "Ollama"
	}
}

// checkProviderModel reports the active provider's missing model name as the
// config error it is. It runs BEFORE the Client seam so a missing model is
// never papered over by an injected client (TestWorker_EmptyModelConfigFails).
func checkProviderModel(settings Settings) error {
	var model string
	switch settings.Provider {
	case ProviderOpenAI:
		model = settings.OpenAIModel
	case ProviderAnthropic:
		model = settings.AnthropicModel
	default:
		model = settings.OllamaModel
	}
	if model == "" {
		return fmt.Errorf("未配置 %s 模型名, 请先在配置页填写", providerLabel(settings.Provider))
	}
	return nil
}

// newRefineClient builds the client for the configured refine endpoint, the
// mirror of newVisionClient for the ai_refine_* group: called per task, so a
// config edit takes effect on the next run without a restart, and the default
// branch is ollama (the same logged fallback the read-side gives a garbage
// ai_refine_provider). It always passes refineMode=true — this client exists
// only for stage two, whose text-only calls keep the JSON response_format.
func newRefineClient(settings Settings) VisionClient {
	switch settings.RefineProvider {
	case ProviderOpenAI:
		return NewOpenAIClient(settings.RefineOpenAIURL, settings.RefineOpenAIModel, settings.RefineOpenAIAPIKey, true)
	case ProviderAnthropic:
		return NewAnthropicClient(settings.RefineAnthropicURL, settings.RefineAnthropicModel, settings.RefineAnthropicAPIKey)
	default:
		return NewOllamaClient(settings.RefineOllamaURL, settings.RefineOllamaModel)
	}
}

// newVisionClient builds the client for the configured endpoint. Called per
// task, so config edits take effect on the next file without a restart. The
// default branch is ollama — also the landing spot of the read-side fallback
// for a garbage ai_provider value.
func newVisionClient(settings Settings) VisionClient {
	switch settings.Provider {
	case ProviderOpenAI:
		return NewOpenAIClient(settings.OpenAIURL, settings.OpenAIModel, settings.OpenAIAPIKey, refineEnabled(settings))
	case ProviderAnthropic:
		return NewAnthropicClient(settings.AnthropicURL, settings.AnthropicModel, settings.AnthropicAPIKey)
	default:
		return NewOllamaClient(settings.OllamaURL, settings.OllamaModel)
	}
}
