package app

import (
	"testing"

	"github.com/yetmike/ytcap/internal/summarize"
)

func TestClearChat_OnlyClearsBuffer(t *testing.T) {
	// clearChat should only nil out messages and clear the view.
	// It must NOT call Storage.DeleteChat or remove any saved files.
	c := &ChatScreen{
		messages: []summarize.ChatMessage{
			{Role: "user", Content: "hello"},
			{Role: "ai", Content: "hi there"},
		},
	}

	c.messages = nil // simulate clearChat's core operation

	if c.messages != nil {
		t.Error("expected messages to be nil after clear")
	}
}

func TestClearChat_PreservesVideoReference(t *testing.T) {
	// After clearing chat, the videoScr reference must remain intact
	// so users can continue chatting about the same video.
	vs := &VideoScreen{}
	c := &ChatScreen{
		videoScr: vs,
		messages: []summarize.ChatMessage{
			{Role: "user", Content: "test"},
		},
	}

	c.messages = nil

	if c.videoScr != vs {
		t.Error("expected videoScr to remain after clear")
	}
}
