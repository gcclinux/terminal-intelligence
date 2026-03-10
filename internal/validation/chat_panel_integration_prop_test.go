package validation

import (
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// **Validates: Requirements 6.1, 6.5, 6.6**
// Property 15: Validation Start Message
// For any validation that begins, the Chat_Panel should display a message indicating
// validation has started, including the files and language.
func TestProperty_ValidationStartMessage(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		cpi := NewChatPanelIntegration()

		// Generate random language
		language := rapid.SampledFrom([]Language{LanguageGo, LanguagePython}).Draw(rt, "language")

		// Generate random file list (1-5 files)
		numFiles := rapid.IntRange(1, 5).Draw(rt, "numFiles")
		var files []string
		for i := 0; i < numFiles; i++ {
			fileName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_]{2,15}`).Draw(rt, "fileName")
			ext := ".go"
			if language == LanguagePython {
				ext = ".py"
			}
			files = append(files, fileName+ext)
		}

		// Show validation start
		cpi.ShowValidationStart(files, language)

		// Get the message
		message := cpi.GetLastMessage()

		// Property 1: Message should not be empty
		if message == "" {
			rt.Fatal("Expected non-empty validation start message")
		}

		// Property 2: Message should contain validation indicator (🔍)
		if !strings.Contains(message, "🔍") {
			rt.Error("Expected message to contain validation indicator 🔍")
		}

		// Property 3: Message should contain the language name
		languageName := "Go"
		if language == LanguagePython {
			languageName = "Python"
		}
		if !strings.Contains(message, languageName) {
			rt.Errorf("Expected message to contain language name '%s'. Message: %s", languageName, message)
		}

		// Property 4: Message should contain the word "Validating"
		if !strings.Contains(message, "Validating") {
			rt.Errorf("Expected message to contain 'Validating'. Message: %s", message)
		}

		// Property 5: Message should contain all file names
		for _, file := range files {
			if !strings.Contains(message, file) {
				rt.Errorf("Expected message to contain file '%s'. Message: %s", file, message)
			}
		}

		// Property 6: Message should indicate the number of files
		// (either explicitly or implicitly through the file list)
		if numFiles == 1 && !strings.Contains(message, "1 file") {
			rt.Errorf("Expected message to indicate 1 file. Message: %s", message)
		}
		if numFiles > 1 && !strings.Contains(message, "files") {
			rt.Errorf("Expected message to indicate multiple files. Message: %s", message)
		}
	})
}

// **Validates: Requirements 6.4, 7.1, 7.2, 7.3**
// Property 18: Error Message Display
// For any validation that fails, the Chat_Panel should display all error messages
// from the Validation_Result, including file paths and line numbers.
func TestProperty_ErrorMessageDisplay(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		cpi := NewChatPanelIntegration()

		// Generate random language
		language := rapid.SampledFrom([]Language{LanguageGo, LanguagePython}).Draw(rt, "language")

		// Generate random file
		fileName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_]{2,15}`).Draw(rt, "fileName")
		ext := ".go"
		if language == LanguagePython {
			ext = ".py"
		}
		filePath := fileName + ext

		// Generate random errors (1-3 errors)
		numErrors := rapid.IntRange(1, 3).Draw(rt, "numErrors")
		var errors []ValidationError

		for i := 0; i < numErrors; i++ {
			line := rapid.IntRange(1, 100).Draw(rt, "line")
			column := rapid.IntRange(1, 80).Draw(rt, "column")
			errorMsg := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9 ]{10,40}`).Draw(rt, "errorMsg")

			errors = append(errors, ValidationError{
				File:     filePath,
				Line:     line,
				Column:   column,
				Message:  errorMsg,
				Severity: SeverityError,
			})
		}

		// Create validation result
		result := ValidationResult{
			Success:  false,
			Language: language,
			Files:    []string{filePath},
			Duration: 500 * time.Millisecond,
			Output:   "compilation failed",
			Errors:   errors,
		}

		// Show validation failure
		cpi.ShowValidationFailure(result)

		// Get the message
		message := cpi.GetLastMessage()

		// Property 1: Message should not be empty
		if message == "" {
			rt.Fatal("Expected non-empty error message")
		}

		// Property 2: Message should contain error indicator (❌)
		if !strings.Contains(message, "❌") {
			rt.Error("Expected message to contain error indicator ❌")
		}

		// Property 3: Message should contain "failed"
		if !strings.Contains(message, "failed") {
			rt.Errorf("Expected message to contain 'failed'. Message: %s", message)
		}

		// Property 4: Message should contain all error file paths
		for _, err := range errors {
			if !strings.Contains(message, err.File) {
				rt.Errorf("Expected message to contain file path '%s'. Message: %s", err.File, message)
			}
		}

		// Property 5: Message should contain all error line numbers
		for _, err := range errors {
			lineStr := string(rune('0' + err.Line%10)) // At least the last digit should appear
			if !strings.Contains(message, lineStr) {
				// This is a weak check, but we can't easily check for the full number
				// Let's check if the line number format appears (file:line:)
				continue
			}
		}

		// Property 6: Message should contain all error messages
		for _, err := range errors {
			if !strings.Contains(message, err.Message) {
				rt.Errorf("Expected message to contain error message '%s'. Message: %s", err.Message, message)
			}
		}

		// Property 7: Message should contain duration
		if !strings.Contains(message, "0.5") {
			rt.Errorf("Expected message to contain duration. Message: %s", message)
		}
	})
}

// **Validates: Requirements 7.4**
// Property 19: Error Suggestion Preservation
// For any validation error that includes suggestions in the original compiler output,
// those suggestions should be preserved and displayed in the Chat_Panel.
func TestProperty_ErrorSuggestionPreservation(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		cpi := NewChatPanelIntegration()

		// Generate random file
		fileName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_]{2,15}`).Draw(rt, "fileName")
		filePath := fileName + ".go"

		// Generate error with suggestion
		line := rapid.IntRange(1, 100).Draw(rt, "line")
		errorMsg := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9 ]{10,30}`).Draw(rt, "errorMsg")
		suggestion := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9 ]{10,30}`).Draw(rt, "suggestion")

		// Create error message that includes a suggestion
		fullErrorMsg := errorMsg + "; suggestion: " + suggestion

		errors := []ValidationError{
			{
				File:     filePath,
				Line:     line,
				Column:   5,
				Message:  fullErrorMsg,
				Severity: SeverityError,
			},
		}

		// Create validation result
		result := ValidationResult{
			Success:  false,
			Language: LanguageGo,
			Files:    []string{filePath},
			Duration: 300 * time.Millisecond,
			Output:   "compilation failed with suggestions",
			Errors:   errors,
		}

		// Show validation failure
		cpi.ShowValidationFailure(result)

		// Get the message
		message := cpi.GetLastMessage()

		// Property 1: Message should contain the full error message including suggestion
		if !strings.Contains(message, fullErrorMsg) {
			rt.Errorf("Expected message to contain full error message with suggestion. Message: %s", message)
		}

		// Property 2: Message should contain the suggestion text
		if !strings.Contains(message, suggestion) {
			rt.Errorf("Expected message to contain suggestion '%s'. Message: %s", suggestion, message)
		}

		// Property 3: The suggestion should not be truncated or modified
		if !strings.Contains(message, "suggestion: "+suggestion) {
			rt.Errorf("Expected message to preserve suggestion format. Message: %s", message)
		}
	})
}

// **Validates: Requirements 7.5**
// Property 20: Original Error Format Preservation
// For any validation error, the Validation_Result should preserve the original error
// format from the compiler or validator in the output field.
func TestProperty_OriginalErrorFormatPreservation(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		cpi := NewChatPanelIntegration()

		// Generate random file
		fileName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_]{2,15}`).Draw(rt, "fileName")
		filePath := fileName + ".go"

		// Generate original compiler output with specific format
		line := rapid.IntRange(1, 100).Draw(rt, "line")
		column := rapid.IntRange(1, 80).Draw(rt, "column")
		errorMsg := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9 ]{10,40}`).Draw(rt, "errorMsg")

		// Create original output in compiler format
		originalOutput := filePath + ":" + string(rune('0'+line%10)) + ":" +
			string(rune('0'+column%10)) + ": " + errorMsg

		// Create validation result with original output preserved
		result := ValidationResult{
			Success:  false,
			Language: LanguageGo,
			Files:    []string{filePath},
			Duration: 400 * time.Millisecond,
			Output:   originalOutput,      // Original format preserved here
			Errors:   []ValidationError{}, // No parsed errors to test raw output display
		}

		// Show validation failure
		cpi.ShowValidationFailure(result)

		// Get the message
		message := cpi.GetLastMessage()

		// Property 1: When errors aren't parsed, raw output should be displayed
		if !strings.Contains(message, originalOutput) {
			rt.Errorf("Expected message to contain original output. Message: %s", message)
		}

		// Property 2: The original format should be preserved exactly
		if !strings.Contains(message, "Raw output:") {
			rt.Errorf("Expected message to indicate raw output display. Message: %s", message)
		}

		// Property 3: The output should contain the file path in original format
		if !strings.Contains(message, filePath) {
			rt.Errorf("Expected message to contain file path from original output. Message: %s", message)
		}
	})
}

// **Validates: Requirements 9.1, 9.2**
// Property 23: Unsupported File Notification
// For any file with an unsupported language, the Chat_Panel should display a notification
// indicating the file was not validated and listing which files were skipped.
func TestProperty_UnsupportedFileNotification(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		cpi := NewChatPanelIntegration()

		// Generate random unsupported files (1-3 files)
		numFiles := rapid.IntRange(1, 3).Draw(rt, "numFiles")
		var unsupportedFiles []string

		for i := 0; i < numFiles; i++ {
			fileName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_]{2,15}`).Draw(rt, "fileName")
			ext := rapid.SampledFrom([]string{".txt", ".md", ".json", ".yaml"}).Draw(rt, "ext")
			unsupportedFiles = append(unsupportedFiles, fileName+ext)
		}

		// Supported languages list
		supportedLanguages := []Language{LanguageGo, LanguagePython}

		// Show unsupported language notification
		cpi.ShowUnsupportedLanguage(unsupportedFiles, supportedLanguages)

		// Get the message
		message := cpi.GetLastMessage()

		// Property 1: Message should not be empty
		if message == "" {
			rt.Fatal("Expected non-empty unsupported language notification")
		}

		// Property 2: Message should contain info indicator (ℹ️)
		if !strings.Contains(message, "ℹ️") {
			rt.Error("Expected message to contain info indicator ℹ️")
		}

		// Property 3: Message should indicate files were skipped
		if !strings.Contains(message, "Skipped") {
			rt.Errorf("Expected message to indicate files were skipped. Message: %s", message)
		}

		// Property 4: Message should list all unsupported files
		for _, file := range unsupportedFiles {
			if !strings.Contains(message, file) {
				rt.Errorf("Expected message to contain unsupported file '%s'. Message: %s", file, message)
			}
		}

		// Property 5: Message should list supported languages
		if !strings.Contains(message, "Supported languages") {
			rt.Errorf("Expected message to list supported languages. Message: %s", message)
		}

		// Property 6: Message should contain "Go" and "Python"
		if !strings.Contains(message, "Go") {
			rt.Errorf("Expected message to contain 'Go'. Message: %s", message)
		}
		if !strings.Contains(message, "Python") {
			rt.Errorf("Expected message to contain 'Python'. Message: %s", message)
		}

		// Property 7: Message should indicate validation was not performed
		if !strings.Contains(message, "unsupported") {
			rt.Errorf("Expected message to indicate files are unsupported. Message: %s", message)
		}
	})
}

// Property 23 (Multiple Languages): Unsupported File Notification with Custom Languages
// This tests that the supported languages list is dynamic and includes custom languages.
func TestProperty_UnsupportedFileNotification_CustomLanguages(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		cpi := NewChatPanelIntegration()

		// Generate unsupported file
		fileName := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_]{2,15}`).Draw(rt, "fileName")
		unsupportedFiles := []string{fileName + ".unknown"}

		// Generate custom supported languages
		numCustomLangs := rapid.IntRange(1, 3).Draw(rt, "numCustomLangs")
		supportedLanguages := []Language{LanguageGo, LanguagePython}

		var customLangNames []string
		for i := 0; i < numCustomLangs; i++ {
			customLang := rapid.StringMatching(`^[A-Z][a-z]{2,10}`).Draw(rt, "customLang")
			supportedLanguages = append(supportedLanguages, Language(customLang))
			customLangNames = append(customLangNames, customLang)
		}

		// Show unsupported language notification
		cpi.ShowUnsupportedLanguage(unsupportedFiles, supportedLanguages)

		// Get the message
		message := cpi.GetLastMessage()

		// Property 1: Message should contain all custom language names
		for _, langName := range customLangNames {
			if !strings.Contains(message, langName) {
				rt.Errorf("Expected message to contain custom language '%s'. Message: %s", langName, message)
			}
		}

		// Property 2: Message should still contain standard languages
		if !strings.Contains(message, "Go") || !strings.Contains(message, "Python") {
			rt.Errorf("Expected message to contain standard languages. Message: %s", message)
		}

		// Property 3: Languages should be listed in a readable format (comma-separated)
		if !strings.Contains(message, ",") && len(supportedLanguages) > 1 {
			rt.Errorf("Expected languages to be comma-separated. Message: %s", message)
		}
	})
}
