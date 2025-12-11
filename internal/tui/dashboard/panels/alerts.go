package panels

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dicklesworthstone/ntm/internal/alerts"
	"github.com/Dicklesworthstone/ntm/internal/tui/layout"
	"github.com/Dicklesworthstone/ntm/internal/tui/theme"
)

// alertsConfig returns the configuration for the alerts panel
func alertsConfig() PanelConfig {
	return PanelConfig{
		ID:              "alerts",
		Title:           "Active Alerts",
		Priority:        PriorityCritical, // Alerts are highest priority
		RefreshInterval: 3 * time.Second,  // Fast refresh for alerts
		MinWidth:        25,
		MinHeight:       6,
		Collapsible:     false, // Don't hide alerts
	}
}

type AlertsPanel struct {
	PanelBase
	alerts []alerts.Alert
	err    error
}

func NewAlertsPanel() *AlertsPanel {
	return &AlertsPanel{
		PanelBase: NewPanelBase(alertsConfig()),
	}
}

func (m *AlertsPanel) SetData(alertList []alerts.Alert, err error) {
	m.alerts = alertList
	m.err = err
}

// HasError returns true if there's an active error
func (m *AlertsPanel) HasError() bool {
	return m.err != nil
}

func (m *AlertsPanel) Init() tea.Cmd {
	return nil
}

func (m *AlertsPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// Keybindings returns alerts panel specific shortcuts
func (m *AlertsPanel) Keybindings() []Keybinding {
	return []Keybinding{
		{
			Key:         key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "dismiss")),
			Description: "Dismiss selected alert",
			Action:      "dismiss",
		},
		{
			Key:         key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "ack all")),
			Description: "Acknowledge all alerts",
			Action:      "ack_all",
		},
	}
}

func (m *AlertsPanel) View() string {
	t := theme.Current()
	w, h := m.Width(), m.Height()

	if w <= 0 {
		return ""
	}

	borderColor := t.Surface1
	if m.IsFocused() {
		borderColor = t.Pink
	}

	// Build header with error badge if needed
	title := m.Config().Title
	if m.err != nil {
		errorBadge := lipgloss.NewStyle().
			Background(t.Red).
			Foreground(t.Base).
			Bold(true).
			Padding(0, 1).
			Render("⚠ Error")
		title = title + " " + errorBadge
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Text).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(borderColor).
		Width(w).
		Padding(0, 1).
		Render(title)
	var content strings.Builder
	content.WriteString(header + "\n")

	// Show error message if present
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(t.Red).
			Italic(true).
			Padding(0, 1)
		errMsg := m.err.Error()
		if len(errMsg) > w-4 {
			errMsg = errMsg[:w-7] + "..."
		}
		content.WriteString(errorStyle.Render("⚠ "+errMsg) + "\n\n")
	}

	if len(m.alerts) == 0 && m.err == nil {
		content.WriteString("\n  " + lipgloss.NewStyle().Foreground(t.Green).Render("✓ System Healthy") + "\n")
		content.WriteString("  " + lipgloss.NewStyle().Foreground(t.Subtext).Render("No active alerts") + "\n")
		return content.String()
	} else if len(m.alerts) == 0 {
		return content.String()
	}

	// Group by severity
	var critical, warning, info []alerts.Alert
	for _, a := range m.alerts {
		switch a.Severity {
		case alerts.SeverityCritical:
			critical = append(critical, a)
		case alerts.SeverityWarning:
			warning = append(warning, a)
		default:
			info = append(info, a)
		}
	}

	// Stats row
	stats := fmt.Sprintf("Crit: %d  Warn: %d  Info: %d", len(critical), len(warning), len(info))
	statsStyled := lipgloss.NewStyle().Foreground(t.Subtext).Padding(0, 1).Render(stats)
	content.WriteString(statsStyled + "\n\n")

	// Calculate display limit based on height
	// Header + Stats + 2 newlines = ~4 lines
	// Each item = 1 line
	availableLines := h - 4
	if availableLines < 0 {
		availableLines = 0
	}

	// Render alerts (Critical > Warning > Info)
	count := 0

	renderList := func(list []alerts.Alert, color lipgloss.Color, icon string) {
		for _, a := range list {
			if count >= availableLines {
				return
			}
			msg := layout.TruncateRunes(a.Message, w-6, "…")
			line := fmt.Sprintf("  %s %s", icon, msg)
			content.WriteString(lipgloss.NewStyle().Foreground(color).Render(line) + "\n")
			count++
		}
	}

	if len(critical) > 0 {
		renderList(critical, t.Red, "✗")
	}
	if len(warning) > 0 {
		renderList(warning, t.Yellow, "⚠")
	}
	if len(info) > 0 {
		renderList(info, t.Blue, "ℹ")
	}

	return content.String()
}
