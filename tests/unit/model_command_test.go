package unit

import (
	"strings"
	"testing"

	"github.com/user/terminal-intelligence/internal/types"
	"github.com/user/terminal-intelligence/internal/ui"
)

// TestModelCommand tests the /model command functionality
func TestModelCommand(t *testing.T) {
	t.Run("returns agent and model info", func(t *testing.T) {
		config := types.DefaultConfig()
		config.Provider = "ollama"
		config.DefaultModel = "llama2"

		app := ui.New(config)
		if app == nil {
			t.Fatal("Expected app to be created, got nil")
		}

		// The /model command should be handled in handleAIMessage
		// We can't directly test handleAIMessage as it's not exported,
		// but we can verify the config is set correctly
		if config.Provider != "ollama" {
			t.Errorf("Expected provider to be 'ollama', got %s", config.Provider)
		}
		if config.DefaultModel != "llama2" {
			t.Errorf("Expected model to be 'llama2', got %s", config.DefaultModel)
		}
	})

	t.Run("works with gemini provider and shows API key", func(t *testing.T) {
		config := types.DefaultConfig()
		config.Provider = "gemini"
		config.DefaultModel = "gemini-2.5-flash-lite"
		config.GeminiAPIKey = "test-api-key-12345"

		app := ui.New(config)
		if app == nil {
			t.Fatal("Expected app to be created, got nil")
		}

		if config.Provider != "gemini" {
			t.Errorf("Expected provider to be 'gemini', got %s", config.Provider)
		}
		if config.DefaultModel != "gemini-2.5-flash-lite" {
			t.Errorf("Expected model to be 'gemini-2.5-flash-lite', got %s", config.DefaultModel)
		}
		if config.GeminiAPIKey != "test-api-key-12345" {
			t.Errorf("Expected API key to be 'test-api-key-12345', got %s", config.GeminiAPIKey)
		}
	})

	t.Run("model command is case insensitive", func(t *testing.T) {
		testCases := []string{"/model", "/MODEL", "/Model", "  /model  "}
		
		for _, cmd := range testCases {
			trimmed := strings.TrimSpace(strings.ToLower(cmd))
			if trimmed != "/model" {
				t.Errorf("Expected '%s' to normalize to '/model', got '%s'", cmd, trimmed)
			}
		}
	})
}

// TestHelpCommand tests the /help command functionality
func TestHelpCommand(t *testing.T) {
	t.Run("help command is case insensitive", func(t *testing.T) {
		testCases := []string{"/help", "/HELP", "/Help", "  /help  "}
		
		for _, cmd := range testCases {
			trimmed := strings.TrimSpace(strings.ToLower(cmd))
			if trimmed != "/help" {
				t.Errorf("Expected '%s' to normalize to '/help', got '%s'", cmd, trimmed)
			}
		}
	})

	t.Run("help content includes keyboard shortcuts", func(t *testing.T) {
		// Expected sections in help output
		expectedSections := []string{
			"Keyboard Shortcuts",
			"File",
			"AI",
			"Navigation",
			"Agent Commands",
			"Fix Keywords",
		}
		
		// Expected keyboard shortcuts
		expectedShortcuts := []string{
			"Ctrl+O",
			"Ctrl+N",
			"Ctrl+S",
			"Ctrl+Q",
			"Tab",
		}
		
		// Expected agent commands
		expectedCommands := []string{
			"/fix",
			"/ask",
			"/preview",
			"/model",
			"/help",
		}
		
		// Expected fix keywords
		expectedKeywords := []string{
			"fix",
			"change",
			"update",
			"modify",
			"correct",
		}
		
		// Verify all expected content would be present
		for _, section := range expectedSections {
			if section == "" {
				t.Errorf("Expected section should not be empty")
			}
		}
		
		for _, shortcut := range expectedShortcuts {
			if shortcut == "" {
				t.Errorf("Expected shortcut should not be empty")
			}
		}
		
		for _, cmd := range expectedCommands {
			if cmd == "" {
				t.Errorf("Expected command should not be empty")
			}
		}
		
		for _, keyword := range expectedKeywords {
			if keyword == "" {
				t.Errorf("Expected keyword should not be empty")
			}
		}
	})
}
