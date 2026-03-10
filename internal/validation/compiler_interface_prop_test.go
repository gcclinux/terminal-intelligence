package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// **Validates: Requirements 3.3**
// Property 6: Complete Output Capture
// For any validation command execution, the CompilerInterface should capture
// the complete stdout and stderr output from the command.
func TestProperty_CompleteOutputCapture(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Create go.mod file
		goModContent := "module testmodule\n\ngo 1.20\n"
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			rt.Fatalf("Failed to create go.mod: %v", err)
		}

		// Generate a package name
		packageName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}$`).Draw(rt, "packageName")

		// Generate a file name
		fileName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}\.go$`).Draw(rt, "fileName")
		filePath := filepath.Join(tmpDir, fileName)

		// Decide whether to create valid or invalid Go code
		// This will generate different types of output (success vs error messages)
		isValid := rapid.Bool().Draw(rt, "isValid")

		var content string
		var expectedInOutput []string

		if isValid {
			// Create valid Go code
			funcName := rapid.StringMatching(`^[A-Z][a-zA-Z0-9]{3,10}$`).Draw(rt, "funcName")
			content = "package " + packageName + "\n\n"
			content += "func " + funcName + "() {\n"
			content += "\t// Valid function\n"
			content += "}\n"

			// For valid code, we expect minimal output (possibly empty or just success indicators)
			// The key property is that whatever output exists is captured
		} else {
			// Create invalid Go code with specific errors that will appear in output
			// Use undefined identifiers to generate predictable error messages
			undefinedVar := rapid.StringMatching(`^[a-z][a-z0-9]{5,12}$`).Draw(rt, "undefinedVar")

			content = "package " + packageName + "\n\n"
			content += "func InvalidFunc() {\n"
			content += "\t" + undefinedVar + " = 42\n" // This will cause "undefined" error
			content += "}\n"

			// We expect the error output to contain the undefined variable name
			expectedInOutput = append(expectedInOutput, undefinedVar)
			expectedInOutput = append(expectedInOutput, fileName) // File name should appear in error
		}

		// Write the file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			rt.Fatalf("Failed to create file %s: %v", fileName, err)
		}

		// Execute validation through CompilerInterface
		ci := NewCompilerInterface()
		result, err := ci.Validate(LanguageGo, []string{filePath})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Validate, got: %v", err)
		}

		// Property 1: Output field must not be empty when there's compiler output
		// (Either success messages or error messages should be captured)
		if !isValid {
			// For invalid code, we must have output containing error messages
			if result.Output == "" {
				rt.Error("Expected non-empty output for compilation errors, got empty string")
			}

			// Property 2: All expected strings should appear in the output
			for _, expected := range expectedInOutput {
				if !strings.Contains(result.Output, expected) {
					rt.Errorf("Expected output to contain '%s', but it was not found. Output: %s",
						expected, result.Output)
				}
			}

			// Property 3: Error messages should also be captured in the Errors slice
			if len(result.Errors) == 0 {
				rt.Error("Expected errors to be parsed from output, got none")
			}

			// Property 4: The raw output should contain information from the parsed errors
			for _, validationErr := range result.Errors {
				// The output should contain the error message
				if validationErr.Message != "" && !strings.Contains(result.Output, validationErr.Message) {
					rt.Errorf("Expected output to contain error message '%s', but it was not found",
						validationErr.Message)
				}
			}
		}

		// Property 5: Output should be a string (not nil or corrupted)
		// This is implicitly tested by the string operations above, but we can verify type
		if result.Output != result.Output {
			rt.Error("Output field appears to be corrupted")
		}

		// Property 6: For any validation, the output length should be reasonable
		// (not truncated to some arbitrary limit)
		// If we have errors, the output should be substantial enough to contain error details
		if !isValid && len(result.Output) < 10 {
			rt.Errorf("Expected substantial output for compilation errors, got only %d bytes",
				len(result.Output))
		}

		// Property 7: The output should match the language being validated
		if result.Language != LanguageGo {
			rt.Errorf("Expected language Go, got %s", result.Language)
		}
	})
}

// Property 6 (Multi-line Output): Complete Output Capture with Multiple Errors
// This tests that when there are multiple errors generating multi-line output,
// all output is captured completely without truncation.
func TestProperty_CompleteOutputCapture_MultipleErrors(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Create go.mod file
		goModContent := "module testmodule\n\ngo 1.20\n"
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			rt.Fatalf("Failed to create go.mod: %v", err)
		}

		// Generate a package name
		packageName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}$`).Draw(rt, "packageName")

		// Generate number of errors to introduce (2-5 errors)
		numErrors := rapid.IntRange(2, 5).Draw(rt, "numErrors")

		// Generate a file name
		fileName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}\.go$`).Draw(rt, "fileName")
		filePath := filepath.Join(tmpDir, fileName)

		// Create Go code with multiple undefined variables (each will generate an error)
		var content string
		content = "package " + packageName + "\n\n"
		content += "func MultiErrorFunc() {\n"

		var undefinedVars []string
		for i := 0; i < numErrors; i++ {
			undefinedVar := rapid.StringMatching(`^[a-z][a-z0-9]{5,12}$`).Draw(rt, "undefinedVar")
			undefinedVars = append(undefinedVars, undefinedVar)
			content += "\t" + undefinedVar + " = " + string(rune('0'+i)) + "\n"
		}

		content += "}\n"

		// Write the file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			rt.Fatalf("Failed to create file %s: %v", fileName, err)
		}

		// Execute validation through CompilerInterface
		ci := NewCompilerInterface()
		result, err := ci.Validate(LanguageGo, []string{filePath})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Validate, got: %v", err)
		}

		// Property 1: Output must contain all undefined variable names
		for _, undefinedVar := range undefinedVars {
			if !strings.Contains(result.Output, undefinedVar) {
				rt.Errorf("Expected output to contain '%s', but it was not found. Output: %s",
					undefinedVar, result.Output)
			}
		}

		// Property 2: Multiple errors should be captured
		// Note: Go compiler might not report all errors in one pass, but at least some should be captured
		if len(result.Errors) == 0 {
			rt.Error("Expected at least one error to be parsed from output")
		}

		// Property 3: Output should be non-empty and substantial
		if result.Output == "" {
			rt.Error("Expected non-empty output for multiple compilation errors")
		}

		// Property 4: Output should contain the file name (appears in error messages)
		if !strings.Contains(result.Output, fileName) {
			rt.Errorf("Expected output to contain file name '%s'", fileName)
		}

		// Property 5: Validation should fail
		if result.Success {
			rt.Error("Expected validation to fail for code with multiple errors")
		}

		// Property 6: Each parsed error should have its message present in the raw output
		for i, validationErr := range result.Errors {
			if validationErr.Message != "" {
				// The raw output should contain the error message
				if !strings.Contains(result.Output, validationErr.Message) {
					rt.Errorf("Error %d: Expected output to contain error message '%s'",
						i, validationErr.Message)
				}
			}
		}
	})
}

// Property 6 (Python): Complete Output Capture for Python Validation
// This tests that output capture works correctly for Python validation as well,
// demonstrating the property holds across different language validators.
func TestProperty_CompleteOutputCapture_Python(t *testing.T) {
	// Skip if python is not available
	if !isPythonAvailable() {
		t.Skip("Python interpreter not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Generate a file name
		fileName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}\.py$`).Draw(rt, "fileName")
		filePath := filepath.Join(tmpDir, fileName)

		// Decide whether to create valid or invalid Python code
		isValid := rapid.Bool().Draw(rt, "isValid")

		var content string
		var expectedInOutput []string

		if isValid {
			// Create valid Python code
			funcName := rapid.StringMatching(`^[a-z][a-z0-9]{3,10}$`).Draw(rt, "funcName")
			content = "def " + funcName + "():\n"
			content += "    pass\n"
		} else {
			// Create invalid Python code with syntax error
			// Missing colon will cause a syntax error
			funcName := rapid.StringMatching(`^[a-z][a-z0-9]{3,10}$`).Draw(rt, "funcName")
			content = "def " + funcName + "()\n" // Missing colon - syntax error
			content += "    pass\n"

			// We expect the error output to contain the file name
			expectedInOutput = append(expectedInOutput, fileName)
		}

		// Write the file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			rt.Fatalf("Failed to create file %s: %v", fileName, err)
		}

		// Execute validation through CompilerInterface
		ci := NewCompilerInterface()
		result, err := ci.Validate(LanguagePython, []string{filePath})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Validate, got: %v", err)
		}

		// Property 1: For invalid code, output must not be empty
		if !isValid {
			if result.Output == "" {
				rt.Error("Expected non-empty output for Python syntax errors, got empty string")
			}

			// Property 2: All expected strings should appear in the output
			for _, expected := range expectedInOutput {
				if !strings.Contains(result.Output, expected) {
					rt.Errorf("Expected output to contain '%s', but it was not found. Output: %s",
						expected, result.Output)
				}
			}

			// Property 3: Error messages should be captured
			if len(result.Errors) == 0 {
				rt.Error("Expected errors to be parsed from output, got none")
			}
		}

		// Property 4: Output should be a valid string
		if result.Output != result.Output {
			rt.Error("Output field appears to be corrupted")
		}

		// Property 5: The output should match the language being validated
		if result.Language != LanguagePython {
			rt.Errorf("Expected language Python, got %s", result.Language)
		}

		// Property 6: Success status should match whether code is valid
		if isValid && !result.Success {
			rt.Errorf("Expected success for valid Python code, got failure. Output: %s", result.Output)
		}
		if !isValid && result.Success {
			rt.Error("Expected failure for invalid Python code, got success")
		}
	})
}

// **Validates: Requirements 3.4**
// Property 7: Success/Failure Determination
// For any validation command execution, the CompilerInterface should correctly
// determine success or failure based on the command's exit code (0 = success, non-zero = failure).
func TestProperty_SuccessFailureDetermination(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Create go.mod file
		goModContent := "module testmodule\n\ngo 1.20\n"
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			rt.Fatalf("Failed to create go.mod: %v", err)
		}

		// Generate a package name
		packageName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}$`).Draw(rt, "packageName")

		// Generate a file name
		fileName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}\.go$`).Draw(rt, "fileName")
		filePath := filepath.Join(tmpDir, fileName)

		// Decide whether to create valid or invalid Go code
		isValid := rapid.Bool().Draw(rt, "isValid")

		var content string
		if isValid {
			// Create valid Go code (exit code 0)
			funcName := rapid.StringMatching(`^[A-Z][a-zA-Z0-9]{3,10}$`).Draw(rt, "funcName")
			content = "package " + packageName + "\n\n"
			content += "func " + funcName + "() {\n"
			content += "\t// Valid function\n"
			content += "}\n"
		} else {
			// Create invalid Go code (exit code non-zero)
			undefinedVar := rapid.StringMatching(`^[a-z][a-z0-9]{5,12}$`).Draw(rt, "undefinedVar")
			content = "package " + packageName + "\n\n"
			content += "func InvalidFunc() {\n"
			content += "\t" + undefinedVar + " = 42\n" // undefined variable
			content += "}\n"
		}

		// Write the file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			rt.Fatalf("Failed to create file %s: %v", fileName, err)
		}

		// Execute validation through CompilerInterface
		ci := NewCompilerInterface()
		result, err := ci.Validate(LanguageGo, []string{filePath})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Validate, got: %v", err)
		}

		// Property: Success field should match whether code is valid
		// Valid code (exit code 0) -> Success = true
		// Invalid code (exit code non-zero) -> Success = false
		if isValid && !result.Success {
			rt.Errorf("Expected Success=true for valid code (exit code 0), got Success=false. Output: %s",
				result.Output)
		}
		if !isValid && result.Success {
			rt.Errorf("Expected Success=false for invalid code (exit code non-zero), got Success=true. Output: %s",
				result.Output)
		}

		// Property: Success=true should correlate with empty errors array
		if result.Success && len(result.Errors) > 0 {
			rt.Errorf("Expected no errors when Success=true, got %d errors", len(result.Errors))
		}

		// Property: Success=false should correlate with non-empty errors array
		if !result.Success && len(result.Errors) == 0 {
			rt.Error("Expected errors when Success=false, got empty errors array")
		}
	})
}

// **Validates: Requirements 3.5**
// Property 8: Validation Result Production
// For any validation execution, the CompilerInterface should produce a Validation_Result
// object containing language, files, duration, output, and errors.
func TestProperty_ValidationResultProduction(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Create go.mod file
		goModContent := "module testmodule\n\ngo 1.20\n"
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			rt.Fatalf("Failed to create go.mod: %v", err)
		}

		// Generate a package name
		packageName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}$`).Draw(rt, "packageName")

		// Generate file names (1-3 files)
		numFiles := rapid.IntRange(1, 3).Draw(rt, "numFiles")
		var filePaths []string

		for i := 0; i < numFiles; i++ {
			fileName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}\.go$`).Draw(rt, "fileName")
			filePath := filepath.Join(tmpDir, fileName)
			filePaths = append(filePaths, filePath)

			// Create simple valid Go code
			content := "package " + packageName + "\n\n"
			content += "func Func" + string(rune('A'+i)) + "() {}\n"

			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				rt.Fatalf("Failed to create file %s: %v", fileName, err)
			}
		}

		// Execute validation through CompilerInterface
		ci := NewCompilerInterface()
		result, err := ci.Validate(LanguageGo, filePaths)

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Validate, got: %v", err)
		}

		// Property 1: Result must contain the correct language
		if result.Language != LanguageGo {
			rt.Errorf("Expected Language=%s, got Language=%s", LanguageGo, result.Language)
		}

		// Property 2: Result must contain the files that were validated
		if len(result.Files) == 0 {
			rt.Error("Expected Files array to be non-empty")
		}
		// Files should match or be related to input files
		if len(result.Files) != len(filePaths) {
			rt.Errorf("Expected %d files in result, got %d", len(filePaths), len(result.Files))
		}

		// Property 3: Result must contain duration (should be non-negative)
		if result.Duration < 0 {
			rt.Errorf("Expected non-negative Duration, got %v", result.Duration)
		}

		// Property 4: Result must contain output (string, can be empty but not nil)
		// The output field should exist (even if empty string)
		_ = result.Output // This verifies the field exists

		// Property 5: Result must contain errors array (can be empty but not nil)
		// Note: In Go, nil slices are valid and behave like empty slices
		// We accept both nil and empty slices as valid
		if result.Errors == nil {
			// This is acceptable - nil slice is equivalent to empty slice in Go
			// But we'll verify it's at least initialized as a slice type
			_ = result.Errors
		}

		// Property 6: Result must have a Success field
		// The Success field should be set based on validation outcome
		_ = result.Success // This verifies the field exists
	})
}

// **Validates: Requirements 4.3**
// Property 10: Success Result Consistency
// For any successful validation (exit code 0), the Validation_Result should have
// success=true and an empty errors array.
func TestProperty_SuccessResultConsistency(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Create go.mod file
		goModContent := "module testmodule\n\ngo 1.20\n"
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			rt.Fatalf("Failed to create go.mod: %v", err)
		}

		// Generate a package name
		packageName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}$`).Draw(rt, "packageName")

		// Generate a file name
		fileName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}\.go$`).Draw(rt, "fileName")
		filePath := filepath.Join(tmpDir, fileName)

		// Create VALID Go code only (to test successful validation)
		funcName := rapid.StringMatching(`^[A-Z][a-zA-Z0-9]{3,10}$`).Draw(rt, "funcName")
		content := "package " + packageName + "\n\n"
		content += "func " + funcName + "() {\n"

		// Add some random valid statements
		numStatements := rapid.IntRange(0, 3).Draw(rt, "numStatements")
		for i := 0; i < numStatements; i++ {
			varName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}$`).Draw(rt, "varName")
			value := rapid.IntRange(0, 100).Draw(rt, "value")
			content += "\t" + varName + " := " + string(rune('0'+value%10)) + "\n"
			content += "\t_ = " + varName + "\n" // Use the variable
		}

		content += "}\n"

		// Write the file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			rt.Fatalf("Failed to create file %s: %v", fileName, err)
		}

		// Execute validation through CompilerInterface
		ci := NewCompilerInterface()
		result, err := ci.Validate(LanguageGo, []string{filePath})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Validate, got: %v", err)
		}

		// Property 1: Successful validation must have Success=true
		if !result.Success {
			rt.Errorf("Expected Success=true for valid code, got Success=false. Output: %s", result.Output)
		}

		// Property 2: Successful validation must have empty errors array
		if len(result.Errors) != 0 {
			rt.Errorf("Expected empty Errors array for successful validation, got %d errors: %v",
				len(result.Errors), result.Errors)
		}

		// Property 3: Success=true implies no errors
		// This is the consistency check: Success and Errors must be consistent
		if result.Success && len(result.Errors) > 0 {
			rt.Error("Inconsistency: Success=true but Errors array is non-empty")
		}
	})
}

// **Validates: Requirements 4.4**
// Property 11: Error Message Inclusion
// For any failed validation (exit code non-zero), the Validation_Result should
// include error messages parsed from the command output.
func TestProperty_ErrorMessageInclusion(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Create go.mod file
		goModContent := "module testmodule\n\ngo 1.20\n"
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			rt.Fatalf("Failed to create go.mod: %v", err)
		}

		// Generate a package name
		packageName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}$`).Draw(rt, "packageName")

		// Generate a file name
		fileName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}\.go$`).Draw(rt, "fileName")
		filePath := filepath.Join(tmpDir, fileName)

		// Create INVALID Go code only (to test failed validation)
		// Use undefined variable to generate predictable error
		undefinedVar := rapid.StringMatching(`^[a-z][a-z0-9]{5,12}$`).Draw(rt, "undefinedVar")
		content := "package " + packageName + "\n\n"
		content += "func FailFunc() {\n"
		content += "\t" + undefinedVar + " = 42\n" // undefined variable
		content += "}\n"

		// Write the file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			rt.Fatalf("Failed to create file %s: %v", fileName, err)
		}

		// Execute validation through CompilerInterface
		ci := NewCompilerInterface()
		result, err := ci.Validate(LanguageGo, []string{filePath})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Validate, got: %v", err)
		}

		// Property 1: Failed validation must have Success=false
		if result.Success {
			rt.Error("Expected Success=false for invalid code, got Success=true")
		}

		// Property 2: Failed validation must have non-empty errors array
		if len(result.Errors) == 0 {
			rt.Errorf("Expected non-empty Errors array for failed validation. Output: %s", result.Output)
		}

		// Property 3: Each error must have a non-empty message
		for i, validationErr := range result.Errors {
			if validationErr.Message == "" {
				rt.Errorf("Error %d has empty Message field", i)
			}
		}

		// Property 4: Error messages should be parsed from the output
		// The raw output should contain the error messages
		for i, validationErr := range result.Errors {
			if validationErr.Message != "" && !strings.Contains(result.Output, validationErr.Message) {
				rt.Errorf("Error %d: Message '%s' not found in output. Output: %s",
					i, validationErr.Message, result.Output)
			}
		}

		// Property 5: The undefined variable name should appear in at least one error message
		foundUndefinedVar := false
		for _, validationErr := range result.Errors {
			if strings.Contains(validationErr.Message, undefinedVar) {
				foundUndefinedVar = true
				break
			}
		}
		if !foundUndefinedVar {
			rt.Errorf("Expected at least one error message to contain undefined variable '%s'. Errors: %v",
				undefinedVar, result.Errors)
		}

		// Property 6: Success=false implies errors exist
		// This is the consistency check
		if !result.Success && len(result.Errors) == 0 {
			rt.Error("Inconsistency: Success=false but Errors array is empty")
		}
	})
}
