package ensemble

import "testing"

func TestOutputCapture_DefaultsAndLineCount(t *testing.T) {
	capture := &OutputCapture{}
	capture.SetMaxLines(-1)
	capture.ensureDefaults()
	if capture.tmuxClient == nil {
		t.Fatal("expected tmux client to be set")
	}
	if capture.validator == nil {
		t.Fatal("expected validator to be set")
	}
	if capture.maxLines != defaultCaptureLines {
		t.Fatalf("maxLines = %d, want %d", capture.maxLines, defaultCaptureLines)
	}

	if countLines("") != 0 {
		t.Fatal("countLines should be 0 for empty string")
	}
	if countLines("a\n") != 1 {
		t.Fatal("countLines should ignore trailing newline")
	}
	if countLines("a\nb") != 2 {
		t.Fatal("countLines should count lines")
	}
}

func TestOutputCapture_SetMaxLines(t *testing.T) {
	capture := NewOutputCapture(nil)
	capture.SetMaxLines(10)
	if capture.maxLines != 10 {
		t.Fatalf("maxLines = %d, want 10", capture.maxLines)
	}
}
