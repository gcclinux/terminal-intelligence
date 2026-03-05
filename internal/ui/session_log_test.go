package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/user/terminal-intelligence/internal/types"
)

func TestAppendMessageToSessionLog(t *testing.T) {
	// Create a temporary workspace directory
	tmpDir := t.TempDir()

	// Create AI chat pane with workspace root
	pane := &AIChatPane{
		workspaceRoot: tmpDir,
		messages:      []types.ChatMessage{},
	}

	// Add a user message
	userMsg := types.ChatMessage{
		Role:      "user",
		Content:   "Hello, AI!",
		Timestamp: time.Now(),
	}
	pane.messages = append(pane.messages, userMsg)
	pane.appendMessageToSessionLog(userMsg)

	// Add an assistant message
	assistantMsg := types.ChatMessage{
		Role:      "assistant",
		Content:   "Hello! How can I help you?",
		Timestamp: time.Now(),
	}
	pane.messages = append(pane.messages, assistantMsg)
	pane.appendMessageToSessionLog(assistantMsg)

	// Verify .ti directory was created
	tiDir := filepath.Join(tmpDir, ".ti")
	if _, err := os.Stat(tiDir); os.IsNotExist(err) {
		t.Fatalf("Expected .ti directory to be created, but it doesn't exist")
	}

	// Verify session file was created
	if pane.sessionFile == "" {
		t.Fatalf("Expected sessionFile to be set, but it's empty")
	}

	// Read the session file
	content, err := os.ReadFile(pane.sessionFile)
	if err != nil {
		t.Fatalf("Failed to read session file: %v", err)
	}

	contentStr := string(content)

	// Verify user message is in the file
	if !strings.Contains(contentStr, "user") {
		t.Errorf("Expected session file to contain 'user', but it doesn't")
	}
	if !strings.Contains(contentStr, "Hello, AI!") {
		t.Errorf("Expected session file to contain 'Hello, AI!', but it doesn't")
	}

	// Verify assistant message is in the file
	if !strings.Contains(contentStr, "assistant") {
		t.Errorf("Expected session file to contain 'assistant', but it doesn't")
	}
	if !strings.Contains(contentStr, "Hello! How can I help you?") {
		t.Errorf("Expected session file to contain 'Hello! How can I help you?', but it doesn't")
	}
}

func TestSessionFileReusedAcrossMessages(t *testing.T) {
	// Create a temporary workspace directory
	tmpDir := t.TempDir()

	// Create AI chat pane with workspace root
	pane := &AIChatPane{
		workspaceRoot: tmpDir,
		messages:      []types.ChatMessage{},
	}

	// Add first message
	msg1 := types.ChatMessage{
		Role:      "user",
		Content:   "First message",
		Timestamp: time.Now(),
	}
	pane.appendMessageToSessionLog(msg1)
	firstSessionFile := pane.sessionFile

	// Add second message
	msg2 := types.ChatMessage{
		Role:      "assistant",
		Content:   "Second message",
		Timestamp: time.Now(),
	}
	pane.appendMessageToSessionLog(msg2)
	secondSessionFile := pane.sessionFile

	// Verify the same session file is used
	if firstSessionFile != secondSessionFile {
		t.Errorf("Expected same session file to be reused, but got different files: %s vs %s", firstSessionFile, secondSessionFile)
	}

	// Verify both messages are in the same file
	content, err := os.ReadFile(pane.sessionFile)
	if err != nil {
		t.Fatalf("Failed to read session file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "First message") {
		t.Errorf("Expected session file to contain 'First message'")
	}
	if !strings.Contains(contentStr, "Second message") {
		t.Errorf("Expected session file to contain 'Second message'")
	}
}

func TestSessionFileResetOnClearHistory(t *testing.T) {
	// Create a temporary workspace directory
	tmpDir := t.TempDir()

	// Create AI chat pane with workspace root
	pane := &AIChatPane{
		workspaceRoot: tmpDir,
		messages:      []types.ChatMessage{},
	}

	// Add a message
	msg := types.ChatMessage{
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	pane.appendMessageToSessionLog(msg)

	// Verify session file is set
	if pane.sessionFile == "" {
		t.Fatalf("Expected sessionFile to be set")
	}

	// Clear history
	pane.ClearHistory()

	// Verify session file is reset
	if pane.sessionFile != "" {
		t.Errorf("Expected sessionFile to be reset after ClearHistory, but got: %s", pane.sessionFile)
	}
}

func TestNoSessionFileWithoutWorkspaceRoot(t *testing.T) {
	// Create AI chat pane without workspace root
	pane := &AIChatPane{
		workspaceRoot: "",
		messages:      []types.ChatMessage{},
	}

	// Add a message
	msg := types.ChatMessage{
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	pane.appendMessageToSessionLog(msg)

	// Verify no session file is created
	if pane.sessionFile != "" {
		t.Errorf("Expected no session file when workspaceRoot is empty, but got: %s", pane.sessionFile)
	}
}

func TestEnsureTiDirInGitignore_NewGitignore(t *testing.T) {
	// Create a temporary workspace directory
	tmpDir := t.TempDir()

	// Create AI chat pane with workspace root
	pane := &AIChatPane{
		workspaceRoot: tmpDir,
		messages:      []types.ChatMessage{},
	}

	// Add a message (this will create .ti/ and update .gitignore)
	msg := types.ChatMessage{
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	pane.appendMessageToSessionLog(msg)

	// Verify .gitignore was created
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		t.Fatalf("Expected .gitignore to be created, but it doesn't exist")
	}

	// Read .gitignore content
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, ".ti/") {
		t.Errorf("Expected .gitignore to contain '.ti/', but got: %s", contentStr)
	}
}

func TestEnsureTiDirInGitignore_ExistingGitignore(t *testing.T) {
	// Create a temporary workspace directory
	tmpDir := t.TempDir()

	// Create existing .gitignore with some content
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	existingContent := "node_modules/\n*.log\n"
	err := os.WriteFile(gitignorePath, []byte(existingContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Create AI chat pane with workspace root
	pane := &AIChatPane{
		workspaceRoot: tmpDir,
		messages:      []types.ChatMessage{},
	}

	// Add a message (this will create .ti/ and update .gitignore)
	msg := types.ChatMessage{
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	pane.appendMessageToSessionLog(msg)

	// Read .gitignore content
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	contentStr := string(content)

	// Verify original content is preserved
	if !strings.Contains(contentStr, "node_modules/") {
		t.Errorf("Expected .gitignore to preserve 'node_modules/', but got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "*.log") {
		t.Errorf("Expected .gitignore to preserve '*.log', but got: %s", contentStr)
	}

	// Verify .ti/ was added
	if !strings.Contains(contentStr, ".ti/") {
		t.Errorf("Expected .gitignore to contain '.ti/', but got: %s", contentStr)
	}
}

func TestEnsureTiDirInGitignore_AlreadyPresent(t *testing.T) {
	// Create a temporary workspace directory
	tmpDir := t.TempDir()

	// Create existing .gitignore with .ti/ already present
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	existingContent := "node_modules/\n.ti/\n*.log\n"
	err := os.WriteFile(gitignorePath, []byte(existingContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Create AI chat pane with workspace root
	pane := &AIChatPane{
		workspaceRoot: tmpDir,
		messages:      []types.ChatMessage{},
	}

	// Add a message (this will create .ti/ but should not duplicate .ti/ in .gitignore)
	msg := types.ChatMessage{
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	pane.appendMessageToSessionLog(msg)

	// Read .gitignore content
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	contentStr := string(content)

	// Count occurrences of .ti/
	count := strings.Count(contentStr, ".ti/")
	if count != 1 {
		t.Errorf("Expected .gitignore to contain '.ti/' exactly once, but found %d occurrences: %s", count, contentStr)
	}

	// Verify content is unchanged
	if contentStr != existingContent {
		t.Errorf("Expected .gitignore to remain unchanged, but got: %s", contentStr)
	}
}

func TestEnsureTiDirInGitignore_VariantFormats(t *testing.T) {
	testCases := []struct {
		name            string
		existingContent string
		shouldAdd       bool
	}{
		{
			name:            ".ti/ already present",
			existingContent: "node_modules/\n.ti/\n",
			shouldAdd:       false,
		},
		{
			name:            ".ti already present (without slash)",
			existingContent: "node_modules/\n.ti\n",
			shouldAdd:       false,
		},
		{
			name:            "/.ti/ already present (with leading slash)",
			existingContent: "node_modules/\n/.ti/\n",
			shouldAdd:       false,
		},
		{
			name:            ".ti not present",
			existingContent: "node_modules/\n*.log\n",
			shouldAdd:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary workspace directory
			tmpDir := t.TempDir()

			// Create existing .gitignore
			gitignorePath := filepath.Join(tmpDir, ".gitignore")
			err := os.WriteFile(gitignorePath, []byte(tc.existingContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create .gitignore: %v", err)
			}

			// Create AI chat pane with workspace root
			pane := &AIChatPane{
				workspaceRoot: tmpDir,
				messages:      []types.ChatMessage{},
			}

			// Add a message
			msg := types.ChatMessage{
				Role:      "user",
				Content:   "Test message",
				Timestamp: time.Now(),
			}
			pane.appendMessageToSessionLog(msg)

			// Read .gitignore content
			content, err := os.ReadFile(gitignorePath)
			if err != nil {
				t.Fatalf("Failed to read .gitignore: %v", err)
			}

			contentStr := string(content)

			if tc.shouldAdd {
				// Verify .ti/ was added
				if !strings.Contains(contentStr, ".ti/") {
					t.Errorf("Expected .gitignore to contain '.ti/', but got: %s", contentStr)
				}
				// Verify it was added only once
				if strings.Count(contentStr, ".ti/") != 1 {
					t.Errorf("Expected .gitignore to contain '.ti/' exactly once, but got: %s", contentStr)
				}
			} else {
				// Verify content is unchanged
				if contentStr != tc.existingContent {
					t.Errorf("Expected .gitignore to remain unchanged, but got: %s", contentStr)
				}
			}
		})
	}
}

func TestEnsureTiDirInGitignore_NoWorkspaceRoot(t *testing.T) {
	// Create AI chat pane without workspace root
	pane := &AIChatPane{
		workspaceRoot: "",
		messages:      []types.ChatMessage{},
	}

	// Add a message
	msg := types.ChatMessage{
		Role:      "user",
		Content:   "Test message",
		Timestamp: time.Now(),
	}
	pane.appendMessageToSessionLog(msg)

	// This should not panic or create any files
	// Test passes if no panic occurs
}

func TestEnsureTiDirInGitignore_OnlyOnFirstCreation(t *testing.T) {
	// Create a temporary workspace directory
	tmpDir := t.TempDir()

	// Create AI chat pane with workspace root
	pane := &AIChatPane{
		workspaceRoot: tmpDir,
		messages:      []types.ChatMessage{},
	}

	// Add first message (creates .ti/ and .gitignore)
	msg1 := types.ChatMessage{
		Role:      "user",
		Content:   "First message",
		Timestamp: time.Now(),
	}
	pane.appendMessageToSessionLog(msg1)

	// Read .gitignore content after first message
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	content1, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	// Add second message (should not modify .gitignore)
	msg2 := types.ChatMessage{
		Role:      "assistant",
		Content:   "Second message",
		Timestamp: time.Now(),
	}
	pane.appendMessageToSessionLog(msg2)

	// Read .gitignore content after second message
	content2, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	// Verify .gitignore is unchanged
	if string(content1) != string(content2) {
		t.Errorf("Expected .gitignore to remain unchanged after second message, but it changed from:\n%s\nto:\n%s", string(content1), string(content2))
	}

	// Verify .ti/ appears only once
	if strings.Count(string(content2), ".ti/") != 1 {
		t.Errorf("Expected .gitignore to contain '.ti/' exactly once, but got: %s", string(content2))
	}
}
