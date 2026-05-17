package app

import (
	"context"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchScreen struct {
	app       *App
	flex      *tview.Flex
	input     *tview.InputField
	status    *tview.TextView
	cancelFn  context.CancelFunc
	isLoading bool
}

func NewSearchScreen(a *App) *SearchScreen {
	s := &SearchScreen{app: a}
	s.build()
	return s
}

func (s *SearchScreen) build() {
	voidBg := s.app.Theme.Color("background_void")
	modalBg := s.app.Theme.Color("modal_bg")
	modalBorder := s.app.Theme.Color("modal_border")

	// Use Nerd Font icon if terminal supports it, fall back gracefully
	s.input = tview.NewInputField().
		SetLabel("  ").
		SetFieldWidth(0).
		SetPlaceholder("YouTube search...")

	s.input.SetBackgroundColor(modalBg)
	s.input.SetFieldBackgroundColor(modalBg)
	s.input.SetFieldTextColor(s.app.Theme.Color("foreground"))
	s.input.SetLabelColor(s.app.Theme.Color("title"))
	s.input.SetPlaceholderTextColor(s.app.Theme.Color("footer_fg"))

	s.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	s.status.SetBackgroundColor(modalBg)

	// Spacers that fill with modal background
	padL := tview.NewBox().SetBackgroundColor(modalBg)
	padR := tview.NewBox().SetBackgroundColor(modalBg)
	padTop := tview.NewBox().SetBackgroundColor(modalBg)
	padBot := tview.NewBox().SetBackgroundColor(modalBg)

	// Horizontal padding: 3 cols left, input expands, 3 cols right
	inputRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(padL, 3, 0, false).
		AddItem(s.input, 0, 1, true).
		AddItem(padR, 3, 0, false)
	inputRow.SetBackgroundColor(modalBg)

	// Vertical layout: 1 row top pad, input, status, 1 row bottom pad
	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(padTop, 1, 0, false).
		AddItem(inputRow, 1, 0, true).
		AddItem(s.status, 1, 0, false).
		AddItem(padBot, 1, 0, false)
	modal.SetBorder(true).
		SetTitle(" ytcap ").
		SetTitleColor(s.app.Theme.Color("title")).
		SetBorderColor(modalBorder).
		SetBackgroundColor(modalBg)

	s.input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			query := strings.TrimSpace(s.input.GetText())
			query = strings.TrimPrefix(query, ":")
			if query == "" {
				s.status.SetText("[" + s.app.Theme.Colors["status_error"] + "]Search query cannot be empty[-]")
				return
			}
			// @handle → open channel directly
			if strings.HasPrefix(query, "@") {
				s.openChannel(query)
				return
			}
			s.doSearch(query)
		case tcell.KeyEscape:
			if len(s.app.screenStack) > 0 {
				s.app.PopScreen()
			} else {
				s.app.TView.Stop()
			}
		}
	})

	// Center the modal in a void background — Spotlight style
	innerFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(modal, 6, 0, true).
		AddItem(nil, 0, 1, false)
	innerFlex.SetBackgroundColor(voidBg)

	s.flex = tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(innerFlex, 64, 0, true).
		AddItem(nil, 0, 1, false)
	s.flex.SetBackgroundColor(voidBg)
}

func (s *SearchScreen) openChannel(handle string) {
	if s.isLoading {
		return
	}
	if !strings.HasPrefix(handle, "@") {
		handle = "@" + handle
	}
	channelURL := "https://www.youtube.com/" + handle

	// If channel is already cached, open immediately
	if _, ok := s.app.sessionChannels[channelURL]; ok {
		cs := NewChannelScreen(s.app, channelURL, handle)
		s.app.PushScreen(handle, cs)
		return
	}

	s.status.SetText("[" + s.app.Theme.Colors["status_loading"] + "]Checking " + handle + "...[-]")
	s.isLoading = true

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn = cancel

	go func() {
		videos, err := s.app.YtDlp.FetchChannelPage(ctx, channelURL, 1, 1)
		s.app.TView.QueueUpdateDraw(func() {
			s.isLoading = false
			if err != nil || len(videos) == 0 {
				if ctx.Err() != nil {
					return
				}
				s.status.SetText("[" + s.app.Theme.Colors["status_error"] + "]Channel not found: " + handle + "[-]")
				return
			}
			s.status.SetText("")
			cs := NewChannelScreen(s.app, channelURL, handle)
			s.app.PushScreen(handle, cs)
		})
	}()
}

func (s *SearchScreen) doSearch(query string) {
	if s.isLoading {
		return
	}

	// Check session cache first
	if cached, ok := s.app.sessionSearches[query]; ok {
		rs := NewResultsScreen(s.app, cached.Results, query)
		s.app.PushScreen(query, rs)
		return
	}

	s.isLoading = true
	s.status.SetText("[" + s.app.Theme.Colors["status_loading"] + "]Searching...[-]")

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFn = cancel

	go func() {
		results, err := s.app.YtDlp.Search(ctx, query, s.app.Config.YtDlp.SearchLimit)
		s.app.TView.QueueUpdateDraw(func() {
			s.isLoading = false
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				s.status.SetText("[" + s.app.Theme.Colors["status_error"] + "]Error: " + err.Error() + "[-]")
				return
			}
			// Cache results
			s.app.sessionSearches[query] = &SessionSearch{Results: results}
			s.status.SetText("")
			rs := NewResultsScreen(s.app, results, query)
			s.app.PushScreen(query, rs)
		})
	}()
}

func (s *SearchScreen) Primitive() tview.Primitive { return s.flex }
func (s *SearchScreen) OnShow() {
	s.app.updateFooter("<Enter> search  <@handle> open channel  <Esc> quit")
	s.app.TView.SetFocus(s.input)
}
func (s *SearchScreen) OnHide() {}
func (s *SearchScreen) Cancel() {
	if s.cancelFn != nil {
		s.cancelFn()
	}
}
func (s *SearchScreen) IsInputActive() bool { return s.input.GetText() != "" }
