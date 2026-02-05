package cli

import (
	"testing"
	"time"
)

func TestStateIcon(t *testing.T) {
	tests := []struct {
		state string
		want  string
	}{
		{"WAITING", "●"},
		{"GENERATING", "▶"},
		{"THINKING", "◐"},
		{"ERROR", "✗"},
		{"STALLED", "◯"},
		{"unknown", "?"},
		{"", "?"},
		{"waiting", "?"}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := stateIcon(tt.state)
			if got != tt.want {
				t.Errorf("stateIcon(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestFormatActivityDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"zero", 0, "-"},
		{"1 second", 1 * time.Second, "1s"},
		{"30 seconds", 30 * time.Second, "30s"},
		{"59 seconds", 59 * time.Second, "59s"},
		{"1 minute", 1 * time.Minute, "1m0s"},
		{"1 minute 30 seconds", 90 * time.Second, "1m30s"},
		{"5 minutes", 5 * time.Minute, "5m0s"},
		{"5 minutes 45 seconds", 5*time.Minute + 45*time.Second, "5m45s"},
		{"59 minutes 59 seconds", 59*time.Minute + 59*time.Second, "59m59s"},
		{"1 hour", 1 * time.Hour, "1h0m"},
		{"1 hour 30 minutes", 90 * time.Minute, "1h30m"},
		{"2 hours 15 minutes", 2*time.Hour + 15*time.Minute, "2h15m"},
		{"24 hours", 24 * time.Hour, "24h0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatActivityDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatActivityDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}
