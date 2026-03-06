package docgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectAnalyzer_DiscoverFiles(t *testing.T) {
	// Create a temporary test directory
	tmpDir := t.TempDir()

	// Create test file structure
	testFiles := map[string]string{
		"main.go":                   "package main",
		"README.md":                 "# Test Project",
		"go.mod":                    "module test",
		"internal/app/app.go":       "package app",
		"internal/app/app_test.go":  "package app",
		"pkg/util/util.py":          "# Python file",
		"config.yaml":               "key: value",
		".gitignore":                "*.log\nnode_modules/",
		"test.log":                  "log content",
		"node_modules/pkg/index.js": "// Should be skipped",
		"vendor/lib/lib.go":         "// Should be skipped",
		".git/config":               "// Should be skipped",
		"build/output.bin":          "// Should be skipped",
		"dist/bundle.js":            "// Should be skipped",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	tests := []struct {
		name         string
		scopeFilters []string
		wantCode     []string
		wantConfig   []string
		wantDoc      []string
		skipFiles    []string // Files that should NOT be in results
	}{
		{
			name:         "discover all files without scope",
			scopeFilters: nil,
			wantCode: []string{
				"main.go",
				filepath.Join("internal", "app", "app.go"),
				filepath.Join("internal", "app", "app_test.go"),
				filepath.Join("pkg", "util", "util.py"),
			},
			wantConfig: []string{
				"go.mod",
				"config.yaml",
				".gitignore",
			},
			wantDoc: []string{
				"README.md",
			},
			skipFiles: []string{
				"test.log", // Ignored by .gitignore
				filepath.Join("node_modules", "pkg", "index.js"),
				filepath.Join("vendor", "lib", "lib.go"),
				filepath.Join(".git", "config"),
				filepath.Join("build", "output.bin"),
				filepath.Join("dist", "bundle.js"),
			},
		},
		{
			name:         "discover with scope filter - internal directory",
			scopeFilters: []string{"internal"},
			wantCode: []string{
				filepath.Join("internal", "app", "app.go"),
				filepath.Join("internal", "app", "app_test.go"),
			},
			wantConfig: []string{},
			wantDoc:    []string{},
			skipFiles: []string{
				"main.go",
				filepath.Join("pkg", "util", "util.py"),
			},
		},
		{
			name:         "discover with scope filter - specific file",
			scopeFilters: []string{"main.go"},
			wantCode: []string{
				"main.go",
			},
			wantConfig: []string{},
			wantDoc:    []string{},
			skipFiles: []string{
				filepath.Join("internal", "app", "app.go"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewProjectAnalyzer(tmpDir, tt.scopeFilters)
			result, err := analyzer.DiscoverFiles()
			if err != nil {
				t.Fatalf("DiscoverFiles() error = %v", err)
			}

			// Check code files
			if !containsAll(result.CodeFiles, tt.wantCode) {
				t.Errorf("CodeFiles missing expected files.\nGot: %v\nWant: %v", result.CodeFiles, tt.wantCode)
			}

			// Check config files
			if !containsAll(result.ConfigFiles, tt.wantConfig) {
				t.Errorf("ConfigFiles missing expected files.\nGot: %v\nWant: %v", result.ConfigFiles, tt.wantConfig)
			}

			// Check doc files
			if !containsAll(result.DocFiles, tt.wantDoc) {
				t.Errorf("DocFiles missing expected files.\nGot: %v\nWant: %v", result.DocFiles, tt.wantDoc)
			}

			// Check that skip files are not present
			for _, skipFile := range tt.skipFiles {
				if contains(result.AllFiles, skipFile) {
					t.Errorf("AllFiles should not contain %s, but it does", skipFile)
				}
			}
		})
	}
}

func TestProjectAnalyzer_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	result, err := analyzer.DiscoverFiles()
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	if len(result.AllFiles) != 0 {
		t.Errorf("Expected no files in empty directory, got %d files", len(result.AllFiles))
	}
}

func TestProjectAnalyzer_OnlyIgnoredDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only ignored directories
	ignoredDirs := []string{"node_modules", "vendor", ".git", "build", "dist"}
	for _, dir := range ignoredDirs {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		// Add a file in each directory
		filePath := filepath.Join(dirPath, "test.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file in %s: %v", dir, err)
		}
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	result, err := analyzer.DiscoverFiles()
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	if len(result.AllFiles) != 0 {
		t.Errorf("Expected no files when only ignored directories present, got %d files: %v", len(result.AllFiles), result.AllFiles)
	}
}

func TestGitignoreParser_IsIgnored(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore file
	gitignoreContent := `# Comment
*.log
*.tmp
node_modules/
build/
test-*
`
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	parser := NewGitignoreParser(tmpDir)

	tests := []struct {
		path    string
		ignored bool
	}{
		{"test.log", true},
		{"debug.log", true},
		{"app.tmp", true},
		{"test-file.txt", true},
		{"test-data.json", true},
		{filepath.Join("node_modules", "pkg", "index.js"), true},
		{filepath.Join("build", "output.bin"), true},
		{"main.go", false},
		{"README.md", false},
		{"test.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := parser.IsIgnored(tt.path)
			if result != tt.ignored {
				t.Errorf("IsIgnored(%s) = %v, want %v", tt.path, result, tt.ignored)
			}
		})
	}
}

func TestGitignoreParser_NoGitignore(t *testing.T) {
	tmpDir := t.TempDir()

	parser := NewGitignoreParser(tmpDir)

	// Should not ignore anything when no .gitignore exists
	if parser.IsIgnored("test.log") {
		t.Error("Should not ignore files when .gitignore doesn't exist")
	}
}

func TestProjectAnalyzer_MultipleScopeFilters(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file structure
	testFiles := map[string]string{
		"main.go":                  "package main",
		"internal/app/app.go":      "package app",
		"internal/util/util.go":    "package util",
		"pkg/api/api.go":           "package api",
		"pkg/client/client.go":     "package client",
		"cmd/server/server.go":     "package main",
		"test/integration/test.go": "package integration",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Test with multiple scope filters
	scopeFilters := []string{"internal", "pkg"}
	analyzer := NewProjectAnalyzer(tmpDir, scopeFilters)
	result, err := analyzer.DiscoverFiles()
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	// Should include files from both internal and pkg directories
	expectedFiles := []string{
		filepath.Join("internal", "app", "app.go"),
		filepath.Join("internal", "util", "util.go"),
		filepath.Join("pkg", "api", "api.go"),
		filepath.Join("pkg", "client", "client.go"),
	}

	for _, expected := range expectedFiles {
		if !contains(result.CodeFiles, expected) {
			t.Errorf("Expected file %s not found in results", expected)
		}
	}

	// Should NOT include files from cmd or test directories
	unexpectedFiles := []string{
		"main.go",
		filepath.Join("cmd", "server", "server.go"),
		filepath.Join("test", "integration", "test.go"),
	}

	for _, unexpected := range unexpectedFiles {
		if contains(result.CodeFiles, unexpected) {
			t.Errorf("Unexpected file %s found in results", unexpected)
		}
	}
}

func TestProjectAnalyzer_NestedScopeFilter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file structure
	testFiles := map[string]string{
		"internal/app/app.go":       "package app",
		"internal/app/handler.go":   "package app",
		"internal/util/util.go":     "package util",
		"internal/config/config.go": "package config",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Test with nested scope filter
	scopeFilters := []string{filepath.Join("internal", "app")}
	analyzer := NewProjectAnalyzer(tmpDir, scopeFilters)
	result, err := analyzer.DiscoverFiles()
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	// Should only include files from internal/app directory
	expectedFiles := []string{
		filepath.Join("internal", "app", "app.go"),
		filepath.Join("internal", "app", "handler.go"),
	}

	for _, expected := range expectedFiles {
		if !contains(result.CodeFiles, expected) {
			t.Errorf("Expected file %s not found in results", expected)
		}
	}

	// Should NOT include files from other internal subdirectories
	unexpectedFiles := []string{
		filepath.Join("internal", "util", "util.go"),
		filepath.Join("internal", "config", "config.go"),
	}

	for _, unexpected := range unexpectedFiles {
		if contains(result.CodeFiles, unexpected) {
			t.Errorf("Unexpected file %s found in results", unexpected)
		}
	}
}

func TestProjectAnalyzer_ScopeFilterNoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file structure
	testFiles := map[string]string{
		"main.go":             "package main",
		"internal/app/app.go": "package app",
		"pkg/util/util.go":    "package util",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Test with scope filter that matches no files
	scopeFilters := []string{"nonexistent"}
	analyzer := NewProjectAnalyzer(tmpDir, scopeFilters)
	result, err := analyzer.DiscoverFiles()
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	// Should return empty results
	if len(result.AllFiles) != 0 {
		t.Errorf("Expected no files with non-matching scope filter, got %d files: %v", len(result.AllFiles), result.AllFiles)
	}
}

func TestProjectAnalyzer_FileCategorization(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files with various extensions and names
	testFiles := map[string]struct {
		content      string
		expectedType string // "code", "config", "doc", or "other"
	}{
		"main.go":       {"package main", "code"},
		"script.py":     {"# Python", "code"},
		"app.js":        {"// JavaScript", "code"},
		"component.tsx": {"// TypeScript React", "code"},
		"README.md":     {"# README", "doc"},
		"CHANGELOG.md":  {"# Changelog", "doc"},
		"LICENSE":       {"MIT License", "doc"},
		"go.mod":        {"module test", "config"},
		"package.json":  {"{}", "config"},
		"config.yaml":   {"key: value", "config"},
		"settings.toml": {"[settings]", "config"},
		".env":          {"KEY=value", "config"},
		"data.txt":      {"text data", "doc"},
		"notes.rst":     {"reStructuredText", "doc"},
		"Makefile":      {"all:", "config"},
		"Dockerfile":    {"FROM alpine", "config"},
		"noextension":   {"no extension", "other"},
		".hidden":       {"hidden file", "other"},
	}

	for path, info := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(info.content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	result, err := analyzer.DiscoverFiles()
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	// Verify each file is categorized correctly
	for path, info := range testFiles {
		switch info.expectedType {
		case "code":
			if !contains(result.CodeFiles, path) {
				t.Errorf("File %s should be categorized as code", path)
			}
		case "config":
			if !contains(result.ConfigFiles, path) {
				t.Errorf("File %s should be categorized as config", path)
			}
		case "doc":
			if !contains(result.DocFiles, path) {
				t.Errorf("File %s should be categorized as doc", path)
			}
		case "other":
			// Should be in AllFiles but not in any specific category
			if !contains(result.AllFiles, path) {
				t.Errorf("File %s should be in AllFiles", path)
			}
			if contains(result.CodeFiles, path) || contains(result.ConfigFiles, path) || contains(result.DocFiles, path) {
				t.Errorf("File %s should not be in any specific category", path)
			}
		}
	}
}

func TestProjectAnalyzer_HiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files including hidden files
	testFiles := map[string]string{
		"main.go":        "package main",
		".hidden.go":     "package hidden",
		".config":        "config",
		".gitignore":     "*.log",
		"internal/.keep": "",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	result, err := analyzer.DiscoverFiles()
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	// Hidden files should be discovered (not automatically ignored)
	expectedFiles := []string{
		"main.go",
		".hidden.go",
		".config",
		".gitignore",
		filepath.Join("internal", ".keep"),
	}

	for _, expected := range expectedFiles {
		if !contains(result.AllFiles, expected) {
			t.Errorf("Expected hidden file %s not found in results", expected)
		}
	}

	// Verify categorization of hidden files
	if !contains(result.CodeFiles, ".hidden.go") {
		t.Error("Hidden .go file should be categorized as code")
	}

	if !contains(result.ConfigFiles, ".gitignore") {
		t.Error(".gitignore should be categorized as config")
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsAll(slice []string, items []string) bool {
	for _, item := range items {
		if !contains(slice, item) {
			return false
		}
	}
	return true
}

func TestProjectAnalyzer_AnalyzeGoFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a sample Go file with various constructs
	goFileContent := `// TestPackage is a test package for documentation generation
// It demonstrates various Go constructs
package testpkg

import "fmt"

// ExportedConst is an exported constant
const ExportedConst = 42

// unexportedConst is not exported
const unexportedConst = 100

// ExportedVar is an exported variable
var ExportedVar string

// ExportedFunc is an exported function that does something
func ExportedFunc(name string, count int) (string, error) {
	return fmt.Sprintf("Hello %s", name), nil
}

// unexportedFunc is not exported
func unexportedFunc() {
	// internal function
}

// ExportedStruct represents a data structure
type ExportedStruct struct {
	// Name is the name field
	Name string ` + "`json:\"name\"`" + `
	// Age is the age field
	Age int ` + "`json:\"age\"`" + `
	// Internal field (not exported)
	internal string
}

// ExportedInterface defines a contract
type ExportedInterface interface {
	// DoSomething performs an action
	DoSomething(input string) error
	// GetValue returns a value
	GetValue() int
}

// unexportedStruct is not exported
type unexportedStruct struct {
	field string
}

// ExportedMethod is a method on ExportedStruct
func (e *ExportedStruct) ExportedMethod(param string) (bool, error) {
	return true, nil
}
`

	goFilePath := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(goFilePath, []byte(goFileContent), 0644); err != nil {
		t.Fatalf("Failed to create test Go file: %v", err)
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	structure, err := analyzer.AnalyzeGoFile("test.go")
	if err != nil {
		t.Fatalf("AnalyzeGoFile() error = %v", err)
	}

	// Verify package information
	if len(structure.Packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(structure.Packages))
	}
	if structure.Packages[0].Name != "testpkg" {
		t.Errorf("Expected package name 'testpkg', got '%s'", structure.Packages[0].Name)
	}
	if !strings.Contains(structure.Packages[0].Description, "test package") {
		t.Errorf("Expected package description to contain 'test package', got '%s'", structure.Packages[0].Description)
	}

	// Verify functions
	if len(structure.Functions) < 2 {
		t.Errorf("Expected at least 2 functions, got %d", len(structure.Functions))
	}

	// Find ExportedFunc
	var exportedFunc *FunctionInfo
	for i := range structure.Functions {
		if structure.Functions[i].Name == "ExportedFunc" {
			exportedFunc = &structure.Functions[i]
			break
		}
	}
	if exportedFunc == nil {
		t.Fatal("ExportedFunc not found in functions")
	}
	if !exportedFunc.IsExported {
		t.Error("ExportedFunc should be marked as exported")
	}
	if len(exportedFunc.Parameters) != 2 {
		t.Errorf("ExportedFunc should have 2 parameters, got %d", len(exportedFunc.Parameters))
	}
	if len(exportedFunc.Returns) != 2 {
		t.Errorf("ExportedFunc should have 2 return values, got %d", len(exportedFunc.Returns))
	}
	if !strings.Contains(exportedFunc.Comment, "exported function") {
		t.Errorf("ExportedFunc comment should contain 'exported function', got '%s'", exportedFunc.Comment)
	}

	// Verify structs
	if len(structure.Structs) < 1 {
		t.Errorf("Expected at least 1 struct, got %d", len(structure.Structs))
	}

	// Find ExportedStruct
	var exportedStruct *StructInfo
	for i := range structure.Structs {
		if structure.Structs[i].Name == "ExportedStruct" {
			exportedStruct = &structure.Structs[i]
			break
		}
	}
	if exportedStruct == nil {
		t.Fatal("ExportedStruct not found in structs")
	}
	if !exportedStruct.IsExported {
		t.Error("ExportedStruct should be marked as exported")
	}
	if len(exportedStruct.Fields) != 3 {
		t.Errorf("ExportedStruct should have 3 fields, got %d", len(exportedStruct.Fields))
	}

	// Check field details
	nameField := exportedStruct.Fields[0]
	if nameField.Name != "Name" {
		t.Errorf("Expected first field name 'Name', got '%s'", nameField.Name)
	}
	if nameField.Type != "string" {
		t.Errorf("Expected first field type 'string', got '%s'", nameField.Type)
	}
	if !strings.Contains(nameField.Tag, "json") {
		t.Errorf("Expected field tag to contain 'json', got '%s'", nameField.Tag)
	}

	// Verify interfaces
	if len(structure.Interfaces) < 1 {
		t.Errorf("Expected at least 1 interface, got %d", len(structure.Interfaces))
	}

	// Find ExportedInterface
	var exportedInterface *InterfaceInfo
	for i := range structure.Interfaces {
		if structure.Interfaces[i].Name == "ExportedInterface" {
			exportedInterface = &structure.Interfaces[i]
			break
		}
	}
	if exportedInterface == nil {
		t.Fatal("ExportedInterface not found in interfaces")
	}
	if !exportedInterface.IsExported {
		t.Error("ExportedInterface should be marked as exported")
	}
	if len(exportedInterface.Methods) != 2 {
		t.Errorf("ExportedInterface should have 2 methods, got %d", len(exportedInterface.Methods))
	}

	// Verify exports
	if len(structure.Exports) < 5 {
		t.Errorf("Expected at least 5 exports, got %d", len(structure.Exports))
	}

	// Check that exported symbols are in exports
	exportNames := make(map[string]bool)
	for _, exp := range structure.Exports {
		exportNames[exp.Name] = true
	}

	expectedExports := []string{"ExportedConst", "ExportedVar", "ExportedFunc", "ExportedStruct", "ExportedInterface"}
	for _, name := range expectedExports {
		if !exportNames[name] {
			t.Errorf("Expected export '%s' not found in exports", name)
		}
	}

	// Check that unexported symbols are NOT in exports
	unexpectedExports := []string{"unexportedConst", "unexportedFunc", "unexportedStruct"}
	for _, name := range unexpectedExports {
		if exportNames[name] {
			t.Errorf("Unexported symbol '%s' should not be in exports", name)
		}
	}
}

func TestProjectAnalyzer_AnalyzeGoFile_ComplexTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Go file with complex types
	goFileContent := `package complex

import "context"

// Handler is a function type
type Handler func(ctx context.Context, data []byte) error

// ProcessFunc processes data with various parameter types
func ProcessFunc(
	ctx context.Context,
	data []byte,
	opts map[string]interface{},
	callback func(string) error,
	ch chan<- int,
) (result *Result, err error) {
	return nil, nil
}

// Result is a result struct
type Result struct {
	Data []byte
}

// VariadicFunc accepts variadic parameters
func VariadicFunc(prefix string, values ...int) []string {
	return nil
}
`

	goFilePath := filepath.Join(tmpDir, "complex.go")
	if err := os.WriteFile(goFilePath, []byte(goFileContent), 0644); err != nil {
		t.Fatalf("Failed to create test Go file: %v", err)
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	structure, err := analyzer.AnalyzeGoFile("complex.go")
	if err != nil {
		t.Fatalf("AnalyzeGoFile() error = %v", err)
	}

	// Find ProcessFunc
	var processFunc *FunctionInfo
	for i := range structure.Functions {
		if structure.Functions[i].Name == "ProcessFunc" {
			processFunc = &structure.Functions[i]
			break
		}
	}
	if processFunc == nil {
		t.Fatal("ProcessFunc not found")
	}

	// Verify complex parameter types are extracted
	if len(processFunc.Parameters) != 5 {
		t.Errorf("ProcessFunc should have 5 parameters, got %d", len(processFunc.Parameters))
	}

	// Check specific parameter types
	expectedTypes := []string{"context.Context", "[]byte", "map[string]interface{}", "func(string) error", "chan<- int"}
	for i, expected := range expectedTypes {
		if i >= len(processFunc.Parameters) {
			break
		}
		if processFunc.Parameters[i].Type != expected {
			t.Errorf("Parameter %d: expected type '%s', got '%s'", i, expected, processFunc.Parameters[i].Type)
		}
	}

	// Verify return types
	if len(processFunc.Returns) != 2 {
		t.Errorf("ProcessFunc should have 2 return values, got %d", len(processFunc.Returns))
	}

	// Find VariadicFunc
	var variadicFunc *FunctionInfo
	for i := range structure.Functions {
		if structure.Functions[i].Name == "VariadicFunc" {
			variadicFunc = &structure.Functions[i]
			break
		}
	}
	if variadicFunc == nil {
		t.Fatal("VariadicFunc not found")
	}

	// Check variadic parameter
	if len(variadicFunc.Parameters) != 2 {
		t.Errorf("VariadicFunc should have 2 parameters, got %d", len(variadicFunc.Parameters))
	}
	if !strings.Contains(variadicFunc.Parameters[1].Type, "...") {
		t.Errorf("Second parameter should be variadic, got type '%s'", variadicFunc.Parameters[1].Type)
	}
}

func TestProjectAnalyzer_AnalyzeGoFile_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid Go file
	invalidContent := `package invalid

func BrokenFunc( {
	// Missing closing parenthesis
}
`

	goFilePath := filepath.Join(tmpDir, "invalid.go")
	if err := os.WriteFile(goFilePath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to create test Go file: %v", err)
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	_, err := analyzer.AnalyzeGoFile("invalid.go")
	if err == nil {
		t.Error("Expected error when analyzing invalid Go file, got nil")
	}
}

func TestProjectAnalyzer_AnalyzeGoFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an empty Go file
	emptyContent := `package empty
`

	goFilePath := filepath.Join(tmpDir, "empty.go")
	if err := os.WriteFile(goFilePath, []byte(emptyContent), 0644); err != nil {
		t.Fatalf("Failed to create test Go file: %v", err)
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	structure, err := analyzer.AnalyzeGoFile("empty.go")
	if err != nil {
		t.Fatalf("AnalyzeGoFile() error = %v", err)
	}

	// Should have package info but no other elements
	if len(structure.Packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(structure.Packages))
	}
	if len(structure.Functions) != 0 {
		t.Errorf("Expected 0 functions, got %d", len(structure.Functions))
	}
	if len(structure.Structs) != 0 {
		t.Errorf("Expected 0 structs, got %d", len(structure.Structs))
	}
	if len(structure.Interfaces) != 0 {
		t.Errorf("Expected 0 interfaces, got %d", len(structure.Interfaces))
	}
	if len(structure.Exports) != 0 {
		t.Errorf("Expected 0 exports, got %d", len(structure.Exports))
	}
}

// Task 7.5: Unit tests for configuration extraction

func TestProjectAnalyzer_Analyze_GoProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Go project structure
	testFiles := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello")
}
`,
		"go.mod": `module example.com/test

go 1.21

require (
	github.com/pkg/errors v0.9.1
	golang.org/x/sync v0.3.0
)
`,
		"README.md": `# Test Project

This is a test project for documentation generation.

## Installation

Run go install.
`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	result, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// Verify code structure
	if len(result.CodeStructure.Functions) == 0 {
		t.Error("Expected at least 1 function (main)")
	}

	// Verify configuration
	if len(result.Configuration.PackageManifests) != 1 {
		t.Errorf("Expected 1 package manifest, got %d", len(result.Configuration.PackageManifests))
	}
	if result.Configuration.PackageManifests[0] != "go.mod" {
		t.Errorf("Expected go.mod in manifests, got %s", result.Configuration.PackageManifests[0])
	}

	// Verify dependencies
	if len(result.Dependencies.Runtime) != 2 {
		t.Errorf("Expected 2 runtime dependencies, got %d", len(result.Dependencies.Runtime))
	}

	// Check specific dependencies
	foundPkgErrors := false
	foundXSync := false
	for _, dep := range result.Dependencies.Runtime {
		if dep.Name == "github.com/pkg/errors" {
			foundPkgErrors = true
			if dep.Version != "v0.9.1" {
				t.Errorf("Expected version v0.9.1 for pkg/errors, got %s", dep.Version)
			}
		}
		if dep.Name == "golang.org/x/sync" {
			foundXSync = true
			if dep.Version != "v0.3.0" {
				t.Errorf("Expected version v0.3.0 for x/sync, got %s", dep.Version)
			}
		}
	}
	if !foundPkgErrors {
		t.Error("Expected to find github.com/pkg/errors in dependencies")
	}
	if !foundXSync {
		t.Error("Expected to find golang.org/x/sync in dependencies")
	}

	// Verify documentation
	if result.Documentation.ReadmeContent == "" {
		t.Error("Expected README content to be extracted")
	}
	if !strings.Contains(result.Documentation.ReadmeContent, "Test Project") {
		t.Error("Expected README content to contain 'Test Project'")
	}
}

func TestProjectAnalyzer_Analyze_PythonProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Python project structure
	testFiles := map[string]string{
		"main.py": `"""Main module"""

def hello():
    """Say hello"""
    print("Hello")

class Greeter:
    """A greeter class"""
    def greet(self, name):
        return f"Hello {name}"
`,
		"requirements.txt": `requests==2.28.0
numpy>=1.24.0
pandas==1.5.3
`,
		"README.md": `# Python Test Project

A test project.
`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	result, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// Verify code structure
	if len(result.CodeStructure.Functions) == 0 {
		t.Error("Expected at least 1 function")
	}
	if len(result.CodeStructure.Classes) == 0 {
		t.Error("Expected at least 1 class")
	}

	// Verify configuration
	if len(result.Configuration.PackageManifests) != 1 {
		t.Errorf("Expected 1 package manifest, got %d", len(result.Configuration.PackageManifests))
	}

	// Verify dependencies
	if len(result.Dependencies.Runtime) != 3 {
		t.Errorf("Expected 3 runtime dependencies, got %d", len(result.Dependencies.Runtime))
	}

	// Check specific dependencies
	foundRequests := false
	foundNumpy := false
	foundPandas := false
	for _, dep := range result.Dependencies.Runtime {
		if dep.Name == "requests" {
			foundRequests = true
			if dep.Version != "2.28.0" {
				t.Errorf("Expected version 2.28.0 for requests, got %s", dep.Version)
			}
		}
		if dep.Name == "numpy" {
			foundNumpy = true
			if !strings.HasPrefix(dep.Version, ">=") {
				t.Errorf("Expected version to start with >=, got %s", dep.Version)
			}
		}
		if dep.Name == "pandas" {
			foundPandas = true
		}
	}
	if !foundRequests {
		t.Error("Expected to find requests in dependencies")
	}
	if !foundNumpy {
		t.Error("Expected to find numpy in dependencies")
	}
	if !foundPandas {
		t.Error("Expected to find pandas in dependencies")
	}

	// Verify documentation
	if result.Documentation.ReadmeContent == "" {
		t.Error("Expected README content to be extracted")
	}
}

func TestProjectAnalyzer_Analyze_MixedLanguageProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mixed language project
	testFiles := map[string]string{
		"main.go": `package main

func GoFunc() {}
`,
		"script.py": `def python_func():
    pass
`,
		"app.js": `export function jsFunc() {
    return 42;
}
`,
		"go.mod": `module test

go 1.21
`,
		"package.json": `{
  "name": "test",
  "dependencies": {
    "express": "^4.18.0"
  }
}
`,
		"README.md": `# Mixed Project`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	result, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// Verify code structure from all languages
	if len(result.CodeStructure.Functions) < 3 {
		t.Errorf("Expected at least 3 functions (Go, Python, JS), got %d", len(result.CodeStructure.Functions))
	}

	// Verify multiple package manifests
	if len(result.Configuration.PackageManifests) < 2 {
		t.Errorf("Expected at least 2 package manifests, got %d", len(result.Configuration.PackageManifests))
	}

	// Verify dependencies from both ecosystems
	if len(result.Dependencies.Runtime) < 1 {
		t.Error("Expected at least 1 dependency")
	}
}

func TestProjectAnalyzer_Analyze_EmptyProject(t *testing.T) {
	tmpDir := t.TempDir()

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	result, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// Should return valid result with empty collections
	if result.CodeStructure == nil {
		t.Error("CodeStructure should not be nil")
	}
	if result.Configuration == nil {
		t.Error("Configuration should not be nil")
	}
	if result.Documentation == nil {
		t.Error("Documentation should not be nil")
	}
	if result.Dependencies == nil {
		t.Error("Dependencies should not be nil")
	}

	// All collections should be empty but not nil
	if len(result.CodeStructure.Functions) != 0 {
		t.Error("Expected no functions in empty project")
	}
	if len(result.Configuration.PackageManifests) != 0 {
		t.Error("Expected no manifests in empty project")
	}
}

func TestProjectAnalyzer_ExtractDependencies_PackageJson(t *testing.T) {
	tmpDir := t.TempDir()

	packageJsonContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.0",
    "lodash": "~4.17.21"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  }
}
`

	packageJsonPath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(packageJsonPath, []byte(packageJsonContent), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	deps, err := analyzer.extractDependencies("package.json")
	if err != nil {
		t.Fatalf("extractDependencies() error = %v", err)
	}

	if len(deps.Runtime) < 2 {
		t.Errorf("Expected at least 2 dependencies, got %d", len(deps.Runtime))
	}

	// Check for specific dependencies
	foundExpress := false
	foundLodash := false
	for _, dep := range deps.Runtime {
		if dep.Name == "express" {
			foundExpress = true
			if !strings.Contains(dep.Version, "4.18") {
				t.Errorf("Expected express version to contain 4.18, got %s", dep.Version)
			}
		}
		if dep.Name == "lodash" {
			foundLodash = true
		}
	}

	if !foundExpress {
		t.Error("Expected to find express in dependencies")
	}
	if !foundLodash {
		t.Error("Expected to find lodash in dependencies")
	}
}

func TestProjectAnalyzer_ExtractDependencies_Pipfile(t *testing.T) {
	tmpDir := t.TempDir()

	pipfileContent := `[[source]]
url = "https://pypi.org/simple"
verify_ssl = true
name = "pypi"

[packages]
django = ">=4.0"
requests = "*"
celery = "==5.2.7"

[dev-packages]
pytest = "*"
`

	pipfilePath := filepath.Join(tmpDir, "Pipfile")
	if err := os.WriteFile(pipfilePath, []byte(pipfileContent), 0644); err != nil {
		t.Fatalf("Failed to create Pipfile: %v", err)
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	deps, err := analyzer.extractDependencies("Pipfile")
	if err != nil {
		t.Fatalf("extractDependencies() error = %v", err)
	}

	if len(deps.Runtime) < 3 {
		t.Errorf("Expected at least 3 dependencies, got %d", len(deps.Runtime))
	}

	// Check for specific dependencies
	foundDjango := false
	foundCelery := false
	for _, dep := range deps.Runtime {
		if dep.Name == "django" {
			foundDjango = true
		}
		if dep.Name == "celery" {
			foundCelery = true
			if dep.Version != "==5.2.7" && dep.Version != "5.2.7" {
				t.Errorf("Expected celery version 5.2.7 or ==5.2.7, got %s", dep.Version)
			}
		}
	}

	if !foundDjango {
		t.Error("Expected to find django in dependencies")
	}
	if !foundCelery {
		t.Error("Expected to find celery in dependencies")
	}
}

func TestProjectAnalyzer_ExtractDependencies_CargoToml(t *testing.T) {
	tmpDir := t.TempDir()

	cargoTomlContent := `[package]
name = "test-project"
version = "0.1.0"
edition = "2021"

[dependencies]
serde = "1.0"
tokio = { version = "1.28", features = ["full"] }
reqwest = "0.11"
`

	cargoTomlPath := filepath.Join(tmpDir, "Cargo.toml")
	if err := os.WriteFile(cargoTomlPath, []byte(cargoTomlContent), 0644); err != nil {
		t.Fatalf("Failed to create Cargo.toml: %v", err)
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	deps, err := analyzer.extractDependencies("Cargo.toml")
	if err != nil {
		t.Fatalf("extractDependencies() error = %v", err)
	}

	if len(deps.Runtime) < 2 {
		t.Errorf("Expected at least 2 dependencies, got %d", len(deps.Runtime))
	}

	// Check for specific dependencies
	foundSerde := false
	for _, dep := range deps.Runtime {
		if dep.Name == "serde" {
			foundSerde = true
			if dep.Version != "1.0" {
				t.Errorf("Expected serde version 1.0, got %s", dep.Version)
			}
		}
	}

	if !foundSerde {
		t.Error("Expected to find serde in dependencies")
	}
}

func TestProjectAnalyzer_Analyze_WithBuildScripts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project with build scripts
	testFiles := map[string]string{
		"main.go": `package main

func main() {}
`,
		"Makefile": `all:
	go build
`,
		"build.sh": `#!/bin/bash
go build -o app
`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	analyzer := NewProjectAnalyzer(tmpDir, nil)
	result, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// Verify build scripts are identified
	if len(result.Configuration.BuildScripts) < 1 {
		t.Errorf("Expected at least 1 build script, got %d", len(result.Configuration.BuildScripts))
	}

	foundMakefile := false
	for _, script := range result.Configuration.BuildScripts {
		if filepath.Base(script) == "Makefile" {
			foundMakefile = true
		}
	}
	if !foundMakefile {
		t.Error("Expected to find Makefile in build scripts")
	}
}
