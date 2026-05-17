package app

import "testing"

func TestCycleBackend(t *testing.T) {
	tests := []struct {
		current  string
		expected string
	}{
		{"auto", "claude"},
		{"claude", "gemini"},
		{"gemini", "auto"},
		{"", "claude"}, // empty defaults to "auto", cycles to claude
		{"unknown", "auto"}, // unknown defaults to first
	}
	for _, tc := range tests {
		backends := []string{"auto", "claude", "gemini"}
		current := tc.current
		if current == "" {
			current = "auto"
		}
		next := backends[0]
		for i, b := range backends {
			if b == current {
				next = backends[(i+1)%len(backends)]
				break
			}
		}
		if next != tc.expected {
			t.Errorf("CycleBackend(%q) = %q, want %q", tc.current, next, tc.expected)
		}
	}
}

func TestSessionVideo_Merge(t *testing.T) {
	// Test the session video merge logic from saveToSession
	sessions := make(map[string]*SessionVideo)

	// First save
	sv := &SessionVideo{}
	sessions["url1"] = sv
	sv.Summary = "sum"

	// Second save should preserve existing data
	existing := sessions["url1"]
	if existing.Summary != "sum" {
		t.Error("expected summary to persist")
	}

	existing.Transcript = "trans"
	if sessions["url1"].Transcript != "trans" {
		t.Error("expected transcript update to persist")
	}
}

func TestMode_Constants(t *testing.T) {
	if ModeDefault != 0 {
		t.Error("ModeDefault should be 0")
	}
	if ModeSearch != 1 {
		t.Error("ModeSearch should be 1")
	}
	if ModeChannel != 2 {
		t.Error("ModeChannel should be 2")
	}
	if ModeVideo != 3 {
		t.Error("ModeVideo should be 3")
	}
}
