package validation

import (
	"testing"
)

// Task 14.3: Write unit tests for per-unit status display

func TestPerUnitStatus_PythonPerFileValidation(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// Multiple Python files
	files := []string{"app.py", "models.py", "views.py"}
	session, err := engine.ValidateFiles(files)

	// Validation may fail (files don't exist), but session should be created
	if session == nil {
		t.Fatal("Expected session to be created")
	}

	// Should have one result per Python file
	pythonResults := 0
	for _, result := range session.Results {
		if result.Language == LanguagePython {
			pythonResults++
		}
	}

	if pythonResults != len(files) {
		t.Errorf("Expected %d Python results (one per file), got %d", len(files), pythonResults)
	}

	// Suppress unused variable warning
	_ = err
}

func TestPerUnitStatus_GoPackageLevelValidation(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// Multiple Go files (should be validated as one package)
	files := []string{"main.go", "handler.go", "utils.go"}
	session, err := engine.ValidateFiles(files)

	// Validation may fail (files don't exist), but session should be created
	if session == nil {
		t.Fatal("Expected session to be created")
	}

	// Should have one result for all Go files (package-level)
	goResults := 0
	for _, result := range session.Results {
		if result.Language == LanguageGo {
			goResults++
		}
	}

	if goResults != 1 {
		t.Errorf("Expected 1 Go result (package-level), got %d", goResults)
	}

	// Suppress unused variable warning
	_ = err
}

func TestPerUnitStatus_MixedLanguages(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// Mix of Go and Python files
	files := []string{"main.go", "handler.go", "app.py", "models.py"}
	session, err := engine.ValidateFiles(files)

	// Validation may fail (files don't exist), but session should be created
	if session == nil {
		t.Fatal("Expected session to be created")
	}

	// Count results by language
	goResults := 0
	pythonResults := 0
	for _, result := range session.Results {
		if result.Language == LanguageGo {
			goResults++
		} else if result.Language == LanguagePython {
			pythonResults++
		}
	}

	// Should have 1 Go result (package-level) and 2 Python results (per-file)
	if goResults != 1 {
		t.Errorf("Expected 1 Go result (package-level), got %d", goResults)
	}

	if pythonResults != 2 {
		t.Errorf("Expected 2 Python results (one per file), got %d", pythonResults)
	}

	// Suppress unused variable warning
	_ = err
}

func TestPerUnitStatus_SinglePythonFile(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// Single Python file
	files := []string{"script.py"}
	session, err := engine.ValidateFiles(files)

	// Validation may fail (file doesn't exist), but session should be created
	if session == nil {
		t.Fatal("Expected session to be created")
	}

	// Should have one result for the Python file
	pythonResults := 0
	for _, result := range session.Results {
		if result.Language == LanguagePython {
			pythonResults++
		}
	}

	if pythonResults != 1 {
		t.Errorf("Expected 1 Python result, got %d", pythonResults)
	}

	// Suppress unused variable warning
	_ = err
}

func TestPerUnitStatus_SingleGoFile(t *testing.T) {
	languageDetector := NewLanguageDetector()
	compilerInterface := NewCompilerInterface()
	chatPanelIntegration := NewChatPanelIntegration()
	engine := NewValidationEngine(languageDetector, compilerInterface, chatPanelIntegration)

	// Single Go file
	files := []string{"main.go"}
	session, err := engine.ValidateFiles(files)

	// Validation may fail (file doesn't exist), but session should be created
	if session == nil {
		t.Fatal("Expected session to be created")
	}

	// Should have one result for the Go file
	goResults := 0
	for _, result := range session.Results {
		if result.Language == LanguageGo {
			goResults++
		}
	}

	if goResults != 1 {
		t.Errorf("Expected 1 Go result, got %d", goResults)
	}

	// Suppress unused variable warning
	_ = err
}
