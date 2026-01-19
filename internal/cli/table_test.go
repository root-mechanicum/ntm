package cli

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestANSIWidthCalculations verifies that lipgloss.Width() correctly ignores ANSI codes
// and that runeWidth() uses lipgloss.Width() properly.
func TestANSIWidthCalculations(t *testing.T) {
	// Verify lipgloss.Width() correctly ignores ANSI codes
	plain := "Hello"
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render("Hello")

	plainWidth := lipgloss.Width(plain)
	styledWidth := lipgloss.Width(styled)

	if styledWidth != plainWidth {
		t.Errorf("styled width %d != plain width %d", styledWidth, plainWidth)
	}

	// Verify runeWidth() uses lipgloss.Width()
	w := runeWidth(styled)
	if w != 5 {
		t.Errorf("runeWidth(styled) = %d, want 5", w)
	}
}

// TestStyledStringWidths verifies various styled string scenarios
func TestStyledStringWidths(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int
	}{
		{"plain", "hello", 5},
		{"bold", lipgloss.NewStyle().Bold(true).Render("hello"), 5},
		{"colored", lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render("hello"), 5},
		{"background", lipgloss.NewStyle().Background(lipgloss.Color("240")).Render("hello"), 5},
		{"multi-style", lipgloss.NewStyle().Bold(true).Italic(true).Foreground(lipgloss.Color("212")).Render("hello"), 5},
		{"raw ANSI", "\x1b[31mred\x1b[0m", 3},
		{"emoji", "🎉", 2},
		{"CJK", "日本語", 6},
		{"mixed", "Hello 🎉 世界", 13}, // 6 + 2 + 5
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lipgloss.Width(tc.input)
			if got != tc.want {
				t.Errorf("lipgloss.Width(%q) = %d, want %d", tc.input, got, tc.want)
			}

			// Also verify runeWidth() wrapper
			gotRuneWidth := runeWidth(tc.input)
			if gotRuneWidth != tc.want {
				t.Errorf("runeWidth(%q) = %d, want %d", tc.input, gotRuneWidth, tc.want)
			}
		})
	}
}

// TestTableColumnAlignment verifies that tables with styled content align correctly
func TestTableColumnAlignment(t *testing.T) {
	table := NewStyledTable("Name", "Value")
	// Add rows with styled content
	table.AddRow(lipgloss.NewStyle().Bold(true).Render("Bold"), "plain")
	table.AddRow("plain", lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render("colored"))
	table.AddRow(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render("Mixed"), "plain")

	rendered := table.Render()
	lines := strings.Split(rendered, "\n")

	// Collect non-empty line widths
	var widths []int
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "╭") &&
			!strings.HasPrefix(strings.TrimSpace(line), "╰") &&
			!strings.HasPrefix(strings.TrimSpace(line), "├") {
			widths = append(widths, lipgloss.Width(line))
		}
	}

	// Check all data row widths are equal (table rows should align)
	if len(widths) > 1 {
		for i := 1; i < len(widths); i++ {
			if widths[i] != widths[0] {
				t.Errorf("line %d width %d != first line width %d", i, widths[i], widths[0])
			}
		}
	}
}

// TestPadRightWithStyledContent verifies padding works correctly with styled strings
func TestPadRightWithStyledContent(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render("Hi")
	padded := padRight(styled, 10)

	// Should be 10 visual columns total
	paddedWidth := lipgloss.Width(padded)
	if paddedWidth != 10 {
		t.Errorf("padRight visual width = %d, want 10", paddedWidth)
	}
}
