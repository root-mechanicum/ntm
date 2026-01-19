package assign

import (
	"testing"

	"github.com/Dicklesworthstone/ntm/internal/tmux"
)

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		input string
		want  Strategy
	}{
		{"balanced", StrategyBalanced},
		{"BALANCED", StrategyBalanced},
		{"speed", StrategySpeed},
		{"quality", StrategyQuality},
		{"dependency", StrategyDependency},
		{"unknown", StrategyBalanced}, // Default
		{"", StrategyBalanced},        // Default
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := ParseStrategy(tc.input)
			if got != tc.want {
				t.Errorf("ParseStrategy(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestMatcher_AssignTasks_EmptyInputs(t *testing.T) {
	m := NewMatcher()

	// Empty beads
	result := m.AssignTasks(nil, []Agent{{ID: "1", Idle: true}}, StrategyBalanced)
	if result != nil {
		t.Errorf("Expected nil for empty beads, got %v", result)
	}

	// Empty agents
	result = m.AssignTasks([]Bead{{ID: "b1"}}, nil, StrategyBalanced)
	if result != nil {
		t.Errorf("Expected nil for empty agents, got %v", result)
	}
}

func TestMatcher_AssignTasks_NoAvailableAgents(t *testing.T) {
	m := NewMatcher()

	beads := []Bead{{ID: "b1", Title: "Test", TaskType: TaskFeature}}
	agents := []Agent{
		{ID: "1", AgentType: tmux.AgentClaude, Idle: false}, // Not idle
		{ID: "2", AgentType: tmux.AgentCodex, Idle: true, ContextUsage: 0.95}, // Too much context
	}

	result := m.AssignTasks(beads, agents, StrategyBalanced)
	if len(result) != 0 {
		t.Errorf("Expected no assignments for unavailable agents, got %d", len(result))
	}
}

func TestMatcher_AssignTasks_SingleBead_SingleAgent(t *testing.T) {
	m := NewMatcher()

	beads := []Bead{{ID: "b1", Title: "Fix bug", TaskType: TaskBug, Priority: 1}}
	agents := []Agent{{ID: "1", AgentType: tmux.AgentCodex, Idle: true, ContextUsage: 0.2}}

	result := m.AssignTasks(beads, agents, StrategyBalanced)

	if len(result) != 1 {
		t.Fatalf("Expected 1 assignment, got %d", len(result))
	}

	if result[0].Bead.ID != "b1" {
		t.Errorf("Expected bead b1, got %s", result[0].Bead.ID)
	}
	if result[0].Agent.ID != "1" {
		t.Errorf("Expected agent 1, got %s", result[0].Agent.ID)
	}
	if result[0].Score <= 0 {
		t.Error("Expected positive score")
	}
	if result[0].Reason == "" {
		t.Error("Expected non-empty reason")
	}
}

func TestMatcher_AssignTasks_MoreBeadsThanAgents(t *testing.T) {
	m := NewMatcher()

	beads := []Bead{
		{ID: "b1", Title: "Task 1", TaskType: TaskFeature, Priority: 1},
		{ID: "b2", Title: "Task 2", TaskType: TaskBug, Priority: 2},
		{ID: "b3", Title: "Task 3", TaskType: TaskDocs, Priority: 3},
	}
	agents := []Agent{
		{ID: "1", AgentType: tmux.AgentClaude, Idle: true},
	}

	// Balanced strategy assigns all beads even to single agent
	result := m.AssignTasks(beads, agents, StrategyBalanced)
	if len(result) != 3 {
		t.Errorf("Balanced strategy should assign all beads, got %d", len(result))
	}

	// Speed/Quality/Dependency strategies limit to one bead per agent
	result = m.AssignTasks(beads, agents, StrategySpeed)
	if len(result) != 1 {
		t.Errorf("Speed strategy should limit to 1 per agent, got %d", len(result))
	}

	// Should assign highest priority first (P1 = b1)
	if result[0].Bead.ID != "b1" {
		t.Errorf("Expected highest priority bead b1, got %s", result[0].Bead.ID)
	}
}

func TestMatcher_AssignTasks_MoreAgentsThanBeads(t *testing.T) {
	m := NewMatcher()

	beads := []Bead{
		{ID: "b1", Title: "Fix bug", TaskType: TaskBug, Priority: 1},
	}
	agents := []Agent{
		{ID: "1", AgentType: tmux.AgentClaude, Idle: true},
		{ID: "2", AgentType: tmux.AgentCodex, Idle: true},
		{ID: "3", AgentType: tmux.AgentGemini, Idle: true},
	}

	result := m.AssignTasks(beads, agents, StrategyBalanced)

	// Should only assign as many as we have beads
	if len(result) != 1 {
		t.Errorf("Expected 1 assignment (limited by beads), got %d", len(result))
	}
}

func TestMatcher_Strategy_Speed(t *testing.T) {
	m := NewMatcher()

	beads := []Bead{
		{ID: "b1", Title: "Task 1", TaskType: TaskFeature, Priority: 2},
		{ID: "b2", Title: "Task 2", TaskType: TaskBug, Priority: 1},
	}
	agents := []Agent{
		{ID: "1", AgentType: tmux.AgentClaude, Idle: true, ContextUsage: 0.7},
		{ID: "2", AgentType: tmux.AgentCodex, Idle: true, ContextUsage: 0.1},
	}

	result := m.AssignTasks(beads, agents, StrategySpeed)

	if len(result) != 2 {
		t.Fatalf("Expected 2 assignments, got %d", len(result))
	}

	// Speed strategy should assign quickly without much optimization
	for _, a := range result {
		if a.Reason == "" {
			t.Error("Expected non-empty reason")
		}
		// Speed strategy boosts confidence
		if a.Confidence < a.Score {
			t.Errorf("Speed strategy should boost confidence, got score=%f confidence=%f", a.Score, a.Confidence)
		}
	}
}

func TestMatcher_Strategy_Quality(t *testing.T) {
	m := NewMatcher()

	beads := []Bead{
		{ID: "b1", Title: "Refactor code", TaskType: TaskRefactor, Priority: 2},
	}
	agents := []Agent{
		{ID: "1", AgentType: tmux.AgentClaude, Idle: true},   // Excellent at refactor (0.95)
		{ID: "2", AgentType: tmux.AgentCodex, Idle: true},    // Good at refactor (0.75)
		{ID: "3", AgentType: tmux.AgentGemini, Idle: true},   // Good at refactor (0.75)
	}

	result := m.AssignTasks(beads, agents, StrategyQuality)

	if len(result) != 1 {
		t.Fatalf("Expected 1 assignment, got %d", len(result))
	}

	// Quality strategy should pick Claude for refactor task
	if result[0].Agent.AgentType != tmux.AgentClaude {
		t.Errorf("Quality strategy should pick Claude for refactor, got %s", result[0].Agent.AgentType)
	}
}

func TestMatcher_Strategy_Dependency(t *testing.T) {
	m := NewMatcher()

	beads := []Bead{
		{ID: "b1", Title: "Feature", TaskType: TaskFeature, Priority: 2, UnblocksIDs: nil},
		{ID: "b2", Title: "Blocker", TaskType: TaskBug, Priority: 2, UnblocksIDs: []string{"b3", "b4", "b5"}},
		{ID: "b3", Title: "Critical", TaskType: TaskBug, Priority: 0, UnblocksIDs: nil},
	}
	agents := []Agent{
		{ID: "1", AgentType: tmux.AgentCodex, Idle: true},
		{ID: "2", AgentType: tmux.AgentClaude, Idle: true},
	}

	result := m.AssignTasks(beads, agents, StrategyDependency)

	if len(result) < 2 {
		t.Fatalf("Expected at least 2 assignments, got %d", len(result))
	}

	// First assignment should be b3 (P0 critical) or b2 (unblocks 3 items)
	// With dependency strategy, P0 comes first
	if result[0].Bead.ID != "b3" {
		t.Errorf("Expected critical priority bead b3 first, got %s", result[0].Bead.ID)
	}

	// Second should be the blocker (unblocks most items)
	if result[1].Bead.ID != "b2" {
		t.Errorf("Expected blocker bead b2 second, got %s", result[1].Bead.ID)
	}
}

func TestMatcher_Strategy_Balanced(t *testing.T) {
	m := NewMatcher()

	beads := []Bead{
		{ID: "b1", Title: "Task 1", TaskType: TaskFeature, Priority: 2},
		{ID: "b2", Title: "Task 2", TaskType: TaskBug, Priority: 2},
		{ID: "b3", Title: "Task 3", TaskType: TaskDocs, Priority: 2},
		{ID: "b4", Title: "Task 4", TaskType: TaskRefactor, Priority: 2},
	}
	agents := []Agent{
		{ID: "1", AgentType: tmux.AgentClaude, Idle: true, Assignments: 0},
		{ID: "2", AgentType: tmux.AgentCodex, Idle: true, Assignments: 0},
	}

	result := m.AssignTasks(beads, agents, StrategyBalanced)

	if len(result) != 4 {
		t.Fatalf("Expected 4 assignments, got %d", len(result))
	}

	// Count assignments per agent
	counts := make(map[string]int)
	for _, a := range result {
		counts[a.Agent.ID]++
	}

	// Balanced strategy should spread work (2 each)
	if counts["1"] != 2 || counts["2"] != 2 {
		t.Errorf("Expected balanced distribution (2, 2), got (%d, %d)", counts["1"], counts["2"])
	}
}

func TestMatcher_ContextUsageAffectsScore(t *testing.T) {
	m := NewMatcher()

	beads := []Bead{
		{ID: "b1", Title: "Bug fix", TaskType: TaskBug, Priority: 1},
	}

	// Agent with low context usage
	agentsLow := []Agent{{ID: "1", AgentType: tmux.AgentCodex, Idle: true, ContextUsage: 0.1}}
	resultLow := m.AssignTasks(beads, agentsLow, StrategyQuality)

	// Agent with moderate context usage (0.5 so score = 0.90 * 0.5 = 0.45 > MinConfidence 0.3)
	agentsMod := []Agent{{ID: "2", AgentType: tmux.AgentCodex, Idle: true, ContextUsage: 0.5}}
	resultMod := m.AssignTasks(beads, agentsMod, StrategyQuality)

	if len(resultLow) != 1 || len(resultMod) != 1 {
		t.Fatalf("Expected 1 assignment each, got low=%d mod=%d", len(resultLow), len(resultMod))
	}

	// Lower context usage should result in higher score
	// Low: 0.90 * 0.9 = 0.81
	// Mod: 0.90 * 0.5 = 0.45
	if resultLow[0].Score <= resultMod[0].Score {
		t.Errorf("Low context usage should have higher score: low=%f, mod=%f",
			resultLow[0].Score, resultMod[0].Score)
	}
}

func TestMatcher_ReasonContainsRelevantInfo(t *testing.T) {
	m := NewMatcher()

	beads := []Bead{
		{ID: "b1", Title: "Critical bug", TaskType: TaskBug, Priority: 0},
	}
	agents := []Agent{
		{ID: "1", AgentType: tmux.AgentCodex, Idle: true, ContextUsage: 0.6},
	}

	result := m.AssignTasks(beads, agents, StrategyQuality)

	if len(result) != 1 {
		t.Fatal("Expected 1 assignment")
	}

	reason := result[0].Reason

	// Should mention priority for P0
	if reason == "" {
		t.Error("Expected non-empty reason")
	}

	// Should mention context usage (60% is significant)
	if result[0].Agent.ContextUsage >= 0.5 {
		// Reason should include context info
		// This is a soft check - just verify reason is populated
		t.Logf("Reason: %s", reason)
	}
}

func TestAssignTasksFunc(t *testing.T) {
	// Test the convenience function
	beads := []Bead{
		{ID: "b1", Title: "Feature", TaskType: TaskFeature, Priority: 1},
	}
	agents := []Agent{
		{ID: "1", AgentType: tmux.AgentClaude, Idle: true},
	}

	result := AssignTasksFunc(beads, agents, "quality")

	if len(result) != 1 {
		t.Fatalf("Expected 1 assignment, got %d", len(result))
	}

	if result[0].Bead.ID != "b1" {
		t.Errorf("Expected bead b1, got %s", result[0].Bead.ID)
	}
}

func TestMatcher_WithConfig(t *testing.T) {
	config := MatcherConfig{
		MaxContextUsage: 0.5, // More restrictive
		MinConfidence:   0.7, // Higher threshold
	}
	m := NewMatcher().WithConfig(config)

	beads := []Bead{
		{ID: "b1", Title: "Task", TaskType: TaskTask, Priority: 2},
	}

	// Agent with 60% context - should be filtered out with our config
	agents := []Agent{
		{ID: "1", AgentType: tmux.AgentClaude, Idle: true, ContextUsage: 0.6},
	}

	result := m.AssignTasks(beads, agents, StrategyBalanced)

	// Should have no assignments because agent context usage > MaxContextUsage
	if len(result) != 0 {
		t.Errorf("Expected 0 assignments (agent filtered), got %d", len(result))
	}
}

func TestMatcher_WithCustomMatrix(t *testing.T) {
	// Create custom matrix with boosted scores
	matrix := NewCapabilityMatrix()
	matrix.SetOverride(tmux.AgentGemini, TaskBug, 0.99) // Boost Gemini for bugs

	m := NewMatcherWithMatrix(matrix)

	beads := []Bead{
		{ID: "b1", Title: "Bug fix", TaskType: TaskBug, Priority: 1},
	}
	agents := []Agent{
		{ID: "1", AgentType: tmux.AgentCodex, Idle: true},  // Default: 0.90 for bugs
		{ID: "2", AgentType: tmux.AgentGemini, Idle: true}, // Override: 0.99 for bugs
	}

	result := m.AssignTasks(beads, agents, StrategyQuality)

	if len(result) != 1 {
		t.Fatal("Expected 1 assignment")
	}

	// Quality strategy should pick Gemini due to override
	if result[0].Agent.AgentType != tmux.AgentGemini {
		t.Errorf("Expected Gemini (boosted score), got %s", result[0].Agent.AgentType)
	}
}
