package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/yetmike/ytcap/internal/config"
	"github.com/yetmike/ytcap/internal/ytdlp"
)

var (
	channelSort string
	channelLimit int
	channelJSON bool
)

var channelCmd = &cobra.Command{
	Use:   "channel <url|@handle|handle>",
	Short: "List channel videos (non-interactive)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		logger := setupLogger(cfg)
		client := ytdlp.New(cfg.YtDlp.Binary, logger)

		channelArg := normalizeChannelArg(args[0])

		// When sorting by anything other than default order, fetch a large
		// batch so the sort is meaningful.
		fetchCount := channelLimit
		if channelSort != "latest" {
			fetchCount = 500
		}

		videos, err := client.FetchChannelPage(cmd.Context(), channelArg, 1, fetchCount)
		if err != nil {
			return err
		}

		sortVideos(videos, channelSort)

		if len(videos) > channelLimit {
			videos = videos[:channelLimit]
		}

		if channelJSON {
			return printVideosJSON(cmd, videos)
		}

		printVideoTable(cmd, videos, true)
		return nil
	},
}

func init() {
	channelCmd.Flags().StringVarP(&channelSort, "sort", "s", "latest", "Sort order: latest, oldest, popular, duration")
	channelCmd.Flags().IntVarP(&channelLimit, "limit", "n", 20, "Number of videos to show")
	channelCmd.Flags().BoolVar(&channelJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(channelCmd)
}

func normalizeChannelArg(arg string) string {
	if strings.Contains(arg, "/") {
		return arg
	}
	if !strings.HasPrefix(arg, "@") {
		arg = "@" + arg
	}
	return "https://www.youtube.com/" + arg
}

func sortVideos(videos []ytdlp.Video, mode string) {
	switch mode {
	case "oldest":
		sort.SliceStable(videos, func(i, j int) bool {
			return videos[i].PlaylistIndex > videos[j].PlaylistIndex
		})
	case "duration":
		sort.SliceStable(videos, func(i, j int) bool {
			return videos[i].Duration > videos[j].Duration
		})
	case "popular":
		sort.SliceStable(videos, func(i, j int) bool {
			return videos[i].ViewCount > videos[j].ViewCount
		})
	default: // "latest" — YouTube's natural order
		sort.SliceStable(videos, func(i, j int) bool {
			return videos[i].PlaylistIndex < videos[j].PlaylistIndex
		})
	}
}

func printVideoTable(cmd *cobra.Command, videos []ytdlp.Video, skipChannel bool) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
	if skipChannel {
		fmt.Fprintf(w, "TITLE\tVIEWS\tDURATION\n")
	} else {
		fmt.Fprintf(w, "TITLE\tCHANNEL\tVIEWS\tDURATION\n")
	}
	for _, v := range videos {
		title := v.Title
		if len(title) > 70 {
			title = title[:67] + "..."
		}
		if skipChannel {
			fmt.Fprintf(w, "%s\t%s\t%s\n", title, formatViews(v.ViewCount), v.DurationStr)
		} else {
			channel := v.Channel
			if len(channel) > 25 {
				channel = channel[:22] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", title, channel, formatViews(v.ViewCount), v.DurationStr)
		}
	}
	w.Flush()
}

type jsonVideo struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Channel  string `json:"channel,omitempty"`
	Views    int64  `json:"views"`
	Duration string `json:"duration"`
}

func printVideosJSON(cmd *cobra.Command, videos []ytdlp.Video) error {
	out := make([]jsonVideo, len(videos))
	for i, v := range videos {
		out[i] = jsonVideo{
			Title:    v.Title,
			URL:      v.URL,
			Channel:  v.Channel,
			Views:    v.ViewCount,
			Duration: v.DurationStr,
		}
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func formatViews(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
