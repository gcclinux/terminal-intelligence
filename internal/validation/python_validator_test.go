package validation

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPythonValidator(t *testing.T) {
	validator := NewPythonValidator()

	if validator == nil {
		t.Fatal("Expected validator to be created")
	}

	info := validator.GetInfo()
	if info.Name != "Python" {
		t.Errorf("Expected name 'Python', got %s", info.Name)
	}
	if info.Command != "python" {
		t.Errorf("Expected command 'python', got %s", info.Command)
	}
}

func TestPythonValidator_ParseErrors_StandardFormat(t *testing.T) {
	validator := NewPythonValidator()

	output := `  File "test.py", line 7
    print("Hello, World!"
                      ^
SyntaxError: '(' was never closed`

	errors, warnings := validator.parseErrors(output, "test.py")

	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(warnings))
	}

	err := errors[0]
	if err.File != "test.py" {
		t.Errorf("Expected file 'test.py', got %s", err.File)
	}
	if err.Line != 7 {
		t.Errorf("Expected line 7, got %d", err.Line)
	}
	if err.Column != 0 {
		t.Errorf("Expected column 0, got %d", err.Column)
	}
	if err.Severity != SeverityError {
		t.Errorf("Expected severity error, got %s", err.Severity)
	}
}

func TestPythonValidator_ParseErrors_IndentationFormat(t *testing.T) {
	validator := NewPythonValidator()

	output := "Sorry: IndentationError: unexpected indent (test.py, line 7)"

	errors, warnings := validator.parseErrors(output, "test.py")

	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(warnings))
	}

	err := errors[0]
	if err.File != "test.py" {
		t.Errorf("Expected file 'test.py', got %s", err.File)
	}
	if err.Line != 7 {
		t.Errorf("Expected line 7, got %d", err.Line)
	}
	if !strings.Contains(err.Message, "IndentationError") {
		t.Errorf("Expected message to contain 'IndentationError', got %s", err.Message)
	}
	if err.Severity != SeverityError {
		t.Errorf("Expected severity error, got %s", err.Severity)
	}
}

func TestPythonValidator_ParseErrors_EmptyOutput(t *testing.T) {
	validator := NewPythonValidator()

	errors, warnings := validator.parseErrors("", "test.py")

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(warnings))
	}
}

func TestPythonValidator_ParseErrors_NoMatches(t *testing.T) {
	validator := NewPythonValidator()

	output := "Some random output that doesn't match the pattern"
	errors, warnings := validator.parseErrors(output, "test.py")

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(warnings))
	}
}

func TestPythonValidator_Execute_NoFiles(t *testing.T) {
	validator := NewPythonValidator()

	_, err := validator.Execute([]string{})

	if err == nil {
		t.Error("Expected error when no files provided")
	}
	if !strings.Contains(err.Error(), "no files provided") {
		t.Errorf("Expected 'no files provided' error, got: %v", err)
	}
}

// TestPythonValidator_Execute_ValidCode tests validation of valid Python code
func TestPythonValidator_Execute_ValidCode(t *testing.T) {
	// Skip if python is not available
	if !isPythonAvailable() {
		t.Skip("Python interpreter not available")
	}

	validator := NewPythonValidator()

	// Use the existing valid test file
	testFile := filepath.Join("testdata", "valid", "script.py")

	// Check if file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	result, err := validator.Execute([]string{testFile})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure. Output: %s", result.Output)
	}
	if result.Language != LanguagePython {
		t.Errorf("Expected language Python, got %s", result.Language)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if result.Duration <= 0 {
		t.Error("Expected positive duration")
	}
}

// TestPythonValidator_Execute_SyntaxError tests validation of Python code with syntax errors
func TestPythonValidator_Execute_SyntaxError(t *testing.T) {
	// Skip if python is not available
	if !isPythonAvailable() {
		t.Skip("Python interpreter not available")
	}

	validator := NewPythonValidator()

	// Use the existing syntax error test file
	testFile := filepath.Join("testdata", "invalid", "syntax_error.py")

	// Check if file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	result, err := validator.Execute([]string{testFile})

	if err != nil {
		t.Fatalf("Expected no error from Execute, got: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for invalid code")
	}
	if result.Language != LanguagePython {
		t.Errorf("Expected language Python, got %s", result.Language)
	}
	if len(result.Errors) == 0 {
		t.Error("Expected at least one error")
	}
	if result.Output == "" {
		t.Error("Expected non-empty output")
	}

	// Verify error details
	if len(result.Errors) > 0 {
		err := result.Errors[0]
		if err.Line <= 0 {
			t.Errorf("Expected positive line number, got %d", err.Line)
		}
	}
}

// TestPythonValidator_Execute_IndentationError tests validation of Python code with indentation errors
func TestPythonValidator_Execute_IndentationError(t *testing.T) {
	// Skip if python is not available
	if !isPythonAvailable() {
		t.Skip("Python interpreter not available")
	}

	validator := NewPythonValidator()

	// Use the existing indentation error test file
	testFile := filepath.Join("testdata", "invalid", "indentation_error.py")

	// Check if file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	result, err := validator.Execute([]string{testFile})

	if err != nil {
		t.Fatalf("Expected no error from Execute, got: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for code with indentation errors")
	}
	if result.Language != LanguagePython {
		t.Errorf("Expected language Python, got %s", result.Language)
	}
	if len(result.Errors) == 0 {
		t.Error("Expected at least one error")
	}

	// Verify it's an indentation error
	if len(result.Errors) > 0 {
		err := result.Errors[0]
		if !strings.Contains(strings.ToLower(err.Message), "indent") {
			t.Errorf("Expected indentation error message, got: %s", err.Message)
		}
	}
}

// TestPythonValidator_Execute_MultipleFiles tests validation of multiple Python files independently
func TestPythonValidator_Execute_MultipleFiles(t *testing.T) {
	// Skip if python is not available
	if !isPythonAvailable() {
		t.Skip("Python interpreter not available")
	}

	validator := NewPythonValidator()

	// Use existing test files
	validFile := filepath.Join("testdata", "valid", "script.py")
	invalidFile := filepath.Join("testdata", "invalid", "syntax_error.py")

	// Check if files exist
	if _, err := os.Stat(validFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", validFile)
	}
	if _, err := os.Stat(invalidFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", invalidFile)
	}

	result, err := validator.Execute([]string{validFile, invalidFile})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should fail because one file has errors
	if result.Success {
		t.Error("Expected failure when one file has errors")
	}

	if len(result.Files) != 2 {
		t.Errorf("Expected 2 files in result, got %d", len(result.Files))
	}

	// Should have errors from the invalid file
	if len(result.Errors) == 0 {
		t.Error("Expected at least one error from invalid file")
	}
}

// TestPythonValidator_Execute_IndependentValidation tests that files are validated independently
func TestPythonValidator_Execute_IndependentValidation(t *testing.T) {
	// Skip if python is not available
	if !isPythonAvailable() {
		t.Skip("Python interpreter not available")
	}

	// Create temporary files
	tmpDir := t.TempDir()

	// Create two valid Python files
	file1 := filepath.Join(tmpDir, "file1.py")
	file2 := filepath.Join(tmpDir, "file2.py")

	code1 := `def func1():
    return "file1"
`
	code2 := `def func2():
    return "file2"
`

	if err := os.WriteFile(file1, []byte(code1), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(code2), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	validator := NewPythonValidator()
	result, err := validator.Execute([]string{file1, file2})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success for valid independent files. Output: %s", result.Output)
	}
	if len(result.Files) != 2 {
		t.Errorf("Expected 2 files in result, got %d", len(result.Files))
	}
}

// isPythonAvailable checks if the Python interpreter is available
func isPythonAvailable() bool {
	cmd := exec.Command("python", "--version")
	err := cmd.Run()
	return err == nil
}
