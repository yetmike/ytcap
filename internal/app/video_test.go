package app

import "testing"

func TestReflowText(t *testing.T) {
	input := `Transcript:
docker a tool that can package software
into containers that run reliably in any
environment but what is a container and
why do you need one let's imagine you
built an app with cobalt that runs on
some weird flavor of linux

you want to share this app with your
friend but he has an entirely different
system

[0:42] this is a timestamp line`

	got := reflowText(input)

	// Should preserve "Transcript:" on its own line
	if !contains(got, "Transcript:\n") {
		t.Error("expected Transcript: on its own line")
	}

	// Should join short lines into one long paragraph
	if contains(got, "software\ninto") {
		t.Error("expected short lines to be joined")
	}

	// Should contain joined text
	if !contains(got, "docker a tool that can package software into containers") {
		t.Error("expected reflowed paragraph")
	}

	// Should preserve paragraph break
	if !contains(got, "\n\n") {
		t.Error("expected paragraph breaks preserved")
	}

	// Should preserve timestamp line
	if !contains(got, "[0:42] this is a timestamp line") {
		t.Error("expected timestamp line preserved")
	}
}

func TestCleanTranscript_RemovesNoiseTags(t *testing.T) {
	input := "[MUSIC]\nHello world\n[APPLAUSE]\nGoodbye\n[LAUGHTER]\n[SILENCE]\n[INAUDIBLE]\n"
	got := cleanTranscript(input)
	for _, tag := range []string{"[MUSIC]", "[APPLAUSE]", "[LAUGHTER]", "[SILENCE]", "[INAUDIBLE]"} {
		if contains(got, tag) {
			t.Errorf("expected %s to be removed", tag)
		}
	}
	if !contains(got, "Hello world") {
		t.Error("expected content to remain")
	}
	if !contains(got, "Goodbye") {
		t.Error("expected content to remain")
	}
}

func TestCleanTranscript_CaseInsensitive(t *testing.T) {
	input := "[music]\n[Music]\n[MUSIC PLAYING]\nreal text"
	got := cleanTranscript(input)
	if contains(got, "[music]") || contains(got, "[Music]") || contains(got, "[MUSIC PLAYING]") {
		t.Error("expected case-insensitive removal")
	}
	if !contains(got, "real text") {
		t.Error("expected real text to remain")
	}
}

func TestCleanTranscript_RemovesTranscriptHeader(t *testing.T) {
	input := "Transcript:\nSome text here"
	got := cleanTranscript(input)
	if contains(got, "Transcript:") {
		t.Error("expected Transcript: header to be removed")
	}
	if !contains(got, "Some text here") {
		t.Error("expected text to remain")
	}
}

func TestReflowText_PreservesHeaderLines(t *testing.T) {
	input := "# Section Title\nsome words\nthat continue"
	got := reflowText(input)
	if !contains(got, "# Section Title\n") {
		t.Error("expected header line preserved")
	}
	if !contains(got, "some words that continue") {
		t.Error("expected lines to be joined")
	}
}

func TestReflowText_PreservesLabelLines(t *testing.T) {
	input := "Speaker:\nthe actual words go here\nand continue"
	got := reflowText(input)
	if !contains(got, "Speaker:\n") {
		t.Error("expected label line preserved on its own line")
	}
}

func TestReflowText_EmptyInput(t *testing.T) {
	got := reflowText("")
	// Empty input produces at most whitespace
	if len(got) > 1 {
		t.Errorf("expected minimal output for empty input, got %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
