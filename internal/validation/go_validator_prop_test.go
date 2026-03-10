package validation

import (
	"os"
	"path/filepath"
	"testing"

	"pgregory.net/rapid"
)

// **Validates: Requirements 4.2**
// Property 9: Go Package Compilation
// For any set of Go files within the same package, the GoValidator should compile
// all files together as a single package unit, allowing cross-file references.
func TestProperty_GoPackageCompilation(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	rapid.Check(t, func(rt *rapid.T) {
		// Generate a package name
		packageName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}$`).Draw(rt, "packageName")

		// Generate number of files (2-5 files to test multi-file compilation)
		numFiles := rapid.IntRange(2, 5).Draw(rt, "numFiles")

		// Create a temporary directory for the package
		tmpDir := t.TempDir()

		// Create go.mod file
		goModContent := "module testmodule\n\ngo 1.20\n"
		goModPath := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			rt.Fatalf("Failed to create go.mod: %v", err)
		}

		// Generate files with cross-references
		var filePaths []string
		var functionNames []string

		// Generate function names for each file
		for i := 0; i < numFiles; i++ {
			funcName := rapid.StringMatching(`^[A-Z][a-zA-Z0-9]{3,10}$`).Draw(rt, "funcName")
			functionNames = append(functionNames, funcName)
		}

		// Create files where each file can reference functions from other files
		for i := 0; i < numFiles; i++ {
			fileName := rapid.StringMatching(`^[a-z][a-z0-9]{2,8}\.go$`).Draw(rt, "fileName")
			filePath := filepath.Join(tmpDir, fileName)

			// Build file content
			var content string
			content += "package " + packageName + "\n\n"

			// Add imports if needed
			needsImport := rapid.Bool().Draw(rt, "needsImport")
			if needsImport {
				content += "import \"fmt\"\n\n"
			}

			// Define this file's function
			content += "func " + functionNames[i] + "() {\n"
			if needsImport {
				content += "\tfmt.Println(\"" + functionNames[i] + "\")\n"
			}

			// Optionally call another function from a different file (cross-reference)
			if i > 0 {
				shouldCallOther := rapid.Bool().Draw(rt, "shouldCallOther")
				if shouldCallOther {
					// Call a function from a previous file
					otherFuncIdx := rapid.IntRange(0, i-1).Draw(rt, "otherFuncIdx")
					content += "\t" + functionNames[otherFuncIdx] + "()\n"
				}
			}

			content += "}\n"

			// Write the file
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				rt.Fatalf("Failed to create file %s: %v", fileName, err)
			}

			filePaths = append(filePaths, filePath)
		}

		// Execute validation with all files
		validator := NewGoValidator()
		result, err := validator.Execute(filePaths)

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Execute, got: %v", err)
		}

		// Property: All files should be compiled together as a package
		// This means cross-file references should work without errors
		if !result.Success {
			rt.Errorf("Expected successful compilation of multi-file package. Errors: %v, Output: %s",
				result.Errors, result.Output)
		}

		// Property: The result should include all files
		if len(result.Files) != numFiles {
			rt.Errorf("Expected %d files in result, got %d", numFiles, len(result.Files))
		}

		// Property: Language should be Go
		if result.Language != LanguageGo {
			rt.Errorf("Expected language Go, got %s", result.Language)
		}

		// Property: No errors should be present for valid cross-file references
		if len(result.Errors) > 0 {
			rt.Errorf("Expected no errors for valid package compilation, got %d errors: %v",
				len(result.Errors), result.Errors)
		}

		// Property: Duration should be positive
		if result.Duration <= 0 {
			rt.Errorf("Expected positive duration, got %v", result.Duration)
		}
	})
}

// Property 9 (Edge Case): Go Package Compilation with Invalid Cross-References
// This tests that when files have invalid cross-references, the validator
// correctly reports errors, demonstrating that files are being compiled together.
func TestProperty_GoPackageCompilation_InvalidReferences(t *testing.T) {
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

		// Create file1.go with a valid function
		file1Content := "package " + packageName + "\n\nfunc ValidFunc() {}\n"
		file1Path := filepath.Join(tmpDir, "file1.go")
		if err := os.WriteFile(file1Path, []byte(file1Content), 0644); err != nil {
			rt.Fatalf("Failed to create file1.go: %v", err)
		}

		// Create file2.go that calls a non-existent function
		undefinedFunc := rapid.StringMatching(`^[A-Z][a-zA-Z0-9]{5,12}$`).Draw(rt, "undefinedFunc")
		file2Content := "package " + packageName + "\n\nfunc Caller() {\n\t" + undefinedFunc + "()\n}\n"
		file2Path := filepath.Join(tmpDir, "file2.go")
		if err := os.WriteFile(file2Path, []byte(file2Content), 0644); err != nil {
			rt.Fatalf("Failed to create file2.go: %v", err)
		}

		// Execute validation with both files
		validator := NewGoValidator()
		result, err := validator.Execute([]string{file1Path, file2Path})

		// Assertions
		if err != nil {
			rt.Fatalf("Expected no error from Execute, got: %v", err)
		}

		// Property: Compilation should fail due to undefined reference
		if result.Success {
			rt.Error("Expected compilation to fail for undefined function reference")
		}

		// Property: Should have at least one error
		if len(result.Errors) == 0 {
			rt.Error("Expected at least one error for undefined function")
		}

		// Property: Error should mention the undefined function
		foundUndefinedError := false
		for _, err := range result.Errors {
			if containsIgnoreCase(err.Message, "undefined") {
				foundUndefinedError = true
				break
			}
		}
		if !foundUndefinedError {
			rt.Errorf("Expected error message to contain 'undefined', got errors: %v", result.Errors)
		}
	})
}

// Helper function for case-insensitive string matching
func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
