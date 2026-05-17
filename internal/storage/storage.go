package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/yetmike/ytcap/internal/summarize"
)

type Storage struct {
	Dir string
}

func New(dir string) *Storage {
	return &Storage{Dir: dir}
}

func (s *Storage) DefaultSummaryPath(videoID, title string) string {
	return filepath.Join(s.Dir, fmt.Sprintf("%s-%s.md", videoID, slugify(title)))
}

func (s *Storage) DefaultTranscriptPath(videoID, title string) string {
	return filepath.Join(s.Dir, fmt.Sprintf("%s-%s-transcript.md", videoID, slugify(title)))
}

func (s *Storage) DefaultChatPath(videoID, title string) string {
	return filepath.Join(s.Dir, fmt.Sprintf("%s-%s-chat.md", videoID, slugify(title)))
}

func (s *Storage) SaveSummary(videoID, title, channel, url, duration, published, summary string) (string, error) {
	return s.SaveSummaryTo(s.Dir, videoID, title, channel, url, duration, published, summary)
}

func (s *Storage) SaveSummaryTo(userPath, videoID, title, channel, url, duration, published, summary string) (string, error) {
	defaultFilename := fmt.Sprintf("%s-%s.md", videoID, slugify(title))
	path, err := resolveSavePath(userPath, defaultFilename)
	if err != nil {
		return "", err
	}

	content := fmt.Sprintf(`# %s

**Channel:** %s
**URL:** %s
**Duration:** %s
**Published:** %s
**Saved:** %s

---

%s
`, title, channel, url, duration, published, time.Now().Format("2006-01-02"), summary)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing summary: %w", err)
	}

	return path, nil
}

func (s *Storage) SaveTranscript(videoID, title, channel, url, duration, published, transcript string) (string, error) {
	return s.SaveTranscriptTo(s.Dir, videoID, title, channel, url, duration, published, transcript)
}

func (s *Storage) SaveTranscriptTo(userPath, videoID, title, channel, url, duration, published, transcript string) (string, error) {
	defaultFilename := fmt.Sprintf("%s-%s-transcript.md", videoID, slugify(title))
	path, err := resolveSavePath(userPath, defaultFilename)
	if err != nil {
		return "", err
	}

	content := fmt.Sprintf(`# %s — Transcript

**Channel:** %s
**URL:** %s
**Duration:** %s
**Published:** %s
**Saved:** %s

---

%s
`, title, channel, url, duration, published, time.Now().Format("2006-01-02"), transcript)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing transcript: %w", err)
	}

	return path, nil
}

func (s *Storage) SaveChat(videoID, title, channel, url string, messages []summarize.ChatMessage) (string, error) {
	return s.SaveChatTo(s.Dir, videoID, title, channel, url, messages)
}

func (s *Storage) SaveChatTo(userPath, videoID, title, channel, url string, messages []summarize.ChatMessage) (string, error) {
	defaultFilename := fmt.Sprintf("%s-%s-chat.md", videoID, slugify(title))
	path, err := resolveSavePath(userPath, defaultFilename)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Chat: %s\n\n", title)
	fmt.Fprintf(&b, "**Video:** %s\n", url)
	fmt.Fprintf(&b, "**Channel:** %s\n", channel)
	fmt.Fprintf(&b, "**Saved:** %s\n\n---\n\n", time.Now().Format("2006-01-02"))

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			fmt.Fprintf(&b, "**You:** %s\n\n", msg.Content)
		case "ai":
			fmt.Fprintf(&b, "**AI:** %s\n\n", msg.Content)
		}
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("writing chat: %w", err)
	}

	return path, nil
}

func (s *Storage) DeleteChat(videoID, title string) {
	filename := fmt.Sprintf("%s-%s-chat.md", videoID, slugify(title))
	path := filepath.Join(s.Dir, filename)
	os.Remove(path)
}

// resolveSavePath determines the final file path from user input.
// Rules:
//   - If userPath is an existing directory, or ends with '/' → save defaultFilename inside it
//   - If userPath is not an existing dir and doesn't end with '/' → treat as full file path
func resolveSavePath(userPath, defaultFilename string) (string, error) {
	// Expand ~
	if len(userPath) > 1 && userPath[:2] == "~/" {
		home, _ := os.UserHomeDir()
		userPath = filepath.Join(home, userPath[2:])
	}

	// Ends with / → it's a directory
	if strings.HasSuffix(userPath, "/") || strings.HasSuffix(userPath, string(filepath.Separator)) {
		if err := os.MkdirAll(userPath, 0o755); err != nil {
			return "", fmt.Errorf("creating directory: %w", err)
		}
		return filepath.Join(userPath, defaultFilename), nil
	}

	// Check if it's an existing directory
	info, err := os.Stat(userPath)
	if err == nil && info.IsDir() {
		return filepath.Join(userPath, defaultFilename), nil
	}

	// Treat as full file path — ensure parent dir exists
	parentDir := filepath.Dir(userPath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}
	return userPath, nil
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9-]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = nonAlphaNum.ReplaceAllString(s, "")
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}
