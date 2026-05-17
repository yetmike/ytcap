package cmd

import (
	"github.com/rs/zerolog"

	"github.com/yetmike/ytcap/internal/config"
	"github.com/yetmike/ytcap/internal/summarize"
)

func newSummarizeClient(cfg *config.Config, logger zerolog.Logger) *summarize.Client {
	return &summarize.Client{
		Binary:   cfg.Summarize.Binary,
		Length:   cfg.Summarize.Length,
		Language: cfg.Summarize.Language,
		CLI:      cfg.Summarize.CLI,
		Logger:   logger,
	}
}
