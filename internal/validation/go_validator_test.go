package validation

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewGoValidator(t *testing.T) {
	validator := NewGoValidator()

	if validator == nil {
		t.Fatal("Expected validator to be created")
	}

	info := validator.GetInfo()
	if info.Name != "Go" {
		t.Errorf("Expected name 'Go', got %s", info.Name)
	}
	if info.Command != "go" {
		t.Errorf("Expected command 'go', got %s", info.Command)
	}
}

func TestGoValidator_ParseErrors_SingleError(t *testing.T) {
	validator := NewGoValidator()

	output := "main.go:15:2: undefined: fmt.Printl"
	errors, warnings := validator.parseErrors(output)

	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(warnings))
	}

	err := errors[0]
	if err.File != "main.go" {
		t.Errorf("Expected file 'main.go', got %s", err.File)
	}
	if err.Line != 15 {
		t.Errorf("Expected line 15, got %d", err.Line)
	}
	if err.Column != 2 {
		t.Errorf("Expected column 2, got %d", err.Column)
	}
	if err.Message != "undefined: fmt.Printl" {
		t.Errorf("Expected message 'undefined: fmt.Printl', got %s", err.Message)
	}
	if err.Severity != SeverityError {
		t.Errorf("Expected severity error, got %s", err.Severity)
	}
}

func TestGoValidator_ParseErrors_MultipleErrors(t *testing.T) {
	validator := NewGoValidator()

	output := `main.go:15:2: undefined: fmt.Printl
handler.go:23:10: syntax error: unexpected newline
utils.go:5:1: expected declaration, found 'EOF'`

	errors, _ := validator.parseErrors(output)

	if len(errors) != 3 {
		t.Fatalf("Expected 3 errors, got %d", len(errors))
	}

	// Check first error
	if errors[0].File != "main.go" || errors[0].Line != 15 {
		t.Errorf("First error incorrect: %+v", errors[0])
	}

	// Check second error
	if errors[1].File != "handler.go" || errors[1].Line != 23 {
		t.Errorf("Second error incorrect: %+v", errors[1])
	}

	// Check third error
	if errors[2].File != "utils.go" || errors[2].Line != 5 {
		t.Errorf("Third error incorrect: %+v", errors[2])
	}
}

func TestGoValidator_ParseErrors_WithWarnings(t *testing.T) {
	validator := NewGoValidator()

	output := `main.go:10:5: warning: unused variable x
main.go:15:2: undefined: fmt.Printl`

	errors, warnings := validator.parseErrors(output)

	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}
	if len(warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(warnings))
	}

	if warnings[0].Severity != SeverityWarning {
		t.Errorf("Expected warning severity, got %s", warnings[0].Severity)
	}
	if errors[0].Severity != SeverityError {
		t.Errorf("Expected error severity, got %s", errors[0].Severity)
	}
}

func TestGoValidator_ParseErrors_EmptyOutput(t *testing.T) {
	validator := NewGoValidator()

	errors, warnings := validator.parseErrors("")

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(warnings))
	}
}

func TestGoValidator_ParseErrors_NoMatches(t *testing.T) {
	validator := NewGoValidator()

	output := "Some random output that doesn't match the pattern"
	errors, warnings := validator.parseErrors(output)

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
	if len(warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(warnings))
	}
}

func TestGoValidator_ParseErrors_WithEmptyLines(t *testing.T) {
	validator := NewGoValidator()

	output := `
main.go:15:2: undefined: fmt.Printl

handler.go:23:10: syntax error: unexpected newline

`

	errors, _ := validator.parseErrors(output)

	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}
}

func TestGoValidator_Execute_NoFiles(t *testing.T) {
	validator := NewGoValidator()

	_, err := validator.Execute([]string{})

	if err == nil {
		t.Error("Expected error when no files provided")
	}
	if !strings.Contains(err.Error(), "no files provided") {
		t.Errorf("Expected 'no files provided' error, got: %v", err)
	}
}

// TestGoValidator_Execute_ValidCode tests validation of valid Go code
func TestGoValidator_Execute_ValidCode(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	// Create a temporary directory with valid Go code
	tmpDir := t.TempDir()

	// Create go.mod file
	goModContent := `module testmodule

go 1.20
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a valid Go file
	validCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte(validCode), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := NewGoValidator()
	result, err := validator.Execute([]string{testFile})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure. Output: %s", result.Output)
	}
	if result.Language != LanguageGo {
		t.Errorf("Expected language Go, got %s", result.Language)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if result.Duration <= 0 {
		t.Error("Expected positive duration")
	}
}

// TestGoValidator_Execute_InvalidCode tests validation of invalid Go code
func TestGoValidator_Execute_InvalidCode(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	// Create a temporary directory with invalid Go code
	tmpDir := t.TempDir()

	// Create go.mod file
	goModContent := `module testmodule

go 1.20
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create an invalid Go file (undefined function)
	invalidCode := `package main

func main() {
	undefinedFunction()
}
`
	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte(invalidCode), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := NewGoValidator()
	result, err := validator.Execute([]string{testFile})

	if err != nil {
		t.Fatalf("Expected no error from Execute, got: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for invalid code")
	}
	if result.Language != LanguageGo {
		t.Errorf("Expected language Go, got %s", result.Language)
	}
	if len(result.Errors) == 0 {
		t.Error("Expected at least one error")
	}
	if result.Output == "" {
		t.Error("Expected non-empty output")
	}
}

// TestGoValidator_Execute_MultipleFiles tests validation of multiple Go files in a package
func TestGoValidator_Execute_MultipleFiles(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	// Create a temporary directory with multiple Go files
	tmpDir := t.TempDir()

	// Create go.mod file
	goModContent := `module testmodule

go 1.20
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create main.go
	mainCode := `package main

func main() {
	helper()
}
`
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(mainCode), 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	// Create helper.go
	helperCode := `package main

import "fmt"

func helper() {
	fmt.Println("Helper function")
}
`
	helperFile := filepath.Join(tmpDir, "helper.go")
	if err := os.WriteFile(helperFile, []byte(helperCode), 0644); err != nil {
		t.Fatalf("Failed to create helper.go: %v", err)
	}

	validator := NewGoValidator()
	result, err := validator.Execute([]string{mainFile, helperFile})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success for valid multi-file package. Output: %s", result.Output)
	}
	if len(result.Files) != 2 {
		t.Errorf("Expected 2 files in result, got %d", len(result.Files))
	}
}

// TestGoValidator_GetPackageDirectory tests package directory resolution
func TestGoValidator_GetPackageDirectory(t *testing.T) {
	// Skip if go is not available
	if !isGoAvailable() {
		t.Skip("Go compiler not available")
	}

	// Create a temporary directory with a Go module
	tmpDir := t.TempDir()

	// Create go.mod file
	goModContent := `module testmodule

go 1.20
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a Go file
	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := NewGoValidator()
	packageDir, err := validator.getPackageDirectory(context.Background(), testFile)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// The package directory should be the tmpDir
	if !strings.Contains(packageDir, tmpDir) && packageDir != tmpDir {
		t.Errorf("Expected package dir to be or contain %s, got %s", tmpDir, packageDir)
	}
}

// TestGoValidator_Execute_Timeout tests that validation respects timeout
func TestGoValidator_Execute_Timeout(t *testing.T) {
	t.Skip("Timeout test is difficult to implement reliably without a very slow build")
}

// isGoAvailable checks if the Go compiler is available
func isGoAvailable() bool {
	cmd := exec.Command("go", "version")
	err := cmd.Run()
	return err == nil
}
