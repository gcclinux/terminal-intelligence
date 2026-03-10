package validation

import "time"

// Operation represents the type of file operation
type Operation string

const (
	OperationCreate Operation = "create"
	OperationModify Operation = "modify"
	OperationDelete Operation = "delete"
)

// FileChangeEvent represents a file modification event from the AI
type FileChangeEvent struct {
	FilePath  string    // Absolute or workspace-relative path
	Operation Operation // Type of file operation
	Timestamp time.Time // Time of the event
}

// ValidationStatus represents the current status of a validation session
type ValidationStatus string

const (
	StatusPending   ValidationStatus = "pending"
	StatusRunning   ValidationStatus = "running"
	StatusCompleted ValidationStatus = "completed"
	StatusFailed    ValidationStatus = "failed"
	StatusCancelled ValidationStatus = "cancelled"
)

// ValidationSession tracks a validation operation across multiple files
type ValidationSession struct {
	ID        string             // Unique session identifier
	Files     []string           // Files being validated
	StartTime time.Time          // Start timestamp
	EndTime   *time.Time         // End timestamp (when completed)
	Status    ValidationStatus   // Current status
	Results   []ValidationResult // Results per language
}

// Language represents a programming language
type Language string

const (
	LanguageGo          Language = "go"
	LanguagePython      Language = "python"
	LanguageUnsupported Language = "unsupported"
)

// ValidationResult contains the outcome of validating files in a specific language
type ValidationResult struct {
	Success  bool              // Overall validation success
	Language Language          // Programming language
	Files    []string          // Files validated
	Duration time.Duration     // Validation time
	Output   string            // Raw command output
	Errors   []ValidationError // Parsed errors (empty if success)
	Warnings []ValidationError // Parsed warnings
}

// Severity represents the severity level of a validation error
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// ValidationError represents a single compilation or validation error
type ValidationError struct {
	File     string   // File path where error occurred
	Line     int      // Line number (1-indexed)
	Column   int      // Column number (1-indexed, 0 if not available)
	Message  string   // Error message from compiler/validator
	Severity Severity // Error severity level
	Code     string   // Error code (if provided by validator)
}

// LanguageInfo defines metadata about a programming language
type LanguageInfo struct {
	Name       string   // Display name
	Extensions []string // File extensions (with dot)
	Validator  string   // Validator identifier
}

// ValidatorInfo contains information about a validator
type ValidatorInfo struct {
	Name    string // Validator name
	Version string // Validator version
	Command string // Base command
}

// ValidatorConfig defines how to validate a specific programming language
type ValidatorConfig struct {
	Command      string        // Base command to execute
	Args         []string      // Command arguments template
	Cwd          string        // Working directory (relative to workspace)
	Timeout      time.Duration // Timeout duration
	ErrorPattern string        // Regex to parse errors
}

// LanguageConfig defines configuration for a programming language
type LanguageConfig struct {
	Name       string          // Display name
	Extensions []string        // File extensions (with dot)
	Validator  ValidatorConfig // Validator configuration
}
