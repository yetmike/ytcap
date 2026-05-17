package summarize

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog"
)

type Client struct {
	Binary   string
	Length   string
	Language string
	CLI      string
	Logger   zerolog.Logger
}

func New(binary, length, language, cli string, logger zerolog.Logger) *Client {
	return &Client{
		Binary:   binary,
		Length:   length,
		Language: language,
		CLI:      cli,
		Logger:   logger,
	}
}

func (c *Client) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, c.Binary, args...)
	c.Logger.Debug().Str("cmd", cmd.String()).Msg("running summarize")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			c.Logger.Error().Str("stderr", errMsg).Msg("summarize failed")
			return "", fmt.Errorf("summarize: %s", errMsg)
		}
		return "", fmt.Errorf("summarize: %w", err)
	}
	return string(out), nil
}
