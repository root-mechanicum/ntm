package panels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dicklesworthstone/ntm/internal/tui/theme"
)

// TickerData holds the data displayed in the ticker
type TickerData struct {
	// Fleet status
	TotalAgents  int
	ActiveAgents int
	ClaudeCount  int
	CodexCount   int
	GeminiCount  int

	// Alerts
	CriticalAlerts int
	WarningAlerts  int
	InfoAlerts     int

	// Beads
	ReadyBeads      int
	InProgressBeads int
	BlockedBeads    int

	// Mail
	UnreadMessages int
	ActiveLocks    int
	MailConnected  bool
}

// TickerPanel displays a scrolling status bar at the bottom of the dashboard
type TickerPanel struct {
	width    int
	height   int
	focused  bool
	data     TickerData
	theme    theme.Theme
	offset   int // Current scroll offset for animation
	animTick int // Animation tick counter
}

// NewTickerPanel creates a new ticker panel
func NewTickerPanel() *TickerPanel {
	return &TickerPanel{
		theme:  theme.Current(),
		height: 1, // Ticker is typically single-line
	}
}

// Init implements tea.Model
func (m *TickerPanel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *TickerPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// SetSize sets the panel dimensions
func (m *TickerPanel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Focus marks the panel as focused
func (m *TickerPanel) Focus() {
	m.focused = true
}

// Blur marks the panel as unfocused
func (m *TickerPanel) Blur() {
	m.focused = false
}

// SetData updates the panel data
func (m *TickerPanel) SetData(data TickerData) {
	m.data = data
}

// SetAnimTick updates the animation tick for scrolling
func (m *TickerPanel) SetAnimTick(tick int) {
	m.animTick = tick
	// Update scroll offset every 2 ticks (~200ms at 100ms tick rate)
	m.offset = tick / 2
}

// View renders the panel
func (m *TickerPanel) View() string {
	t := m.theme

	if m.width <= 0 {
		return ""
	}

	// Build ticker segments
	segments := m.buildSegments()

	// Join all segments with separator
	separator := lipgloss.NewStyle().
		Foreground(t.Surface2).
		Render(" | ")

	fullText := strings.Join(segments, separator)

	// Calculate visible portion based on scroll offset
	visibleText := m.scrollText(fullText)

	// Style the ticker bar
	tickerStyle := lipgloss.NewStyle().
		Width(m.width).
		Background(t.Surface0).
		Foreground(t.Text)

	if m.focused {
		tickerStyle = tickerStyle.
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(t.Primary)
	}

	return tickerStyle.Render(visibleText)
}

// buildSegments creates the ticker content segments
func (m *TickerPanel) buildSegments() []string {
	t := m.theme
	var segments []string

	// Fleet segment
	fleetSegment := m.buildFleetSegment(t)
	segments = append(segments, fleetSegment)

	// Alerts segment
	alertsSegment := m.buildAlertsSegment(t)
	segments = append(segments, alertsSegment)

	// Beads segment
	beadsSegment := m.buildBeadsSegment(t)
	segments = append(segments, beadsSegment)

	// Mail segment
	mailSegment := m.buildMailSegment(t)
	segments = append(segments, mailSegment)

	return segments
}

// buildFleetSegment creates the Fleet status segment
func (m *TickerPanel) buildFleetSegment(t theme.Theme) string {
	var parts []string

	// Fleet icon and total
	fleetLabel := lipgloss.NewStyle().
		Foreground(t.Blue).
		Bold(true).
		Render("Fleet")

	activeStatus := fmt.Sprintf("%d/%d", m.data.ActiveAgents, m.data.TotalAgents)
	activeStyled := lipgloss.NewStyle().Foreground(t.Text).Render(activeStatus)

	parts = append(parts, fleetLabel+": "+activeStyled)

	// Agent type breakdown (if any agents exist)
	if m.data.TotalAgents > 0 {
		var agentParts []string

		if m.data.ClaudeCount > 0 {
			ccStyled := lipgloss.NewStyle().
				Foreground(t.Claude).
				Render(fmt.Sprintf("C:%d", m.data.ClaudeCount))
			agentParts = append(agentParts, ccStyled)
		}

		if m.data.CodexCount > 0 {
			codStyled := lipgloss.NewStyle().
				Foreground(t.Codex).
				Render(fmt.Sprintf("X:%d", m.data.CodexCount))
			agentParts = append(agentParts, codStyled)
		}

		if m.data.GeminiCount > 0 {
			gmiStyled := lipgloss.NewStyle().
				Foreground(t.Gemini).
				Render(fmt.Sprintf("G:%d", m.data.GeminiCount))
			agentParts = append(agentParts, gmiStyled)
		}

		if len(agentParts) > 0 {
			parts = append(parts, "("+strings.Join(agentParts, " ")+")")
		}
	}

	return strings.Join(parts, " ")
}

// buildAlertsSegment creates the Alerts status segment
func (m *TickerPanel) buildAlertsSegment(t theme.Theme) string {
	alertLabel := lipgloss.NewStyle().
		Foreground(t.Pink).
		Bold(true).
		Render("Alerts")

	totalAlerts := m.data.CriticalAlerts + m.data.WarningAlerts + m.data.InfoAlerts

	if totalAlerts == 0 {
		okStyled := lipgloss.NewStyle().
			Foreground(t.Green).
			Render("OK")
		return alertLabel + ": " + okStyled
	}

	var alertParts []string

	if m.data.CriticalAlerts > 0 {
		critStyled := lipgloss.NewStyle().
			Foreground(t.Red).
			Bold(true).
			Render(fmt.Sprintf("%d!", m.data.CriticalAlerts))
		alertParts = append(alertParts, critStyled)
	}

	if m.data.WarningAlerts > 0 {
		warnStyled := lipgloss.NewStyle().
			Foreground(t.Yellow).
			Render(fmt.Sprintf("%dw", m.data.WarningAlerts))
		alertParts = append(alertParts, warnStyled)
	}

	if m.data.InfoAlerts > 0 {
		infoStyled := lipgloss.NewStyle().
			Foreground(t.Blue).
			Render(fmt.Sprintf("%di", m.data.InfoAlerts))
		alertParts = append(alertParts, infoStyled)
	}

	return alertLabel + ": " + strings.Join(alertParts, "/")
}

// buildBeadsSegment creates the Beads status segment
func (m *TickerPanel) buildBeadsSegment(t theme.Theme) string {
	beadsLabel := lipgloss.NewStyle().
		Foreground(t.Green).
		Bold(true).
		Render("Beads")

	var beadParts []string

	// Ready (most important)
	readyStyled := lipgloss.NewStyle().
		Foreground(t.Green).
		Render(fmt.Sprintf("R:%d", m.data.ReadyBeads))
	beadParts = append(beadParts, readyStyled)

	// In Progress
	if m.data.InProgressBeads > 0 {
		ipStyled := lipgloss.NewStyle().
			Foreground(t.Blue).
			Render(fmt.Sprintf("I:%d", m.data.InProgressBeads))
		beadParts = append(beadParts, ipStyled)
	}

	// Blocked
	if m.data.BlockedBeads > 0 {
		blockedStyled := lipgloss.NewStyle().
			Foreground(t.Red).
			Render(fmt.Sprintf("B:%d", m.data.BlockedBeads))
		beadParts = append(beadParts, blockedStyled)
	}

	return beadsLabel + ": " + strings.Join(beadParts, " ")
}

// buildMailSegment creates the Mail status segment
func (m *TickerPanel) buildMailSegment(t theme.Theme) string {
	mailLabel := lipgloss.NewStyle().
		Foreground(t.Lavender).
		Bold(true).
		Render("Mail")

	if !m.data.MailConnected {
		offlineStyled := lipgloss.NewStyle().
			Foreground(t.Overlay).
			Italic(true).
			Render("offline")
		return mailLabel + ": " + offlineStyled
	}

	var mailParts []string

	// Unread messages
	if m.data.UnreadMessages > 0 {
		unreadStyled := lipgloss.NewStyle().
			Foreground(t.Yellow).
			Bold(true).
			Render(fmt.Sprintf("%d unread", m.data.UnreadMessages))
		mailParts = append(mailParts, unreadStyled)
	} else {
		noMailStyled := lipgloss.NewStyle().
			Foreground(t.Green).
			Render("0 unread")
		mailParts = append(mailParts, noMailStyled)
	}

	// Active locks
	if m.data.ActiveLocks > 0 {
		locksStyled := lipgloss.NewStyle().
			Foreground(t.Peach).
			Render(fmt.Sprintf("%d locks", m.data.ActiveLocks))
		mailParts = append(mailParts, locksStyled)
	}

	return mailLabel + ": " + strings.Join(mailParts, " ")
}

// scrollText handles the horizontal scrolling animation
func (m *TickerPanel) scrollText(text string) string {
	// Get the display width of the text (using rune count as approximation)
	textRunes := []rune(text)
	textLen := len(textRunes)

	// If text fits in width, center it
	if textLen <= m.width {
		padding := (m.width - textLen) / 2
		return strings.Repeat(" ", padding) + text + strings.Repeat(" ", m.width-textLen-padding)
	}

	// For scrolling, we need to handle wrap-around
	// Add some padding spaces between end and start for smooth loop
	paddedText := text + "    " + text // Duplicate for seamless loop
	paddedRunes := []rune(paddedText)
	paddedLen := len(paddedRunes)

	// Calculate scroll position (wrap around)
	scrollPos := m.offset % (textLen + 4) // +4 for the padding spaces

	// Extract visible portion
	endPos := scrollPos + m.width
	if endPos > paddedLen {
		endPos = paddedLen
	}

	visible := string(paddedRunes[scrollPos:endPos])

	// Pad if needed (shouldn't happen with duplicated text, but safety)
	if len([]rune(visible)) < m.width {
		visible += strings.Repeat(" ", m.width-len([]rune(visible)))
	}

	return visible
}

// GetHeight returns the preferred height for the ticker (single line)
func (m *TickerPanel) GetHeight() int {
	return 1
}
