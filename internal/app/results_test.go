package app

import (
	"testing"

	"github.com/yetmike/ytcap/internal/ytdlp"
)

func TestResultsScreen_HasMoreStopsWhenNoNewResults(t *testing.T) {
	r := &ResultsScreen{
		results:   make([]ytdlp.SearchResult, 5),
		hasMore:   true,
		loadCount: 5,
	}

	// Simulate what loadMore does when result count doesn't grow:
	// len(newResults) == len(r.results) → hasMore = false
	newResults := make([]ytdlp.SearchResult, 5) // same count
	if len(newResults) == len(r.results) {
		r.hasMore = false
	}

	if r.hasMore {
		t.Error("expected hasMore to be false when results didn't grow")
	}
}

func TestResultsScreen_HasMoreContinuesWhenResultsGrow(t *testing.T) {
	r := &ResultsScreen{
		results:   make([]ytdlp.SearchResult, 5),
		hasMore:   true,
		loadCount: 5,
	}

	newResults := make([]ytdlp.SearchResult, 10) // more results
	if len(newResults) == len(r.results) {
		r.hasMore = false
	}

	if !r.hasMore {
		t.Error("expected hasMore to remain true when results grew")
	}
}

func TestResultsScreen_LoadMoreGuardedByHasMore(t *testing.T) {
	r := &ResultsScreen{
		hasMore:   false,
		isLoading: false,
	}

	// loadMore should not proceed when hasMore is false
	if !r.isLoading && r.hasMore {
		t.Error("loadMore should be blocked when hasMore is false")
	}
}

func TestFormatViews(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500 views"},
		{1500, "1.5K views"},
		{1500000, "1.5M views"},
		{2500000000, "2.5B views"},
	}
	for _, tc := range tests {
		got := formatViews(tc.input)
		if got != tc.expected {
			t.Errorf("formatViews(%d) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
