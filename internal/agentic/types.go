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
//  1. Create backup before applying changes
//  2. Apply changes to temporary copy
//  3. Validate the result
//  4. Commit or rollback based on validation
//
// This ensures that original content is always preserved on error, maintaining system consistency.
package agentic

import (
	"fmt"
	"strings"
	"time"
)

// FixRequest represents a request to fix code
// Invariants:
// - UserMessage must not be empty
// - FileContent may be empty (for new files)
// - FilePath must be a valid path string
// - FileType must be one of: "bash", "shell", "powershell", "markdown", "python", "go"
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
		"python":     true,
		"go":         true,
	}

	if !validFileTypes[fr.FileType] {
		return fmt.Errorf("FileType must be one of: bash, shell, powershell, markdown, python, go; got: %s", fr.FileType)
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
	InputTokens      int
	OutputTokens     int
	TotalTokens      int
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

// FileResult records the outcome of editing a single file.
type FileResult struct {
	Path         string
	RelPath      string
	LinesAdded   int
	LinesRemoved int
}

// PatchFailure records a file whose patch could not be applied.
type PatchFailure struct {
	Path   string
	Reason string
}

// ChangeReport summarises the outcome of a project-wide agentic operation.
type ChangeReport struct {
	FilesRead         []string
	FilesModified     []FileResult
	FilesUnreadable   []string
	HallucinatedPaths []string
	OutOfScopePaths   []string
	PatchFailures     []PatchFailure
	TruncationWarning string // set when >500 files were found
	PreviewMode       bool
	InputTokens       int
	OutputTokens      int
	TotalTokens       int
}

// ProjectFixRequest is the input to a project-wide fix operation.
type ProjectFixRequest struct {
	Message     string
	ProjectRoot string
	PreviewMode bool
}

// FixSession holds all state for a single /fix invocation.
// Invariants:
// - OriginalAsk must not be empty
// - StartTime must not be zero
// - Snapshots must not be nil
type FixSession struct {
	OriginalAsk    string
	StartTime      time.Time
	Attempts       []FixAttempt
	Snapshots      map[string][]byte
	CurrentCycle   int
	AttemptInCycle int
	RankedFiles    []string
}

// Validate checks if the FixSession satisfies its invariants.
func (fs *FixSession) Validate() error {
	if strings.TrimSpace(fs.OriginalAsk) == "" {
		return fmt.Errorf("OriginalAsk must not be empty")
	}
	if fs.StartTime.IsZero() {
		return fmt.Errorf("StartTime must not be zero")
	}
	if fs.Snapshots == nil {
		return fmt.Errorf("Snapshots must not be nil")
	}
	return nil
}

// FixAttempt records a single attempt within a fix session.
// Invariants:
// - Number must be > 0
// - Timestamp must not be zero
type FixAttempt struct {
	Number         int
	Cycle          int
	Strategy       Strategy
	FilesModified  []FileResult
	PatchesApplied []searchReplacePatch
	TestCommand    string
	TestResult     *TestResult
	Timestamp      time.Time
}

// Strategy describes the approach taken in a fix attempt.
type Strategy struct {
	Description string
	Prompt      string
	AIResponse  string
}

// FixSessionRequest is the input to the agentic fixer.
// Invariants:
// - Message must not be empty
// - ProjectRoot must not be empty
// - MaxAttempts must be > 0
// - MaxCycles must be > 0
type FixSessionRequest struct {
	Message      string
	ProjectRoot  string
	OpenFilePath string
	MaxAttempts  int
	MaxCycles    int
}

// Validate checks if the FixSessionRequest satisfies its invariants.
func (r *FixSessionRequest) Validate() error {
	if strings.TrimSpace(r.Message) == "" {
		return fmt.Errorf("Message must not be empty")
	}
	if strings.TrimSpace(r.ProjectRoot) == "" {
		return fmt.Errorf("ProjectRoot must not be empty")
	}
	if r.MaxAttempts <= 0 {
		return fmt.Errorf("MaxAttempts must be > 0; got: %d", r.MaxAttempts)
	}
	if r.MaxCycles <= 0 {
		return fmt.Errorf("MaxCycles must be > 0; got: %d", r.MaxCycles)
	}
	return nil
}

// FixSessionResult is the output from the agentic fixer.
type FixSessionResult struct {
	Success       bool
	TotalAttempts int
	TotalCycles   int
	Attempts      []FixAttempt
	FinalReport   *ChangeReport
	ErrorMessage  string
	InputTokens   int
	OutputTokens  int
	TotalTokens   int
}

// TestResult captures the outcome of running a test command.
type TestResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
	TimedOut bool
}
