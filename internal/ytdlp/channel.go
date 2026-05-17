package ytdlp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// normalizeChannelURL ensures the URL points to the /videos tab,
// which is required for --playlist-items pagination to work.
func normalizeChannelURL(url string) string {
	// Strip trailing slash
	url = strings.TrimRight(url, "/")
	// If it already ends with a tab path, replace with /videos
	for _, tab := range []string{"/featured", "/shorts", "/streams", "/playlists", "/community", "/about"} {
		if strings.HasSuffix(url, tab) {
			url = strings.TrimSuffix(url, tab)
			break
		}
	}
	if !strings.HasSuffix(url, "/videos") {
		url += "/videos"
	}
	return url
}

// CountChannelVideos returns the total number of videos on a channel.
func (c *Client) CountChannelVideos(ctx context.Context, channelURL string) (int, error) {
	channelURL = normalizeChannelURL(channelURL)
	args := []string{
		channelURL,
		"--flat-playlist",
		"--print", "id",
		"--quiet",
		"--no-warnings",
	}

	out, err := c.run(ctx, args...)
	if err != nil {
		return 0, fmt.Errorf("yt-dlp count: %w", err)
	}

	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}

// FetchChannelPage fetches videos from a channel with pagination.
// start is 1-based, count is how many to fetch.
func (c *Client) FetchChannelPage(ctx context.Context, channelURL string, start, count int) ([]Video, error) {
	// Ensure we target the /videos tab for reliable pagination
	channelURL = normalizeChannelURL(channelURL)
	end := start + count - 1
	args := []string{
		channelURL,
		"--flat-playlist",
		"--dump-json",
		"--no-warnings",
		"--quiet",
		"--playlist-items", fmt.Sprintf("%d:%d", start, end),
	}

	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("yt-dlp channel: %w", err)
	}

	return parseVideoNDJSON(out)
}

func parseVideoNDJSON(data []byte) ([]Video, error) {
	var videos []Video
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}

		v := Video{
			ID:          getString(raw, "id"),
			Title:       getString(raw, "title"),
			URL:         getString(raw, "url"),
			Channel:     getString(raw, "channel"),
			ChannelURL:  getString(raw, "channel_url"),
			UploadDate:  getString(raw, "upload_date"),
			Description: getString(raw, "description"),
			Thumbnail:   getString(raw, "thumbnail"),
		}
		// Flat-playlist has channel info in playlist_channel/playlist_uploader
		if v.Channel == "" {
			v.Channel = getString(raw, "playlist_channel")
		}
		if v.Channel == "" {
			v.Channel = getString(raw, "playlist_uploader")
		}
		if v.Channel == "" {
			v.Channel = getString(raw, "uploader")
		}

		if v.URL == "" {
			v.URL = getString(raw, "webpage_url")
		}
		if v.URL == "" && v.ID != "" {
			v.URL = "https://www.youtube.com/watch?v=" + v.ID
		}

		if d, ok := raw["duration"].(float64); ok {
			v.Duration = d
			v.DurationStr = formatDuration(d)
		}
		// Flat-playlist may provide duration_string instead
		if v.DurationStr == "" {
			if ds := getString(raw, "duration_string"); ds != "" {
				v.DurationStr = ds
			}
		}
		if vc, ok := raw["view_count"].(float64); ok {
			v.ViewCount = int64(vc)
		}
		if pi, ok := raw["playlist_index"].(float64); ok {
			v.PlaylistIndex = int(pi)
		} else if pi, ok := raw["playlist_autonumber"].(float64); ok {
			v.PlaylistIndex = int(pi)
		}

		videos = append(videos, v)
	}

	return videos, nil
}
