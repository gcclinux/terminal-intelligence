package executor

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/user/terminal-intelligence/internal/types"
)

// CommandExecutor handles command and script execution
type CommandExecutor struct{}

// NewCommandExecutor creates a new command executor
func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{}
}

// ExecuteCommand executes a system command
// Args:
//
//	command: string - command string to execute
//	cwd: string - optional working directory for command execution
//
// Returns: CommandResult with stdout, stderr, and exit code
func (ce *CommandExecutor) ExecuteCommand(command string, cwd string) (*types.CommandResult, error) {
	startTime := time.Now()

	// Validate command
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, fmt.Errorf("empty command")
	}

	// Create command using shell to properly handle complex commands
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-NoProfile", "-Command", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	// Set working directory if provided
	if cwd != "" {
		cmd.Dir = cwd
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()
	executionTime := time.Since(startTime)

	// Determine exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start or other error
			return nil, fmt.Errorf("command execution failed: %w", err)
		}
	}

	result := &types.CommandResult{
		Stdout:        stdout.String(),
		Stderr:        stderr.String(),
		ExitCode:      exitCode,
		ExecutionTime: executionTime,
	}

	return result, nil
}

// ExecuteScript executes a script file with appropriate interpreter
// Args:
//
//	scriptPath: string - path to the script file
//
// Returns: CommandResult with stdout, stderr, and exit code
func (ce *CommandExecutor) ExecuteScript(scriptPath string) (*types.CommandResult, error) {
	// Get the appropriate interpreter
	interpreter := ce.GetInterpreter(scriptPath)
	if interpreter == "" {
		return nil, fmt.Errorf("unsupported script type: %s", scriptPath)
	}

	startTime := time.Now()

	// Create command with interpreter
	cmd := exec.Command(interpreter, scriptPath)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute script
	err := cmd.Run()
	executionTime := time.Since(startTime)

	// Determine exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Script failed to start or other error
			return nil, fmt.Errorf("script execution failed: %w", err)
		}
	}

	result := &types.CommandResult{
		Stdout:        stdout.String(),
		Stderr:        stderr.String(),
		ExitCode:      exitCode,
		ExecutionTime: executionTime,
	}

	return result, nil
}

// GetInterpreter determines the appropriate interpreter for a script
// Args:
//
//	scriptPath: string - path to the script file
//
// Returns: interpreter command (e.g., "bash", "sh", "powershell", "go")
func (ce *CommandExecutor) GetInterpreter(scriptPath string) string {
	ext := strings.ToLower(filepath.Ext(scriptPath))

	switch ext {
	case ".sh":
		// For .sh files, prefer bash if available, otherwise sh
		if runtime.GOOS == "windows" {
			// On Windows, use sh from Git Bash or WSL if available
			return "sh"
		}
		return "bash"
	case ".bash":
		return "bash"
	case ".ps1":
		// PowerShell script
		if runtime.GOOS == "windows" {
			return "powershell"
		}
		// On Unix systems, use pwsh (PowerShell Core) if available
		return "pwsh"
	case ".py":
		// Python script
		return "python3"
	case ".go":
		// Go source file - use "go run" for execution
		return "go"
	default:
		// Unsupported script type
		return ""
	}
}
