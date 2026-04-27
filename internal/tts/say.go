package tts

import (
	"context"
	"fmt"
	"os/exec"
)

// SayProvider speaks text using the macOS say command.
// Output device follows the system default — no explicit device targeting needed.
type SayProvider struct{}

// New returns a SayProvider.
func New() (*SayProvider, error) {
	if _, err := exec.LookPath("say"); err != nil {
		return nil, fmt.Errorf("tts: say command not found: %w", err)
	}
	return &SayProvider{}, nil
}

// Speak runs say with the given text. Blocks until speech is complete or ctx is cancelled.
func (s *SayProvider) Speak(ctx context.Context, text string) error {
	if err := exec.CommandContext(ctx, "say", text).Run(); err != nil {
		return fmt.Errorf("tts speak: %w", err)
	}
	return nil
}
