package validation

import (
	"context"
	"fmt"
	"sync"
)

// CompilerInterface manages language-specific validators and delegates validation
type CompilerInterface struct {
	validators map[Language]Validator
	mu         sync.RWMutex
}

// NewCompilerInterface creates a new CompilerInterface with default validators registered
func NewCompilerInterface() *CompilerInterface {
	ci := &CompilerInterface{
		validators: make(map[Language]Validator),
	}

	// Register default validators
	ci.registerValidator(LanguageGo, NewGoValidator())
	ci.registerValidator(LanguagePython, NewPythonValidator())

	return ci
}

// RegisterValidator registers a validator for a specific language
// This method is exported for external use
func (ci *CompilerInterface) RegisterValidator(language Language, validator Validator) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.registerValidator(language, validator)
}

// registerValidator is the internal implementation without locking
func (ci *CompilerInterface) registerValidator(language Language, validator Validator) {
	ci.validators[language] = validator
}

// GetValidator retrieves the validator for a specific language
// Returns nil if no validator is registered for the language
func (ci *CompilerInterface) GetValidator(language Language) Validator {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return ci.validators[language]
}

// Validate executes validation for the given files using the appropriate language validator
// Returns an error if the language is not supported or if validation fails to execute
func (ci *CompilerInterface) Validate(language Language, files []string) (ValidationResult, error) {
	if len(files) == 0 {
		return ValidationResult{}, fmt.Errorf("no files provided for validation")
	}

	// Get the validator for the language
	validator := ci.GetValidator(language)
	if validator == nil {
		return ValidationResult{}, fmt.Errorf("no validator registered for language: %s", language)
	}

	// Delegate to the language-specific validator
	return validator.Execute(files)
}

// ValidateWithContext executes validation with a context for cancellation support
func (ci *CompilerInterface) ValidateWithContext(ctx context.Context, language Language, files []string) (ValidationResult, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return ValidationResult{}, ctx.Err()
	default:
	}

	// For now, delegate to the regular Validate method
	// Future enhancement: pass context to validators for cancellation support
	return ci.Validate(language, files)
}
