// Package robot provides machine-readable output for AI agents and automation.
// synthesis.go implements file conflict detection across multiple agents.
package robot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/agentmail"
)

// ConflictReason describes why a file conflict was detected.
type ConflictReason string

const (
	// ReasonConcurrentActivity indicates multiple panes had activity while file was modified.
	ReasonConcurrentActivity ConflictReason = "concurrent_activity"
	// ReasonReservationViolation indicates a file was modified without holding a reservation.
	ReasonReservationViolation ConflictReason = "reservation_violation"
	// ReasonOverlappingReservations indicates multiple agents have reservations for same file.
	ReasonOverlappingReservations ConflictReason = "overlapping_reservations"
	// ReasonUnclaimedModification indicates a modified file with no known modifier.
	ReasonUnclaimedModification ConflictReason = "unclaimed_modification"
)

// DetectedConflict represents a detected or potential file conflict from synthesis analysis.
// This extends the simpler FileConflict in tui_parity.go with more detailed conflict analysis.
type DetectedConflict struct {
	// Path is the file path relative to the repository root.
	Path string `json:"path"`

	// LikelyModifiers are pane IDs that may have modified this file.
	LikelyModifiers []string `json:"likely_modifiers"`

	// GitStatus is the git status code (M=modified, A=added, D=deleted, ??=untracked).
	GitStatus string `json:"git_status"`

	// Confidence is a score from 0.0-1.0 indicating conflict likelihood.
	// 0.9+ = high, 0.7-0.9 = medium, 0.5-0.7 = low
	Confidence float64 `json:"confidence"`

	// Reason explains why this conflict was detected.
	Reason ConflictReason `json:"reason"`

	// ReservationHolders are agents with active reservations for this file.
	ReservationHolders []string `json:"reservation_holders,omitempty"`

	// ModifiedAt is when the file was last modified (from filesystem).
	ModifiedAt time.Time `json:"modified_at,omitempty"`

	// Details provides additional context for the conflict.
	Details string `json:"details,omitempty"`
}

// ConflictConfidence categorizes confidence levels.
type ConflictConfidence string

const (
	// ConfidenceHigh indicates strong evidence of conflict (0.9+).
	ConfidenceHigh ConflictConfidence = "high"
	// ConfidenceMedium indicates moderate evidence (0.7-0.9).
	ConfidenceMedium ConflictConfidence = "medium"
	// ConfidenceLow indicates weak evidence (0.5-0.7).
	ConfidenceLow ConflictConfidence = "low"
	// ConfidenceNone indicates no significant conflict evidence (<0.5).
	ConfidenceNone ConflictConfidence = "none"
)

// ConfidenceLevel returns the categorical confidence level.
func (dc *DetectedConflict) ConfidenceLevel() ConflictConfidence {
	switch {
	case dc.Confidence >= 0.9:
		return ConfidenceHigh
	case dc.Confidence >= 0.7:
		return ConfidenceMedium
	case dc.Confidence >= 0.5:
		return ConfidenceLow
	default:
		return ConfidenceNone
	}
}

// ActivityWindow represents a time window of agent activity.
type ActivityWindow struct {
	PaneID    string    `json:"pane_id"`
	AgentType string    `json:"agent_type"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	HasOutput bool      `json:"has_output"` // Whether output was detected during window
}

// Overlaps returns true if this window overlaps with another.
func (aw *ActivityWindow) Overlaps(other *ActivityWindow) bool {
	return aw.Start.Before(other.End) && other.Start.Before(aw.End)
}

// Contains returns true if the given time falls within this window.
func (aw *ActivityWindow) Contains(t time.Time) bool {
	return !t.Before(aw.Start) && !t.After(aw.End)
}

// GitFileStatus represents a file's status from git.
type GitFileStatus struct {
	Path       string    `json:"path"`
	Status     string    `json:"status"` // M, A, D, ??, etc.
	Staged     bool      `json:"staged"`
	ModifiedAt time.Time `json:"modified_at,omitempty"`
}

// ConflictDetector detects potential file conflicts across agents.
type ConflictDetector struct {
	repoPath        string
	activityWindows map[string][]ActivityWindow // paneID -> windows
	amClient        *agentmail.Client
	projectKey      string

	mu sync.RWMutex
}

// ConflictDetectorConfig holds configuration for conflict detection.
type ConflictDetectorConfig struct {
	RepoPath   string
	ProjectKey string
	AMClient   *agentmail.Client
}

// NewConflictDetector creates a new conflict detector.
func NewConflictDetector(cfg *ConflictDetectorConfig) *ConflictDetector {
	if cfg == nil {
		cfg = &ConflictDetectorConfig{}
	}

	repoPath := cfg.RepoPath
	if repoPath == "" {
		repoPath, _ = os.Getwd()
	}

	return &ConflictDetector{
		repoPath:        repoPath,
		activityWindows: make(map[string][]ActivityWindow),
		amClient:        cfg.AMClient,
		projectKey:      cfg.ProjectKey,
	}
}

// RecordActivity records an activity window for a pane.
func (cd *ConflictDetector) RecordActivity(paneID, agentType string, start, end time.Time, hasOutput bool) {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	window := ActivityWindow{
		PaneID:    paneID,
		AgentType: agentType,
		Start:     start,
		End:       end,
		HasOutput: hasOutput,
	}

	cd.activityWindows[paneID] = append(cd.activityWindows[paneID], window)

	// Keep only windows from the last hour to prevent unbounded growth
	cutoff := time.Now().Add(-1 * time.Hour)
	cd.pruneWindowsLocked(cutoff)
}

// pruneWindowsLocked removes activity windows older than cutoff.
// Must be called with mu held.
func (cd *ConflictDetector) pruneWindowsLocked(cutoff time.Time) {
	for paneID, windows := range cd.activityWindows {
		var kept []ActivityWindow
		for _, w := range windows {
			if w.End.After(cutoff) {
				kept = append(kept, w)
			}
		}
		if len(kept) > 0 {
			cd.activityWindows[paneID] = kept
		} else {
			delete(cd.activityWindows, paneID)
		}
	}
}

// GetGitStatus returns the current git status of modified files.
func (cd *ConflictDetector) GetGitStatus() ([]GitFileStatus, error) {
	cmd := exec.Command("git", "-C", cd.repoPath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseGitStatusPorcelain(string(output), cd.repoPath)
}

// parseGitStatusPorcelain parses `git status --porcelain` output.
func parseGitStatusPorcelain(output, repoPath string) ([]GitFileStatus, error) {
	var results []GitFileStatus

	// Don't TrimSpace the whole output - it would remove leading spaces from status codes
	// like " M file.go" where space means "not staged"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r") // Handle CRLF
		if len(line) < 3 {
			continue
		}

		// Format: XY path
		// X = index status, Y = work tree status
		xy := line[:2]
		path := strings.TrimSpace(line[3:])

		// Handle renamed files (path contains " -> ")
		if idx := strings.Index(path, " -> "); idx >= 0 {
			path = path[idx+4:]
		}

		status := GitFileStatus{
			Path:   path,
			Status: strings.TrimSpace(xy),
			Staged: xy[0] != ' ' && xy[0] != '?',
		}

		// Get file modification time
		fullPath := filepath.Join(repoPath, path)
		if info, err := os.Stat(fullPath); err == nil {
			status.ModifiedAt = info.ModTime()
		}

		results = append(results, status)
	}

	return results, nil
}

// DetectConflicts analyzes git status and activity windows to detect conflicts.
func (cd *ConflictDetector) DetectConflicts(ctx context.Context) ([]DetectedConflict, error) {
	// Get current git status
	gitStatus, err := cd.GetGitStatus()
	if err != nil {
		return nil, err
	}

	if len(gitStatus) == 0 {
		return nil, nil // No modified files
	}

	// Get file reservations from Agent Mail if available
	var reservations []agentmail.FileReservation
	if cd.amClient != nil && cd.projectKey != "" {
		// List all reservations (not filtered by agent)
		reservations, _ = cd.amClient.ListReservations(ctx, cd.projectKey, "", true)
	}

	cd.mu.RLock()
	defer cd.mu.RUnlock()

	var conflicts []DetectedConflict

	for _, file := range gitStatus {
		conflict := cd.analyzeFileConflict(file, reservations)
		if conflict != nil && conflict.Confidence >= 0.5 {
			conflicts = append(conflicts, *conflict)
		}
	}

	return conflicts, nil
}

// analyzeFileConflict analyzes a single file for conflicts.
func (cd *ConflictDetector) analyzeFileConflict(file GitFileStatus, reservations []agentmail.FileReservation) *DetectedConflict {
	conflict := &DetectedConflict{
		Path:       file.Path,
		GitStatus:  file.Status,
		ModifiedAt: file.ModifiedAt,
		Confidence: 0.0,
	}

	// Find reservation holders for this file
	holders := cd.findReservationHolders(file.Path, reservations)
	conflict.ReservationHolders = holders

	// Find panes with activity during file modification window
	modifiers := cd.findLikelyModifiers(file)
	conflict.LikelyModifiers = modifiers

	// Score the conflict
	cd.scoreConflict(conflict, len(modifiers), len(holders))

	return conflict
}

// findReservationHolders returns agents with reservations matching the file path.
func (cd *ConflictDetector) findReservationHolders(filePath string, reservations []agentmail.FileReservation) []string {
	var holders []string
	seen := make(map[string]bool)

	for _, r := range reservations {
		// Skip released reservations
		if r.ReleasedTS != nil {
			continue
		}
		// Skip expired reservations
		if r.ExpiresTS.Before(time.Now()) {
			continue
		}

		if matchesPattern(filePath, r.PathPattern) && !seen[r.AgentName] {
			holders = append(holders, r.AgentName)
			seen[r.AgentName] = true
		}
	}

	return holders
}

// matchesPattern checks if a file path matches a glob pattern.
// Supports:
// - Exact match: "src/main.go"
// - Prefix match: "src/" matches "src/main.go"
// - Single * wildcard: "src/*.go" matches "src/main.go"
// - Double ** wildcard: "src/**" matches any path under src/
// - Combined: "src/**/test.go" matches "src/foo/bar/test.go"
func matchesPattern(filePath, pattern string) bool {
	// Exact match
	if filePath == pattern {
		return true
	}

	// Handle ** patterns (match any number of path segments)
	if strings.Contains(pattern, "**") {
		parts := strings.SplitN(pattern, "**", 2)
		prefix := parts[0]
		suffix := ""
		if len(parts) > 1 {
			suffix = strings.TrimPrefix(parts[1], "/")
		}

		// Path must start with prefix
		if !strings.HasPrefix(filePath, prefix) {
			return false
		}

		// If no suffix, just prefix match is enough
		if suffix == "" {
			return true
		}

		// Path must end with suffix (after stripping prefix)
		remaining := strings.TrimPrefix(filePath, prefix)
		return strings.HasSuffix(remaining, suffix)
	}

	// Handle single * patterns (match single path segment)
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")

		// Must start with first part and end with last part
		if !strings.HasPrefix(filePath, parts[0]) {
			return false
		}
		if !strings.HasSuffix(filePath, parts[len(parts)-1]) {
			return false
		}

		// For multiple wildcards, check that all parts appear in order
		remaining := filePath
		for _, part := range parts {
			if part == "" {
				continue
			}
			idx := strings.Index(remaining, part)
			if idx == -1 {
				return false
			}
			remaining = remaining[idx+len(part):]
		}
		return true
	}

	// Prefix match (pattern is a directory)
	return strings.HasPrefix(filePath, pattern+"/")
}

// findLikelyModifiers returns pane IDs with activity around the file modification time.
func (cd *ConflictDetector) findLikelyModifiers(file GitFileStatus) []string {
	if file.ModifiedAt.IsZero() {
		return nil
	}

	var modifiers []string
	seen := make(map[string]bool)

	// Look for activity windows that contain the file modification time
	// Use a tolerance window of 60 seconds before and after
	tolerance := 60 * time.Second
	checkStart := file.ModifiedAt.Add(-tolerance)
	checkEnd := file.ModifiedAt.Add(tolerance)

	for paneID, windows := range cd.activityWindows {
		for _, w := range windows {
			// Check if window overlaps with modification time window
			if w.Start.Before(checkEnd) && w.End.After(checkStart) {
				if !seen[paneID] {
					modifiers = append(modifiers, paneID)
					seen[paneID] = true
				}
				break
			}
		}
	}

	return modifiers
}

// scoreConflict calculates the conflict confidence score.
func (cd *ConflictDetector) scoreConflict(conflict *DetectedConflict, modifierCount, holderCount int) {
	// Base confidence based on situation
	switch {
	case modifierCount > 1:
		// Multiple modifiers - high confidence of conflict
		conflict.Confidence = 0.9
		conflict.Reason = ReasonConcurrentActivity
		conflict.Details = "Multiple agents had activity when this file was modified"

	case modifierCount == 1 && holderCount > 0:
		// Single modifier with reservation holders
		if !containsAny(conflict.LikelyModifiers, conflict.ReservationHolders) {
			// Modifier doesn't hold the reservation
			conflict.Confidence = 0.85
			conflict.Reason = ReasonReservationViolation
			conflict.Details = "File modified by agent without active reservation"
		} else {
			// Modifier holds reservation - likely OK
			conflict.Confidence = 0.3
			conflict.Reason = ReasonConcurrentActivity
			conflict.Details = "File modified by reservation holder"
		}

	case modifierCount == 0 && holderCount > 1:
		// No detected modifier but multiple reservation holders
		conflict.Confidence = 0.75
		conflict.Reason = ReasonOverlappingReservations
		conflict.Details = "Multiple agents have reservations for this file"

	case modifierCount == 0 && holderCount == 0:
		// Unknown modifier, no reservations
		conflict.Confidence = 0.6
		conflict.Reason = ReasonUnclaimedModification
		conflict.Details = "File modified with no tracked activity or reservations"

	case modifierCount == 1 && holderCount == 0:
		// Single modifier, no reservations (normal case)
		conflict.Confidence = 0.4
		conflict.Reason = ReasonConcurrentActivity
		conflict.Details = "File modified by single agent without reservation"

	default:
		conflict.Confidence = 0.5
		conflict.Reason = ReasonUnclaimedModification
	}
}

// containsAny returns true if any element of a is in b.
func containsAny(a, b []string) bool {
	bSet := make(map[string]bool, len(b))
	for _, s := range b {
		bSet[s] = true
	}
	for _, s := range a {
		if bSet[s] {
			return true
		}
	}
	return false
}

// GetActivityWindows returns all tracked activity windows.
func (cd *ConflictDetector) GetActivityWindows() map[string][]ActivityWindow {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	// Return a copy
	result := make(map[string][]ActivityWindow, len(cd.activityWindows))
	for paneID, windows := range cd.activityWindows {
		windowsCopy := make([]ActivityWindow, len(windows))
		copy(windowsCopy, windows)
		result[paneID] = windowsCopy
	}
	return result
}

// ClearActivityWindows removes all tracked activity windows.
func (cd *ConflictDetector) ClearActivityWindows() {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	cd.activityWindows = make(map[string][]ActivityWindow)
}

// ConflictSummary provides a summary of detected conflicts.
type ConflictSummary struct {
	TotalConflicts int                `json:"total_conflicts"`
	HighConfidence int                `json:"high_confidence"` // 0.9+
	MedConfidence  int                `json:"med_confidence"`  // 0.7-0.9
	LowConfidence  int                `json:"low_confidence"`  // 0.5-0.7
	ByReason       map[string]int     `json:"by_reason"`
	Conflicts      []DetectedConflict `json:"conflicts"`
	Timestamp      string             `json:"timestamp"`
}

// SummarizeConflicts generates a summary from a list of conflicts.
func SummarizeConflicts(conflicts []DetectedConflict) *ConflictSummary {
	summary := &ConflictSummary{
		TotalConflicts: len(conflicts),
		ByReason:       make(map[string]int),
		Conflicts:      conflicts,
		Timestamp:      FormatTimestamp(time.Now()),
	}

	for _, c := range conflicts {
		switch c.ConfidenceLevel() {
		case ConfidenceHigh:
			summary.HighConfidence++
		case ConfidenceMedium:
			summary.MedConfidence++
		case ConfidenceLow:
			summary.LowConfidence++
		}
		summary.ByReason[string(c.Reason)]++
	}

	return summary
}

// ConflictDetectionResponse is the robot command response for conflict detection.
type ConflictDetectionResponse struct {
	RobotResponse
	Summary *ConflictSummary `json:"summary,omitempty"`
}

// NewConflictDetectionResponse creates a new conflict detection response.
func NewConflictDetectionResponse(conflicts []DetectedConflict) *ConflictDetectionResponse {
	resp := &ConflictDetectionResponse{
		RobotResponse: NewRobotResponse(true),
	}
	if len(conflicts) > 0 {
		resp.Summary = SummarizeConflicts(conflicts)
	}
	return resp
}

// ============================================================================
// Output Capture & Tagging
// ============================================================================

// CapturedOutput represents a captured output from an agent pane with extracted structures.
type CapturedOutput struct {
	PaneID    string    `json:"pane_id"`
	AgentType string    `json:"agent_type,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	RawLength int       `json:"raw_length"` // Length of raw content (for metrics)
	Prompt    string    `json:"prompt,omitempty"`

	// Extracted structures
	CodeBlocks  []CodeBlock       `json:"code_blocks,omitempty"`
	JSONOutputs []JSONOutput      `json:"json_outputs,omitempty"`
	FilePaths   []FileMention     `json:"file_paths,omitempty"`
	Commands    []CommandMention  `json:"commands,omitempty"`
}

// CodeBlock represents an extracted code block from agent output.
type CodeBlock struct {
	Language  string `json:"language"`
	Content   string `json:"content"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
}

// JSONOutput represents a detected JSON object or array in output.
type JSONOutput struct {
	Raw       string `json:"raw"`        // Original JSON string
	IsArray   bool   `json:"is_array"`   // True if JSON array, false if object
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
}

// FileMention represents a file path mentioned in agent output.
type FileMention struct {
	Path      string `json:"path"`
	Action    string `json:"action"` // created, modified, deleted, read
	LineNum   int    `json:"line_num,omitempty"`
	Confidence float64 `json:"confidence"` // 0.0-1.0 how confident we are about the action
}

// FileAction constants for FileMention.
const (
	FileActionCreated  = "created"
	FileActionModified = "modified"
	FileActionDeleted  = "deleted"
	FileActionRead     = "read"
	FileActionUnknown  = "unknown"
)

// CommandMention represents a shell command detected in output.
type CommandMention struct {
	Command   string `json:"command"`
	LineNum   int    `json:"line_num"`
	ExitCode  *int   `json:"exit_code,omitempty"` // nil if not visible
}

// ExtractCodeBlocks extracts markdown code blocks from output.
// Handles ```lang ... ``` syntax with optional language tags.
func ExtractCodeBlocks(content string) []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(content, "\n")

	inBlock := false
	var currentBlock CodeBlock
	var contentLines []string

	for i, line := range lines {
		lineNum := i + 1 // 1-indexed

		if !inBlock {
			// Look for opening ```
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inBlock = true
				currentBlock = CodeBlock{
					LineStart: lineNum,
				}
				// Extract language from opening fence
				trimmed := strings.TrimSpace(line)
				if len(trimmed) > 3 {
					currentBlock.Language = strings.TrimSpace(trimmed[3:])
				}
				contentLines = nil
			}
		} else {
			// Look for closing ```
			if strings.TrimSpace(line) == "```" {
				inBlock = false
				currentBlock.LineEnd = lineNum
				currentBlock.Content = strings.Join(contentLines, "\n")
				blocks = append(blocks, currentBlock)
			} else {
				contentLines = append(contentLines, line)
			}
		}
	}

	return blocks
}

// ExtractJSONOutputs detects JSON objects and arrays in output.
// Only extracts complete, valid JSON.
func ExtractJSONOutputs(content string) []JSONOutput {
	var outputs []JSONOutput
	lines := strings.Split(content, "\n")

	// Track potential JSON start positions
	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Look for lines starting with { or [
		if len(trimmed) == 0 {
			continue
		}

		if trimmed[0] == '{' || trimmed[0] == '[' {
			// Try to find a complete JSON block starting here
			jsonStr, endLine := extractCompleteJSON(lines, i)
			if jsonStr != "" {
				outputs = append(outputs, JSONOutput{
					Raw:       jsonStr,
					IsArray:   trimmed[0] == '[',
					LineStart: lineNum,
					LineEnd:   endLine,
				})
			}
		}
	}

	return outputs
}

// extractCompleteJSON tries to extract a complete JSON object/array starting at line index.
// Returns the JSON string and end line number (1-indexed), or empty string if invalid.
func extractCompleteJSON(lines []string, startIdx int) (string, int) {
	// Build potential JSON string line by line until we get valid JSON
	var builder strings.Builder
	depth := 0
	inString := false
	escaped := false

	for i := startIdx; i < len(lines) && i < startIdx+100; i++ { // Limit to 100 lines
		if builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(lines[i])

		// Track bracket depth to know when JSON is complete
		for _, ch := range lines[i] {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' && inString {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = !inString
				continue
			}
			if inString {
				continue
			}
			switch ch {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		}

		// If depth returns to 0, we have a complete structure
		if depth == 0 && builder.Len() > 0 {
			jsonStr := strings.TrimSpace(builder.String())
			// Validate it's actually valid JSON
			if isValidJSON(jsonStr) {
				return jsonStr, i + 1 // 1-indexed
			}
			return "", 0
		}

		// If depth goes negative, invalid structure
		if depth < 0 {
			return "", 0
		}
	}

	return "", 0
}

// isValidJSON checks if a string is valid JSON.
func isValidJSON(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return false
	}
	// Quick validation: try to decode
	var js interface{}
	decoder := json.NewDecoder(strings.NewReader(s))
	decoder.UseNumber()
	return decoder.Decode(&js) == nil
}

// ExtractFileMentions extracts file path mentions from output with action context.
// Looks for common patterns like src/..., ./..., internal/..., and action keywords.
func ExtractFileMentions(content string) []FileMention {
	var mentions []FileMention
	seen := make(map[string]bool)
	lines := strings.Split(content, "\n")

	// Patterns that look like file paths
	// We're looking for paths, not URLs
	for i, line := range lines {
		lineNum := i + 1

		// Extract paths and determine action from context
		paths := extractPathsFromLine(line)
		for _, path := range paths {
			if seen[path] {
				continue
			}
			seen[path] = true

			action, confidence := inferFileAction(line, path)
			mentions = append(mentions, FileMention{
				Path:       path,
				Action:     action,
				LineNum:    lineNum,
				Confidence: confidence,
			})
		}
	}

	return mentions
}

// extractPathsFromLine extracts file paths from a single line.
func extractPathsFromLine(line string) []string {
	var paths []string

	// Common file path patterns
	patterns := []string{
		// Paths starting with common prefixes
		`(?:^|[\s'"(,])((src|internal|pkg|cmd|lib|test|tests|spec|app|api|web|frontend|backend|client|server|utils|util|common|shared|core|modules|components|services|models|views|controllers|middleware|config|configs|scripts|tools|build|dist|bin|docs|examples|assets|resources|public|private|vendor|third_party|node_modules)\/[\w\-./]+\.\w+)`,
		// Relative paths
		`(?:^|[\s'"(,])(\.\/[\w\-./]+\.\w+)`,
		// Paths with file extensions
		`(?:^|[\s'"(,])([\w\-./]+\.(?:go|py|js|ts|jsx|tsx|rs|rb|java|c|cpp|h|hpp|cs|php|swift|kt|scala|vue|svelte|md|txt|json|yaml|yml|toml|xml|html|css|scss|sass|less))(?:[\s'")\]:,]|$)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				path := strings.Trim(match[1], `'"`)
				// Skip if it looks like a URL
				if strings.Contains(path, "://") || strings.HasPrefix(path, "http") {
					continue
				}
				// Skip if too short or doesn't look like a path
				if len(path) < 3 || !strings.Contains(path, "/") && !strings.Contains(path, ".") {
					continue
				}
				paths = append(paths, path)
			}
		}
	}

	return paths
}

// inferFileAction determines the likely action on a file from context.
func inferFileAction(line, path string) (string, float64) {
	lineLower := strings.ToLower(line)

	// High confidence patterns
	if strings.Contains(lineLower, "created "+path) ||
	   strings.Contains(lineLower, "creating "+path) ||
	   strings.Contains(lineLower, "create file") ||
	   strings.Contains(lineLower, "new file") ||
	   strings.Contains(lineLower, "write to "+strings.ToLower(path)) {
		return FileActionCreated, 0.9
	}

	if strings.Contains(lineLower, "modified "+path) ||
	   strings.Contains(lineLower, "modifying "+path) ||
	   strings.Contains(lineLower, "updated "+path) ||
	   strings.Contains(lineLower, "updating "+path) ||
	   strings.Contains(lineLower, "edited "+path) ||
	   strings.Contains(lineLower, "editing "+path) ||
	   strings.Contains(lineLower, "changed "+path) {
		return FileActionModified, 0.9
	}

	if strings.Contains(lineLower, "deleted "+path) ||
	   strings.Contains(lineLower, "deleting "+path) ||
	   strings.Contains(lineLower, "removed "+path) ||
	   strings.Contains(lineLower, "removing "+path) {
		return FileActionDeleted, 0.9
	}

	if strings.Contains(lineLower, "reading "+path) ||
	   strings.Contains(lineLower, "read "+path) ||
	   strings.Contains(lineLower, "opened "+path) ||
	   strings.Contains(lineLower, "loading "+path) {
		return FileActionRead, 0.9
	}

	// Medium confidence: action keywords near path
	if strings.Contains(lineLower, "creat") || strings.Contains(lineLower, "new") {
		return FileActionCreated, 0.6
	}
	if strings.Contains(lineLower, "modif") || strings.Contains(lineLower, "edit") ||
	   strings.Contains(lineLower, "updat") || strings.Contains(lineLower, "chang") {
		return FileActionModified, 0.6
	}
	if strings.Contains(lineLower, "delet") || strings.Contains(lineLower, "remov") {
		return FileActionDeleted, 0.6
	}
	if strings.Contains(lineLower, "read") || strings.Contains(lineLower, "open") ||
	   strings.Contains(lineLower, "load") || strings.Contains(lineLower, "view") {
		return FileActionRead, 0.6
	}

	return FileActionUnknown, 0.3
}

// ExtractCommands extracts shell commands from output.
// Looks for lines starting with $, %, >, or common shell patterns.
func ExtractCommands(content string) []CommandMention {
	var commands []CommandMention
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		if len(trimmed) == 0 {
			continue
		}

		// Check for command prompt patterns
		var cmd string

		// $ command (most common)
		if strings.HasPrefix(trimmed, "$ ") {
			cmd = strings.TrimPrefix(trimmed, "$ ")
		} else if strings.HasPrefix(trimmed, "% ") {
			cmd = strings.TrimPrefix(trimmed, "% ")
		} else if strings.HasPrefix(trimmed, "> ") && !strings.HasPrefix(trimmed, ">>") {
			// > but not >> (append redirect)
			cmd = strings.TrimPrefix(trimmed, "> ")
		} else if strings.HasPrefix(trimmed, ">>> ") {
			// Python REPL - skip
			continue
		}

		if cmd != "" {
			mention := CommandMention{
				Command: cmd,
				LineNum: lineNum,
			}

			// Try to find exit code in subsequent lines
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				// Look for "exit code: N" or similar patterns
				if exitCode := parseExitCode(nextLine); exitCode != nil {
					mention.ExitCode = exitCode
				}
			}

			commands = append(commands, mention)
		}
	}

	return commands
}

// parseExitCode tries to parse an exit code from a line.
func parseExitCode(line string) *int {
	lineLower := strings.ToLower(line)

	// Common patterns for exit codes
	patterns := []string{
		`exit(?:\s+code)?[:\s]+(\d+)`,
		`returned?\s+(\d+)`,
		`status[:\s]+(\d+)`,
		`\[(\d+)\]$`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(lineLower); len(match) > 1 {
			var code int
			if _, err := fmt.Sscanf(match[1], "%d", &code); err == nil {
				return &code
			}
		}
	}

	return nil
}

// ============================================================================
// Output Capture Storage
// ============================================================================

// OutputCaptureConfig holds configuration for output capture.
type OutputCaptureConfig struct {
	MaxCapturesPerPane int           // Maximum captures per pane (ring buffer size)
	MaxRetention       time.Duration // Maximum age of captures to keep
}

// DefaultOutputCaptureConfig returns default configuration.
func DefaultOutputCaptureConfig() *OutputCaptureConfig {
	return &OutputCaptureConfig{
		MaxCapturesPerPane: 100,
		MaxRetention:       1 * time.Hour,
	}
}

// OutputCapture stores captured outputs in a ring buffer per pane.
type OutputCapture struct {
	config   *OutputCaptureConfig
	captures map[string][]CapturedOutput // paneID -> ring buffer of captures
	mu       sync.RWMutex
}

// NewOutputCapture creates a new output capture store.
func NewOutputCapture(cfg *OutputCaptureConfig) *OutputCapture {
	if cfg == nil {
		cfg = DefaultOutputCaptureConfig()
	}
	return &OutputCapture{
		config:   cfg,
		captures: make(map[string][]CapturedOutput),
	}
}

// CaptureAndExtract captures raw output and extracts all structured data.
func (oc *OutputCapture) CaptureAndExtract(paneID, agentType, rawContent, prompt string) *CapturedOutput {
	capture := &CapturedOutput{
		PaneID:    paneID,
		AgentType: agentType,
		Timestamp: time.Now(),
		RawLength: len(rawContent),
		Prompt:    prompt,
	}

	// Extract all structures
	capture.CodeBlocks = ExtractCodeBlocks(rawContent)
	capture.JSONOutputs = ExtractJSONOutputs(rawContent)
	capture.FilePaths = ExtractFileMentions(rawContent)
	capture.Commands = ExtractCommands(rawContent)

	// Store in ring buffer
	oc.store(paneID, *capture)

	return capture
}

// store adds a capture to the ring buffer for a pane.
func (oc *OutputCapture) store(paneID string, capture CapturedOutput) {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	// Prune old captures first
	oc.pruneOldCapturesLocked()

	// Add to ring buffer
	captures := oc.captures[paneID]
	captures = append(captures, capture)

	// Enforce max size
	if len(captures) > oc.config.MaxCapturesPerPane {
		captures = captures[len(captures)-oc.config.MaxCapturesPerPane:]
	}

	oc.captures[paneID] = captures
}

// pruneOldCapturesLocked removes captures older than MaxRetention.
// Must be called with mu held.
func (oc *OutputCapture) pruneOldCapturesLocked() {
	cutoff := time.Now().Add(-oc.config.MaxRetention)

	for paneID, captures := range oc.captures {
		var kept []CapturedOutput
		for _, c := range captures {
			if c.Timestamp.After(cutoff) {
				kept = append(kept, c)
			}
		}
		if len(kept) > 0 {
			oc.captures[paneID] = kept
		} else {
			delete(oc.captures, paneID)
		}
	}
}

// GetCaptures returns captures for a pane, optionally limited and filtered.
func (oc *OutputCapture) GetCaptures(paneID string, limit int, since *time.Time) []CapturedOutput {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	captures := oc.captures[paneID]
	if captures == nil {
		return nil
	}

	// Filter by time if requested
	var filtered []CapturedOutput
	for _, c := range captures {
		if since != nil && !c.Timestamp.After(*since) {
			continue
		}
		filtered = append(filtered, c)
	}

	// Apply limit (from the end - most recent)
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}

	return filtered
}

// GetAllCaptures returns all captures across all panes.
func (oc *OutputCapture) GetAllCaptures() map[string][]CapturedOutput {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	result := make(map[string][]CapturedOutput, len(oc.captures))
	for paneID, captures := range oc.captures {
		capturesCopy := make([]CapturedOutput, len(captures))
		copy(capturesCopy, captures)
		result[paneID] = capturesCopy
	}
	return result
}

// GetLatestCapture returns the most recent capture for a pane.
func (oc *OutputCapture) GetLatestCapture(paneID string) *CapturedOutput {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	captures := oc.captures[paneID]
	if len(captures) == 0 {
		return nil
	}

	latest := captures[len(captures)-1]
	return &latest
}

// ClearCaptures removes all captures for a pane.
func (oc *OutputCapture) ClearCaptures(paneID string) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	delete(oc.captures, paneID)
}

// ClearAllCaptures removes all captures.
func (oc *OutputCapture) ClearAllCaptures() {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	oc.captures = make(map[string][]CapturedOutput)
}

// Stats returns statistics about the capture store.
func (oc *OutputCapture) Stats() OutputCaptureStats {
	oc.mu.RLock()
	defer oc.mu.RUnlock()

	stats := OutputCaptureStats{
		PaneCount: len(oc.captures),
		Timestamp: time.Now(),
	}

	for paneID, captures := range oc.captures {
		stats.TotalCaptures += len(captures)
		if len(captures) > 0 {
			stats.OldestCapture = captures[0].Timestamp
			if stats.NewestCapture.IsZero() || captures[len(captures)-1].Timestamp.After(stats.NewestCapture) {
				stats.NewestCapture = captures[len(captures)-1].Timestamp
			}
		}
		stats.CaptureCounts = append(stats.CaptureCounts, PaneCaptureCount{
			PaneID: paneID,
			Count:  len(captures),
		})
	}

	return stats
}

// OutputCaptureStats provides statistics about the capture store.
type OutputCaptureStats struct {
	PaneCount      int               `json:"pane_count"`
	TotalCaptures  int               `json:"total_captures"`
	OldestCapture  time.Time         `json:"oldest_capture,omitempty"`
	NewestCapture  time.Time         `json:"newest_capture,omitempty"`
	CaptureCounts  []PaneCaptureCount `json:"capture_counts,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
}

// PaneCaptureCount shows capture count for a single pane.
type PaneCaptureCount struct {
	PaneID string `json:"pane_id"`
	Count  int    `json:"count"`
}

// OutputCaptureResponse is the robot command response for output capture info.
type OutputCaptureResponse struct {
	RobotResponse
	Stats   *OutputCaptureStats `json:"stats,omitempty"`
	Panes   []string            `json:"panes,omitempty"`
}

// NewOutputCaptureResponse creates a response with capture statistics.
func NewOutputCaptureResponse(stats OutputCaptureStats) *OutputCaptureResponse {
	return &OutputCaptureResponse{
		RobotResponse: NewRobotResponse(true),
		Stats:         &stats,
	}
}
