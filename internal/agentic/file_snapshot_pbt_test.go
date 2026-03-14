package agentic

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"pgregory.net/rapid"
)

// Feature: project-wide-agentic-fixer, Property 8: File snapshot round-trip
// **Validates: Requirements 5.2, 8.1, 8.2**
func TestProperty8_FileSnapshotRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dir := t.TempDir()

		// Generate 1-5 files with random content.
		numFiles := rapid.IntRange(1, 5).Draw(rt, "numFiles")
		originals := make(map[string][]byte, numFiles)
		paths := make([]string, 0, numFiles)

		for i := 0; i < numFiles; i++ {
			name := fmt.Sprintf("file_%d.txt", i)
			p := filepath.Join(dir, name)
			// Random content: 0-1024 bytes (includes empty files).
			content := rapid.SliceOfN(rapid.Byte(), 0, 1024).Draw(rt, fmt.Sprintf("content_%d", i))
			if err := os.WriteFile(p, content, 0644); err != nil {
				rt.Fatalf("failed to write test file %s: %v", p, err)
			}
			originals[p] = content
			paths = append(paths, p)
		}

		// Capture snapshots.
		fsm := NewFileSnapshotManager()
		if err := fsm.Capture(paths); err != nil {
			rt.Fatalf("Capture failed: %v", err)
		}

		// Write random modifications to each file.
		for _, p := range paths {
			modified := rapid.SliceOfN(rapid.Byte(), 0, 1024).Draw(rt, fmt.Sprintf("modified_%s", filepath.Base(p)))
			if err := os.WriteFile(p, modified, 0644); err != nil {
				rt.Fatalf("failed to write modified content to %s: %v", p, err)
			}
		}

		// Restore from snapshots.
		failed := fsm.Restore()
		if len(failed) != 0 {
			rt.Fatalf("Restore reported failures: %v", failed)
		}

		// Verify byte-for-byte equality with original content.
		for p, expected := range originals {
			actual, err := os.ReadFile(p)
			if err != nil {
				rt.Fatalf("failed to read restored file %s: %v", p, err)
			}
			if !bytes.Equal(actual, expected) {
				rt.Fatalf("file %s: content mismatch after restore\nexpected %d bytes, got %d bytes", p, len(expected), len(actual))
			}
		}
	})
}
