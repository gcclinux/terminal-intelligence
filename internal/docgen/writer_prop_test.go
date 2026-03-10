package docgen

import (
	"os"
	"path/filepath"
	"testing"

	"pgregory.net/rapid"
)

// Feature: project-documentation-generation, Property 18: File Output Creation
// **Validates: Requirements 7.1**
// For any generated doc, file created on disk at workspace root
func TestProperty18_FileOutputCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping property-based test in short mode")
	}
	rapid.Check(t, func(rt *rapid.T) {
		// Create temporary workspace
		tmpDir := t.TempDir()

		// Generate random documentation
		docType := rapid.SampledFrom([]DocumentationType{
			DocTypeUserManual,
			DocTypeInstallation,
			DocTypeAPI,
			DocTypeTutorial,
			DocTypeGeneral,
		}).Draw(rt, "docType")

		content := rapid.String().Draw(rt, "content")
		filename := getFilenameForType(docType)

		doc := &GeneratedDoc{
			Type:     docType,
			Content:  content,
			Filename: filename,
		}

		// Create writer and write document
		writer := NewFileWriter(tmpDir)
		result, err := writer.Write(doc, true)

		// Verify file was created
		if err != nil {
			rt.Fatalf("Write failed: %v", err)
		}

		if !result.Written {
			rt.Fatalf("Write result indicates file was not written")
		}

		// Verify file exists on disk
		fullPath := filepath.Join(tmpDir, filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			rt.Fatalf("File was not created on disk: %s", fullPath)
		}

		// Verify content matches
		readContent, err := os.ReadFile(fullPath)
		if err != nil {
			rt.Fatalf("Failed to read written file: %v", err)
		}

		if string(readContent) != content {
			rt.Fatalf("File content does not match. Expected: %q, Got: %q", content, string(readContent))
		}
	})
}

// Feature: project-documentation-generation, Property 19: Standard Filename Mapping
// **Validates: Requirements 7.2, 7.3, 7.4, 7.5**
// For any doc type, correct standard filename used
func TestProperty19_StandardFilenameMapping(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping property-based test in short mode")
	}
	rapid.Check(t, func(rt *rapid.T) {
		// Create temporary workspace
		tmpDir := t.TempDir()

		// Generate random documentation type
		docType := rapid.SampledFrom([]DocumentationType{
			DocTypeUserManual,
			DocTypeInstallation,
			DocTypeAPI,
			DocTypeTutorial,
			DocTypeGeneral,
		}).Draw(rt, "docType")

		expectedFilename := getFilenameForType(docType)

		doc := &GeneratedDoc{
			Type:     docType,
			Content:  "test content",
			Filename: expectedFilename,
		}

		// Create writer and write document
		writer := NewFileWriter(tmpDir)
		result, err := writer.Write(doc, true)

		if err != nil {
			rt.Fatalf("Write failed: %v", err)
		}

		// Verify correct filename was used
		if result.Filename != expectedFilename {
			rt.Fatalf("Incorrect filename. Expected: %s, Got: %s", expectedFilename, result.Filename)
		}

		// Verify file exists with correct name
		fullPath := filepath.Join(tmpDir, expectedFilename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			rt.Fatalf("File with correct name was not created: %s", fullPath)
		}
	})
}

// Feature: project-documentation-generation, Property 20: File Conflict Detection
// **Validates: Requirements 7.6**
// For any generation where file exists, conflict detected
func TestProperty20_FileConflictDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping property-based test in short mode")
	}
	rapid.Check(t, func(rt *rapid.T) {
		// Create temporary workspace
		tmpDir := t.TempDir()

		// Generate random documentation
		docType := rapid.SampledFrom([]DocumentationType{
			DocTypeUserManual,
			DocTypeInstallation,
			DocTypeAPI,
			DocTypeTutorial,
			DocTypeGeneral,
		}).Draw(rt, "docType")

		filename := getFilenameForType(docType)
		originalContent := rapid.String().Draw(rt, "originalContent")
		newContent := rapid.String().Draw(rt, "newContent")

		// Create existing file
		fullPath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(fullPath, []byte(originalContent), 0644)
		if err != nil {
			rt.Fatalf("Failed to create existing file: %v", err)
		}

		doc := &GeneratedDoc{
			Type:     docType,
			Content:  newContent,
			Filename: filename,
		}

		// Create writer and attempt to write without overwrite
		writer := NewFileWriter(tmpDir)
		result, err := writer.Write(doc, false)

		// Verify conflict was detected
		if err == nil {
			rt.Fatalf("Expected error for file conflict, but got none")
		}

		if result == nil {
			rt.Fatalf("Expected WriteResult even on error")
		}

		if !result.Existed {
			rt.Fatalf("WriteResult should indicate file existed")
		}

		if result.Written {
			rt.Fatalf("WriteResult should indicate file was not written")
		}

		// Verify original file was not modified
		readContent, err := os.ReadFile(fullPath)
		if err != nil {
			rt.Fatalf("Failed to read file: %v", err)
		}

		if string(readContent) != originalContent {
			rt.Fatalf("Original file was modified. Expected: %q, Got: %q", originalContent, string(readContent))
		}

		// Now test with overwrite enabled
		result2, err2 := writer.Write(doc, true)

		if err2 != nil {
			rt.Fatalf("Write with overwrite failed: %v", err2)
		}

		if !result2.Existed {
			rt.Fatalf("WriteResult should indicate file existed")
		}

		if !result2.Written {
			rt.Fatalf("WriteResult should indicate file was written")
		}

		// Verify file was overwritten
		readContent2, err := os.ReadFile(fullPath)
		if err != nil {
			rt.Fatalf("Failed to read file: %v", err)
		}

		if string(readContent2) != newContent {
			rt.Fatalf("File was not overwritten. Expected: %q, Got: %q", newContent, string(readContent2))
		}
	})
}

// Helper function to get expected filename for a documentation type
func getFilenameForType(docType DocumentationType) string {
	switch docType {
	case DocTypeUserManual:
		return "USER_MANUAL.md"
	case DocTypeInstallation:
		return "INSTALLATION.md"
	case DocTypeAPI:
		return "API_REFERENCE.md"
	case DocTypeTutorial:
		return "TUTORIAL.md"
	case DocTypeGeneral:
		return "DOCUMENTATION.md"
	default:
		return "DOCUMENTATION.md"
	}
}
