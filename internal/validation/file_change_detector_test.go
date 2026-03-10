package validation

import (
	"sync"
	"testing"
	"time"
)

func TestNewFileChangeDetector(t *testing.T) {
	fcd := NewFileChangeDetector()

	if fcd == nil {
		t.Fatal("NewFileChangeDetector returned nil")
	}

	if fcd.modifiedFiles == nil {
		t.Error("modifiedFiles should be initialized")
	}

	if fcd.callbacks == nil {
		t.Error("callbacks should be initialized")
	}

	if fcd.nonCodeExts == nil {
		t.Error("nonCodeExts should be initialized")
	}
}

func TestOnFileChange(t *testing.T) {
	fcd := NewFileChangeDetector()

	callbackCalled := false
	fcd.OnFileChange(func(event FileChangeEvent) {
		callbackCalled = true
	})

	// Trigger a file change
	fcd.RecordFileChange("test.go", OperationModify)

	// Give callback time to execute
	time.Sleep(10 * time.Millisecond)

	if !callbackCalled {
		t.Error("Expected callback to be called")
	}
}

func TestRecordFileChange_SingleFile(t *testing.T) {
	fcd := NewFileChangeDetector()

	var capturedEvent *FileChangeEvent
	var mu sync.Mutex

	fcd.OnFileChange(func(event FileChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		capturedEvent = &event
	})

	filePath := "src/main.go"
	fcd.RecordFileChange(filePath, OperationCreate)

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if capturedEvent == nil {
		t.Fatal("Expected event to be captured")
	}

	if capturedEvent.FilePath != filePath {
		t.Errorf("Expected FilePath=%s, got %s", filePath, capturedEvent.FilePath)
	}

	if capturedEvent.Operation != OperationCreate {
		t.Errorf("Expected Operation=%s, got %s", OperationCreate, capturedEvent.Operation)
	}

	if capturedEvent.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestRecordFileChange_MultipleFiles(t *testing.T) {
	fcd := NewFileChangeDetector()

	var capturedEvents []FileChangeEvent
	var mu sync.Mutex

	fcd.OnFileChange(func(event FileChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		capturedEvents = append(capturedEvents, event)
	})

	files := []string{"main.go", "handler.go", "utils.go"}
	for _, file := range files {
		fcd.RecordFileChange(file, OperationModify)
	}

	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(capturedEvents) != len(files) {
		t.Errorf("Expected %d events, got %d", len(files), len(capturedEvents))
	}

	for i, file := range files {
		if capturedEvents[i].FilePath != file {
			t.Errorf("Event %d: Expected FilePath=%s, got %s", i, file, capturedEvents[i].FilePath)
		}
	}
}

func TestGetModifiedFiles(t *testing.T) {
	fcd := NewFileChangeDetector()

	// Initially should be empty
	files := fcd.GetModifiedFiles()
	if len(files) != 0 {
		t.Errorf("Expected empty list initially, got %d files", len(files))
	}

	// Add some files
	fcd.RecordFileChange("file1.go", OperationCreate)
	fcd.RecordFileChange("file2.py", OperationModify)

	time.Sleep(10 * time.Millisecond)

	files = fcd.GetModifiedFiles()
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// Verify files are present
	expectedFiles := map[string]bool{"file1.go": true, "file2.py": true}
	for _, file := range files {
		if !expectedFiles[file] {
			t.Errorf("Unexpected file in list: %s", file)
		}
	}
}

func TestClearModifiedFiles(t *testing.T) {
	fcd := NewFileChangeDetector()

	// Add some files
	fcd.RecordFileChange("file1.go", OperationCreate)
	fcd.RecordFileChange("file2.go", OperationModify)

	time.Sleep(10 * time.Millisecond)

	files := fcd.GetModifiedFiles()
	if len(files) != 2 {
		t.Errorf("Expected 2 files before clear, got %d", len(files))
	}

	// Clear the list
	fcd.ClearModifiedFiles()

	files = fcd.GetModifiedFiles()
	if len(files) != 0 {
		t.Errorf("Expected 0 files after clear, got %d", len(files))
	}
}

func TestNonCodeFileFiltering(t *testing.T) {
	fcd := NewFileChangeDetector()

	var capturedEvents []FileChangeEvent
	var mu sync.Mutex

	fcd.OnFileChange(func(event FileChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		capturedEvents = append(capturedEvents, event)
	})

	// Record both code and non-code files
	fcd.RecordFileChange("main.go", OperationModify)
	fcd.RecordFileChange("README.md", OperationModify)
	fcd.RecordFileChange("config.json", OperationModify)
	fcd.RecordFileChange("utils.py", OperationModify)
	fcd.RecordFileChange("notes.txt", OperationModify)

	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Only code files should be captured
	if len(capturedEvents) != 2 {
		t.Errorf("Expected 2 code file events, got %d", len(capturedEvents))
	}

	// Verify only code files are present
	for _, event := range capturedEvents {
		if event.FilePath == "README.md" || event.FilePath == "config.json" || event.FilePath == "notes.txt" {
			t.Errorf("Non-code file should not be captured: %s", event.FilePath)
		}
	}

	// Verify GetModifiedFiles also filters
	modifiedFiles := fcd.GetModifiedFiles()
	if len(modifiedFiles) != 2 {
		t.Errorf("Expected 2 code files in GetModifiedFiles(), got %d", len(modifiedFiles))
	}
}

func TestIsNonCodeFile(t *testing.T) {
	fcd := NewFileChangeDetector()

	tests := []struct {
		name      string
		filePath  string
		isNonCode bool
	}{
		{"Go file", "main.go", false},
		{"Python file", "script.py", false},
		{"JavaScript file", "app.js", false},
		{"Markdown file", "README.md", true},
		{"Text file", "notes.txt", true},
		{"JSON file", "config.json", true},
		{"YAML file", "config.yaml", true},
		{"YML file", "config.yml", true},
		{"XML file", "data.xml", true},
		{"HTML file", "index.html", true},
		{"CSS file", "style.css", true},
		{"Image file", "logo.png", true},
		{"Case insensitive", "README.MD", true},
		{"No extension", "Makefile", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fcd.isNonCodeFile(tt.filePath)
			if result != tt.isNonCode {
				t.Errorf("isNonCodeFile(%s) = %v, want %v", tt.filePath, result, tt.isNonCode)
			}
		})
	}
}

func TestMultipleCallbacks(t *testing.T) {
	fcd := NewFileChangeDetector()

	var callback1Called, callback2Called bool
	var mu sync.Mutex

	fcd.OnFileChange(func(event FileChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		callback1Called = true
	})

	fcd.OnFileChange(func(event FileChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		callback2Called = true
	})

	fcd.RecordFileChange("test.go", OperationModify)

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !callback1Called {
		t.Error("Expected callback1 to be called")
	}

	if !callback2Called {
		t.Error("Expected callback2 to be called")
	}
}

func TestConcurrentAccess(t *testing.T) {
	fcd := NewFileChangeDetector()

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fcd.RecordFileChange("file.go", OperationModify)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = fcd.GetModifiedFiles()
		}()
	}

	wg.Wait()

	// Should not panic and should have recorded files
	files := fcd.GetModifiedFiles()
	if len(files) == 0 {
		t.Error("Expected some files to be recorded")
	}
}

func TestFilePathExtraction(t *testing.T) {
	fcd := NewFileChangeDetector()

	var capturedEvent *FileChangeEvent
	var mu sync.Mutex

	fcd.OnFileChange(func(event FileChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		capturedEvent = &event
	})

	// Test with complex path
	complexPath := "src/internal/validation/handler.go"
	fcd.RecordFileChange(complexPath, OperationCreate)

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if capturedEvent == nil {
		t.Fatal("Expected event to be captured")
	}

	if capturedEvent.FilePath != complexPath {
		t.Errorf("Expected FilePath=%s, got %s", complexPath, capturedEvent.FilePath)
	}

	// Verify in GetModifiedFiles
	files := fcd.GetModifiedFiles()
	found := false
	for _, f := range files {
		if f == complexPath {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected path %s in GetModifiedFiles()", complexPath)
	}
}
