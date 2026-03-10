package validation

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PythonValidator implements validation for Python code
type PythonValidator struct {
	*BaseValidator
	errorPattern *regexp.Regexp
}

// NewPythonValidator creates a new Python validator
func NewPythonValidator() *PythonValidator {
	config := ValidatorConfig{
		Command:      "python",
		Args:         []string{"-m", "py_compile"},
		Timeout:      10 * time.Second,
		ErrorPattern: `File "(.+)", line (\d+)`,
	}

	info := ValidatorInfo{
		Name:    "Python",
		Version: "1.0.0",
		Command: "python",
	}

	// Compile the error pattern - matches both formats:
	// 1. File "path", line N
	// 2. (path, line N) - for IndentationError format
	errorPattern := regexp.MustCompile(config.ErrorPattern)

	return &PythonValidator{
		BaseValidator: NewBaseValidator(config, info),
		errorPattern:  errorPattern,
	}
}

// Execute runs Python syntax validation for the given files
// Each file is validated independently
func (pv *PythonValidator) Execute(files []string) (ValidationResult, error) {
	if len(files) == 0 {
		return ValidationResult{}, fmt.Errorf("no files provided for validation")
	}

	ctx := context.Background()
	startTime := time.Now()

	var allErrors []ValidationError
	var allWarnings []ValidationError
	var combinedStdout strings.Builder
	var combinedStderr strings.Builder
	overallExitCode := 0

	// Validate each file independently
	for _, file := range files {
		// Build args with the file path
		args := append(pv.config.Args, file)

		stdout, stderr, exitCode, err := pv.BaseValidator.ExecuteCommand(
			ctx,
			pv.config.Command,
			args,
			"", // No specific working directory needed
		)

		// Accumulate output
		if stdout != "" {
			combinedStdout.WriteString(stdout)
			combinedStdout.WriteString("\n")
		}
		if stderr != "" {
			combinedStderr.WriteString(stderr)
			combinedStderr.WriteString("\n")
		}

		// Track the worst exit code
		if exitCode != 0 {
			overallExitCode = exitCode
		}

		// Handle command execution errors (timeout, command not found, etc.)
		if err != nil {
			return ValidationResult{}, fmt.Errorf("failed to execute python validation for %s: %w", file, err)
		}

		// Parse errors from stderr for this file
		errors, warnings := pv.parseErrors(stderr, file)
		allErrors = append(allErrors, errors...)
		allWarnings = append(allWarnings, warnings...)
	}

	duration := time.Since(startTime)

	// Create and return the validation result
	result := pv.BaseValidator.CreateValidationResult(
		LanguagePython,
		files,
		duration,
		combinedStdout.String(),
		combinedStderr.String(),
		overallExitCode,
		allErrors,
		allWarnings,
	)

	return result, nil
}

// parseErrors parses Python error output and extracts ValidationError objects
// Handles two formats:
// 1. File "path", line N (standard format)
// 2. Sorry: ErrorType: message (path, line N) (IndentationError format)
func (pv *PythonValidator) parseErrors(output string, _ string) ([]ValidationError, []ValidationError) {
	var errors []ValidationError
	var warnings []ValidationError

	if output == "" {
		return errors, warnings
	}

	lines := strings.Split(output, "\n")

	// Pattern for IndentationError format: (path, line N)
	indentPattern := regexp.MustCompile(`\((.+),\s*line\s+(\d+)\)`)

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		var file string
		var lineNum int
		var message string
		matched := false

		// Try to match the standard error pattern: File "(.+)", line (\d+)
		matches := pv.errorPattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			// Extract file and line number
			file = matches[1]
			lineNum, _ = strconv.Atoi(matches[2])
			matched = true

			// The error message typically follows on the next line(s)
			// Collect the error message from subsequent lines
			var messageLines []string
			for j := i + 1; j < len(lines); j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" {
					break
				}
				// Stop if we hit another "File" line
				if strings.HasPrefix(nextLine, "File \"") {
					break
				}
				messageLines = append(messageLines, nextLine)
			}

			if len(messageLines) > 0 {
				message = strings.Join(messageLines, " ")
			} else {
				message = line
			}
		} else {
			// Try to match IndentationError format: (path, line N)
			indentMatches := indentPattern.FindStringSubmatch(line)
			if len(indentMatches) == 3 {
				file = indentMatches[1]
				lineNum, _ = strconv.Atoi(indentMatches[2])
				matched = true

				// Extract the error message (everything before the file path)
				idx := strings.Index(line, "(")
				if idx > 0 {
					message = strings.TrimSpace(line[:idx])
					// Remove "Sorry: " prefix if present
					message = strings.TrimPrefix(message, "Sorry: ")
				} else {
					message = line
				}
			}
		}

		if matched {
			// Determine severity - Python typically reports errors, not warnings
			severity := SeverityError
			if strings.Contains(strings.ToLower(message), "warning") {
				severity = SeverityWarning
			}

			validationError := ValidationError{
				File:     file,
				Line:     lineNum,
				Column:   0, // Python error format doesn't always include column
				Message:  message,
				Severity: severity,
			}

			if severity == SeverityWarning {
				warnings = append(warnings, validationError)
			} else {
				errors = append(errors, validationError)
			}
		}
	}

	return errors, warnings
}
