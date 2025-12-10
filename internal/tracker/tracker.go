// Package tracker provides state change tracking for delta snapshot queries.
// It maintains a ring buffer of state changes with configurable size and age limits.
package tracker

import (
	"bytes"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ChangeType represents the type of state change
type ChangeType string

const (
	ChangeAgentOutput    ChangeType = "agent_output"
	ChangeAgentState     ChangeType = "agent_state"
	ChangeBeadUpdate     ChangeType = "bead_update"
	ChangeMailReceived   ChangeType = "mail_received"
	ChangeAlert          ChangeType = "alert"
	ChangePaneCreated    ChangeType = "pane_created"
	ChangePaneRemoved    ChangeType = "pane_removed"
	ChangeSessionCreated ChangeType = "session_created"
	ChangeSessionRemoved ChangeType = "session_removed"
	ChangeFileChange     ChangeType = "file_change"
)

// StateChange represents a single state change event
type StateChange struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      ChangeType             `json:"type"`
	Session   string                 `json:"session,omitempty"`
	Pane      string                 `json:"pane,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// StateTracker maintains a ring buffer of state changes
type StateTracker struct {
	changes []StateChange
	maxAge  time.Duration
	maxSize int
	mu      sync.RWMutex
}

// DefaultMaxSize is the default maximum number of changes to retain
const DefaultMaxSize = 1000

// DefaultMaxAge is the default maximum age of changes to retain
const DefaultMaxAge = 5 * time.Minute

// New creates a new StateTracker with default settings
func New() *StateTracker {
	return NewWithConfig(DefaultMaxSize, DefaultMaxAge)
}

// NewWithConfig creates a new StateTracker with custom settings
func NewWithConfig(maxSize int, maxAge time.Duration) *StateTracker {
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}
	if maxAge <= 0 {
		maxAge = DefaultMaxAge
	}
	return &StateTracker{
		changes: make([]StateChange, 0, maxSize),
		maxAge:  maxAge,
		maxSize: maxSize,
	}
}

// Record adds a new state change to the tracker
func (t *StateTracker) Record(change StateChange) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Set timestamp if not provided
	if change.Timestamp.IsZero() {
		change.Timestamp = time.Now()
	}

	// Prune old entries first
	t.pruneOld()

	// If at capacity, remove oldest
	if len(t.changes) >= t.maxSize {
		t.changes = t.changes[1:]
	}

	t.changes = append(t.changes, change)
}

// Since returns all changes since the given timestamp
func (t *StateTracker) Since(ts time.Time) []StateChange {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]StateChange, 0)
	for _, change := range t.changes {
		if change.Timestamp.After(ts) {
			// Deep copy Details to prevent data races
			newChange := change
			if change.Details != nil {
				newChange.Details = make(map[string]interface{}, len(change.Details))
				for k, v := range change.Details {
					newChange.Details[k] = v
				}
			}
			result = append(result, newChange)
		}
	}
	return result
}

// All returns all tracked changes
func (t *StateTracker) All() []StateChange {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]StateChange, 0, len(t.changes))
	for _, change := range t.changes {
		// Deep copy Details to prevent data races
		newChange := change
		if change.Details != nil {
			newChange.Details = make(map[string]interface{}, len(change.Details))
			for k, v := range change.Details {
				newChange.Details[k] = v
			}
		}
		result = append(result, newChange)
	}
	return result
}

// Count returns the number of tracked changes
func (t *StateTracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.changes)
}

// Clear removes all tracked changes
func (t *StateTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.changes = make([]StateChange, 0, t.maxSize)
}

// pruneOld removes changes older than maxAge (must be called with lock held)
func (t *StateTracker) pruneOld() {
	if len(t.changes) == 0 {
		return
	}

	cutoff := time.Now().Add(-t.maxAge)
	keepFrom := 0
	for i, change := range t.changes {
		if change.Timestamp.After(cutoff) {
			keepFrom = i
			break
		}
		keepFrom = i + 1
	}

	if keepFrom > 0 && keepFrom <= len(t.changes) {
		t.changes = t.changes[keepFrom:]
	}
}

// Prune manually triggers pruning of old entries
func (t *StateTracker) Prune() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pruneOld()
}

// CoalescedChange represents multiple changes merged into one summary
type CoalescedChange struct {
	Type    ChangeType `json:"type"`
	Session string     `json:"session,omitempty"`
	Pane    string     `json:"pane,omitempty"`
	Count   int        `json:"count"`
	FirstAt time.Time  `json:"first_at"`
	LastAt  time.Time  `json:"last_at"`
}

// Coalesce merges consecutive changes of the same type for the same pane
// into summary entries. Useful for reducing output volume.
func (t *StateTracker) Coalesce() []CoalescedChange {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.changes) == 0 {
		return nil
	}

	result := make([]CoalescedChange, 0)
	var current *CoalescedChange

	for _, change := range t.changes {
		// Check if we can merge with current
		if current != nil &&
			current.Type == change.Type &&
			current.Session == change.Session &&
			current.Pane == change.Pane {
			// Merge
			current.Count++
			current.LastAt = change.Timestamp
		} else {
			// Start new group
			if current != nil {
				result = append(result, *current)
			}
			current = &CoalescedChange{
				Type:    change.Type,
				Session: change.Session,
				Pane:    change.Pane,
				Count:   1,
				FirstAt: change.Timestamp,
				LastAt:  change.Timestamp,
			}
		}
	}

	if current != nil {
		result = append(result, *current)
	}

	return result
}

// SinceByType returns changes since the given timestamp, filtered by type
func (t *StateTracker) SinceByType(ts time.Time, changeType ChangeType) []StateChange {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]StateChange, 0)
	for _, change := range t.changes {
		if change.Timestamp.After(ts) && change.Type == changeType {
			// Deep copy Details
			newChange := change
			if change.Details != nil {
				newChange.Details = make(map[string]interface{}, len(change.Details))
				for k, v := range change.Details {
					newChange.Details[k] = v
				}
			}
			result = append(result, newChange)
		}
	}
	return result
}

// SinceBySession returns changes since the given timestamp for a specific session
func (t *StateTracker) SinceBySession(ts time.Time, session string) []StateChange {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]StateChange, 0)
	for _, change := range t.changes {
		if change.Timestamp.After(ts) && change.Session == session {
			// Deep copy Details
			newChange := change
			if change.Details != nil {
				newChange.Details = make(map[string]interface{}, len(change.Details))
				for k, v := range change.Details {
					newChange.Details[k] = v
				}
			}
			result = append(result, newChange)
		}
	}
	return result
}

// Helper functions for common change types

// RecordAgentOutput records an agent output change
func (t *StateTracker) RecordAgentOutput(session, pane, output string) {
	t.Record(StateChange{
		Type:    ChangeAgentOutput,
		Session: session,
		Pane:    pane,
		Details: map[string]interface{}{
			"output_length": len(output),
		},
	})
}

// RecordAgentState records an agent state change (idle, active, error)
func (t *StateTracker) RecordAgentState(session, pane, state string) {
	t.Record(StateChange{
		Type:    ChangeAgentState,
		Session: session,
		Pane:    pane,
		Details: map[string]interface{}{
			"state": state,
		},
	})
}

// RecordAlert records an alert
func (t *StateTracker) RecordAlert(session, pane, alertType, message string) {
	t.Record(StateChange{
		Type:    ChangeAlert,
		Session: session,
		Pane:    pane,
		Details: map[string]interface{}{
			"alert_type": alertType,
			"message":    message,
		},
	})
}

// RecordPaneCreated records a new pane
func (t *StateTracker) RecordPaneCreated(session, pane, agentType string) {
	t.Record(StateChange{
		Type:    ChangePaneCreated,
		Session: session,
		Pane:    pane,
		Details: map[string]interface{}{
			"agent_type": agentType,
		},
	})
}

// RecordSessionCreated records a new session
func (t *StateTracker) RecordSessionCreated(session string) {
	t.Record(StateChange{
		Type:    ChangeSessionCreated,
		Session: session,
	})
}

// FileState captures minimal file metadata for change detection.
type FileState struct {
	ModTime time.Time `json:"mod_time"`
	Size    int64     `json:"size"`
}

// FileChangeType indicates what happened to a file.
type FileChangeType string

const (
	FileAdded    FileChangeType = "added"
	FileModified FileChangeType = "modified"
	FileDeleted  FileChangeType = "deleted"
)

// FileChange represents a single file change between two snapshots.
type FileChange struct {
	Path   string         `json:"path"`
	Type   FileChangeType `json:"type"`
	Before *FileState     `json:"before,omitempty"`
	After  *FileState     `json:"after,omitempty"`
}

// fileEntry is an internal representation used during snapshotting.
type fileEntry struct {
	path  string
	state FileState
}

// SnapshotOptions controls how directory snapshots are taken.
type SnapshotOptions struct {
	// IgnoreHidden skips files/dirs beginning with '.'
	IgnoreHidden bool
	// IgnorePaths are path prefixes (absolute) to skip.
	IgnorePaths []string
	// IgnoreGitIgnored attempts to skip files ignored by git (best-effort).
	IgnoreGitIgnored bool
}

// DefaultSnapshotOptions provides conservative defaults (skip .git).
func DefaultSnapshotOptions(root string) SnapshotOptions {
	return SnapshotOptions{
		IgnoreHidden:     false,
		IgnorePaths:      []string{root + "/.git"},
		IgnoreGitIgnored: true,
	}
}

// SnapshotDirectory walks a directory and captures file modtime/size.
// Returns a map keyed by absolute path.
func SnapshotDirectory(root string, opts SnapshotOptions) (map[string]FileState, error) {
	entries := make([]fileEntry, 0)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip configured ignore prefixes
		for _, p := range opts.IgnorePaths {
			if strings.HasPrefix(path, p) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Optionally skip hidden entries
		if opts.IgnoreHidden {
			_, name := filepath.Split(path)
			if strings.HasPrefix(name, ".") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		entries = append(entries, fileEntry{
			path: path,
			state: FileState{
				ModTime: info.ModTime(),
				Size:    info.Size(),
			},
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	ignored := map[string]bool{}
	if opts.IgnoreGitIgnored && len(entries) > 0 {
		if ig, err := gitCheckIgnored(root, entries); err == nil {
			ignored = ig
		}
	}

	snap := make(map[string]FileState, len(entries))
	for _, entry := range entries {
		if ignored[entry.path] {
			continue
		}
		snap[entry.path] = entry.state
	}

	return snap, nil
}

// gitCheckIgnored returns a set of paths that git considers ignored.
// Best-effort: failures return an empty map.
func gitCheckIgnored(root string, entries []fileEntry) (map[string]bool, error) {
	result := make(map[string]bool)

	var buf bytes.Buffer
	for _, e := range entries {
		buf.WriteString(e.path)
		buf.WriteByte('\n')
	}

	cmd := exec.Command("git", "-C", root, "check-ignore", "--stdin")
	cmd.Stdin = &buf
	out, err := cmd.Output()
	if err != nil {
		return result, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// git may echo only the path
		result[line] = true
	}
	return result, nil
}

// DetectFileChanges compares two snapshots and returns the delta.
func DetectFileChanges(before, after map[string]FileState) []FileChange {
	changes := make([]FileChange, 0)

	// Detect additions and modifications
	for path, afterState := range after {
		beforeState, existed := before[path]
		if !existed {
			changes = append(changes, FileChange{
				Path:  path,
				Type:  FileAdded,
				After: &afterState,
			})
			continue
		}

		if !afterState.ModTime.Equal(beforeState.ModTime) || afterState.Size != beforeState.Size {
			// Modified
			b := beforeState
			a := afterState
			changes = append(changes, FileChange{
				Path:   path,
				Type:   FileModified,
				Before: &b,
				After:  &a,
			})
		}
	}

	// Detect deletions
	for path, beforeState := range before {
		if _, ok := after[path]; !ok {
			b := beforeState
			changes = append(changes, FileChange{
				Path:   path,
				Type:   FileDeleted,
				Before: &b,
			})
		}
	}

	return changes
}

// RecordedFileChange captures file changes with attribution metadata.
type RecordedFileChange struct {
	Timestamp time.Time  `json:"timestamp"`
	Session   string     `json:"session"`
	Agents    []string   `json:"agents,omitempty"`
	Change    FileChange `json:"change"`
}

// FileChangeStore keeps a bounded buffer of recent file changes.
type FileChangeStore struct {
	mu      sync.Mutex
	limit   int
	entries []RecordedFileChange
}

// NewFileChangeStore creates a store with the provided capacity.
func NewFileChangeStore(limit int) *FileChangeStore {
	if limit <= 0 {
		limit = 500
	}
	return &FileChangeStore{
		limit:   limit,
		entries: make([]RecordedFileChange, 0, limit),
	}
}

// Add records a change and prunes oldest entries when over capacity.
func (s *FileChangeStore) Add(entry RecordedFileChange) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	if len(s.entries) >= s.limit {
		// drop oldest
		s.entries = s.entries[1:]
	}
	s.entries = append(s.entries, entry)
}

// Since returns changes after the provided timestamp.
func (s *FileChangeStore) Since(ts time.Time) []RecordedFileChange {
	s.mu.Lock()
	defer s.mu.Unlock()

	results := make([]RecordedFileChange, 0)
	for _, e := range s.entries {
		if e.Timestamp.After(ts) {
			results = append(results, e)
		}
	}
	return results
}

// All returns a copy of all recorded changes.
func (s *FileChangeStore) All() []RecordedFileChange {
	s.mu.Lock()
	defer s.mu.Unlock()

	results := make([]RecordedFileChange, len(s.entries))
	copy(results, s.entries)
	return results
}

// GlobalFileChanges is the shared change store.
var GlobalFileChanges = NewFileChangeStore(500)

// RecordedChangesSince returns file changes after the provided timestamp.
func RecordedChangesSince(ts time.Time) []RecordedFileChange {
	return GlobalFileChanges.Since(ts)
}

// RecordedChanges returns all recorded file changes.
func RecordedChanges() []RecordedFileChange {
	return GlobalFileChanges.All()
}

// RecordFileChanges captures and stores file changes after a delay.
// It is best-effort and meant to attribute changes to targeted agents.
func RecordFileChanges(session, root string, agents []string, before map[string]FileState, delay time.Duration) {
	if root == "" || len(before) == 0 {
		return
	}

	go func() {
		time.Sleep(delay)

		after, err := SnapshotDirectory(root, DefaultSnapshotOptions(root))
		if err != nil {
			return
		}

		changes := DetectFileChanges(before, after)
		if len(changes) == 0 {
			return
		}

		for _, change := range changes {
			GlobalFileChanges.Add(RecordedFileChange{
				Session:   session,
				Agents:    append([]string{}, agents...),
				Change:    change,
				Timestamp: time.Now(),
			})
		}
	}()
}
