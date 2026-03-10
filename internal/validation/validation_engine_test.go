package validation

import (
	"strings"
	"testing"
	"time"
)

func TestValidationEngine_GroupFilesByLanguage(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected map[Language][]string
	}{
		{
			name:  "single Go file",
			files: []string{"main.go"},
			expected: map[Language][]string{
				LanguageGo: {"main.go"},
			},
		},
		{
			name:  "single Python file",
			files: []string{"script.py"},
			expected: map[Language][]string{
				LanguagePython: {"script.py"},
			},
		},
		{
			name:  "multiple Go files",
			files: []string{"main.go", "handler.go", "utils.go"},
			expected: map[Language][]string{
				LanguageGo: {"main.go", "handler.go", "utils.go"},
			},
		},
		{
			name:  "multiple Python files",
			files: []string{"app.py", "models.py", "views.py"},
			expected: map[Language][]string{
				LanguagePython: {"app.py", "models.py", "views.py"},
			},
		},
		{
			name:  "mixed Go and Python files",
			files: []string{"main.go", "script.py", "handler.go", "utils.py"},
			expected: map[Language][]string{
				LanguageGo:     {"main.go", "handler.go"},
				LanguagePython: {"script.py", "utils.py"},
			},
		},
		{
			name:  "unsupported files only",
			files: []string{"README.md", "config.yaml"},
			expected: map[Language][]string{
				LanguageUnsupported: {"README.md", "config.yaml"},
			},
		},
		{
			name:  "mixed supported and unsupported files",
			files: []string{"main.go", "README.md", "script.py", "config.yaml"},
			expected: map[Language][]string{
				LanguageGo:          {"main.go"},
				LanguagePython:      {"script.py"},
				LanguageUnsupported: {"README.md", "config.yaml"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			languageDetector := NewLanguageDetector()
			compilerInterface := NewCompilerInterface()
			chatPanelIntegration := NewChatPanelIntegration()
			engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

			result := engine.groupFilesByLanguage(tt.files)

			// Verify all expected languages are present
			for lang, expectedFiles := range tt.expected {
				actualFiles, exists := result[lang]
				if !exists {
					t.Errorf("Expected language %s not found in result", lang)
					continue
				}

				if len(actualFiles) != len(expectedFiles) {
					t.Errorf("Expected %d files for language %s, got %d", len(expectedFiles), lang, len(actualFiles))
					continue
				}

				// Verify all expected files are present
				fileSet := make(map[string]bool)
				for _, f := range actualFiles {
					fileSet[f] = true
				}

				for _, expectedFile := range expectedFiles {
					if !fileSet[expectedFile] {
						t.Errorf("Expected file %s not found in language group %s", expectedFile, lang)
					}
				}
			}

			// Verify no extra languages in result
			for lang := range result {
				if _, exists := tt.expected[lang]; !exists {
					t.Errorf("Unexpected language %s in result", lang)
				}
			}
		})
	}
}

func TestValidationEngine_ValidateFiles_EmptyList(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	session, err := engine.ValidateFiles([]string{})

	if err != nil {
		t.Errorf("Expected no error for empty file list, got %v", err)
	}

	if session != nil {
		t.Errorf("Expected nil session for empty file list, got %v", session)
	}

	// Verify no messages were sent to chat panel
	messages := chatPanelIntegration.GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected no chat messages for empty file list, got %d", len(messages))
	}
}

func TestValidationEngine_ValidateFiles_UnsupportedOnly(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	files := []string{"README.md", "config.yaml", "data.json"}
	session, err := engine.ValidateFiles(files)

	if err != nil {
		t.Errorf("Expected no error for unsupported files, got %v", err)
	}

	if session == nil {
		t.Fatalf("Expected session to be created")
	}

	// Verify session is completed
	if session.Status != StatusCompleted {
		t.Errorf("Expected status %s, got %s", StatusCompleted, session.Status)
	}

	// Verify no validation results (no supported files)
	if len(session.Results) != 0 {
		t.Errorf("Expected no validation results, got %d", len(session.Results))
	}

	// Verify unsupported file notification was sent
	messages := chatPanelIntegration.GetMessages()
	if len(messages) == 0 {
		t.Fatalf("Expected unsupported file notification")
	}

	lastMessage := messages[len(messages)-1]
	if !strings.HasPrefix(lastMessage, "ℹ️") {
		t.Errorf("Expected unsupported file notification (ℹ️), got: %s", lastMessage)
	}
}

func TestValidationEngine_GetStatus(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// Initially, status should be pending (no session)
	status := engine.GetStatus()
	if status == nil {
		t.Fatalf("Expected status to be non-nil")
	}
	if *status != StatusPending {
		t.Errorf("Expected initial status %s, got %s", StatusPending, *status)
	}
}

func TestValidationEngine_GetCurrentSession(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// Initially, no session
	session := engine.GetCurrentSession()
	if session != nil {
		t.Errorf("Expected no current session initially, got %v", session)
	}

	// Create a session by validating unsupported files (quick operation)
	files := []string{"README.md"}
	_, err := engine.ValidateFiles(files)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Now there should be a current session
	session = engine.GetCurrentSession()
	if session == nil {
		t.Errorf("Expected current session after validation")
	}
}

func TestValidationEngine_SessionCreation(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	files := []string{"README.md"}
	session, err := engine.ValidateFiles(files)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if session == nil {
		t.Fatalf("Expected session to be created")
	}

	// Verify session fields
	if session.ID == "" {
		t.Errorf("Expected session ID to be set")
	}

	if len(session.Files) != len(files) {
		t.Errorf("Expected %d files in session, got %d", len(files), len(session.Files))
	}

	if session.StartTime.IsZero() {
		t.Errorf("Expected StartTime to be set")
	}

	if session.EndTime == nil {
		t.Errorf("Expected EndTime to be set after validation completes")
	}

	if session.Status != StatusCompleted {
		t.Errorf("Expected status %s, got %s", StatusCompleted, session.Status)
	}
}

func TestValidationEngine_TimingRecording(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	files := []string{"README.md"}
	startBefore := time.Now()
	session, err := engine.ValidateFiles(files)
	endAfter := time.Now()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if session == nil {
		t.Fatalf("Expected session to be created")
	}

	// Verify StartTime is within expected range
	if session.StartTime.Before(startBefore) || session.StartTime.After(endAfter) {
		t.Errorf("StartTime %v is outside expected range [%v, %v]", session.StartTime, startBefore, endAfter)
	}

	// Verify EndTime is set and after or equal to StartTime
	if session.EndTime == nil {
		t.Fatalf("Expected EndTime to be set")
	}

	if session.EndTime.Before(session.StartTime) {
		t.Errorf("EndTime %v should not be before StartTime %v", session.EndTime, session.StartTime)
	}

	// Verify EndTime is within expected range
	if session.EndTime.Before(startBefore) || session.EndTime.After(endAfter) {
		t.Errorf("EndTime %v is outside expected range [%v, %v]", session.EndTime, startBefore, endAfter)
	}
}

func TestValidationEngine_Cancel(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// Cancel without a session should not panic
	engine.Cancel()

	// Create a session
	files := []string{"README.md"}
	_, err := engine.ValidateFiles(files)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Cancel after session completes should not panic
	engine.Cancel()
}
