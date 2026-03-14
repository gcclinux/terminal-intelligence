package agentic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileSnapshotManager(t *testing.T) {
	fsm := NewFileSnapshotManager()
	if fsm == nil {
		t.Fatal("NewFileSnapshotManager returned nil")
	}
	if len(fsm.Paths()) != 0 {
		t.Fatalf("expected 0 paths, got %d", len(fsm.Paths()))
	}
}

func TestCaptureAndHasSnapshot(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	os.WriteFile(f1, []byte("hello"), 0644)

	fsm := NewFileSnapshotManager()
	if err := fsm.Capture([]string{f1}); err != nil {
		t.Fatalf("Capture failed: %v", err)
	}

	if !fsm.HasSnapshot(f1) {
		t.Fatal("expected HasSnapshot to return true")
	}
	if fsm.HasSnapshot(filepath.Join(dir, "nonexistent.txt")) {
		t.Fatal("expected HasSnapshot to return false for unknown path")
	}
}

func TestCaptureErrorOnMissingFile(t *testing.T) {
	fsm := NewFileSnapshotManager()
	err := fsm.Capture([]string{"/nonexistent/path/file.txt"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRestoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "file.txt")
	original := []byte("original content")
	os.WriteFile(f1, original, 0644)

	fsm := NewFileSnapshotManager()
	if err := fsm.Capture([]string{f1}); err != nil {
		t.Fatalf("Capture failed: %v", err)
	}

	// Modify the file on disk
	os.WriteFile(f1, []byte("modified content"), 0644)

	// Restore should bring back original
	failed := fsm.Restore()
	if len(failed) != 0 {
		t.Fatalf("expected no failures, got %v", failed)
	}

	data, _ := os.ReadFile(f1)
	if string(data) != string(original) {
		t.Fatalf("expected %q, got %q", original, data)
	}
}

func TestRestoreMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string][]byte{
		filepath.Join(dir, "a.go"):   []byte("package a"),
		filepath.Join(dir, "b.go"):   []byte("package b"),
		filepath.Join(dir, "c.txt"):  []byte("some text"),
	}

	for p, content := range files {
		os.WriteFile(p, content, 0644)
	}

	fsm := NewFileSnapshotManager()
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	if err := fsm.Capture(paths); err != nil {
		t.Fatalf("Capture failed: %v", err)
	}

	// Modify all files
	for p := range files {
		os.WriteFile(p, []byte("overwritten"), 0644)
	}

	failed := fsm.Restore()
	if len(failed) != 0 {
		t.Fatalf("expected no failures, got %v", failed)
	}

	for p, expected := range files {
		data, _ := os.ReadFile(p)
		if string(data) != string(expected) {
			t.Errorf("file %s: expected %q, got %q", p, expected, data)
		}
	}
}

func TestDiscard(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "file.txt")
	os.WriteFile(f1, []byte("content"), 0644)

	fsm := NewFileSnapshotManager()
	fsm.Capture([]string{f1})

	if !fsm.HasSnapshot(f1) {
		t.Fatal("expected snapshot before discard")
	}

	fsm.Discard()

	if fsm.HasSnapshot(f1) {
		t.Fatal("expected no snapshot after discard")
	}
	if len(fsm.Paths()) != 0 {
		t.Fatalf("expected 0 paths after discard, got %d", len(fsm.Paths()))
	}
}

func TestPathsReturnsSorted(t *testing.T) {
	dir := t.TempDir()
	names := []string{"c.txt", "a.txt", "b.txt"}
	var paths []string
	for _, n := range names {
		p := filepath.Join(dir, n)
		os.WriteFile(p, []byte(n), 0644)
		paths = append(paths, p)
	}

	fsm := NewFileSnapshotManager()
	fsm.Capture(paths)

	result := fsm.Paths()
	for i := 1; i < len(result); i++ {
		if result[i] < result[i-1] {
			t.Fatalf("Paths() not sorted: %v", result)
		}
	}
}

func TestCaptureEmptyFile(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "empty.txt")
	os.WriteFile(f1, []byte{}, 0644)

	fsm := NewFileSnapshotManager()
	if err := fsm.Capture([]string{f1}); err != nil {
		t.Fatalf("Capture failed on empty file: %v", err)
	}

	// Modify and restore
	os.WriteFile(f1, []byte("not empty anymore"), 0644)
	fsm.Restore()

	data, _ := os.ReadFile(f1)
	if len(data) != 0 {
		t.Fatalf("expected empty file after restore, got %q", data)
	}
}

func TestCaptureOverwritesExistingSnapshot(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "file.txt")
	os.WriteFile(f1, []byte("v1"), 0644)

	fsm := NewFileSnapshotManager()
	fsm.Capture([]string{f1})

	// Update file and re-capture
	os.WriteFile(f1, []byte("v2"), 0644)
	fsm.Capture([]string{f1})

	// Modify again
	os.WriteFile(f1, []byte("v3"), 0644)

	// Restore should bring back v2 (the latest capture)
	fsm.Restore()
	data, _ := os.ReadFile(f1)
	if string(data) != "v2" {
		t.Fatalf("expected v2 after restore, got %q", data)
	}
}
