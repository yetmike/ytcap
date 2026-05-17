package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Summarize SummarizeConfig `yaml:"summarize"`
	YtDlp     YtDlpConfig     `yaml:"ytdlp"`
	Cache     CacheConfig     `yaml:"cache"`
	Storage   StorageConfig   `yaml:"storage"`
	UI        UIConfig        `yaml:"ui"`
	Keybindings KeybindingsConfig `yaml:"keybindings"`
}

type SummarizeConfig struct {
	Binary   string `yaml:"binary"`
	Length   string `yaml:"length"`
	Language string `yaml:"language"`
	CLI      string `yaml:"cli"`
}

type YtDlpConfig struct {
	Binary      string `yaml:"binary"`
	SearchLimit int    `yaml:"search_limit"`
}

type CacheConfig struct {
	Enabled bool   `yaml:"enabled"`
	Dir     string `yaml:"dir"`
	TTLDays int    `yaml:"ttl_days"`
}

type StorageConfig struct {
	Dir string `yaml:"dir"`
}

type UIConfig struct {
	Skin       string `yaml:"skin"`
	DateFormat string `yaml:"date_format"`
	Mouse      bool   `yaml:"mouse"`
}

type KeybindingsConfig struct {
	Quit        string `yaml:"quit"`
	Back        string `yaml:"back"`
	Search      string `yaml:"search"`
	Filter      string `yaml:"filter"`
	SaveSummary string `yaml:"save_summary"`
	SaveChat    string `yaml:"save_chat"`
	ToggleView  string `yaml:"toggle_view"`
	Help        string `yaml:"help"`
	Chat        string `yaml:"chat"`
}

func Default() *Config {
	return &Config{
		Summarize: SummarizeConfig{
			Binary:   "summarize",
			Length:   "long",
			Language: "auto",
			CLI:      "auto",
		},
		YtDlp: YtDlpConfig{
			Binary:      "yt-dlp",
			SearchLimit: 20,
		},
		Cache: CacheConfig{
			Enabled: true,
			Dir:     filepath.Join(DataDir(), "cache"),
			TTLDays: 7,
		},
		Storage: StorageConfig{
			Dir: filepath.Join(os.Getenv("HOME"), "ytcap-notes"),
		},
		UI: UIConfig{
			Skin:       "default",
			DateFormat: "2006-01-02",
			Mouse:      false,
		},
		Keybindings: KeybindingsConfig{
			Quit:        "q",
			Back:        "Escape",
			Search:      ":",
			Filter:      "/",
			SaveSummary: "S",
			SaveChat:    "C",
			ToggleView:  "Tab",
			Help:        "?",
			Chat:        "/",
		},
	}
}

func DataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ytcap")
}

func FilePath() string {
	return filepath.Join(DataDir(), "config.yaml")
}

func Load() (*Config, error) {
	if err := os.MkdirAll(DataDir(), 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	cfg := Default()
	path := FilePath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := saveWithComments(); err != nil {
				return nil, fmt.Errorf("saving default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Expand ~ in paths
	cfg.Cache.Dir = expandHome(cfg.Cache.Dir)
	cfg.Storage.Dir = expandHome(cfg.Storage.Dir)

	return cfg, nil
}

func Print(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

func Reset() error {
	return saveWithComments()
}

// saveWithComments writes the default config with inline documentation.
func saveWithComments() error {
	content := `# ytcap configuration
# Edit with: ytcap config edit
# Reset with: ytcap config reset

summarize:
  binary: "summarize"    # path to summarize CLI binary
  length: "long"         # summary length: short | medium | long | xl
  language: "auto"       # output language: auto (match source), en, de, etc.
  cli: "auto"            # AI backend: auto | claude | gemini

ytdlp:
  binary: "yt-dlp"       # path to yt-dlp binary
  search_limit: 20       # max results per search page

cache:
  enabled: true          # set to false to disable caching
  dir: "~/.ytcap/cache"  # cache directory for transcripts/summaries
  ttl_days: 7            # cache expiry in days

storage:
  dir: "~/ytcap-notes"   # where saved markdown files go

ui:
  # Custom skins: drop a YAML file in ~/.ytcap/skins/<name>.yaml
  skin: "default"
  date_format: "2006-01-02"   # Go time format string
  mouse: false                # enable mouse support in the TUI

keybindings:
  quit: "q"
  back: "Escape"
  search: ":"
  filter: "/"
  save_summary: "S"
  save_chat: "C"
  toggle_view: "Tab"
  help: "?"
  chat: "/"
`
	return os.WriteFile(FilePath(), []byte(content), 0o644)
}

func expandHome(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
