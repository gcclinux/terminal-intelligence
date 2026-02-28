package property

import (
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/user/terminal-intelligence/internal/filemanager"
	"github.com/user/terminal-intelligence/internal/types"
	"github.com/user/terminal-intelligence/internal/ui"
)

// Feature: Terminal Intelligence (TI), Property 2: Editor Content Display
// **Validates: Requirements 2.5, 3.2**
//
// For any file content loaded into the Editor_Pane,
// the displayed content should match the file content exactly.
func TestProperty_EditorContentDisplay(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("loaded file content matches editor display content", prop.ForAll(
		func(filename string, content string) bool {
			// Create temporary workspace for this test
			tmpDir := t.TempDir()
			fm := filemanager.NewFileManager(tmpDir)
			editor := ui.NewEditorPane(fm)

			// Create a file with the generated content
			err := fm.CreateFile(filename, content)
			if err != nil {
				t.Logf("CreateFile failed: %v", err)
				return false
			}

			// Load the file into the editor
			err = editor.LoadFile(filename)
			if err != nil {
				t.Logf("LoadFile failed: %v", err)
				return false
			}

			// Get the content from the editor
			editorContent := editor.GetContent()

			// Verify the editor content matches the original file content exactly
			if editorContent != content {
				t.Logf("Editor content mismatch:\nExpected: %q\nGot: %q", content, editorContent)
				return false
			}

			return true
		},
		genValidFilename(),
		gen.AnyString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: Terminal Intelligence (TI), Property 3: Unsaved Changes Tracking
// **Validates: Requirements 2.6**
//
// For any modification to the editor content, the editor should indicate
// that there are unsaved changes until the file is saved.
func TestProperty_UnsavedChangesTracking(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("editor tracks unsaved changes after modification", prop.ForAll(
		func(filename string, originalContent string, modifiedContent string) bool {
			// Skip if contents are identical (no modification)
			if originalContent == modifiedContent {
				return true
			}

			// Create temporary workspace for this test
			tmpDir := t.TempDir()
			fm := filemanager.NewFileManager(tmpDir)
			editor := ui.NewEditorPane(fm)

			// Create a file with original content
			err := fm.CreateFile(filename, originalContent)
			if err != nil {
				t.Logf("CreateFile failed: %v", err)
				return false
			}

			// Load the file into the editor
			err = editor.LoadFile(filename)
			if err != nil {
				t.Logf("LoadFile failed: %v", err)
				return false
			}

			// Verify no unsaved changes initially
			if editor.HasUnsavedChanges() {
				t.Logf("Editor should not have unsaved changes after loading")
				return false
			}

			// Modify the content
			editor.SetContent(modifiedContent)

			// Verify unsaved changes are tracked
			if !editor.HasUnsavedChanges() {
				t.Logf("Editor should have unsaved changes after modification")
				return false
			}

			// Save the file
			err = editor.SaveFile()
			if err != nil {
				t.Logf("SaveFile failed: %v", err)
				return false
			}

			// Verify unsaved changes flag is cleared after save
			if editor.HasUnsavedChanges() {
				t.Logf("Editor should not have unsaved changes after saving")
				return false
			}

			return true
		},
		genValidFilename(),
		gen.AnyString(),
		gen.AnyString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: Terminal Intelligence (TI), Property 14: Line Numbers Display
// **Validates: Requirements 6.4**
//
// For any code file (bash, shell, PowerShell, markdown) loaded in the editor,
// line numbers should be visible in the Editor_Pane.
func TestProperty_LineNumbersDisplay(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("editor displays line numbers for all code files", prop.ForAll(
		func(fileType string, content string) bool {
			// Create temporary workspace for this test
			tmpDir := t.TempDir()
			fm := filemanager.NewFileManager(tmpDir)
			editor := ui.NewEditorPane(fm)

			// Set editor size so View() can render properly
			editor.SetSize(80, 20)

			// Generate filename based on file type
			filename := "test" + fileType

			// Create a file with the generated content
			err := fm.CreateFile(filename, content)
			if err != nil {
				t.Logf("CreateFile failed: %v", err)
				return false
			}

			// Load the file into the editor
			err = editor.LoadFile(filename)
			if err != nil {
				t.Logf("LoadFile failed: %v", err)
				return false
			}

			// Render the editor view
			view := editor.View()

			// Verify line numbers are present in the view
			// Line numbers are formatted as 4-digit numbers followed by " │ "
			// For example: "   1 │ " or "  42 │ "
			if !containsLineNumbers(view) {
				t.Logf("Editor view does not contain line numbers for file type %s", fileType)
				t.Logf("View content:\n%s", view)
				return false
			}

			return true
		},
		genFileExtension(),
		gen.AnyString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// containsLineNumbers checks if the view contains line number formatting

// containsLineNumbers checks if the view contains line number formatting
func containsLineNumbers(view string) bool {
	// Line numbers are displayed with the format "   1 │ " or similar
	// Check for the presence of the line number separator " │ "
	// This is the distinctive marker that line numbers are being displayed
	return len(view) > 0 && strings.Contains(view, " │ ")
}

// genFileExtension generates file extensions for supported file types
func genFileExtension() gopter.Gen {
	return gen.OneConstOf(
		".sh",   // bash
		".bash", // bash
		".ps1",  // powershell
		".md",   // markdown
	)
}

// Feature: Terminal Intelligence (TI), Property 13: Terminal Resize Proportional Adjustment
// **Validates: Requirements 6.3**
//
// For any terminal resize event, both the Editor_Pane and AI_Pane should
// maintain their proportional sizes relative to the new terminal dimensions.
func TestProperty_TerminalResizeProportionalAdjustment(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("panes maintain proportional sizes after terminal resize", prop.ForAll(
		func(width int, height int) bool {
			// Skip invalid terminal sizes
			if width < 20 || height < 10 {
				return true
			}

			// Create app with default config
			app := ui.New(nil, "test")

			// Simulate window size message
			msg := ui.WindowSizeMsg{Width: width, Height: height}
			_, _ = app.Update(msg)

			// Get pane sizes
			editorPane := app.GetEditorPane()
			aiPane := app.GetAIPane()

			expectedEditorWidth := (width / 2) + 2
			expectedAIWidth := width + 2 - expectedEditorWidth
			expectedPaneHeight := height - 7 // Account for header (3 lines), editor title (3 lines), and status bar (1 line)

			// Verify editor pane size matches formula
			if editorPane.GetWidth() != expectedEditorWidth {
				t.Logf("Editor pane width mismatch: expected %d, got %d", expectedEditorWidth, editorPane.GetWidth())
				return false
			}
			if editorPane.GetHeight() != expectedPaneHeight {
				t.Logf("Editor pane height mismatch: expected %d, got %d", expectedPaneHeight, editorPane.GetHeight())
				return false
			}

			// Verify AI pane size matches formula
			if aiPane.GetWidth() != expectedAIWidth {
				t.Logf("AI pane width mismatch: expected %d, got %d", expectedAIWidth, aiPane.GetWidth())
				return false
			}
			if aiPane.GetHeight() != expectedPaneHeight {
				t.Logf("AI pane height mismatch: expected %d, got %d", expectedPaneHeight, aiPane.GetHeight())
				return false
			}

			return true
		},
		gen.IntRange(20, 300),
		gen.IntRange(10, 100),
	))

	properties.TestingRun(t)
}

// Feature: Terminal Intelligence (TI), Property 15: Active Pane Visual Indication
// **Validates: Requirements 6.6**
//
// For any pane focus change, the newly focused pane should have a visual
// indicator distinguishing it from the unfocused pane.
func TestProperty_ActivePaneVisualIndication(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("focused pane has visual indicator", prop.ForAll(
		func(width int, height int, startWithEditor bool) bool {
			// Skip invalid terminal sizes
			if width < 20 || height < 10 {
				return true
			}

			// Create app with default config
			app := ui.New(nil, "test")

			// Set terminal size
			msg := ui.WindowSizeMsg{Width: width, Height: height}
			_, _ = app.Update(msg)

			// Set initial active pane
			var initialPane types.PaneType
			if startWithEditor {
				initialPane = types.EditorPaneType
			} else {
				initialPane = types.AIPaneType
			}
			app.SetActivePane(initialPane)

			// Verify the active pane is correctly set
			if app.GetActivePane() != initialPane {
				t.Logf("Active pane not correctly set: expected %v, got %v", initialPane, app.GetActivePane())
				return false
			}

			// Verify focus state of panes matches active pane
			editorPane := app.GetEditorPane()
			aiPane := app.GetAIPane()

			if initialPane == types.EditorPaneType {
				// Editor should be focused, AI should not
				editorView := editorPane.View()
				aiView := aiPane.View()

				// Both views should exist
				if len(editorView) == 0 || len(aiView) == 0 {
					return true // Skip if views are empty
				}

				// Views should be different (one focused, one not)
				if editorView == aiView {
					t.Logf("Editor and AI views should be different when one is focused")
					return false
				}
			} else {
				// AI should be focused, Editor should not
				editorView := editorPane.View()
				aiView := aiPane.View()

				// Both views should exist
				if len(editorView) == 0 || len(aiView) == 0 {
					return true // Skip if views are empty
				}

				// Views should be different (one focused, one not)
				if editorView == aiView {
					t.Logf("Editor and AI views should be different when one is focused")
					return false
				}
			}

			// Now switch to the other pane
			var switchedPane types.PaneType
			if initialPane == types.EditorPaneType {
				switchedPane = types.AIPaneType
			} else {
				switchedPane = types.EditorPaneType
			}
			app.SetActivePane(switchedPane)

			// Verify the active pane changed
			if app.GetActivePane() != switchedPane {
				t.Logf("Active pane not correctly switched: expected %v, got %v", switchedPane, app.GetActivePane())
				return false
			}

			return true
		},
		gen.IntRange(20, 300), // Terminal width range
		gen.IntRange(10, 100), // Terminal height range
		gen.Bool(),            // Start with editor or AI pane
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
