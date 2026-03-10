package validation

import (
	"testing"
	"time"
)

// TestFileChangeEvent tests the FileChangeEvent type
func TestFileChangeEvent(t *testing.T) {
	event := FileChangeEvent{
		FilePath:  "test.go",
		Operation: OperationCreate,
		Timestamp: time.Now(),
	}

	if event.FilePath != "test.go" {
		t.Errorf("Expected FilePath to be 'test.go', got '%s'", event.FilePath)
	}

	if event.Operation != OperationCreate {
		t.Errorf("Expected Operation to be 'create', got '%s'", event.Operation)
	}
}

// TestValidationSession tests the ValidationSession type
func TestValidationSession(t *testing.T) {
	session := ValidationSession{
		ID:        "test-session-1",
		Files:     []string{"test.go", "main.go"},
		StartTime: time.Now(),
		Status:    StatusPending,
		Results:   []ValidationResult{},
	}

	if session.ID != "test-session-1" {
		t.Errorf("Expected ID to be 'test-session-1', got '%s'", session.ID)
	}

	if len(session.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(session.Files))
	}

	if session.Status != StatusPending {
		t.Errorf("Expected Status to be 'pending', got '%s'", session.Status)
	}
}

// TestValidationResult tests the ValidationResult type
func TestValidationResult(t *testing.T) {
	result := ValidationResult{
		Success:  true,
		Language: LanguageGo,
		Files:    []string{"test.go"},
		Duration: 100 * time.Millisecond,
		Output:   "compilation successful",
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.Language != LanguageGo {
		t.Errorf("Expected Language to be 'go', got '%s'", result.Language)
	}

	if result.Duration != 100*time.Millisecond {
		t.Errorf("Expected Duration to be 100ms, got %v", result.Duration)
	}
}

// TestValidationError tests the ValidationError type
func TestValidationError(t *testing.T) {
	err := ValidationError{
		File:     "test.go",
		Line:     10,
		Column:   5,
		Message:  "undefined: fmt.Printl",
		Severity: SeverityError,
		Code:     "E001",
	}

	if err.File != "test.go" {
		t.Errorf("Expected File to be 'test.go', got '%s'", err.File)
	}

	if err.Line != 10 {
		t.Errorf("Expected Line to be 10, got %d", err.Line)
	}

	if err.Column != 5 {
		t.Errorf("Expected Column to be 5, got %d", err.Column)
	}

	if err.Severity != SeverityError {
		t.Errorf("Expected Severity to be 'error', got '%s'", err.Severity)
	}
}

// TestLanguageConstants tests the Language constants
func TestLanguageConstants(t *testing.T) {
	tests := []struct {
		lang     Language
		expected string
	}{
		{LanguageGo, "go"},
		{LanguagePython, "python"},
		{LanguageUnsupported, "unsupported"},
	}

	for _, tt := range tests {
		if string(tt.lang) != tt.expected {
			t.Errorf("Expected language '%s', got '%s'", tt.expected, string(tt.lang))
		}
	}
}

// TestOperationConstants tests the Operation constants
func TestOperationConstants(t *testing.T) {
	tests := []struct {
		op       Operation
		expected string
	}{
		{OperationCreate, "create"},
		{OperationModify, "modify"},
		{OperationDelete, "delete"},
	}

	for _, tt := range tests {
		if string(tt.op) != tt.expected {
			t.Errorf("Expected operation '%s', got '%s'", tt.expected, string(tt.op))
		}
	}
}

// TestValidationStatusConstants tests the ValidationStatus constants
func TestValidationStatusConstants(t *testing.T) {
	tests := []struct {
		status   ValidationStatus
		expected string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Expected status '%s', got '%s'", tt.expected, string(tt.status))
		}
	}
}

// TestSeverityConstants tests the Severity constants
func TestSeverityConstants(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityError, "error"},
		{SeverityWarning, "warning"},
		{SeverityInfo, "info"},
	}

	for _, tt := range tests {
		if string(tt.severity) != tt.expected {
			t.Errorf("Expected severity '%s', got '%s'", tt.expected, string(tt.severity))
		}
	}
}
