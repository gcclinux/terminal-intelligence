package property

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/user/terminal-intelligence/internal/types"
	"github.com/user/terminal-intelligence/internal/ui"
)

// Feature: Terminal Intelligence (TI), Property 16: Keyboard Shortcut Action Execution
// **Validates: Requirements 7.6**
//
// For any registered keyboard shortcut, pressing that shortcut should
// immediately trigger its associated action.
func TestProperty_KeyboardShortcutActionExecution(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("tab switches active pane immediately", prop.ForAll(
		func(width int, height int) bool {
			// Skip invalid terminal sizes
			if width < 20 || height < 10 {
				return true
			}

			// Create app
			app := ui.New(nil)

			// Initialize with window size
			msg := ui.WindowSizeMsg{Width: width, Height: height}
			_, _ = app.Update(msg)

			// Get initial active pane (should be Editor)
			initialPane := app.GetActivePane()

			// Press tab - should switch to AI Input
			keyMsg := tea.KeyMsg{Type: tea.KeyTab}
			_, _ = app.Update(keyMsg)

			// Verify active pane changed to AI
			newPane := app.GetActivePane()
			if newPane == initialPane {
				t.Logf("Tab should switch active pane")
				return false
			}

			// Press tab again - should switch to AI Response (still AI pane)
			_, _ = app.Update(keyMsg)

			// Verify still in AI pane
			secondPane := app.GetActivePane()
			if secondPane != newPane {
				t.Logf("Second tab should stay in AI pane (response area)")
				return false
			}

			// Press tab again - should switch back to Editor
			_, _ = app.Update(keyMsg)

			// Verify active pane switched back to initial
			finalPane := app.GetActivePane()
			if finalPane != initialPane {
				t.Logf("Third tab should switch back to initial pane")
				return false
			}

			return true
		},
		gen.IntRange(20, 300),  // Terminal width range
		gen.IntRange(10, 100),  // Terminal height range
	))

	properties.Property("ctrl+s saves file immediately", prop.ForAll(
		func(width int, height int, filename string, content string) bool {
			// Skip invalid terminal sizes
			if width < 20 || height < 10 {
				return true
			}

			// Create app
			tmpDir := t.TempDir()
			config := types.DefaultConfig()
			config.WorkspaceDir = tmpDir
			app := ui.New(config)

			// Initialize with window size
			msg := ui.WindowSizeMsg{Width: width, Height: height}
			_, _ = app.Update(msg)

			// Set editor as active pane
			app.SetActivePane(types.EditorPaneType)

			// Load a file in editor
			editorPane := app.GetEditorPane()
			err := editorPane.GetFileManager().CreateFile(filename, content)
			if err != nil {
				return true // Skip if file creation fails
			}

			err = editorPane.LoadFile(filename)
			if err != nil {
				return true // Skip if file load fails
			}

			// Modify content
			modifiedContent := content + "\n// Modified"
			editorPane.SetContent(modifiedContent)

			// Verify unsaved changes
			if !editorPane.HasUnsavedChanges() {
				t.Logf("Editor should have unsaved changes after modification")
				return false
			}

			// Press ctrl+s
			keyMsg := tea.KeyMsg{Type: tea.KeyCtrlS}
			_, _ = app.Update(keyMsg)

			// Verify file was saved (no unsaved changes)
			if editorPane.HasUnsavedChanges() {
				t.Logf("Ctrl+S should save file and clear unsaved changes flag")
				return false
			}

			return true
		},
		gen.IntRange(20, 300),  // Terminal width range
		gen.IntRange(10, 100),  // Terminal height range
		genValidFilename(),
		gen.AnyString(),
	))

	properties.Property("ctrl+c quits immediately", prop.ForAll(
		func(width int, height int) bool {
			// Skip invalid terminal sizes
			if width < 20 || height < 10 {
				return true
			}

			// Create app
			app := ui.New(nil)

			// Initialize with window size
			msg := ui.WindowSizeMsg{Width: width, Height: height}
			_, _ = app.Update(msg)

			// Press ctrl+c
			keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
			_, cmd := app.Update(keyMsg)

			// Verify quit command was returned
			if cmd == nil {
				t.Logf("Ctrl+C should return quit command")
				return false
			}

			return true
		},
		gen.IntRange(20, 300),  // Terminal width range
		gen.IntRange(10, 100),  // Terminal height range
	))

	properties.Property("ctrl+q quits immediately", prop.ForAll(
		func(width int, height int) bool {
			// Skip invalid terminal sizes
			if width < 20 || height < 10 {
				return true
			}

			// Create app
			app := ui.New(nil)

			// Initialize with window size
			msg := ui.WindowSizeMsg{Width: width, Height: height}
			_, _ = app.Update(msg)

			// Press ctrl+q
			keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
			_, cmd := app.Update(keyMsg)

			// Verify quit command was returned
			if cmd == nil {
				t.Logf("Ctrl+Q should return quit command")
				return false
			}

			return true
		},
		gen.IntRange(20, 300),  // Terminal width range
		gen.IntRange(10, 100),  // Terminal height range
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
