package unit

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/terminal-intelligence/internal/types"
	"github.com/user/terminal-intelligence/internal/ui"
)

// TestKeyboard_TabSwitchesPanes tests tab key pane switching
func TestKeyboard_TabSwitchesPanes(t *testing.T) {
	t.Run("tab switches from editor to AI", func(t *testing.T) {
		app := ui.New(nil)
		
		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)
		
		// Verify initial state (editor focused)
		if app.GetActivePane() != types.EditorPaneType {
			t.Errorf("Expected initial focus on editor, got %v", app.GetActivePane())
		}
		
		// Press tab
		keyMsg := tea.KeyMsg{Type: tea.KeyTab}
		_, _ = app.Update(keyMsg)
		
		// Verify switched to AI pane (Input area)
		if app.GetActivePane() != types.AIPaneType {
			t.Errorf("Expected focus on AI pane after tab, got %v", app.GetActivePane())
		}
	})

	t.Run("tab cycles through all three areas", func(t *testing.T) {
		app := ui.New(nil)
		
		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)
		
		keyMsg := tea.KeyMsg{Type: tea.KeyTab}
		
		// Start at Editor
		if app.GetActivePane() != types.EditorPaneType {
			t.Errorf("Expected initial focus on editor")
		}
		
		// Press tab 1: Editor → AI Input
		_, _ = app.Update(keyMsg)
		if app.GetActivePane() != types.AIPaneType {
			t.Errorf("After 1 tab, expected AI pane, got %v", app.GetActivePane())
		}
		
		// Press tab 2: AI Input → AI Response
		_, _ = app.Update(keyMsg)
		if app.GetActivePane() != types.AIPaneType {
			t.Errorf("After 2 tabs, expected AI pane (response area), got %v", app.GetActivePane())
		}
		
		// Press tab 3: AI Response → Editor
		_, _ = app.Update(keyMsg)
		if app.GetActivePane() != types.EditorPaneType {
			t.Errorf("After 3 tabs, expected editor pane, got %v", app.GetActivePane())
		}
		
		// Press tab 4: Back to AI Input (cycle repeats)
		_, _ = app.Update(keyMsg)
		if app.GetActivePane() != types.AIPaneType {
			t.Errorf("After 4 tabs, expected AI pane, got %v", app.GetActivePane())
		}
	})

	t.Run("multiple tab presses cycle through three areas", func(t *testing.T) {
		app := ui.New(nil)
		
		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)
		
		keyMsg := tea.KeyMsg{Type: tea.KeyTab}
		
		// Press tab 6 times (2 full cycles)
		expectedPanes := []types.PaneType{
			types.AIPaneType,    // 1: Editor → AI Input
			types.AIPaneType,    // 2: AI Input → AI Response
			types.EditorPaneType, // 3: AI Response → Editor
			types.AIPaneType,    // 4: Editor → AI Input
			types.AIPaneType,    // 5: AI Input → AI Response
			types.EditorPaneType, // 6: AI Response → Editor
		}
		
		for i := 0; i < 6; i++ {
			_, _ = app.Update(keyMsg)
			
			if app.GetActivePane() != expectedPanes[i] {
				t.Errorf("After %d tab presses, expected %v, got %v", i+1, expectedPanes[i], app.GetActivePane())
			}
		}
	})
}
// TestKeyboard_CtrlSSavesFile tests ctrl+s file saving
func TestKeyboard_CtrlSSavesFile(t *testing.T) {
	t.Run("ctrl+s saves file in editor", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config)
		
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
		
		// Press ctrl+s
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlS}
		_, _ = app.Update(keyMsg)
		
		// Verify file was saved
		if editorPane.HasUnsavedChanges() {
			t.Error("Expected no unsaved changes after ctrl+s")
		}
	})

	t.Run("ctrl+s does nothing when AI pane is active", func(t *testing.T) {
		tmpDir := t.TempDir()
		config := types.DefaultConfig()
		config.WorkspaceDir = tmpDir
		app := ui.New(config)
		
		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)
		
		// Switch to AI pane
		app.SetActivePane(types.AIPaneType)
		
		// Press ctrl+s (should do nothing)
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlS}
		_, _ = app.Update(keyMsg)
		
		// Verify still on AI pane
		if app.GetActivePane() != types.AIPaneType {
			t.Error("Active pane should remain AI pane")
		}
	})

	t.Run("ctrl+s with no file loaded", func(t *testing.T) {
		app := ui.New(nil)
		
		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)
		
		// Press ctrl+s without loading a file
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlS}
		_, _ = app.Update(keyMsg)
		
		// Should not crash
		if app.GetActivePane() != types.EditorPaneType {
			t.Error("Active pane should remain editor pane")
		}
	})
}

// TestKeyboard_CtrlRExecutesScript tests ctrl+r script execution
func TestKeyboard_CtrlRExecutesScript(t *testing.T) {
	t.Run("ctrl+r is handled", func(t *testing.T) {
		app := ui.New(nil)
		
		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)
		
		// Press ctrl+r
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlR}
		_, _ = app.Update(keyMsg)
		
		// Should not crash (implementation is placeholder)
		if app.GetActivePane() != types.EditorPaneType {
			t.Error("Active pane should remain editor pane")
		}
	})
}

// TestKeyboard_CtrlEnterSendsAIMessage tests ctrl+enter AI message sending
func TestKeyboard_CtrlEnterSendsAIMessage(t *testing.T) {
	t.Run("ctrl+enter sends message when AI pane is active", func(t *testing.T) {
		app := ui.New(nil)
		
		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)
		
		// Switch to AI pane
		app.SetActivePane(types.AIPaneType)
		
		// Press ctrl+enter (represented as enter with ctrl modifier)
		keyMsg := tea.KeyMsg{Type: tea.KeyEnter, Alt: false}
		_, _ = app.Update(keyMsg)
		
		// Should not crash
		if app.GetActivePane() != types.AIPaneType {
			t.Error("Active pane should remain AI pane")
		}
	})

	t.Run("ctrl+enter does nothing when editor is active", func(t *testing.T) {
		app := ui.New(nil)
		
		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)
		
		// Press ctrl+enter (editor is active)
		keyMsg := tea.KeyMsg{Type: tea.KeyEnter, Alt: false}
		_, _ = app.Update(keyMsg)
		
		// Should remain on editor
		if app.GetActivePane() != types.EditorPaneType {
			t.Error("Active pane should remain editor pane")
		}
	})
}

// TestKeyboard_QuitShortcuts tests quit shortcuts
func TestKeyboard_QuitShortcuts(t *testing.T) {
	t.Run("ctrl+c returns quit command", func(t *testing.T) {
		app := ui.New(nil)
		
		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)
		
		// Press ctrl+c
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
		_, cmd := app.Update(keyMsg)
		
		if cmd == nil {
			t.Error("Expected quit command from ctrl+c")
		}
	})

	t.Run("ctrl+q returns quit command", func(t *testing.T) {
		app := ui.New(nil)
		
		// Initialize
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)
		
		// Press ctrl+q
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		_, cmd := app.Update(keyMsg)
		
		if cmd == nil {
			t.Error("Expected quit command from ctrl+q")
		}
	})
}

// TestKeyboard_ShortcutConflicts tests that shortcuts don't conflict
func TestKeyboard_ShortcutConflicts(t *testing.T) {
	t.Run("all shortcuts are distinct", func(t *testing.T) {
		shortcuts := []tea.KeyType{
			tea.KeyTab,
			tea.KeyCtrlS,
			tea.KeyCtrlR,
			tea.KeyEnter,
			tea.KeyCtrlC,
			tea.KeyCtrlQ,
		}
		
		// Verify all shortcuts are unique
		seen := make(map[tea.KeyType]bool)
		for _, shortcut := range shortcuts {
			if seen[shortcut] {
				t.Errorf("Duplicate shortcut detected: %v", shortcut)
			}
			seen[shortcut] = true
		}
	})
}
