package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/yetmike/ytcap/internal/app"
	"github.com/yetmike/ytcap/internal/config"
)

var (
	version string
	noCache bool
)

func SetVersion(v string) {
	version = v
}

var rootCmd = &cobra.Command{
	Use:   "ytcap [search query]",
	Short: "Distraction-free YouTube TUI with AI summaries and Q&A",
	Long:  "A k9s-inspired TUI for distraction-free YouTube browsing, transcript reading, and AI-powered Q&A.",
	Args:  cobra.ArbitraryArgs,
	RunE:  runRoot,
}

var (
	flagChannel string
	flagVideo   string
)

func init() {
	rootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "Skip cache reads for this invocation")
	rootCmd.Flags().StringVar(&flagChannel, "channel", "", "Open directly on a channel URL or handle")
	rootCmd.Flags().StringVar(&flagVideo, "video", "", "Open directly on a video URL")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(summaryCmd)
	rootCmd.AddCommand(transcriptCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(cacheCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

func runRoot(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := checkDependencies(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger := setupLogger(cfg)

	a := app.New(cfg, logger, version, noCache)

	switch {
	case flagChannel != "":
		a.SetInitialMode(app.ModeChannel, flagChannel)
	case flagVideo != "":
		a.SetInitialMode(app.ModeVideo, flagVideo)
	case len(args) > 0:
		query := args[0]
		for _, arg := range args[1:] {
			query += " " + arg
		}
		a.SetInitialMode(app.ModeSearch, query)
	}

	return a.Run()
}

func checkDependencies(cfg *config.Config) error {
	if _, err := exec.LookPath(cfg.YtDlp.Binary); err != nil {
		return fmt.Errorf(
			"yt-dlp not found in PATH\n\nInstall with:\n  brew install yt-dlp\n  pip install yt-dlp",
		)
	}
	if _, err := exec.LookPath(cfg.Summarize.Binary); err != nil {
		return fmt.Errorf(
			"summarize not found in PATH\n\nInstall with:\n  npm i -g @steipete/summarize",
		)
	}
	return nil
}

func setupLogger(cfg *config.Config) zerolog.Logger {
	logPath := config.DataDir() + "/ytcap.log"
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot open log file: %v\n", err)
		return zerolog.Nop()
	}

	logger := zerolog.New(logFile).With().Timestamp().Logger()
	if os.Getenv("YTCAP_LOG") == "debug" {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	return logger
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print ytcap version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ytcap %s\n", version)
	},
}

var summaryCmd = &cobra.Command{
	Use:   "summary <url>",
	Short: "Print video summary to stdout",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		logger := setupLogger(cfg)
		s := newSummarizeClient(cfg, logger)
		result, err := s.GetSummary(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		fmt.Print(result)
		return nil
	},
}

var transcriptCmd = &cobra.Command{
	Use:   "transcript <url>",
	Short: "Print video transcript to stdout",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		logger := setupLogger(cfg)
		s := newSummarizeClient(cfg, logger)
		result, err := s.GetTranscript(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		fmt.Print(result)
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ytcap configuration",
}

func init() {
	configCmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Print current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return config.Print(cfg)
		},
	})
	configCmd.AddCommand(&cobra.Command{
		Use:   "edit",
		Short: "Open config in $EDITOR (default: vim)",
		RunE: func(cmd *cobra.Command, args []string) error {
			editor := "vi"
			p := config.FilePath()
			c := exec.Command(editor, p)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	})
	configCmd.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "Reset config to defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print("Reset config to defaults? [y/N] ")
			var answer string
			_, _ = fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" {
				return nil
			}
			return config.Reset()
		},
	})
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage ytcap cache",
}

func init() {
	cacheCmd.AddCommand(&cobra.Command{
		Use:   "clear [video-id]",
		Short: "Clear cached data",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheDir := config.DataDir() + "/cache"
			if len(args) > 0 {
				dir := cacheDir + "/" + args[0]
				fmt.Printf("Clearing cache for %s\n", args[0])
				return os.RemoveAll(dir)
			}
			fmt.Println("Clearing all cache")
			return os.RemoveAll(cacheDir)
		},
	})
}
