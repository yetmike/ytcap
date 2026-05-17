package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetmike/ytcap/internal/summarize"
)

func TestDeleteChat_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	_, err := s.SaveChat("vid1", "Test Video", "channel", "https://example.com", []summarize.ChatMessage{
		{Role: "user", Content: "hello"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	pattern := filepath.Join(dir, "vid1-*.md")
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		t.Fatal("expected chat file to exist")
	}

	s.DeleteChat("vid1", "Test Video")

	// Verify file removed
	if _, err := os.Stat(matches[0]); !os.IsNotExist(err) {
		t.Error("expected chat file to be deleted")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"Hello World", "hello-world"},
		{"Test!@#Video", "testvideo"},
		{"", ""},
		{"---leading-trailing---", "leading-trailing"},
		{"A " + string(make([]byte, 100)), ""}, // long string gets truncated
	}
	for _, tc := range tests {
		got := slugify(tc.input)
		if len(got) > 60 {
			t.Errorf("slugify(%q) length %d exceeds 60", tc.input, len(got))
		}
		if tc.expected != "" {
			if got != tc.expected {
				t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		}
	}
}

func TestSlugify_MaxLength(t *testing.T) {
	long := "this is a really long title that should be truncated to sixty characters at most by the slugify function"
	got := slugify(long)
	if len(got) > 60 {
		t.Errorf("slug length %d > 60", len(got))
	}
}

func TestSaveSummary_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	path, err := s.SaveSummary("vid1", "My Video", "Channel", "https://example.com", "10:00", "2024-01-01", "This is the summary")
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "# My Video") {
		t.Error("expected title in output")
	}
	if !strings.Contains(content, "**Channel:** Channel") {
		t.Error("expected channel in output")
	}
	if !strings.Contains(content, "This is the summary") {
		t.Error("expected summary content")
	}
}

func TestSaveTranscript_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	path, err := s.SaveTranscript("vid1", "My Video", "Ch", "https://example.com", "5:00", "2024-01-01", "Full transcript here")
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "— Transcript") {
		t.Error("expected Transcript marker")
	}
	if !strings.Contains(content, "Full transcript here") {
		t.Error("expected transcript content")
	}
}

func TestSaveSummaryTo_DirectoryPath(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	// Path ending with / should save inside directory
	subdir := filepath.Join(dir, "subdir") + "/"
	path, err := s.SaveSummaryTo(subdir, "vid1", "Test", "Ch", "http://x", "1:00", "2024", "sum")
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Dir(path) != filepath.Join(dir, "subdir") {
		t.Errorf("expected file in subdir, got %s", path)
	}
}

func TestSaveSummaryTo_ExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	// Path pointing to existing directory should save inside it
	path, err := s.SaveSummaryTo(dir, "vid1", "Test", "Ch", "http://x", "1:00", "2024", "sum")
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Dir(path) != dir {
		t.Errorf("expected file in %s, got dir %s", dir, filepath.Dir(path))
	}
}

func TestSaveSummaryTo_FullFilePath(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	fullPath := filepath.Join(dir, "custom-name.md")
	path, err := s.SaveSummaryTo(fullPath, "vid1", "Test", "Ch", "http://x", "1:00", "2024", "sum")
	if err != nil {
		t.Fatal(err)
	}

	if path != fullPath {
		t.Errorf("expected %s, got %s", fullPath, path)
	}
}

func TestDefaultPaths(t *testing.T) {
	s := New("/tmp/notes")

	sumPath := s.DefaultSummaryPath("abc123", "My Video Title")
	if !strings.Contains(sumPath, "abc123-my-video-title.md") {
		t.Errorf("unexpected summary path: %s", sumPath)
	}

	transPath := s.DefaultTranscriptPath("abc123", "My Video Title")
	if !strings.Contains(transPath, "abc123-my-video-title-transcript.md") {
		t.Errorf("unexpected transcript path: %s", transPath)
	}

	chatPath := s.DefaultChatPath("abc123", "My Video Title")
	if !strings.Contains(chatPath, "abc123-my-video-title-chat.md") {
		t.Errorf("unexpected chat path: %s", chatPath)
	}
}

func TestSaveChat_MultipleMessages(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	messages := []summarize.ChatMessage{
		{Role: "user", Content: "What is this about?"},
		{Role: "ai", Content: "This video covers Docker."},
		{Role: "user", Content: "Tell me more"},
		{Role: "ai", Content: "Docker uses containers."},
	}

	path, err := s.SaveChat("vid1", "Docker Tutorial", "TechChannel", "https://example.com", messages)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "**You:** What is this about?") {
		t.Error("expected user message")
	}
	if !strings.Contains(content, "**AI:** This video covers Docker.") {
		t.Error("expected AI message")
	}
	if strings.Count(content, "**You:**") != 2 {
		t.Error("expected 2 user messages")
	}
}
