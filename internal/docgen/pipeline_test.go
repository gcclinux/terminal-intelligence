package docgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// MockAIClient is a mock implementation of ai.AIClient for testing
type MockAIClient struct{}

func (m *MockAIClient) Generate(prompt string, model string, context []int) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- "Mock AI response"
	close(ch)
	return ch, nil
}

func (m *MockAIClient) IsAvailable() (bool, error) {
	return true, nil
}

func (m *MockAIClient) ListModels() ([]string, error) {
	return []string{"test-model"}, nil
}

// Feature: project-documentation-generation, Property 26: Multi-File Output
// **Validates: Requirements 9.3**
// For any multi-type generation, separate file for each type with appropriate filename
func TestProperty26_MultiFileOutput(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tmpDir := t.TempDir()

		// Create a simple test project
		createTestProject(t, tmpDir)

		mockPane := &MockChatPane{}
		mockAI := &MockAIClient{}
		pipeline := NewPipeline(tmpDir, mockAI, "test-model", mockPane)

		// Generate command with multiple doc types
		command := "/project /doc create user manual and API reference"

		isDocCommand, err := pipeline.ProcessCommand(command)

		if !isDocCommand {
			rt.Fatalf("Command should be recognized as doc command")
		}

		if err != nil {
			rt.Fatalf("Pipeline failed: %v", err)
		}

		// Verify multiple files were created
		expectedFiles := []string{"USER_MANUAL.md", "API_REFERENCE.md"}
		for _, filename := range expectedFiles {
			fullPath := filepath.Join(tmpDir, filename)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				rt.Fatalf("Expected file %s was not created", filename)
			}
		}

		// Verify each file has content
		for _, filename := range expectedFiles {
			fullPath := filepath.Join(tmpDir, filename)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				rt.Fatalf("Failed to read %s: %v", filename, err)
			}

			if len(content) == 0 {
				rt.Fatalf("File %s is empty", filename)
			}
		}
	})
}

// Feature: project-documentation-generation, Property 29: Scope-Limited Documentation
// **Validates: Requirements 10.3**
// For any generation with scope filters, content only from filtered scope
func TestProperty29_ScopeLimitedDocumentation(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		tmpDir := t.TempDir()

		// Create test project with multiple directories
		createTestProjectWithMultipleDirs(t, tmpDir)

		mockPane := &MockChatPane{}
		mockAI := &MockAIClient{}
		pipeline := NewPipeline(tmpDir, mockAI, "test-model", mockPane)

		// Generate command with scope filter
		command := "/project /doc create API reference for module internal"

		isDocCommand, err := pipeline.ProcessCommand(command)

		if !isDocCommand {
			rt.Fatalf("Command should be recognized as doc command")
		}

		if err != nil {
			rt.Fatalf("Pipeline failed: %v", err)
		}

		// Verify file was created
		fullPath := filepath.Join(tmpDir, "API_REFERENCE.md")
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			rt.Fatalf("Expected file API_REFERENCE.md was not created")
		}

		// Verify completion message mentions scope
		notifications := mockPane.GetNotifications()
		foundScopeNotification := false
		for _, notif := range notifications {
			if strings.Contains(notif, "Scope") || strings.Contains(notif, "scope") {
				foundScopeNotification = true
				break
			}
		}

		if !foundScopeNotification {
			rt.Fatalf("Completion notification should mention scope")
		}
	})
}

// Test full command → Documentation files created
func TestPipeline_EndToEnd_FullCommand(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir)

	mockPane := &MockChatPane{}
	mockAI := &MockAIClient{}
	pipeline := NewPipeline(tmpDir, mockAI, "test-model", mockPane)

	command := "/project /doc create user manual"

	isDocCommand, err := pipeline.ProcessCommand(command)

	if !isDocCommand {
		t.Fatalf("Command should be recognized as doc command")
	}

	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}

	// Verify file was created
	fullPath := filepath.Join(tmpDir, "USER_MANUAL.md")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Fatalf("Expected file USER_MANUAL.md was not created")
	}

	// Verify notifications were sent
	notifications := mockPane.GetNotifications()
	if len(notifications) == 0 {
		t.Fatalf("Expected notifications to be sent")
	}

	// Verify start notification
	foundStart := false
	for _, notif := range notifications {
		if strings.Contains(strings.ToLower(notif), "starting") {
			foundStart = true
			break
		}
	}

	if !foundStart {
		t.Errorf("Expected start notification")
	}

	// Verify completion notification
	foundComplete := false
	for _, notif := range notifications {
		if strings.Contains(strings.ToLower(notif), "complete") {
			foundComplete = true
			break
		}
	}

	if !foundComplete {
		t.Errorf("Expected completion notification")
	}
}

// Test multi-type request → Multiple files created
func TestPipeline_EndToEnd_MultiType(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir)

	mockPane := &MockChatPane{}
	mockAI := &MockAIClient{}
	pipeline := NewPipeline(tmpDir, mockAI, "test-model", mockPane)

	command := "/project /doc create installation guide and API reference"

	isDocCommand, err := pipeline.ProcessCommand(command)

	if !isDocCommand {
		t.Fatalf("Command should be recognized as doc command")
	}

	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}

	// Verify both files were created
	expectedFiles := []string{"INSTALLATION.md", "API_REFERENCE.md"}
	for _, filename := range expectedFiles {
		fullPath := filepath.Join(tmpDir, filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", filename)
		}
	}
}

// Test scoped request → Only scoped content included
func TestPipeline_EndToEnd_ScopedRequest(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProjectWithMultipleDirs(t, tmpDir)

	mockPane := &MockChatPane{}
	mockAI := &MockAIClient{}
	pipeline := NewPipeline(tmpDir, mockAI, "test-model", mockPane)

	command := "/project /doc create documentation for module internal"

	isDocCommand, err := pipeline.ProcessCommand(command)

	if !isDocCommand {
		t.Fatalf("Command should be recognized as doc command")
	}

	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}

	// Verify file was created
	fullPath := filepath.Join(tmpDir, "DOCUMENTATION.md")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Fatalf("Expected file DOCUMENTATION.md was not created")
	}

	// Verify scope was reported in notifications
	notifications := mockPane.GetNotifications()
	foundScope := false
	for _, notif := range notifications {
		if strings.Contains(notif, "internal") {
			foundScope = true
			break
		}
	}

	if !foundScope {
		t.Errorf("Expected scope to be mentioned in notifications")
	}
}

// Test invalid command → Error message displayed
func TestPipeline_EndToEnd_InvalidCommand(t *testing.T) {
	tmpDir := t.TempDir()

	mockPane := &MockChatPane{}
	mockAI := &MockAIClient{}
	pipeline := NewPipeline(tmpDir, mockAI, "test-model", mockPane)

	command := "just a regular message without flags"

	isDocCommand, err := pipeline.ProcessCommand(command)

	if isDocCommand {
		t.Errorf("Command should not be recognized as doc command")
	}

	if err != nil {
		t.Errorf("Should not return error for non-doc command: %v", err)
	}
}

// Test empty project → Minimal documentation created
func TestPipeline_EndToEnd_EmptyProject(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create any files - empty project

	mockPane := &MockChatPane{}
	mockAI := &MockAIClient{}
	pipeline := NewPipeline(tmpDir, mockAI, "test-model", mockPane)

	command := "/project /doc create user manual"

	isDocCommand, err := pipeline.ProcessCommand(command)

	if !isDocCommand {
		t.Fatalf("Command should be recognized as doc command")
	}

	if err != nil {
		t.Fatalf("Pipeline should handle empty project gracefully: %v", err)
	}

	// Verify file was still created (even if minimal)
	fullPath := filepath.Join(tmpDir, "USER_MANUAL.md")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Fatalf("Expected file USER_MANUAL.md was not created")
	}
}

// Test file conflict handling
func TestPipeline_FileConflict(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir)

	// Create existing file
	existingFile := filepath.Join(tmpDir, "USER_MANUAL.md")
	err := os.WriteFile(existingFile, []byte("Existing content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	mockPane := &MockChatPane{}
	mockAI := &MockAIClient{}
	pipeline := NewPipeline(tmpDir, mockAI, "test-model", mockPane)

	command := "/project /doc create user manual"

	isDocCommand, err := pipeline.ProcessCommand(command)

	if !isDocCommand {
		t.Fatalf("Command should be recognized as doc command")
	}

	// Should not return error, but should report conflict
	if err != nil {
		t.Fatalf("Pipeline should handle conflict gracefully: %v", err)
	}

	// Verify conflict notification was sent
	notifications := mockPane.GetNotifications()
	foundConflict := false
	for _, notif := range notifications {
		if strings.Contains(strings.ToLower(notif), "conflict") {
			foundConflict = true
			break
		}
	}

	if !foundConflict {
		t.Errorf("Expected conflict notification")
	}

	// Verify original file was not modified
	content, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != "Existing content" {
		t.Errorf("Original file was modified")
	}
}

// MockUnavailableAIClient is a mock AIClient where IsAvailable returns false
type MockUnavailableAIClient struct{}

func (m *MockUnavailableAIClient) Generate(prompt string, model string, context []int) (<-chan string, error) {
	ch := make(chan string)
	close(ch)
	return ch, nil
}

func (m *MockUnavailableAIClient) IsAvailable() (bool, error) {
	return false, nil
}

func (m *MockUnavailableAIClient) ListModels() ([]string, error) {
	return nil, nil
}

// TestNewPipeline_ConstructsNonNilAIGenerator verifies that NewPipeline returns a non-nil
// pipeline (and by extension a non-nil aiGenerator, since NewPipeline always constructs one).
// Requirements: 6.2
func TestNewPipeline_ConstructsNonNilAIGenerator(t *testing.T) {
	tmpDir := t.TempDir()
	mockPane := &MockChatPane{}
	mockAI := &MockAIClient{}

	p := NewPipeline(tmpDir, mockAI, "test-model", mockPane)

	if p == nil {
		t.Fatal("NewPipeline returned nil pipeline")
	}
	if p.aiGenerator == nil {
		t.Fatal("NewPipeline did not initialise aiGenerator (got nil)")
	}
}

// TestPipeline_AIUnavailable_NoFilesWritten verifies that when IsAvailable returns false,
// NotifyError is called and no files are written to disk.
// Requirements: 4.1, 6.2
func TestPipeline_AIUnavailable_NoFilesWritten(t *testing.T) {
	tmpDir := t.TempDir()
	mockPane := &MockChatPane{}
	mockAI := &MockUnavailableAIClient{}

	pipeline := NewPipeline(tmpDir, mockAI, "test-model", mockPane)

	isDocCommand, err := pipeline.ProcessCommand("/project /doc create user manual")

	if !isDocCommand {
		t.Fatal("Command should be recognised as a doc command")
	}
	if err != nil {
		t.Fatalf("Pipeline returned unexpected error: %v", err)
	}

	// No .md files should have been written
	entries, readErr := os.ReadDir(tmpDir)
	if readErr != nil {
		t.Fatalf("Failed to read temp dir: %v", readErr)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			t.Errorf("Unexpected file written when AI unavailable: %s", e.Name())
		}
	}

	// NotifyError must have been called with an "unavailable" message
	notifications := mockPane.GetNotifications()
	foundError := false
	for _, n := range notifications {
		if strings.Contains(strings.ToLower(n), "unavailable") || strings.Contains(strings.ToLower(n), "error") {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Errorf("Expected an error notification about AI unavailability, got: %v", notifications)
	}
}

// Helper function to create a simple test project
func createTestProject(t *testing.T, dir string) {
	// Create a simple Go file
	mainGo := `package main

import "fmt"

// HelloWorld prints a greeting
func HelloWorld() {
	fmt.Println("Hello, World!")
}

func main() {
	HelloWorld()
}
`
	err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a README
	readme := `# Test Project

This is a test project for documentation generation.
`
	err = os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0644)
	if err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
}

// Helper function to create a test project with multiple directories
func createTestProjectWithMultipleDirs(t *testing.T, dir string) {
	// Create internal directory
	internalDir := filepath.Join(dir, "internal")
	err := os.MkdirAll(internalDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create internal dir: %v", err)
	}

	// Create file in internal
	internalGo := `package internal

// InternalFunc is an internal function
func InternalFunc() string {
	return "internal"
}
`
	err = os.WriteFile(filepath.Join(internalDir, "internal.go"), []byte(internalGo), 0644)
	if err != nil {
		t.Fatalf("Failed to create internal file: %v", err)
	}

	// Create external directory
	externalDir := filepath.Join(dir, "external")
	err = os.MkdirAll(externalDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create external dir: %v", err)
	}

	// Create file in external
	externalGo := `package external

// ExternalFunc is an external function
func ExternalFunc() string {
	return "external"
}
`
	err = os.WriteFile(filepath.Join(externalDir, "external.go"), []byte(externalGo), 0644)
	if err != nil {
		t.Fatalf("Failed to create external file: %v", err)
	}
}
