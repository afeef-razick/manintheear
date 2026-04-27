package llm_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/afeef-razick/manintheear/internal/llm"
)

func TestNew_MissingCommand(t *testing.T) {
	_, err := llm.New("no_such_cli_command_xyz")
	if err == nil {
		t.Error("New() expected error for missing command")
	}
}

func TestNew_EmptyCommand(t *testing.T) {
	_, err := llm.New("")
	if err == nil {
		t.Error("New() expected error for empty command")
	}
}

func TestDecide_CapturesStdout(t *testing.T) {
	// write a tiny script that echoes a fixed JSON payload
	want := `{"state":{"current_phase":1,"beats_covered":[],"beats_remaining":[]},"whisper":null,"urgency":"low"}`
	script := fmt.Sprintf("#!/bin/sh\necho '%s'", want)

	dir := t.TempDir()
	bin := filepath.Join(dir, "fakellm")
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake llm: %v", err)
	}

	p, err := llm.New(bin)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	got, err := p.Decide(context.Background(), "any prompt")
	if err != nil {
		t.Fatalf("Decide() error: %v", err)
	}
	if got != want {
		t.Errorf("Decide() = %q, want %q", got, want)
	}
}

func TestDecide_NonZeroExitReturnsError(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "failcli")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 1"), 0o755); err != nil {
		t.Fatalf("write fail cli: %v", err)
	}

	p, err := llm.New(bin)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	_, err = p.Decide(context.Background(), "prompt")
	if err == nil {
		t.Error("Decide() expected error on non-zero exit")
	}
}
