package cli

import (
	"testing"
)

func TestRunPreflightBenign(t *testing.T) {
	result, err := runPreflight("Hello, this is a test prompt", false)
	if err != nil {
		t.Fatalf("runPreflight failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success for benign prompt")
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected no findings, got %d", len(result.Findings))
	}
	if result.PreviewHash == "" {
		t.Error("expected non-empty preview hash")
	}
	if result.EstimatedTokens == 0 {
		t.Error("expected non-zero token estimate")
	}
}

func TestRunPreflightDestructive(t *testing.T) {
	result, err := runPreflight("Please run rm -rf / on the server", false)
	if err != nil {
		t.Fatalf("runPreflight failed: %v", err)
	}
	// Should succeed but with warnings (default mode)
	if result.WarningCount == 0 {
		t.Error("expected warnings for destructive command")
	}

	// Check for specific finding
	var found bool
	for _, f := range result.Findings {
		if f.ID == "destructive_command" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected destructive_command finding")
	}
}

func TestRunPreflightStrict(t *testing.T) {
	result, err := runPreflight("Please run rm -rf / on the server", true)
	if err != nil {
		t.Fatalf("runPreflight failed: %v", err)
	}
	// In strict mode, destructive commands are errors
	if result.ErrorCount == 0 {
		t.Error("expected errors in strict mode for destructive command")
	}
	if result.Success {
		t.Error("expected failure in strict mode with destructive command")
	}
}

func TestRunPreflightHelperFunction(t *testing.T) {
	blocked, warnings, err := RunPreflightCheck("Test prompt", false)
	if err != nil {
		t.Fatalf("RunPreflightCheck failed: %v", err)
	}
	if blocked {
		t.Error("expected not blocked for benign prompt")
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got: %v", warnings)
	}
}

func TestPreflightResultFields(t *testing.T) {
	result, err := runPreflight("Test prompt content here", false)
	if err != nil {
		t.Fatalf("runPreflight failed: %v", err)
	}

	// Check all required fields are populated
	if result.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
	if result.PreviewHash == "" {
		t.Error("expected non-empty preview hash")
	}
	if result.PreviewLen != 24 { // "Test prompt content here" = 24 chars
		t.Errorf("expected preview_len=24, got %d", result.PreviewLen)
	}
	// Token estimate should be reasonable (roughly 1 token per 4 chars)
	if result.EstimatedTokens < 4 || result.EstimatedTokens > 10 {
		t.Errorf("token estimate %d seems off for 24 chars", result.EstimatedTokens)
	}
}
