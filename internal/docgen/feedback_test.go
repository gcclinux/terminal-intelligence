package docgen

import (
	"errors"
	"strings"
	"testing"
)

// Test NotifyStart with single documentation type
func TestFeedbackManager_NotifyStart_SingleType(t *testing.T) {
	mockPane := &MockChatPane{}
	feedback := NewFeedbackManager(mockPane)

	feedback.NotifyStart([]DocumentationType{DocTypeUserManual})

	notifications := mockPane.GetNotifications()
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	notification := notifications[0]
	if !strings.Contains(notification, "User Manual") {
		t.Errorf("Notification should contain 'User Manual': %q", notification)
	}

	if !strings.Contains(strings.ToLower(notification), "starting") {
		t.Errorf("Notification should indicate starting: %q", notification)
	}
}

// Test NotifyStart with multiple documentation types
func TestFeedbackManager_NotifyStart_MultipleTypes(t *testing.T) {
	mockPane := &MockChatPane{}
	feedback := NewFeedbackManager(mockPane)

	docTypes := []DocumentationType{
		DocTypeUserManual,
		DocTypeInstallation,
		DocTypeAPI,
	}

	feedback.NotifyStart(docTypes)

	notifications := mockPane.GetNotifications()
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	notification := notifications[0]
	expectedTypes := []string{"User Manual", "Installation Guide", "API Reference"}

	for _, typeName := range expectedTypes {
		if !strings.Contains(notification, typeName) {
			t.Errorf("Notification should contain %q: %q", typeName, notification)
		}
	}
}

// Test NotifyProgress with stage and details
func TestFeedbackManager_NotifyProgress(t *testing.T) {
	mockPane := &MockChatPane{}
	feedback := NewFeedbackManager(mockPane)

	feedback.NotifyProgress("Analyzing project", "Found 42 files")

	notifications := mockPane.GetNotifications()
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	notification := notifications[0]
	if !strings.Contains(notification, "Analyzing project") {
		t.Errorf("Notification should contain stage: %q", notification)
	}

	if !strings.Contains(notification, "Found 42 files") {
		t.Errorf("Notification should contain details: %q", notification)
	}
}

// Test NotifyProgress with stage only (no details)
func TestFeedbackManager_NotifyProgress_NoDetails(t *testing.T) {
	mockPane := &MockChatPane{}
	feedback := NewFeedbackManager(mockPane)

	feedback.NotifyProgress("Generating documentation", "")

	notifications := mockPane.GetNotifications()
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	notification := notifications[0]
	if !strings.Contains(notification, "Generating documentation") {
		t.Errorf("Notification should contain stage: %q", notification)
	}
}

// Test NotifyComplete with single file
func TestFeedbackManager_NotifyComplete_SingleFile(t *testing.T) {
	mockPane := &MockChatPane{}
	feedback := NewFeedbackManager(mockPane)

	results := []*WriteResult{
		{
			Filename: "USER_MANUAL.md",
			Path:     "/workspace/USER_MANUAL.md",
			Existed:  false,
			Written:  true,
		},
	}

	feedback.NotifyComplete(results, nil)

	notifications := mockPane.GetNotifications()
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	notification := notifications[0]
	if !strings.Contains(strings.ToLower(notification), "complete") {
		t.Errorf("Notification should indicate completion: %q", notification)
	}

	if !strings.Contains(notification, "/workspace/USER_MANUAL.md") {
		t.Errorf("Notification should contain file path: %q", notification)
	}

	if !strings.Contains(notification, "created") {
		t.Errorf("Notification should indicate file was created: %q", notification)
	}
}

// Test NotifyComplete with updated file
func TestFeedbackManager_NotifyComplete_UpdatedFile(t *testing.T) {
	mockPane := &MockChatPane{}
	feedback := NewFeedbackManager(mockPane)

	results := []*WriteResult{
		{
			Filename: "API_REFERENCE.md",
			Path:     "/workspace/API_REFERENCE.md",
			Existed:  true,
			Written:  true,
		},
	}

	feedback.NotifyComplete(results, nil)

	notifications := mockPane.GetNotifications()
	notification := notifications[0]

	if !strings.Contains(notification, "updated") {
		t.Errorf("Notification should indicate file was updated: %q", notification)
	}
}

// Test NotifyComplete with scope filters
func TestFeedbackManager_NotifyComplete_WithScope(t *testing.T) {
	mockPane := &MockChatPane{}
	feedback := NewFeedbackManager(mockPane)

	results := []*WriteResult{
		{
			Filename: "DOCUMENTATION.md",
			Path:     "/workspace/DOCUMENTATION.md",
			Existed:  false,
			Written:  true,
		},
	}

	scopeFilters := []string{"internal/docgen", "*.go"}

	feedback.NotifyComplete(results, scopeFilters)

	notifications := mockPane.GetNotifications()
	notification := notifications[0]

	if !strings.Contains(notification, "Scope") {
		t.Errorf("Notification should mention scope: %q", notification)
	}

	if !strings.Contains(notification, "internal/docgen") {
		t.Errorf("Notification should contain scope filter: %q", notification)
	}
}

// Test NotifyError
func TestFeedbackManager_NotifyError(t *testing.T) {
	mockPane := &MockChatPane{}
	feedback := NewFeedbackManager(mockPane)

	err := errors.New("failed to parse code structure")
	feedback.NotifyError(err)

	notifications := mockPane.GetNotifications()
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	notification := notifications[0]
	lowerNotif := strings.ToLower(notification)

	if !strings.Contains(lowerNotif, "failed") && !strings.Contains(lowerNotif, "error") {
		t.Errorf("Notification should indicate error: %q", notification)
	}

	if !strings.Contains(notification, "failed to parse code structure") {
		t.Errorf("Notification should contain error message: %q", notification)
	}
}

// Test NotifyFileConflict
func TestFeedbackManager_NotifyFileConflict(t *testing.T) {
	mockPane := &MockChatPane{}
	feedback := NewFeedbackManager(mockPane)

	conflicts := []*WriteResult{
		{
			Filename: "USER_MANUAL.md",
			Path:     "/workspace/USER_MANUAL.md",
			Existed:  true,
			Written:  false,
		},
		{
			Filename: "API_REFERENCE.md",
			Path:     "/workspace/API_REFERENCE.md",
			Existed:  true,
			Written:  false,
		},
	}

	feedback.NotifyFileConflict(conflicts)

	notifications := mockPane.GetNotifications()
	if len(notifications) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifications))
	}

	notification := notifications[0]

	if !strings.Contains(strings.ToLower(notification), "conflict") {
		t.Errorf("Notification should mention conflict: %q", notification)
	}

	if !strings.Contains(notification, "USER_MANUAL.md") {
		t.Errorf("Notification should list first conflicting file: %q", notification)
	}

	if !strings.Contains(notification, "API_REFERENCE.md") {
		t.Errorf("Notification should list second conflicting file: %q", notification)
	}

	if !strings.Contains(notification, "overwrite") {
		t.Errorf("Notification should mention overwrite option: %q", notification)
	}
}

// Test with nil chat pane (should not panic)
func TestFeedbackManager_NilChatPane(t *testing.T) {
	feedback := NewFeedbackManager(nil)

	// These should not panic
	feedback.NotifyStart([]DocumentationType{DocTypeUserManual})
	feedback.NotifyProgress("test", "details")
	feedback.NotifyComplete([]*WriteResult{}, nil)
	feedback.NotifyError(errors.New("test error"))
	feedback.NotifyFileConflict([]*WriteResult{})
}

// Test documentationTypeName helper
func TestDocumentationTypeName(t *testing.T) {
	testCases := []struct {
		docType  DocumentationType
		expected string
	}{
		{DocTypeUserManual, "User Manual"},
		{DocTypeInstallation, "Installation Guide"},
		{DocTypeAPI, "API Reference"},
		{DocTypeTutorial, "Tutorial"},
		{DocTypeGeneral, "General Documentation"},
	}

	for _, tc := range testCases {
		result := documentationTypeName(tc.docType)
		if result != tc.expected {
			t.Errorf("documentationTypeName(%v) = %q, expected %q", tc.docType, result, tc.expected)
		}
	}
}
