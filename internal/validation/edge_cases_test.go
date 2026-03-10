package validation

import (
	"strings"
	"testing"
	"time"
)

// Task 12.2: Write unit tests for edge cases

func TestEdgeCase_EmptyFileList(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// Empty file list should be handled gracefully
	session, err := pipeline.TriggerValidation([]string{})

	if err != nil {
		t.Errorf("Expected no error for empty file list, got %v", err)
	}

	if session != nil {
		t.Errorf("Expected nil session for empty file list, got %v", session)
	}

	// No messages should be sent
	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected no messages for empty file list, got %d", len(messages))
	}
}

func TestEdgeCase_AllFilesUnsupported(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// All unsupported files should show notification only
	files := []string{"README.md", "config.yaml", "data.json", "notes.txt"}
	session, err := pipeline.TriggerValidation(files)

	if err != nil {
		t.Errorf("Expected no error for unsupported files, got %v", err)
	}

	if session == nil {
		t.Fatal("Expected session to be created")
	}

	// Session should be completed
	if session.Status != StatusCompleted {
		t.Errorf("Expected status %s, got %s", StatusCompleted, session.Status)
	}

	// No validation results (no supported files)
	if len(session.Results) != 0 {
		t.Errorf("Expected no validation results, got %d", len(session.Results))
	}

	// Unsupported file notification should be sent
	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected unsupported file notification")
	}

	lastMessage := messages[len(messages)-1]
	if !strings.HasPrefix(lastMessage, "ℹ️") {
		t.Errorf("Expected unsupported file notification, got: %s", lastMessage)
	}
}

func TestEdgeCase_MixedSupportedAndUnsupported(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// Mix of supported and unsupported files
	files := []string{"main.go", "README.md", "script.py", "config.yaml"}
	session, err := pipeline.TriggerValidation(files)

	// Validation may fail (files don't exist), but session should be created
	if session == nil {
		t.Fatal("Expected session to be created")
	}

	// Unsupported file notification should be sent
	messages := pipeline.GetChatPanelIntegration().GetMessages()
	foundUnsupportedNotification := false
	for _, msg := range messages {
		if strings.HasPrefix(msg, "ℹ️") {
			foundUnsupportedNotification = true
			break
		}
	}

	if !foundUnsupportedNotification {
		t.Error("Expected unsupported file notification")
	}

	// Validation should have been attempted for supported files
	// (results may be empty if validation failed, but session should exist)
	if session.Status == StatusPending {
		t.Error("Validation should have progressed beyond pending status")
	}

	// Suppress unused variable warning
	_ = err
}

func TestEdgeCase_ConcurrentValidationRequests(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// Queue multiple validation requests concurrently
	files1 := []string{"README.md"}
	files2 := []string{"config.yaml"}
	files3 := []string{"notes.txt"}

	pipeline.QueueValidation(files1)
	pipeline.QueueValidation(files2)
	pipeline.QueueValidation(files3)

	// Wait for all validations to complete (reduced wait time)
	time.Sleep(150 * time.Millisecond)

	// All validations should have been processed
	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) < 3 {
		t.Errorf("Expected at least 3 messages (one per validation), got %d", len(messages))
	}
}

func TestEdgeCase_FileWithoutExtension(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// File without extension should be treated as unsupported
	files := []string{"Makefile", "Dockerfile"}
	session, err := engine.ValidateFiles(files)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if session == nil {
		t.Fatal("Expected session to be created")
	}

	// Should be completed with no validation results
	if session.Status != StatusCompleted {
		t.Errorf("Expected status %s, got %s", StatusCompleted, session.Status)
	}

	if len(session.Results) != 0 {
		t.Errorf("Expected no validation results, got %d", len(session.Results))
	}

	// Unsupported file notification should be sent
	messages := chatPanelIntegration.GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected unsupported file notification")
	}
}

func TestEdgeCase_CaseInsensitiveExtension(t *testing.T) {
	languageDetector := NewLanguageDetector()

	// Test case-insensitive extension matching
	testCases := []struct {
		file     string
		expected Language
	}{
		{"main.go", LanguageGo},
		{"main.GO", LanguageGo},
		{"main.Go", LanguageGo},
		{"script.py", LanguagePython},
		{"script.PY", LanguagePython},
		{"script.Py", LanguagePython},
	}

	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			lang := languageDetector.DetectLanguage(tc.file)
			if lang != tc.expected {
				t.Errorf("Expected language %s for file %s, got %s", tc.expected, tc.file, lang)
			}
		})
	}
}

func TestEdgeCase_DuplicateFiles(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// Duplicate files in the list
	files := []string{"main.go", "main.go", "script.py", "script.py"}
	session, err := engine.ValidateFiles(files)

	// Validation may fail (files don't exist), but session should be created
	if session == nil {
		t.Fatal("Expected session to be created")
	}

	// Session should have been created
	if session.StartTime.IsZero() {
		t.Error("Expected StartTime to be set")
	}

	// Suppress unused variable warning
	_ = err
}

func TestEdgeCase_VeryLongFilePath(t *testing.T) {
	languageDetector := NewLanguageDetector()

	// Very long file path
	longPath := strings.Repeat("a/", 100) + "file.go"
	lang := languageDetector.DetectLanguage(longPath)

	if lang != LanguageGo {
		t.Errorf("Expected language %s for long path, got %s", LanguageGo, lang)
	}
}

func TestEdgeCase_SpecialCharactersInFilePath(t *testing.T) {
	languageDetector := NewLanguageDetector()

	// File paths with special characters
	testCases := []struct {
		file     string
		expected Language
	}{
		{"file-name.go", LanguageGo},
		{"file_name.py", LanguagePython},
		{"file.name.go", LanguageGo},
		{"file name.py", LanguagePython},
		{"file@name.go", LanguageGo},
		{"file#name.py", LanguagePython},
	}

	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			lang := languageDetector.DetectLanguage(tc.file)
			if lang != tc.expected {
				t.Errorf("Expected language %s for file %s, got %s", tc.expected, tc.file, lang)
			}
		})
	}
}

func TestEdgeCase_MultipleExtensions(t *testing.T) {
	languageDetector := NewLanguageDetector()

	// Files with multiple extensions (should use the last one)
	testCases := []struct {
		file     string
		expected Language
	}{
		{"file.tar.go", LanguageGo},
		{"file.backup.py", LanguagePython},
		{"file.test.go", LanguageGo},
		{"file.old.py", LanguagePython},
	}

	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			lang := languageDetector.DetectLanguage(tc.file)
			if lang != tc.expected {
				t.Errorf("Expected language %s for file %s, got %s", tc.expected, tc.file, lang)
			}
		})
	}
}

func TestEdgeCase_EmptyFileName(t *testing.T) {
	languageDetector := NewLanguageDetector()

	// Empty file name
	lang := languageDetector.DetectLanguage("")
	if lang != LanguageUnsupported {
		t.Errorf("Expected language %s for empty file name, got %s", LanguageUnsupported, lang)
	}
}

func TestEdgeCase_OnlyExtension(t *testing.T) {
	languageDetector := NewLanguageDetector()

	// File name is just an extension
	testCases := []struct {
		file     string
		expected Language
	}{
		{".go", LanguageGo},
		{".py", LanguagePython},
		{".txt", LanguageUnsupported},
	}

	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			lang := languageDetector.DetectLanguage(tc.file)
			if lang != tc.expected {
				t.Errorf("Expected language %s for file %s, got %s", tc.expected, tc.file, lang)
			}
		})
	}
}
