package loop

import (
	"context"
	"time"

	"github.com/afeef-razick/manintheear/internal/session"
)

// STTProvider transcribes a WAV audio payload to text.
type STTProvider interface {
	Transcribe(ctx context.Context, audio []byte) (string, error)
}

// LLMProvider sends a prompt to the AI CLI and returns the raw response.
type LLMProvider interface {
	Decide(ctx context.Context, prompt string) (string, error)
}

// TTSProvider speaks a short text phrase aloud.
type TTSProvider interface {
	Speak(ctx context.Context, text string) error
}

// Activity describes what the loop is doing right now.
type Activity string

const (
	ActivityListening    Activity = "listening"
	ActivityTranscribing Activity = "transcribing"
	ActivityDeciding     Activity = "deciding"
	ActivitySpeaking     Activity = "speaking"
)

// Status carries real-time loop internals for the TUI to display.
type Status struct {
	Activity         Activity
	WordsSince       int       // words accumulated since last LLM fire
	LastSTTAt        time.Time // zero = never succeeded
	LastLLMAt        time.Time // zero = never succeeded
	LastErr          string    // most recent error, cleared on next success
	WhisperBlockedMs int64     // ms until rate cap clears; 0 = ready to speak
}

// Update carries the latest loop state to the TUI on each cycle.
type Update struct {
	State          session.State
	LastTranscript string
	Whisper        string
	Urgency        string
	Status         Status
}

type transcriptChunk struct {
	text  string
	at    time.Time
	words int
}

type aiResponse struct {
	State   session.State `json:"state"`
	Whisper *string       `json:"whisper"`
	Urgency string        `json:"urgency"`
}
