package summarize

import (
	"context"
	"fmt"
)

func (c *Client) GetTranscript(ctx context.Context, videoURL string) (string, error) {
	args := []string{
		videoURL,
		"--extract",
		"--youtube", "auto",
	}

	result, err := c.run(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("get transcript: %w", err)
	}

	return result, nil
}
