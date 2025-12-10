package bv

import (
	"testing"
)

func TestIsInstalled(t *testing.T) {
	// This test verifies the function works - actual result depends on environment
	result := IsInstalled()
	t.Logf("bv installed: %v", result)
}

func TestDriftStatusString(t *testing.T) {
	tests := []struct {
		status DriftStatus
		want   string
	}{
		{DriftOK, "OK"},
		{DriftCritical, "critical"},
		{DriftWarning, "warning"},
		{DriftNoBaseline, "no baseline"},
		{DriftStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.want {
				t.Errorf("DriftStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestCheckDrift(t *testing.T) {
	if !IsInstalled() {
		t.Skip("bv not installed")
	}

	result := CheckDrift()
	t.Logf("Drift status: %s, message: %s", result.Status, result.Message)

	// Status should be one of the defined values
	switch result.Status {
	case DriftOK, DriftCritical, DriftWarning, DriftNoBaseline:
		// Valid status
	default:
		t.Errorf("Unexpected drift status: %d", result.Status)
	}
}

func TestGetInsights(t *testing.T) {
	if !IsInstalled() {
		t.Skip("bv not installed")
	}

	insights, err := GetInsights()
	if err != nil {
		t.Fatalf("GetInsights() error: %v", err)
	}

	t.Logf("Got %d bottlenecks", len(insights.Bottlenecks))

	// Verify structure
	for _, b := range insights.Bottlenecks {
		if b.ID == "" {
			t.Error("Bottleneck has empty ID")
		}
	}
}

func TestGetPriority(t *testing.T) {
	if !IsInstalled() {
		t.Skip("bv not installed")
	}

	priority, err := GetPriority()
	if err != nil {
		t.Fatalf("GetPriority() error: %v", err)
	}

	t.Logf("Got %d recommendations", len(priority.Recommendations))

	// Verify structure
	for _, r := range priority.Recommendations {
		if r.IssueID == "" {
			t.Error("Recommendation has empty IssueID")
		}
		if r.Confidence < 0 || r.Confidence > 1 {
			t.Errorf("Invalid confidence: %f", r.Confidence)
		}
	}
}

func TestGetPlan(t *testing.T) {
	if !IsInstalled() {
		t.Skip("bv not installed")
	}

	plan, err := GetPlan()
	if err != nil {
		t.Fatalf("GetPlan() error: %v", err)
	}

	t.Logf("Got %d tracks", len(plan.Plan.Tracks))

	// Verify structure
	for _, track := range plan.Plan.Tracks {
		if track.TrackID == "" {
			t.Error("Track has empty TrackID")
		}
		if len(track.Items) == 0 {
			t.Errorf("Track %s has no items", track.TrackID)
		}
	}
}

func TestGetRecipes(t *testing.T) {
	if !IsInstalled() {
		t.Skip("bv not installed")
	}

	recipes, err := GetRecipes()
	if err != nil {
		t.Fatalf("GetRecipes() error: %v", err)
	}

	t.Logf("Got %d recipes", len(recipes.Recipes))

	// Should have at least the builtin recipes
	if len(recipes.Recipes) == 0 {
		t.Error("Expected at least one recipe")
	}

	// Verify structure
	for _, r := range recipes.Recipes {
		if r.Name == "" {
			t.Error("Recipe has empty name")
		}
		if r.Source == "" {
			t.Error("Recipe has empty source")
		}
	}
}

func TestGetTopBottlenecks(t *testing.T) {
	if !IsInstalled() {
		t.Skip("bv not installed")
	}

	bottlenecks, err := GetTopBottlenecks(3)
	if err != nil {
		t.Fatalf("GetTopBottlenecks() error: %v", err)
	}

	if len(bottlenecks) > 3 {
		t.Errorf("Expected at most 3 bottlenecks, got %d", len(bottlenecks))
	}

	t.Logf("Top bottlenecks: %v", bottlenecks)
}

func TestGetNextActions(t *testing.T) {
	if !IsInstalled() {
		t.Skip("bv not installed")
	}

	actions, err := GetNextActions(5)
	if err != nil {
		t.Fatalf("GetNextActions() error: %v", err)
	}

	if len(actions) > 5 {
		t.Errorf("Expected at most 5 actions, got %d", len(actions))
	}

	t.Logf("Next actions: %d items", len(actions))
}

func TestGetParallelTracks(t *testing.T) {
	if !IsInstalled() {
		t.Skip("bv not installed")
	}

	tracks, err := GetParallelTracks()
	if err != nil {
		t.Fatalf("GetParallelTracks() error: %v", err)
	}

	t.Logf("Parallel tracks: %d", len(tracks))
}

func TestIsBottleneck(t *testing.T) {
	if !IsInstalled() {
		t.Skip("bv not installed")
	}

	// Test with a likely non-existent ID
	isBottle, score, err := IsBottleneck("nonexistent-issue-xyz")
	if err != nil {
		t.Fatalf("IsBottleneck() error: %v", err)
	}

	if isBottle {
		t.Error("Expected nonexistent issue to not be a bottleneck")
	}
	if score != 0 {
		t.Errorf("Expected score 0 for non-bottleneck, got %f", score)
	}
}

func TestGetHealthSummary(t *testing.T) {
	if !IsInstalled() {
		t.Skip("bv not installed")
	}

	summary, err := GetHealthSummary()
	if err != nil {
		t.Fatalf("GetHealthSummary() error: %v", err)
	}

	t.Logf("Health: drift=%s, bottlenecks=%d, top=%s",
		summary.DriftStatus, summary.BottleneckCount, summary.TopBottleneck)
}

func TestNotInstalled(t *testing.T) {
	// Test error behavior when bv is not in PATH
	// We can't easily test this without modifying PATH, so just verify the error exists
	if ErrNotInstalled == nil {
		t.Error("ErrNotInstalled should not be nil")
	}
	if ErrNoBaseline == nil {
		t.Error("ErrNoBaseline should not be nil")
	}
}
