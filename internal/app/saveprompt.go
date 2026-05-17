package app

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// showSavePrompt shows a vim-style :w command line at the bottom of the current
// screen. Esc cancels, Enter saves. Does not push a new screen.
func (a *App) showSavePrompt(defaultPath string, onSave func(path string)) {
	a.savePromptOpen = true

	saveInput := tview.NewInputField().
		SetLabel(" :w ").
		SetText(defaultPath).
		SetFieldWidth(0)
	saveInput.SetFieldBackgroundColor(a.Theme.Color("selected_bg"))
	saveInput.SetFieldTextColor(a.Theme.Color("foreground"))
	saveInput.SetLabelColor(a.Theme.Color("key_hint"))

	restoreFooter := func() {
		a.savePromptOpen = false
		a.body.RemoveItem(saveInput)
		a.body.AddItem(a.footer, 1, 0, false)
		if a.currentScreen != nil {
			a.currentScreen.OnShow()
			a.TView.SetFocus(a.currentScreen.Primitive())
		}
	}

	saveInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			path := saveInput.GetText()
			if path == "" {
				path = defaultPath
			}
			restoreFooter()
			onSave(path)
		case tcell.KeyEscape:
			restoreFooter()
		}
	})

	// Ctrl+C also cancels
	saveInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			restoreFooter()
			return nil
		}
		return event
	})

	a.body.RemoveItem(a.footer)
	a.body.AddItem(saveInput, 1, 0, true)
	a.TView.SetFocus(saveInput)
}
