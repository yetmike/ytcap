package ytdlp

import (
	"testing"
)

func TestParseSearchNDJSON(t *testing.T) {
	input := []byte(`{"id":"vid1","title":"Go Tutorial","url":"https://youtube.com/watch?v=vid1","_type":"video","channel":"TechChan","duration":300.0,"view_count":50000.0}
{"id":"ch1","title":"TechChan","url":"https://youtube.com/@techchan","_type":"channel","subscriber_count":1500000.0}
`)

	results, err := parseSearchNDJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Channels should be sorted first
	if results[0].Type != "channel" {
		t.Errorf("expected channel first, got %q", results[0].Type)
	}
	if results[0].Subscribers != "1.5M" {
		t.Errorf("subscribers = %q, want 1.5M", results[0].Subscribers)
	}

	if results[1].Type != "video" {
		t.Errorf("expected video second, got %q", results[1].Type)
	}
	if results[1].DurationStr != "5:00" {
		t.Errorf("duration = %q, want 5:00", results[1].DurationStr)
	}
	if results[1].ViewCount != 50000 {
		t.Errorf("view_count = %d, want 50000", results[1].ViewCount)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds float64
		want    string
	}{
		{0, "0:00"},
		{30, "0:30"},
		{90, "1:30"},
		{3661, "1:01:01"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.seconds)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestFormatCount(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{500, "500"},
		{1500, "1.5K"},
		{1500000, "1.5M"},
		{2500000000, "2.5B"},
	}
	for _, tt := range tests {
		got := formatCount(tt.n)
		if got != tt.want {
			t.Errorf("formatCount(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
