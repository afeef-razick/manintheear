package config

import (
	"errors"
	"os"
)

// Config holds all runtime configuration loaded from the environment.
type Config struct {
	STTProvider string
	LLMProvider string
	TTSProvider string
	OpenAIKey   string
	SessionsDir string
}

// Load reads configuration from environment variables and returns an error
// if any required value is absent.
func Load() (*Config, error) {
	cfg := &Config{
		STTProvider: envOr("STT_PROVIDER", "whisper"),
		LLMProvider: envOr("LLM_PROVIDER", "ai_cli"),
		TTSProvider: envOr("TTS_PROVIDER", "say"),
		OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
		SessionsDir: envOr("SESSIONS_DIR", "./sessions"),
	}

	if cfg.STTProvider == "whisper" && cfg.OpenAIKey == "" {
		return nil, errors.New("OPENAI_API_KEY is required when STT_PROVIDER=whisper")
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
