package ensemble

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStateStore_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.db")

	store, err := NewStateStore(path)
	if err != nil {
		t.Fatalf("NewStateStore error: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	createdAt := time.Now().UTC().Truncate(time.Second)
	synthAt := createdAt.Add(2 * time.Minute)
	completedAt := createdAt.Add(5 * time.Minute)

	session := &EnsembleSession{
		SessionName:       "test-session",
		Question:          "What is the issue?",
		PresetUsed:        "diagnosis",
		Status:            EnsembleActive,
		SynthesisStrategy: StrategyConsensus,
		CreatedAt:         createdAt,
		SynthesizedAt:     &synthAt,
		SynthesisOutput:   "summary",
		Error:             "",
		Assignments: []ModeAssignment{
			{
				ModeID:      "deductive",
				PaneName:    "pane-1",
				AgentType:   "cc",
				Status:      AssignmentActive,
				OutputPath:  "/tmp/out.txt",
				AssignedAt:  createdAt,
				CompletedAt: &completedAt,
			},
		},
	}

	if err := store.Save(session); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := store.Load("test-session")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected loaded session, got nil")
	}

	if loaded.Question != session.Question {
		t.Errorf("Question = %q, want %q", loaded.Question, session.Question)
	}
	if loaded.PresetUsed != session.PresetUsed {
		t.Errorf("PresetUsed = %q, want %q", loaded.PresetUsed, session.PresetUsed)
	}
	if loaded.Status != session.Status {
		t.Errorf("Status = %q, want %q", loaded.Status, session.Status)
	}
	if loaded.SynthesisStrategy != session.SynthesisStrategy {
		t.Errorf("SynthesisStrategy = %q, want %q", loaded.SynthesisStrategy, session.SynthesisStrategy)
	}
	if !loaded.CreatedAt.Equal(session.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", loaded.CreatedAt, session.CreatedAt)
	}
	if loaded.SynthesizedAt == nil || !loaded.SynthesizedAt.Equal(*session.SynthesizedAt) {
		t.Errorf("SynthesizedAt = %v, want %v", loaded.SynthesizedAt, session.SynthesizedAt)
	}
	if loaded.SynthesisOutput != session.SynthesisOutput {
		t.Errorf("SynthesisOutput = %q, want %q", loaded.SynthesisOutput, session.SynthesisOutput)
	}

	if len(loaded.Assignments) != 1 {
		t.Fatalf("Assignments len = %d, want 1", len(loaded.Assignments))
	}
	assignment := loaded.Assignments[0]
	if assignment.ModeID != "deductive" {
		t.Errorf("Assignment ModeID = %q, want deductive", assignment.ModeID)
	}
	if assignment.PaneName != "pane-1" {
		t.Errorf("Assignment PaneName = %q, want pane-1", assignment.PaneName)
	}
	if assignment.AgentType != "cc" {
		t.Errorf("Assignment AgentType = %q, want cc", assignment.AgentType)
	}
	if assignment.Status != AssignmentActive {
		t.Errorf("Assignment Status = %q, want %q", assignment.Status, AssignmentActive)
	}
	if assignment.OutputPath != "/tmp/out.txt" {
		t.Errorf("Assignment OutputPath = %q, want /tmp/out.txt", assignment.OutputPath)
	}
	if assignment.AssignedAt.IsZero() {
		t.Errorf("Assignment AssignedAt should be set")
	}
	if assignment.CompletedAt == nil || !assignment.CompletedAt.Equal(completedAt) {
		t.Errorf("Assignment CompletedAt = %v, want %v", assignment.CompletedAt, completedAt)
	}
}

func TestOutputCapture_ExtractYAML_PrefersValidBlock(t *testing.T) {
	capture := NewOutputCapture(nil)
	raw := strings.Join([]string{
		"noise before",
		"```yaml",
		": bad yaml",
		"```",
		"more noise",
		"```yaml",
		"mode_id: deductive",
		"thesis: something",
		"```",
	}, "\n")

	block, ok := capture.extractYAML(raw)
	if !ok {
		t.Fatal("expected YAML block to be found")
	}
	if !strings.Contains(block, "mode_id: deductive") {
		t.Fatalf("expected valid YAML block, got: %q", block)
	}
}

func TestOutputCapture_CapturePane_Empty(t *testing.T) {
	capture := NewOutputCapture(nil)
	if _, err := capture.capturePane(""); err == nil {
		t.Fatal("expected error for empty pane")
	}
}

func TestStateStore_UpdateListDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.db")

	store, err := NewStateStore(path)
	if err != nil {
		t.Fatalf("NewStateStore error: %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &EnsembleSession{
		SessionName:       "update-session",
		Question:          "Question",
		Status:            EnsembleActive,
		SynthesisStrategy: StrategyConsensus,
		CreatedAt:         time.Now().UTC(),
		Assignments: []ModeAssignment{
			{
				ModeID:    "deductive",
				PaneName:  "pane-1",
				AgentType: "cc",
				Status:    AssignmentPending,
			},
		},
	}

	if err := store.Save(session); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	if err := store.UpdateStatus(session.SessionName, EnsembleComplete); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if err := store.UpdateAssignmentStatus(session.SessionName, "deductive", AssignmentDone); err != nil {
		t.Fatalf("UpdateAssignmentStatus error: %v", err)
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List len = %d, want 1", len(list))
	}
	if list[0].Status != EnsembleComplete {
		t.Fatalf("List status = %q, want %q", list[0].Status, EnsembleComplete)
	}
	if len(list[0].Assignments) != 1 || list[0].Assignments[0].Status != AssignmentDone {
		t.Fatalf("assignment status = %q, want %q", list[0].Assignments[0].Status, AssignmentDone)
	}

	if err := store.Delete(session.SessionName); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if _, err := store.Load(session.SessionName); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load after delete error = %v, want os.ErrNotExist", err)
	}
}
