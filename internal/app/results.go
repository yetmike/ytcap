package app

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/yetmike/ytcap/internal/ytdlp"
)

const (
	searchInitialLimit    = 20
	searchBackgroundLimit = 100 // passive background load target
)

type ResultsScreen struct {
	app       *App
	flex      *tview.Flex
	table     *tview.Table
	status    *tview.TextView
	results   []ytdlp.SearchResult
	query     string
	cancelFn  context.CancelFunc
	isLoading bool
	hasMore   bool
	loadCount int // how many results we've requested so far
}

func NewResultsScreen(a *App, results []ytdlp.SearchResult, query string) *ResultsScreen {
	r := &ResultsScreen{
		app:       a,
		results:   results,
		query:     query,
		loadCount: len(results),
		hasMore:   len(results) >= a.Config.YtDlp.SearchLimit,
	}
	r.build()
	return r
}

func (r *ResultsScreen) build() {
	r.table = tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 0)

	r.table.SetBackgroundColor(r.app.Theme.Color("background"))
	r.table.SetSelectedStyle(tcell.StyleDefault.
		Background(r.app.Theme.Color("selected_bg")).
		Foreground(r.app.Theme.Color("selected_fg")))

	r.setHeaders()
	r.renderResults()

	// Enter opens the video, or the channel if the row is a channel result
	r.table.SetSelectedFunc(func(row, col int) {
		idx := row - 1
		if idx < 0 || idx >= len(r.results) {
			return
		}
		res := r.results[idx]
		if res.Type == "channel" {
			cs := NewChannelScreen(r.app, res.URL, res.Title)
			r.app.PushScreen(res.Title, cs)
			return
		}
		vs := NewVideoScreen(r.app, res.URL, nil)
		r.app.PushScreen(res.Title, vs)
	})

	r.table.SetSelectionChangedFunc(func(row, col int) {
		// Auto-load more when user scrolls near the bottom
		if row >= r.table.GetRowCount()-3 && !r.isLoading && r.hasMore {
			r.loadMore()
		}
	})

	r.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() != tcell.KeyRune {
			return event
		}
		switch event.Rune() {
		case 'c':
			row, _ := r.table.GetSelection()
			idx := row - 1
			if idx < 0 || idx >= len(r.results) {
				return nil
			}
			res := r.results[idx]
			if res.ChannelURL == "" {
				return nil
			}
			cs := NewChannelScreen(r.app, res.ChannelURL, res.Channel)
			r.app.PushScreen(res.Channel, cs)
			return nil
		case '/':
			r.fzfFilter()
			return nil
		case 'l', 'L':
			r.loadMore()
			return nil
		}
		return event
	})

	r.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	r.status.SetBackgroundColor(r.app.Theme.Color("header_bg"))

	r.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(r.status, 1, 0, false).
		AddItem(r.table, 0, 1, true)

	r.updateStatus()

	// Passively load more results in background
	if r.hasMore {
		r.backgroundLoad()
	}
}

func (r *ResultsScreen) setHeaders() {
	headers := []string{"#", "TITLE", "CHANNEL", "VIEWS", "DURATION"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(r.app.Theme.Color("header_fg")).
			SetSelectable(false)
		if i == 1 {
			cell.SetExpansion(1)
		}
		r.table.SetCell(0, i, cell)
	}
}

func (r *ResultsScreen) renderResults() {
	// Clear rows after header
	for row := r.table.GetRowCount() - 1; row > 0; row-- {
		r.table.RemoveRow(row)
	}

	for i, res := range r.results {
		row := i + 1
		r.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(r.app.Theme.Color("video_views")))
		r.table.SetCell(row, 1, tview.NewTableCell(res.Title).
			SetTextColor(r.app.Theme.Color("video_title")).SetExpansion(1))
		r.table.SetCell(row, 2, tview.NewTableCell(res.Channel).
			SetTextColor(r.app.Theme.Color("channel_name")))

		views := ""
		if res.ViewCount > 0 {
			views = formatViews(res.ViewCount)
		}
		r.table.SetCell(row, 3, tview.NewTableCell(views).
			SetTextColor(r.app.Theme.Color("video_views")))
		r.table.SetCell(row, 4, tview.NewTableCell(res.DurationStr).
			SetTextColor(r.app.Theme.Color("video_duration")))
	}
}

func (r *ResultsScreen) updateStatus() {
	count := fmt.Sprintf("%d results", len(r.results))

	loadStatus := ""
	if r.hasMore {
		if r.isLoading {
			loadStatus = "  |  [" + r.app.Theme.Colors["status_loading"] + "]loading...[-]"
		}
	} else {
		loadStatus = "  |  [" + r.app.Theme.Colors["status_ok"] + "]all loaded[-]"
	}

	r.status.SetText(fmt.Sprintf(
		"[%s]Search: %s[-]  |  [%s]%s[-]%s",
		r.app.Theme.Colors["channel_name"], r.query,
		r.app.Theme.Colors["footer_fg"], count,
		loadStatus,
	))
}

// backgroundLoad passively fetches more results up to searchBackgroundLimit.
func (r *ResultsScreen) backgroundLoad() {
	r.isLoading = true
	r.updateStatus()

	ctx, cancel := context.WithCancel(context.Background())
	r.cancelFn = cancel

	go func() {
		results, err := r.app.YtDlp.Search(ctx, r.query, searchBackgroundLimit)
		r.app.TView.QueueUpdateDraw(func() {
			r.isLoading = false
			if err != nil || ctx.Err() != nil {
				r.updateStatus()
				return
			}
			selectedRow, _ := r.table.GetSelection()
			rowOffset, colOffset := r.table.GetOffset()
			added := r.mergeResults(results)
			if added == 0 || len(results) < searchBackgroundLimit {
				r.hasMore = false
			}
			r.loadCount = searchBackgroundLimit
			r.app.sessionSearches[r.query] = &SessionSearch{Results: r.results}
			r.appendRows(len(r.results) - added)
			// SetOffset clears tview's internal trackEnd flag (set when the table
			// fit entirely in the viewport) — otherwise the viewport would jump to
			// the end of the now-larger table. Preserve selection & scroll.
			if selectedRow > 0 {
				r.table.Select(selectedRow, 0)
			}
			r.table.SetOffset(rowOffset, colOffset)
			r.updateStatus()
		})
	}()
}

// mergeResults appends entries from incoming that aren't already in r.results
// (keyed by ID, or by URL for entries with empty IDs). Returns how many rows
// were appended. Existing rows keep their positions so the cursor doesn't jump.
func (r *ResultsScreen) mergeResults(incoming []ytdlp.SearchResult) int {
	seen := make(map[string]struct{}, len(r.results))
	for _, res := range r.results {
		seen[resultKey(res)] = struct{}{}
	}
	before := len(r.results)
	for _, res := range incoming {
		if _, ok := seen[resultKey(res)]; ok {
			continue
		}
		seen[resultKey(res)] = struct{}{}
		r.results = append(r.results, res)
	}
	return len(r.results) - before
}

func resultKey(res ytdlp.SearchResult) string {
	if res.ID != "" {
		return res.ID
	}
	return res.URL
}

// appendRows renders rows starting at index `from` in r.results. It does NOT
// touch earlier rows, so the user's selection (by row index) is preserved.
func (r *ResultsScreen) appendRows(from int) {
	for i := from; i < len(r.results); i++ {
		res := r.results[i]
		row := i + 1
		r.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(r.app.Theme.Color("video_views")))
		r.table.SetCell(row, 1, tview.NewTableCell(res.Title).
			SetTextColor(r.app.Theme.Color("video_title")).SetExpansion(1))
		r.table.SetCell(row, 2, tview.NewTableCell(res.Channel).
			SetTextColor(r.app.Theme.Color("channel_name")))

		views := ""
		if res.ViewCount > 0 {
			views = formatViews(res.ViewCount)
		}
		r.table.SetCell(row, 3, tview.NewTableCell(views).
			SetTextColor(r.app.Theme.Color("video_views")))
		r.table.SetCell(row, 4, tview.NewTableCell(res.DurationStr).
			SetTextColor(r.app.Theme.Color("video_duration")))
	}
}

func (r *ResultsScreen) loadMore() {
	if r.isLoading || !r.hasMore {
		return
	}
	r.isLoading = true
	newCount := r.loadCount + r.app.Config.YtDlp.SearchLimit
	r.updateStatus()

	ctx, cancel := context.WithCancel(context.Background())
	r.cancelFn = cancel

	go func() {
		results, err := r.app.YtDlp.Search(ctx, r.query, newCount)
		r.app.TView.QueueUpdateDraw(func() {
			r.isLoading = false
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				r.updateStatus()
				return
			}
			selectedRow, _ := r.table.GetSelection()
			rowOffset, colOffset := r.table.GetOffset()
			added := r.mergeResults(results)
			if added == 0 || len(results) < newCount {
				r.hasMore = false
			}
			r.loadCount = newCount
			r.app.sessionSearches[r.query] = &SessionSearch{Results: r.results}
			r.appendRows(len(r.results) - added)
			if selectedRow > 0 {
				r.table.Select(selectedRow, 0)
			}
			r.table.SetOffset(rowOffset, colOffset)
			r.updateStatus()
		})
	}()
}

func (r *ResultsScreen) fzfFilter() {
	if len(r.results) == 0 {
		return
	}

	var items []string
	for _, res := range r.results {
		label := res.Title + "  |  " + res.Channel
		if res.DurationStr != "" {
			label += "  |  " + res.DurationStr
		}
		if res.ViewCount > 0 {
			label += "  |  " + formatViews(res.ViewCount)
		}
		items = append(items, label)
	}

	r.app.fzfPickAsync(items, " Filter > ", fmt.Sprintf("Search: %s (%d results)", r.query, len(r.results)), func(index int) {
		if index < 0 || index >= len(r.results) {
			return
		}
		res := r.results[index]
		if res.Type == "channel" {
			cs := NewChannelScreen(r.app, res.URL, res.Title)
			r.app.PushScreen(res.Title, cs)
		} else {
			vs := NewVideoScreen(r.app, res.URL, nil)
			r.app.PushScreen(res.Title, vs)
		}
	})
}

func (r *ResultsScreen) Primitive() tview.Primitive { return r.flex }
func (r *ResultsScreen) OnShow() {
	r.app.updateFooter("<j/k> navigate  <Enter> open  <c> channel  </> filter  <l> load more  <Esc> back  <:> search  <q> quit")
	r.app.TView.SetFocus(r.table)
}
func (r *ResultsScreen) OnHide() {}
func (r *ResultsScreen) Cancel() {
	if r.cancelFn != nil {
		r.cancelFn()
	}
}

func formatViews(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB views", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM views", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK views", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d views", n)
	}
}
