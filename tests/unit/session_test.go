package unit

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/terminal-intelligence/internal/types"
	"github.com/user/terminal-intelligence/internal/ui"
)

// TestSession_ExitWithUnsavedChanges tests exit confirmation with unsaved changes
func TestSession_ExitWithUnsavedChanges(t *testing.T) {
	t.Run("shows confirmation dialog when exiting with unsaved changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Create and load a file
		editorPane := app.GetEditorPane()
		filename := "test.sh"
		content := "#!/bin/bash\necho 'test'"

		err := editorPane.GetFileManager().CreateFile(filename, content)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err = editorPane.LoadFile(filename)
		if err != nil {
			t.Fatalf("Failed to load file: %v", err)
		}

		// Modify content
		editorPane.SetContent(content + "\n# Modified")

		// Verify unsaved changes
		if !editorPane.HasUnsavedChanges() {
			t.Error("Expected unsaved changes after modification")
		}

		// Try to quit
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		_, cmd := app.Update(keyMsg)

		// Should show confirmation dialog, not quit
		if cmd != nil {
			t.Error("Should not quit immediately with unsaved changes")
		}

		if !app.IsShowingExitConfirmation() {
			t.Error("Expected exit confirmation dialog to be shown")
		}
	})

	t.Run("confirms exit without saving", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Create and load a file with unsaved changes
		editorPane := app.GetEditorPane()
		filename := "test.sh"
		content := "#!/bin/bash\necho 'test'"

		err := editorPane.GetFileManager().CreateFile(filename, content)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err = editorPane.LoadFile(filename)
		if err != nil {
			t.Fatalf("Failed to load file: %v", err)
		}

		editorPane.SetContent(content + "\n# Modified")

		// Try to quit
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		_, _ = app.Update(keyMsg)

		// Confirm exit
		confirmMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		_, cmd := app.Update(confirmMsg)

		// Should quit now
		if cmd == nil {
			t.Error("Expected quit command after confirming exit")
		}
	})

	t.Run("cancels exit and returns to editing", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Create and load a file with unsaved changes
		editorPane := app.GetEditorPane()
		filename := "test.sh"
		content := "#!/bin/bash\necho 'test'"

		err := editorPane.GetFileManager().CreateFile(filename, content)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err = editorPane.LoadFile(filename)
		if err != nil {
			t.Fatalf("Failed to load file: %v", err)
		}

		editorPane.SetContent(content + "\n# Modified")

		// Try to quit
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		_, _ = app.Update(keyMsg)

		// Cancel exit
		cancelMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		_, cmd := app.Update(cancelMsg)

		// Should not quit
		if cmd != nil {
			t.Error("Should not quit after canceling exit")
		}

		// Should hide confirmation dialog
		if app.IsShowingExitConfirmation() {
			t.Error("Expected exit confirmation dialog to be hidden")
		}

		// Should still have unsaved changes
		if !editorPane.HasUnsavedChanges() {
			t.Error("Expected unsaved changes to remain after canceling exit")
		}
	})

	t.Run("exits immediately without unsaved changes", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Try to quit without unsaved changes
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		_, cmd := app.Update(keyMsg)

		// Should quit immediately
		if cmd == nil {
			t.Error("Expected quit command when no unsaved changes")
		}

		// Should not show confirmation dialog
		if app.IsShowingExitConfirmation() {
			t.Error("Should not show exit confirmation without unsaved changes")
		}
	})
}

// TestSession_AIHistoryClearing tests AI conversation history clearing
func TestSession_AIHistoryClearing(t *testing.T) {
	t.Run("clears AI history on normal exit", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Add some messages to AI history
		aiPane := app.GetAIPane()
		aiPane.DisplayResponse("Test response 1")
		aiPane.DisplayResponse("Test response 2")

		// Verify history has messages
		if len(aiPane.GetHistory()) == 0 {
			t.Error("Expected AI history to have messages")
		}

		// Quit (no unsaved changes)
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		_, _ = app.Update(keyMsg)

		// History should be cleared
		if len(aiPane.GetHistory()) != 0 {
			t.Error("Expected AI history to be cleared on exit")
		}
	})

	t.Run("clears AI history on confirmed exit with unsaved changes", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Add some messages to AI history
		aiPane := app.GetAIPane()
		aiPane.DisplayResponse("Test response 1")
		aiPane.DisplayResponse("Test response 2")

		// Create unsaved changes
		editorPane := app.GetEditorPane()
		filename := "test.sh"
		content := "#!/bin/bash\necho 'test'"

		err := editorPane.GetFileManager().CreateFile(filename, content)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err = editorPane.LoadFile(filename)
		if err != nil {
			t.Fatalf("Failed to load file: %v", err)
		}

		editorPane.SetContent(content + "\n# Modified")

		// Try to quit
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		_, _ = app.Update(keyMsg)

		// Confirm exit
		confirmMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		_, _ = app.Update(confirmMsg)

		// History should be cleared
		if len(aiPane.GetHistory()) != 0 {
			t.Error("Expected AI history to be cleared on confirmed exit")
		}
	})

	t.Run("does not clear AI history on canceled exit", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Add some messages to AI history
		aiPane := app.GetAIPane()
		aiPane.DisplayResponse("Test response 1")
		aiPane.DisplayResponse("Test response 2")

		initialHistoryLen := len(aiPane.GetHistory())

		// Create unsaved changes
		editorPane := app.GetEditorPane()
		filename := "test.sh"
		content := "#!/bin/bash\necho 'test'"

		err := editorPane.GetFileManager().CreateFile(filename, content)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		err = editorPane.LoadFile(filename)
		if err != nil {
			t.Fatalf("Failed to load file: %v", err)
		}

		editorPane.SetContent(content + "\n# Modified")

		// Try to quit
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		_, _ = app.Update(keyMsg)

		// Cancel exit
		cancelMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		_, _ = app.Update(cancelMsg)

		// History should not be cleared
		if len(aiPane.GetHistory()) != initialHistoryLen {
			t.Errorf("Expected AI history to remain unchanged, had %d messages, now has %d",
				initialHistoryLen, len(aiPane.GetHistory()))
		}
	})
}

// TestSession_ExitConfirmationDialog tests the exit confirmation dialog behavior
func TestSession_ExitConfirmationDialog(t *testing.T) {
	t.Run("accepts Y for yes", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize and create unsaved changes
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		editorPane := app.GetEditorPane()
		editorPane.SetContent("modified content")

		// Trigger confirmation
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		_, _ = app.Update(keyMsg)

		// Press Y (uppercase)
		confirmMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}}
		_, cmd := app.Update(confirmMsg)

		if cmd == nil {
			t.Error("Expected quit command after pressing Y")
		}
	})

	t.Run("accepts N for no", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize and create unsaved changes
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		editorPane := app.GetEditorPane()
		editorPane.SetContent("modified content")

		// Trigger confirmation
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		_, _ = app.Update(keyMsg)

		// Press N (uppercase)
		cancelMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}}
		_, cmd := app.Update(cancelMsg)

		if cmd != nil {
			t.Error("Should not quit after pressing N")
		}

		if app.IsShowingExitConfirmation() {
			t.Error("Should hide confirmation dialog after pressing N")
		}
	})

	t.Run("accepts esc to cancel", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config, "test")

		// Initialize and create unsaved changes
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		editorPane := app.GetEditorPane()
		editorPane.SetContent("modified content")

		// Trigger confirmation
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		_, _ = app.Update(keyMsg)

		// Press esc
		escMsg := tea.KeyMsg{Type: tea.KeyEsc}
		_, cmd := app.Update(escMsg)

		if cmd != nil {
			t.Error("Should not quit after pressing esc")
		}

		if app.IsShowingExitConfirmation() {
			t.Error("Should hide confirmation dialog after pressing esc")
		}
	})
}
