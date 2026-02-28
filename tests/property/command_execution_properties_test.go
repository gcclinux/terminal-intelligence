package property

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/user/terminal-intelligence/internal/executor"
)

// Feature: Terminal Intelligence (TI), Property 7: Command Output Capture
// **Validates: Requirements 4.1, 4.2, 4.3**
//
// For any system command executed, the captured stdout and stderr should match
// the actual output produced by running the command directly in the terminal.
func TestProperty_CommandOutputCapture(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("captured stdout matches direct command execution", prop.ForAll(
		func(message string) bool {
			ce := executor.NewCommandExecutor()

			// Create a simple echo command
			var cmd string
			if runtime.GOOS == "windows" {
				cmd = "echo " + message
			} else {
				cmd = "echo " + message
			}

			// Execute through CommandExecutor
			result, err := ce.ExecuteCommand(cmd, "")
			if err != nil {
				t.Logf("ExecuteCommand failed: %v", err)
				return false
			}

			// Execute directly to compare
			var directCmd *exec.Cmd
			if runtime.GOOS == "windows" {
				directCmd = exec.Command("cmd", "/c", cmd)
			} else {
				directCmd = exec.Command("sh", "-c", cmd)
			}

			directOutput, err := directCmd.Output()
			if err != nil {
				t.Logf("Direct command execution failed: %v", err)
				return false
			}

			// Compare outputs (trim whitespace for comparison)
			capturedStdout := strings.TrimSpace(result.Stdout)
			directStdout := strings.TrimSpace(string(directOutput))

			if capturedStdout != directStdout {
				t.Logf("Output mismatch: captured=%q, direct=%q", capturedStdout, directStdout)
				return false
			}

			return true
		},
		genSafeCommandMessage(),
	))

	properties.Property("captured stderr matches direct command execution", prop.ForAll(
		func(message string) bool {
			ce := executor.NewCommandExecutor()

			// Create a command that outputs to stderr
			var cmd string
			if runtime.GOOS == "windows" {
				cmd = "cmd /c echo " + message + " 1>&2"
			} else {
				cmd = "sh -c 'echo " + message + " >&2'"
			}

			// Execute through CommandExecutor
			result, err := ce.ExecuteCommand(cmd, "")
			if err != nil {
				t.Logf("ExecuteCommand failed: %v", err)
				return false
			}

			// Execute directly to compare
			var directCmd *exec.Cmd
			if runtime.GOOS == "windows" {
				directCmd = exec.Command("cmd", "/c", cmd)
			} else {
				directCmd = exec.Command("sh", "-c", cmd)
			}

			directOutput, err := directCmd.CombinedOutput()
			if err != nil {
				// For stderr commands, we expect an exit code but still get output
				if exitErr, ok := err.(*exec.ExitError); ok {
					directOutput = exitErr.Stderr
				}
			}

			// Compare stderr outputs (trim whitespace for comparison)
			capturedStderr := strings.TrimSpace(result.Stderr)
			directStderr := strings.TrimSpace(string(directOutput))

			if capturedStderr != directStderr {
				t.Logf("Stderr mismatch: captured=%q, direct=%q", capturedStderr, directStderr)
				return false
			}

			return true
		},
		genSafeCommandMessage(),
	))

	properties.Property("both stdout and stderr captured correctly", prop.ForAll(
		func(stdoutMsg string, stderrMsg string) bool {
			ce := executor.NewCommandExecutor()

			// Create a command that outputs to both stdout and stderr
			var cmd string
			if runtime.GOOS == "windows" {
				cmd = "cmd /c (echo " + stdoutMsg + " && echo " + stderrMsg + " 1>&2)"
			} else {
				cmd = "sh -c 'echo " + stdoutMsg + " && echo " + stderrMsg + " >&2'"
			}

			// Execute through CommandExecutor
			result, err := ce.ExecuteCommand(cmd, "")
			if err != nil {
				t.Logf("ExecuteCommand failed: %v", err)
				return false
			}

			// Verify stdout contains the stdout message
			if !strings.Contains(result.Stdout, stdoutMsg) {
				t.Logf("Stdout does not contain expected message: %q", stdoutMsg)
				return false
			}

			// Verify stderr contains the stderr message
			if !strings.Contains(result.Stderr, stderrMsg) {
				t.Logf("Stderr does not contain expected message: %q", stderrMsg)
				return false
			}

			return true
		},
		genSafeCommandMessage(),
		genSafeCommandMessage(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: Terminal Intelligence (TI), Property 8: Script Interpreter Selection
// **Validates: Requirements 4.4**
//
// For any script file with a recognized extension (.sh, .bash, .ps1),
// the CommandExecutor should select the appropriate interpreter for that file type.
func TestProperty_ScriptInterpreterSelection(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("sh files use bash or sh interpreter", prop.ForAll(
		func(filename string) bool {
			ce := executor.NewCommandExecutor()
			scriptPath := filename + ".sh"

			interpreter := ce.GetInterpreter(scriptPath)

			// On Windows, .sh files should use "sh"
			// On Unix systems, .sh files should use "bash"
			if runtime.GOOS == "windows" {
				return interpreter == "sh"
			}
			return interpreter == "bash"
		},
		gen.Identifier(),
	))

	properties.Property("bash files use bash interpreter", prop.ForAll(
		func(filename string) bool {
			ce := executor.NewCommandExecutor()
			scriptPath := filename + ".bash"

			interpreter := ce.GetInterpreter(scriptPath)

			return interpreter == "bash"
		},
		gen.Identifier(),
	))

	properties.Property("ps1 files use powershell or pwsh interpreter", prop.ForAll(
		func(filename string) bool {
			ce := executor.NewCommandExecutor()
			scriptPath := filename + ".ps1"

			interpreter := ce.GetInterpreter(scriptPath)

			// On Windows, .ps1 files should use "powershell"
			// On Unix systems, .ps1 files should use "pwsh" (PowerShell Core)
			if runtime.GOOS == "windows" {
				return interpreter == "powershell"
			}
			return interpreter == "pwsh"
		},
		gen.Identifier(),
	))

	properties.Property("unsupported extensions return empty string", prop.ForAll(
		func(filename string, ext string) bool {
			ce := executor.NewCommandExecutor()

			// Use extensions that are not supported script runners
			scriptPath := filename + "." + ext

			interpreter := ce.GetInterpreter(scriptPath)

			return interpreter == ""
		},
		gen.Identifier(),
		gen.OneConstOf("txt", "js", "md", "json", "xml", "html"),
	))

	properties.Property("case insensitive extension matching", prop.ForAll(
		func(filename string, caseVariant int) bool {
			ce := executor.NewCommandExecutor()

			// Test different case variants of .sh extension
			var scriptPath string
			switch caseVariant % 4 {
			case 0:
				scriptPath = filename + ".sh"
			case 1:
				scriptPath = filename + ".SH"
			case 2:
				scriptPath = filename + ".Sh"
			case 3:
				scriptPath = filename + ".sH"
			}

			interpreter := ce.GetInterpreter(scriptPath)

			// Should still recognize the extension regardless of case
			if runtime.GOOS == "windows" {
				return interpreter == "sh"
			}
			return interpreter == "bash"
		},
		gen.Identifier(),
		gen.IntRange(0, 100),
	))

	properties.Property("paths with directories handled correctly", prop.ForAll(
		func(dir string, filename string, ext string) bool {
			ce := executor.NewCommandExecutor()

			// Create a path with directory components
			scriptPath := dir + "/" + filename + ext

			interpreter := ce.GetInterpreter(scriptPath)

			// Verify interpreter is selected based on extension, not path
			switch ext {
			case ".sh":
				if runtime.GOOS == "windows" {
					return interpreter == "sh"
				}
				return interpreter == "bash"
			case ".bash":
				return interpreter == "bash"
			case ".ps1":
				if runtime.GOOS == "windows" {
					return interpreter == "powershell"
				}
				return interpreter == "pwsh"
			case ".py":
				return interpreter == "python3"
			case ".go":
				return interpreter == "go"
			default:
				return interpreter == ""
			}
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.OneConstOf(".sh", ".bash", ".ps1", ".txt", ".py", ".go"),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genSafeCommandMessage generates safe strings for command messages
// Avoids special characters that could cause shell injection or parsing issues
func genSafeCommandMessage() gopter.Gen {
	return gen.OneGenOf(
		// Simple alphanumeric strings
		gen.Identifier(),
		// Words with spaces
		gen.SliceOfN(3, gen.Identifier()).Map(func(words []string) string {
			return strings.Join(words, " ")
		}),
		// Common test messages
		gen.OneConstOf(
			"hello",
			"test message",
			"output",
			"error message",
			"command result",
			"success",
			"failure",
		),
	)
}

// Feature: Terminal Intelligence (TI), Property 9: Command Exit Code Capture
// **Validates: Requirements 4.5, 4.6**
//
// For any command or script execution, the captured exit code should match
// the actual exit code returned by the process.
func TestProperty_CommandExitCodeCapture(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("successful commands return exit code 0", prop.ForAll(
		func(message string) bool {
			ce := executor.NewCommandExecutor()

			// Create a simple command that should succeed
			var cmd string
			if runtime.GOOS == "windows" {
				cmd = "echo " + message
			} else {
				cmd = "echo " + message
			}

			// Execute through CommandExecutor
			result, err := ce.ExecuteCommand(cmd, "")
			if err != nil {
				t.Logf("ExecuteCommand failed: %v", err)
				return false
			}

			// Successful commands should have exit code 0
			if result.ExitCode != 0 {
				t.Logf("Expected exit code 0, got %d", result.ExitCode)
				return false
			}

			return true
		},
		genSafeCommandMessage(),
	))

	properties.Property("failing commands return non-zero exit code", prop.ForAll(
		func(exitCode int) bool {
			ce := executor.NewCommandExecutor()

			// Create a command that exits with a specific code
			var cmd string
			if runtime.GOOS == "windows" {
				cmd = "cmd /c exit " + strconv.Itoa(exitCode)
			} else {
				cmd = "sh -c 'exit " + strconv.Itoa(exitCode) + "'"
			}

			// Execute through CommandExecutor
			result, err := ce.ExecuteCommand(cmd, "")
			if err != nil {
				t.Logf("ExecuteCommand failed: %v", err)
				return false
			}

			// Exit code should match the requested exit code
			if result.ExitCode != exitCode {
				t.Logf("Expected exit code %d, got %d", exitCode, result.ExitCode)
				return false
			}

			return true
		},
		gen.IntRange(1, 255), // Exit codes are typically 0-255
	))

	properties.Property("command not found returns non-zero exit code", prop.ForAll(
		func(invalidCmd string) bool {
			ce := executor.NewCommandExecutor()

			// Create a command that doesn't exist
			cmd := invalidCmd + "_nonexistent_command_12345"

			// Execute through CommandExecutor
			result, err := ce.ExecuteCommand(cmd, "")
			if err != nil {
				t.Logf("ExecuteCommand failed: %v", err)
				return false
			}

			// Command not found should have non-zero exit code
			if result.ExitCode == 0 {
				t.Logf("Expected non-zero exit code for command not found, got 0")
				return false
			}

			return true
		},
		gen.Identifier(),
	))

	properties.Property("script execution captures exit code correctly", prop.ForAll(
		func(exitCode int) bool {
			ce := executor.NewCommandExecutor()

			// Create a temporary script file that exits with a specific code
			var scriptContent string
			var scriptPath string
			if runtime.GOOS == "windows" {
				scriptPath = "test_exit_" + strconv.Itoa(exitCode) + ".ps1"
				scriptContent = "exit " + strconv.Itoa(exitCode)
			} else {
				scriptPath = "test_exit_" + strconv.Itoa(exitCode) + ".sh"
				scriptContent = "#!/bin/bash\nexit " + strconv.Itoa(exitCode)
			}

			// Write script file
			err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
			if err != nil {
				t.Logf("Failed to create script file: %v", err)
				return false
			}
			defer os.Remove(scriptPath)

			// Execute script through CommandExecutor
			result, err := ce.ExecuteScript(scriptPath)
			if err != nil {
				t.Logf("ExecuteScript failed: %v", err)
				return false
			}

			// Exit code should match the script's exit code
			if result.ExitCode != exitCode {
				t.Logf("Expected exit code %d, got %d", exitCode, result.ExitCode)
				return false
			}

			return true
		},
		gen.IntRange(0, 255),
	))

	properties.Property("exit code matches direct execution", prop.ForAll(
		func(exitCode int) bool {
			ce := executor.NewCommandExecutor()

			// Create a command that exits with a specific code
			var cmd string
			if runtime.GOOS == "windows" {
				cmd = "cmd /c exit " + strconv.Itoa(exitCode)
			} else {
				cmd = "sh -c 'exit " + strconv.Itoa(exitCode) + "'"
			}

			// Execute through CommandExecutor
			result, err := ce.ExecuteCommand(cmd, "")
			if err != nil {
				t.Logf("ExecuteCommand failed: %v", err)
				return false
			}

			// Execute directly to compare
			var directCmd *exec.Cmd
			if runtime.GOOS == "windows" {
				directCmd = exec.Command("cmd", "/c", cmd)
			} else {
				directCmd = exec.Command("sh", "-c", cmd)
			}

			directErr := directCmd.Run()
			directExitCode := 0
			if directErr != nil {
				if exitErr, ok := directErr.(*exec.ExitError); ok {
					directExitCode = exitErr.ExitCode()
				}
			}

			// Exit codes should match
			if result.ExitCode != directExitCode {
				t.Logf("Exit code mismatch: captured=%d, direct=%d", result.ExitCode, directExitCode)
				return false
			}

			return true
		},
		gen.IntRange(0, 255),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
