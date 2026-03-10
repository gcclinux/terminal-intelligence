package validation

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Validator defines the interface for language-specific validators
type Validator interface {
	// Execute runs validation for the given files and returns the result
	Execute(files []string) (ValidationResult, error)

	// GetInfo returns information about the validator
	GetInfo() ValidatorInfo
}

// BaseValidator provides common functionality for validators
type BaseValidator struct {
	config ValidatorConfig
	info   ValidatorInfo
}

// NewBaseValidator creates a new BaseValidator with the given configuration
func NewBaseValidator(config ValidatorConfig, info ValidatorInfo) *BaseValidator {
	return &BaseValidator{
		config: config,
		info:   info,
	}
}

// GetInfo returns information about the validator
func (bv *BaseValidator) GetInfo() ValidatorInfo {
	return bv.info
}

// ExecuteCommand runs a command with timeout and captures stdout/stderr
// Returns the combined output, exit code, and any error
func (bv *BaseValidator) ExecuteCommand(ctx context.Context, command string, args []string, workingDir string) (stdout string, stderr string, exitCode int, err error) {
	// Create context with timeout if specified
	if bv.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, bv.config.Timeout)
		defer cancel()
	}

	// Create command
	cmd := exec.CommandContext(ctx, command, args...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Create buffers for stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Execute command
	err = cmd.Run()

	// Capture output
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	// Determine exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			// Timeout occurred
			return stdout, stderr, -1, fmt.Errorf("command timed out after %v", bv.config.Timeout)
		} else {
			// Command failed to start or other error
			return stdout, stderr, -1, err
		}
	} else {
		exitCode = 0
	}

	return stdout, stderr, exitCode, nil
}

// CreateValidationResult creates a ValidationResult from command execution output
func (bv *BaseValidator) CreateValidationResult(
	language Language,
	files []string,
	duration time.Duration,
	stdout string,
	stderr string,
	exitCode int,
	errors []ValidationError,
	warnings []ValidationError,
) ValidationResult {
	// Combine stdout and stderr for the output field
	output := stdout
	if stderr != "" {
		if output != "" {
			output += "\n"
		}
		output += stderr
	}

	return ValidationResult{
		Success:  exitCode == 0 && len(errors) == 0,
		Language: language,
		Files:    files,
		Duration: duration,
		Output:   output,
		Errors:   errors,
		Warnings: warnings,
	}
}
