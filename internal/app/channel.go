package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/yetmike/ytcap/internal/ytdlp"
)

const (
	initialPageSize    = 20
	backgroundPageSize = 80 // 20 + 80 = 100 total before pausing
	manualLoadSize     = 50
)

type sortOrder int

const (
	sortDateDesc sortOrder = iota
	sortDateAsc
	sortDuration
	sortViews
	sortCount // must be last
)

var sortLabels = map[sortOrder]string{
	sortDateDesc: "Newest first",
	sortDateAsc:  "Oldest first",
	sortDuration: "Duration",
	sortViews:    "Views",
}

type ChannelScreen struct {
	app           *App
	flex          *tview.Flex
	table         *tview.Table
	status        *tview.TextView
	channelURL    string
	channelName   string // display name e.g. "Fireship"
	channelHandle string // e.g. "@fireship"
	videos        []ytdlp.Video
	cancelFn    context.CancelFunc
	isLoading   bool
	sortBy      sortOrder
	nextStart   int  // 1-based next page start
	hasMore     bool // whether more pages might exist
	totalVideos int  // total videos on channel, 0 = unknown
}

func NewChannelScreen(a *App, channelURL, channelName string) *ChannelScreen {
	// Extract handle from URL
	handle := ""
	for _, prefix := range []string{"https://www.youtube.com/", "http://www.youtube.com/"} {
		if len(channelURL) > len(prefix) {
			rest := channelURL[len(prefix):]
			for i, ch := range rest {
				if ch == '/' {
					rest = rest[:i]
					break
				}
			}
			if len(rest) > 0 && rest[0] == '@' {
				handle = rest
			}
		}
	}

	// If channelName is actually a handle, move it
	if len(channelName) > 0 && channelName[0] == '@' {
		handle = channelName
		channelName = "" // will be filled from first video
	}

	c := &ChannelScreen{
		app:           a,
		channelURL:    channelURL,
		channelName:   channelName,
		channelHandle: handle,
		sortBy:        sortDateDesc,
		nextStart:     1,
		hasMore:       true,
	}

	// Restore from session cache if available
	if cached, ok := a.sessionChannels[channelURL]; ok {
		c.videos = cached.Videos
		c.totalVideos = cached.TotalVideos
		if cached.ChannelName != "" {
			c.channelName = cached.ChannelName
		}
		if cached.Handle != "" {
			c.channelHandle = cached.Handle
		}
		c.nextStart = len(cached.Videos) + 1
		c.hasMore = c.totalVideos == 0 || len(cached.Videos) < c.totalVideos
		c.build()
		c.renderVideos()
		c.updateStatus()
		return c
	}

	c.build()
	c.loadInitial()
	c.fetchTotal()
	return c
}

func (c *ChannelScreen) build() {
	c.table = tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 0)

	c.table.SetBackgroundColor(c.app.Theme.Color("background"))
	c.table.SetSelectedStyle(tcell.StyleDefault.
		Background(c.app.Theme.Color("selected_bg")).
		Foreground(c.app.Theme.Color("selected_fg")))

	c.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	c.status.SetBackgroundColor(c.app.Theme.Color("header_bg"))

	c.setHeaders()

	c.table.SetSelectedFunc(func(row, col int) {
		idx := row - 1
		if idx < 0 || idx >= len(c.videos) {
			return
		}
		v := c.videos[idx]
		vs := NewVideoScreen(c.app, v.URL, &v)
		c.app.PushScreen(v.Title, vs)
	})

	c.table.SetSelectionChangedFunc(func(row, col int) {
		if row >= c.table.GetRowCount()-3 && !c.isLoading && c.hasMore {
			c.loadMore()
		}
	})

	c.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() != tcell.KeyRune {
			return event
		}
		switch event.Rune() {
		case 's':
			c.cycleSort()
			return nil
		case 'l', 'L':
			if c.hasMore && !c.isLoading {
				c.loadMore()
			}
			return nil
		case '/':
			c.fzfFilter()
			return nil
		}
		return event
	})

	c.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(c.status, 1, 0, false).
		AddItem(c.table, 0, 1, true)
}

func (c *ChannelScreen) saveToSession() {
	c.app.sessionChannels[c.channelURL] = &SessionChannel{
		Videos:      c.videos,
		ChannelName: c.channelName,
		Handle:      c.channelHandle,
		TotalVideos: c.totalVideos,
	}
}

func (c *ChannelScreen) setHeaders() {
	headers := []string{"#", "TITLE", "DURATION", "VIEWS"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(c.app.Theme.Color("header_fg")).
			SetSelectable(false)
		if i == 1 {
			cell.SetExpansion(1)
		}
		c.table.SetCell(0, i, cell)
	}
}

func (c *ChannelScreen) cycleSort() {
	c.sortBy = (c.sortBy + 1) % sortCount
	c.sortVideos()
	c.renderVideos()
	c.updateStatus()
}

func (c *ChannelScreen) sortVideos() {
	switch c.sortBy {
	case sortDateDesc:
		// YouTube returns newest first; PlaylistIndex preserves that order
		sort.SliceStable(c.videos, func(i, j int) bool {
			return c.videos[i].PlaylistIndex < c.videos[j].PlaylistIndex
		})
	case sortDateAsc:
		sort.SliceStable(c.videos, func(i, j int) bool {
			return c.videos[i].PlaylistIndex > c.videos[j].PlaylistIndex
		})
	case sortDuration:
		sort.SliceStable(c.videos, func(i, j int) bool {
			return c.videos[i].Duration > c.videos[j].Duration
		})
	case sortViews:
		sort.SliceStable(c.videos, func(i, j int) bool {
			return c.videos[i].ViewCount > c.videos[j].ViewCount
		})
	}
}

func (c *ChannelScreen) updateStatus() {
	loaded := len(c.videos)
	sortLabel := sortLabels[c.sortBy]

	videoCount := fmt.Sprintf("%d", loaded)
	if c.totalVideos > 0 {
		videoCount = fmt.Sprintf("%d / %d", loaded, c.totalVideos)
	}

	loadStatus := ""
	if c.hasMore {
		if c.isLoading {
			loadStatus = "  |  [" + c.app.Theme.Colors["status_loading"] + "]loading...[-]"
		}
	} else {
		loadStatus = "  |  [" + c.app.Theme.Colors["status_ok"] + "]all loaded[-]"
	}

	displayName := c.channelName
	if c.channelHandle != "" {
		if displayName != "" {
			displayName += "  |  " + c.channelHandle
		} else {
			displayName = c.channelHandle
		}
	}

	c.status.SetText(fmt.Sprintf(
		"[%s]%s[-]  |  [%s]%s videos[-]  |  [%s]Sort: %s[-]%s",
		c.app.Theme.Colors["channel_name"], displayName,
		c.app.Theme.Colors["footer_fg"], videoCount,
		c.app.Theme.Colors["key_hint"], sortLabel,
		loadStatus,
	))

	// Persist to session cache
	c.saveToSession()
}

func (c *ChannelScreen) fetchTotal() {
	go func() {
		total, err := c.app.YtDlp.CountChannelVideos(context.Background(), c.channelURL)
		c.app.TView.QueueUpdateDraw(func() {
			if err != nil {
				return // silently ignore, not critical
			}
			c.totalVideos = total
			c.updateStatus()
		})
	}()
}

// loadInitial fetches the first page quickly, then continues loading in background.
func (c *ChannelScreen) loadInitial() {
	c.isLoading = true
	c.status.SetText("[" + c.app.Theme.Colors["status_loading"] + "]Loading channel videos...[-]")

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFn = cancel

	go func() {
		videos, err := c.app.YtDlp.FetchChannelPage(ctx, c.channelURL, 1, initialPageSize)
		c.app.TView.QueueUpdateDraw(func() {
			c.isLoading = false
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if len(c.videos) == 0 {
					c.app.PopScreen()
					c.app.SetStatusLong("Channel not found: " + c.channelName)
					return
				}
				return
			}
			if len(videos) == 0 {
				c.app.PopScreen()
				c.app.SetStatusLong("No videos found for " + c.channelName)
				return
			}
			if len(videos) < initialPageSize {
				c.hasMore = false
			}
			// Fill in channel name from first video if we only have a handle
			if len(videos) > 0 && videos[0].Channel != "" {
				if c.channelName == "" {
					c.channelName = videos[0].Channel
				}
				if c.channelHandle == "" {
					// Try to get handle from uploader_id or channel URL
					if videos[0].ChannelURL != "" {
						for _, prefix := range []string{"https://www.youtube.com/", "http://www.youtube.com/"} {
							if len(videos[0].ChannelURL) > len(prefix) {
								rest := videos[0].ChannelURL[len(prefix):]
								for i, ch := range rest {
									if ch == '/' {
										rest = rest[:i]
										break
									}
								}
								if len(rest) > 0 && rest[0] == '@' {
									c.channelHandle = rest
								}
							}
						}
					}
				}
			}
			c.videos = videos
			c.nextStart = 1 + len(videos)
			c.sortVideos()
			c.renderVideos()
			c.updateStatus()

			// Auto-load more in background
			if c.hasMore {
				c.backgroundLoad(ctx)
			}
		})
	}()
}

// backgroundLoad fetches one more page silently to reach ~100 videos total.
func (c *ChannelScreen) backgroundLoad(ctx context.Context) {
	go func() {
		start := c.nextStart
		videos, err := c.app.YtDlp.FetchChannelPage(ctx, c.channelURL, start, backgroundPageSize)
		if err != nil || ctx.Err() != nil {
			return
		}
		c.app.TView.QueueUpdateDraw(func() {
			if len(videos) < backgroundPageSize {
				c.hasMore = false
			}
			selectedID := c.selectedID()
			rowOffset, colOffset := c.table.GetOffset()
			c.videos = append(c.videos, videos...)
			c.nextStart = start + len(videos)
			c.sortVideos()
			c.renderVideos()
			c.selectByID(selectedID)
			c.table.SetOffset(rowOffset, colOffset)
			c.updateStatus()
		})
	}()
}

// selectedID returns the ID of the currently selected video, or "" if none.
func (c *ChannelScreen) selectedID() string {
	row, _ := c.table.GetSelection()
	idx := row - 1
	if idx < 0 || idx >= len(c.videos) {
		return ""
	}
	return c.videos[idx].ID
}

// selectByID restores the selection to the row whose video has the given ID.
// Falls back to row 1 if the ID is empty or no longer present.
func (c *ChannelScreen) selectByID(id string) {
	if id != "" {
		for i, v := range c.videos {
			if v.ID == id {
				c.table.Select(i+1, 0)
				return
			}
		}
	}
	c.table.Select(1, 0)
}

func (c *ChannelScreen) loadMore() {
	if c.isLoading {
		return
	}
	c.isLoading = true

	if len(c.videos) == 0 {
		c.status.SetText("[" + c.app.Theme.Colors["status_loading"] + "]Loading channel videos...[-]")
	} else {
		c.status.SetText("[" + c.app.Theme.Colors["status_loading"] + "]Loading more videos...[-]")
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFn = cancel
	start := c.nextStart

	go func() {
		count := manualLoadSize
		if start == 1 {
			count = initialPageSize
		}
		videos, err := c.app.YtDlp.FetchChannelPage(ctx, c.channelURL, start, count)
		c.app.TView.QueueUpdateDraw(func() {
			c.isLoading = false
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if len(c.videos) == 0 {
					// First load failed — channel doesn't exist, pop back
					c.app.PopScreen()
					c.app.SetStatusLong("Channel not found: " + c.channelName)
					return
				}
				c.status.SetText("[" + c.app.Theme.Colors["status_error"] + "]Error: " + err.Error() + "[-]")
				return
			}
			if len(videos) == 0 && len(c.videos) == 0 {
				c.app.PopScreen()
				c.app.SetStatusLong("No videos found for " + c.channelName)
				return
			}
			if len(videos) < count {
				c.hasMore = false
			}
			selectedID := c.selectedID()
			rowOffset, colOffset := c.table.GetOffset()
			c.videos = append(c.videos, videos...)
			c.nextStart = start + len(videos)
			c.sortVideos()
			c.renderVideos()
			c.selectByID(selectedID)
			c.table.SetOffset(rowOffset, colOffset)
			c.updateStatus()
		})
	}()
}

func (c *ChannelScreen) fzfFilter() {
	if len(c.videos) == 0 {
		return
	}

	var items []string
	for _, v := range c.videos {
		label := v.Title + "  |  " + v.DurationStr
		if v.ViewCount > 0 {
			label += "  |  " + formatViews(v.ViewCount)
		}
		items = append(items, label)
	}

	c.app.fzfPickAsync(items, " Filter > ", fmt.Sprintf("%s (%d videos)", c.channelName, len(c.videos)), func(index int) {
		if index < 0 || index >= len(c.videos) {
			return
		}
		v := c.videos[index]
		vs := NewVideoScreen(c.app, v.URL, &v)
		c.app.PushScreen(v.Title, vs)
	})
}

func (c *ChannelScreen) renderVideos() {
	for row := c.table.GetRowCount() - 1; row > 0; row-- {
		c.table.RemoveRow(row)
	}

	for i, v := range c.videos {
		row := i + 1
		c.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", i+1)).
			SetTextColor(c.app.Theme.Color("video_views")))
		c.table.SetCell(row, 1, tview.NewTableCell(v.Title).
			SetTextColor(c.app.Theme.Color("video_title")).SetExpansion(1))
		c.table.SetCell(row, 2, tview.NewTableCell(v.DurationStr).
			SetTextColor(c.app.Theme.Color("video_duration")))

		views := ""
		if v.ViewCount > 0 {
			views = formatViews(v.ViewCount)
		}
		c.table.SetCell(row, 3, tview.NewTableCell(views).
			SetTextColor(c.app.Theme.Color("video_views")))
	}
}

func (c *ChannelScreen) Primitive() tview.Primitive { return c.flex }
func (c *ChannelScreen) OnShow() {
	c.app.updateFooter("<j/k> navigate  <Enter> open  </> filter  <s> sort  <l> load more  <Esc> back  <q> quit")
	c.app.TView.SetFocus(c.table)
}
func (c *ChannelScreen) OnHide() {}
func (c *ChannelScreen) Cancel() {
	if c.cancelFn != nil {
		c.cancelFn()
	}
}
