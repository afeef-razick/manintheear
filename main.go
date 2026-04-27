package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/afeef-razick/manintheear/internal/audio"
	"github.com/afeef-razick/manintheear/internal/config"
	"github.com/afeef-razick/manintheear/internal/llm"
	"github.com/afeef-razick/manintheear/internal/loop"
	"github.com/afeef-razick/manintheear/internal/script"
	"github.com/afeef-razick/manintheear/internal/session"
	"github.com/afeef-razick/manintheear/internal/stt"
	"github.com/afeef-razick/manintheear/internal/tts"
	"github.com/afeef-razick/manintheear/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: manintheear <script.md>")
		os.Exit(1)
	}

	if err := run(os.Args[1]); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(scriptPath string) error {
	s, err := script.Parse(scriptPath)
	if err != nil {
		return fmt.Errorf("parse script: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logFile := redirectLogsToFile(cfg.SessionsDir)
	if logFile != nil {
		defer logFile.Close()
	}

	sess, err := openSession(cfg.SessionsDir, s.TalkID)
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}

	capture, err := audio.New()
	if err != nil {
		return fmt.Errorf("audio init: %w", err)
	}

	sttProvider, err := stt.New(cfg.OpenAIKey)
	if err != nil {
		return fmt.Errorf("stt init: %w", err)
	}

	llmProvider, err := llm.New(cfg.LLMCmd)
	if err != nil {
		return fmt.Errorf("llm init: %w", err)
	}

	ttsProvider, err := tts.New()
	if err != nil {
		return fmt.Errorf("tts init: %w", err)
	}

	l, err := loop.New(s, sttProvider, llmProvider, ttsProvider, sess)
	if err != nil {
		return fmt.Errorf("loop init: %w", err)
	}

	ui, err := tui.New(s, sess.Dir())
	if err != nil {
		return fmt.Errorf("tui init: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2 slots: audio sampler fires every 7s, STT call takes ~1-2s
	audioCh := make(chan []byte, 2)

	// 64 slots: one per LLM cycle at maximum burst rate
	updateCh := make(chan loop.Update, 64)

	var wg sync.WaitGroup

	wg.Add(1)
	go runCapture(ctx, &wg, capture)

	wg.Add(1)
	go runAudioSampler(ctx, &wg, capture, audioCh)

	wg.Add(1)
	go runLoop(ctx, &wg, l, audioCh, updateCh)

	if err := ui.Run(updateCh); err != nil {
		stop()
		wg.Wait()
		if closeErr := capture.Close(); closeErr != nil {
			slog.Warn("audio close failed", "err", closeErr)
		}
		return fmt.Errorf("tui: %w", err)
	}

	stop()
	wg.Wait()
	if err := capture.Close(); err != nil {
		slog.Warn("audio close failed", "err", err)
	}
	return nil
}

func runCapture(ctx context.Context, wg *sync.WaitGroup, c *audio.Capture) {
	defer wg.Done()
	if err := c.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Warn("audio capture stopped", "err", err)
	}
}

func runAudioSampler(ctx context.Context, wg *sync.WaitGroup, c *audio.Capture, out chan<- []byte) {
	defer wg.Done()
	defer close(out)
	ticker := time.NewTicker(7 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wav := c.Drain()
			select {
			case out <- wav:
			default:
				// drop if loop is backed up — next tick will send a fresh window
			}
		}
	}
}

func runLoop(ctx context.Context, wg *sync.WaitGroup, l *loop.Loop, audioCh <-chan []byte, updateCh chan<- loop.Update) {
	defer wg.Done()
	defer close(updateCh)
	if err := l.Run(ctx, audioCh, updateCh); err != nil && !errors.Is(err, context.Canceled) {
		slog.Warn("loop exited unexpectedly", "err", err)
	}
}

func redirectLogsToFile(sessionsDir string) *os.File {
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		slog.Warn("could not create sessions dir for logging", "err", err)
		return nil
	}
	lf, err := os.OpenFile(sessionsDir+"/manintheear.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Warn("could not open log file", "err", err)
		return nil
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(lf, nil)))
	return lf
}

func openSession(baseDir string, talkID string) (*session.Session, error) {
	existing, err := session.FindLatest(baseDir, talkID)
	if err != nil {
		return nil, err
	}
	if existing != "" {
		sess, err := session.Resume(existing)
		if err == nil {
			return sess, nil
		}
		slog.Warn("could not resume session, starting fresh", "err", err)
	}
	return session.New(baseDir, talkID)
}
