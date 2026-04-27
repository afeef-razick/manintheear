package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

var logger = slog.Default().With("package", "session")

type State struct {
	CurrentPhase   int      `json:"current_phase"`
	BeatsCovered   []string `json:"beats_covered"`
	BeatsRemaining []string `json:"beats_remaining"`
}

type TranscriptEntry struct {
	Timestamp time.Time `json:"ts"`
	Text      string    `json:"text"`
	WordCount int       `json:"word_count"`
}

type WhisperEntry struct {
	Timestamp time.Time `json:"ts"`
	Text      string    `json:"text"`
	Urgency   string    `json:"urgency"`
}

type manifest struct {
	TalkID    string    `json:"talk_id"`
	StartedAt time.Time `json:"started_at"`
}

type Session struct {
	dir    string
	talkID string
}

func New(baseDir string, talkID string) (*Session, error) {
	dir := filepath.Join(baseDir, fmt.Sprintf("%d", time.Now().UnixMilli()))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("session: create dir: %w", err)
	}
	m := manifest{TalkID: talkID, StartedAt: time.Now()}
	if err := writeJSON(filepath.Join(dir, "manifest.json"), m); err != nil {
		return nil, fmt.Errorf("session: write manifest: %w", err)
	}
	logger.Info("session created", "session_id", filepath.Base(dir), "talk_id", talkID)
	return &Session{dir: dir, talkID: talkID}, nil
}

func Resume(dir string) (*Session, error) {
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("session: no manifest in %s: %w", dir, err)
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("session: parse manifest: %w", err)
	}
	logger.Info("session resumed", "session_id", filepath.Base(dir), "talk_id", m.TalkID)
	return &Session{dir: dir, talkID: m.TalkID}, nil
}

// FindLatest returns the most recent session directory for the given talkID,
// or an empty string if none exists.
func FindLatest(baseDir string, talkID string) (string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("session: read dir: %w", err)
	}
	for i := len(entries) - 1; i >= 0; i-- {
		dir := filepath.Join(baseDir, entries[i].Name())
		data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
		if err != nil {
			continue
		}
		var m manifest
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		if m.TalkID == talkID {
			return dir, nil
		}
	}
	return "", nil
}

func (s *Session) LoadState() (*State, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, "state.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("session: read state: %w", err)
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("session: parse state: %w", err)
	}
	return &st, nil
}

func (s *Session) SaveState(st State) error {
	if err := writeJSON(filepath.Join(s.dir, "state.json"), st); err != nil {
		logger.Warn("state write failed", "err", err)
		return fmt.Errorf("session: save state: %w", err)
	}
	return nil
}

func (s *Session) AppendTranscript(entry TranscriptEntry) error {
	if err := appendJSONL(filepath.Join(s.dir, "transcript.jsonl"), entry); err != nil {
		logger.Warn("transcript write failed", "err", err)
		return err
	}
	return nil
}

func (s *Session) AppendWhisper(entry WhisperEntry) error {
	if err := appendJSONL(filepath.Join(s.dir, "whispers.jsonl"), entry); err != nil {
		logger.Warn("whisper write failed", "err", err)
		return err
	}
	return nil
}

func (s *Session) Dir() string { return s.dir }

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func appendJSONL(path string, v any) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(v)
}
