package docgen

import (
	"os"
	"path/filepath"
	"testing"
)

// Test write to empty directory → File created
func TestWriter_WriteToEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	doc := &GeneratedDoc{
		Type:     DocTypeUserManual,
		Content:  "# User Manual\n\nThis is a test manual.",
		Filename: "USER_MANUAL.md",
	}

	writer := NewFileWriter(tmpDir)
	result, err := writer.Write(doc, true)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Written {
		t.Errorf("Expected Written to be true")
	}

	if result.Existed {
		t.Errorf("Expected Existed to be false for new file")
	}

	// Verify file exists
	fullPath := filepath.Join(tmpDir, "USER_MANUAL.md")
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(content) != doc.Content {
		t.Errorf("Content mismatch. Expected: %q, Got: %q", doc.Content, string(content))
	}
}

// Test write with existing file → Conflict detected
func TestWriter_WriteWithExistingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing file
	existingContent := "# Old Manual\n\nOld content."
	fullPath := filepath.Join(tmpDir, "USER_MANUAL.md")
	err := os.WriteFile(fullPath, []byte(existingContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	doc := &GeneratedDoc{
		Type:     DocTypeUserManual,
		Content:  "# New Manual\n\nNew content.",
		Filename: "USER_MANUAL.md",
	}

	writer := NewFileWriter(tmpDir)

	// Test without overwrite - should fail
	result, err := writer.Write(doc, false)

	if err == nil {
		t.Errorf("Expected error for file conflict, got none")
	}

	if result == nil {
		t.Fatalf("Expected WriteResult even on error")
	}

	if !result.Existed {
		t.Errorf("Expected Existed to be true")
	}

	if result.Written {
		t.Errorf("Expected Written to be false when conflict detected")
	}

	// Verify original file unchanged
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != existingContent {
		t.Errorf("Original file was modified. Expected: %q, Got: %q", existingContent, string(content))
	}

	// Test with overwrite - should succeed
	result2, err2 := writer.Write(doc, true)

	if err2 != nil {
		t.Fatalf("Expected no error with overwrite, got: %v", err2)
	}

	if !result2.Written {
		t.Errorf("Expected Written to be true with overwrite")
	}

	if !result2.Existed {
		t.Errorf("Expected Existed to be true")
	}

	// Verify file was overwritten
	content2, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content2) != doc.Content {
		t.Errorf("File not overwritten. Expected: %q, Got: %q", doc.Content, string(content2))
	}
}

// Test write to read-only directory → Error returned
func TestWriter_WriteToReadOnlyDirectory(t *testing.T) {
	// Skip this test on Windows as chmod doesn't work the same way
	if os.Getenv("CI") != "" || os.Getenv("OS") == "Windows_NT" {
		t.Skip("Skipping read-only directory test on Windows/CI")
	}

	tmpDir := t.TempDir()

	// Make directory read-only (Unix-like systems only)
	err := os.Chmod(tmpDir, 0444)
	if err != nil {
		t.Fatalf("Failed to make directory read-only: %v", err)
	}

	// Restore permissions after test
	defer os.Chmod(tmpDir, 0755)

	doc := &GeneratedDoc{
		Type:     DocTypeUserManual,
		Content:  "# User Manual",
		Filename: "USER_MANUAL.md",
	}

	writer := NewFileWriter(tmpDir)
	result, err := writer.Write(doc, true)

	if err == nil {
		t.Errorf("Expected error when writing to read-only directory, got none")
	}

	if result == nil {
		t.Fatalf("Expected WriteResult even on error")
	}

	if result.Written {
		t.Errorf("Expected Written to be false when write fails")
	}
}

// Test CheckExists method
func TestWriter_CheckExists(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewFileWriter(tmpDir)

	// Check non-existent file
	if writer.CheckExists("NONEXISTENT.md") {
		t.Errorf("CheckExists returned true for non-existent file")
	}

	// Create a file
	fullPath := filepath.Join(tmpDir, "EXISTING.md")
	err := os.WriteFile(fullPath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Check existing file
	if !writer.CheckExists("EXISTING.md") {
		t.Errorf("CheckExists returned false for existing file")
	}
}

// Test nil document handling
func TestWriter_WriteNilDocument(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewFileWriter(tmpDir)
	result, err := writer.Write(nil, true)

	if err == nil {
		t.Errorf("Expected error for nil document, got none")
	}

	if result != nil {
		t.Errorf("Expected nil result for nil document")
	}
}

// Test all documentation types have correct filenames
func TestWriter_AllDocumentationTypes(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		docType  DocumentationType
		filename string
	}{
		{DocTypeUserManual, "USER_MANUAL.md"},
		{DocTypeInstallation, "INSTALLATION.md"},
		{DocTypeAPI, "API_REFERENCE.md"},
		{DocTypeTutorial, "TUTORIAL.md"},
		{DocTypeGeneral, "DOCUMENTATION.md"},
	}

	writer := NewFileWriter(tmpDir)

	for _, tc := range testCases {
		doc := &GeneratedDoc{
			Type:     tc.docType,
			Content:  "Test content",
			Filename: tc.filename,
		}

		result, err := writer.Write(doc, true)

		if err != nil {
			t.Errorf("Failed to write %s: %v", tc.filename, err)
			continue
		}

		if result.Filename != tc.filename {
			t.Errorf("Expected filename %s, got %s", tc.filename, result.Filename)
		}

		// Verify file exists
		fullPath := filepath.Join(tmpDir, tc.filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("File %s was not created", tc.filename)
		}
	}
}
