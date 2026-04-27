package tui_test

import (
	"testing"

	"github.com/afeef-razick/manintheear/internal/script"
	"github.com/afeef-razick/manintheear/internal/tui"
)

func TestNew_NilScriptReturnsError(t *testing.T) {
	_, err := tui.New(nil, "/tmp/session")
	if err == nil {
		t.Error("New() expected error for nil script")
	}
}

func TestNew_ValidScript(t *testing.T) {
	s, err := script.Parse("../../internal/script/testdata/example_talk.md")
	if err != nil {
		t.Fatalf("script.Parse() error: %v", err)
	}
	_, err = tui.New(s, "/tmp/session")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
}
