package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Cache struct {
	Dir     string
	TTLDays int
	Enabled bool
}

type Meta struct {
	Title    string    `json:"title"`
	Channel  string    `json:"channel"`
	Duration string    `json:"duration"`
	Views    int64     `json:"views"`
	CachedAt time.Time `json:"cached_at"`
}

func New(dir string, ttlDays int, enabled bool) *Cache {
	return &Cache{Dir: dir, TTLDays: ttlDays, Enabled: enabled}
}

func (c *Cache) videoDir(videoID string) string {
	return filepath.Join(c.Dir, videoID)
}

func (c *Cache) GetTranscript(videoID string) (string, bool) {
	return c.getFile(videoID, "transcript.txt")
}

func (c *Cache) GetSummary(videoID string) (string, bool) {
	return c.getFile(videoID, "summary.txt")
}

func (c *Cache) GetMeta(videoID string) (*Meta, bool) {
	if !c.Enabled {
		return nil, false
	}
	path := filepath.Join(c.videoDir(videoID), "meta.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, false
	}
	if time.Since(meta.CachedAt) > time.Duration(c.TTLDays)*24*time.Hour {
		return nil, false
	}
	return &meta, true
}

func (c *Cache) SetTranscript(videoID, content string) error {
	return c.setFile(videoID, "transcript.txt", content)
}

func (c *Cache) SetSummary(videoID, content string) error {
	return c.setFile(videoID, "summary.txt", content)
}

func (c *Cache) SetMeta(videoID string, meta *Meta) error {
	meta.CachedAt = time.Now()
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling meta: %w", err)
	}
	return c.setFile(videoID, "meta.json", string(data))
}

func (c *Cache) getFile(videoID, filename string) (string, bool) {
	if !c.Enabled {
		return "", false
	}
	// Check TTL via meta
	metaPath := filepath.Join(c.videoDir(videoID), "meta.json")
	metaData, err := os.ReadFile(metaPath)
	if err == nil {
		var meta Meta
		if json.Unmarshal(metaData, &meta) == nil {
			if time.Since(meta.CachedAt) > time.Duration(c.TTLDays)*24*time.Hour {
				return "", false
			}
		}
	}

	path := filepath.Join(c.videoDir(videoID), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

func (c *Cache) setFile(videoID, filename, content string) error {
	dir := c.videoDir(videoID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}
	path := filepath.Join(dir, filename)
	return os.WriteFile(path, []byte(content), 0o644)
}
