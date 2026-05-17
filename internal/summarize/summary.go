package summarize

import (
	"context"
	"fmt"
)

func (c *Client) GetSummary(ctx context.Context, videoURL string) (string, error) {
	args := []string{
		videoURL,
		"--length", c.Length,
		"--youtube", "auto",
	}

	result, err := c.run(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("get summary: %w", err)
	}

	return result, nil
}
