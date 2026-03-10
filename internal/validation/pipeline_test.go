package validation

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPipeline_Creation(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	if pipeline == nil {
		t.Fatal("Expected pipeline to be created")
	}

	if pipeline.validationEngine == nil {
		t.Error("Expected validation engine to be initialized")
	}

	if pipeline.fileChangeDetector == nil {
		t.Error("Expected file change detector to be initialized")
	}

	if pipeline.chatPanelIntegration == nil {
		t.Error("Expected chat panel integration to be initialized")
	}
}

func TestPipeline_TriggerValidation_UnsupportedFiles(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	files := []string{"README.md", "config.yaml"}
	session, err := pipeline.TriggerValidation(files)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if session == nil {
		t.Fatal("Expected session to be created")
	}

	if session.Status != StatusCompleted {
		t.Errorf("Expected status %s, got %s", StatusCompleted, session.Status)
	}

	// Verify unsupported file notification was sent
	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected unsupported file notification")
	}

	lastMessage := messages[len(messages)-1]
	if !strings.HasPrefix(lastMessage, "ℹ️") {
		t.Errorf("Expected unsupported file notification, got: %s", lastMessage)
	}
}

func TestPipeline_TriggerValidation_EmptyFiles(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	files := []string{}
	session, err := pipeline.TriggerValidation(files)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if session != nil {
		t.Errorf("Expected nil session for empty files, got %v", session)
	}

	// Verify no messages were sent
	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected no messages for empty files, got %d", len(messages))
	}
}

func TestPipeline_QueueValidation(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	files := []string{"README.md"}
	pipeline.QueueValidation(files)

	// Wait a bit for queue processing
	time.Sleep(100 * time.Millisecond)

	// Verify validation was processed
	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) == 0 {
		t.Error("Expected validation to be processed from queue")
	}
}

func TestPipeline_QueueValidation_EmptyFiles(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	files := []string{}
	pipeline.QueueValidation(files)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Verify no messages (empty files should be skipped)
	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected no messages for empty files, got %d", len(messages))
	}
}

func TestPipeline_FileChangeDetector_Integration(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// Simulate file change event with a code file (not filtered)
	detector := pipeline.GetFileChangeDetector()
	detector.NotifyFileChange(FileChangeEvent{
		FilePath:  "test.go",
		Operation: OperationModify,
		Timestamp: time.Now(),
	})

	// Wait for queue processing
	time.Sleep(100 * time.Millisecond)

	// Verify validation was triggered
	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) == 0 {
		t.Error("Expected validation to be triggered by file change event")
	}
}

func TestPipeline_FileChangeDetector_DeleteOperation(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// Simulate delete operation
	detector := pipeline.GetFileChangeDetector()
	detector.NotifyFileChange(FileChangeEvent{
		FilePath:  "test.go",
		Operation: OperationDelete,
		Timestamp: time.Now(),
	})

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Verify no validation was triggered (delete operations are skipped)
	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected no validation for delete operation, got %d messages", len(messages))
	}
}

func TestPipeline_IsProcessing(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// Initially not processing
	if pipeline.IsProcessing() {
		t.Error("Expected pipeline to not be processing initially")
	}

	// Queue a validation
	files := []string{"README.md"}
	pipeline.QueueValidation(files)

	// Wait a tiny bit for processing to start
	time.Sleep(10 * time.Millisecond)

	// May or may not be processing depending on timing
	// Just verify the method doesn't panic
	_ = pipeline.IsProcessing()
}

func TestPipeline_Shutdown(t *testing.T) {
	pipeline := NewPipeline()

	// Shutdown should not panic
	pipeline.Shutdown()

	// Queue validation after shutdown should not panic
	files := []string{"test.go"}
	pipeline.QueueValidation(files)
}

func TestPipeline_ErrorHandling_Timeout(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// Simulate timeout error
	err := context.DeadlineExceeded
	pipeline.handleValidationError(err)

	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected error message")
	}

	lastMessage := messages[len(messages)-1]
	if !strings.Contains(lastMessage, "timed out") {
		t.Errorf("Expected timeout message, got: %s", lastMessage)
	}
}

func TestPipeline_ErrorHandling_Cancelled(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// Simulate cancelled error
	err := context.Canceled
	pipeline.handleValidationError(err)

	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected error message")
	}

	lastMessage := messages[len(messages)-1]
	if !strings.Contains(lastMessage, "cancelled") {
		t.Errorf("Expected cancelled message, got: %s", lastMessage)
	}
}

func TestPipeline_ErrorHandling_CommandNotFound(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// Simulate command not found error
	err := fmt.Errorf("command not found: go")
	pipeline.handleValidationError(err)

	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected error message")
	}

	lastMessage := messages[len(messages)-1]
	if !strings.Contains(lastMessage, "Validator not found") {
		t.Errorf("Expected validator not found message, got: %s", lastMessage)
	}
}

func TestPipeline_ErrorHandling_PermissionDenied(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// Simulate permission denied error
	err := fmt.Errorf("permission denied: /path/to/file")
	pipeline.handleValidationError(err)

	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected error message")
	}

	lastMessage := messages[len(messages)-1]
	if !strings.Contains(lastMessage, "Permission denied") {
		t.Errorf("Expected permission denied message, got: %s", lastMessage)
	}
}

func TestPipeline_ErrorHandling_FileNotFound(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// Simulate file not found error
	err := fmt.Errorf("no such file or directory: test.go")
	pipeline.handleValidationError(err)

	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected error message")
	}

	lastMessage := messages[len(messages)-1]
	if !strings.Contains(lastMessage, "File not found") {
		t.Errorf("Expected file not found message, got: %s", lastMessage)
	}
}

func TestPipeline_ErrorHandling_GenericError(t *testing.T) {
	pipeline := NewPipeline()
	defer pipeline.Shutdown()

	// Simulate generic error
	err := fmt.Errorf("something went wrong")
	pipeline.handleValidationError(err)

	messages := pipeline.GetChatPanelIntegration().GetMessages()
	if len(messages) == 0 {
		t.Fatal("Expected error message")
	}

	lastMessage := messages[len(messages)-1]
	if !strings.Contains(lastMessage, "Validation error") {
		t.Errorf("Expected validation error message, got: %s", lastMessage)
	}
}
