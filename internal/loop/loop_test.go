package loop

import (
	"testing"
	"time"
)

// --- trigger ---

func TestTrigger_SixSecAndTwentyWords(t *testing.T) {
	tr := &trigger{lastFire: time.Now().Add(-7 * time.Second), wordsSince: 20}
	if !tr.shouldFire() {
		t.Error("should fire: elapsed>=6s AND words>=20")
	}
}

func TestTrigger_EighteenSecCap(t *testing.T) {
	tr := &trigger{lastFire: time.Now().Add(-19 * time.Second), wordsSince: 0}
	if !tr.shouldFire() {
		t.Error("should fire: elapsed>=18s regardless of words")
	}
}

func TestTrigger_SixtyWordCap(t *testing.T) {
	tr := &trigger{lastFire: time.Now(), wordsSince: 60}
	if !tr.shouldFire() {
		t.Error("should fire: words>=60 regardless of elapsed")
	}
}

func TestTrigger_NoFireWhenBelowThresholds(t *testing.T) {
	tr := &trigger{lastFire: time.Now().Add(-5 * time.Second), wordsSince: 10}
	if tr.shouldFire() {
		t.Error("should not fire: elapsed<6s AND words<20")
	}
}

func TestTrigger_Reset(t *testing.T) {
	tr := &trigger{lastFire: time.Now().Add(-20 * time.Second), wordsSince: 100}
	tr.reset()
	if tr.shouldFire() {
		t.Error("should not fire immediately after reset")
	}
}

// --- whisper manager ---

func TestWhisperManager_AllowsFirstSpeak(t *testing.T) {
	wm := newWhisperManager()
	if !wm.canSpeak("", "tell the joke") {
		t.Error("should allow first speak")
	}
}

func TestWhisperManager_RateCap(t *testing.T) {
	wm := newWhisperManager()
	wm.record("", "tell the joke")
	// immediately after, rate cap should block
	if wm.canSpeak("", "something else") {
		t.Error("rate cap should block within 15s")
	}
}

func TestWhisperManager_MaxAttempts(t *testing.T) {
	wm := newWhisperManager()
	// record twice to hit max
	wm.lastSpoken = time.Now().Add(-20 * time.Second)
	wm.record("", "tell the joke")
	wm.lastSpoken = time.Now().Add(-20 * time.Second)
	wm.record("", "tell the joke")
	wm.lastSpoken = time.Now().Add(-20 * time.Second)
	if wm.canSpeak("", "tell the joke") {
		t.Error("should suppress after 2 attempts")
	}
}

func TestWhisperManager_ResolveAddsSTILL(t *testing.T) {
	wm := newWhisperManager()
	wm.record("", "tell the joke")
	got := wm.resolve("", "tell the joke")
	if got != "again: tell the joke" {
		t.Errorf("resolve() = %q, want %q", got, "again: tell the joke")
	}
}

func TestWhisperManager_ResolveFirstAttempt(t *testing.T) {
	wm := newWhisperManager()
	got := wm.resolve("", "tell the joke")
	if got != "tell the joke" {
		t.Errorf("resolve() = %q, want %q", got, "tell the joke")
	}
}

// --- parse response ---

func TestParseResponse_HappyPath(t *testing.T) {
	raw := `{"points_covered":["1_hook"],"points_remaining":["2_problem"],"whisper":"tell the joke","urgency":"high"}`
	resp, err := parseResponse(raw)
	if err != nil {
		t.Fatalf("parseResponse() error: %v", err)
	}
	if len(resp.PointsCovered) != 1 || resp.PointsCovered[0] != "1_hook" {
		t.Errorf("PointsCovered = %v, want [1_hook]", resp.PointsCovered)
	}
	if resp.Whisper == nil || *resp.Whisper != "tell the joke" {
		t.Errorf("Whisper = %v, want 'tell the joke'", resp.Whisper)
	}
}

func TestParseResponse_NullWhisper(t *testing.T) {
	raw := `{"points_covered":[],"points_remaining":[],"whisper":null,"urgency":"low"}`
	resp, err := parseResponse(raw)
	if err != nil {
		t.Fatalf("parseResponse() error: %v", err)
	}
	if resp.Whisper != nil {
		t.Errorf("Whisper = %v, want nil", resp.Whisper)
	}
}

func TestParseResponse_StripsMarkdownFences(t *testing.T) {
	cases := []string{
		"```json\n{\"points_covered\":[],\"points_remaining\":[],\"whisper\":null,\"urgency\":\"low\"}\n```",
		"```\n{\"points_covered\":[],\"points_remaining\":[],\"whisper\":null,\"urgency\":\"low\"}\n```",
	}
	for _, raw := range cases {
		_, err := parseResponse(raw)
		if err != nil {
			t.Errorf("parseResponse() should strip fences, got error: %v\nfor input: %q", err, raw)
		}
	}
}

func TestParseResponse_InvalidJSON(t *testing.T) {
	_, err := parseResponse("not json")
	if err == nil {
		t.Error("parseResponse() expected error for invalid JSON")
	}
}

// --- word count ---

func TestCountWords(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"hello world", 2},
		{"", 0},
		{"  spaces   everywhere  ", 2},
		{"one", 1},
	}
	for _, tc := range cases {
		got := countWords(tc.input)
		if got != tc.want {
			t.Errorf("countWords(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
