package docgen

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// MockChatPane is a mock implementation of ChatPaneInterface for testing
type MockChatPane struct {
	notifications []string
}

func (m *MockChatPane) DisplayNotification(notification string) {
	m.notifications = append(m.notifications, notification)
}

func (m *MockChatPane) GetNotifications() []string {
	return m.notifications
}

func (m *MockChatPane) Clear() {
	m.notifications = nil
}

// Feature: project-documentation-generation, Property 21: Progress Feedback
// **Validates: Requirements 8.2**
// For any generation in progress, progress messages emitted for major stages
func TestProperty21_ProgressFeedback(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mockPane := &MockChatPane{}
		feedback := NewFeedbackManager(mockPane)

		// Generate random stages
		numStages := rapid.IntRange(1, 5).Draw(rt, "numStages")
		stages := make([]string, numStages)
		for i := 0; i < numStages; i++ {
			stages[i] = rapid.StringMatching(`[a-zA-Z ]+`).Draw(rt, "stage")
		}

		// Emit progress for each stage
		for _, stage := range stages {
			details := rapid.String().Draw(rt, "details")
			feedback.NotifyProgress(stage, details)
		}

		// Verify progress messages were emitted
		notifications := mockPane.GetNotifications()
		if len(notifications) != numStages {
			rt.Fatalf("Expected %d progress notifications, got %d", numStages, len(notifications))
		}

		// Verify each notification contains the stage name
		for i, stage := range stages {
			if !strings.Contains(notifications[i], stage) {
				rt.Fatalf("Progress notification %d does not contain stage %q: %q", i, stage, notifications[i])
			}
		}
	})
}

// Feature: project-documentation-generation, Property 22: Completion Feedback
// **Validates: Requirements 8.3, 8.4**
// For any successful generation, message contains file path and summary
func TestProperty22_CompletionFeedback(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mockPane := &MockChatPane{}
		feedback := NewFeedbackManager(mockPane)

		// Generate random write results
		numResults := rapid.IntRange(1, 5).Draw(rt, "numResults")
		results := make([]*WriteResult, numResults)
		for i := 0; i < numResults; i++ {
			filename := rapid.StringMatching(`[A-Z_]+\.md`).Draw(rt, "filename")
			path := rapid.StringMatching(`/[a-z/]+/`+filename).Draw(rt, "path")
			results[i] = &WriteResult{
				Filename: filename,
				Path:     path,
				Existed:  rapid.Bool().Draw(rt, "existed"),
				Written:  true,
			}
		}

		// Notify completion
		feedback.NotifyComplete(results, nil)

		// Verify completion message was sent
		notifications := mockPane.GetNotifications()
		if len(notifications) != 1 {
			rt.Fatalf("Expected 1 completion notification, got %d", len(notifications))
		}

		notification := notifications[0]

		// Verify notification contains "complete"
		if !strings.Contains(strings.ToLower(notification), "complete") {
			rt.Fatalf("Completion notification does not contain 'complete': %q", notification)
		}

		// Verify notification contains all file paths
		for _, result := range results {
			if !strings.Contains(notification, result.Path) {
				rt.Fatalf("Completion notification does not contain path %q: %q", result.Path, notification)
			}
		}
	})
}

// Feature: project-documentation-generation, Property 23: Error Feedback
// **Validates: Requirements 8.5**
// For any failed generation, error message describes failure
func TestProperty23_ErrorFeedback(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mockPane := &MockChatPane{}
		feedback := NewFeedbackManager(mockPane)

		// Generate random error message
		errorMsg := rapid.StringMatching(`[a-zA-Z0-9 ]+`).Draw(rt, "errorMsg")
		err := &mockError{msg: errorMsg}

		// Notify error
		feedback.NotifyError(err)

		// Verify error message was sent
		notifications := mockPane.GetNotifications()
		if len(notifications) != 1 {
			rt.Fatalf("Expected 1 error notification, got %d", len(notifications))
		}

		notification := notifications[0]

		// Verify notification contains "failed" or "error"
		lowerNotif := strings.ToLower(notification)
		if !strings.Contains(lowerNotif, "failed") && !strings.Contains(lowerNotif, "error") {
			rt.Fatalf("Error notification does not contain 'failed' or 'error': %q", notification)
		}

		// Verify notification contains the error message
		if !strings.Contains(notification, errorMsg) {
			rt.Fatalf("Error notification does not contain error message %q: %q", errorMsg, notification)
		}
	})
}

// Feature: project-documentation-generation, Property 27: Multi-File Completion Report
// **Validates: Requirements 9.4**
// For any multi-type generation, all created files listed
func TestProperty27_MultiFileCompletionReport(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mockPane := &MockChatPane{}
		feedback := NewFeedbackManager(mockPane)

		// Generate multiple write results (at least 2)
		numResults := rapid.IntRange(2, 5).Draw(rt, "numResults")
		results := make([]*WriteResult, numResults)
		filenames := make([]string, numResults)

		for i := 0; i < numResults; i++ {
			filename := rapid.StringMatching(`[A-Z_]+\.md`).Draw(rt, "filename")
			filenames[i] = filename
			results[i] = &WriteResult{
				Filename: filename,
				Path:     "/workspace/" + filename,
				Existed:  false,
				Written:  true,
			}
		}

		// Notify completion
		feedback.NotifyComplete(results, nil)

		// Verify completion message was sent
		notifications := mockPane.GetNotifications()
		if len(notifications) != 1 {
			rt.Fatalf("Expected 1 completion notification, got %d", len(notifications))
		}

		notification := notifications[0]

		// Verify all filenames are listed in the notification
		for _, filename := range filenames {
			if !strings.Contains(notification, filename) {
				rt.Fatalf("Completion notification does not list file %q: %q", filename, notification)
			}
		}
	})
}

// Feature: project-documentation-generation, Property 30: Scope Reporting
// **Validates: Requirements 10.4**
// For any generation with scope filters, scope indicated in summary
func TestProperty30_ScopeReporting(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		mockPane := &MockChatPane{}
		feedback := NewFeedbackManager(mockPane)

		// Generate random scope filters (at least 1)
		numFilters := rapid.IntRange(1, 3).Draw(rt, "numFilters")
		scopeFilters := make([]string, numFilters)
		for i := 0; i < numFilters; i++ {
			scopeFilters[i] = rapid.StringMatching(`[a-z/]+`).Draw(rt, "filter")
		}

		// Create a write result
		result := &WriteResult{
			Filename: "DOCUMENTATION.md",
			Path:     "/workspace/DOCUMENTATION.md",
			Existed:  false,
			Written:  true,
		}

		// Notify completion with scope filters
		feedback.NotifyComplete([]*WriteResult{result}, scopeFilters)

		// Verify completion message was sent
		notifications := mockPane.GetNotifications()
		if len(notifications) != 1 {
			rt.Fatalf("Expected 1 completion notification, got %d", len(notifications))
		}

		notification := notifications[0]

		// Verify notification contains "scope" or "Scope"
		if !strings.Contains(notification, "scope") && !strings.Contains(notification, "Scope") {
			rt.Fatalf("Completion notification does not mention scope: %q", notification)
		}

		// Verify notification contains at least one scope filter
		foundFilter := false
		for _, filter := range scopeFilters {
			if strings.Contains(notification, filter) {
				foundFilter = true
				break
			}
		}

		if !foundFilter {
			rt.Fatalf("Completion notification does not contain any scope filter: %q", notification)
		}
	})
}

// mockError is a simple error implementation for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
