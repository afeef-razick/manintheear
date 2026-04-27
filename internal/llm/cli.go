package llm

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

var logger = slog.Default().With("package", "llm")

// The command is split on whitespace; the prompt is appended as the final argument.
type CLIProvider struct {
	cmd  string
	args []string
}

func New(fullCmd string) (*CLIProvider, error) {
	parts := strings.Fields(fullCmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("llm: empty command")
	}
	if _, err := exec.LookPath(parts[0]); err != nil {
		return nil, fmt.Errorf("llm: command %q not found: %w", parts[0], err)
	}
	return &CLIProvider{cmd: parts[0], args: parts[1:]}, nil
}

func (c *CLIProvider) Decide(ctx context.Context, prompt string) (string, error) {
	args := make([]string, len(c.args)+1)
	copy(args, c.args)
	args[len(c.args)] = prompt

	start := time.Now()
	cmd := exec.CommandContext(ctx, c.cmd, args...)
	cmd.Stderr = io.Discard
	out, err := cmd.Output()
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		logger.Warn("llm cli error", "err", err, "elapsed_ms", elapsed)
		return "", fmt.Errorf("llm decide: %w", err)
	}

	logger.LogAttrs(ctx, slog.LevelDebug, "llm decide",
		slog.Int64("elapsed_ms", elapsed),
	)
	return strings.TrimSpace(string(out)), nil
}
