package llm

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

var logger = slog.Default().With("package", "llm")

// The command is split on whitespace; the prompt is appended as the final argument.
// If the last token is "-", the prompt is written to stdin instead (e.g. "codex exec -").
type CLIProvider struct {
	cmd   string
	args  []string
	stdin bool // true when last arg was "-"
}

func New(fullCmd string) (*CLIProvider, error) {
	parts := strings.Fields(fullCmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("llm: empty command")
	}
	if _, err := exec.LookPath(parts[0]); err != nil {
		return nil, fmt.Errorf("llm: command %q not found: %w", parts[0], err)
	}
	p := &CLIProvider{cmd: parts[0], args: parts[1:]}
	if len(p.args) > 0 && p.args[len(p.args)-1] == "-" {
		p.stdin = true
		p.args = p.args[:len(p.args)-1]
	}
	return p, nil
}

func (c *CLIProvider) Decide(ctx context.Context, prompt string) (string, error) {
	var args []string
	if c.stdin {
		args = c.args
	} else {
		args = make([]string, len(c.args)+1)
		copy(args, c.args)
		args[len(c.args)] = prompt
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, c.cmd, args...)
	if c.stdin {
		cmd.Stdin = strings.NewReader(prompt)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		se := strings.TrimSpace(stderr.String())
		logger.Warn("llm cli error", "err", err, "stderr", se, "elapsed_ms", elapsed)
		if se != "" {
			return "", fmt.Errorf("llm decide: %w: %s", err, se)
		}
		return "", fmt.Errorf("llm decide: %w", err)
	}

	logger.LogAttrs(ctx, slog.LevelDebug, "llm decide",
		slog.Int64("elapsed_ms", elapsed),
	)
	return strings.TrimSpace(string(out)), nil
}
