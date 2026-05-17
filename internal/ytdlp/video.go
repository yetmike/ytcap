package ytdlp

import (
	"context"
	"encoding/json"
	"fmt"
)

func (c *Client) FetchVideo(ctx context.Context, videoURL string) (*Video, error) {
	args := []string{
		videoURL,
		"--dump-json",
		"--no-playlist",
		"--no-warnings",
		"--quiet",
	}

	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("yt-dlp video: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("yt-dlp video parse: %w", err)
	}

	v := &Video{
		ID:          getString(raw, "id"),
		Title:       getString(raw, "title"),
		URL:         getString(raw, "webpage_url"),
		Channel:     getString(raw, "channel"),
		ChannelURL:  getString(raw, "channel_url"),
		UploadDate:  getString(raw, "upload_date"),
		Description: getString(raw, "description"),
		Thumbnail:   getString(raw, "thumbnail"),
	}

	if d, ok := raw["duration"].(float64); ok {
		v.Duration = d
		v.DurationStr = formatDuration(d)
	}
	if vc, ok := raw["view_count"].(float64); ok {
		v.ViewCount = int64(vc)
	}

	return v, nil
}
