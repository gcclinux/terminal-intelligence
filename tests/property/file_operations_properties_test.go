package property

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/user/terminal-intelligence/internal/filemanager"
)

// Feature: Terminal Intelligence (TI), Property 1: File Content Round-Trip Persistence
// **Validates: Requirements 2.7, 3.3**
//
// For any file content written to disk through the File_Manager,
// reading the file back should return identical content.
func TestProperty_FileContentRoundTripPersistence(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("writing and reading file returns identical content", prop.ForAll(
		func(filename string, content string) bool {
			// Create temporary workspace for this test
			tmpDir := t.TempDir()
			fm := filemanager.NewFileManager(tmpDir)

			// Write content to file
			err := fm.WriteFile(filename, content)
			if err != nil {
				t.Logf("WriteFile failed: %v", err)
				return false
			}

			// Read content back
			readContent, err := fm.ReadFile(filename)
			if err != nil {
				t.Logf("ReadFile failed: %v", err)
				return false
			}

			// Verify content matches
			return readContent == content
		},
		genValidFilename(),
		gen.AnyString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: Terminal Intelligence (TI), Property 4: File Creation and Existence
// **Validates: Requirements 3.1**
//
// For any valid filename and file type, creating a file through the File_Manager
// should result in that file existing on the file system.
func TestProperty_FileCreationAndExistence(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("creating a file makes it exist on the file system", prop.ForAll(
		func(filename string, content string) bool {
			// Create temporary workspace for this test
			tmpDir := t.TempDir()
			fm := filemanager.NewFileManager(tmpDir)

			// Verify file doesn't exist before creation
			if fm.FileExists(filename) {
				t.Logf("File %s already exists before creation", filename)
				return false
			}

			// Create the file
			err := fm.CreateFile(filename, content)
			if err != nil {
				t.Logf("CreateFile failed: %v", err)
				return false
			}

			// Verify file exists after creation
			if !fm.FileExists(filename) {
				t.Logf("File %s does not exist after creation", filename)
				return false
			}

			return true
		},
		genValidFilename(),
		gen.AnyString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: Terminal Intelligence (TI), Property 5: File Deletion
// **Validates: Requirements 3.4**
//
// For any existing file, deleting it through the File_Manager
// should result in the file no longer existing on the file system.
func TestProperty_FileDeletion(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("deleting a file removes it from the file system", prop.ForAll(
		func(filename string, content string) bool {
			// Create temporary workspace for this test
			tmpDir := t.TempDir()
			fm := filemanager.NewFileManager(tmpDir)

			// Create a file first
			err := fm.CreateFile(filename, content)
			if err != nil {
				t.Logf("CreateFile failed: %v", err)
				return false
			}

			// Verify file exists before deletion
			if !fm.FileExists(filename) {
				t.Logf("File %s does not exist before deletion", filename)
				return false
			}

			// Delete the file
			err = fm.DeleteFile(filename)
			if err != nil {
				t.Logf("DeleteFile failed: %v", err)
				return false
			}

			// Verify file no longer exists after deletion
			if fm.FileExists(filename) {
				t.Logf("File %s still exists after deletion", filename)
				return false
			}

			return true
		},
		genValidFilename(),
		gen.AnyString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: Terminal Intelligence (TI), Property 6: File Operation Error Handling
// **Validates: Requirements 3.5**
//
// For any invalid file operation (opening non-existent file, writing to read-only location),
// the File_Manager should return an error result with a descriptive message.
func TestProperty_FileOperationErrorHandling(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Reading non-existent files returns descriptive error
	properties.Property("reading non-existent file returns descriptive error", prop.ForAll(
		func(filename string) bool {
			tmpDir := t.TempDir()
			fm := filemanager.NewFileManager(tmpDir)

			// Ensure file doesn't exist
			if fm.FileExists(filename) {
				return true // Skip if file somehow exists
			}

			// Attempt to read non-existent file
			_, err := fm.ReadFile(filename)
			
			// Should return an error
			if err == nil {
				t.Logf("Expected error when reading non-existent file %s, got nil", filename)
				return false
			}

			// Error message should be descriptive (contain "not found" or similar)
			errMsg := err.Error()
			if errMsg == "" {
				t.Logf("Error message is empty for non-existent file %s", filename)
				return false
			}

			return true
		},
		genValidFilename(),
	))

	// Property: Deleting non-existent files returns descriptive error
	properties.Property("deleting non-existent file returns descriptive error", prop.ForAll(
		func(filename string) bool {
			tmpDir := t.TempDir()
			fm := filemanager.NewFileManager(tmpDir)

			// Ensure file doesn't exist
			if fm.FileExists(filename) {
				return true // Skip if file somehow exists
			}

			// Attempt to delete non-existent file
			err := fm.DeleteFile(filename)
			
			// Should return an error
			if err == nil {
				t.Logf("Expected error when deleting non-existent file %s, got nil", filename)
				return false
			}

			// Error message should be descriptive
			errMsg := err.Error()
			if errMsg == "" {
				t.Logf("Error message is empty for non-existent file %s", filename)
				return false
			}

			return true
		},
		genValidFilename(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
