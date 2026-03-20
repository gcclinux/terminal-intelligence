package agentic

import (
	"testing"
	"time"
)

// TestFixRequest_Validate tests the validation of FixRequest
func TestFixRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request FixRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: FixRequest{
				UserMessage: "fix the bug",
				FileContent: "some code",
				FilePath:    "/path/to/file.sh",
				FileType:    "bash",
			},
			wantErr: false,
		},
		{
			name: "valid request with empty file content",
			request: FixRequest{
				UserMessage: "add new code",
				FileContent: "",
				FilePath:    "/path/to/file.sh",
				FileType:    "shell",
			},
			wantErr: false,
		},
		{
			name: "empty user message",
			request: FixRequest{
				UserMessage: "",
				FileContent: "some code",
				FilePath:    "/path/to/file.sh",
				FileType:    "bash",
			},
			wantErr: true,
		},
		{
			name: "whitespace-only user message",
			request: FixRequest{
				UserMessage: "   ",
				FileContent: "some code",
				FilePath:    "/path/to/file.sh",
				FileType:    "bash",
			},
			wantErr: true,
		},
		{
			name: "empty file path",
			request: FixRequest{
				UserMessage: "fix the bug",
				FileContent: "some code",
				FilePath:    "",
				FileType:    "bash",
			},
			wantErr: true,
		},
		{
			name: "valid python file type",
			request: FixRequest{
				UserMessage: "fix the bug",
				FileContent: "some code",
				FilePath:    "/path/to/file.py",
				FileType:    "python",
			},
			wantErr: false,
		},
		{
			name: "invalid file type",
			request: FixRequest{
				UserMessage: "fix the bug",
				FileContent: "some code",
				FilePath:    "/path/to/file.java",
				FileType:    "java",
			},
			wantErr: true,
		},
		{
			name: "valid powershell file type",
			request: FixRequest{
				UserMessage: "fix the bug",
				FileContent: "some code",
				FilePath:    "/path/to/file.ps1",
				FileType:    "powershell",
			},
			wantErr: false,
		},
		{
			name: "valid markdown file type",
			request: FixRequest{
				UserMessage: "fix the bug",
				FileContent: "some markdown",
				FilePath:    "/path/to/file.md",
				FileType:    "markdown",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FixRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFixResult_Validate tests the validation of FixResult
func TestFixResult_Validate(t *testing.T) {
	tests := []struct {
		name    string
		result  FixResult
		wantErr bool
	}{
		{
			name: "valid successful result",
			result: FixResult{
				Success:          true,
				ModifiedContent:  "fixed code",
				ChangesSummary:   "Fixed the bug",
				ErrorMessage:     "",
				IsConversational: false,
			},
			wantErr: false,
		},
		{
			name: "valid conversational result",
			result: FixResult{
				Success:          false,
				ModifiedContent:  "",
				ChangesSummary:   "",
				ErrorMessage:     "",
				IsConversational: true,
			},
			wantErr: false,
		},
		{
			name: "valid error result",
			result: FixResult{
				Success:          false,
				ModifiedContent:  "",
				ChangesSummary:   "",
				ErrorMessage:     "AI service unavailable",
				IsConversational: false,
			},
			wantErr: false,
		},
		{
			name: "success without modified content",
			result: FixResult{
				Success:          true,
				ModifiedContent:  "",
				ChangesSummary:   "Fixed the bug",
				ErrorMessage:     "",
				IsConversational: false,
			},
			wantErr: true,
		},
		{
			name: "success without changes summary",
			result: FixResult{
				Success:          true,
				ModifiedContent:  "fixed code",
				ChangesSummary:   "",
				ErrorMessage:     "",
				IsConversational: false,
			},
			wantErr: true,
		},
		{
			name: "success with conversational flag",
			result: FixResult{
				Success:          true,
				ModifiedContent:  "fixed code",
				ChangesSummary:   "Fixed the bug",
				ErrorMessage:     "",
				IsConversational: true,
			},
			wantErr: true,
		},
		{
			name: "failure with modified content",
			result: FixResult{
				Success:          false,
				ModifiedContent:  "some code",
				ChangesSummary:   "",
				ErrorMessage:     "Failed to apply",
				IsConversational: false,
			},
			wantErr: true,
		},
		{
			name: "failure without error message and not conversational",
			result: FixResult{
				Success:          false,
				ModifiedContent:  "",
				ChangesSummary:   "",
				ErrorMessage:     "",
				IsConversational: false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FixResult.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCodeBlock_Validate tests the validation of CodeBlock
func TestCodeBlock_Validate(t *testing.T) {
	tests := []struct {
		name    string
		block   CodeBlock
		wantErr bool
	}{
		{
			name: "valid code block with language",
			block: CodeBlock{
				Language: "bash",
				Code:     "echo 'hello'",
				IsWhole:  false,
			},
			wantErr: false,
		},
		{
			name: "valid code block without language",
			block: CodeBlock{
				Language: "",
				Code:     "some code",
				IsWhole:  true,
			},
			wantErr: false,
		},
		{
			name: "empty code",
			block: CodeBlock{
				Language: "bash",
				Code:     "",
				IsWhole:  false,
			},
			wantErr: true,
		},
		{
			name: "whitespace-only code",
			block: CodeBlock{
				Language: "bash",
				Code:     "   \n  \t  ",
				IsWhole:  false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.block.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CodeBlock.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFixDetectionResult_Validate tests the validation of FixDetectionResult
func TestFixDetectionResult_Validate(t *testing.T) {
	tests := []struct {
		name    string
		result  FixDetectionResult
		wantErr bool
	}{
		{
			name: "valid fix request detection",
			result: FixDetectionResult{
				IsFixRequest: true,
				Confidence:   0.9,
				Keywords:     []string{"fix", "change"},
			},
			wantErr: false,
		},
		{
			name: "valid non-fix request detection",
			result: FixDetectionResult{
				IsFixRequest: false,
				Confidence:   0.3,
				Keywords:     []string{},
			},
			wantErr: false,
		},
		{
			name: "confidence below 0",
			result: FixDetectionResult{
				IsFixRequest: false,
				Confidence:   -0.1,
				Keywords:     []string{},
			},
			wantErr: true,
		},
		{
			name: "confidence above 1",
			result: FixDetectionResult{
				IsFixRequest: true,
				Confidence:   1.5,
				Keywords:     []string{"fix"},
			},
			wantErr: true,
		},
		{
			name: "fix request with low confidence",
			result: FixDetectionResult{
				IsFixRequest: true,
				Confidence:   0.5,
				Keywords:     []string{"fix"},
			},
			wantErr: true,
		},
		{
			name: "fix request without keywords",
			result: FixDetectionResult{
				IsFixRequest: true,
				Confidence:   0.9,
				Keywords:     []string{},
			},
			wantErr: true,
		},
		{
			name: "fix request with exactly 0.7 confidence",
			result: FixDetectionResult{
				IsFixRequest: true,
				Confidence:   0.7,
				Keywords:     []string{"fix"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FixDetectionResult.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFixSession_Validate tests the validation of FixSession
func TestFixSession_Validate(t *testing.T) {
	tests := []struct {
		name    string
		session FixSession
		wantErr bool
	}{
		{
			name: "valid session",
			session: FixSession{
				OriginalAsk: "fix the tests",
				StartTime:   time.Now(),
				Snapshots:   map[string][]byte{},
			},
			wantErr: false,
		},
		{
			name: "empty original ask",
			session: FixSession{
				OriginalAsk: "",
				StartTime:   time.Now(),
				Snapshots:   map[string][]byte{},
			},
			wantErr: true,
		},
		{
			name: "whitespace-only original ask",
			session: FixSession{
				OriginalAsk: "   ",
				StartTime:   time.Now(),
				Snapshots:   map[string][]byte{},
			},
			wantErr: true,
		},
		{
			name: "zero start time",
			session: FixSession{
				OriginalAsk: "fix the tests",
				StartTime:   time.Time{},
				Snapshots:   map[string][]byte{},
			},
			wantErr: true,
		},
		{
			name: "nil snapshots",
			session: FixSession{
				OriginalAsk: "fix the tests",
				StartTime:   time.Now(),
				Snapshots:   nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FixSession.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFixSessionRequest_Validate tests the validation of FixSessionRequest
func TestFixSessionRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request FixSessionRequest
		wantErr bool
	}{
		{
			name: "valid request with defaults",
			request: FixSessionRequest{
				Message:     "fix the bug",
				ProjectRoot: "/home/user/project",
				MaxAttempts: 9,
				MaxCycles:   3,
			},
			wantErr: false,
		},
		{
			name: "valid request with open file",
			request: FixSessionRequest{
				Message:      "fix the bug",
				ProjectRoot:  "/home/user/project",
				OpenFilePath: "/home/user/project/main.go",
				MaxAttempts:  9,
				MaxCycles:    3,
			},
			wantErr: false,
		},
		{
			name: "empty message",
			request: FixSessionRequest{
				Message:     "",
				ProjectRoot: "/home/user/project",
				MaxAttempts: 9,
				MaxCycles:   3,
			},
			wantErr: true,
		},
		{
			name: "whitespace-only message",
			request: FixSessionRequest{
				Message:     "   ",
				ProjectRoot: "/home/user/project",
				MaxAttempts: 9,
				MaxCycles:   3,
			},
			wantErr: true,
		},
		{
			name: "empty project root",
			request: FixSessionRequest{
				Message:     "fix the bug",
				ProjectRoot: "",
				MaxAttempts: 9,
				MaxCycles:   3,
			},
			wantErr: true,
		},
		{
			name: "zero max attempts",
			request: FixSessionRequest{
				Message:     "fix the bug",
				ProjectRoot: "/home/user/project",
				MaxAttempts: 0,
				MaxCycles:   3,
			},
			wantErr: true,
		},
		{
			name: "negative max attempts",
			request: FixSessionRequest{
				Message:     "fix the bug",
				ProjectRoot: "/home/user/project",
				MaxAttempts: -1,
				MaxCycles:   3,
			},
			wantErr: true,
		},
		{
			name: "zero max cycles",
			request: FixSessionRequest{
				Message:     "fix the bug",
				ProjectRoot: "/home/user/project",
				MaxAttempts: 9,
				MaxCycles:   0,
			},
			wantErr: true,
		},
		{
			name: "negative max cycles",
			request: FixSessionRequest{
				Message:     "fix the bug",
				ProjectRoot: "/home/user/project",
				MaxAttempts: 9,
				MaxCycles:   -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FixSessionRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestEditIntent_Validate tests the validation of EditIntent
func TestEditIntent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		intent  EditIntent
		wantErr bool
	}{
		{
			name:    "valid replace intent",
			intent:  EditIntent{OperationType: "replace", Confidence: 0.9, Keywords: []string{"replace"}},
			wantErr: false,
		},
		{
			name:    "valid append intent",
			intent:  EditIntent{OperationType: "append", Confidence: 0.8, Keywords: []string{"add"}},
			wantErr: false,
		},
		{
			name:    "valid insert intent",
			intent:  EditIntent{OperationType: "insert", Confidence: 0.85, Keywords: []string{"insert"}},
			wantErr: false,
		},
		{
			name:    "valid patch intent",
			intent:  EditIntent{OperationType: "patch", Confidence: 1.0, Keywords: nil},
			wantErr: false,
		},
		{
			name:    "confidence at lower bound",
			intent:  EditIntent{OperationType: "patch", Confidence: 0.0},
			wantErr: false,
		},
		{
			name:    "confidence at upper bound",
			intent:  EditIntent{OperationType: "patch", Confidence: 1.0},
			wantErr: false,
		},
		{
			name:    "invalid operation type",
			intent:  EditIntent{OperationType: "delete", Confidence: 0.5},
			wantErr: true,
		},
		{
			name:    "empty operation type",
			intent:  EditIntent{OperationType: "", Confidence: 0.5},
			wantErr: true,
		},
		{
			name:    "confidence below zero",
			intent:  EditIntent{OperationType: "patch", Confidence: -0.1},
			wantErr: true,
		},
		{
			name:    "confidence above one",
			intent:  EditIntent{OperationType: "patch", Confidence: 1.1},
			wantErr: true,
		},
		{
			name:    "invalid operation type and out-of-range confidence",
			intent:  EditIntent{OperationType: "unknown", Confidence: 2.0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.intent.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EditIntent.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
