package cmd

import (
	"os"
	"testing"

	"github.com/yetmike/ytcap/internal/config"
)

func TestSetVersion(t *testing.T) {
	SetVersion("1.2.3")
	if version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %s", version)
	}
}

func TestCheckDependencies_MissingYtDlp(t *testing.T) {
	cfg := config.Default()
	cfg.YtDlp.Binary = "nonexistent-binary-ytdlp-xyz"

	err := checkDependencies(cfg)
	if err == nil {
		t.Error("expected error for missing yt-dlp")
	}
}

func TestCheckDependencies_MissingSummarize(t *testing.T) {
	cfg := config.Default()
	// yt-dlp might not exist in test env either, so use a real binary
	cfg.YtDlp.Binary = "true" // /usr/bin/true exists everywhere
	cfg.Summarize.Binary = "nonexistent-binary-summarize-xyz"

	err := checkDependencies(cfg)
	if err == nil {
		t.Error("expected error for missing summarize")
	}
}

func TestVersionCmd(t *testing.T) {
	SetVersion("test-version")
	rootCmd.SetArgs([]string{"version"})
	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = rootCmd.Execute()

	w.Close()
	os.Stdout = old

	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if output != "ytcap test-version\n" {
		t.Errorf("unexpected version output: %q", output)
	}
}

func TestCacheCmd_ClearNonexistent(t *testing.T) {
	// Clearing a non-existent cache dir should not error
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	rootCmd.SetArgs([]string{"cache", "clear"})
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCacheCmd_ClearSpecificVideo(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	rootCmd.SetArgs([]string{"cache", "clear", "vid123"})
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
