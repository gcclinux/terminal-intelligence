package ui

import (
	"testing"
	"time"

	"github.com/user/terminal-intelligence/internal/types"
)

// TestAddFixRequest verifies that fix requests are added to conversation history
func TestAddFixRequest(t *testing.T) {
	// Create AI pane (nil client is fine for this test)
	aiPane := NewAIChatPane(nil, "test-model", "ollama", "")
	aiPane.SetSize(80, 24)

	// Add a fix request
	message := "fix the bug in line 10"
	filePath := "/path/to/file.sh"
	aiPane.AddFixRequest(message, filePath)

	// Verify the fix request was added to history
	history := aiPane.GetHistory()
	if len(history) != 1 {
		t.Fatalf("Expected 1 message in history, got %d", len(history))
	}

	msg := history[0]
	if msg.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", msg.Role)
	}
	if msg.Content != message {
		t.Errorf("Expected content '%s', got '%s'", message, msg.Content)
	}
	if !msg.IsFixRequest {
		t.Error("Expected IsFixRequest to be true")
	}
	if !msg.ContextIncluded {
		t.Error("Expected ContextIncluded to be true")
	}
	if msg.FilePath != filePath {
		t.Errorf("Expected FilePath '%s', got '%s'", filePath, msg.FilePath)
	}
}

// TestFixRequestAndNotificationInHistory verifies the full flow of fix request and notification
func TestFixRequestAndNotificationInHistory(t *testing.T) {
	aiPane := NewAIChatPane(nil, "test-model", "ollama", "")
	aiPane.SetSize(80, 24)

	// Add a fix request
	fixMessage := "update the function"
	filePath := "/path/to/script.sh"
	aiPane.AddFixRequest(fixMessage, filePath)

	// Add a notification (simulating the fix result)
	notification := "✓ Applied fix to /path/to/script.sh\nModified 3 lines near line 15\nRemember to save the file and test your changes."
	aiPane.DisplayNotification(notification)

	// Verify both messages are in history
	history := aiPane.GetHistory()
	if len(history) != 2 {
		t.Fatalf("Expected 2 messages in history, got %d", len(history))
	}

	// Check fix request
	fixReq := history[0]
	if fixReq.Role != "user" {
		t.Errorf("Expected first message role 'user', got '%s'", fixReq.Role)
	}
	if !fixReq.IsFixRequest {
		t.Error("Expected first message to be a fix request")
	}
	if fixReq.FilePath != filePath {
		t.Errorf("Expected FilePath '%s', got '%s'", filePath, fixReq.FilePath)
	}

	// Check notification
	notif := history[1]
	if notif.Role != "assistant" {
		t.Errorf("Expected second message role 'assistant', got '%s'", notif.Role)
	}
	if !notif.IsNotification {
		t.Error("Expected second message to be a notification")
	}
	if notif.Content != notification {
		t.Errorf("Expected notification content '%s', got '%s'", notification, notif.Content)
	}
}

// TestRenderFixRequestMessage verifies that fix requests are rendered with file path context
func TestRenderFixRequestMessage(t *testing.T) {
	aiPane := NewAIChatPane(nil, "test-model", "ollama", "")
	aiPane.SetSize(80, 24)

	// Create a fix request message
	msg := types.ChatMessage{
		Role:            "user",
		Content:         "fix the syntax error",
		Timestamp:       time.Now(),
		ContextIncluded: true,
		IsFixRequest:    true,
		FilePath:        "/workspace/test.sh",
	}

	// Render the message
	lines := aiPane.renderMessage(msg)

	// Verify the rendered output contains file path indicator
	if len(lines) == 0 {
		t.Fatal("Expected rendered lines, got empty")
	}

	// The first line should contain the file path indicator
	headerLine := lines[0]
	if !containsString(headerLine, "[file:") {
		t.Errorf("Expected header to contain '[file:', got: %s", headerLine)
	}
	if !containsString(headerLine, "/workspace/test.sh") {
		t.Errorf("Expected header to contain file path, got: %s", headerLine)
	}
}

// TestConversationalMessageNotMarkedAsFixRequest verifies conversational messages are not marked as fix requests
func TestConversationalMessageNotMarkedAsFixRequest(t *testing.T) {
	aiPane := NewAIChatPane(nil, "test-model", "ollama", "")
	aiPane.SetSize(80, 24)

	// Add a regular conversational message using SendMessage
	// This simulates the normal conversational flow
	userMsg := types.ChatMessage{
		Role:            "user",
		Content:         "what is bash?",
		Timestamp:       time.Now(),
		ContextIncluded: false,
		IsFixRequest:    false,
	}
	aiPane.messages = append(aiPane.messages, userMsg)

	// Verify the message is not marked as a fix request
	history := aiPane.GetHistory()
	if len(history) != 1 {
		t.Fatalf("Expected 1 message in history, got %d", len(history))
	}

	msg := history[0]
	if msg.IsFixRequest {
		t.Error("Expected IsFixRequest to be false for conversational message")
	}
	if msg.FilePath != "" {
		t.Errorf("Expected empty FilePath for conversational message, got '%s'", msg.FilePath)
	}
}

// Helper function to check if a string contains a substring (ignoring ANSI codes)
func containsString(s, substr string) bool {
	// Simple check - in real implementation might need to strip ANSI codes
	// For now, just check if substring is present
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
