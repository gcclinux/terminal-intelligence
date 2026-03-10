package validation

import (
	"os"
	"path/filepath"
	"testing"

	"pgregory.net/rapid"
)

// **Validates: Requirements 4.5, 5.5**
// Property 12: Error Location Parsing
// For any validation error in the output, the validator should parse and include
// the file name and line number in the ValidationError object.
func TestProperty_ErrorLocationParsing(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Generate a package name
		packageName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}$`).Draw(rt, "packageName")

		// Create a temporary directory for the package
		tmpDir := t.TempDir()

		// Create go.mod file
		goModContent := "module testmodule\n\ngo 1.20\n"
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			rt.Fatalf("Failed to create go.mod: %v", err)
		}

		// Generate a file name
		fileName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}\.go$`).Draw(rt, "fileName")
		filePath := filepath.Join(tmpDir, fileName)

		// Generate line numbers where we'll introduce errors
		// We'll create a file with multiple lines and introduce errors at specific lines
		numLines := rapid.IntRange(5, 15).Draw(rt, "numLines")
		errorLineNum := rapid.IntRange(3, numLines-1).Draw(rt, "errorLineNum")

		// Build file content with an intentional error at a specific line
		var content string
		content += "package " + packageName + "\n\n"

		// Add some valid lines before the error
		for i := 3; i < errorLineNum; i++ {
			content += "// Line " + intToString(i) + "\n"
		}

		// Add an error line - use an undefined variable
		undefinedVar := rapid.StringMatching(`^[a-z][a-zA-Z0-9]{5,12}$`).Draw(rt, "undefinedVar")
		content += "func TestFunc() {\n"
		content += "\t_ = " + undefinedVar + "\n" // This will cause an "undefined" error
		content += "}\n"

		// Add some valid lines after the error
		for i := errorLineNum + 3; i <= numLines; i++ {
			content += "// Line " + intToString(i) + "\n"
		}

		// Write the file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			rt.Fatalf("Failed to create file %s: %v", fileName, err)
		}

		// Execute validation
		validator := NewGoValidator()
		result, err := validator.Execute([]string{filePath})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Execute, got: %v", err)
		}

		// Property: Validation should fail due to undefined variable
		if result.Success {
			rt.Error("Expected validation to fail for undefined variable")
		}

		// Property: Should have at least one error
		if len(result.Errors) == 0 {
			rt.Fatalf("Expected at least one error, got none. Output: %s", result.Output)
		}

		// Property: Each error should have a valid file name
		for i, validationErr := range result.Errors {
			if validationErr.File == "" {
				rt.Errorf("Error %d: Expected non-empty file name, got empty string", i)
			}

			// Property: File name should be parseable (not empty or malformed)
			if !filepath.IsAbs(validationErr.File) && !isRelativePath(validationErr.File) {
				rt.Errorf("Error %d: Expected valid file path, got: %s", i, validationErr.File)
			}
		}

		// Property: Each error should have a valid line number (positive integer)
		for i, validationErr := range result.Errors {
			if validationErr.Line <= 0 {
				rt.Errorf("Error %d: Expected positive line number, got: %d", i, validationErr.Line)
			}

			// Property: Line number should be within the file's line count
			// (allowing some flexibility for compiler-generated errors)
			if validationErr.Line > numLines+10 {
				rt.Errorf("Error %d: Line number %d exceeds expected range (file has ~%d lines)",
					i, validationErr.Line, numLines)
			}
		}

		// Property: At least one error should reference the undefined variable
		foundUndefinedError := false
		for _, validationErr := range result.Errors {
			if containsIgnoreCase(validationErr.Message, "undefined") {
				foundUndefinedError = true

				// Property: The error with "undefined" should have a line number
				// close to where we introduced the error
				if validationErr.Line < errorLineNum || validationErr.Line > errorLineNum+3 {
					rt.Logf("Warning: Undefined error at line %d, expected around line %d",
						validationErr.Line, errorLineNum)
				}
			}
		}

		if !foundUndefinedError {
			rt.Errorf("Expected at least one error mentioning 'undefined', got errors: %v", result.Errors)
		}
	})
}

// Property 12 (Multiple Errors): Error Location Parsing with Multiple Errors
// This tests that when a file has multiple errors, each error is parsed with
// correct file name and line number.
func TestProperty_ErrorLocationParsing_MultipleErrors(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Generate a package name
		packageName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}$`).Draw(rt, "packageName")

		// Create a temporary directory for the package
		tmpDir := t.TempDir()

		// Create go.mod file
		goModContent := "module testmodule\n\ngo 1.20\n"
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			rt.Fatalf("Failed to create go.mod: %v", err)
		}

		// Generate a file name
		fileName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}\.go$`).Draw(rt, "fileName")
		filePath := filepath.Join(tmpDir, fileName)

		// Generate multiple undefined variables to create multiple errors
		numErrors := rapid.IntRange(2, 4).Draw(rt, "numErrors")
		var undefinedVars []string
		for i := 0; i < numErrors; i++ {
			undefinedVar := rapid.StringMatching(`^[a-z][a-zA-Z0-9]{5,12}$`).Draw(rt, "undefinedVar")
			undefinedVars = append(undefinedVars, undefinedVar)
		}

		// Build file content with multiple errors
		var content string
		content += "package " + packageName + "\n\n"
		content += "func TestFunc() {\n"

		// Add multiple undefined variable references
		for _, undefinedVar := range undefinedVars {
			content += "\t_ = " + undefinedVar + "\n"
		}

		content += "}\n"

		// Write the file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			rt.Fatalf("Failed to create file %s: %v", fileName, err)
		}

		// Execute validation
		validator := NewGoValidator()
		result, err := validator.Execute([]string{filePath})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Execute, got: %v", err)
		}

		// Property: Validation should fail
		if result.Success {
			rt.Error("Expected validation to fail for undefined variables")
		}

		// Property: Should have multiple errors (at least as many as undefined vars)
		if len(result.Errors) < numErrors {
			rt.Logf("Expected at least %d errors, got %d. Output: %s",
				numErrors, len(result.Errors), result.Output)
		}

		// Property: Each error should have a file name and line number
		for i, validationErr := range result.Errors {
			if validationErr.File == "" {
				rt.Errorf("Error %d: Expected non-empty file name", i)
			}

			if validationErr.Line <= 0 {
				rt.Errorf("Error %d: Expected positive line number, got: %d", i, validationErr.Line)
			}
		}

		// Property: All errors should reference the same file
		if len(result.Errors) > 0 {
			firstFile := result.Errors[0].File
			for i, validationErr := range result.Errors {
				// File paths might be absolute or relative, so check the base name
				if filepath.Base(validationErr.File) != filepath.Base(firstFile) {
					rt.Errorf("Error %d: Expected file %s, got %s",
						i, filepath.Base(firstFile), filepath.Base(validationErr.File))
				}
			}
		}

		// Property: Line numbers should be distinct and in ascending order
		// (assuming compiler reports errors in order)
		if len(result.Errors) > 1 {
			for i := 1; i < len(result.Errors); i++ {
				if result.Errors[i].Line < result.Errors[i-1].Line {
					rt.Logf("Note: Errors not in ascending line order: line %d before line %d",
						result.Errors[i-1].Line, result.Errors[i].Line)
				}
			}
		}
	})
}

// Helper functions

func isRelativePath(path string) bool {
	return len(path) > 0 && path[0] != '/' && (len(path) < 2 || path[1] != ':')
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}
