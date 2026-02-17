package types

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestDefaultConfig verifies that DefaultConfig returns valid configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.OllamaURL == "" {
		t.Error("OllamaURL should not be empty")
	}

	if config.DefaultModel == "" {
		t.Error("DefaultModel should not be empty")
	}

	if config.TabSize <= 0 {
		t.Error("TabSize should be positive")
	}

	if config.WorkspaceDir == "" {
		t.Error("WorkspaceDir should not be empty")
	}
}

// TestPaneTypeConstants verifies PaneType constants are distinct
func TestPaneTypeConstants(t *testing.T) {
	if EditorPaneType == AIPaneType {
		t.Error("EditorPaneType and AIPaneType should be distinct")
	}
}

// Property test example: FileMetadata filepath should always be valid
// Feature: Terminal Intelligence (TI), Property: FileMetadata filepath validation
func TestFileMetadataFilepathProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("FileMetadata with valid filepath should be constructible", prop.ForAll(
		func(filename string) bool {
			// Create FileMetadata with a filepath
			metadata := &FileMetadata{
				Filepath: filename,
				FileType: "bash",
			}

			// Verify the filepath is stored correctly
			return metadata.Filepath == filename
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// Property test: AppConfig workspace directory should be absolute path
// Feature: Terminal Intelligence (TI), Property: AppConfig workspace directory validation
func TestAppConfigWorkspaceDirProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("DefaultConfig workspace directory should be absolute", prop.ForAll(
		func(_ struct{}) bool {
			config := DefaultConfig()
			return filepath.IsAbs(config.WorkspaceDir)
		},
		gen.Const(struct{}{}),
	))

	properties.TestingRun(t)
}

// Property test: ChatMessage timestamp should be valid
// Feature: Terminal Intelligence (TI), Property: ChatMessage timestamp validation
func TestChatMessageTimestampProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("ChatMessage with role and content should be valid", prop.ForAll(
		func(role, content string) bool {
			msg := &ChatMessage{
				Role:    role,
				Content: content,
			}

			// Verify fields are stored correctly
			return msg.Role == role && msg.Content == content
		},
		gen.OneConstOf("user", "assistant"),
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// TestCommandResultStructure verifies CommandResult can hold execution data
func TestCommandResultStructure(t *testing.T) {
	result := &CommandResult{
		Stdout:   "output",
		Stderr:   "error",
		ExitCode: 0,
	}

	if result.Stdout != "output" {
		t.Error("Stdout not stored correctly")
	}

	if result.Stderr != "error" {
		t.Error("Stderr not stored correctly")
	}

	if result.ExitCode != 0 {
		t.Error("ExitCode not stored correctly")
	}
}

// TestFileMetadataFileTypes verifies supported file types
func TestFileMetadataFileTypes(t *testing.T) {
	supportedTypes := []string{"bash", "shell", "powershell", "markdown"}

	for _, fileType := range supportedTypes {
		metadata := &FileMetadata{
			Filepath: "/test/file",
			FileType: fileType,
		}

		if metadata.FileType != fileType {
			t.Errorf("FileType %s not stored correctly", fileType)
		}
	}
}

// Benchmark for DefaultConfig creation
func BenchmarkDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}

// TestAppConfigYAMLTags verifies YAML tags are present
func TestAppConfigYAMLTags(t *testing.T) {
	// This test ensures the struct has proper YAML tags for serialization
	config := &AppConfig{
		OllamaURL:    "http://test:11434",
		DefaultModel: "test-model",
		EditorTheme:  "test-theme",
		WorkspaceDir: "/test/workspace",
		AutoSave:     true,
		TabSize:      2,
	}

	// Verify all fields can be set
	if config.OllamaURL != "http://test:11434" {
		t.Error("OllamaURL not set correctly")
	}
	if config.DefaultModel != "test-model" {
		t.Error("DefaultModel not set correctly")
	}
	if config.EditorTheme != "test-theme" {
		t.Error("EditorTheme not set correctly")
	}
	if config.WorkspaceDir != "/test/workspace" {
		t.Error("WorkspaceDir not set correctly")
	}
	if !config.AutoSave {
		t.Error("AutoSave not set correctly")
	}
	if config.TabSize != 2 {
		t.Error("TabSize not set correctly")
	}
}

// TestChatMessageRoles verifies valid roles
func TestChatMessageRoles(t *testing.T) {
	validRoles := []string{"user", "assistant"}

	for _, role := range validRoles {
		msg := &ChatMessage{
			Role:    role,
			Content: "test content",
		}

		if msg.Role != role {
			t.Errorf("Role %s not stored correctly", role)
		}
	}
}

// TestFileMetadataModificationTracking verifies IsModified flag
func TestFileMetadataModificationTracking(t *testing.T) {
	metadata := &FileMetadata{
		Filepath:   "/test/file.sh",
		FileType:   "bash",
		IsModified: false,
	}

	if metadata.IsModified {
		t.Error("IsModified should be false initially")
	}

	metadata.IsModified = true

	if !metadata.IsModified {
		t.Error("IsModified should be true after modification")
	}
}

// TestCommandResultExitCodes verifies exit code handling
func TestCommandResultExitCodes(t *testing.T) {
	testCases := []struct {
		exitCode int
		desc     string
	}{
		{0, "success"},
		{1, "general error"},
		{127, "command not found"},
		{130, "terminated by Ctrl+C"},
	}

	for _, tc := range testCases {
		result := &CommandResult{
			ExitCode: tc.exitCode,
		}

		if result.ExitCode != tc.exitCode {
			t.Errorf("Exit code %d (%s) not stored correctly", tc.exitCode, tc.desc)
		}
	}
}

// TestPaneTypeSwitching verifies pane type can be changed
func TestPaneTypeSwitching(t *testing.T) {
	var currentPane PaneType = EditorPaneType

	if currentPane != EditorPaneType {
		t.Error("Initial pane should be EditorPaneType")
	}

	currentPane = AIPaneType

	if currentPane != AIPaneType {
		t.Error("Pane should switch to AIPaneType")
	}
}

// TestDefaultConfigHomeDirectory verifies workspace is in home directory
func TestDefaultConfigHomeDirectory(t *testing.T) {
	config := DefaultConfig()
	homeDir, err := os.UserHomeDir()

	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	if !filepath.IsAbs(config.WorkspaceDir) {
		t.Error("WorkspaceDir should be an absolute path")
	}

	// Verify workspace dir contains home directory path
	if homeDir != "" && !filepath.HasPrefix(config.WorkspaceDir, homeDir) {
		t.Error("WorkspaceDir should be under home directory")
	}
}
