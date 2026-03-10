package validation

import (
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Feature: automatic-code-validation, Property 16: Validation Progress Display
// **Validates: Requirements 6.2**
//
// For any validation in progress, the Chat_Panel should display the current
// validation activity status.
func TestProperty16_ValidationProgressDisplay(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a validation session that's in progress
		language := rapid.SampledFrom([]Language{LanguageGo, LanguagePython}).Draw(t, "language")
		duration := rapid.Float64Range(0.1, 30.0).Draw(t, "duration")

		// Create chat panel integration
		chatPanel := NewChatPanelIntegration()

		// Show validation progress
		chatPanel.ShowValidationProgress(language, duration)

		// Get messages
		messages := chatPanel.GetMessages()

		// Should have at least 1 message
		if len(messages) < 1 {
			t.Fatalf("Expected at least 1 message, got %d", len(messages))
		}

		// Last message should be a progress message
		lastMsg := messages[len(messages)-1]

		// Progress message should contain progress indicator
		if !strings.Contains(lastMsg, "⏳") {
			t.Errorf("Progress message should contain ⏳ emoji, got: %s", lastMsg)
		}

		// Progress message should indicate validation is in progress
		if !strings.Contains(strings.ToLower(lastMsg), "compil") {
			t.Errorf("Progress message should mention compilation, got: %s", lastMsg)
		}

		// Progress message should include duration
		if !strings.Contains(lastMsg, "s)") {
			t.Errorf("Progress message should include duration in seconds, got: %s", lastMsg)
		}
	})
}

// Feature: automatic-code-validation, Property 17: Success Message Display
// **Validates: Requirements 6.3**
//
// For any validation that completes successfully, the Chat_Panel should display
// a success message with the duration.
func TestProperty17_SuccessMessageDisplay(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a successful validation result
		files := rapid.SliceOfN(rapid.String(), 1, 5).Draw(t, "files")
		language := rapid.SampledFrom([]Language{LanguageGo, LanguagePython}).Draw(t, "language")
		duration := time.Duration(rapid.Int64Range(100, 10000).Draw(t, "duration_ms")) * time.Millisecond

		result := ValidationResult{
			Success:  true,
			Language: language,
			Files:    files,
			Duration: duration,
			Output:   "",
			Errors:   []ValidationError{},
			Warnings: []ValidationError{},
		}

		// Create chat panel integration
		chatPanel := NewChatPanelIntegration()

		// Show validation success
		chatPanel.ShowValidationSuccess(result)

		// Get messages
		messages := chatPanel.GetMessages()

		// Should have at least 1 message
		if len(messages) < 1 {
			t.Fatalf("Expected at least 1 message, got %d", len(messages))
		}

		// Last message should be a success message
		lastMsg := messages[len(messages)-1]

		// Success message should contain success indicator
		if !strings.Contains(lastMsg, "✅") {
			t.Errorf("Success message should contain ✅ emoji, got: %s", lastMsg)
		}

		// Success message should mention success
		if !strings.Contains(strings.ToLower(lastMsg), "success") {
			t.Errorf("Success message should mention 'success', got: %s", lastMsg)
		}

		// Success message should include duration
		// Duration should be formatted (e.g., "1.2s", "500ms")
		hasDuration := strings.Contains(lastMsg, "s)") || strings.Contains(lastMsg, "ms)")
		if !hasDuration {
			t.Errorf("Success message should include duration, got: %s", lastMsg)
		}
	})
}

// Feature: automatic-code-validation, Property 25: Supported Languages Listing
// **Validates: Requirements 9.4**
//
// For any unsupported file notification, the Chat_Panel should list all
// currently supported languages.
func TestProperty25_SupportedLanguagesListing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate unsupported files
		numFiles := rapid.IntRange(1, 5).Draw(t, "num_files")
		unsupportedFiles := make([]string, numFiles)
		for i := 0; i < numFiles; i++ {
			unsupportedFiles[i] = rapid.String().Draw(t, "file") + ".unsupported"
		}

		// Generate supported languages (at least Go and Python)
		supportedLanguages := []Language{LanguageGo, LanguagePython}

		// Optionally add more languages
		addMore := rapid.Bool().Draw(t, "add_more_languages")
		if addMore {
			extraLangs := rapid.SliceOfN(
				rapid.SampledFrom([]Language{"javascript", "typescript", "ruby", "rust"}),
				1, 3,
			).Draw(t, "extra_languages")
			supportedLanguages = append(supportedLanguages, extraLangs...)
		}

		// Create chat panel integration
		chatPanel := NewChatPanelIntegration()

		// Show unsupported language notification
		chatPanel.ShowUnsupportedLanguage(unsupportedFiles, supportedLanguages)

		// Get messages
		messages := chatPanel.GetMessages()

		// Should have at least 1 message
		if len(messages) < 1 {
			t.Fatalf("Expected at least 1 message, got %d", len(messages))
		}

		// Last message should be an unsupported language notification
		lastMsg := messages[len(messages)-1]

		// Message should contain info indicator
		if !strings.Contains(lastMsg, "ℹ️") {
			t.Errorf("Unsupported language message should contain ℹ️ emoji, got: %s", lastMsg)
		}

		// Message should list all supported languages
		for _, lang := range supportedLanguages {
			langName := strings.Title(string(lang))
			if !strings.Contains(lastMsg, langName) && !strings.Contains(lastMsg, string(lang)) {
				t.Errorf("Message should list supported language '%s', got: %s", lang, lastMsg)
			}
		}

		// Message should mention "supported" or "support"
		if !strings.Contains(strings.ToLower(lastMsg), "support") {
			t.Errorf("Message should mention supported languages, got: %s", lastMsg)
		}
	})
}
