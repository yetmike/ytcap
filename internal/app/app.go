package app

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog"

	"github.com/yetmike/ytcap/internal/cache"
	"github.com/yetmike/ytcap/internal/config"
	"github.com/yetmike/ytcap/internal/storage"
	"github.com/yetmike/ytcap/internal/summarize"
	"github.com/yetmike/ytcap/internal/ytdlp"
)

// SessionVideo stores video data in RAM for the duration of the session.
type SessionVideo struct {
	Video      *ytdlp.Video
	Summary    string
	Transcript string
}

// SessionSearch stores search results in RAM.
type SessionSearch struct {
	Results []ytdlp.SearchResult
}

// SessionChannel stores channel video list in RAM.
type SessionChannel struct {
	Videos      []ytdlp.Video
	ChannelName string
	Handle      string
	TotalVideos int
}

type Mode int

const (
	ModeDefault Mode = iota
	ModeSearch
	ModeChannel
	ModeVideo
)

type Screen interface {
	Primitive() tview.Primitive
	OnShow()
	OnHide()
	Cancel()
}

type App struct {
	TView      *tview.Application
	Config     *config.Config
	Logger     zerolog.Logger
	Version    string
	NoCache    bool
	Theme      *Theme
	Cache      *cache.Cache
	Storage    *storage.Storage
	YtDlp      *ytdlp.Client
	Summarize  *summarize.Client
	KeyTracker *KeyTracker

	headerFlex      *tview.Flex
	headerLeft      *tview.TextView
	headerMid       *tview.TextView
	headerRight     *tview.TextView
	footer          *tview.TextView
	lastFooterHints string
	body       *tview.Flex
	breadcrumb []string

	currentScreen  Screen
	screenStack    []Screen
	helpVisible    bool
	savePromptOpen bool
	sessionVideos   map[string]*SessionVideo   // keyed by video URL
	sessionSearches map[string]*SessionSearch  // keyed by query
	sessionChannels map[string]*SessionChannel // keyed by channel URL

	initialMode  Mode
	initialParam string
}

func New(cfg *config.Config, logger zerolog.Logger, version string, noCache bool) *App {
	theme := LoadTheme(cfg.UI.Skin)
	cacheEnabled := cfg.Cache.Enabled && !noCache

	a := &App{
		TView:      tview.NewApplication(),
		Config:     cfg,
		Logger:     logger,
		Version:    version,
		NoCache:    noCache,
		Theme:      theme,
		Cache:      cache.New(cfg.Cache.Dir, cfg.Cache.TTLDays, cacheEnabled),
		Storage:    storage.New(cfg.Storage.Dir),
		YtDlp:      ytdlp.New(cfg.YtDlp.Binary, logger),
		Summarize:  summarize.New(cfg.Summarize.Binary, cfg.Summarize.Length, cfg.Summarize.Language, cfg.Summarize.CLI, logger),
		KeyTracker:      &KeyTracker{},
		sessionVideos:   make(map[string]*SessionVideo),
		sessionSearches: make(map[string]*SessionSearch),
		sessionChannels: make(map[string]*SessionChannel),
	}

	a.setupChrome()
	return a
}

func (a *App) SetInitialMode(mode Mode, param string) {
	a.initialMode = mode
	a.initialParam = param
}

func (a *App) setupChrome() {
	hbg := a.Theme.Color("header_bg")

	a.headerLeft = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	a.headerLeft.SetBackgroundColor(hbg)
	a.headerMid = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	a.headerMid.SetBackgroundColor(hbg)
	a.headerRight = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignRight)
	a.headerRight.SetBackgroundColor(hbg)

	a.headerFlex = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(a.headerLeft, 0, 1, false).
		AddItem(a.headerMid, 0, 2, false).
		AddItem(a.headerRight, 0, 1, false)
	a.headerFlex.SetBackgroundColor(hbg)

	a.footer = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	a.footer.SetBackgroundColor(a.Theme.Color("footer_bg"))

	a.body = tview.NewFlex().SetDirection(tview.FlexRow)
	a.body.SetBackgroundColor(a.Theme.Color("background"))

	a.updateHeader()
	a.updateFooter("")
}

func (a *App) updateHeader() {
	bc := ""
	for i, s := range a.breadcrumb {
		if i > 0 {
			bc += " > "
		}
		bc += s
	}

	cli := a.Summarize.CLI
	if cli == "" {
		cli = "auto"
	}

	titleColor := a.Theme.Colors["title"]
	headerColor := a.Theme.Colors["header_fg"]
	cliColor := a.Theme.Colors["channel_name"]
	kh := a.Theme.Colors["key_hint"]

	ver := a.Version
	// Only show version if it looks like a proper semver tag
	if ver == "" || ver == "dev" || strings.Contains(ver, "-dirty") || len(ver) < 4 {
		ver = ""
	}
	if ver != "" {
		ver = " " + ver
	}
	a.headerLeft.SetText(" [" + titleColor + "]ytcap" + ver + "[-]")
	a.headerMid.SetText("[" + headerColor + "]" + bc + "[-]")
	a.headerRight.SetText("[" + cliColor + "]AI:" + cli + "[-]  [" + kh + "]B[-] switch  [" + kh + "]?[-] help ")
}

var cliBackends = []string{"auto", "claude", "gemini"}

func (a *App) CycleBackend() {
	current := a.Summarize.CLI
	if current == "" {
		current = "auto"
	}
	next := cliBackends[0]
	for i, b := range cliBackends {
		if b == current {
			next = cliBackends[(i+1)%len(cliBackends)]
			break
		}
	}
	a.Summarize.CLI = next
	a.Config.Summarize.CLI = next
	a.updateHeader()
}

func (a *App) updateFooter(hints string) {
	a.lastFooterHints = hints
	footerColor := a.Theme.Colors["footer_fg"]
	a.footer.Clear()
	a.footer.SetText("[" + footerColor + "]" + hints + "[-]")
}

// SetStatus shows a temporary status message in the footer for 2 seconds.
func (a *App) SetStatus(status string) {
	a.setStatusDuration(status, 2*time.Second)
}

// SetStatusLong shows a temporary status message for 4 seconds.
func (a *App) SetStatusLong(status string) {
	a.setStatusDuration(status, 4*time.Second)
}

func (a *App) setStatusDuration(status string, d time.Duration) {
	statusColor := a.Theme.Colors["status_ok"]
	a.footer.Clear()
	a.footer.SetText("[" + statusColor + "]" + status + "[-]")

	go func() {
		time.Sleep(d)
		a.TView.QueueUpdateDraw(func() {
			a.updateFooter(a.lastFooterHints)
		})
	}()
}

func (a *App) PushScreen(name string, screen Screen) {
	if a.currentScreen != nil {
		a.currentScreen.OnHide()
		a.screenStack = append(a.screenStack, a.currentScreen)
	}
	a.breadcrumb = append(a.breadcrumb, name)
	a.currentScreen = screen
	a.showScreen(screen)
	screen.OnShow()
}

func (a *App) PopScreen() {
	if len(a.screenStack) == 0 {
		return
	}

	if a.currentScreen != nil {
		a.currentScreen.OnHide()
		a.currentScreen.Cancel()
	}

	a.currentScreen = a.screenStack[len(a.screenStack)-1]
	a.screenStack = a.screenStack[:len(a.screenStack)-1]

	if len(a.breadcrumb) > 0 {
		a.breadcrumb = a.breadcrumb[:len(a.breadcrumb)-1]
	}

	a.showScreen(a.currentScreen)
	a.currentScreen.OnShow()
}

func (a *App) showScreen(screen Screen) {
	a.body.Clear()
	a.body.AddItem(a.headerFlex, 1, 0, false)
	a.body.AddItem(screen.Primitive(), 0, 1, true)
	a.body.AddItem(a.footer, 1, 0, false)

	a.updateHeader()
	a.TView.SetRoot(a.body, true)
}

func (a *App) toggleHelp() {
	if a.helpVisible {
		a.helpVisible = false
		a.showScreen(a.currentScreen)
		a.currentScreen.OnShow()
		return
	}
	a.helpVisible = true

	help := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	help.SetBackgroundColor(a.Theme.Color("background"))
	help.SetTitle(" Help — press ? to close ").
		SetBorder(true).
		SetBorderColor(a.Theme.Color("border_focus"))

	kh := a.Theme.Colors["key_hint"]
	fg := a.Theme.Colors["foreground"]
	title := a.Theme.Colors["title"]

	text := "[" + title + "]Global[-]\n" +
		"  [" + kh + "]q[-]         [" + fg + "]Quit[-]\n" +
		"  [" + kh + "]Esc[-]       [" + fg + "]Go back[-]\n" +
		"  [" + kh + "]:[-]         [" + fg + "]Jump to Search[-]\n" +
		"  [" + kh + "]?[-]         [" + fg + "]Toggle this help[-]\n" +
		"  [" + kh + "]B[-]         [" + fg + "]Cycle AI backend (auto/claude/gemini)[-]\n\n" +
		"[" + title + "]Search[-]\n" +
		"  [" + kh + "]Enter[-]     [" + fg + "]Run search[-]\n\n" +
		"[" + title + "]Results[-]\n" +
		"  [" + kh + "]j/k[-]       [" + fg + "]Navigate[-]\n" +
		"  [" + kh + "]Enter[-]     [" + fg + "]Open video[-]\n" +
		"  [" + kh + "]c[-]         [" + fg + "]Open channel[-]\n\n" +
		"[" + title + "]Channel[-]\n" +
		"  [" + kh + "]j/k[-]       [" + fg + "]Navigate[-]\n" +
		"  [" + kh + "]Enter[-]     [" + fg + "]Open video[-]\n" +
		"  [" + kh + "]s[-]         [" + fg + "]Cycle sort[-]\n" +
		"  [" + kh + "]l[-]         [" + fg + "]Load more videos[-]\n" +
		"  [" + kh + "]r[-]         [" + fg + "]Refresh[-]\n\n" +
		"[" + title + "]Video[-]\n" +
		"  [" + kh + "]Tab[-]       [" + fg + "]Cycle tabs (Info/Summary/Transcript)[-]\n" +
		"  [" + kh + "]1/2/3[-]     [" + fg + "]Jump to tab[-]\n" +
		"  [" + kh + "]c[-]         [" + fg + "]Open chat[-]\n" +
		"  [" + kh + "]Ctrl+S[-]    [" + fg + "]Save current tab[-]\n" +
		"  [" + kh + "]y[-]         [" + fg + "]Yank (copy) URL[-]\n" +
		"  [" + kh + "]Ctrl+C[-]    [" + fg + "]Cancel loading[-]\n\n" +
		"[" + title + "]Chat[-]\n" +
		"  [" + kh + "]Enter[-]     [" + fg + "]Send message[-]\n" +
		"  [" + kh + "]Tab[-]       [" + fg + "]Toggle scroll/input[-]\n" +
		"  [" + kh + "]Ctrl+L[-]    [" + fg + "]Clear conversation[-]\n" +
		"  [" + kh + "]Ctrl+C[-]    [" + fg + "]Cancel request[-]\n" +
		"  [" + kh + "]C[-]         [" + fg + "]Save conversation[-]\n"

	help.SetText(text)

	a.body.Clear()
	a.body.AddItem(a.headerFlex, 1, 0, false)
	a.body.AddItem(help, 0, 1, true)
	a.body.AddItem(a.footer, 1, 0, false)
	a.TView.SetRoot(a.body, true)
	a.TView.SetFocus(help)
}

func (a *App) Run() error {
	a.TView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Save prompt handles its own keys
		if a.savePromptOpen {
			return event
		}

		// Check if current screen has an active input field
		inputActive := false
		if a.currentScreen != nil {
			if ia, ok := a.currentScreen.(interface{ IsInputActive() bool }); ok {
				inputActive = ia.IsInputActive()
			}
		}

		switch {
		case event.Key() == tcell.KeyRune && event.Rune() == 'q':
			if inputActive {
				return event // let the input field handle it
			}
			a.TView.Stop()
			return nil
		case event.Key() == tcell.KeyEscape:
			if inputActive {
				// Let input screens handle their own Esc
				// (e.g. chat input -> go back to video)
				return event
			}
			if len(a.screenStack) > 0 {
				a.PopScreen()
				return nil
			}
			a.TView.Stop()
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == ':':
			if inputActive {
				return event
			}
			a.navigateToSearch()
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == '?':
			if inputActive {
				return event
			}
			a.toggleHelp()
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 'B':
			if inputActive {
				return event
			}
			a.CycleBackend()
			return nil
		}
		return event
	})

	// Set up initial screen
	a.navigateToSearch()

	if a.Config.UI.Mouse {
		a.TView.EnableMouse(true)
	}

	return a.TView.Run()
}

func (a *App) navigateToSearch() {
	// Reset breadcrumb and stack
	if a.currentScreen != nil {
		a.currentScreen.OnHide()
		a.currentScreen.Cancel()
	}
	for _, s := range a.screenStack {
		s.Cancel()
	}
	a.screenStack = nil
	a.breadcrumb = nil
	a.currentScreen = nil

	search := NewSearchScreen(a)
	a.PushScreen("Search", search)
}
