package ensemble

import (
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/tmux"
)

func TestAssignRoundRobin_Success(t *testing.T) {
	panes := []tmux.Pane{
		{Title: "pane-b", Type: tmux.AgentCodex, Index: 2, NTMIndex: 2},
		{Title: "pane-a", Type: tmux.AgentClaude, Index: 1, NTMIndex: 1},
	}
	modes := []string{"deductive", "abductive"}

	assignments := AssignRoundRobin(modes, panes)
	if assignments == nil {
		t.Fatal("expected assignments, got nil")
	}
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(assignments))
	}

	if assignments[0].ModeID != "abductive" {
		t.Errorf("assignment[0] mode = %q, want abductive", assignments[0].ModeID)
	}
	if assignments[0].PaneName != "pane-a" {
		t.Errorf("assignment[0] pane = %q, want pane-a", assignments[0].PaneName)
	}
	if assignments[0].Status != AssignmentPending {
		t.Errorf("assignment[0] status = %q, want %q", assignments[0].Status, AssignmentPending)
	}
	if assignments[0].AssignedAt.IsZero() {
		t.Error("assignment[0] AssignedAt should be set")
	}
	if assignments[1].ModeID != "deductive" {
		t.Errorf("assignment[1] mode = %q, want deductive", assignments[1].ModeID)
	}
	if assignments[1].PaneName != "pane-b" {
		t.Errorf("assignment[1] pane = %q, want pane-b", assignments[1].PaneName)
	}
}

func TestAssignRoundRobin_TooManyModes(t *testing.T) {
	panes := []tmux.Pane{
		{Title: "pane-a", Type: tmux.AgentClaude, Index: 1, NTMIndex: 1},
	}
	modes := []string{"deductive", "abductive"}

	assignments := AssignRoundRobin(modes, panes)
	if assignments != nil {
		t.Fatalf("expected nil assignments, got %v", assignments)
	}
}

func TestAssignByCategory_PrefersAffinities(t *testing.T) {
	catalog := testModeCatalog(t)
	panes := []tmux.Pane{
		{Title: "pane-claude", Type: tmux.AgentClaude, Index: 1, NTMIndex: 1},
		{Title: "pane-codex", Type: tmux.AgentCodex, Index: 2, NTMIndex: 2},
	}
	modes := []string{"deductive", "practical"}

	assignments := AssignByCategory(modes, panes, catalog)
	if assignments == nil {
		t.Fatal("expected assignments, got nil")
	}
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(assignments))
	}

	modeToPane := map[string]string{}
	for _, assignment := range assignments {
		modeToPane[assignment.ModeID] = assignment.PaneName
	}

	if modeToPane["deductive"] != "pane-claude" {
		t.Errorf("deductive pane = %q, want pane-claude", modeToPane["deductive"])
	}
	if modeToPane["practical"] != "pane-codex" {
		t.Errorf("practical pane = %q, want pane-codex", modeToPane["practical"])
	}
}

func TestAssignExplicit_Success(t *testing.T) {
	panes := []tmux.Pane{
		{Title: "pane-1", Type: tmux.AgentClaude, Index: 1, NTMIndex: 1},
		{Title: "pane-2", Type: tmux.AgentCodex, Index: 2, NTMIndex: 2},
	}
	specs := []string{"deductive:cc", "abductive:cod"}

	assignments, err := AssignExplicit(specs, panes)
	if err != nil {
		t.Fatalf("AssignExplicit error: %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(assignments))
	}

	modeToAgent := map[string]string{}
	for _, assignment := range assignments {
		modeToAgent[assignment.ModeID] = assignment.AgentType
		if assignment.AssignedAt.IsZero() {
			t.Error("AssignedAt should be set")
		}
	}

	if modeToAgent["deductive"] != string(tmux.AgentClaude) {
		t.Errorf("deductive agent = %q, want %q", modeToAgent["deductive"], tmux.AgentClaude)
	}
	if modeToAgent["abductive"] != string(tmux.AgentCodex) {
		t.Errorf("abductive agent = %q, want %q", modeToAgent["abductive"], tmux.AgentCodex)
	}
}

func TestAssignExplicit_NotEnoughPanes(t *testing.T) {
	panes := []tmux.Pane{
		{Title: "pane-1", Type: tmux.AgentClaude, Index: 1, NTMIndex: 1},
	}
	specs := []string{"deductive:cc", "abductive:cod"}

	_, err := AssignExplicit(specs, panes)
	if err == nil {
		t.Fatal("expected error for insufficient panes")
	}
}

func TestAssignExplicit_InvalidSpec(t *testing.T) {
	panes := []tmux.Pane{
		{Title: "pane-1", Type: tmux.AgentClaude, Index: 1, NTMIndex: 1},
	}

	if _, err := AssignExplicit([]string{"deductive"}, panes); err == nil {
		t.Fatal("expected error for invalid spec without agent type")
	}
	if _, err := AssignExplicit([]string{"deductive:"}, panes); err == nil {
		t.Fatal("expected error for empty agent type")
	}
}

func TestValidateAssignments_DuplicatePane(t *testing.T) {
	now := time.Now().UTC()
	assignments := []ModeAssignment{
		{ModeID: "deductive", PaneName: "pane-1", AgentType: string(tmux.AgentClaude), Status: AssignmentPending, AssignedAt: now},
		{ModeID: "abductive", PaneName: "pane-1", AgentType: string(tmux.AgentCodex), Status: AssignmentPending, AssignedAt: now},
	}

	err := ValidateAssignments(assignments, []string{"deductive", "abductive"})
	if err == nil {
		t.Fatal("expected error for duplicate pane, got nil")
	}
}

func TestNormalizeModeKeys_Empty(t *testing.T) {
	if _, err := normalizeModeKeys([]string{"", "deductive"}); err == nil {
		t.Fatal("expected error for empty mode key")
	}
}

func TestPickAvailablePaneWithReason_Fallback(t *testing.T) {
	panes := []tmux.Pane{
		{Title: "pane-cc", Type: tmux.AgentClaude, Index: 1, NTMIndex: 1},
		{Title: "pane-cod", Type: tmux.AgentCodex, Index: 2, NTMIndex: 2},
	}
	byType := groupPanesByType(panes)
	used := []ModeAssignment{{ModeID: "deductive", PaneName: "pane-cc"}}

	choice, fallback, reason := pickAvailablePaneWithReason(byType, []string{string(tmux.AgentClaude)}, used)
	if choice.Title != "pane-cod" {
		t.Fatalf("choice = %q, want pane-cod", choice.Title)
	}
	if !fallback {
		t.Fatal("expected fallback to be true")
	}
	if reason == "" {
		t.Fatal("expected fallback reason to be set")
	}
}

func TestPickAvailablePane_ReturnsPreferred(t *testing.T) {
	panes := []tmux.Pane{
		{Title: "pane-cc", Type: tmux.AgentClaude, Index: 1, NTMIndex: 1},
		{Title: "pane-cod", Type: tmux.AgentCodex, Index: 2, NTMIndex: 2},
	}
	byType := groupPanesByType(panes)
	choice := pickAvailablePane(byType, []string{string(tmux.AgentClaude)}, nil)
	if choice.Title != "pane-cc" {
		t.Fatalf("choice = %q, want pane-cc", choice.Title)
	}
}

func TestResolveMode_WithNilCatalog(t *testing.T) {
	modeID, mode, err := resolveMode("Deductive", nil)
	if err != nil {
		t.Fatalf("resolveMode error: %v", err)
	}
	if modeID != "deductive" {
		t.Fatalf("modeID = %q, want deductive", modeID)
	}
	if mode != nil {
		t.Fatal("expected nil mode when catalog is nil")
	}
}

func TestResolveMode_ByCode(t *testing.T) {
	catalog := testModeCatalog(t)
	modeID, mode, err := resolveMode("A1", catalog)
	if err != nil {
		t.Fatalf("resolveMode error: %v", err)
	}
	if modeID != "deductive" || mode == nil {
		t.Fatalf("resolveMode returned %q/%v, want deductive", modeID, mode)
	}
}

func TestPaneHelpers(t *testing.T) {
	paneA := tmux.Pane{Title: "pane-a", Index: 2, NTMIndex: 2, Type: tmux.AgentClaude}
	paneB := tmux.Pane{Title: "pane-b", Index: 1, NTMIndex: 1, Type: tmux.AgentCodex}

	if !paneLess(paneB, paneA) {
		t.Fatal("expected paneB < paneA by NTMIndex")
	}
	if isAssignablePane(tmux.Pane{Title: "", Type: tmux.AgentClaude}) {
		t.Fatal("expected pane without title to be unassignable")
	}
	if isAssignablePane(tmux.Pane{Title: "user", Type: tmux.AgentUser}) {
		t.Fatal("expected user pane to be unassignable")
	}
}

func testModeCatalog(t *testing.T) *ModeCatalog {
	t.Helper()
	modes := []ReasoningMode{
		{
			ID:        "deductive",
			Code:      "A1",
			Name:      "Deductive",
			Category:  CategoryFormal,
			Tier:      TierCore,
			ShortDesc: "Deductive logic",
		},
		{
			ID:        "abductive",
			Code:      "C1",
			Name:      "Abductive",
			Category:  CategoryUncertainty,
			Tier:      TierCore,
			ShortDesc: "Abductive inference",
		},
		{
			ID:        "practical",
			Code:      "G1",
			Name:      "Practical",
			Category:  CategoryPractical,
			Tier:      TierCore,
			ShortDesc: "Practical reasoning",
		},
		{
			ID:        "advanced-mode",
			Code:      "A2",
			Name:      "Advanced",
			Category:  CategoryFormal,
			Tier:      TierAdvanced,
			ShortDesc: "Advanced logic",
		},
	}

	catalog, err := NewModeCatalog(modes, "1.0.0")
	if err != nil {
		t.Fatalf("NewModeCatalog error: %v", err)
	}
	return catalog
}
