package app

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestLoadTheme_Default(t *testing.T) {
	theme := LoadTheme("default")
	if theme == nil {
		t.Fatal("expected non-nil theme")
	}
	if theme.Colors["background"] == "" {
		t.Error("expected background color to be set")
	}
	if theme.Colors["title"] == "" {
		t.Error("expected title color to be set")
	}
}

func TestLoadTheme_FallbackToDefault(t *testing.T) {
	theme := LoadTheme("nonexistent-skin-name")
	if theme == nil {
		t.Fatal("expected fallback theme")
	}
	// Should have loaded the default skin
	if theme.Colors["background"] == "" {
		t.Error("expected fallback to have colors")
	}
}

func TestTheme_Color_Known(t *testing.T) {
	theme := &Theme{
		Colors: map[string]string{
			"title": "#89b4fa",
		},
	}
	c := theme.Color("title")
	if c == tcell.ColorDefault {
		t.Error("expected a real color, got default")
	}
}

func TestTheme_Color_Unknown(t *testing.T) {
	theme := &Theme{
		Colors: map[string]string{},
	}
	c := theme.Color("nonexistent")
	if c != tcell.ColorDefault {
		t.Error("expected default color for unknown key")
	}
}
