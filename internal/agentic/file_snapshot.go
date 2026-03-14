package agentic

import (
	"fmt"
	"os"
	"sort"
)

// FileSnapshotManager manages in-memory file snapshots for rollback.
// It stores the original content of files so they can be restored
// after failed fix attempts.
type FileSnapshotManager struct {
	snapshots map[string][]byte // absolute path → original content
}

// NewFileSnapshotManager creates an empty FileSnapshotManager.
func NewFileSnapshotManager() *FileSnapshotManager {
	return &FileSnapshotManager{
		snapshots: make(map[string][]byte),
	}
}

// Capture reads and stores the current content of the given file paths.
// If a file cannot be read, Capture returns an error immediately.
func (fsm *FileSnapshotManager) Capture(paths []string) error {
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("failed to capture snapshot of %s: %w", p, err)
		}
		fsm.snapshots[p] = data
	}
	return nil
}

// Restore writes all snapshot contents back to disk.
// Returns a list of paths that failed to restore.
func (fsm *FileSnapshotManager) Restore() []string {
	var failed []string
	for p, data := range fsm.snapshots {
		if err := os.WriteFile(p, data, 0644); err != nil {
			failed = append(failed, p)
		}
	}
	return failed
}

// Discard clears all stored snapshots.
func (fsm *FileSnapshotManager) Discard() {
	fsm.snapshots = make(map[string][]byte)
}

// HasSnapshot returns true if a snapshot exists for the given path.
func (fsm *FileSnapshotManager) HasSnapshot(path string) bool {
	_, ok := fsm.snapshots[path]
	return ok
}

// Paths returns all paths that have snapshots.
func (fsm *FileSnapshotManager) Paths() []string {
	paths := make([]string, 0, len(fsm.snapshots))
	for p := range fsm.snapshots {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}
