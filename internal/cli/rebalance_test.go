package cli

import (
	"testing"
)

func TestCalculateImbalanceScore(t *testing.T) {
	tests := []struct {
		name      string
		workloads []RebalanceWorkload
		want      float64
		tolerance float64
	}{
		{
			name:      "empty workloads",
			workloads: []RebalanceWorkload{},
			want:      0,
			tolerance: 0.001,
		},
		{
			name: "single workload",
			workloads: []RebalanceWorkload{
				{Pane: 1, TaskCount: 5},
			},
			want:      0, // stddev/mean = 0/5 = 0
			tolerance: 0.001,
		},
		{
			name: "perfectly balanced",
			workloads: []RebalanceWorkload{
				{Pane: 1, TaskCount: 3},
				{Pane: 2, TaskCount: 3},
				{Pane: 3, TaskCount: 3},
			},
			want:      0,
			tolerance: 0.001,
		},
		{
			name: "moderate imbalance",
			workloads: []RebalanceWorkload{
				{Pane: 1, TaskCount: 4},
				{Pane: 2, TaskCount: 2},
				{Pane: 3, TaskCount: 3},
			},
			// mean = 3, variance = ((4-3)^2 + (2-3)^2 + (3-3)^2)/3 = 2/3
			// stddev = sqrt(2/3) ≈ 0.816
			// CV = 0.816/3 ≈ 0.272
			want:      0.272,
			tolerance: 0.01,
		},
		{
			name: "severe imbalance",
			workloads: []RebalanceWorkload{
				{Pane: 1, TaskCount: 10},
				{Pane: 2, TaskCount: 0},
				{Pane: 3, TaskCount: 0},
			},
			// mean = 10/3 ≈ 3.33
			// variance = ((10-3.33)^2 + (0-3.33)^2 + (0-3.33)^2)/3 ≈ 22.22
			// stddev ≈ 4.71
			// CV ≈ 4.71/3.33 ≈ 1.414
			want:      1.414,
			tolerance: 0.01,
		},
		{
			name: "all zero tasks",
			workloads: []RebalanceWorkload{
				{Pane: 1, TaskCount: 0},
				{Pane: 2, TaskCount: 0},
			},
			want:      0, // No tasks = balanced
			tolerance: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateImbalanceScore(tt.workloads)
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("calculateImbalanceScore() = %v, want %v (tolerance %v)", got, tt.want, tt.tolerance)
			}
		})
	}
}

func TestGetRecommendation(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		want  string
	}{
		{name: "zero score", score: 0.0, want: "balanced"},
		{name: "low score", score: 0.2, want: "balanced"},
		{name: "at threshold 0.3", score: 0.3, want: "moderate_imbalance"},
		{name: "moderate score", score: 0.5, want: "moderate_imbalance"},
		{name: "at threshold 0.7", score: 0.7, want: "rebalance_recommended"},
		{name: "high score", score: 1.0, want: "rebalance_recommended"},
		{name: "very high score", score: 2.0, want: "rebalance_recommended"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRecommendation(tt.score)
			if got != tt.want {
				t.Errorf("getRecommendation(%v) = %v, want %v", tt.score, got, tt.want)
			}
		})
	}
}

func TestMatchesRebalanceFilter(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
		filter    string
		want      bool
	}{
		{name: "no filter", agentType: "claude", filter: "", want: true},
		{name: "cc matches claude", agentType: "claude", filter: "cc", want: true},
		{name: "claude matches claude", agentType: "claude", filter: "claude", want: true},
		{name: "cc prefix matches", agentType: "cc_1", filter: "cc", want: true},
		{name: "cod matches codex", agentType: "codex", filter: "cod", want: true},
		{name: "codex matches codex", agentType: "codex", filter: "codex", want: true},
		{name: "cod prefix matches", agentType: "cod_2", filter: "cod", want: true},
		{name: "gmi matches gemini", agentType: "gemini", filter: "gmi", want: true},
		{name: "gemini matches gemini", agentType: "gemini", filter: "gemini", want: true},
		{name: "gmi prefix matches", agentType: "gmi_3", filter: "gmi", want: true},
		{name: "claude does not match cod", agentType: "claude", filter: "cod", want: false},
		{name: "codex does not match cc", agentType: "codex", filter: "cc", want: false},
		{name: "unknown type with filter returns false", agentType: "unknown", filter: "cc", want: false},
		{name: "case insensitive filter", agentType: "Claude", filter: "CC", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesRebalanceFilter(tt.agentType, tt.filter)
			if got != tt.want {
				t.Errorf("matchesRebalanceFilter(%q, %q) = %v, want %v", tt.agentType, tt.filter, got, tt.want)
			}
		})
	}
}

func TestRebalanceWorkloadCounts(t *testing.T) {
	workloads := []RebalanceWorkload{
		{Pane: 1, TaskCount: 5},
		{Pane: 2, TaskCount: 3},
		{Pane: 3, TaskCount: 0},
	}

	counts := rebalanceWorkloadCounts(workloads)

	if len(counts) != 3 {
		t.Errorf("expected 3 counts, got %d", len(counts))
	}
	if counts[1] != 5 {
		t.Errorf("expected pane 1 count = 5, got %d", counts[1])
	}
	if counts[2] != 3 {
		t.Errorf("expected pane 2 count = 3, got %d", counts[2])
	}
	if counts[3] != 0 {
		t.Errorf("expected pane 3 count = 0, got %d", counts[3])
	}
}

func TestCalculateAfterState(t *testing.T) {
	workloads := []RebalanceWorkload{
		{Pane: 1, TaskCount: 5},
		{Pane: 2, TaskCount: 0},
	}

	transfers := []RebalanceTransfer{
		{FromPane: 1, ToPane: 2},
		{FromPane: 1, ToPane: 2},
	}

	after := calculateAfterState(workloads, transfers)

	if after[1] != 3 {
		t.Errorf("expected pane 1 after = 3, got %d", after[1])
	}
	if after[2] != 2 {
		t.Errorf("expected pane 2 after = 2, got %d", after[2])
	}
}

func TestCalculateAfterStateEmpty(t *testing.T) {
	workloads := []RebalanceWorkload{
		{Pane: 1, TaskCount: 3},
	}

	transfers := []RebalanceTransfer{}

	after := calculateAfterState(workloads, transfers)

	if after[1] != 3 {
		t.Errorf("expected pane 1 after = 3 (unchanged), got %d", after[1])
	}
}
