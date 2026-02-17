// Package agentic provides autonomous AI code fixing capabilities for the Terminal Intelligence (TI) application.
//
// This package implements the core functionality for detecting fix requests, generating code fixes
// through AI models, and applying those fixes to files. It transforms the AI assistant from a
// conversational tool into an autonomous coding partner that can read, analyze, and fix code issues
// directly in the editor.
//
// Key Components:
//   - AgenticCodeFixer: Main orchestrator for the fix workflow
//   - FixParser: Extracts and validates code fixes from AI responses
//   - Type definitions: FixRequest, FixResult, CodeBlock, FixDetectionResult
//
// Error Handling Strategy:
// The package follows a transactional approach to error handling:
//   1. Create backup before applying changes
//   2. Apply changes to temporary copy
//   3. Validate the result
//   4. Commit or rollback based on validation
//
// This ensures that original content is always preserved on error, maintaining system consistency.
package agentic

import (
	"fmt"
	"strings"
)

// FixRequest represents a request to fix code
// Invariants:
// - UserMessage must not be empty
// - FileContent may be empty (for new files)
// - FilePath must be a valid path string
// - FileType must be one of: "bash", "shell", "powershell", "markdown"
// - PreviewMode indicates whether to show changes without applying them
type FixRequest struct {
	UserMessage string
	FileContent string
	FilePath    string
	FileType    string
	PreviewMode bool
}

// Validate checks if the FixRequest satisfies its invariants
func (fr *FixRequest) Validate() error {
	if strings.TrimSpace(fr.UserMessage) == "" {
		return fmt.Errorf("UserMessage must not be empty")
	}
	
	if fr.FilePath == "" {
		return fmt.Errorf("FilePath must not be empty")
	}
	
	validFileTypes := map[string]bool{
		"bash":       true,
		"shell":      true,
		"powershell": true,
		"markdown":   true,
	}
	
	if !validFileTypes[fr.FileType] {
		return fmt.Errorf("FileType must be one of: bash, shell, powershell, markdown; got: %s", fr.FileType)
	}
	
	return nil
}

// FixResult represents the outcome of a fix operation
// Invariants:
// - If Success is true, ModifiedContent and ChangesSummary must not be empty
// - If Success is false, ErrorMessage must not be empty
// - If IsConversational is true, Success should be false
// - ModifiedContent should only be set when Success is true
// - PreviewMode indicates this is a preview without actual modification
type FixResult struct {
	Success          bool
	ModifiedContent  string
	ChangesSummary   string
	ErrorMessage     string
	IsConversational bool
	PreviewMode      bool
}

// Validate checks if the FixResult satisfies its invariants
func (fr *FixResult) Validate() error {
	if fr.Success {
		if strings.TrimSpace(fr.ModifiedContent) == "" {
			return fmt.Errorf("ModifiedContent must not be empty when Success is true")
		}
		if strings.TrimSpace(fr.ChangesSummary) == "" {
			return fmt.Errorf("ChangesSummary must not be empty when Success is true")
		}
		if fr.IsConversational {
			return fmt.Errorf("IsConversational should be false when Success is true")
		}
	} else {
		if !fr.IsConversational && strings.TrimSpace(fr.ErrorMessage) == "" {
			return fmt.Errorf("ErrorMessage must not be empty when Success is false and not conversational")
		}
		if fr.ModifiedContent != "" {
			return fmt.Errorf("ModifiedContent should be empty when Success is false")
		}
	}
	
	return nil
}

// CodeBlock represents a code block extracted from an AI response
// Invariants:
// - Code must not be empty
// - Language may be empty (for unspecified language)
// - IsWhole determines replacement strategy
type CodeBlock struct {
	Language string
	Code     string
	IsWhole  bool
}

// Validate checks if the CodeBlock satisfies its invariants
func (cb *CodeBlock) Validate() error {
	if strings.TrimSpace(cb.Code) == "" {
		return fmt.Errorf("Code must not be empty")
	}
	
	return nil
}

// FixDetectionResult represents the result of fix request detection
// Invariants:
// - Confidence must be between 0.0 and 1.0
// - If IsFixRequest is true, Confidence should be >= 0.7
// - Keywords should contain at least one keyword if IsFixRequest is true
type FixDetectionResult struct {
	IsFixRequest bool
	Confidence   float64
	Keywords     []string
}

// Validate checks if the FixDetectionResult satisfies its invariants
func (fdr *FixDetectionResult) Validate() error {
	if fdr.Confidence < 0.0 || fdr.Confidence > 1.0 {
		return fmt.Errorf("Confidence must be between 0.0 and 1.0; got: %f", fdr.Confidence)
	}
	
	if fdr.IsFixRequest {
		if fdr.Confidence < 0.7 {
			return fmt.Errorf("Confidence should be >= 0.7 when IsFixRequest is true; got: %f", fdr.Confidence)
		}
		if len(fdr.Keywords) == 0 {
			return fmt.Errorf("Keywords should contain at least one keyword when IsFixRequest is true")
		}
	}
	
	return nil
}
