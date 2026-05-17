package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/yetmike/ytcap/internal/ytdlp"
)

type VideoScreen struct {
	app        *App
	flex       *tview.Flex
	tabBar     *tview.TextView
	content    *tview.TextView
	metaView   *tview.TextView
	videoURL   string
	video      *ytdlp.Video
	cancelFn   context.CancelFunc
	activeTab  int // 0=info, 1=summary, 2=transcript
	summary    string
	transcript string
	isLoading  bool
	summaryLoading    bool
	transcriptLoading bool
	summaryErr        error
	transcriptErr     error
	chatScreen        *ChatScreen // persisted so chat survives navigation
}

func NewVideoScreen(a *App, videoURL string, video *ytdlp.Video) *VideoScreen {
	v := &VideoScreen{
		app:      a,
		videoURL: videoURL,
		video:    video,
	}

	// Restore from session cache if available
	if cached, ok := a.sessionVideos[videoURL]; ok {
		if cached.Video != nil {
			v.video = cached.Video
		}
		v.summary = cached.Summary
		v.transcript = cached.Transcript
	}

	v.build()
	if v.video != nil {
		v.renderInfo()
		v.prefetch()
	}
	// Always fetch full metadata – flat-playlist results (from channel/search)
	// carry truncated descriptions, so we need a full video lookup.
	v.fetchMetadata()
	return v
}

func (v *VideoScreen) build() {
	v.metaView = tview.NewTextView().
		SetDynamicColors(true)
	v.metaView.SetBackgroundColor(v.app.Theme.Color("background"))

	v.tabBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	v.tabBar.SetBackgroundColor(v.app.Theme.Color("header_bg"))

	v.content = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	v.content.SetBackgroundColor(v.app.Theme.Color("background"))

	v.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v.metaView, 4, 0, false).
		AddItem(v.tabBar, 1, 0, false).
		AddItem(v.content, 0, 1, true)

	v.flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyCtrlC:
			if v.cancelFn != nil {
				v.cancelFn()
				v.isLoading = false
				v.content.SetText("Cancelled.")
			}
			return nil
		case event.Key() == tcell.KeyCtrlS:
			v.saveCurrentTab()
			return nil
		case event.Key() == tcell.KeyEscape:
			if v.cancelFn != nil {
				v.cancelFn()
			}
			v.app.PopScreen()
			return nil
		case event.Key() == tcell.KeyTab:
			v.activeTab = (v.activeTab + 1) % 3
			v.switchTab()
			return nil
		case event.Key() == tcell.KeyRune:
			switch event.Rune() {
			case '1':
				v.activeTab = 0
				v.switchTab()
				return nil
			case '2':
				v.activeTab = 1
				v.switchTab()
				return nil
			case '3':
				v.activeTab = 2
				v.switchTab()
				return nil
			case 'c':
				if v.video != nil {
					if v.chatScreen == nil {
						v.chatScreen = NewChatScreen(v.app, v)
					}
					v.app.PushScreen("Chat", v.chatScreen)
				}
				return nil
			case 'y':
				v.yankCurrentTab()
				return nil
			}
		}
		return event
	})

	v.updateTabBar()
}

func (v *VideoScreen) updateTabBar() {
	type tabInfo struct {
		name    string
		status  string // "", "...", "ok", "err"
	}
	tabs := []tabInfo{
		{"Info", ""},
		{"Summary", v.tabStatus(v.summary, v.summaryLoading, v.summaryErr)},
		{"Transcript", v.tabStatus(v.transcript, v.transcriptLoading, v.transcriptErr)},
	}
	var parts []string
	for i, tab := range tabs {
		label := tab.name
		if tab.status != "" {
			label += " " + tab.status
		}
		if i == v.activeTab {
			parts = append(parts, "["+v.app.Theme.Colors["selected_fg"]+"][ "+label+" ][-]")
		} else {
			parts = append(parts, "["+v.app.Theme.Colors["footer_fg"]+"]  "+label+"  [-]")
		}
	}
	v.tabBar.SetText(strings.Join(parts, ""))
}

func (v *VideoScreen) tabStatus(content string, loading bool, err error) string {
	if content != "" {
		return "[" + v.app.Theme.Colors["status_ok"] + "]ok[-]"
	}
	if err != nil {
		return "[" + v.app.Theme.Colors["status_error"] + "]err[-]"
	}
	if loading {
		return "[" + v.app.Theme.Colors["status_loading"] + "]...[-]"
	}
	return ""
}

// saveToSession persists current video data in the session cache.
func (v *VideoScreen) saveToSession() {
	sv, ok := v.app.sessionVideos[v.videoURL]
	if !ok {
		sv = &SessionVideo{}
		v.app.sessionVideos[v.videoURL] = sv
	}
	if v.video != nil {
		sv.Video = v.video
	}
	if v.summary != "" {
		sv.Summary = v.summary
	}
	if v.transcript != "" {
		sv.Transcript = v.transcript
	}
}

func (v *VideoScreen) switchTab() {
	v.updateTabBar()
	switch v.activeTab {
	case 0:
		v.renderInfo()
	case 1:
		v.renderSummary()
	case 2:
		v.renderTranscript()
	}
}

func (v *VideoScreen) fetchMetadata() {
	ctx, cancel := context.WithCancel(context.Background())
	v.cancelFn = cancel

	go func() {
		video, err := v.app.YtDlp.FetchVideo(ctx, v.videoURL)
		v.app.TView.QueueUpdateDraw(func() {
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				// Only show error if we have no data at all
				if v.video == nil {
					v.content.SetText("[" + v.app.Theme.Colors["status_error"] + "]Error: " + err.Error() + "[-]")
				}
				return
			}
			firstLoad := v.video == nil
			v.video = video
			v.saveToSession()
			v.renderInfo()
			if firstLoad {
				v.prefetch()
			}
		})
	}()
}

// prefetch kicks off summary and transcript fetches in parallel background goroutines.
func (v *VideoScreen) prefetch() {
	// Check caches first (fast, no goroutine needed)
	if v.video != nil {
		if cached, ok := v.app.Cache.GetSummary(v.video.ID); ok {
			v.summary = cached
		}
		if cached, ok := v.app.Cache.GetTranscript(v.video.ID); ok {
			v.transcript = cached
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	v.cancelFn = cancel

	needSummary := v.summary == ""
	needTranscript := v.transcript == ""

	if needSummary {
		v.summaryLoading = true
	}
	if needTranscript {
		v.transcriptLoading = true
	}
	v.updateTabBar()

	// Serialize summarize calls in a single goroutine to avoid
	// "database is locked" errors from concurrent SQLite access.
	go func() {
		if needTranscript {
			result, err := v.app.Summarize.GetTranscript(ctx, v.videoURL)
			v.app.TView.QueueUpdateDraw(func() {
				v.transcriptLoading = false
				if err != nil {
					if ctx.Err() == nil {
						v.transcriptErr = err
					}
				} else {
					v.transcript = result
					v.saveToSession()
					if v.video != nil {
						_ = v.app.Cache.SetTranscript(v.video.ID, result)
					}
				}
				if v.activeTab == 2 {
					v.renderTranscript()
				}
				v.updateTabBar()
			})
		}

		if needSummary {
			result, err := v.app.Summarize.GetSummary(ctx, v.videoURL)
			v.app.TView.QueueUpdateDraw(func() {
				v.summaryLoading = false
				if err != nil {
					if ctx.Err() == nil {
						v.summaryErr = err
					}
				} else {
					v.summary = result
					v.saveToSession()
					if v.video != nil {
						_ = v.app.Cache.SetSummary(v.video.ID, result)
					}
				}
				if v.activeTab == 1 {
					v.renderSummary()
				}
				v.updateTabBar()
			})
		}
	}()
}

func (v *VideoScreen) renderInfo() {
	if v.video == nil {
		v.content.SetText("Loading...")
		return
	}

	vid := v.video
	date := vid.UploadDate
	if len(date) == 8 {
		date = date[:4] + "-" + date[4:6] + "-" + date[6:8]
	}

	meta := fmt.Sprintf("[%s]%s[-]\n[%s]%s  |  %s  |  %s views  |  %s[-]\n[%s]%s[-]",
		v.app.Theme.Colors["title"], vid.Title,
		v.app.Theme.Colors["footer_fg"], vid.Channel, vid.DurationStr, formatViews(vid.ViewCount), date,
		v.app.Theme.Colors["footer_fg"], vid.URL,
	)
	v.metaView.SetText(meta)

	info := fmt.Sprintf(`[%s]Title:[-]        %s
[%s]Channel:[-]      %s
[%s]Published:[-]    %s
[%s]Duration:[-]     %s
[%s]Views:[-]        %s
[%s]URL:[-]          %s

[%s]Description:[-]
  %s`,
		v.app.Theme.Colors["key_hint"], vid.Title,
		v.app.Theme.Colors["key_hint"], vid.Channel,
		v.app.Theme.Colors["key_hint"], date,
		v.app.Theme.Colors["key_hint"], vid.DurationStr,
		v.app.Theme.Colors["key_hint"], formatViews(vid.ViewCount),
		v.app.Theme.Colors["key_hint"], vid.URL,
		v.app.Theme.Colors["key_hint"],
		vid.Description,
	)
	v.content.SetText(info)
}

func (v *VideoScreen) renderSummary() {
	if v.summary != "" {
		v.content.SetText("[" + v.app.Theme.Colors["summary_text"] + "]" + v.summary + "[-]")
		return
	}
	if v.summaryErr != nil {
		v.content.SetText("[" + v.app.Theme.Colors["status_error"] + "]Error: " + v.summaryErr.Error() + "[-]")
		return
	}
	v.content.SetText("[" + v.app.Theme.Colors["status_loading"] + "]Loading summary...[-]")
}

// cleanTranscript strips noise tags like [MUSIC], [Applause], [Laughter] etc.
// and removes the "Transcript:" header line.
func cleanTranscript(text string) string {
	lines := strings.Split(text, "\n")
	var clean []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip noise tags
		upper := strings.ToUpper(trimmed)
		if upper == "[MUSIC]" || upper == "[APPLAUSE]" || upper == "[LAUGHTER]" ||
			upper == "[SILENCE]" || upper == "[INAUDIBLE]" ||
			strings.HasPrefix(upper, "[MUSIC") || strings.HasPrefix(upper, "[APPLAUSE") {
			continue
		}
		// Skip "Transcript:" header
		if trimmed == "Transcript:" {
			continue
		}
		clean = append(clean, line)
	}
	return strings.Join(clean, "\n")
}

// reflowText joins short lines into paragraphs for full-width display.
// Preserves paragraph breaks (empty lines) and timestamp lines like [0:00].
func reflowText(text string) string {
	lines := strings.Split(text, "\n")
	var result strings.Builder
	var paragraph strings.Builder

	flushParagraph := func() {
		if paragraph.Len() > 0 {
			result.WriteString(strings.TrimSpace(paragraph.String()))
			result.WriteByte('\n')
			paragraph.Reset()
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flushParagraph()
			result.WriteByte('\n')
			continue
		}
		// Preserve lines that look like headers, timestamps, or labels
		if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "Transcript:") || strings.HasSuffix(trimmed, ":") {
			flushParagraph()
			result.WriteString(trimmed)
			result.WriteByte('\n')
			continue
		}
		if paragraph.Len() > 0 {
			paragraph.WriteByte(' ')
		}
		paragraph.WriteString(trimmed)
	}
	flushParagraph()
	return result.String()
}

func (v *VideoScreen) renderTranscript() {
	if v.transcript != "" {
		reflowed := reflowText(cleanTranscript(v.transcript))
		v.content.SetText("[" + v.app.Theme.Colors["summary_text"] + "]" + reflowed + "[-]")
		return
	}
	if v.transcriptErr != nil {
		v.content.SetText("[" + v.app.Theme.Colors["status_error"] + "]Error: " + v.transcriptErr.Error() + "[-]")
		return
	}
	v.content.SetText("[" + v.app.Theme.Colors["status_loading"] + "]Loading transcript...[-]")
}

func (v *VideoScreen) saveCurrentTab() {
	switch v.activeTab {
	case 1:
		v.saveSummary()
	case 2:
		v.saveTranscript()
	default:
		v.app.SetStatus("Switch to Summary or Transcript tab to save")
	}
}

func (v *VideoScreen) saveSummary() {
	if v.video == nil {
		v.app.SetStatus("No video loaded")
		return
	}
	if v.summary == "" {
		if v.summaryLoading {
			v.app.SetStatus("Summary still loading...")
		} else {
			v.app.SetStatus("No summary available")
		}
		return
	}
	vid := v.video
	date := vid.UploadDate
	if len(date) == 8 {
		date = date[:4] + "-" + date[4:6] + "-" + date[6:8]
	}
	defaultPath := v.app.Storage.DefaultSummaryPath(vid.ID, vid.Title)
	v.app.showSavePrompt(defaultPath, func(userPath string) {
		path, err := v.app.Storage.SaveSummaryTo(userPath, vid.ID, vid.Title, vid.Channel, vid.URL, vid.DurationStr, date, v.summary)
		if err != nil {
			v.app.SetStatus("Error saving: " + err.Error())
			return
		}
		v.app.CopyToClipboard(path)
		v.app.SetStatusLong("Saved to " + path + "  (path copied)")
	})
}

func (v *VideoScreen) saveTranscript() {
	if v.video == nil {
		v.app.SetStatus("No video loaded")
		return
	}
	if v.transcript == "" {
		if v.transcriptLoading {
			v.app.SetStatus("Transcript still loading...")
		} else {
			v.app.SetStatus("No transcript available")
		}
		return
	}
	vid := v.video
	date := vid.UploadDate
	if len(date) == 8 {
		date = date[:4] + "-" + date[4:6] + "-" + date[6:8]
	}
	defaultPath := v.app.Storage.DefaultTranscriptPath(vid.ID, vid.Title)
	v.app.showSavePrompt(defaultPath, func(userPath string) {
		path, err := v.app.Storage.SaveTranscriptTo(userPath, vid.ID, vid.Title, vid.Channel, vid.URL, vid.DurationStr, date, reflowText(cleanTranscript(v.transcript)))
		if err != nil {
			v.app.SetStatus("Error saving: " + err.Error())
			return
		}
		v.app.CopyToClipboard(path)
		v.app.SetStatusLong("Saved to " + path + "  (path copied)")
	})
}

func (v *VideoScreen) yankCurrentTab() {
	switch v.activeTab {
	case 0:
		// Info tab — yank URL
		v.app.CopyToClipboard(v.videoURL)
		v.app.SetStatus("Yanked URL")
	case 1:
		if v.summary == "" {
			v.app.SetStatus("No summary to yank")
			return
		}
		v.app.CopyToClipboard(v.summary)
		v.app.SetStatus("Yanked summary")
	case 2:
		if v.transcript == "" {
			v.app.SetStatus("No transcript to yank")
			return
		}
		v.app.CopyToClipboard(reflowText(cleanTranscript(v.transcript)))
		v.app.SetStatus("Yanked transcript")
	}
}

func (v *VideoScreen) GetTranscript() string  { return v.transcript }
func (v *VideoScreen) GetVideo() *ytdlp.Video { return v.video }
func (v *VideoScreen) GetVideoURL() string     { return v.videoURL }

func (v *VideoScreen) Primitive() tview.Primitive { return v.flex }
func (v *VideoScreen) OnShow() {
	v.app.updateFooter("<Tab> tabs  <1-3> jump  <c> chat  <Ctrl+S> save  <y> yank  <Ctrl+C> cancel  <Esc> back")
	v.app.TView.SetFocus(v.flex)
}
func (v *VideoScreen) OnHide() {}
func (v *VideoScreen) Cancel() {
	if v.cancelFn != nil {
		v.cancelFn()
	}
}
