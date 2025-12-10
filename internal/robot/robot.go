// Package robot provides machine-readable output for AI agents and automation.
// Use --robot-* flags to get JSON output suitable for piping to other tools.
package robot

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/tmux"
)

// Build info - these will be set by the caller from cli package
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "unknown"
)

// SessionInfo contains machine-readable session information
type SessionInfo struct {
	Name      string     `json:"name"`
	Exists    bool       `json:"exists"`
	Attached  bool       `json:"attached,omitempty"`
	Windows   int        `json:"windows,omitempty"`
	Panes     int        `json:"panes,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	Agents    []Agent    `json:"agents,omitempty"`
}

// Agent represents an AI agent in a session
type Agent struct {
	Type     string `json:"type"` // claude, codex, gemini
	Pane     string `json:"pane"`
	Window   int    `json:"window"`
	PaneIdx  int    `json:"pane_idx"`
	IsActive bool   `json:"is_active"`
}

// SystemInfo contains system and runtime information
type SystemInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	TmuxOK    bool   `json:"tmux_available"`
}

// StatusOutput is the structured output for robot-status
type StatusOutput struct {
	GeneratedAt time.Time     `json:"generated_at"`
	System      SystemInfo    `json:"system"`
	Sessions    []SessionInfo `json:"sessions"`
	Summary     StatusSummary `json:"summary"`
}

// StatusSummary provides aggregate stats
type StatusSummary struct {
	TotalSessions   int `json:"total_sessions"`
	TotalAgents     int `json:"total_agents"`
	AttachedCount   int `json:"attached_count"`
	ClaudeCount     int `json:"claude_count"`
	CodexCount      int `json:"codex_count"`
	GeminiCount     int `json:"gemini_count"`
	CursorCount     int `json:"cursor_count"`
	WindsurfCount   int `json:"windsurf_count"`
	AiderCount      int `json:"aider_count"`
}

// PlanOutput provides an execution plan for what can be done
type PlanOutput struct {
	GeneratedAt   time.Time    `json:"generated_at"`
	Recommendation string      `json:"recommendation"`
	Actions       []PlanAction `json:"actions"`
	Warnings      []string     `json:"warnings,omitempty"`
}

// PlanAction is a suggested action
type PlanAction struct {
	Priority    int      `json:"priority"` // 1=high, 2=medium, 3=low
	Command     string   `json:"command"`
	Description string   `json:"description"`
	Args        []string `json:"args,omitempty"`
}

// PrintHelp outputs AI agent help documentation
func PrintHelp() {
	help := `ntm (Named Tmux Manager) AI Agent Interface
=============================================
This tool helps AI agents manage tmux sessions with multiple coding assistants.

Commands for AI Agents:
-----------------------

--robot-status
    Outputs JSON with all session information and agent counts.
    Key fields:
    - sessions: Array of active sessions with their agents
    - summary: Aggregate counts (total_agents, claude_count, etc.)
    - system: Version, OS, tmux availability

--robot-plan
    Outputs a recommended execution plan based on current state.
    Key fields:
    - recommendation: What to do first
    - actions: Prioritized list of commands to run
    - warnings: Issues that need attention

--robot-sessions
    Outputs minimal session list for quick lookup.
    Faster than --robot-status when you only need names.

--robot-version
    Outputs version info as JSON.

Common Workflows:
-----------------

1) Start a coding session:
   ntm spawn myproject --cc=2 --cod=1 --gem=1 --json

2) Check session state:
   ntm status --robot-status

3) Send a prompt to all agents:
   ntm send myproject --all "implement feature X"

4) Get output from a specific agent:
   ntm copy myproject:1 --last=50

Tips for AI Agents:
-------------------
- Use --json flag on spawn/create for structured output
- Parse ntm status --robot-status to understand current state
- Use ntm send --all for broadcast, --pane=N for targeted
- Output is always UTF-8 JSON, one object per line where applicable
`
	fmt.Println(help)
}

// PrintStatus outputs machine-readable status
func PrintStatus() error {
	output := StatusOutput{
		GeneratedAt: time.Now().UTC(),
		System: SystemInfo{
			Version:   Version,
			Commit:    Commit,
			BuildDate: Date,
			GoVersion: runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
			TmuxOK:    tmux.IsInstalled(),
		},
		Sessions: []SessionInfo{},
		Summary:  StatusSummary{},
	}

	// Get all sessions
	sessions, err := tmux.ListSessions()
	if err != nil {
		// tmux not running is not an error for status
		return encodeJSON(output)
	}

	for _, sess := range sessions {
		info := SessionInfo{
			Name:     sess.Name,
			Exists:   true,
			Attached: sess.Attached,
			Windows:  sess.Windows,
		}

		// Try to get agents from panes
		panes, err := tmux.GetPanes(sess.Name)
		if err == nil {
			info.Panes = len(panes)
			for _, pane := range panes {
				agent := Agent{
					Pane:     pane.ID,
					Window:   0, // GetPanes doesn't include window index
					PaneIdx:  pane.Index,
					IsActive: pane.Active,
				}

				// Try to detect agent type from pane title or content
				agent.Type = detectAgentType(pane.Title)
				info.Agents = append(info.Agents, agent)

				// Update summary counts
				switch agent.Type {
				case "claude":
					output.Summary.ClaudeCount++
				case "codex":
					output.Summary.CodexCount++
				case "gemini":
					output.Summary.GeminiCount++
				case "cursor":
					output.Summary.CursorCount++
				case "windsurf":
					output.Summary.WindsurfCount++
				case "aider":
					output.Summary.AiderCount++
				}
				output.Summary.TotalAgents++
			}
		}

		output.Sessions = append(output.Sessions, info)
		output.Summary.TotalSessions++
		if sess.Attached {
			output.Summary.AttachedCount++
		}
	}

	return encodeJSON(output)
}

// PrintVersion outputs version as JSON
func PrintVersion() error {
	info := struct {
		Version   string `json:"version"`
		Commit    string `json:"commit"`
		BuildDate string `json:"build_date"`
		BuiltBy   string `json:"built_by"`
		GoVersion string `json:"go_version"`
		OS        string `json:"os"`
		Arch      string `json:"arch"`
	}{
		Version:   Version,
		Commit:    Commit,
		BuildDate: Date,
		BuiltBy:   BuiltBy,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
	return encodeJSON(info)
}

// PrintSessions outputs minimal session list
func PrintSessions() error {
	sessions, err := tmux.ListSessions()
	if err != nil {
		return encodeJSON([]SessionInfo{})
	}

	output := make([]SessionInfo, 0, len(sessions))
	for _, sess := range sessions {
		output = append(output, SessionInfo{
			Name:     sess.Name,
			Exists:   true,
			Attached: sess.Attached,
			Windows:  sess.Windows,
		})
	}
	return encodeJSON(output)
}

// PrintPlan outputs an execution plan
func PrintPlan() error {
	plan := PlanOutput{
		GeneratedAt: time.Now().UTC(),
		Actions:     []PlanAction{},
	}

	// Check tmux availability
	if !tmux.IsInstalled() {
		plan.Recommendation = "Install tmux first"
		plan.Warnings = append(plan.Warnings, "tmux is not installed or not in PATH")
		plan.Actions = append(plan.Actions, PlanAction{
			Priority:    1,
			Command:     "brew install tmux",
			Description: "Install tmux using Homebrew (macOS)",
		})
		return encodeJSON(plan)
	}

	// Check for existing sessions
	sessions, _ := tmux.ListSessions()

	if len(sessions) == 0 {
		plan.Recommendation = "Create your first coding session"
		plan.Actions = append(plan.Actions, PlanAction{
			Priority:    1,
			Command:     "ntm spawn myproject --cc=2",
			Description: "Create session with 2 Claude Code agents",
			Args:        []string{"spawn", "myproject", "--cc=2"},
		})
		plan.Actions = append(plan.Actions, PlanAction{
			Priority:    2,
			Command:     "ntm tutorial",
			Description: "Learn NTM with an interactive tutorial",
			Args:        []string{"tutorial"},
		})
	} else {
		plan.Recommendation = "Attach to an existing session or create a new one"

		// Find unattached sessions
		for _, sess := range sessions {
			if !sess.Attached {
				plan.Actions = append(plan.Actions, PlanAction{
					Priority:    1,
					Command:     fmt.Sprintf("ntm attach %s", sess.Name),
					Description: fmt.Sprintf("Attach to session '%s' (%d windows)", sess.Name, sess.Windows),
					Args:        []string{"attach", sess.Name},
				})
			}
		}

		plan.Actions = append(plan.Actions, PlanAction{
			Priority:    2,
			Command:     "ntm palette",
			Description: "Open command palette for quick actions",
			Args:        []string{"palette"},
		})
		plan.Actions = append(plan.Actions, PlanAction{
			Priority:    3,
			Command:     "ntm dashboard",
			Description: "Open visual session dashboard",
			Args:        []string{"dashboard"},
		})
	}

	return encodeJSON(plan)
}

func detectAgentType(title string) string {
	// Try to detect from pane title
	switch {
	case contains(title, "claude"):
		return "claude"
	case contains(title, "codex"):
		return "codex"
	case contains(title, "gemini"):
		return "gemini"
	case contains(title, "cursor"):
		return "cursor"
	case contains(title, "windsurf"):
		return "windsurf"
	case contains(title, "aider"):
		return "aider"
	default:
		return "unknown"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func encodeJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
