package tts_test

import (
	"context"
	"testing"

	"github.com/afeef-razick/manintheear/internal/tts"
)

func TestNew_FindsSay(t *testing.T) {
	_, err := tts.New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
}

func TestSpeak_EmptyString(t *testing.T) {
	p, err := tts.New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if err := p.Speak(context.Background(), ""); err != nil {
		t.Errorf("Speak() with empty string error: %v", err)
	}
}

func TestSpeak_CancelledContext(t *testing.T) {
	p, err := tts.New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = p.Speak(ctx, "this should not be spoken")
	if err == nil {
		t.Error("Speak() with cancelled context should return error")
	}
}
