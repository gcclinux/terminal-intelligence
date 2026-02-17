package ui

import (
	"testing"

	"github.com/user/terminal-intelligence/internal/filemanager"
)

// TestFixApplicationMarksFileModified verifies that when a fix is applied via SetContent,
// the file is marked as modified but NOT automatically saved
// Requirements: 6.2
func TestFixApplicationMarksFileModified(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)

	// Create a test file
	testFile := "test.sh"
	initialContent := "#!/bin/bash\necho 'original'\n"
	err := fm.CreateFile(testFile, initialContent)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create editor pane and load the file
	editor := NewEditorPane(fm)
	err = editor.LoadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}

	// Verify initial state - file should not be modified
	if editor.HasUnsavedChanges() {
		t.Error("Newly loaded file should not have unsaved changes")
	}

	// Simulate applying a fix by calling SetContent (as done in handleAIMessage)
	fixedContent := "#!/bin/bash\necho 'fixed'\n"
	editor.SetContent(fixedContent)

	// Verify the file is marked as modified
	if !editor.HasUnsavedChanges() {
		t.Error("After SetContent, file should be marked as modified")
	}

	// Verify the content was updated
	if editor.GetContent() != fixedContent {
		t.Errorf("Content = %q, want %q", editor.GetContent(), fixedContent)
	}

	// Verify the file was NOT automatically saved to disk
	diskContent, err := fm.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file from disk: %v", err)
	}
	if diskContent != initialContent {
		t.Errorf("File on disk should still have original content.\nGot: %q\nWant: %q", diskContent, initialContent)
	}

	// Verify user can manually save with SaveFile (simulating Ctrl+S)
	err = editor.SaveFile()
	if err != nil {
		t.Fatalf("Failed to save file: %v", err)
	}

	// After save, file should not be marked as modified
	if editor.HasUnsavedChanges() {
		t.Error("After SaveFile, file should not have unsaved changes")
	}

	// Verify the file was saved to disk
	diskContent, err = fm.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file from disk after save: %v", err)
	}
	if diskContent != fixedContent {
		t.Errorf("After save, file on disk should have fixed content.\nGot: %q\nWant: %q", diskContent, fixedContent)
	}
}

// TestSetContentWithNoFileOpen verifies that SetContent works even when no file is open
func TestSetContentWithNoFileOpen(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := NewEditorPane(fm)

	// No file is loaded
	if editor.currentFile != nil {
		t.Fatal("Expected no file to be loaded initially")
	}

	// SetContent should still work
	content := "some content"
	editor.SetContent(content)

	// Content should be set
	if editor.GetContent() != content {
		t.Errorf("Content = %q, want %q", editor.GetContent(), content)
	}

	// HasUnsavedChanges should return true since we have content but no saved file
	if !editor.HasUnsavedChanges() {
		t.Error("With content but no file open, HasUnsavedChanges should return true")
	}

	// currentFile.IsModified should not be set since no file is open
	if editor.currentFile != nil && editor.currentFile.IsModified {
		t.Error("With no file open, currentFile.IsModified should not be set")
	}
}

// TestModifiedFlagInUI verifies that the modified flag appears in the UI
func TestModifiedFlagInUI(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)

	// Create and load a test file
	testFile := "test.sh"
	initialContent := "#!/bin/bash\n"
	err := fm.CreateFile(testFile, initialContent)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	editor := NewEditorPane(fm)
	err = editor.LoadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}

	// Verify currentFile.IsModified is false initially
	if editor.currentFile.IsModified {
		t.Error("Newly loaded file should have IsModified = false")
	}

	// Apply a fix
	editor.SetContent("#!/bin/bash\necho 'modified'\n")

	// Verify currentFile.IsModified is true
	if !editor.currentFile.IsModified {
		t.Error("After SetContent, currentFile.IsModified should be true")
	}

	// Save the file
	err = editor.SaveFile()
	if err != nil {
		t.Fatalf("Failed to save file: %v", err)
	}

	// Verify currentFile.IsModified is false after save
	if editor.currentFile.IsModified {
		t.Error("After SaveFile, currentFile.IsModified should be false")
	}
}
