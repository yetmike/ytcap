package ytdlp

import (
	"testing"
)

func TestNormalizeChannelURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://www.youtube.com/@fireship", "https://www.youtube.com/@fireship/videos"},
		{"https://www.youtube.com/@fireship/", "https://www.youtube.com/@fireship/videos"},
		{"https://www.youtube.com/@fireship/featured", "https://www.youtube.com/@fireship/videos"},
		{"https://www.youtube.com/@fireship/shorts", "https://www.youtube.com/@fireship/videos"},
		{"https://www.youtube.com/@fireship/videos", "https://www.youtube.com/@fireship/videos"},
		{"https://www.youtube.com/channel/UCsBjURrPoezykLs9EqgamOA", "https://www.youtube.com/channel/UCsBjURrPoezykLs9EqgamOA/videos"},
	}

	for _, tt := range tests {
		got := normalizeChannelURL(tt.input)
		if got != tt.want {
			t.Errorf("normalizeChannelURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseVideoNDJSON(t *testing.T) {
	input := []byte(`{"id":"abc123","title":"Test Video","url":"https://youtube.com/watch?v=abc123","duration":102.0,"duration_string":"1:42","view_count":1000.0,"playlist_autonumber":1.0}
{"id":"def456","title":"Another Video","url":"","webpage_url":"https://youtube.com/watch?v=def456","view_count":500.0,"duration_string":"5:30","playlist_autonumber":2.0}
`)

	videos, err := parseVideoNDJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(videos) != 2 {
		t.Fatalf("expected 2 videos, got %d", len(videos))
	}

	// First video: has duration float
	if videos[0].ID != "abc123" {
		t.Errorf("video[0].ID = %q, want abc123", videos[0].ID)
	}
	if videos[0].DurationStr != "1:42" {
		t.Errorf("video[0].DurationStr = %q, want 1:42", videos[0].DurationStr)
	}
	if videos[0].ViewCount != 1000 {
		t.Errorf("video[0].ViewCount = %d, want 1000", videos[0].ViewCount)
	}
	if videos[0].PlaylistIndex != 1 {
		t.Errorf("video[0].PlaylistIndex = %d, want 1", videos[0].PlaylistIndex)
	}

	// Second video: no duration float, uses duration_string; URL falls back to webpage_url
	if videos[1].URL != "https://youtube.com/watch?v=def456" {
		t.Errorf("video[1].URL = %q, want webpage_url fallback", videos[1].URL)
	}
	if videos[1].DurationStr != "5:30" {
		t.Errorf("video[1].DurationStr = %q, want 5:30", videos[1].DurationStr)
	}
	if videos[1].PlaylistIndex != 2 {
		t.Errorf("video[1].PlaylistIndex = %d, want 2", videos[1].PlaylistIndex)
	}
}

func TestParseVideoNDJSON_MissingIDFallback(t *testing.T) {
	input := []byte(`{"id":"xyz789","title":"ID Only"}
`)
	videos, err := parseVideoNDJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(videos) != 1 {
		t.Fatalf("expected 1 video, got %d", len(videos))
	}
	if videos[0].URL != "https://www.youtube.com/watch?v=xyz789" {
		t.Errorf("expected URL fallback from ID, got %q", videos[0].URL)
	}
}

func TestParseVideoNDJSON_EmptyInput(t *testing.T) {
	videos, err := parseVideoNDJSON([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(videos) != 0 {
		t.Errorf("expected 0 videos for empty input, got %d", len(videos))
	}
}

func TestParseVideoNDJSON_ChannelFallbacks(t *testing.T) {
	// playlist_channel should be used as channel name
	input := []byte(`{"id":"v1","title":"T","playlist_channel":"PlaylistChan","url":"http://x"}
`)
	videos, _ := parseVideoNDJSON(input)
	if len(videos) != 1 {
		t.Fatalf("expected 1 video, got %d", len(videos))
	}
	if videos[0].Channel != "PlaylistChan" {
		t.Errorf("expected channel = PlaylistChan, got %q", videos[0].Channel)
	}
}

func TestParseVideoNDJSON_PlaylistIndex(t *testing.T) {
	// playlist_index should take precedence over playlist_autonumber
	input := []byte(`{"id":"v1","title":"T","url":"http://x","playlist_index":5.0,"playlist_autonumber":3.0}
`)
	videos, _ := parseVideoNDJSON(input)
	if videos[0].PlaylistIndex != 5 {
		t.Errorf("expected PlaylistIndex = 5, got %d", videos[0].PlaylistIndex)
	}
}

func TestParseSearchNDJSON_EmptyLines(t *testing.T) {
	input := []byte("\n\n{\"id\":\"v1\",\"title\":\"T\",\"url\":\"http://x\"}\n\n")
	results, err := parseSearchNDJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestParseSearchNDJSON_EmptyInput(t *testing.T) {
	results, err := parseSearchNDJSON([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestGetString_MissingKey(t *testing.T) {
	m := map[string]interface{}{
		"present": "value",
		"number":  42.0,
	}
	if getString(m, "present") != "value" {
		t.Error("expected 'value'")
	}
	if getString(m, "missing") != "" {
		t.Error("expected empty string for missing key")
	}
	if getString(m, "number") != "" {
		t.Error("expected empty string for non-string value")
	}
}

func TestNormalizeChannelURL_AlreadyHasVideos(t *testing.T) {
	url := "https://www.youtube.com/@test/videos"
	got := normalizeChannelURL(url)
	if got != url {
		t.Errorf("expected unchanged URL, got %q", got)
	}
}

func TestFormatDuration_EdgeCases(t *testing.T) {
	tests := []struct {
		seconds float64
		want    string
	}{
		{59, "0:59"},
		{60, "1:00"},
		{3600, "1:00:00"},
		{7261, "2:01:01"},
	}
	for _, tc := range tests {
		got := formatDuration(tc.seconds)
		if got != tc.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}
