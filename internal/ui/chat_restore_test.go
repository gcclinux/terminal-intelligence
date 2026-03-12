package ui

import (
	"strings"
	"testing"

	"github.com/user/terminal-intelligence/internal/agentic"
	"github.com/user/terminal-intelligence/internal/types"
)

// TestRestoreAutonomousState verifies that loading a chat with an autonomous
// creation plan waiting for approval restores the AutonomousCreator state.
func TestRestoreAutonomousState(t *testing.T) {
	// Create a mock app with minimal setup
	app := &App{
		aiPane: &AIChatPane{
			messages: []types.ChatMessage{},
		},
		config: &types.AppConfig{
			WorkspaceDir: "/test/workspace",
			DefaultModel: "test-model",
		},
	}

	// Simulate a chat history with a /create command and plan
	// This matches the actual format saved by appendMessageToSessionLog
	app.aiPane.messages = []types.ChatMessage{
		{
			Role:    "user",
			Content: "/create A simple web server",
		},
		{
			Role: "assistant",
			Content: `ai-assist 10:21:43
Plan generated:

1. Project Name: simple-web-server
2. Architecture: Basic HTTP server using Go
3. Files:
   - main.go
   - README.md

Do you want to proceed? Type /proceed to continue or /cancel to abort.`,
		},
	}

	// Call restoreAutonomousState
	app.restoreAutonomousState()

	// Verify AutonomousCreator was created
	if app.autonomousCreator == nil {
		t.Fatal("Expected autonomousCreator to be restored, got nil")
	}

	// Verify state is StateWaitingApproval
	if app.autonomousCreator.State != agentic.StateWaitingApproval {
		t.Errorf("Expected state StateWaitingApproval, got %v", app.autonomousCreator.State)
	}

	// Verify description was extracted
	expectedDesc := "A simple web server"
	if app.autonomousCreator.Description != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, app.autonomousCreator.Description)
	}

	// Verify plan was extracted
	if !strings.Contains(app.autonomousCreator.Plan, "simple-web-server") {
		t.Error("Expected plan to contain project name")
	}

	// Verify project name was extracted
	if app.autonomousCreator.ProjectName == "" {
		t.Error("Expected project name to be extracted")
	}
}

// TestRestoreAutonomousState_NoProceededPrompt verifies that if the chat
// doesn't contain a proceed prompt, no state is restored.
func TestRestoreAutonomousState_NoProceededPrompt(t *testing.T) {
	app := &App{
		aiPane: &AIChatPane{
			messages: []types.ChatMessage{},
		},
		config: &types.AppConfig{
			WorkspaceDir: "/test/workspace",
		},
	}

	// Simulate a normal chat without proceed prompt
	app.aiPane.messages = []types.ChatMessage{
		{
			Role:    "user",
			Content: "Hello",
		},
		{
			Role:    "assistant",
			Content: "Hi there!",
		},
	}

	// Call restoreAutonomousState
	app.restoreAutonomousState()

	// Verify AutonomousCreator was NOT created
	if app.autonomousCreator != nil {
		t.Error("Expected autonomousCreator to be nil for normal chat")
	}
}

// TestExtractProjectNameFromPlan verifies project name extraction from plan text.
func TestExtractProjectNameFromPlan(t *testing.T) {
	tests := []struct {
		name     string
		plan     string
		expected string
	}{
		{
			name: "Simple project name",
			plan: `1. Project Name: my-web-app
2. Architecture: Simple`,
			expected: "my-web-app",
		},
		{
			name:     "Project name with backticks",
			plan:     "Project name: `simple-server`\nOther content",
			expected: "simple-server",
		},
		{
			name: "Project name with quotes",
			plan: `The project name: "test-app"
More details`,
			expected: "test-app",
		},
		{
			name:     "No project name",
			plan:     "Just some random text",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractProjectNameFromPlan(tt.plan)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
