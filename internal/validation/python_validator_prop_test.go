package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// **Validates: Requirements 5.2**
// Property 13: Python Syntax Error Detection
// For any Python file with syntax errors, the Python validator should detect
// and report those errors in the Validation_Result.
func TestProperty_PythonSyntaxErrorDetection(t *testing.T) {
	// Skip if python is not available
	if !isPythonAvailable() {
		t.Skip("Python interpreter not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Generate a Python file name
		fileName := rapid.StringMatching(`^[a-z][a-z0-9_]{2,10}\.py$`).Draw(rt, "fileName")
		filePath := filepath.Join(tmpDir, fileName)

		// Generate Python code with intentional syntax errors
		// We'll use a variety of common syntax error patterns
		errorType := rapid.IntRange(0, 4).Draw(rt, "errorType")

		var content string
		var expectedErrorKeywords []string

		switch errorType {
		case 0:
			// Missing closing parenthesis
			funcName := rapid.StringMatching(`^[a-z][a-z0-9_]{2,8}$`).Draw(rt, "funcName")
			content = "def " + funcName + "():\n"
			content += "    print(\"Hello\"\n"
			content += "    return None\n"
			expectedErrorKeywords = []string{"syntax", "paren", "close"}

		case 1:
			// Missing colon after if statement
			varName := rapid.StringMatching(`^[a-z][a-z0-9_]{2,8}$`).Draw(rt, "varName")
			content = "def test():\n"
			content += "    " + varName + " = True\n"
			content += "    if " + varName + "\n"
			content += "        print(\"test\")\n"
			expectedErrorKeywords = []string{"syntax", "invalid"}

		case 2:
			// Invalid indentation
			content = "def bad_indent():\n"
			content += "    print(\"line1\")\n"
			content += "      print(\"line2\")\n"
			expectedErrorKeywords = []string{"indent"}

		case 3:
			// Unclosed string literal
			content = "def test():\n"
			content += "    msg = \"unclosed string\n"
			content += "    return msg\n"
			expectedErrorKeywords = []string{"syntax", "eol", "string"}

		case 4:
			// Invalid assignment target
			content = "def test():\n"
			content += "    \"string\" = 5\n"
			content += "    return None\n"
			expectedErrorKeywords = []string{"syntax", "assign"}
		}

		// Write the file with syntax errors
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			rt.Fatalf("Failed to create file %s: %v", fileName, err)
		}

		// Execute validation
		validator := NewPythonValidator()
		result, err := validator.Execute([]string{filePath})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Execute, got: %v", err)
		}

		// Property: Validation should fail for files with syntax errors
		if result.Success {
			rt.Errorf("Expected validation to fail for file with syntax errors. Output: %s", result.Output)
		}

		// Property: Should have at least one error
		if len(result.Errors) == 0 {
			rt.Errorf("Expected at least one error for syntax error. Output: %s", result.Output)
		}

		// Property: Language should be Python
		if result.Language != LanguagePython {
			rt.Errorf("Expected language Python, got %s", result.Language)
		}

		// Property: Error should have a valid line number (positive)
		if len(result.Errors) > 0 {
			for _, validationErr := range result.Errors {
				if validationErr.Line <= 0 {
					rt.Errorf("Expected positive line number, got %d", validationErr.Line)
				}
			}
		}

		// Property: Error message should contain relevant keywords
		if len(result.Errors) > 0 {
			foundRelevantError := false
			for _, validationErr := range result.Errors {
				msgLower := strings.ToLower(validationErr.Message)
				for _, keyword := range expectedErrorKeywords {
					if strings.Contains(msgLower, keyword) {
						foundRelevantError = true
						break
					}
				}
				if foundRelevantError {
					break
				}
			}
			// Note: We don't strictly require keyword matching as error messages
			// can vary across Python versions, but we log if not found
			if !foundRelevantError {
				rt.Logf("Warning: Error message doesn't contain expected keywords %v. Got: %v",
					expectedErrorKeywords, result.Errors[0].Message)
			}
		}

		// Property: Output should not be empty (should contain error details)
		if result.Output == "" {
			rt.Error("Expected non-empty output for validation with errors")
		}

		// Property: Duration should be positive
		if result.Duration <= 0 {
			rt.Errorf("Expected positive duration, got %v", result.Duration)
		}

		// Property: Files list should contain the validated file
		if len(result.Files) != 1 {
			rt.Errorf("Expected 1 file in result, got %d", len(result.Files))
		}
		if len(result.Files) > 0 && result.Files[0] != filePath {
			rt.Errorf("Expected file path %s in result, got %s", filePath, result.Files[0])
		}
	})
}

// Property 13 (Edge Case): Python Syntax Error Detection with Multiple Errors
// This tests that the validator can detect and report multiple syntax errors
// in a single Python file.
func TestProperty_PythonSyntaxErrorDetection_MultipleErrors(t *testing.T) {
	// Skip if python is not available
	if !isPythonAvailable() {
		t.Skip("Python interpreter not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Generate a Python file name
		fileName := rapid.StringMatching(`^[a-z][a-z0-9_]{2,10}\.py$`).Draw(rt, "fileName")
		filePath := filepath.Join(tmpDir, fileName)

		// Generate Python code with multiple syntax errors
		// Note: Python typically stops at the first syntax error, but we test
		// that the validator handles this correctly
		content := "def func1():\n"
		content += "    print(\"missing paren\"\n" // First error
		content += "\n"
		content += "def func2():\n"
		content += "    if True\n" // Second error (may not be detected if first stops parsing)
		content += "        print(\"test\")\n"

		// Write the file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			rt.Fatalf("Failed to create file %s: %v", fileName, err)
		}

		// Execute validation
		validator := NewPythonValidator()
		result, err := validator.Execute([]string{filePath})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Execute, got: %v", err)
		}

		// Property: Validation should fail
		if result.Success {
			rt.Error("Expected validation to fail for file with syntax errors")
		}

		// Property: Should have at least one error (Python may stop at first error)
		if len(result.Errors) == 0 {
			rt.Errorf("Expected at least one error. Output: %s", result.Output)
		}

		// Property: All errors should have valid line numbers
		for _, validationErr := range result.Errors {
			if validationErr.Line <= 0 {
				rt.Errorf("Expected positive line number, got %d", validationErr.Line)
			}
		}
	})
}

// Property 13 (Valid Code Case): Python Syntax Error Detection with Valid Code
// This tests that the validator correctly reports success for valid Python code
// (no false positives).
func TestProperty_PythonSyntaxErrorDetection_ValidCode(t *testing.T) {
	// Skip if python is not available
	if !isPythonAvailable() {
		t.Skip("Python interpreter not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Generate a Python file name
		fileName := rapid.StringMatching(`^[a-z][a-z0-9_]{2,10}\.py$`).Draw(rt, "fileName")
		filePath := filepath.Join(tmpDir, fileName)

		// Generate valid Python code
		funcName := rapid.StringMatching(`^[a-z][a-z0-9_]{2,8}$`).Draw(rt, "funcName")
		varName := rapid.StringMatching(`^[a-z][a-z0-9_]{2,8}$`).Draw(rt, "varName")

		// Generate a random integer value
		intValue := rapid.IntRange(0, 1000).Draw(rt, "intValue")

		// Generate a random string value
		strValue := rapid.StringMatching(`^[a-zA-Z0-9 ]{3,20}$`).Draw(rt, "strValue")

		content := "def " + funcName + "():\n"
		content += "    " + varName + " = " + intToString(intValue) + "\n"
		content += "    msg = \"" + strValue + "\"\n"
		content += "    return " + varName + "\n"
		content += "\n"
		content += "if __name__ == \"__main__\":\n"
		content += "    result = " + funcName + "()\n"
		content += "    print(result)\n"

		// Write the file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			rt.Fatalf("Failed to create file %s: %v", fileName, err)
		}

		// Execute validation
		validator := NewPythonValidator()
		result, err := validator.Execute([]string{filePath})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Execute, got: %v", err)
		}

		// Property: Validation should succeed for valid code
		if !result.Success {
			rt.Errorf("Expected validation to succeed for valid code. Errors: %v, Output: %s",
				result.Errors, result.Output)
		}

		// Property: Should have no errors
		if len(result.Errors) > 0 {
			rt.Errorf("Expected no errors for valid code, got %d: %v", len(result.Errors), result.Errors)
		}

		// Property: Language should be Python
		if result.Language != LanguagePython {
			rt.Errorf("Expected language Python, got %s", result.Language)
		}

		// Property: Duration should be positive
		if result.Duration <= 0 {
			rt.Errorf("Expected positive duration, got %v", result.Duration)
		}
	})
}

// **Validates: Requirements 8.4**
// Property 14: Python Independent Validation
// For any set of Python files, the validator should validate each file independently
// rather than as a group. This means:
// 1. Each file is validated separately (not as a package)
// 2. Errors in one file don't affect validation of other files
// 3. Files can have different/conflicting definitions without causing errors
func TestProperty_PythonIndependentValidation(t *testing.T) {
	// Skip if python is not available
	if !isPythonAvailable() {
		t.Skip("Python interpreter not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Generate number of files (2-5 files to test independent validation)
		numFiles := rapid.IntRange(2, 5).Draw(rt, "numFiles")

		var filePaths []string
		var functionNames []string

		// Generate function names - intentionally use the SAME function name
		// across multiple files to prove they're validated independently
		sharedFuncName := rapid.StringMatching(`^[a-z][a-z0-9_]{3,10}`).Draw(rt, "sharedFuncName")

		// Create multiple Python files with the same function name
		// If they were validated together (like Go packages), this would cause conflicts
		// But since Python validates independently, each file should validate successfully
		for i := 0; i < numFiles; i++ {
			fileName := rapid.StringMatching(`^[a-z][a-z0-9_]{2,10}\.py`).Draw(rt, "fileName")
			filePath := filepath.Join(tmpDir, fileName)

			// Generate a random integer value
			intValue := rapid.IntRange(0, 1000).Draw(rt, "intValue")

			// Each file defines the SAME function name with different implementations
			// This proves independent validation - no conflict should occur
			content := "def " + sharedFuncName + "():\n"
			content += "    value = " + intToString(intValue) + "\n"
			content += "    return value\n"
			content += "\n"

			// Optionally add a class with the same name across files
			shouldAddClass := rapid.Bool().Draw(rt, "shouldAddClass")
			if shouldAddClass {
				className := rapid.StringMatching(`^[A-Z][a-zA-Z0-9]{3,10}`).Draw(rt, "className")
				content += "class " + className + ":\n"
				content += "    def __init__(self):\n"
				content += "        self.value = " + intToString(intValue) + "\n"
				content += "\n"
			}

			// Write the file
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				rt.Fatalf("Failed to create file %s: %v", fileName, err)
			}

			filePaths = append(filePaths, filePath)
			functionNames = append(functionNames, sharedFuncName)
		}

		// Execute validation with all files
		validator := NewPythonValidator()
		result, err := validator.Execute(filePaths)

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Execute, got: %v", err)
		}

		// Property: All files should validate successfully despite having identical
		// function/class names, proving they are validated independently
		if !result.Success {
			rt.Errorf("Expected successful validation of independent files. Errors: %v, Output: %s",
				result.Errors, result.Output)
		}

		// Property: Should have no errors (duplicate names are OK for independent validation)
		if len(result.Errors) > 0 {
			rt.Errorf("Expected no errors for independently validated files, got %d: %v",
				len(result.Errors), result.Errors)
		}

		// Property: The result should include all files
		if len(result.Files) != numFiles {
			rt.Errorf("Expected %d files in result, got %d", numFiles, len(result.Files))
		}

		// Property: Language should be Python
		if result.Language != LanguagePython {
			rt.Errorf("Expected language Python, got %s", result.Language)
		}

		// Property: Duration should be positive
		if result.Duration <= 0 {
			rt.Errorf("Expected positive duration, got %v", result.Duration)
		}
	})
}

// Property 14 (Error Isolation): Python Independent Validation with Errors
// This tests that when one file has errors, other files are still validated
// independently and their validation results are not affected.
func TestProperty_PythonIndependentValidation_ErrorIsolation(t *testing.T) {
	// Skip if python is not available
	if !isPythonAvailable() {
		t.Skip("Python interpreter not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Generate number of files (3-5 files)
		numFiles := rapid.IntRange(3, 5).Draw(rt, "numFiles")

		// Randomly select which file will have an error (not the first or last)
		errorFileIdx := rapid.IntRange(1, numFiles-2).Draw(rt, "errorFileIdx")

		var filePaths []string

		// Create multiple Python files
		for i := 0; i < numFiles; i++ {
			fileName := rapid.StringMatching(`^[a-z][a-z0-9_]{2,10}\.py`).Draw(rt, "fileName")
			filePath := filepath.Join(tmpDir, fileName)

			var content string

			if i == errorFileIdx {
				// This file has a syntax error
				funcName := rapid.StringMatching(`^[a-z][a-z0-9_]{3,10}`).Draw(rt, "funcName")
				content = "def " + funcName + "():\n"
				content += "    print(\"missing paren\"\n" // Syntax error: missing closing paren
				content += "    return None\n"
			} else {
				// This file is valid
				funcName := rapid.StringMatching(`^[a-z][a-z0-9_]{3,10}`).Draw(rt, "funcName")
				intValue := rapid.IntRange(0, 1000).Draw(rt, "intValue")
				content = "def " + funcName + "():\n"
				content += "    value = " + intToString(intValue) + "\n"
				content += "    return value\n"
			}

			// Write the file
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				rt.Fatalf("Failed to create file %s: %v", fileName, err)
			}

			filePaths = append(filePaths, filePath)
		}

		// Execute validation with all files
		validator := NewPythonValidator()
		result, err := validator.Execute(filePaths)

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Execute, got: %v", err)
		}

		// Property: Validation should fail overall (because one file has errors)
		if result.Success {
			rt.Error("Expected validation to fail when one file has syntax errors")
		}

		// Property: Should have at least one error (from the error file)
		if len(result.Errors) == 0 {
			rt.Errorf("Expected at least one error. Output: %s", result.Output)
		}

		// Property: The error should be from the file with the syntax error
		foundErrorFromCorrectFile := false
		for _, validationErr := range result.Errors {
			// Check if the error is from the file we intentionally broke
			if validationErr.File == filePaths[errorFileIdx] ||
				filepath.Base(validationErr.File) == filepath.Base(filePaths[errorFileIdx]) {
				foundErrorFromCorrectFile = true
				break
			}
		}

		if !foundErrorFromCorrectFile {
			rt.Errorf("Expected error from file %s, but got errors from: %v",
				filePaths[errorFileIdx], result.Errors)
		}

		// Property: All files should still be in the result (all were validated)
		if len(result.Files) != numFiles {
			rt.Errorf("Expected %d files in result, got %d", numFiles, len(result.Files))
		}

		// Property: Language should be Python
		if result.Language != LanguagePython {
			rt.Errorf("Expected language Python, got %s", result.Language)
		}
	})
}

// Property 14 (No Cross-File Dependencies): Python Independent Validation
// This tests that Python files cannot reference functions/classes from other files
// in the validation set, proving they are truly validated independently.
func TestProperty_PythonIndependentValidation_NoCrossFileReferences(t *testing.T) {
	// Skip if python is not available
	if !isPythonAvailable() {
		t.Skip("Python interpreter not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Create file1.py with a function definition
		funcName := rapid.StringMatching(`^[a-z][a-z0-9_]{3,10}`).Draw(rt, "funcName")
		file1Content := "def " + funcName + "():\n"
		file1Content += "    return 42\n"

		file1Path := filepath.Join(tmpDir, "file1.py")
		if err := os.WriteFile(file1Path, []byte(file1Content), 0644); err != nil {
			rt.Fatalf("Failed to create file1.py: %v", err)
		}

		// Create file2.py that tries to call the function from file1
		// This should validate successfully because py_compile only checks syntax,
		// not runtime imports/references
		file2Content := "def caller():\n"
		file2Content += "    result = " + funcName + "()\n" // Reference to file1's function
		file2Content += "    return result\n"

		file2Path := filepath.Join(tmpDir, "file2.py")
		if err := os.WriteFile(file2Path, []byte(file2Content), 0644); err != nil {
			rt.Fatalf("Failed to create file2.py: %v", err)
		}

		// Execute validation with both files
		validator := NewPythonValidator()
		result, err := validator.Execute([]string{file1Path, file2Path})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Execute, got: %v", err)
		}

		// Property: Both files should validate successfully
		// Python's py_compile only checks syntax, not semantic references
		// This proves files are validated independently - no cross-file resolution
		if !result.Success {
			rt.Errorf("Expected validation to succeed (py_compile only checks syntax). Errors: %v, Output: %s",
				result.Errors, result.Output)
		}

		// Property: Should have no errors
		if len(result.Errors) > 0 {
			rt.Errorf("Expected no errors for syntax-valid files, got %d: %v",
				len(result.Errors), result.Errors)
		}

		// Property: Both files should be in the result
		if len(result.Files) != 2 {
			rt.Errorf("Expected 2 files in result, got %d", len(result.Files))
		}

		// Property: Language should be Python
		if result.Language != LanguagePython {
			rt.Errorf("Expected language Python, got %s", result.Language)
		}
	})
}
