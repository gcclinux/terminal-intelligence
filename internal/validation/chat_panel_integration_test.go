package validation

import (
	"strings"
	"testing"
	"time"
)

func TestNewChatPanelIntegration(t *testing.T) {
	cpi := NewChatPanelIntegration()

	if cpi == nil {
		t.Fatal("NewChatPanelIntegration returned nil")
	}

	if cpi.messages == nil {
		t.Error("messages should be initialized")
	}

	if len(cpi.messages) != 0 {
		t.Error("messages should be empty initially")
	}
}

func TestShowValidationStart_SingleFile(t *testing.T) {
	cpi := NewChatPanelIntegration()

	files := []string{"main.go"}
	cpi.ShowValidationStart(files, LanguageGo)

	message := cpi.GetLastMessage()

	if message == "" {
		t.Fatal("Expected non-empty message")
	}

	if !strings.Contains(message, "🔍") {
		t.Error("Expected message to contain 🔍")
	}

	if !strings.Contains(message, "Validating") {
		t.Error("Expected message to contain 'Validating'")
	}

	if !strings.Contains(message, "Go") {
		t.Error("Expected message to contain 'Go'")
	}

	if !strings.Contains(message, "1 file") {
		t.Error("Expected message to indicate 1 file")
	}

	if !strings.Contains(message, "main.go") {
		t.Error("Expected message to contain file name")
	}
}

func TestShowValidationStart_MultipleFiles(t *testing.T) {
	cpi := NewChatPanelIntegration()

	files := []string{"main.go", "handler.go", "utils.go"}
	cpi.ShowValidationStart(files, LanguageGo)

	message := cpi.GetLastMessage()

	if !strings.Contains(message, "3 files") {
		t.Error("Expected message to indicate 3 files")
	}

	for _, file := range files {
		if !strings.Contains(message, file) {
			t.Errorf("Expected message to contain file %s", file)
		}
	}
}

func TestShowValidationStart_Python(t *testing.T) {
	cpi := NewChatPanelIntegration()

	files := []string{"script.py"}
	cpi.ShowValidationStart(files, LanguagePython)

	message := cpi.GetLastMessage()

	if !strings.Contains(message, "Python") {
		t.Error("Expected message to contain 'Python'")
	}

	if !strings.Contains(message, "script.py") {
		t.Error("Expected message to contain file name")
	}
}

func TestShowValidationProgress(t *testing.T) {
	cpi := NewChatPanelIntegration()

	cpi.ShowValidationProgress(LanguageGo, 5.2)

	message := cpi.GetLastMessage()

	if !strings.Contains(message, "⏳") {
		t.Error("Expected message to contain ⏳")
	}

	if !strings.Contains(message, "Compiling") {
		t.Error("Expected message to contain 'Compiling'")
	}

	if !strings.Contains(message, "Go") {
		t.Error("Expected message to contain 'Go'")
	}

	if !strings.Contains(message, "5.2") {
		t.Error("Expected message to contain duration")
	}
}

func TestShowValidationSuccess(t *testing.T) {
	cpi := NewChatPanelIntegration()

	result := ValidationResult{
		Success:  true,
		Language: LanguageGo,
		Files:    []string{"main.go"},
		Duration: 1200 * time.Millisecond,
		Output:   "",
		Errors:   []ValidationError{},
	}

	cpi.ShowValidationSuccess(result)

	message := cpi.GetLastMessage()

	if !strings.Contains(message, "✅") {
		t.Error("Expected message to contain ✅")
	}

	if !strings.Contains(message, "successful") {
		t.Error("Expected message to contain 'successful'")
	}

	if !strings.Contains(message, "Go") {
		t.Error("Expected message to contain 'Go'")
	}

	if !strings.Contains(message, "1.2") {
		t.Error("Expected message to contain duration in seconds")
	}
}

func TestShowValidationFailure_WithErrors(t *testing.T) {
	cpi := NewChatPanelIntegration()

	result := ValidationResult{
		Success:  false,
		Language: LanguageGo,
		Files:    []string{"main.go"},
		Duration: 800 * time.Millisecond,
		Output:   "compilation failed",
		Errors: []ValidationError{
			{
				File:     "main.go",
				Line:     15,
				Column:   2,
				Message:  "undefined: fmt.Printl",
				Severity: SeverityError,
			},
			{
				File:     "main.go",
				Line:     23,
				Column:   10,
				Message:  "syntax error: unexpected newline",
				Severity: SeverityError,
			},
		},
	}

	cpi.ShowValidationFailure(result)

	message := cpi.GetLastMessage()

	if !strings.Contains(message, "❌") {
		t.Error("Expected message to contain ❌")
	}

	if !strings.Contains(message, "failed") {
		t.Error("Expected message to contain 'failed'")
	}

	if !strings.Contains(message, "0.8") {
		t.Error("Expected message to contain duration")
	}

	// Check error details
	if !strings.Contains(message, "main.go:15:2") {
		t.Error("Expected message to contain file:line:column for first error")
	}

	if !strings.Contains(message, "undefined: fmt.Printl") {
		t.Error("Expected message to contain first error message")
	}

	if !strings.Contains(message, "main.go:23:10") {
		t.Error("Expected message to contain file:line:column for second error")
	}

	if !strings.Contains(message, "syntax error: unexpected newline") {
		t.Error("Expected message to contain second error message")
	}
}

func TestShowValidationFailure_WithoutColumn(t *testing.T) {
	cpi := NewChatPanelIntegration()

	result := ValidationResult{
		Success:  false,
		Language: LanguagePython,
		Files:    []string{"script.py"},
		Duration: 500 * time.Millisecond,
		Output:   "syntax error",
		Errors: []ValidationError{
			{
				File:     "script.py",
				Line:     10,
				Column:   0, // No column information
				Message:  "invalid syntax",
				Severity: SeverityError,
			},
		},
	}

	cpi.ShowValidationFailure(result)

	message := cpi.GetLastMessage()

	// Should show file:line format without column
	if !strings.Contains(message, "script.py:10:") {
		t.Error("Expected message to contain file:line: format")
	}

	if !strings.Contains(message, "invalid syntax") {
		t.Error("Expected message to contain error message")
	}
}

func TestShowValidationFailure_RawOutput(t *testing.T) {
	cpi := NewChatPanelIntegration()

	rawOutput := "main.go:15:2: undefined: fmt.Printl\nmain.go:23:10: syntax error"

	result := ValidationResult{
		Success:  false,
		Language: LanguageGo,
		Files:    []string{"main.go"},
		Duration: 600 * time.Millisecond,
		Output:   rawOutput,
		Errors:   []ValidationError{}, // No parsed errors
	}

	cpi.ShowValidationFailure(result)

	message := cpi.GetLastMessage()

	// Should display raw output when errors aren't parsed
	if !strings.Contains(message, "Raw output:") {
		t.Error("Expected message to indicate raw output")
	}

	if !strings.Contains(message, rawOutput) {
		t.Error("Expected message to contain raw output")
	}
}

func TestShowUnsupportedLanguage(t *testing.T) {
	cpi := NewChatPanelIntegration()

	unsupportedFiles := []string{"README.md", "config.yaml"}
	supportedLanguages := []Language{LanguageGo, LanguagePython}

	cpi.ShowUnsupportedLanguage(unsupportedFiles, supportedLanguages)

	message := cpi.GetLastMessage()

	if !strings.Contains(message, "ℹ️") {
		t.Error("Expected message to contain ℹ️")
	}

	if !strings.Contains(message, "Skipped") {
		t.Error("Expected message to contain 'Skipped'")
	}

	if !strings.Contains(message, "unsupported") {
		t.Error("Expected message to contain 'unsupported'")
	}

	for _, file := range unsupportedFiles {
		if !strings.Contains(message, file) {
			t.Errorf("Expected message to contain file %s", file)
		}
	}

	if !strings.Contains(message, "Supported languages:") {
		t.Error("Expected message to list supported languages")
	}

	if !strings.Contains(message, "Go") {
		t.Error("Expected message to contain 'Go'")
	}

	if !strings.Contains(message, "Python") {
		t.Error("Expected message to contain 'Python'")
	}
}

func TestGetMessages(t *testing.T) {
	cpi := NewChatPanelIntegration()

	// Initially empty
	messages := cpi.GetMessages()
	if len(messages) != 0 {
		t.Error("Expected empty messages initially")
	}

	// Add some messages
	cpi.ShowValidationStart([]string{"test.go"}, LanguageGo)
	cpi.ShowValidationProgress(LanguageGo, 2.5)

	messages = cpi.GetMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
}

func TestClearMessages(t *testing.T) {
	cpi := NewChatPanelIntegration()

	// Add messages
	cpi.ShowValidationStart([]string{"test.go"}, LanguageGo)
	cpi.ShowValidationProgress(LanguageGo, 1.0)

	messages := cpi.GetMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages before clear, got %d", len(messages))
	}

	// Clear messages
	cpi.ClearMessages()

	messages = cpi.GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(messages))
	}
}

func TestGetLastMessage(t *testing.T) {
	cpi := NewChatPanelIntegration()

	// Initially empty
	lastMessage := cpi.GetLastMessage()
	if lastMessage != "" {
		t.Error("Expected empty string for last message initially")
	}

	// Add messages
	cpi.ShowValidationStart([]string{"test.go"}, LanguageGo)
	firstMessage := cpi.GetLastMessage()

	cpi.ShowValidationProgress(LanguageGo, 3.0)
	secondMessage := cpi.GetLastMessage()

	if firstMessage == secondMessage {
		t.Error("Expected different messages")
	}

	if !strings.Contains(secondMessage, "⏳") {
		t.Error("Expected last message to be the progress message")
	}
}

func TestDurationFormatting(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "Less than 1 second",
			duration: 500 * time.Millisecond,
			expected: "0.5",
		},
		{
			name:     "Exactly 1 second",
			duration: 1000 * time.Millisecond,
			expected: "1.0",
		},
		{
			name:     "Multiple seconds",
			duration: 2500 * time.Millisecond,
			expected: "2.5",
		},
		{
			name:     "Long duration",
			duration: 10200 * time.Millisecond,
			expected: "10.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi := NewChatPanelIntegration()

			result := ValidationResult{
				Success:  true,
				Language: LanguageGo,
				Files:    []string{"test.go"},
				Duration: tt.duration,
				Output:   "",
				Errors:   []ValidationError{},
			}

			cpi.ShowValidationSuccess(result)
			message := cpi.GetLastMessage()

			if !strings.Contains(message, tt.expected) {
				t.Errorf("Expected message to contain duration %s, got: %s", tt.expected, message)
			}
		})
	}
}

func TestErrorSuggestionPreservation(t *testing.T) {
	cpi := NewChatPanelIntegration()

	result := ValidationResult{
		Success:  false,
		Language: LanguageGo,
		Files:    []string{"main.go"},
		Duration: 700 * time.Millisecond,
		Output:   "error with suggestion",
		Errors: []ValidationError{
			{
				File:     "main.go",
				Line:     20,
				Column:   5,
				Message:  "undefined: Printl; did you mean: Println?",
				Severity: SeverityError,
			},
		},
	}

	cpi.ShowValidationFailure(result)
	message := cpi.GetLastMessage()

	// Suggestion should be preserved in the error message
	if !strings.Contains(message, "did you mean: Println?") {
		t.Error("Expected message to preserve error suggestion")
	}

	if !strings.Contains(message, "undefined: Printl") {
		t.Error("Expected message to contain original error")
	}
}

func TestLongValidationProgressIndicator(t *testing.T) {
	cpi := NewChatPanelIntegration()

	// Test progress indicator for validation >5 seconds
	cpi.ShowValidationProgress(LanguageGo, 6.5)

	message := cpi.GetLastMessage()

	if !strings.Contains(message, "6.5") {
		t.Error("Expected message to show duration >5 seconds")
	}

	if !strings.Contains(message, "⏳") {
		t.Error("Expected progress indicator for long validation")
	}
}

func TestMultipleLanguageNames(t *testing.T) {
	tests := []struct {
		name         string
		language     Language
		expectedName string
	}{
		{
			name:         "Go language",
			language:     LanguageGo,
			expectedName: "Go",
		},
		{
			name:         "Python language",
			language:     LanguagePython,
			expectedName: "Python",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi := NewChatPanelIntegration()

			cpi.ShowValidationStart([]string{"test.file"}, tt.language)
			message := cpi.GetLastMessage()

			if !strings.Contains(message, tt.expectedName) {
				t.Errorf("Expected message to contain language name '%s'", tt.expectedName)
			}
		})
	}
}
