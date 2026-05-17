package ytdlp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	args := []string{
		fmt.Sprintf("ytsearch%d:%s", limit, query),
		"--flat-playlist",
		"--dump-json",
		"--no-warnings",
		"--quiet",
	}

	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("yt-dlp search: %w", err)
	}

	return parseSearchNDJSON(out)
}

func parseSearchNDJSON(data []byte) ([]SearchResult, error) {
	var results []SearchResult
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

		r := SearchResult{
			ID:    getString(raw, "id"),
			Title: getString(raw, "title"),
			URL:   getString(raw, "url"),
		}

		// yt-dlp identifies channel entries in search results by ie_key
		// ("YoutubeTab"/"channel"/"Channel") or by URL shape (/channel/, /@handle).
		// _type is usually just "url" for both videos and channels.
		typ := getString(raw, "_type")
		ieKey := getString(raw, "ie_key")
		url := getString(raw, "url")
		isChannel := typ == "channel" || typ == "Channel" ||
			ieKey == "YoutubeTab" || ieKey == "channel" || ieKey == "Channel" ||
			strings.Contains(url, "/channel/") || strings.Contains(url, "youtube.com/@")

		switch {
		case isChannel:
			r.Type = "channel"
			r.Channel = r.Title
			r.ChannelURL = getString(raw, "url")
			if sub, ok := raw["subscriber_count"]; ok {
				if f, ok := sub.(float64); ok {
					r.Subscribers = formatCount(int64(f))
				}
			}
		default:
			r.Type = "video"
			r.Channel = getString(raw, "channel")
			r.ChannelURL = getString(raw, "channel_url")
			if d, ok := raw["duration"].(float64); ok {
				r.Duration = d
				r.DurationStr = formatDuration(d)
			}
			if vc, ok := raw["view_count"].(float64); ok {
				r.ViewCount = int64(vc)
			}
			r.UploadDate = getString(raw, "upload_date")
			r.Description = getString(raw, "description")
		}

		if r.URL == "" {
			r.URL = getString(raw, "webpage_url")
		}

		results = append(results, r)
	}

	// Sort: channels first, then videos
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Type != results[j].Type {
			return results[i].Type == "channel"
		}
		return false
	})

	return results, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func formatDuration(seconds float64) string {
	s := int(seconds)
	if s < 60 {
		return fmt.Sprintf("0:%02d", s)
	}
	h := s / 3600
	m := (s % 3600) / 60
	sec := s % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%d:%02d", m, sec)
}

func formatCount(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
