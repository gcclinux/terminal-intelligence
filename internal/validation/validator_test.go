package validation

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestBaseValidator_GetInfo(t *testing.T) {
	config := ValidatorConfig{
		Command: "test-command",
		Timeout: 5 * time.Second,
	}
	info := ValidatorInfo{
		Name:    "Test Validator",
		Version: "1.0.0",
		Command: "test-command",
	}

	validator := NewBaseValidator(config, info)
	result := validator.GetInfo()

	if result.Name != info.Name {
		t.Errorf("Expected name %s, got %s", info.Name, result.Name)
	}
	if result.Version != info.Version {
		t.Errorf("Expected version %s, got %s", info.Version, result.Version)
	}
	if result.Command != info.Command {
		t.Errorf("Expected command %s, got %s", info.Command, result.Command)
	}
}

func TestBaseValidator_ExecuteCommand_Success(t *testing.T) {
	config := ValidatorConfig{
		Command: "echo",
		Timeout: 5 * time.Second,
	}
	info := ValidatorInfo{
		Name:    "Echo Validator",
		Version: "1.0.0",
		Command: "echo",
	}

	validator := NewBaseValidator(config, info)
	ctx := context.Background()

	// Use platform-appropriate echo command
	var command string
	var args []string
	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/C", "echo", "test output"}
	} else {
		command = "echo"
		args = []string{"test output"}
	}

	stdout, stderr, exitCode, err := validator.ExecuteCommand(ctx, command, args, "")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if stdout == "" {
		t.Error("Expected stdout to contain output")
	}
	if stderr != "" {
		t.Errorf("Expected empty stderr, got %s", stderr)
	}
}

func TestBaseValidator_ExecuteCommand_Failure(t *testing.T) {
	config := ValidatorConfig{
		Command: "false",
		Timeout: 5 * time.Second,
	}
	info := ValidatorInfo{
		Name:    "False Validator",
		Version: "1.0.0",
		Command: "false",
	}

	validator := NewBaseValidator(config, info)
	ctx := context.Background()

	// Use platform-appropriate command that fails
	var command string
	var args []string
	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/C", "exit", "1"}
	} else {
		command = "false"
		args = []string{}
	}

	_, _, exitCode, err := validator.ExecuteCommand(ctx, command, args, "")

	if err != nil {
		t.Fatalf("Expected no error from ExecuteCommand, got %v", err)
	}
	if exitCode == 0 {
		t.Error("Expected non-zero exit code")
	}
}

func TestBaseValidator_ExecuteCommand_Timeout(t *testing.T) {
	// Skip on Windows as timeout behavior is inconsistent with different commands
	if runtime.GOOS == "windows" {
		t.Skip("Skipping timeout test on Windows due to platform-specific behavior")
	}

	config := ValidatorConfig{
		Command: "sleep",
		Timeout: 100 * time.Millisecond, // Very short timeout
	}
	info := ValidatorInfo{
		Name:    "Sleep Validator",
		Version: "1.0.0",
		Command: "sleep",
	}

	validator := NewBaseValidator(config, info)
	ctx := context.Background()

	command := "sleep"
	args := []string{"5"}

	start := time.Now()
	_, _, exitCode, err := validator.ExecuteCommand(ctx, command, args, "")
	elapsed := time.Since(start)

	// Verify timeout occurred (should be around 100ms, not 5 seconds)
	if elapsed > 1*time.Second {
		t.Errorf("Command took too long (%v), timeout may not have worked", elapsed)
	}

	if err == nil {
		t.Error("Expected timeout error")
	}
	if exitCode != -1 {
		t.Errorf("Expected exit code -1 for timeout, got %d", exitCode)
	}
}

func TestBaseValidator_ExecuteCommand_CommandNotFound(t *testing.T) {
	config := ValidatorConfig{
		Command: "nonexistent-command-12345",
		Timeout: 5 * time.Second,
	}
	info := ValidatorInfo{
		Name:    "Nonexistent Validator",
		Version: "1.0.0",
		Command: "nonexistent-command-12345",
	}

	validator := NewBaseValidator(config, info)
	ctx := context.Background()

	_, _, exitCode, err := validator.ExecuteCommand(ctx, "nonexistent-command-12345", []string{}, "")

	if err == nil {
		t.Error("Expected error for nonexistent command")
	}
	if exitCode != -1 {
		t.Errorf("Expected exit code -1 for command not found, got %d", exitCode)
	}
}

func TestBaseValidator_ExecuteCommand_StderrCapture(t *testing.T) {
	config := ValidatorConfig{
		Command: "test",
		Timeout: 5 * time.Second,
	}
	info := ValidatorInfo{
		Name:    "Test Validator",
		Version: "1.0.0",
		Command: "test",
	}

	validator := NewBaseValidator(config, info)
	ctx := context.Background()

	// Use platform-appropriate command that writes to stderr
	var command string
	var args []string
	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/C", "echo error message 1>&2"}
	} else {
		command = "sh"
		args = []string{"-c", "echo 'error message' >&2"}
	}

	stdout, stderr, exitCode, err := validator.ExecuteCommand(ctx, command, args, "")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
	if stderr == "" {
		t.Error("Expected stderr to contain output")
	}
	if stdout != "" {
		t.Logf("Stdout: %s", stdout)
	}
}

func TestBaseValidator_CreateValidationResult_Success(t *testing.T) {
	config := ValidatorConfig{
		Command: "test",
		Timeout: 5 * time.Second,
	}
	info := ValidatorInfo{
		Name:    "Test Validator",
		Version: "1.0.0",
		Command: "test",
	}

	validator := NewBaseValidator(config, info)
	files := []string{"test.go"}
	duration := 100 * time.Millisecond
	stdout := "compilation successful"
	stderr := ""
	exitCode := 0
	errors := []ValidationError{}
	warnings := []ValidationError{}

	result := validator.CreateValidationResult(
		LanguageGo,
		files,
		duration,
		stdout,
		stderr,
		exitCode,
		errors,
		warnings,
	)

	if !result.Success {
		t.Error("Expected success to be true")
	}
	if result.Language != LanguageGo {
		t.Errorf("Expected language Go, got %s", result.Language)
	}
	if len(result.Files) != 1 || result.Files[0] != "test.go" {
		t.Errorf("Expected files [test.go], got %v", result.Files)
	}
	if result.Duration != duration {
		t.Errorf("Expected duration %v, got %v", duration, result.Duration)
	}
	if result.Output != stdout {
		t.Errorf("Expected output %s, got %s", stdout, result.Output)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(result.Errors))
	}
}

func TestBaseValidator_CreateValidationResult_Failure(t *testing.T) {
	config := ValidatorConfig{
		Command: "test",
		Timeout: 5 * time.Second,
	}
	info := ValidatorInfo{
		Name:    "Test Validator",
		Version: "1.0.0",
		Command: "test",
	}

	validator := NewBaseValidator(config, info)
	files := []string{"test.go"}
	duration := 100 * time.Millisecond
	stdout := ""
	stderr := "compilation error"
	exitCode := 1
	errors := []ValidationError{
		{
			File:     "test.go",
			Line:     10,
			Column:   5,
			Message:  "undefined: fmt.Printl",
			Severity: SeverityError,
		},
	}
	warnings := []ValidationError{}

	result := validator.CreateValidationResult(
		LanguageGo,
		files,
		duration,
		stdout,
		stderr,
		exitCode,
		errors,
		warnings,
	)

	if result.Success {
		t.Error("Expected success to be false")
	}
	if result.Language != LanguageGo {
		t.Errorf("Expected language Go, got %s", result.Language)
	}
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
	if result.Output != stderr {
		t.Errorf("Expected output to be stderr, got %s", result.Output)
	}
}

func TestBaseValidator_CreateValidationResult_CombinedOutput(t *testing.T) {
	config := ValidatorConfig{
		Command: "test",
		Timeout: 5 * time.Second,
	}
	info := ValidatorInfo{
		Name:    "Test Validator",
		Version: "1.0.0",
		Command: "test",
	}

	validator := NewBaseValidator(config, info)
	files := []string{"test.go"}
	duration := 100 * time.Millisecond
	stdout := "some output"
	stderr := "some error"
	exitCode := 0
	errors := []ValidationError{}
	warnings := []ValidationError{}

	result := validator.CreateValidationResult(
		LanguageGo,
		files,
		duration,
		stdout,
		stderr,
		exitCode,
		errors,
		warnings,
	)

	expectedOutput := "some output\nsome error"
	if result.Output != expectedOutput {
		t.Errorf("Expected combined output %q, got %q", expectedOutput, result.Output)
	}
}

func TestBaseValidator_CreateValidationResult_SuccessWithWarnings(t *testing.T) {
	config := ValidatorConfig{
		Command: "test",
		Timeout: 5 * time.Second,
	}
	info := ValidatorInfo{
		Name:    "Test Validator",
		Version: "1.0.0",
		Command: "test",
	}

	validator := NewBaseValidator(config, info)
	files := []string{"test.go"}
	duration := 100 * time.Millisecond
	stdout := "compilation successful with warnings"
	stderr := ""
	exitCode := 0
	errors := []ValidationError{}
	warnings := []ValidationError{
		{
			File:     "test.go",
			Line:     5,
			Column:   1,
			Message:  "unused variable: x",
			Severity: SeverityWarning,
		},
	}

	result := validator.CreateValidationResult(
		LanguageGo,
		files,
		duration,
		stdout,
		stderr,
		exitCode,
		errors,
		warnings,
	)

	if !result.Success {
		t.Error("Expected success to be true (warnings don't fail validation)")
	}
	if len(result.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(result.Warnings))
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(result.Errors))
	}
}
