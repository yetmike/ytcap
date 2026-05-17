package summarize

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type ChatMessage struct {
	Role    string // "user" or "ai"
	Content string
}

func (c *Client) Ask(ctx context.Context, transcript string, history []ChatMessage, question string) (string, error) {
	var content string
	content += "=== VIDEO TRANSCRIPT ===\n\n"
	content += transcript
	content += "\n\n=== CONVERSATION HISTORY ===\n\n"

	for _, msg := range history {
		switch msg.Role {
		case "user":
			content += "User: " + msg.Content + "\n\n"
		case "ai":
			content += "AI: " + msg.Content + "\n\n"
		}
	}

	content += "User: " + question + "\n"

	// Use stdin ("-") instead of a temp file so that CLI backends like
	// gemini receive the content correctly (file path mode hangs with
	// gemini's --output-format json).
	args := []string{
		"-",
		"--length", "medium",
	}
	if c.CLI != "" && c.CLI != "auto" {
		args = append(args, "--cli", c.CLI)
	}

	cmd := exec.CommandContext(ctx, c.Binary, args...)
	cmd.Stdin = strings.NewReader(content)
	c.Logger.Debug().Str("cmd", cmd.String()).Msg("running summarize (chat)")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			c.Logger.Error().Str("stderr", errMsg).Msg("summarize chat failed")
			return "", fmt.Errorf("chat ask: %s", errMsg)
		}
		return "", fmt.Errorf("chat ask: %w", err)
	}

	return string(out), nil
}
