package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCache_SetAndGetTranscript(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 7, true)

	// Set meta first (required for TTL check)
	if err := c.SetMeta("vid1", &Meta{Title: "Test", Channel: "Ch"}); err != nil {
		t.Fatal(err)
	}

	if err := c.SetTranscript("vid1", "hello world"); err != nil {
		t.Fatal(err)
	}

	got, ok := c.GetTranscript("vid1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestCache_SetAndGetSummary(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 7, true)
	if err := c.SetMeta("vid1", &Meta{Title: "Test"}); err != nil {
		t.Fatal(err)
	}
	if err := c.SetSummary("vid1", "summary content"); err != nil {
		t.Fatal(err)
	}
	got, ok := c.GetSummary("vid1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got != "summary content" {
		t.Errorf("got %q", got)
	}
}

func TestCache_DisabledReturnsNothing(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 7, false) // disabled

	if err := c.SetMeta("vid1", &Meta{Title: "Test"}); err != nil {
		t.Fatal(err)
	}
	if err := c.SetTranscript("vid1", "hello"); err != nil {
		t.Fatal(err)
	}

	_, ok := c.GetTranscript("vid1")
	if ok {
		t.Error("expected cache miss when disabled")
	}
}

func TestCache_ExpiredTTL(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 7, true)

	// Write an expired meta
	meta := &Meta{
		Title:    "Old Video",
		CachedAt: time.Now().Add(-8 * 24 * time.Hour), // 8 days ago
	}
	data, _ := json.MarshalIndent(meta, "", "  ")
	videoDir := filepath.Join(dir, "vid1")
	if err := os.MkdirAll(videoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(videoDir, "meta.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(videoDir, "transcript.txt"), []byte("old text"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, ok := c.GetTranscript("vid1")
	if ok {
		t.Error("expected cache miss for expired TTL")
	}
}

func TestCache_GetMeta_Expired(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 1, true)

	meta := &Meta{
		Title:    "Test",
		CachedAt: time.Now().Add(-2 * 24 * time.Hour),
	}
	data, _ := json.MarshalIndent(meta, "", "  ")
	videoDir := filepath.Join(dir, "vid1")
	if err := os.MkdirAll(videoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(videoDir, "meta.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	_, ok := c.GetMeta("vid1")
	if ok {
		t.Error("expected expired meta to return false")
	}
}

func TestCache_GetMeta_Valid(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 7, true)

	if err := c.SetMeta("vid1", &Meta{Title: "Fresh", Channel: "Chan", Views: 1000}); err != nil {
		t.Fatal(err)
	}

	meta, ok := c.GetMeta("vid1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if meta.Title != "Fresh" {
		t.Errorf("got title %q", meta.Title)
	}
	if meta.Views != 1000 {
		t.Errorf("got views %d", meta.Views)
	}
}

func TestCache_MissingFile(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 7, true)

	_, ok := c.GetTranscript("nonexistent")
	if ok {
		t.Error("expected miss for nonexistent video")
	}
}
