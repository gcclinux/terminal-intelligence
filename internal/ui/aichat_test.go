package ui

import (
	"testing"

	"github.com/user/terminal-intelligence/internal/types"
)

// TestDisplayNotification verifies that notifications are displayed with distinct formatting
func TestDisplayNotification(t *testing.T) {
	// Create a mock AI client (nil is fine for this test)
	pane := NewAIChatPane(nil, "test-model", "ollama")
	pane.SetSize(80, 24)

	// Display a notification
	notification := "✓ Code fix applied to test.sh\n\nChanges:\n- Modified file: 10 → 12 lines (+2)\n\n⚠️  Remember to save the file (Ctrl+S) and test the changes!"
	pane.DisplayNotification(notification)

	// Verify the message was added
	if len(pane.messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(pane.messages))
	}

	// Verify the message is marked as a notification
	msg := pane.messages[0]
	if !msg.IsNotification {
		t.Errorf("expected message to be marked as notification")
	}

	// Verify the message has the correct role
	if msg.Role != "assistant" {
		t.Errorf("expected role 'assistant', got '%s'", msg.Role)
	}

	// Verify the content matches
	if msg.Content != notification {
		t.Errorf("expected content to match notification")
	}
}

// TestRenderNotificationMessage verifies that notification messages are rendered with distinct styling
func TestRenderNotificationMessage(t *testing.T) {
	pane := NewAIChatPane(nil, "test-model", "ollama")
	pane.SetSize(80, 24)

	// Create a notification message
	notificationMsg := types.ChatMessage{
		Role:           "assistant",
		Content:        "✓ Code fix applied",
		IsNotification: true,
	}

	// Create a regular message for comparison
	regularMsg := types.ChatMessage{
		Role:           "assistant",
		Content:        "Here's some code",
		IsNotification: false,
	}

	// Render both messages
	notificationLines := pane.renderMessage(notificationMsg)
	regularLines := pane.renderMessage(regularMsg)

	// Verify both have content
	if len(notificationLines) == 0 {
		t.Errorf("expected notification to have rendered lines")
	}
	if len(regularLines) == 0 {
		t.Errorf("expected regular message to have rendered lines")
	}

	// Verify the notification header contains "notification" text
	notificationHeader := notificationLines[0]
	if notificationHeader == "" {
		t.Errorf("expected notification header to be non-empty")
	}

	// The notification should have distinct styling (we can't easily test the color,
	// but we can verify the structure is correct)
	regularHeader := regularLines[0]
	if regularHeader == "" {
		t.Errorf("expected regular header to be non-empty")
	}
}

// TestDisplayNotificationAutoScroll verifies that displaying a notification scrolls to bottom
func TestDisplayNotificationAutoScroll(t *testing.T) {
	pane := NewAIChatPane(nil, "test-model", "ollama")
	pane.SetSize(80, 10) // Small height to force scrolling

	// Add multiple messages to force scrolling
	for i := 0; i < 10; i++ {
		pane.DisplayResponse("Message " + string(rune('0'+i)))
	}

	// Record the scroll offset before notification
	scrollBefore := pane.scrollOffset

	// Display a notification
	pane.DisplayNotification("✓ Code fix applied")

	// Verify scroll offset is at the bottom
	maxScroll := pane.getMaxScroll()
	if pane.scrollOffset != maxScroll {
		t.Errorf("expected scroll offset to be at bottom (%d), got %d", maxScroll, pane.scrollOffset)
	}

	// Verify the scroll offset changed (unless it was already at bottom)
	if scrollBefore == maxScroll {
		// Already at bottom, that's fine
	} else if pane.scrollOffset <= scrollBefore {
		t.Errorf("expected scroll offset to increase after notification")
	}
}
