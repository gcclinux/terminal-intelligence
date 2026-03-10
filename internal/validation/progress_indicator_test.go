package validation

import (
	"strings"
	"testing"
)

// Task 13.2: Write unit tests for progress indicator

func TestProgressIndicator_NotShownForQuickValidation(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// Quick validation (unsupported files)
	files := []string{"README.md"}
	_, err := engine.ValidateFiles(files)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check messages - should not contain progress indicator
	messages := chatPanelIntegration.GetMessages()
	for _, msg := range messages {
		if strings.Contains(msg, "⏳") {
			t.Errorf("Progress indicator should not be shown for quick validation, got: %s", msg)
		}
	}
}

func TestProgressIndicator_ShowsElapsedTime(t *testing.T) {
	chatPanelIntegration := NewChatPanelIntegration()

	// Directly test the chat panel progress message
	elapsed := 6.0
	chatPanelIntegration.ShowValidationProgress(LanguageGo, elapsed)

	// Check that progress message was sent
	messages := chatPanelIntegration.GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected progress message")
	}

	lastMessage := messages[len(messages)-1]
	if !strings.Contains(lastMessage, "⏳") {
		t.Errorf("Expected progress indicator (⏳), got: %s", lastMessage)
	}

	if !strings.Contains(lastMessage, "6.0s") {
		t.Errorf("Expected elapsed time in message, got: %s", lastMessage)
	}
}

func TestProgressIndicator_StopsAfterValidation(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// Validate files
	files := []string{"README.md"}
	_, err := engine.ValidateFiles(files)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Progress tracking should be stopped
	engine.mu.RLock()
	ticker := engine.progressTicker
	done := engine.progressDone
	engine.mu.RUnlock()

	if ticker != nil {
		t.Error("Progress ticker should be stopped after validation")
	}

	if done != nil {
		t.Error("Progress done channel should be closed after validation")
	}
}

func TestProgressIndicator_StopsOnCancel(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// Validate files (quick operation)
	files := []string{"README.md"}
	_, err := engine.ValidateFiles(files)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Cancel after validation completes (should not panic)
	engine.Cancel()

	// Progress tracking should be stopped
	engine.mu.RLock()
	ticker := engine.progressTicker
	done := engine.progressDone
	engine.mu.RUnlock()

	if ticker != nil {
		t.Error("Progress ticker should be stopped after cancel")
	}

	if done != nil {
		t.Error("Progress done channel should be closed after cancel")
	}
}

func TestProgressIndicator_MessageFormat(t *testing.T) {
	chatPanelIntegration := NewChatPanelIntegration()

	// Test progress message format
	testCases := []struct {
		language Language
		duration float64
	}{
		{LanguageGo, 5.5},
		{LanguagePython, 10.2},
		{LanguageGo, 15.0},
	}

	for _, tc := range testCases {
		chatPanelIntegration.ClearMessages()
		chatPanelIntegration.ShowValidationProgress(tc.language, tc.duration)

		messages := chatPanelIntegration.GetMessages()
		if len(messages) == 0 {
			t.Fatal("Expected progress message")
		}

		msg := messages[0]

		// Check for progress indicator icon
		if !strings.Contains(msg, "⏳") {
			t.Errorf("Expected progress indicator (⏳) in message: %s", msg)
		}

		// Check for language name
		langName := string(tc.language)
		if tc.language == LanguageGo {
			langName = "Go"
		} else if tc.language == LanguagePython {
			langName = "Python"
		}

		if !strings.Contains(msg, langName) {
			t.Errorf("Expected language name %s in message: %s", langName, msg)
		}

		// Check for duration
		if !strings.Contains(msg, "s)") {
			t.Errorf("Expected duration in message: %s", msg)
		}
	}
}
