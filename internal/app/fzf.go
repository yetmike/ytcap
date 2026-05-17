package app

import (
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
	"github.com/rivo/tview"
)

type fuzzyMatch struct {
	Index int
	Text  string
	Score int
}

// fuzzyPickerScreen is a native in-app fuzzy finder screen.
type fuzzyPickerScreen struct {
	app    *App
	flex   *tview.Flex
	input  *tview.InputField
	list   *tview.List
	items  []string
	onPick func(index int)
}

func newFuzzyPickerScreen(a *App, items []string, prompt, title string, onPick func(index int)) *fuzzyPickerScreen {
	fp := &fuzzyPickerScreen{
		app:    a,
		items:  items,
		onPick: onPick,
	}
	fp.build(prompt, title)
	return fp
}

func (fp *fuzzyPickerScreen) build(prompt, title string) {
	fp.input = tview.NewInputField().
		SetLabel(prompt).
		SetFieldWidth(0)
	fp.input.SetFieldBackgroundColor(fp.app.Theme.Color("selected_bg"))
	fp.input.SetFieldTextColor(fp.app.Theme.Color("foreground"))
	fp.input.SetLabelColor(fp.app.Theme.Color("title"))

	fp.list = tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true)
	fp.list.SetBackgroundColor(fp.app.Theme.Color("background"))
	fp.list.SetMainTextColor(fp.app.Theme.Color("foreground"))
	fp.list.SetSelectedTextColor(fp.app.Theme.Color("selected_fg"))
	fp.list.SetSelectedBackgroundColor(fp.app.Theme.Color("selected_bg"))

	// Populate initial list
	for _, item := range fp.items {
		fp.list.AddItem(item, "", 0, nil)
	}

	// Fuzzy filter on text change
	fp.input.SetChangedFunc(func(text string) {
		fp.list.Clear()
		if text == "" {
			for _, item := range fp.items {
				fp.list.AddItem(item, "", 0, nil)
			}
			return
		}

		matches := fp.fuzzyMatch(text)
		for _, m := range matches {
			fp.list.AddItem(m.Text, "", 0, nil)
		}
	})

	fp.input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			if fp.list.GetItemCount() == 0 {
				return
			}
			idx := fp.list.GetCurrentItem()
			selectedText, _ := fp.list.GetItemText(idx)
			for i, orig := range fp.items {
				if orig == selectedText {
					fp.app.PopScreen()
					fp.onPick(i)
					return
				}
			}
		case tcell.KeyEscape:
			fp.app.PopScreen()
		}
	})

	// Arrow keys navigate the list while typing
	fp.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyDown, tcell.KeyCtrlN:
			count := fp.list.GetItemCount()
			if count > 0 {
				cur := fp.list.GetCurrentItem()
				if cur < count-1 {
					fp.list.SetCurrentItem(cur + 1)
				}
			}
			return nil
		case tcell.KeyUp, tcell.KeyCtrlP:
			cur := fp.list.GetCurrentItem()
			if cur > 0 {
				fp.list.SetCurrentItem(cur - 1)
			}
			return nil
		case tcell.KeyTab:
			// Tab moves to next match
			count := fp.list.GetItemCount()
			if count > 0 {
				cur := fp.list.GetCurrentItem()
				fp.list.SetCurrentItem((cur + 1) % count)
			}
			return nil
		}
		return event
	})

	fp.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(fp.input, 1, 0, true).
		AddItem(fp.list, 0, 1, false)
	fp.flex.SetBorder(true).
		SetTitle(" " + title + " ").
		SetTitleColor(fp.app.Theme.Color("title")).
		SetBorderColor(fp.app.Theme.Color("border_focus")).
		SetBackgroundColor(fp.app.Theme.Color("background"))
}

func (fp *fuzzyPickerScreen) fuzzyMatch(query string) []fuzzyMatch {
	pattern := []rune(query)
	var matches []fuzzyMatch

	for i, item := range fp.items {
		chars := util.RunesToChars([]rune(item))
		res, _ := algo.FuzzyMatchV2(false, true, true, &chars, pattern, true, nil)
		if res.Start >= 0 {
			matches = append(matches, fuzzyMatch{Index: i, Text: item, Score: res.Score})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches
}

func (fp *fuzzyPickerScreen) Primitive() tview.Primitive { return fp.flex }
func (fp *fuzzyPickerScreen) OnShow() {
	fp.app.updateFooter("<type> filter  <↑/↓> navigate  <Enter> select  <Esc> cancel")
	fp.app.TView.SetFocus(fp.input)
}
func (fp *fuzzyPickerScreen) OnHide()             {}
func (fp *fuzzyPickerScreen) Cancel()              {}
func (fp *fuzzyPickerScreen) IsInputActive() bool  { return true }

// fzfPickAsync opens a native fuzzy picker as a pushed screen.
// Calls onPick with the selected index when the user picks an item.
func (a *App) fzfPickAsync(items []string, prompt, title string, onPick func(index int)) {
	picker := newFuzzyPickerScreen(a, items, prompt, title, onPick)
	a.PushScreen("Filter", picker)
}
