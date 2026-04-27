package tts

import (
	"context"
	"fmt"
	"os/exec"
)

// follows system default audio output device — no explicit device targeting needed.
type SayProvider struct{}

func New() (*SayProvider, error) {
	if _, err := exec.LookPath("say"); err != nil {
		return nil, fmt.Errorf("tts: say command not found: %w", err)
	}
	return &SayProvider{}, nil
}

func (s *SayProvider) Speak(ctx context.Context, text string) error {
	if err := exec.CommandContext(ctx, "say", text).Run(); err != nil {
		return fmt.Errorf("tts speak: %w", err)
	}
	return nil
}
