package ensemble

import "testing"

func TestValidationReport_HasErrors(t *testing.T) {
	report := NewValidationReport()
	if report.HasErrors() {
		t.Fatal("expected HasErrors false for empty report")
	}
	report.add(ValidationIssue{Code: "ERR", Severity: SeverityError, Message: "boom"})
	if !report.HasErrors() {
		t.Fatal("expected HasErrors true after adding error")
	}
}

func TestValidateModeIDs_EmptyAndDuplicate(t *testing.T) {
	catalog := testModeCatalog(t)
	report := NewValidationReport()
	validateModeIDs(nil, catalog, false, report)
	if !report.HasErrors() {
		t.Fatal("expected errors for empty mode list")
	}

	report = NewValidationReport()
	validateModeIDs([]string{"deductive", "deductive"}, catalog, false, report)
	if !report.HasErrors() {
		t.Fatal("expected errors for duplicate modes")
	}
}

func TestSuggestModeCodes(t *testing.T) {
	catalog := testModeCatalog(t)
	suggestions := suggestModeCodes("A", catalog)
	if len(suggestions) == 0 {
		t.Fatal("expected suggestions for mode codes")
	}
}

func TestValidateEnsemblePresets_Extensions(t *testing.T) {
	catalog := testModeCatalog(t)
	presets := []EnsemblePreset{
		{
			Name:        "base",
			Description: "base",
			Modes:       []ModeRef{ModeRefFromID("deductive"), ModeRefFromID("abductive")},
		},
		{
			Name:        "child",
			Description: "child",
			Extends:     "bas",
			Modes:       []ModeRef{ModeRefFromID("deductive"), ModeRefFromID("abductive")},
		},
	}

	report := ValidateEnsemblePresets(presets, catalog)
	if report == nil || !report.HasErrors() {
		t.Fatal("expected validation errors for missing extends")
	}
}

func TestValidateBudgetConfig_TooHigh(t *testing.T) {
	report := NewValidationReport()
	validateBudgetConfig(BudgetConfig{
		MaxTokensPerMode: 500000,
		MaxTotalTokens:   2000000,
	}, report)
	if !report.HasErrors() {
		t.Fatal("expected budget validation errors")
	}
}
