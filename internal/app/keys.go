package app

import (
	"time"

	"github.com/gdamore/tcell/v2"
)

type KeyTracker struct {
	lastKey     rune
	lastKeyTime time.Time
}

func (kt *KeyTracker) IsDoubleG(key rune) bool {
	if key != 'g' {
		kt.lastKey = key
		kt.lastKeyTime = time.Now()
		return false
	}

	if kt.lastKey == 'g' && time.Since(kt.lastKeyTime) < 300*time.Millisecond {
		kt.lastKey = 0
		return true
	}

	kt.lastKey = 'g'
	kt.lastKeyTime = time.Now()
	return false
}

func KeyName(ev *tcell.EventKey) string {
	if ev.Key() == tcell.KeyRune {
		return string(ev.Rune())
	}
	return ev.Name()
}
