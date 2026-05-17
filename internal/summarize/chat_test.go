package summarize

import "testing"

func TestChatMessage_Types(t *testing.T) {
	user := ChatMessage{Role: "user", Content: "hello"}
	ai := ChatMessage{Role: "ai", Content: "hi"}

	if user.Role != "user" {
		t.Error("expected user role")
	}
	if ai.Role != "ai" {
		t.Error("expected ai role")
	}
}
