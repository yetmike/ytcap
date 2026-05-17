package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/yetmike/ytcap/internal/config"
	"github.com/yetmike/ytcap/internal/ytdlp"
)

var (
	searchSort  string
	searchLimit int
	searchJSON  bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search YouTube videos (non-interactive)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		logger := setupLogger(cfg)
		client := ytdlp.New(cfg.YtDlp.Binary, logger)

		query := args[0]
		for _, a := range args[1:] {
			query += " " + a
		}

		// Fetch more than needed when sorting, since yt-dlp search
		// returns results in YouTube's relevance order.
		fetchCount := searchLimit
		if searchSort != "relevance" {
			if fetchCount < 50 {
				fetchCount = 50
			}
		}

		results, err := client.Search(cmd.Context(), query, fetchCount)
		if err != nil {
			return err
		}

		// Filter to videos only (skip channels)
		var videos []ytdlp.SearchResult
		for _, r := range results {
			if r.Type == "video" {
				videos = append(videos, r)
			}
		}

		sortSearchResults(videos, searchSort)

		if len(videos) > searchLimit {
			videos = videos[:searchLimit]
		}

		if searchJSON {
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

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
		fmt.Fprintf(w, "TITLE\tCHANNEL\tVIEWS\tDURATION\n")
		for _, v := range videos {
			title := v.Title
			if len(title) > 70 {
				title = title[:67] + "..."
			}
			channel := v.Channel
			if len(channel) > 25 {
				channel = channel[:22] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				title,
				channel,
				formatViews(v.ViewCount),
				v.DurationStr,
			)
		}
		w.Flush()
		return nil
	},
}

func init() {
	searchCmd.Flags().StringVarP(&searchSort, "sort", "s", "relevance", "Sort order: relevance, popular, duration")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 20, "Number of results to show")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(searchCmd)
}

func sortSearchResults(results []ytdlp.SearchResult, mode string) {
	switch mode {
	case "popular":
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].ViewCount > results[j].ViewCount
		})
	case "duration":
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Duration > results[j].Duration
		})
	// "relevance" — keep YouTube's default order
	}
}
