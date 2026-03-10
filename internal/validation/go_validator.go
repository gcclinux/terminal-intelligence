package validation

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GoValidator implements validation for Go code
type GoValidator struct {
	*BaseValidator
	errorPattern *regexp.Regexp
}

// NewGoValidator creates a new Go validator
func NewGoValidator() *GoValidator {
	config := ValidatorConfig{
		Command:      "go",
		Args:         []string{"build", "./..."},
		Timeout:      30 * time.Second,
		ErrorPattern: `^(.+):(\d+):(\d+): (.+)$`,
	}

	info := ValidatorInfo{
		Name:    "Go",
		Version: "1.0.0",
		Command: "go",
	}

	// Compile the error pattern
	errorPattern := regexp.MustCompile(config.ErrorPattern)

	return &GoValidator{
		BaseValidator: NewBaseValidator(config, info),
		errorPattern:  errorPattern,
	}
}

// Execute runs Go compilation for the given files
func (gv *GoValidator) Execute(files []string) (ValidationResult, error) {
	if len(files) == 0 {
		return ValidationResult{}, fmt.Errorf("no files provided for validation")
	}

	ctx := context.Background()
	startTime := time.Now()

	// Determine the package directory from the first file
	packageDir, err := gv.getPackageDirectory(ctx, files[0])
	if err != nil {
		return ValidationResult{}, fmt.Errorf("failed to determine package directory: %w", err)
	}

	// Execute go build in the package directory
	stdout, stderr, exitCode, err := gv.BaseValidator.ExecuteCommand(
		ctx,
		gv.config.Command,
		gv.config.Args,
		packageDir,
	)

	duration := time.Since(startTime)

	// Parse errors from the output
	errors, warnings := gv.parseErrors(stderr)

	// Create and return the validation result
	result := gv.BaseValidator.CreateValidationResult(
		LanguageGo,
		files,
		duration,
		stdout,
		stderr,
		exitCode,
		errors,
		warnings,
	)

	return result, nil
}

// getPackageDirectory uses `go list` to determine the package directory for a file
func (gv *GoValidator) getPackageDirectory(ctx context.Context, filePath string) (string, error) {
	// Get the directory containing the file
	fileDir := filepath.Dir(filePath)

	// Use go list to get the package directory
	cmd := exec.CommandContext(ctx, "go", "list", "-f", "{{.Dir}}", ".")
	cmd.Dir = fileDir

	output, err := cmd.Output()
	if err != nil {
		// If go list fails, fall back to the file's directory
		return fileDir, nil
	}

	packageDir := strings.TrimSpace(string(output))
	if packageDir == "" {
		return fileDir, nil
	}

	return packageDir, nil
}

// parseErrors parses Go compiler error output and extracts ValidationError objects
func (gv *GoValidator) parseErrors(output string) ([]ValidationError, []ValidationError) {
	var errors []ValidationError
	var warnings []ValidationError

	if output == "" {
		return errors, warnings
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to match the error pattern
		matches := gv.errorPattern.FindStringSubmatch(line)
		if len(matches) == 5 {
			// Extract components
			file := matches[1]
			lineNum, _ := strconv.Atoi(matches[2])
			colNum, _ := strconv.Atoi(matches[3])
			message := matches[4]

			// Determine severity based on message content
			severity := SeverityError
			if strings.Contains(strings.ToLower(message), "warning") {
				severity = SeverityWarning
			}

			validationError := ValidationError{
				File:     file,
				Line:     lineNum,
				Column:   colNum,
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
