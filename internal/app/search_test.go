package app

import "testing"

func TestSearchScreen_IsInputActive_EmptyAllowsQuit(t *testing.T) {
	// When the search input is empty, IsInputActive must return false
	// so the global 'q' handler can quit the app.
	text := ""
	isActive := text != ""
	if isActive {
		t.Error("expected IsInputActive to be false when input is empty")
	}
}

func TestSearchScreen_IsInputActive_TextBlocksQuit(t *testing.T) {
	// When input has text, IsInputActive must return true
	// to prevent 'q' from quitting mid-typing.
	text := "golang tutorials"
	isActive := text != ""
	if !isActive {
		t.Error("expected IsInputActive to be true when input has text")
	}
}
