package config_test

import (
	"testing"

	"github.com/afeef-razick/manintheear/internal/config"
)

func TestLoad_DefaultProviders(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("STT_PROVIDER", "")
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("TTS_PROVIDER", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.STTProvider != "whisper" {
		t.Errorf("STTProvider = %q, want %q", cfg.STTProvider, "whisper")
	}
	if cfg.LLMProvider != "ai_cli" {
		t.Errorf("LLMProvider = %q, want %q", cfg.LLMProvider, "ai_cli")
	}
	if cfg.TTSProvider != "say" {
		t.Errorf("TTSProvider = %q, want %q", cfg.TTSProvider, "say")
	}
}

func TestLoad_MissingOpenAIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("STT_PROVIDER", "whisper")

	_, err := config.Load()
	if err == nil {
		t.Error("Load() expected error when OPENAI_API_KEY is absent, got nil")
	}
}

func TestLoad_DefaultLLMCmd(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	// do not set AI_CLI_CMD so the default path is exercised
	t.Setenv("AI_CLI_CMD", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// envOr treats empty string the same as unset — default kicks in
	if cfg.LLMCmd != "claude -p" {
		t.Errorf("LLMCmd = %q, want %q", cfg.LLMCmd, "claude -p")
	}
}

func TestLoad_CustomSessionsDir(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("SESSIONS_DIR", "/tmp/sessions")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.SessionsDir != "/tmp/sessions" {
		t.Errorf("SessionsDir = %q, want %q", cfg.SessionsDir, "/tmp/sessions")
	}
}
