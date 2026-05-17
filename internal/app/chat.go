package app

import (
	"context"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/yetmike/ytcap/internal/summarize"
)

type ChatScreen struct {
	app        *App
	flex       *tview.Flex
	chatView   *tview.TextView
	input      *tview.InputField
	videoScr   *VideoScreen
	messages   []summarize.ChatMessage
	cancelFn   context.CancelFunc
	isLoading  bool
	focusChat  bool // true = focus on chat view for scrolling, false = focus on input

	// Input history (arrow up/down)
	inputHistory []string
	historyIdx   int // points past the end when not browsing; decremented by Up, incremented by Down
	historyDraft string // text typed before browsing started
}

func NewChatScreen(a *App, videoScr *VideoScreen) *ChatScreen {
	c := &ChatScreen{
		app:      a,
		videoScr: videoScr,
	}
	c.build()
	return c
}

func (c *ChatScreen) build() {
	title := "Chat"
	if v := c.videoScr.GetVideo(); v != nil {
		title = "Chat - " + v.Title
	}

	c.chatView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	c.chatView.SetBackgroundColor(c.app.Theme.Color("background"))
	c.chatView.SetTitle(title).SetBorder(true).
		SetBorderColor(c.app.Theme.Color("border"))

	c.chatView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyEscape:
			c.app.PopScreen()
			return nil
		case event.Key() == tcell.KeyTab:
			c.focusInput()
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 'i':
			c.focusInput()
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 'C':
			c.saveChat()
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 'j':
			row, _ := c.chatView.GetScrollOffset()
			c.chatView.ScrollTo(row+1, 0)
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 'k':
			row, _ := c.chatView.GetScrollOffset()
			if row > 0 {
				c.chatView.ScrollTo(row-1, 0)
			}
			return nil
		case event.Key() == tcell.KeyRune && event.Rune() == 'G':
			c.chatView.ScrollToEnd()
			return nil
		case event.Key() == tcell.KeyCtrlL:
			c.clearChat()
			return nil
		}
		return event
	})

	c.input = tview.NewInputField().
		SetLabel("> ").
		SetFieldWidth(0)
	c.input.SetFieldBackgroundColor(c.app.Theme.Color("chat_input_bg"))
	c.input.SetLabelColor(c.app.Theme.Color("chat_user"))

	c.input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			question := strings.TrimSpace(c.input.GetText())
			if question == "" {
				c.appendStatus("Message cannot be empty")
				return
			}
			c.inputHistory = append(c.inputHistory, question)
			c.historyIdx = len(c.inputHistory)
			c.input.SetText("")
			c.sendMessage(question)
		}
	})

	c.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyEscape:
			c.app.PopScreen()
			return nil
		case event.Key() == tcell.KeyUp:
			if len(c.inputHistory) == 0 {
				return nil
			}
			if c.historyIdx == len(c.inputHistory) {
				c.historyDraft = c.input.GetText()
			}
			if c.historyIdx > 0 {
				c.historyIdx--
				c.input.SetText(c.inputHistory[c.historyIdx])
			}
			return nil
		case event.Key() == tcell.KeyDown:
			if c.historyIdx < len(c.inputHistory)-1 {
				c.historyIdx++
				c.input.SetText(c.inputHistory[c.historyIdx])
			} else if c.historyIdx == len(c.inputHistory)-1 {
				c.historyIdx = len(c.inputHistory)
				c.input.SetText(c.historyDraft)
			}
			return nil
		case event.Key() == tcell.KeyTab:
			c.focusChatView()
			return nil
		case event.Key() == tcell.KeyCtrlL:
			c.clearChat()
			return nil
		case event.Key() == tcell.KeyCtrlC:
			if c.cancelFn != nil {
				c.cancelFn()
			}
			return nil
		case event.Key() == tcell.KeyCtrlS:
			c.saveChat()
			return nil
		}
		return event
	})

	c.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(c.chatView, 0, 1, false).
		AddItem(c.input, 1, 0, true)
}

func (c *ChatScreen) focusChatView() {
	c.focusChat = true
	c.chatView.SetBorderColor(c.app.Theme.Color("border_focus"))
	c.app.TView.SetFocus(c.chatView)
	c.updateHints()
}

func (c *ChatScreen) focusInput() {
	c.focusChat = false
	c.chatView.SetBorderColor(c.app.Theme.Color("border"))
	c.app.TView.SetFocus(c.input)
	c.updateHints()
}

func (c *ChatScreen) updateHints() {
	if c.focusChat {
		c.app.updateFooter("<j/k> scroll  <Tab/i> input  <C> save  <Esc> back")
	} else {
		c.app.updateFooter("<Enter> send  <Tab> scroll  <Ctrl+S> save  <Ctrl+L> clear  <Ctrl+C> cancel  <Esc> back")
	}
}

func (c *ChatScreen) sendMessage(question string) {
	if c.isLoading {
		return
	}

	c.messages = append(c.messages, summarize.ChatMessage{Role: "user", Content: question})
	c.renderChat()

	transcript := c.videoScr.GetTranscript()
	if transcript == "" {
		c.appendStatus("Fetching transcript first...")
		c.isLoading = true

		ctx, cancel := context.WithCancel(context.Background())
		c.cancelFn = cancel

		go func() {
			result, err := c.app.Summarize.GetTranscript(ctx, c.videoScr.GetVideoURL())
			c.app.TView.QueueUpdateDraw(func() {
				if err != nil {
					c.isLoading = false
					if ctx.Err() != nil {
						return
					}
					c.appendStatus("Error fetching transcript: " + err.Error())
					return
				}
				c.doAsk(result, question)
			})
		}()
		return
	}

	c.doAsk(transcript, question)
}

func (c *ChatScreen) doAsk(transcript, question string) {
	c.isLoading = true
	c.appendStatus("Thinking...")

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFn = cancel

	go func() {
		result, err := c.app.Summarize.Ask(ctx, transcript, c.messages[:len(c.messages)-1], question)
		c.app.TView.QueueUpdateDraw(func() {
			c.isLoading = false
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.appendStatus("Error: " + err.Error())
				return
			}
			c.messages = append(c.messages, summarize.ChatMessage{Role: "ai", Content: strings.TrimSpace(result)})
			c.renderChat()
		})
	}()
}

// mdToTview does basic markdown-to-tview color conversion
var (
	mdBold   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	mdItalic = regexp.MustCompile(`\*(.+?)\*`)
	mdCode   = regexp.MustCompile("`([^`]+)`")
	mdH1     = regexp.MustCompile(`(?m)^# (.+)$`)
	mdH2     = regexp.MustCompile(`(?m)^## (.+)$`)
	mdH3     = regexp.MustCompile(`(?m)^### (.+)$`)
	mdBullet = regexp.MustCompile(`(?m)^[-*] (.+)$`)
)

func (c *ChatScreen) mdToTview(text string) string {
	titleColor := c.app.Theme.Colors["title"]
	keyColor := c.app.Theme.Colors["key_hint"]
	durationColor := c.app.Theme.Colors["video_duration"]

	// Strip code fences
	text = strings.ReplaceAll(text, "```", "")
	// Headers
	text = mdH1.ReplaceAllString(text, "["+titleColor+"]$1[-]")
	text = mdH2.ReplaceAllString(text, "["+titleColor+"]$1[-]")
	text = mdH3.ReplaceAllString(text, "["+keyColor+"]$1[-]")
	// Bold
	text = mdBold.ReplaceAllString(text, "["+keyColor+"]$1[-]")
	// Inline code
	text = mdCode.ReplaceAllString(text, "["+durationColor+"]$1[-]")
	// Italic (after bold so ** is consumed first)
	text = mdItalic.ReplaceAllString(text, "$1")
	// Bullets
	text = mdBullet.ReplaceAllString(text, "  "+tview.Escape("*")+" $1")

	return text
}

func (c *ChatScreen) renderChat() {
	var b strings.Builder
	for _, msg := range c.messages {
		switch msg.Role {
		case "user":
			b.WriteString("[" + c.app.Theme.Colors["chat_user"] + "]You:[-]  " + tview.Escape(msg.Content) + "\n\n")
		case "ai":
			b.WriteString("[" + c.app.Theme.Colors["chat_ai"] + "]AI:[-]\n" + c.mdToTview(msg.Content) + "\n\n")
		}
	}
	c.chatView.SetText(b.String())
	c.chatView.ScrollToEnd()
}

func (c *ChatScreen) appendStatus(status string) {
	current := c.chatView.GetText(false)
	c.chatView.SetText(current + "[" + c.app.Theme.Colors["status_loading"] + "]" + status + "[-]\n")
	c.chatView.ScrollToEnd()
}

func (c *ChatScreen) clearChat() {
	c.messages = nil
	c.chatView.Clear()
	c.app.SetStatus("Chat cleared")
}

func (c *ChatScreen) saveChat() {
	if c.videoScr.GetVideo() == nil || len(c.messages) == 0 {
		c.app.SetStatus("No chat to save")
		return
	}
	v := c.videoScr.GetVideo()
	messages := c.messages
	defaultPath := c.app.Storage.DefaultChatPath(v.ID, v.Title)
	c.app.showSavePrompt(defaultPath, func(userPath string) {
		path, err := c.app.Storage.SaveChatTo(userPath, v.ID, v.Title, v.Channel, v.URL, messages)
		if err != nil {
			c.app.SetStatus("Error saving: " + err.Error())
			return
		}
		c.app.CopyToClipboard(path)
		c.app.SetStatusLong("Saved to " + path + "  (path copied)")
	})
}

func (c *ChatScreen) Primitive() tview.Primitive { return c.flex }
func (c *ChatScreen) OnShow() {
	c.updateHints()
	c.app.TView.SetFocus(c.input)
}
func (c *ChatScreen) OnHide() {}
func (c *ChatScreen) Cancel() {
	if c.cancelFn != nil {
		c.cancelFn()
	}
}
func (c *ChatScreen) IsInputActive() bool { return !c.focusChat }
