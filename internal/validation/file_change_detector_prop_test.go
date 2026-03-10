package validation

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// **Validates: Requirements 1.1, 1.2, 1.4**
// Property 1: File Change Event Capture
// For any code file that the AI modifies or creates, the Validation_Engine should
// capture a File_Change_Event containing the correct file path.
func TestProperty_FileChangeEventCapture(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Create a FileChangeDetector
		fcd := NewFileChangeDetector()

		// Generate a random code file path
		// Use common code extensions: .go, .py, .js, .ts, .java, .c, .cpp
		ext := rapid.SampledFrom([]string{".go", ".py", ".js", ".ts", ".java", ".c", ".cpp"}).Draw(rt, "extension")
		fileName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_]{2,15}`).Draw(rt, "fileName")
		filePath := fileName + ext

		// Generate a random operation (create or modify, not delete for this test)
		operation := rapid.SampledFrom([]Operation{OperationCreate, OperationModify}).Draw(rt, "operation")

		// Set up a callback to capture the event
		var capturedEvent *FileChangeEvent
		var mu sync.Mutex
		fcd.OnFileChange(func(event FileChangeEvent) {
			mu.Lock()
			defer mu.Unlock()
			capturedEvent = &event
		})

		// Record the file change
		fcd.RecordFileChange(filePath, operation)

		// Give a small amount of time for the callback to execute
		time.Sleep(10 * time.Millisecond)

		// Property 1: The event should be captured
		mu.Lock()
		defer mu.Unlock()
		if capturedEvent == nil {
			rt.Fatalf("Expected file change event to be captured, but got nil")
		}

		// Property 2: The captured event should contain the correct file path
		if capturedEvent.FilePath != filePath {
			rt.Errorf("Expected FilePath=%s, got FilePath=%s", filePath, capturedEvent.FilePath)
		}

		// Property 3: The captured event should contain the correct operation
		if capturedEvent.Operation != operation {
			rt.Errorf("Expected Operation=%s, got Operation=%s", operation, capturedEvent.Operation)
		}

		// Property 4: The captured event should have a timestamp
		if capturedEvent.Timestamp.IsZero() {
			rt.Error("Expected non-zero Timestamp")
		}

		// Property 5: The timestamp should be recent (within last second)
		timeDiff := time.Since(capturedEvent.Timestamp)
		if timeDiff > time.Second {
			rt.Errorf("Expected recent timestamp, but got timestamp from %v ago", timeDiff)
		}

		// Property 6: The file should appear in GetModifiedFiles()
		modifiedFiles := fcd.GetModifiedFiles()
		found := false
		for _, f := range modifiedFiles {
			if f == filePath {
				found = true
				break
			}
		}
		if !found {
			rt.Errorf("Expected file %s to appear in GetModifiedFiles(), but it was not found. Got: %v",
				filePath, modifiedFiles)
		}
	})
}

// **Validates: Requirements 1.3**
// Property 2: Multiple File Event Capture
// For any set of files modified in a single AI operation, the Validation_Engine should
// capture File_Change_Events for all files in the set.
func TestProperty_MultipleFileEventCapture(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Create a FileChangeDetector
		fcd := NewFileChangeDetector()

		// Generate multiple file paths (2-5 files)
		numFiles := rapid.IntRange(2, 5).Draw(rt, "numFiles")
		var filePaths []string
		var operations []Operation

		for i := 0; i < numFiles; i++ {
			ext := rapid.SampledFrom([]string{".go", ".py", ".js", ".ts"}).Draw(rt, "extension")
			fileName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_]{2,15}`).Draw(rt, "fileName")
			filePath := fileName + ext
			filePaths = append(filePaths, filePath)

			operation := rapid.SampledFrom([]Operation{OperationCreate, OperationModify}).Draw(rt, "operation")
			operations = append(operations, operation)
		}

		// Set up a callback to capture all events
		var capturedEvents []FileChangeEvent
		var mu sync.Mutex
		fcd.OnFileChange(func(event FileChangeEvent) {
			mu.Lock()
			defer mu.Unlock()
			capturedEvents = append(capturedEvents, event)
		})

		// Record all file changes
		for i, filePath := range filePaths {
			fcd.RecordFileChange(filePath, operations[i])
		}

		// Give a small amount of time for callbacks to execute
		time.Sleep(20 * time.Millisecond)

		// Property 1: All events should be captured
		mu.Lock()
		defer mu.Unlock()
		if len(capturedEvents) != numFiles {
			rt.Fatalf("Expected %d events to be captured, got %d", numFiles, len(capturedEvents))
		}

		// Property 2: Each file path should appear in the captured events
		for i, expectedPath := range filePaths {
			found := false
			for _, event := range capturedEvents {
				if event.FilePath == expectedPath {
					found = true
					// Property 3: The operation should match
					if event.Operation != operations[i] {
						rt.Errorf("File %s: Expected Operation=%s, got Operation=%s",
							expectedPath, operations[i], event.Operation)
					}
					break
				}
			}
			if !found {
				rt.Errorf("Expected file %s to be captured in events, but it was not found", expectedPath)
			}
		}

		// Property 4: All files should appear in GetModifiedFiles()
		modifiedFiles := fcd.GetModifiedFiles()
		if len(modifiedFiles) != numFiles {
			rt.Errorf("Expected %d files in GetModifiedFiles(), got %d", numFiles, len(modifiedFiles))
		}

		for _, expectedPath := range filePaths {
			found := false
			for _, f := range modifiedFiles {
				if f == expectedPath {
					found = true
					break
				}
			}
			if !found {
				rt.Errorf("Expected file %s in GetModifiedFiles(), but it was not found. Got: %v",
					expectedPath, modifiedFiles)
			}
		}
	})
}

// Property 2 (Non-Code Filtering): Multiple File Event Capture with Non-Code Files
// This tests that non-code files are filtered out and don't appear in events or modified files.
func TestProperty_MultipleFileEventCapture_NonCodeFiltering(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Create a FileChangeDetector
		fcd := NewFileChangeDetector()

		// Generate a mix of code and non-code files
		numCodeFiles := rapid.IntRange(1, 3).Draw(rt, "numCodeFiles")
		numNonCodeFiles := rapid.IntRange(1, 3).Draw(rt, "numNonCodeFiles")

		var codeFilePaths []string
		var nonCodeFilePaths []string

		// Generate code files
		for i := 0; i < numCodeFiles; i++ {
			ext := rapid.SampledFrom([]string{".go", ".py", ".js"}).Draw(rt, "codeExtension")
			fileName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_]{2,15}`).Draw(rt, "codeFileName")
			filePath := fileName + ext
			codeFilePaths = append(codeFilePaths, filePath)
		}

		// Generate non-code files
		for i := 0; i < numNonCodeFiles; i++ {
			ext := rapid.SampledFrom([]string{".md", ".txt", ".json", ".yaml"}).Draw(rt, "nonCodeExtension")
			fileName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_]{2,15}`).Draw(rt, "nonCodeFileName")
			filePath := fileName + ext
			nonCodeFilePaths = append(nonCodeFilePaths, filePath)
		}

		// Set up a callback to capture events
		var capturedEvents []FileChangeEvent
		var mu sync.Mutex
		fcd.OnFileChange(func(event FileChangeEvent) {
			mu.Lock()
			defer mu.Unlock()
			capturedEvents = append(capturedEvents, event)
		})

		// Record all file changes (both code and non-code)
		for _, filePath := range codeFilePaths {
			fcd.RecordFileChange(filePath, OperationModify)
		}
		for _, filePath := range nonCodeFilePaths {
			fcd.RecordFileChange(filePath, OperationModify)
		}

		// Give a small amount of time for callbacks to execute
		time.Sleep(20 * time.Millisecond)

		// Property 1: Only code files should be captured in events
		mu.Lock()
		defer mu.Unlock()
		if len(capturedEvents) != numCodeFiles {
			rt.Errorf("Expected %d code file events, got %d events", numCodeFiles, len(capturedEvents))
		}

		// Property 2: Non-code files should NOT appear in captured events
		for _, nonCodePath := range nonCodeFilePaths {
			for _, event := range capturedEvents {
				if event.FilePath == nonCodePath {
					rt.Errorf("Non-code file %s should not be captured in events", nonCodePath)
				}
			}
		}

		// Property 3: Only code files should appear in GetModifiedFiles()
		modifiedFiles := fcd.GetModifiedFiles()
		if len(modifiedFiles) != numCodeFiles {
			rt.Errorf("Expected %d code files in GetModifiedFiles(), got %d", numCodeFiles, len(modifiedFiles))
		}

		// Property 4: Non-code files should NOT appear in GetModifiedFiles()
		for _, nonCodePath := range nonCodeFilePaths {
			for _, f := range modifiedFiles {
				if f == nonCodePath {
					rt.Errorf("Non-code file %s should not appear in GetModifiedFiles()", nonCodePath)
				}
			}
		}

		// Property 5: All code files should appear in GetModifiedFiles()
		for _, codePath := range codeFilePaths {
			found := false
			for _, f := range modifiedFiles {
				if f == codePath {
					found = true
					break
				}
			}
			if !found {
				rt.Errorf("Code file %s should appear in GetModifiedFiles(), but was not found", codePath)
			}
		}
	})
}

// Property 1 (Path Preservation): File Change Event Capture with Complex Paths
// This tests that file paths with directories are preserved correctly.
func TestProperty_FileChangeEventCapture_ComplexPaths(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Create a FileChangeDetector
		fcd := NewFileChangeDetector()

		// Generate a file path with directories
		numDirs := rapid.IntRange(1, 4).Draw(rt, "numDirs")
		var pathParts []string

		for i := 0; i < numDirs; i++ {
			dirName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_]{2,10}`).Draw(rt, "dirName")
			pathParts = append(pathParts, dirName)
		}

		ext := rapid.SampledFrom([]string{".go", ".py", ".js"}).Draw(rt, "extension")
		fileName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_]{2,15}`).Draw(rt, "fileName")
		pathParts = append(pathParts, fileName+ext)

		filePath := filepath.Join(pathParts...)

		// Set up a callback to capture the event
		var capturedEvent *FileChangeEvent
		var mu sync.Mutex
		fcd.OnFileChange(func(event FileChangeEvent) {
			mu.Lock()
			defer mu.Unlock()
			capturedEvent = &event
		})

		// Record the file change
		fcd.RecordFileChange(filePath, OperationModify)

		// Give a small amount of time for the callback to execute
		time.Sleep(10 * time.Millisecond)

		// Property 1: The event should be captured
		mu.Lock()
		defer mu.Unlock()
		if capturedEvent == nil {
			rt.Fatalf("Expected file change event to be captured for path %s", filePath)
		}

		// Property 2: The full path should be preserved exactly
		if capturedEvent.FilePath != filePath {
			rt.Errorf("Expected FilePath=%s, got FilePath=%s", filePath, capturedEvent.FilePath)
		}

		// Property 3: The path should contain all directory components
		for _, part := range pathParts[:len(pathParts)-1] { // Check all directory parts
			if !strings.Contains(capturedEvent.FilePath, part) {
				rt.Errorf("Expected FilePath to contain directory '%s', but it was not found in %s",
					part, capturedEvent.FilePath)
			}
		}

		// Property 4: The file should appear in GetModifiedFiles() with the full path
		modifiedFiles := fcd.GetModifiedFiles()
		found := false
		for _, f := range modifiedFiles {
			if f == filePath {
				found = true
				break
			}
		}
		if !found {
			rt.Errorf("Expected full path %s in GetModifiedFiles(), but it was not found", filePath)
		}
	})
}
