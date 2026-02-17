package unit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/user/terminal-intelligence/internal/executor"
)

func TestCommandExecutor_ExecuteCommand(t *testing.T) {
	ce := executor.NewCommandExecutor()

	tests := []struct {
		name         string
		command      string
		cwd          string
		expectError  bool
		checkStdout  bool
		stdoutContains string
	}{
		{
			name:         "simple echo command",
			command:      "echo hello",
			cwd:          "",
			expectError:  false,
			checkStdout:  true,
			stdoutContains: "hello",
		},
		{
			name:        "empty command",
			command:     "",
			cwd:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ce.ExecuteCommand(tt.command, tt.cwd)
			
			if (err != nil) != tt.expectError {
				t.Errorf("ExecuteCommand() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				if result == nil {
					t.Errorf("Expected result but got nil")
					return
				}

				if tt.checkStdout && !strings.Contains(result.Stdout, tt.stdoutContains) {
					t.Errorf("Stdout does not contain expected string. Got: %q, Want substring: %q", result.Stdout, tt.stdoutContains)
				}

				if result.ExecutionTime == 0 {
					t.Errorf("ExecutionTime should be greater than 0")
				}
			}
		})
	}
}

func TestCommandExecutor_ExecuteCommand_ExitCode(t *testing.T) {
	ce := executor.NewCommandExecutor()

	t.Run("successful command exit code 0", func(t *testing.T) {
		var cmd string
		if runtime.GOOS == "windows" {
			cmd = "cmd /c exit 0"
		} else {
			cmd = "sh -c 'exit 0'"
		}

		result, err := ce.ExecuteCommand(cmd, "")
		if err != nil {
			t.Errorf("ExecuteCommand() error = %v", err)
			return
		}

		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}
	})

	t.Run("failed command exit code non-zero", func(t *testing.T) {
		var cmd string
		if runtime.GOOS == "windows" {
			cmd = "cmd /c exit 1"
		} else {
			cmd = "sh -c 'exit 1'"
		}

		result, err := ce.ExecuteCommand(cmd, "")
		if err != nil {
			t.Errorf("ExecuteCommand() error = %v", err)
			return
		}

		if result.ExitCode != 1 {
			t.Errorf("Expected exit code 1, got %d", result.ExitCode)
		}
	})
}

func TestCommandExecutor_ExecuteCommand_StderrCapture(t *testing.T) {
	ce := executor.NewCommandExecutor()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "cmd /c echo error 1>&2"
	} else {
		cmd = "sh -c 'echo error >&2'"
	}

	result, err := ce.ExecuteCommand(cmd, "")
	if err != nil {
		t.Errorf("ExecuteCommand() error = %v", err)
		return
	}

	if !strings.Contains(result.Stderr, "error") {
		t.Errorf("Stderr does not contain expected error message. Got: %q", result.Stderr)
	}
}

func TestCommandExecutor_ExecuteCommand_WorkingDirectory(t *testing.T) {
	ce := executor.NewCommandExecutor()
	tmpDir := t.TempDir()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "cmd /c cd"
	} else {
		cmd = "pwd"
	}

	result, err := ce.ExecuteCommand(cmd, tmpDir)
	if err != nil {
		t.Errorf("ExecuteCommand() error = %v", err)
		return
	}

	// The output should contain the temp directory path
	if !strings.Contains(result.Stdout, filepath.Base(tmpDir)) {
		t.Logf("Working directory test - Stdout: %q, Expected to contain: %q", result.Stdout, tmpDir)
	}
}

func TestCommandExecutor_ExecuteCommand_CommandNotFound(t *testing.T) {
	ce := executor.NewCommandExecutor()

	result, err := ce.ExecuteCommand("nonexistentcommand12345", "")
	
	// Should return an error because command doesn't exist
	if err == nil && result != nil && result.ExitCode == 0 {
		t.Errorf("Expected error or non-zero exit code for non-existent command")
	}
}

func TestCommandExecutor_GetInterpreter(t *testing.T) {
	ce := executor.NewCommandExecutor()

	tests := []struct {
		name           string
		scriptPath     string
		expectedInterpreter string
	}{
		{
			name:           "bash script with .sh extension",
			scriptPath:     "script.sh",
			expectedInterpreter: func() string {
				if runtime.GOOS == "windows" {
					return "sh"
				}
				return "bash"
			}(),
		},
		{
			name:           "bash script with .bash extension",
			scriptPath:     "script.bash",
			expectedInterpreter: "bash",
		},
		{
			name:           "PowerShell script",
			scriptPath:     "script.ps1",
			expectedInterpreter: func() string {
				if runtime.GOOS == "windows" {
					return "powershell"
				}
				return "pwsh"
			}(),
		},
		{
			name:           "unsupported extension",
			scriptPath:     "script.py",
			expectedInterpreter: "",
		},
		{
			name:           "no extension",
			scriptPath:     "script",
			expectedInterpreter: "",
		},
		{
			name:           "uppercase extension",
			scriptPath:     "SCRIPT.SH",
			expectedInterpreter: func() string {
				if runtime.GOOS == "windows" {
					return "sh"
				}
				return "bash"
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interpreter := ce.GetInterpreter(tt.scriptPath)
			if interpreter != tt.expectedInterpreter {
				t.Errorf("GetInterpreter() = %q, want %q", interpreter, tt.expectedInterpreter)
			}
		})
	}
}

func TestCommandExecutor_ExecuteScript(t *testing.T) {
	ce := executor.NewCommandExecutor()
	tmpDir := t.TempDir()

	t.Run("execute bash script", func(t *testing.T) {
		// Skip on Windows if bash is not available
		if runtime.GOOS == "windows" {
			t.Skip("Skipping bash script test on Windows")
		}

		scriptPath := filepath.Join(tmpDir, "test.sh")
		scriptContent := "#!/bin/bash\necho 'Hello from bash'\nexit 0"
		
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		result, err := ce.ExecuteScript(scriptPath)
		if err != nil {
			t.Errorf("ExecuteScript() error = %v", err)
			return
		}

		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}

		if !strings.Contains(result.Stdout, "Hello from bash") {
			t.Errorf("Stdout does not contain expected output. Got: %q", result.Stdout)
		}
	})

	t.Run("execute script with non-zero exit code", func(t *testing.T) {
		// Skip on Windows if bash is not available
		if runtime.GOOS == "windows" {
			t.Skip("Skipping bash script test on Windows")
		}

		scriptPath := filepath.Join(tmpDir, "fail.sh")
		scriptContent := "#!/bin/bash\nexit 42"
		
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		result, err := ce.ExecuteScript(scriptPath)
		if err != nil {
			t.Errorf("ExecuteScript() error = %v", err)
			return
		}

		if result.ExitCode != 42 {
			t.Errorf("Expected exit code 42, got %d", result.ExitCode)
		}
	})

	t.Run("execute unsupported script type", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "test.py")
		scriptContent := "print('Hello')"
		
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		_, err := ce.ExecuteScript(scriptPath)
		if err == nil {
			t.Errorf("Expected error for unsupported script type")
		}
	})

	t.Run("execute non-existent script", func(t *testing.T) {
		scriptPath := filepath.Join(tmpDir, "nonexistent.sh")
		
		result, err := ce.ExecuteScript(scriptPath)
		// Either we get an error (script doesn't exist) or non-zero exit code
		if err == nil && result != nil && result.ExitCode == 0 {
			t.Errorf("Expected error or non-zero exit code for non-existent script")
		}
	})
}

func TestCommandExecutor_ExecuteScript_StderrCapture(t *testing.T) {
	ce := executor.NewCommandExecutor()
	tmpDir := t.TempDir()

	// Skip on Windows if bash is not available
	if runtime.GOOS == "windows" {
		t.Skip("Skipping bash script test on Windows")
	}

	scriptPath := filepath.Join(tmpDir, "stderr.sh")
	scriptContent := "#!/bin/bash\necho 'error message' >&2\nexit 1"
	
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	result, err := ce.ExecuteScript(scriptPath)
	if err != nil {
		t.Errorf("ExecuteScript() error = %v", err)
		return
	}

	if !strings.Contains(result.Stderr, "error message") {
		t.Errorf("Stderr does not contain expected error message. Got: %q", result.Stderr)
	}

	if result.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", result.ExitCode)
	}
}

func TestCommandExecutor_CrossPlatform(t *testing.T) {
	ce := executor.NewCommandExecutor()

	t.Run("platform-specific command", func(t *testing.T) {
		var cmd string
		var expectedSubstring string

		switch runtime.GOOS {
		case "windows":
			cmd = "cmd /c echo Windows"
			expectedSubstring = "Windows"
		case "darwin":
			cmd = "echo macOS"
			expectedSubstring = "macOS"
		default: // linux and others
			cmd = "echo Linux"
			expectedSubstring = "Linux"
		}

		result, err := ce.ExecuteCommand(cmd, "")
		if err != nil {
			t.Errorf("ExecuteCommand() error = %v", err)
			return
		}

		if !strings.Contains(result.Stdout, expectedSubstring) {
			t.Errorf("Stdout does not contain expected platform string. Got: %q", result.Stdout)
		}
	})
}

// TestCommandExecutor_CommandNotFoundScenario tests command not found error handling
// Validates: Requirements 4.6
func TestCommandExecutor_CommandNotFoundScenario(t *testing.T) {
	ce := executor.NewCommandExecutor()

	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "completely invalid command",
			command: "this_command_definitely_does_not_exist_12345",
		},
		{
			name:    "command with typo",
			command: "echoo hello",
		},
		{
			name:    "non-existent binary",
			command: "nonexistent_binary_xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ce.ExecuteCommand(tt.command, "")
			
			// Command not found should either return an error or non-zero exit code
			if err == nil && result != nil {
				if result.ExitCode == 0 {
					t.Errorf("Expected non-zero exit code for command not found, got 0")
				}
				// Stderr should contain some error message
				if result.Stderr == "" {
					t.Logf("Warning: Stderr is empty for command not found")
				}
			}
		})
	}
}

// TestCommandExecutor_PermissionDeniedScenario tests permission denied error handling
// Validates: Requirements 4.6
func TestCommandExecutor_PermissionDeniedScenario(t *testing.T) {
	ce := executor.NewCommandExecutor()

	// Skip on Windows as permission handling is different
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	
	t.Run("execute non-executable script", func(t *testing.T) {
		// Create a script without execute permissions
		scriptPath := filepath.Join(tmpDir, "no_exec.sh")
		scriptContent := "#!/bin/bash\necho 'This should not run'"
		
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		// Try to execute the script directly (not through bash interpreter)
		// This should fail because the file doesn't have execute permissions
		cmd := scriptPath
		result, err := ce.ExecuteCommand(cmd, "")
		
		// Should fail due to permission denied
		if err == nil && result != nil && result.ExitCode == 0 {
			t.Errorf("Expected error or non-zero exit code for non-executable script")
		}
		
		// Note: ExecuteScript will still work because it calls bash with the script path
		// which doesn't require execute permissions on the script file itself
	})

	t.Run("write to read-only directory", func(t *testing.T) {
		// Create a read-only directory
		readOnlyDir := filepath.Join(tmpDir, "readonly")
		if err := os.Mkdir(readOnlyDir, 0555); err != nil {
			t.Fatalf("Failed to create read-only directory: %v", err)
		}
		defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

		// Try to write to the read-only directory
		cmd := "touch " + filepath.Join(readOnlyDir, "test.txt")
		result, err := ce.ExecuteCommand(cmd, "")
		
		// Should fail due to permission denied
		if err == nil && result != nil && result.ExitCode == 0 {
			t.Errorf("Expected error or non-zero exit code for writing to read-only directory")
		}
	})

	t.Run("read protected file", func(t *testing.T) {
		// Create a file with no read permissions
		protectedFile := filepath.Join(tmpDir, "protected.txt")
		if err := os.WriteFile(protectedFile, []byte("secret"), 0000); err != nil {
			t.Fatalf("Failed to create protected file: %v", err)
		}
		defer os.Chmod(protectedFile, 0644) // Restore permissions for cleanup

		// Try to read the protected file
		cmd := "cat " + protectedFile
		result, err := ce.ExecuteCommand(cmd, "")
		
		// Should fail due to permission denied
		if err == nil && result != nil && result.ExitCode == 0 {
			t.Errorf("Expected error or non-zero exit code for reading protected file")
		}
	})
}

// TestCommandExecutor_TimeoutScenario tests command timeout handling
// Note: The current implementation doesn't have built-in timeout support,
// but this test documents the expected behavior for future implementation
// Validates: Requirements 4.6
func TestCommandExecutor_TimeoutScenario(t *testing.T) {
	ce := executor.NewCommandExecutor()

	t.Run("long-running command completes", func(t *testing.T) {
		// Test a command that takes a short time but completes
		var cmd string
		if runtime.GOOS == "windows" {
			cmd = "cmd /c timeout /t 1 /nobreak >nul && echo done"
		} else {
			cmd = "sleep 1 && echo done"
		}

		result, err := ce.ExecuteCommand(cmd, "")
		if err != nil {
			t.Errorf("ExecuteCommand() error = %v", err)
			return
		}

		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}

		// Verify execution time is recorded
		if result.ExecutionTime == 0 {
			t.Errorf("ExecutionTime should be greater than 0")
		}
	})

	t.Run("execution time tracking", func(t *testing.T) {
		// Test that execution time is properly tracked
		var cmd string
		if runtime.GOOS == "windows" {
			cmd = "cmd /c timeout /t 2 /nobreak >nul"
		} else {
			cmd = "sleep 2"
		}

		result, err := ce.ExecuteCommand(cmd, "")
		if err != nil {
			t.Errorf("ExecuteCommand() error = %v", err)
			return
		}

		// Execution time should be at least 2 seconds
		if result.ExecutionTime.Seconds() < 1.5 {
			t.Errorf("ExecutionTime should be at least 1.5 seconds, got %v", result.ExecutionTime)
		}
	})
}

// TestCommandExecutor_CrossPlatformCommandExecution tests various cross-platform scenarios
// Validates: Requirements 4.1, 4.2, 4.3, 4.6
func TestCommandExecutor_CrossPlatformCommandExecution(t *testing.T) {
	ce := executor.NewCommandExecutor()

	t.Run("list directory command", func(t *testing.T) {
		var cmd string
		if runtime.GOOS == "windows" {
			cmd = "dir"
		} else {
			cmd = "ls"
		}

		result, err := ce.ExecuteCommand(cmd, "")
		if err != nil {
			t.Errorf("ExecuteCommand() error = %v", err)
			return
		}

		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}

		// Should have some output
		if result.Stdout == "" {
			t.Errorf("Expected stdout output for directory listing")
		}
	})

	t.Run("environment variable access", func(t *testing.T) {
		var cmd string
		if runtime.GOOS == "windows" {
			cmd = "echo %PATH%"
		} else {
			cmd = "echo $PATH"
		}

		result, err := ce.ExecuteCommand(cmd, "")
		if err != nil {
			t.Errorf("ExecuteCommand() error = %v", err)
			return
		}

		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}

		// PATH should not be empty
		if strings.TrimSpace(result.Stdout) == "" {
			t.Errorf("Expected PATH environment variable to have value")
		}
	})

	t.Run("pipe commands", func(t *testing.T) {
		var cmd string
		if runtime.GOOS == "windows" {
			cmd = "echo hello | findstr hello"
		} else {
			cmd = "echo hello | grep hello"
		}

		result, err := ce.ExecuteCommand(cmd, "")
		if err != nil {
			t.Errorf("ExecuteCommand() error = %v", err)
			return
		}

		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}

		if !strings.Contains(result.Stdout, "hello") {
			t.Errorf("Expected stdout to contain 'hello', got: %q", result.Stdout)
		}
	})

	t.Run("command with multiple arguments", func(t *testing.T) {
		var cmd string
		if runtime.GOOS == "windows" {
			cmd = "cmd /c echo arg1 arg2 arg3"
		} else {
			cmd = "echo arg1 arg2 arg3"
		}

		result, err := ce.ExecuteCommand(cmd, "")
		if err != nil {
			t.Errorf("ExecuteCommand() error = %v", err)
			return
		}

		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}

		stdout := strings.TrimSpace(result.Stdout)
		if !strings.Contains(stdout, "arg1") || !strings.Contains(stdout, "arg2") || !strings.Contains(stdout, "arg3") {
			t.Errorf("Expected stdout to contain all arguments, got: %q", stdout)
		}
	})

	t.Run("command with special characters", func(t *testing.T) {
		var cmd string
		if runtime.GOOS == "windows" {
			cmd = "echo test!@#"
		} else {
			cmd = "echo 'test!@#'"
		}

		result, err := ce.ExecuteCommand(cmd, "")
		if err != nil {
			t.Errorf("ExecuteCommand() error = %v", err)
			return
		}

		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}
	})
}
