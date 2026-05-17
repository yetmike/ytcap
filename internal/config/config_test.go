package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault_ReturnsValidConfig(t *testing.T) {
	cfg := Default()

	if cfg.Summarize.Binary != "summarize" {
		t.Errorf("expected summarize binary = 'summarize', got %q", cfg.Summarize.Binary)
	}
	if cfg.YtDlp.Binary != "yt-dlp" {
		t.Errorf("expected yt-dlp binary = 'yt-dlp', got %q", cfg.YtDlp.Binary)
	}
	if cfg.YtDlp.SearchLimit != 20 {
		t.Errorf("expected search_limit = 20, got %d", cfg.YtDlp.SearchLimit)
	}
	if cfg.Cache.TTLDays != 7 {
		t.Errorf("expected ttl_days = 7, got %d", cfg.Cache.TTLDays)
	}
	if !cfg.Cache.Enabled {
		t.Error("expected cache to be enabled by default")
	}
	if cfg.UI.Skin != "default" {
		t.Errorf("expected skin = 'default', got %q", cfg.UI.Skin)
	}
	if cfg.UI.DateFormat != "2006-01-02" {
		t.Errorf("expected date format = '2006-01-02', got %q", cfg.UI.DateFormat)
	}
	if cfg.Summarize.CLI != "auto" {
		t.Errorf("expected CLI = 'auto', got %q", cfg.Summarize.CLI)
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/foo/bar", filepath.Join(home, "foo/bar")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~", "~"}, // single ~ without / is not expanded
		{"", ""},
	}
	for _, tc := range tests {
		got := expandHome(tc.input)
		if got != tc.expected {
			t.Errorf("expandHome(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestLoad_CreatesDefaultOnMissing(t *testing.T) {
	// Override HOME to a temp dir so we don't touch real config
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.YtDlp.Binary != "yt-dlp" {
		t.Errorf("expected default yt-dlp binary, got %q", cfg.YtDlp.Binary)
	}

	// Config file should have been created
	path := FilePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected config file to be created")
	}
}

func TestLoad_ParsesExistingConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create data dir and config
	dataDir := filepath.Join(tmp, ".ytcap")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	yaml := `ytdlp:
  binary: custom-ytdlp
  search_limit: 50
cache:
  enabled: false
  ttl_days: 14
`
	if err := os.WriteFile(filepath.Join(dataDir, "config.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.YtDlp.Binary != "custom-ytdlp" {
		t.Errorf("expected custom binary, got %q", cfg.YtDlp.Binary)
	}
	if cfg.YtDlp.SearchLimit != 50 {
		t.Errorf("expected search_limit = 50, got %d", cfg.YtDlp.SearchLimit)
	}
	if cfg.Cache.Enabled {
		t.Error("expected cache to be disabled")
	}
	if cfg.Cache.TTLDays != 14 {
		t.Errorf("expected ttl_days = 14, got %d", cfg.Cache.TTLDays)
	}
}

func TestReset_WritesDefaults(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if err := os.MkdirAll(filepath.Join(tmp, ".ytcap"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Reset(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.YtDlp.SearchLimit != 20 {
		t.Errorf("expected default search_limit after reset, got %d", cfg.YtDlp.SearchLimit)
	}
}
