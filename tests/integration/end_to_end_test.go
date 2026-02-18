package integration

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/terminal-intelligence/internal/types"
	"github.com/user/terminal-intelligence/internal/ui"
)

// TestEndToEnd_CreateEditSaveWorkflow tests the complete workflow of creating, editing, and saving a file
func TestEndToEnd_CreateEditSaveWorkflow(t *testing.T) {
	t.Run("create file, edit, and save", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Get editor pane
		editorPane := app.GetEditorPane()
		fm := editorPane.GetFileManager()

		// Create a new file
		filename := "script.sh"
		initialContent := "#!/bin/bash\n"
		err := fm.CreateFile(filename, initialContent)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Load file into editor
		err = editorPane.LoadFile(filename)
		if err != nil {
			t.Fatalf("Failed to load file: %v", err)
		}

		// Verify content loaded
		if editorPane.GetContent() != initialContent {
			t.Errorf("Expected content %q, got %q", initialContent, editorPane.GetContent())
		}

		// Edit the file
		newContent := initialContent + "echo 'Hello, World!'\n"
		editorPane.SetContent(newContent)

		// Verify unsaved changes
		if !editorPane.HasUnsavedChanges() {
			t.Error("Expected unsaved changes after editing")
		}

		// Save the file using keyboard shortcut
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlS}
		_, _ = app.Update(keyMsg)

		// Verify file was saved
		if editorPane.HasUnsavedChanges() {
			t.Error("Expected no unsaved changes after saving")
		}

		// Verify file content on disk
		savedContent, err := fm.ReadFile(filename)
		if err != nil {
			t.Fatalf("Failed to read saved file: %v", err)
		}

		if savedContent != newContent {
			t.Errorf("Expected saved content %q, got %q", newContent, savedContent)
		}
	})
}

// TestEndToEnd_AIInteractionWithCodeContext tests AI interaction with code context
func TestEndToEnd_AIInteractionWithCodeContext(t *testing.T) {
	t.Run("send message to AI with code context", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Load code into editor
		editorPane := app.GetEditorPane()
		codeContent := "#!/bin/bash\necho 'test'\n"
		editorPane.SetContent(codeContent)

		// Switch to AI pane
		app.SetActivePane(types.AIPaneType)

		// Get AI pane
		aiPane := app.GetAIPane()

		// Verify initial history is empty
		if len(aiPane.GetHistory()) != 0 {
			t.Error("Expected empty AI history initially")
		}

		// Send a message (this will include code context)
		// Note: We can't test actual Ollama responses without a running server,
		// but we can verify the message is added to history
		cmd := aiPane.SendMessage("Explain this code", codeContent)
		if cmd == nil {
			t.Error("Expected command from SendMessage")
		}

		// Verify message was added to history
		history := aiPane.GetHistory()
		if len(history) != 1 {
			t.Errorf("Expected 1 message in history, got %d", len(history))
		}

		if history[0].Role != "user" {
			t.Errorf("Expected user message, got %s", history[0].Role)
		}

		if !history[0].ContextIncluded {
			t.Error("Expected context to be included in message")
		}
	})
}

// TestEndToEnd_PaneSwitchingWorkflow tests switching between panes
func TestEndToEnd_PaneSwitchingWorkflow(t *testing.T) {
	t.Run("switch between editor and AI pane", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Verify initial state (editor focused)
		if app.GetActivePane() != types.EditorPaneType {
			t.Error("Expected editor pane to be active initially")
		}

		// Switch to AI pane (Input area)
		keyMsg := tea.KeyMsg{Type: tea.KeyTab}
		_, _ = app.Update(keyMsg)

		if app.GetActivePane() != types.AIPaneType {
			t.Error("Expected AI pane to be active after tab")
		}

		// Switch to AI Response area (still AI pane)
		_, _ = app.Update(keyMsg)

		if app.GetActivePane() != types.AIPaneType {
			t.Error("Expected AI pane (response) to be active after second tab")
		}

		// Switch back to editor
		_, _ = app.Update(keyMsg)

		if app.GetActivePane() != types.EditorPaneType {
			t.Error("Expected editor pane to be active after third tab")
		}
	})
}

// TestEndToEnd_ErrorRecoveryScenarios tests error recovery
func TestEndToEnd_ErrorRecoveryScenarios(t *testing.T) {
	t.Run("handle file not found error", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Try to load non-existent file
		editorPane := app.GetEditorPane()
		err := editorPane.LoadFile("nonexistent.sh")

		// Should return error
		if err == nil {
			t.Error("Expected error when loading non-existent file")
		}

		// App should still be functional
		if app.GetActivePane() != types.EditorPaneType {
			t.Error("App should remain functional after error")
		}
	})

	t.Run("handle save error gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Try to save without loading a file
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlS}
		_, _ = app.Update(keyMsg)

		// App should still be functional
		if app.GetActivePane() != types.EditorPaneType {
			t.Error("App should remain functional after save error")
		}
	})
}

// TestEndToEnd_MultipleFileEditing tests editing multiple files in sequence
func TestEndToEnd_MultipleFileEditing(t *testing.T) {
	t.Run("edit multiple files sequentially", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		editorPane := app.GetEditorPane()
		fm := editorPane.GetFileManager()

		// Create and edit first file
		file1 := "script1.sh"
		content1 := "#!/bin/bash\necho 'file1'\n"
		err := fm.CreateFile(file1, content1)
		if err != nil {
			t.Fatalf("Failed to create file1: %v", err)
		}

		err = editorPane.LoadFile(file1)
		if err != nil {
			t.Fatalf("Failed to load file1: %v", err)
		}

		editorPane.SetContent(content1 + "# Modified\n")

		// Save first file
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlS}
		_, _ = app.Update(keyMsg)

		// Create and edit second file
		file2 := "script2.sh"
		content2 := "#!/bin/bash\necho 'file2'\n"
		err = fm.CreateFile(file2, content2)
		if err != nil {
			t.Fatalf("Failed to create file2: %v", err)
		}

		err = editorPane.LoadFile(file2)
		if err != nil {
			t.Fatalf("Failed to load file2: %v", err)
		}

		// Verify content switched to file2
		if editorPane.GetContent() != content2 {
			t.Errorf("Expected content %q, got %q", content2, editorPane.GetContent())
		}

		// Verify no unsaved changes for file2
		if editorPane.HasUnsavedChanges() {
			t.Error("Expected no unsaved changes for newly loaded file")
		}
	})
}

// TestEndToEnd_ExitWorkflow tests the exit workflow
func TestEndToEnd_ExitWorkflow(t *testing.T) {
	t.Run("exit with unsaved changes workflow", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Create unsaved changes
		editorPane := app.GetEditorPane()
		editorPane.SetContent("unsaved content")

		// Try to quit
		quitMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
		_, cmd := app.Update(quitMsg)

		// Should show confirmation, not quit
		if cmd != nil {
			t.Error("Should not quit immediately with unsaved changes")
		}

		if !app.IsShowingExitConfirmation() {
			t.Error("Expected exit confirmation dialog")
		}

		// Cancel exit
		cancelMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		_, cmd = app.Update(cancelMsg)

		if cmd != nil {
			t.Error("Should not quit after canceling")
		}

		// Save changes
		saveMsg := tea.KeyMsg{Type: tea.KeyCtrlS}
		_, _ = app.Update(saveMsg)

		// Now quit should work (but will fail because no file is loaded)
		// This is expected behavior
	})
}

// TestEndToEnd_TerminalResizeHandling tests terminal resize during operation
func TestEndToEnd_TerminalResizeHandling(t *testing.T) {
	t.Run("resize terminal during editing", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize with initial size
		msg1 := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg1)

		// Load a file
		editorPane := app.GetEditorPane()
		fm := editorPane.GetFileManager()
		filename := "test.sh"
		content := "#!/bin/bash\necho 'test'\n"
		
		err := fm.CreateFile(filename, content)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err = editorPane.LoadFile(filename)
		if err != nil {
			t.Fatalf("Failed to load file: %v", err)
		}

		// Resize terminal
		msg2 := tea.WindowSizeMsg{Width: 120, Height: 40}
		_, _ = app.Update(msg2)

		// Verify panes resized
		if editorPane.GetWidth() != 64 {  // (120 / 2) + 4
			t.Errorf("Expected editor width 64, got %d", editorPane.GetWidth())
		}

		if editorPane.GetHeight() != 33 {  // 40 - 3 for header - 3 for editor title - 1 for status bar
			t.Errorf("Expected editor height 33, got %d", editorPane.GetHeight())
		}

		// Verify content is still intact
		if editorPane.GetContent() != content {
			t.Error("Content should remain intact after resize")
		}
	})
}
