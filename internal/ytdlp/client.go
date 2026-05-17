package ytdlp

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog"
)

type Client struct {
	Binary string
	Logger zerolog.Logger
}

func New(binary string, logger zerolog.Logger) *Client {
	return &Client{Binary: binary, Logger: logger}
}

func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.Binary, args...)
	c.Logger.Debug().Str("cmd", cmd.String()).Msg("running yt-dlp")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			c.Logger.Error().Str("stderr", errMsg).Msg("yt-dlp failed")
			return nil, fmt.Errorf("yt-dlp: %s", errMsg)
		}
		return nil, fmt.Errorf("yt-dlp: %w", err)
	}
	return out, nil
}
