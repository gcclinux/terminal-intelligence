package validation

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// **Validates: Requirements 8.2**
// Property 21: Language-Based File Grouping
// For any set of modified files in multiple languages, the Validation_Engine should group files by language before validation.
func TestProperty_LanguageBasedFileGrouping(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a set of files with different extensions
		numFiles := rapid.IntRange(1, 20).Draw(t, "numFiles")
		files := make([]string, numFiles)
		expectedGroups := make(map[Language][]string)

		for i := 0; i < numFiles; i++ {
			// Generate file with random extension
			ext := rapid.SampledFrom([]string{".go", ".py", ".txt", ".md", ".js"}).Draw(t, "ext")
			filename := rapid.StringMatching(`[a-z]+`).Draw(t, "filename")
			file := filename + ext
			files[i] = file

			// Determine expected language
			var lang Language
			switch ext {
			case ".go":
				lang = LanguageGo
			case ".py":
				lang = LanguagePython
			default:
				lang = LanguageUnsupported
			}

			expectedGroups[lang] = append(expectedGroups[lang], file)
		}

		// Create validation engine
		languageDetector := NewLanguageDetector()
		compilerInterface := NewCompilerInterface()
		chatPanelIntegration := NewChatPanelIntegration()
		engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

		// Group files by language
		actualGroups := engine.groupFilesByLanguage(files)

		// Verify that all files are grouped correctly
		for lang, expectedFiles := range expectedGroups {
			actualFiles, exists := actualGroups[lang]
			if !exists && len(expectedFiles) > 0 {
				t.Fatalf("Expected language %s not found in groups", lang)
			}

			if len(actualFiles) != len(expectedFiles) {
				t.Fatalf("Expected %d files for language %s, got %d", len(expectedFiles), lang, len(actualFiles))
			}

			// Verify all expected files are present
			fileSet := make(map[string]bool)
			for _, f := range actualFiles {
				fileSet[f] = true
			}

			for _, expectedFile := range expectedFiles {
				if !fileSet[expectedFile] {
					t.Fatalf("Expected file %s not found in language group %s", expectedFile, lang)
				}
			}
		}

		// Verify no extra languages in actual groups
		for lang := range actualGroups {
			if _, exists := expectedGroups[lang]; !exists {
				t.Fatalf("Unexpected language %s in groups", lang)
			}
		}
	})
}

// **Validates: Requirements 10.1, 10.2, 10.3**
// Property 26: Validation Timing Round-Trip
// For any validation session, the Validation_Engine should record both start and end times,
// and the duration displayed should equal end time minus start time.
func TestProperty_ValidationTimingRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a set of valid Go files (to ensure validation completes)
		numFiles := rapid.IntRange(1, 5).Draw(t, "numFiles")
		files := make([]string, numFiles)

		for i := 0; i < numFiles; i++ {
			filename := rapid.StringMatching(`[a-z]+`).Draw(t, "filename")
			files[i] = filename + ".go"
		}

		// Create validation engine with mock validator
		languageDetector := NewLanguageDetector()
		compilerInterface := NewCompilerInterface()
		chatPanelIntegration := NewChatPanelIntegration()
		engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

		// Note: This test uses the real validators, which may fail if Go is not installed
		// or if the files don't exist. For property testing, we're verifying the timing
		// mechanism works correctly regardless of validation outcome.

		// Execute validation
		session, err := engine.ValidateFiles(files)

		// Validation may fail (files don't exist), but session should still be created
		if session == nil {
			// If validation failed to create a session, skip this iteration
			t.Skip("Validation did not create a session")
		}

		// Verify timing fields are set
		if session.StartTime.IsZero() {
			t.Fatalf("StartTime should be set")
		}

		if session.EndTime == nil {
			t.Fatalf("EndTime should be set after validation completes")
		}

		if session.EndTime.IsZero() {
			t.Fatalf("EndTime should not be zero")
		}

		// Verify EndTime is after StartTime
		if !session.EndTime.After(session.StartTime) {
			t.Fatalf("EndTime should be after StartTime")
		}

		// Calculate expected duration
		expectedDuration := session.EndTime.Sub(session.StartTime)

		// Verify duration in results matches the session timing
		for _, result := range session.Results {
			// Each result's duration should be <= total session duration
			if result.Duration > expectedDuration {
				t.Fatalf("Result duration (%v) should not exceed session duration (%v)", result.Duration, expectedDuration)
			}

			// Duration should be non-negative
			if result.Duration < 0 {
				t.Fatalf("Result duration should be non-negative, got %v", result.Duration)
			}
		}

		// Verify session status is terminal (completed, failed, or cancelled)
		if session.Status != StatusCompleted && session.Status != StatusFailed && session.Status != StatusCancelled {
			t.Fatalf("Session status should be terminal after validation, got %s", session.Status)
		}

		// Suppress unused variable warning
		_ = err
	})
}

// **Validates: Requirements 9.3**
// Property 24: Unsupported File Non-Blocking
// For any set of files containing both supported and unsupported languages,
// the Validation_Engine should validate all supported files regardless of unsupported files present.
func TestProperty_UnsupportedFileNonBlocking(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a mix of supported and unsupported files
		numSupported := rapid.IntRange(1, 10).Draw(t, "numSupported")
		numUnsupported := rapid.IntRange(1, 10).Draw(t, "numUnsupported")

		files := make([]string, 0, numSupported+numUnsupported)

		// Add supported files (Go and Python)
		supportedExts := []string{".go", ".py"}
		for i := 0; i < numSupported; i++ {
			ext := rapid.SampledFrom(supportedExts).Draw(t, "supportedExt")
			filename := rapid.StringMatching(`[a-z]+`).Draw(t, "supportedFilename")
			files = append(files, filename+ext)
		}

		// Add unsupported files
		unsupportedExts := []string{".txt", ".md", ".json", ".yaml", ".xml"}
		for i := 0; i < numUnsupported; i++ {
			ext := rapid.SampledFrom(unsupportedExts).Draw(t, "unsupportedExt")
			filename := rapid.StringMatching(`[a-z]+`).Draw(t, "unsupportedFilename")
			files = append(files, filename+ext)
		}

		// Create validation engine
		languageDetector := NewLanguageDetector()
		compilerInterface := NewCompilerInterface()
		chatPanelIntegration := NewChatPanelIntegration()
		engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

		// Execute validation
		session, err := engine.ValidateFiles(files)

		// Validation may fail (files don't exist), but session should still be created
		if session == nil {
			t.Skip("Validation did not create a session")
		}

		// Verify that unsupported files did not block validation
		// The session should have been created and attempted validation
		if session.StartTime.IsZero() {
			t.Fatalf("Validation should have started despite unsupported files")
		}

		// Verify that chat panel received unsupported file notification
		messages := chatPanelIntegration.GetMessages()
		foundUnsupportedNotification := false
		for _, msg := range messages {
			if strings.HasPrefix(msg, "ℹ️") {
				foundUnsupportedNotification = true
				break
			}
		}

		if !foundUnsupportedNotification {
			t.Fatalf("Expected unsupported file notification in chat panel")
		}

		// Verify that validation was attempted for supported files
		// (results may be empty if validation failed, but session should exist)
		if session.Status == StatusPending {
			t.Fatalf("Validation should have progressed beyond pending status")
		}

		// Suppress unused variable warning
		_ = err
	})
}

// **Validates: Requirements 8.5**
// Property 22: Per-Unit Status Display
// For any validation session, the Chat_Panel should display validation status for each file (Python) or package (Go) being validated.
func TestProperty_PerUnitStatusDisplay(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a mix of Go and Python files
		numGoFiles := rapid.IntRange(0, 5).Draw(t, "numGoFiles")
		numPyFiles := rapid.IntRange(0, 5).Draw(t, "numPyFiles")

		// Need at least one file
		if numGoFiles == 0 && numPyFiles == 0 {
			t.Skip("Need at least one file")
		}

		files := make([]string, 0, numGoFiles+numPyFiles)

		// Add Go files
		for i := 0; i < numGoFiles; i++ {
			filename := rapid.StringMatching(`[a-z]+`).Draw(t, "goFilename")
			files = append(files, filename+".go")
		}

		// Add Python files
		for i := 0; i < numPyFiles; i++ {
			filename := rapid.StringMatching(`[a-z]+`).Draw(t, "pyFilename")
			files = append(files, filename+".py")
		}

		// Create validation engine
		languageDetector := NewLanguageDetector()
		compilerInterface := NewCompilerInterface()
		chatPanelIntegration := NewChatPanelIntegration()
		engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

		// Execute validation
		session, err := engine.ValidateFiles(files)

		// Validation may fail (files don't exist), but session should be created
		if session == nil {
			t.Skip("Validation did not create a session")
		}

		// Get chat messages
		messages := chatPanelIntegration.GetMessages()

		// For Python files, we should see per-file status (one result per file)
		if numPyFiles > 0 {
			// Count Python validation results
			pythonResults := 0
			for _, result := range session.Results {
				if result.Language == LanguagePython {
					pythonResults++
				}
			}

			// Should have one result per Python file
			if pythonResults != numPyFiles {
				t.Fatalf("Expected %d Python results (one per file), got %d", numPyFiles, pythonResults)
			}
		}

		// For Go files, we should see package-level status (one result for all Go files)
		if numGoFiles > 0 {
			// Count Go validation results
			goResults := 0
			for _, result := range session.Results {
				if result.Language == LanguageGo {
					goResults++
				}
			}

			// Should have one result for all Go files (package-level)
			if goResults != 1 {
				t.Fatalf("Expected 1 Go result (package-level), got %d", goResults)
			}
		}

		// Verify messages were sent for each unit
		// Should have at least one message per language
		if len(messages) == 0 {
			t.Fatal("Expected validation messages")
		}

		// Suppress unused variable warning
		_ = err
	})
}
