package ai

import (
	"log"
	"strconv"
)

// Config keys of the vision tagging feature.
const (
	KeyOllamaURL     = "ollama_url"
	KeyOllamaModel   = "ollama_model"
	KeyFrameInterval = "ai_frame_interval"
	KeyDiffThreshold = "ai_frame_diff_threshold"
	KeyMaxFrames     = "ai_max_frames"
)

// Fallback values used when a key is missing or holds an unusable value. An
// empty OllamaModel means "not configured" and the recognition path reports it.
const (
	defaultOllamaURL     = "http://127.0.0.1:11434"
	defaultOllamaModel   = ""
	defaultFrameInterval = 1.0
	defaultDiffThreshold = 10
	defaultMaxFrames     = 10
)

// Settings is the user-facing configuration snapshot of one recognition run.
type Settings struct {
	OllamaURL     string
	OllamaModel   string
	FrameInterval float64
	DiffThreshold int
	MaxFrames     int
}

// Settings reads every key independently and never fails: a missing key, a read
// error or an unparseable/non-positive number falls back to its default. A zero
// diff threshold falls back too, since 0 would keep every frame and defeat the
// dedupe filter.
func (s *Service) Settings() Settings {
	return Settings{
		OllamaURL:     s.stringSetting(KeyOllamaURL, defaultOllamaURL),
		OllamaModel:   s.stringSetting(KeyOllamaModel, defaultOllamaModel),
		FrameInterval: s.floatSetting(KeyFrameInterval, defaultFrameInterval),
		DiffThreshold: s.intSetting(KeyDiffThreshold, defaultDiffThreshold),
		MaxFrames:     s.intSetting(KeyMaxFrames, defaultMaxFrames),
	}
}

func (s *Service) stringSetting(key string, def string) string {
	cfg, err := s.configSvc.Get(key)
	if err != nil {
		return def
	}
	return cfg.ConfigValue
}

func (s *Service) intSetting(key string, def int) int {
	cfg, err := s.configSvc.Get(key)
	if err != nil {
		return def
	}
	value, err := strconv.Atoi(cfg.ConfigValue)
	if err != nil || value <= 0 {
		log.Printf("[ai] config %s = %q, fallback to %d", key, cfg.ConfigValue, def)
		return def
	}
	return value
}

func (s *Service) floatSetting(key string, def float64) float64 {
	cfg, err := s.configSvc.Get(key)
	if err != nil {
		return def
	}
	value, err := strconv.ParseFloat(cfg.ConfigValue, 64)
	if err != nil || value <= 0 {
		log.Printf("[ai] config %s = %q, fallback to %v", key, cfg.ConfigValue, def)
		return def
	}
	return value
}
