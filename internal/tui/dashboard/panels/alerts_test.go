package panels

import (
	"strings"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/alerts"
)

func TestNewAlertsPanel(t *testing.T) {
	panel := NewAlertsPanel()
	if panel == nil {
		t.Fatal("NewAlertsPanel returned nil")
	}
}

func TestAlertsPanelConfig(t *testing.T) {
	panel := NewAlertsPanel()
	cfg := panel.Config()

	if cfg.ID != "alerts" {
		t.Errorf("expected ID 'alerts', got %q", cfg.ID)
	}
	if cfg.Title != "Active Alerts" {
		t.Errorf("expected Title 'Active Alerts', got %q", cfg.Title)
	}
	if cfg.Priority != PriorityCritical {
		t.Errorf("expected PriorityCritical, got %v", cfg.Priority)
	}
	if cfg.RefreshInterval != 3*time.Second {
		t.Errorf("expected 3s refresh, got %v", cfg.RefreshInterval)
	}
	if cfg.Collapsible {
		t.Error("expected Collapsible to be false for alerts")
	}
}

func TestAlertsPanelSetSize(t *testing.T) {
	panel := NewAlertsPanel()
	panel.SetSize(80, 20)

	if panel.Width() != 80 {
		t.Errorf("expected width 80, got %d", panel.Width())
	}
	if panel.Height() != 20 {
		t.Errorf("expected height 20, got %d", panel.Height())
	}
}

func TestAlertsPanelFocusBlur(t *testing.T) {
	panel := NewAlertsPanel()

	panel.Focus()
	if !panel.IsFocused() {
		t.Error("expected IsFocused to be true after Focus()")
	}

	panel.Blur()
	if panel.IsFocused() {
		t.Error("expected IsFocused to be false after Blur()")
	}
}

func TestAlertsPanelSetData(t *testing.T) {
	panel := NewAlertsPanel()

	testAlerts := []alerts.Alert{
		{Severity: alerts.SeverityCritical, Message: "Critical error"},
		{Severity: alerts.SeverityWarning, Message: "Warning message"},
	}

	panel.SetData(testAlerts, nil)

	if len(panel.alerts) != 2 {
		t.Errorf("expected 2 alerts, got %d", len(panel.alerts))
	}
}

func TestAlertsPanelKeybindings(t *testing.T) {
	panel := NewAlertsPanel()
	bindings := panel.Keybindings()

	if len(bindings) == 0 {
		t.Error("expected non-empty keybindings")
	}

	// Check for expected actions
	actions := make(map[string]bool)
	for _, b := range bindings {
		actions[b.Action] = true
	}

	if !actions["dismiss"] {
		t.Error("expected 'dismiss' action in keybindings")
	}
	if !actions["ack_all"] {
		t.Error("expected 'ack_all' action in keybindings")
	}
}

func TestAlertsPanelInit(t *testing.T) {
	panel := NewAlertsPanel()
	cmd := panel.Init()
	if cmd != nil {
		t.Error("expected Init() to return nil")
	}
}

func TestAlertsPanelViewZeroWidth(t *testing.T) {
	panel := NewAlertsPanel()
	panel.SetSize(0, 10)

	view := panel.View()
	if view != "" {
		t.Errorf("expected empty view for zero width, got: %s", view)
	}
}

func TestAlertsPanelViewNoAlerts(t *testing.T) {
	panel := NewAlertsPanel()
	panel.SetSize(80, 20)
	panel.SetData([]alerts.Alert{}, nil)

	view := panel.View()

	if !strings.Contains(view, "System Healthy") {
		t.Error("expected view to contain 'System Healthy' when no alerts")
	}
	if !strings.Contains(view, "No active alerts") {
		t.Error("expected view to contain 'No active alerts' when no alerts")
	}
}

func TestAlertsPanelViewWithAlerts(t *testing.T) {
	panel := NewAlertsPanel()
	panel.SetSize(80, 20)

	testAlerts := []alerts.Alert{
		{Severity: alerts.SeverityCritical, Message: "Critical error occurred"},
		{Severity: alerts.SeverityWarning, Message: "Warning about disk space"},
		{Severity: alerts.SeverityInfo, Message: "Informational message"},
	}
	panel.SetData(testAlerts, nil)

	view := panel.View()

	// Should contain title
	if !strings.Contains(view, "Active Alerts") {
		t.Error("expected view to contain title 'Active Alerts'")
	}

	// Should contain stats
	if !strings.Contains(view, "Crit:") || !strings.Contains(view, "Warn:") || !strings.Contains(view, "Info:") {
		t.Error("expected view to contain alert stats")
	}
}

func TestAlertsPanelViewGroupsBySeverity(t *testing.T) {
	panel := NewAlertsPanel()
	panel.SetSize(120, 30)

	testAlerts := []alerts.Alert{
		{Severity: alerts.SeverityCritical, Message: "First critical"},
		{Severity: alerts.SeverityCritical, Message: "Second critical"},
		{Severity: alerts.SeverityWarning, Message: "First warning"},
		{Severity: alerts.SeverityInfo, Message: "First info"},
	}
	panel.SetData(testAlerts, nil)

	view := panel.View()

	// Stats should show correct counts
	if !strings.Contains(view, "Crit: 2") {
		t.Error("expected 'Crit: 2' in view")
	}
	if !strings.Contains(view, "Warn: 1") {
		t.Error("expected 'Warn: 1' in view")
	}
	if !strings.Contains(view, "Info: 1") {
		t.Error("expected 'Info: 1' in view")
	}
}

func TestAlertsPanelUpdate(t *testing.T) {
	panel := NewAlertsPanel()

	newModel, cmd := panel.Update(nil)

	if newModel != panel {
		t.Error("expected Update to return same model")
	}
	if cmd != nil {
		t.Error("expected Update to return nil cmd")
	}
}
