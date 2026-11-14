package ai

import (
	"fmt"
	"log"
	"math"
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

// Default values as stored in the config table. bootstrap seeds a fresh
// database from these and Settings falls back to them, so the two can never
// disagree; an empty OllamaModel means "not configured" and the recognition
// path reports it.
const (
	DefaultOllamaURL     = "http://127.0.0.1:11434"
	DefaultOllamaModel   = ""
	DefaultFrameInterval = "1"
	DefaultDiffThreshold = "10"
	DefaultMaxFrames     = "10"
)

// Bounds of the numeric keys, both enforced through intSetting's logged fallback.
//
// maxFramesLimit caps ai_max_frames because Extract hands ffmpeg
// -frames:v maxFrames*10 and then reads every JPEG it writes back into process
// memory: an unbounded setting one typo away (1000 -> ~10,000 frames) would
// OOM-kill the WebUI. 200 sits far above the default of 10, so no legitimate
// workflow reaches it.
//
// noUpperBound disables the cap for a key that has none: a parsed int can never
// exceed it, so such keys keep exactly the previous semantics.
const (
	maxFramesLimit = 200
	noUpperBound   = math.MaxInt
)

// Typed forms of the string defaults above, derived rather than restated.
var (
	defaultFrameInterval = mustParseFloat(DefaultFrameInterval)
	defaultDiffThreshold = mustParseInt(DefaultDiffThreshold)
	defaultMaxFrames     = mustParseInt(DefaultMaxFrames)
)

func mustParseInt(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil {
		panic(fmt.Sprintf("ai: default config value %q is not an integer", raw))
	}
	return value
}

func mustParseFloat(raw string) float64 {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		panic(fmt.Sprintf("ai: default config value %q is not a number", raw))
	}
	return value
}

// Settings is the user-facing configuration snapshot of one recognition run.
type Settings struct {
	OllamaURL     string
	OllamaModel   string
	FrameInterval float64
	DiffThreshold int
	MaxFrames     int
}

// Settings reads every key independently and never fails: a missing key, a read
// error or an unparseable/non-positive/out-of-range number falls back to its
// default. A zero diff threshold falls back too, since 0 would keep every frame
// and defeat the dedupe filter.
func (s *Service) Settings() Settings {
	return Settings{
		OllamaURL:     s.stringSetting(KeyOllamaURL, DefaultOllamaURL),
		OllamaModel:   s.stringSetting(KeyOllamaModel, DefaultOllamaModel),
		FrameInterval: s.floatSetting(KeyFrameInterval, defaultFrameInterval),
		DiffThreshold: s.intSetting(KeyDiffThreshold, defaultDiffThreshold, noUpperBound),
		MaxFrames:     s.intSetting(KeyMaxFrames, defaultMaxFrames, maxFramesLimit),
	}
}

func (s *Service) stringSetting(key string, def string) string {
	cfg, err := s.configSvc.Get(key)
	if err != nil {
		return def
	}
	return cfg.ConfigValue
}

// intSetting reads a positive int that must not exceed upperBound. Anything
// else - missing key, read error, unparseable text, non-positive or out-of-range
// number - is unusable and takes the same logged fallback to def, so no caller
// has to remember which out-of-range direction clamps and which falls back.
func (s *Service) intSetting(key string, def int, upperBound int) int {
	cfg, err := s.configSvc.Get(key)
	if err != nil {
		return def
	}
	value, err := strconv.Atoi(cfg.ConfigValue)
	if err != nil || value <= 0 || value > upperBound {
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
