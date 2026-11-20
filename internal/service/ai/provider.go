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
