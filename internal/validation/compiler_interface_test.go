package validation

import (
	"context"
	"testing"
	"time"
)

// MockValidator is a mock implementation of the Validator interface for testing
type MockValidator struct {
	executeFunc func(files []string) (ValidationResult, error)
	info        ValidatorInfo
}

func (m *MockValidator) Execute(files []string) (ValidationResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(files)
	}
	return ValidationResult{
		Success:  true,
		Language: LanguageGo,
		Files:    files,
		Duration: 100 * time.Millisecond,
		Output:   "mock output",
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}, nil
}

func (m *MockValidator) GetInfo() ValidatorInfo {
	return m.info
}

func TestNewCompilerInterface(t *testing.T) {
	ci := NewCompilerInterface()

	if ci == nil {
		t.Fatal("NewCompilerInterface returned nil")
	}

	if ci.validators == nil {
		t.Fatal("validators map is nil")
	}

	// Verify GoValidator is registered
	goValidator := ci.GetValidator(LanguageGo)
	if goValidator == nil {
		t.Error("GoValidator not registered by default")
	}

	// Verify PythonValidator is registered
	pythonValidator := ci.GetValidator(LanguagePython)
	if pythonValidator == nil {
		t.Error("PythonValidator not registered by default")
	}
}

func TestRegisterValidator(t *testing.T) {
	ci := NewCompilerInterface()

	mockValidator := &MockValidator{
		info: ValidatorInfo{
			Name:    "MockValidator",
			Version: "1.0.0",
			Command: "mock",
		},
	}

	// Register a custom validator
	customLang := Language("custom")
	ci.RegisterValidator(customLang, mockValidator)

	// Verify it was registered
	retrieved := ci.GetValidator(customLang)
	if retrieved == nil {
		t.Fatal("Custom validator not registered")
	}

	if retrieved != mockValidator {
		t.Error("Retrieved validator is not the same as registered")
	}
}

func TestGetValidator(t *testing.T) {
	ci := NewCompilerInterface()

	tests := []struct {
		name     string
		language Language
		wantNil  bool
	}{
		{
			name:     "Get Go validator",
			language: LanguageGo,
			wantNil:  false,
		},
		{
			name:     "Get Python validator",
			language: LanguagePython,
			wantNil:  false,
		},
		{
			name:     "Get unsupported language",
			language: LanguageUnsupported,
			wantNil:  true,
		},
		{
			name:     "Get non-existent language",
			language: Language("nonexistent"),
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := ci.GetValidator(tt.language)
			if tt.wantNil && validator != nil {
				t.Errorf("Expected nil validator for %s, got %v", tt.language, validator)
			}
			if !tt.wantNil && validator == nil {
				t.Errorf("Expected non-nil validator for %s, got nil", tt.language)
			}
		})
	}
}

func TestValidate_Success(t *testing.T) {
	ci := NewCompilerInterface()

	mockValidator := &MockValidator{
		executeFunc: func(files []string) (ValidationResult, error) {
			return ValidationResult{
				Success:  true,
				Language: Language("mock"),
				Files:    files,
				Duration: 100 * time.Millisecond,
				Output:   "validation successful",
				Errors:   []ValidationError{},
				Warnings: []ValidationError{},
			}, nil
		},
		info: ValidatorInfo{
			Name:    "MockValidator",
			Version: "1.0.0",
			Command: "mock",
		},
	}

	mockLang := Language("mock")
	ci.RegisterValidator(mockLang, mockValidator)

	files := []string{"test1.mock", "test2.mock"}
	result, err := ci.Validate(mockLang, files)

	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful validation")
	}

	if result.Language != mockLang {
		t.Errorf("Expected language %s, got %s", mockLang, result.Language)
	}

	if len(result.Files) != len(files) {
		t.Errorf("Expected %d files, got %d", len(files), len(result.Files))
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(result.Errors))
	}
}

func TestValidate_NoFilesProvided(t *testing.T) {
	ci := NewCompilerInterface()

	result, err := ci.Validate(LanguageGo, []string{})

	if err == nil {
		t.Fatal("Expected error for empty files list")
	}

	if err.Error() != "no files provided for validation" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}

	// Result should be empty
	if result.Success {
		t.Error("Expected unsuccessful result for empty files")
	}
}

func TestValidate_UnsupportedLanguage(t *testing.T) {
	ci := NewCompilerInterface()

	files := []string{"test.unknown"}
	result, err := ci.Validate(Language("unknown"), files)

	if err == nil {
		t.Fatal("Expected error for unsupported language")
	}

	expectedError := "no validator registered for language: unknown"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}

	// Result should be empty
	if result.Success {
		t.Error("Expected unsuccessful result for unsupported language")
	}
}

func TestValidate_ValidatorReturnsError(t *testing.T) {
	ci := NewCompilerInterface()

	mockValidator := &MockValidator{
		executeFunc: func(files []string) (ValidationResult, error) {
			return ValidationResult{
				Success:  false,
				Language: Language("mock"),
				Files:    files,
				Duration: 50 * time.Millisecond,
				Output:   "error output",
				Errors: []ValidationError{
					{
						File:     files[0],
						Line:     10,
						Column:   5,
						Message:  "syntax error",
						Severity: SeverityError,
					},
				},
				Warnings: []ValidationError{},
			}, nil
		},
		info: ValidatorInfo{
			Name:    "MockValidator",
			Version: "1.0.0",
			Command: "mock",
		},
	}

	mockLang := Language("mock")
	ci.RegisterValidator(mockLang, mockValidator)

	files := []string{"test_error.mock"}
	result, err := ci.Validate(mockLang, files)

	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}

	if result.Success {
		t.Error("Expected unsuccessful validation")
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	if result.Errors[0].Message != "syntax error" {
		t.Errorf("Expected error message 'syntax error', got '%s'", result.Errors[0].Message)
	}
}

func TestValidate_OutputCapture(t *testing.T) {
	ci := NewCompilerInterface()

	expectedOutput := "stdout content\nstderr content"
	mockValidator := &MockValidator{
		executeFunc: func(files []string) (ValidationResult, error) {
			return ValidationResult{
				Success:  true,
				Language: Language("mock"),
				Files:    files,
				Duration: 100 * time.Millisecond,
				Output:   expectedOutput,
				Errors:   []ValidationError{},
				Warnings: []ValidationError{},
			}, nil
		},
		info: ValidatorInfo{
			Name:    "MockValidator",
			Version: "1.0.0",
			Command: "mock",
		},
	}

	mockLang := Language("mock")
	ci.RegisterValidator(mockLang, mockValidator)

	files := []string{"test.mock"}
	result, err := ci.Validate(mockLang, files)

	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}

	if result.Output != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, result.Output)
	}
}

func TestValidate_ExitCodeInterpretation(t *testing.T) {
	tests := []struct {
		name        string
		exitCode    int
		expectError bool
	}{
		{
			name:        "Exit code 0 means success",
			exitCode:    0,
			expectError: false,
		},
		{
			name:        "Exit code 1 means failure",
			exitCode:    1,
			expectError: true,
		},
		{
			name:        "Exit code 2 means failure",
			exitCode:    2,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ci := NewCompilerInterface()

			mockValidator := &MockValidator{
				executeFunc: func(files []string) (ValidationResult, error) {
					var errors []ValidationError
					if tt.exitCode != 0 {
						errors = append(errors, ValidationError{
							File:     files[0],
							Line:     1,
							Message:  "compilation error",
							Severity: SeverityError,
						})
					}

					return ValidationResult{
						Success:  tt.exitCode == 0 && len(errors) == 0,
						Language: Language("mock"),
						Files:    files,
						Duration: 100 * time.Millisecond,
						Output:   "output",
						Errors:   errors,
						Warnings: []ValidationError{},
					}, nil
				},
				info: ValidatorInfo{
					Name:    "MockValidator",
					Version: "1.0.0",
					Command: "mock",
				},
			}

			mockLang := Language("mock")
			ci.RegisterValidator(mockLang, mockValidator)

			files := []string{"test.mock"}
			result, err := ci.Validate(mockLang, files)

			if err != nil {
				t.Fatalf("Validate returned error: %v", err)
			}

			if tt.expectError && result.Success {
				t.Error("Expected validation failure for non-zero exit code")
			}

			if !tt.expectError && !result.Success {
				t.Error("Expected validation success for zero exit code")
			}

			if tt.expectError && len(result.Errors) == 0 {
				t.Error("Expected errors for non-zero exit code")
			}

			if !tt.expectError && len(result.Errors) != 0 {
				t.Error("Expected no errors for zero exit code")
			}
		})
	}
}

func TestValidateWithContext_Success(t *testing.T) {
	ci := NewCompilerInterface()

	mockValidator := &MockValidator{
		executeFunc: func(files []string) (ValidationResult, error) {
			return ValidationResult{
				Success:  true,
				Language: Language("mock"),
				Files:    files,
				Duration: 100 * time.Millisecond,
				Output:   "validation successful",
				Errors:   []ValidationError{},
				Warnings: []ValidationError{},
			}, nil
		},
		info: ValidatorInfo{
			Name:    "MockValidator",
			Version: "1.0.0",
			Command: "mock",
		},
	}

	mockLang := Language("mock")
	ci.RegisterValidator(mockLang, mockValidator)

	ctx := context.Background()
	files := []string{"test.mock"}
	result, err := ci.ValidateWithContext(ctx, mockLang, files)

	if err != nil {
		t.Fatalf("ValidateWithContext returned error: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful validation")
	}
}

func TestValidateWithContext_CancelledContext(t *testing.T) {
	ci := NewCompilerInterface()

	mockValidator := &MockValidator{
		info: ValidatorInfo{
			Name:    "MockValidator",
			Version: "1.0.0",
			Command: "mock",
		},
	}

	mockLang := Language("mock")
	ci.RegisterValidator(mockLang, mockValidator)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	files := []string{"test.mock"}
	_, err := ci.ValidateWithContext(ctx, mockLang, files)

	if err == nil {
		t.Fatal("Expected error for cancelled context")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestValidate_ConcurrentAccess(t *testing.T) {
	ci := NewCompilerInterface()

	mockValidator := &MockValidator{
		executeFunc: func(files []string) (ValidationResult, error) {
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			return ValidationResult{
				Success:  true,
				Language: Language("mock"),
				Files:    files,
				Duration: 10 * time.Millisecond,
				Output:   "success",
				Errors:   []ValidationError{},
				Warnings: []ValidationError{},
			}, nil
		},
		info: ValidatorInfo{
			Name:    "MockValidator",
			Version: "1.0.0",
			Command: "mock",
		},
	}

	mockLang := Language("mock")
	ci.RegisterValidator(mockLang, mockValidator)

	// Run multiple validations concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			files := []string{"test.mock"}
			_, err := ci.Validate(mockLang, files)
			if err != nil {
				t.Errorf("Concurrent validation %d failed: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestValidate_TimeoutHandling(t *testing.T) {
	ci := NewCompilerInterface()

	// Create a mock validator that simulates a long-running operation
	mockValidator := &MockValidator{
		executeFunc: func(files []string) (ValidationResult, error) {
			// Simulate a timeout scenario by returning an error
			return ValidationResult{
				Success:  false,
				Language: Language("mock"),
				Files:    files,
				Duration: 30 * time.Second,
				Output:   "validation timed out after 30s",
				Errors: []ValidationError{
					{
						File:     files[0],
						Line:     0,
						Message:  "validation timeout",
						Severity: SeverityError,
					},
				},
				Warnings: []ValidationError{},
			}, nil
		},
		info: ValidatorInfo{
			Name:    "MockValidator",
			Version: "1.0.0",
			Command: "mock",
		},
	}

	mockLang := Language("mock")
	ci.RegisterValidator(mockLang, mockValidator)

	files := []string{"test_timeout.mock"}
	result, err := ci.Validate(mockLang, files)

	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}

	// Verify timeout is handled gracefully
	if result.Success {
		t.Error("Expected validation failure for timeout scenario")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected timeout error to be reported")
	}

	// Verify timeout message is present
	if result.Errors[0].Message != "validation timeout" {
		t.Errorf("Expected timeout error message, got '%s'", result.Errors[0].Message)
	}
}

func TestValidate_CommandNotFound(t *testing.T) {
	ci := NewCompilerInterface()

	// Create a mock validator that simulates command not found
	mockValidator := &MockValidator{
		executeFunc: func(files []string) (ValidationResult, error) {
			// Simulate command not found scenario
			return ValidationResult{
				Success:  false,
				Language: Language("mock"),
				Files:    files,
				Duration: 0,
				Output:   "mock: command not found",
				Errors: []ValidationError{
					{
						File:     files[0],
						Line:     0,
						Message:  "validator not found: mock. Please ensure mock is installed.",
						Severity: SeverityError,
					},
				},
				Warnings: []ValidationError{},
			}, nil
		},
		info: ValidatorInfo{
			Name:    "MockValidator",
			Version: "1.0.0",
			Command: "mock",
		},
	}

	mockLang := Language("mock")
	ci.RegisterValidator(mockLang, mockValidator)

	files := []string{"test.mock"}
	result, err := ci.Validate(mockLang, files)

	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}

	// Verify command not found is handled gracefully
	if result.Success {
		t.Error("Expected validation failure for command not found")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected command not found error to be reported")
	}

	// Verify error message contains helpful information
	errorMsg := result.Errors[0].Message
	if !stringContains(errorMsg, "not found") && !stringContains(errorMsg, "installed") {
		t.Errorf("Expected helpful error message about command not found, got '%s'", errorMsg)
	}
}

// Helper function for string contains check (to avoid conflict with go_validator_prop_test.go)
func stringContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
