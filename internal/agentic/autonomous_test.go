package agentic

import (
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

func TestFindMainPythonFile(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected string
	}{
		{
			name: "main.py exists",
			files: map[string]string{
				"main.py":  "print('hello')",
				"utils.py": "# utils",
			},
			expected: "main.py",
		},
		{
			name: "app.py when no main.py",
			files: map[string]string{
				"app.py":   "print('hello')",
				"utils.py": "# utils",
			},
			expected: "app.py",
		},
		{
			name: "any .py file",
			files: map[string]string{
				"script.py": "print('hello')",
			},
			expected: "script.py",
		},
		{
			name:     "no python files",
			files:    map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &AutonomousCreator{
				FilesToMake: tt.files,
			}
			result := creator.findMainPythonFile()
			if result != tt.expected {
				t.Errorf("findMainPythonFile() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindMainShellFile(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected string
	}{
		{
			name: "main.sh exists",
			files: map[string]string{
				"main.sh":  "#!/bin/bash",
				"utils.sh": "# utils",
			},
			expected: "main.sh",
		},
		{
			name: "run.sh when no main.sh",
			files: map[string]string{
				"run.sh": "#!/bin/bash",
			},
			expected: "run.sh",
		},
		{
			name:     "no shell files",
			files:    map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &AutonomousCreator{
				FilesToMake: tt.files,
			}
			result := creator.findMainShellFile()
			if result != tt.expected {
				t.Errorf("findMainShellFile() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindMainPowerShellFile(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected string
	}{
		{
			name: "main.ps1 exists",
			files: map[string]string{
				"main.ps1":  "Write-Host 'hello'",
				"utils.ps1": "# utils",
			},
			expected: "main.ps1",
		},
		{
			name: "run.ps1 when no main.ps1",
			files: map[string]string{
				"run.ps1": "Write-Host 'hello'",
			},
			expected: "run.ps1",
		},
		{
			name:     "no PowerShell files",
			files:    map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &AutonomousCreator{
				FilesToMake: tt.files,
			}
			result := creator.findMainPowerShellFile()
			if result != tt.expected {
				t.Errorf("findMainPowerShellFile() = %v, want %v", result, tt.expected)
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
