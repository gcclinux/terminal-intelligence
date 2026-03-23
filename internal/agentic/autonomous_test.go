package agentic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectProjectType(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected string
	}{
		{
			name: "Go project with go.mod",
			files: map[string]string{
				"go.mod":  "module test",
				"main.go": "package main",
			},
			expected: "Go",
		},
		{
			name: "Go project with .go files",
			files: map[string]string{
				"main.go": "package main",
			},
			expected: "Go",
		},
		{
			name: "Python project with requirements.txt",
			files: map[string]string{
				"requirements.txt": "flask==2.0.0",
				"main.py":          "print('hello')",
			},
			expected: "Python",
		},
		{
			name: "Python project with .py files",
			files: map[string]string{
				"app.py": "print('hello')",
			},
			expected: "Python",
		},
		{
			name: "Bash project",
			files: map[string]string{
				"script.sh": "#!/bin/bash",
			},
			expected: "Bash/Shell",
		},
		{
			name: "PowerShell project",
			files: map[string]string{
				"script.ps1": "Write-Host 'hello'",
			},
			expected: "PowerShell",
		},
		{
			name: "Node.js project (not supported)",
			files: map[string]string{
				"package.json": "{}",
				"index.js":     "console.log('hello')",
			},
			expected: "Node.js (NOT SUPPORTED - use Go, Python, Bash, or PowerShell instead)",
		},
		{
			name:     "Unknown project type",
			files:    map[string]string{},
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &AutonomousCreator{
				FilesToMake: tt.files,
			}
			result := creator.detectProjectType()
			if result != tt.expected {
				t.Errorf("detectProjectType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetFileList(t *testing.T) {
	files := map[string]string{
		"main.go":   "package main",
		"go.mod":    "module test",
		"README.md": "# Test",
	}

	result := getFileList(files)

	if len(result) != 3 {
		t.Errorf("getFileList() returned %d files, want 3", len(result))
	}

	// Check that all expected files are in the result
	expectedFiles := map[string]bool{
		"main.go":   false,
		"go.mod":    false,
		"README.md": false,
	}

	for _, file := range result {
		if _, exists := expectedFiles[file]; exists {
			expectedFiles[file] = true
		}
	}

	for file, found := range expectedFiles {
		if !found {
			t.Errorf("getFileList() missing expected file: %s", file)
		}
	}
}

func TestDetectWebServer(t *testing.T) {
	tests := []struct {
		name          string
		plan          string
		files         map[string]string
		expectedIsWeb bool
		expectedPort  string
	}{
		{
			name: "Go web server with explicit port in plan",
			plan: "Create a web server on port 7777",
			files: map[string]string{
				"main.go": `package main
import "net/http"
func main() {
	http.ListenAndServe(":7777", nil)
}`,
			},
			expectedIsWeb: true,
			expectedPort:  "7777",
		},
		{
			name: "Python Flask app",
			plan: "Create a REST API on port 5000",
			files: map[string]string{
				"app.py": `from flask import Flask
app = Flask(__name__)
app.run(port=5000)`,
			},
			expectedIsWeb: true,
			expectedPort:  "5000",
		},
		{
			name: "Non-web application",
			plan: "Create a CLI tool",
			files: map[string]string{
				"main.go": `package main
import "fmt"
func main() {
	fmt.Println("Hello")
}`,
			},
			expectedIsWeb: false,
			expectedPort:  "",
		},
		{
			name: "Web server with localhost in plan",
			plan: "Run on localhost:3000",
			files: map[string]string{
				"server.py": "# server code",
			},
			expectedIsWeb: true,
			expectedPort:  "3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &AutonomousCreator{
				Plan:        tt.plan,
				FilesToMake: tt.files,
			}
			isWeb, port := creator.detectWebServer()
			if isWeb != tt.expectedIsWeb {
				t.Errorf("detectWebServer() isWeb = %v, want %v", isWeb, tt.expectedIsWeb)
			}
			if isWeb && port != tt.expectedPort {
				t.Errorf("detectWebServer() port = %v, want %v", port, tt.expectedPort)
			}
		})
	}
}

func TestExtractProjectName(t *testing.T) {
	tests := []struct {
		name     string
		plan     string
		expected string
	}{
		{
			name:     "Standard format with colon",
			plan:     `### 1. Project Name: my-app`,
			expected: "my-app",
		},
		{
			name:     "Backticks format",
			plan:     "### 1. Project Name\n`go-time-app`",
			expected: "go-time-app",
		},
		{
			name:     "Backticks with underscores",
			plan:     "Project Name: `sys_stats`",
			expected: "sys_stats",
		},
		{
			name:     "Multiple backticks, use first with hyphens",
			plan:     "`go-time-app`\nSome text\n`another`",
			expected: "go-time-app",
		},
		{
			name:     "No project name found",
			plan:     "This is a plan without a project name",
			expected: "autonomous-app",
		},
		{
			name:     "Project name with numbers",
			plan:     "Project Name: `app-v2`",
			expected: "app-v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractProjectName(tt.plan)
			if result != tt.expected {
				t.Errorf("extractProjectName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetectPythonBinary(t *testing.T) {
	// This test just verifies the function runs without error
	// The actual result depends on the system's Python installation
	binary := detectPythonBinary()

	// Should return either "python", "python3", or empty string
	if binary != "" && binary != "python" && binary != "python3" {
		t.Errorf("detectPythonBinary returned unexpected value: %s", binary)
	}
}

func TestConvertToWindowsCommands(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		pythonBinary   string
		expectedOutput string
	}{
		{
			name:           "Replace python3 with detected binary",
			input:          "python3 -m venv venv && source venv/bin/activate && pip install requests",
			pythonBinary:   "python",
			expectedOutput: "python -m venv venv && venv\\Scripts\\activate && pip install requests",
		},
		{
			name:           "Replace python3 with python when no binary detected",
			input:          "python3 -m venv venv && source venv/bin/activate",
			pythonBinary:   "",
			expectedOutput: "python -m venv venv && venv\\Scripts\\activate",
		},
		{
			name:           "Replace Unix venv paths with Windows paths",
			input:          "source venv/bin/activate && venv/bin/pip install flask",
			pythonBinary:   "python",
			expectedOutput: "venv\\Scripts\\activate && venv\\Scripts\\pip install flask",
		},
		{
			name:           "Handle backslash paths",
			input:          "source venv\\bin\\activate",
			pythonBinary:   "python",
			expectedOutput: "venv\\Scripts\\activate",
		},
		{
			name:           "No changes needed for Go commands",
			input:          "go mod init my-app && go mod tidy",
			pythonBinary:   "",
			expectedOutput: "go mod init my-app && go mod tidy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToWindowsCommands(tt.input, tt.pythonBinary)
			if result != tt.expectedOutput {
				t.Errorf("convertToWindowsCommands() = %q, want %q", result, tt.expectedOutput)
			}
		})
	}
}

// ─── Unit Tests for fallbackFix, buildFallbackRequest, extractFileFromError ──

// TestFallbackFix_NilFixer verifies that calling fallbackFix on an
// AutonomousCreator with a nil fixer returns (nil, nil).
// **Validates: Requirements 1.4**
func TestFallbackFix_NilFixer(t *testing.T) {
	stubClient := &stubAIClient{response: "ok"}
	creator := NewAutonomousCreator(stubClient, "model", "/workspace", "desc", nil, nil)

	result, err := creator.fallbackFix("some error output", "test", "go test ./...")

	if result != nil {
		t.Fatalf("expected nil result when fixer is nil, got %+v", result)
	}
	if err != nil {
		t.Fatalf("expected nil error when fixer is nil, got %v", err)
	}
}

// TestFallbackFix_NilLogger verifies that calling fallbackFix with a non-nil
// fixer but nil logger does not panic.
// **Validates: Requirements 4.5**
func TestFallbackFix_NilLogger(t *testing.T) {
	stubClient := &stubAIClient{response: "ok"}
	fixer := NewAgenticProjectFixer(stubClient, "model", NewActionLogger(func(msg string) {}))

	creator := NewAutonomousCreator(stubClient, "model", "/workspace", "desc", fixer, nil)
	creator.ProjectDir = t.TempDir()
	creator.FilesToMake = map[string]string{"main.go": "package main"}

	// The test passes if no panic occurs.
	_, _ = creator.fallbackFix("undefined: foo", "test", "go test ./...")
}

// TestBuildFallbackRequest_GoError verifies that buildFallbackRequest constructs
// a correct FixSessionRequest for a Go compiler error, including OpenFilePath.
// **Validates: Requirements 6.1, 6.5**
func TestBuildFallbackRequest_GoError(t *testing.T) {
	projectDir := "/home/user/myproject"
	errorOutput := "main.go:10:5: undefined: foo"

	req := buildFallbackRequest(errorOutput, "build", "go build -o myapp", "Go", projectDir)

	if !strings.Contains(req.Message, errorOutput) {
		t.Fatalf("Message does not contain error output: %s", req.Message)
	}
	if !strings.Contains(req.Message, "go build -o myapp") {
		t.Fatalf("Message does not contain failed command: %s", req.Message)
	}
	if !strings.Contains(req.Message, "Go") {
		t.Fatalf("Message does not contain project type: %s", req.Message)
	}
	if req.ProjectRoot != projectDir {
		t.Fatalf("ProjectRoot = %q, want %q", req.ProjectRoot, projectDir)
	}
	if req.MaxAttempts != 5 {
		t.Fatalf("MaxAttempts = %d, want 5", req.MaxAttempts)
	}
	if req.MaxCycles != 2 {
		t.Fatalf("MaxCycles = %d, want 2", req.MaxCycles)
	}

	wantPath := filepath.Join(projectDir, "main.go")
	if req.OpenFilePath != wantPath {
		t.Fatalf("OpenFilePath = %q, want %q", req.OpenFilePath, wantPath)
	}
}

// TestBuildFallbackRequest_PythonError verifies that buildFallbackRequest
// constructs a correct FixSessionRequest for a Python traceback, with an
// empty OpenFilePath since no .go file pattern is present.
// **Validates: Requirements 6.1, 6.6**
func TestBuildFallbackRequest_PythonError(t *testing.T) {
	projectDir := "/home/user/pyproject"
	errorOutput := "Traceback (most recent call last):\n  File \"app.py\", line 5, in <module>\n    import nonexistent\nModuleNotFoundError: No module named 'nonexistent'"

	req := buildFallbackRequest(errorOutput, "test", "python -m pytest", "Python", projectDir)

	if !strings.Contains(req.Message, errorOutput) {
		t.Fatalf("Message does not contain error output: %s", req.Message)
	}
	if !strings.Contains(req.Message, "python -m pytest") {
		t.Fatalf("Message does not contain failed command: %s", req.Message)
	}
	if !strings.Contains(req.Message, "Python") {
		t.Fatalf("Message does not contain project type: %s", req.Message)
	}
	if req.ProjectRoot != projectDir {
		t.Fatalf("ProjectRoot = %q, want %q", req.ProjectRoot, projectDir)
	}
	if req.MaxAttempts != 5 {
		t.Fatalf("MaxAttempts = %d, want 5", req.MaxAttempts)
	}
	if req.MaxCycles != 2 {
		t.Fatalf("MaxCycles = %d, want 2", req.MaxCycles)
	}
	if req.OpenFilePath != "" {
		t.Fatalf("OpenFilePath = %q, want empty string (no .go file pattern)", req.OpenFilePath)
	}
}

// TestExtractFileFromError_GoCompiler verifies that extractFileFromError
// correctly extracts the file path from a Go compiler error.
// **Validates: Requirements 6.5**
func TestExtractFileFromError_GoCompiler(t *testing.T) {
	projectDir := "/home/user/myproject"
	errorOutput := "main.go:10:5: undefined: foo"

	got := extractFileFromError(errorOutput, projectDir)
	want := filepath.Join(projectDir, "main.go")

	if got != want {
		t.Fatalf("extractFileFromError(%q, %q) = %q, want %q", errorOutput, projectDir, got, want)
	}
}

// TestExtractFileFromError_NoMatch verifies that extractFileFromError returns
// an empty string when the error output does not contain a Go file pattern.
// **Validates: Requirements 6.6**
func TestExtractFileFromError_NoMatch(t *testing.T) {
	projectDir := "/home/user/myproject"
	errorOutput := "some random error"

	got := extractFileFromError(errorOutput, projectDir)

	if got != "" {
		t.Fatalf("extractFileFromError(%q, %q) = %q, want empty string", errorOutput, projectDir, got)
	}
}

// TestFindBuildRoot verifies that findBuildRoot correctly locates the directory
// containing the build manifest (go.mod, package.json, etc.), even when it
// lives in a subdirectory rather than the project root.
func TestFindBuildRoot(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string // relative paths to create
		expected string            // expected suffix of the returned path
	}{
		{
			name: "go.mod at project root",
			files: map[string]string{
				"go.mod":  "module test",
				"main.go": "package main",
			},
			expected: "", // project root itself
		},
		{
			name: "go.mod in backend subdirectory",
			files: map[string]string{
				"backend/go.mod":      "module test",
				"backend/main.go":     "package main",
				"frontend/index.html": "<html></html>",
			},
			expected: "backend",
		},
		{
			name: "package.json in subdirectory",
			files: map[string]string{
				"app/package.json": `{"name":"test"}`,
				"app/index.js":     "console.log('hi')",
			},
			expected: "app",
		},
		{
			name: "no manifest anywhere returns project root",
			files: map[string]string{
				"main.go": "package main",
				"util.go": "package main",
			},
			expected: "",
		},
		{
			name: "multiple subdirs with manifests returns project root (ambiguous)",
			files: map[string]string{
				"svc1/go.mod":       "module svc1",
				"svc2/package.json": `{"name":"svc2"}`,
			},
			expected: "", // ambiguous, falls back to root
		},
		{
			name: "requirements.txt in subdirectory",
			files: map[string]string{
				"api/requirements.txt": "flask",
				"api/app.py":           "print('hi')",
			},
			expected: "api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create the file structure
			for relPath, content := range tt.files {
				absPath := filepath.Join(tmpDir, filepath.FromSlash(relPath))
				dir := filepath.Dir(absPath)
				if err := mkdirAll(dir); err != nil {
					t.Fatalf("failed to create dir %s: %v", dir, err)
				}
				if err := writeFile(absPath, content); err != nil {
					t.Fatalf("failed to write %s: %v", relPath, err)
				}
			}

			creator := &AutonomousCreator{
				ProjectDir: tmpDir,
			}

			got := creator.findBuildRoot()
			want := tmpDir
			if tt.expected != "" {
				want = filepath.Join(tmpDir, tt.expected)
			}

			if got != want {
				t.Errorf("findBuildRoot() = %q, want %q", got, want)
			}
		})
	}
}

// mkdirAll is a test helper that creates directories.
func mkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

// writeFile is a test helper that writes content to a file.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
