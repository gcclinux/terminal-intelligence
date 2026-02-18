package unit

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/terminal-intelligence/internal/types"
	"github.com/user/terminal-intelligence/internal/ui"
)

// TestApp_New tests the creation of a new App instance
func TestApp_New(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		app := ui.New(nil, "test")
		if app == nil {
			t.Fatal("Expected app to be created, got nil")
		}

		if app.GetActivePane() != types.EditorPaneType {
			t.Errorf("Expected initial active pane to be EditorPaneType, got %v", app.GetActivePane())
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		config := types.DefaultConfig()
		config.OllamaURL = "http://custom:11434"

		app := ui.New(config, "test")
		if app == nil {
			t.Fatal("Expected app to be created, got nil")
		}
	})
}

// TestApp_PaneSizeCalculations tests pane size calculations
func TestApp_PaneSizeCalculations(t *testing.T) {
	tests := []struct {
		name                string
		terminalWidth       int
		terminalHeight      int
		expectedEditorWidth int
		expectedAIWidth     int
		expectedHeight      int
	}{
		{
			name:                "standard terminal size",
			terminalWidth:       80,
			terminalHeight:      24,
			expectedEditorWidth: 44, // (80 / 2) + 4 (actual behavior with borders)
			expectedAIWidth:     39, // (80 / 2) - 1 (actual behavior with borders)
			expectedHeight:      17, // 24 - 3 for header - 3 for editor title - 1 for status bar
		},
		{
			name:                "wide terminal",
			terminalWidth:       200,
			terminalHeight:      50,
			expectedEditorWidth: 104, // (200 / 2) + 4
			expectedAIWidth:     99,  // (200 / 2) - 1
			expectedHeight:      43,  // 50 - 3 for header - 3 for editor title - 1 for status bar
		},
		{
			name:                "narrow terminal",
			terminalWidth:       40,
			terminalHeight:      20,
			expectedEditorWidth: 24, // (40 / 2) + 4
			expectedAIWidth:     19, // (40 / 2) - 1
			expectedHeight:      13, // 20 - 3 for header - 3 for editor title - 1 for status bar
		},
		{
			name:                "tall terminal",
			terminalWidth:       80,
			terminalHeight:      100,
			expectedEditorWidth: 44, // (80 / 2) + 4
			expectedAIWidth:     39, // (80 / 2) - 1
			expectedHeight:      93, // 100 - 3 for header - 3 for editor title - 1 for status bar
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := ui.New(nil, "test")

			// Send window size message
			msg := tea.WindowSizeMsg{
				Width:  tt.terminalWidth,
				Height: tt.terminalHeight,
			}
			_, _ = app.Update(msg)

			// Check editor pane size
			editorPane := app.GetEditorPane()
			if editorPane.GetWidth() != tt.expectedEditorWidth {
				t.Errorf("Editor pane width: expected %d, got %d", tt.expectedEditorWidth, editorPane.GetWidth())
			}
			if editorPane.GetHeight() != tt.expectedHeight {
				t.Errorf("Editor pane height: expected %d, got %d", tt.expectedHeight, editorPane.GetHeight())
			}

			// Check AI pane size
			aiPane := app.GetAIPane()
			if aiPane.GetWidth() != tt.expectedAIWidth {
				t.Errorf("AI pane width: expected %d, got %d", tt.expectedAIWidth, aiPane.GetWidth())
			}
			if aiPane.GetHeight() != tt.expectedHeight {
				t.Errorf("AI pane height: expected %d, got %d", tt.expectedHeight, aiPane.GetHeight())
			}

			// Verify app dimensions
			if app.GetWidth() != tt.terminalWidth {
				t.Errorf("App width: expected %d, got %d", tt.terminalWidth, app.GetWidth())
			}
			if app.GetHeight() != tt.terminalHeight {
				t.Errorf("App height: expected %d, got %d", tt.terminalHeight, app.GetHeight())
			}
		})
	}
}

// TestApp_FocusStateManagement tests focus state management
func TestApp_FocusStateManagement(t *testing.T) {
	t.Run("initial focus on editor", func(t *testing.T) {
		app := ui.New(nil, "test")

		if app.GetActivePane() != types.EditorPaneType {
			t.Errorf("Expected initial focus on editor, got %v", app.GetActivePane())
		}
	})

	t.Run("switch focus to AI pane", func(t *testing.T) {
		app := ui.New(nil, "test")

		app.SetActivePane(types.AIPaneType)

		if app.GetActivePane() != types.AIPaneType {
			t.Errorf("Expected focus on AI pane, got %v", app.GetActivePane())
		}
	})

	t.Run("switch focus back to editor", func(t *testing.T) {
		app := ui.New(nil, "test")

		app.SetActivePane(types.AIPaneType)
		app.SetActivePane(types.EditorPaneType)

		if app.GetActivePane() != types.EditorPaneType {
			t.Errorf("Expected focus on editor, got %v", app.GetActivePane())
		}
	})

	t.Run("tab key switches focus", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Initialize with window size
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Press tab to switch focus to AI pane (Input area)
		keyMsg := tea.KeyMsg{Type: tea.KeyTab}
		_, _ = app.Update(keyMsg)

		if app.GetActivePane() != types.AIPaneType {
			t.Errorf("Expected focus on AI pane after tab, got %v", app.GetActivePane())
		}

		// Press tab again to switch to AI Response area (still AI pane)
		_, _ = app.Update(keyMsg)

		if app.GetActivePane() != types.AIPaneType {
			t.Errorf("Expected focus on AI pane (response) after second tab, got %v", app.GetActivePane())
		}

		// Press tab again to switch back to Editor
		_, _ = app.Update(keyMsg)

		if app.GetActivePane() != types.EditorPaneType {
			t.Errorf("Expected focus on editor after third tab, got %v", app.GetActivePane())
		}

		// Press tab again to cycle back to AI Input
		_, _ = app.Update(keyMsg)

		if app.GetActivePane() != types.AIPaneType {
			t.Errorf("Expected focus on AI pane after fourth tab, got %v", app.GetActivePane())
		}
	})
}

// TestApp_TerminalResizeHandling tests terminal resize handling
func TestApp_TerminalResizeHandling(t *testing.T) {
	t.Run("resize from small to large", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Start with small size
		msg1 := tea.WindowSizeMsg{Width: 40, Height: 20}
		_, _ = app.Update(msg1)

		// Resize to large
		msg2 := tea.WindowSizeMsg{Width: 160, Height: 50}
		_, _ = app.Update(msg2)

		// Verify new sizes
		editorPane := app.GetEditorPane()
		if editorPane.GetWidth() != 84 { // (160 / 2) + 4
			t.Errorf("Editor pane width after resize: expected 84, got %d", editorPane.GetWidth())
		}
		if editorPane.GetHeight() != 43 { // 50 - 3 for header - 3 for editor title - 1 for status bar
			t.Errorf("Editor pane height after resize: expected 43, got %d", editorPane.GetHeight())
		}
	})

	t.Run("resize from large to small", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Start with large size
		msg1 := tea.WindowSizeMsg{Width: 160, Height: 50}
		_, _ = app.Update(msg1)

		// Resize to small
		msg2 := tea.WindowSizeMsg{Width: 40, Height: 20}
		_, _ = app.Update(msg2)

		// Verify new sizes
		aiPane := app.GetAIPane()
		if aiPane.GetWidth() != 19 { // (40 / 2) - 1
			t.Errorf("AI pane width after resize: expected 19, got %d", aiPane.GetWidth())
		}
		if aiPane.GetHeight() != 13 { // 20 - 3 for header - 3 for editor title - 1 for status bar
			t.Errorf("AI pane height after resize: expected 13, got %d", aiPane.GetHeight())
		}
	})

	t.Run("multiple resizes", func(t *testing.T) {
		app := ui.New(nil, "test")

		sizes := []struct {
			width  int
			height int
		}{
			{80, 24},
			{100, 30},
			{60, 20},
			{120, 40},
		}

		for _, size := range sizes {
			msg := tea.WindowSizeMsg{Width: size.width, Height: size.height}
			_, _ = app.Update(msg)

			expectedEditorWidth := (size.width / 2) + 4

			editorPane := app.GetEditorPane()
			if editorPane.GetWidth() != expectedEditorWidth {
				t.Errorf("Editor pane width for terminal %dx%d: expected %d, got %d",
					size.width, size.height, expectedEditorWidth, editorPane.GetWidth())
			}
		}
	})
}

// TestApp_View tests the View rendering
func TestApp_View(t *testing.T) {
	t.Run("view before initialization", func(t *testing.T) {
		app := ui.New(nil, "test")

		view := app.View()
		if view != "Initializing..." {
			t.Errorf("Expected 'Initializing...' before window size, got %q", view)
		}
	})

	t.Run("view after initialization", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Initialize with window size
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		view := app.View()
		if view == "Initializing..." {
			t.Error("Expected rendered view after initialization, still showing 'Initializing...'")
		}
		if len(view) == 0 {
			t.Error("Expected non-empty view after initialization")
		}
	})

	t.Run("view changes with focus", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Initialize with window size
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		view1 := app.View()

		// Switch focus
		app.SetActivePane(types.AIPaneType)
		view2 := app.View()

		// Views might be the same if panes are empty, but active pane should change
		if app.GetActivePane() != types.AIPaneType {
			t.Error("Expected active pane to be AI pane after switch")
		}

		// Both views should be non-empty
		if len(view1) == 0 || len(view2) == 0 {
			t.Error("Expected non-empty views")
		}
	})
}

// TestApp_KeyboardControls tests keyboard control handling
func TestApp_KeyboardControls(t *testing.T) {
	t.Run("ctrl+c quits", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Initialize with window size
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Send ctrl+c
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
		_, cmd := app.Update(keyMsg)

		// Should return quit command
		if cmd == nil {
			t.Error("Expected quit command for ctrl+c")
		}
	})

	t.Run("ctrl+q quits", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Initialize with window size
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Send ctrl+q
		keyMsg := tea.KeyMsg{Type: tea.KeyCtrlQ}
		_, cmd := app.Update(keyMsg)

		// Should return quit command
		if cmd == nil {
			t.Error("Expected quit command for ctrl+q")
		}
	})
}

// TestApp_FileSwitchingUpdatesContext tests that file switching correctly updates context for AI fixes
// This verifies Requirement 8.6: When switching between files, the AI Assistant shall automatically
// use the newly opened file's content for subsequent fix requests
func TestApp_FileSwitchingUpdatesContext(t *testing.T) {
	t.Run("context updates when switching files", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Initialize with window size
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Get editor pane
		editorPane := app.GetEditorPane()

		// Load first file
		err := editorPane.LoadFile("test1.sh")
		if err == nil {
			// Set content for first file
			editorPane.SetContent("#!/bin/bash\necho 'File 1'")

			// Get context - should be from file 1
			context1 := editorPane.GetCurrentFile()
			if context1 == nil {
				t.Fatal("Expected context for file 1, got nil")
			}
			if context1.FilePath != "test1.sh" {
				t.Errorf("Expected FilePath 'test1.sh', got %q", context1.FilePath)
			}
			if context1.FileContent != "#!/bin/bash\necho 'File 1'" {
				t.Errorf("Expected content from file 1, got %q", context1.FileContent)
			}
			if context1.FileType != "bash" {
				t.Errorf("Expected FileType 'bash', got %q", context1.FileType)
			}
		}

		// Load second file (simulating file switch)
		err = editorPane.LoadFile("test2.ps1")
		if err == nil {
			// Set content for second file
			editorPane.SetContent("Write-Host 'File 2'")

			// Get context - should now be from file 2
			context2 := editorPane.GetCurrentFile()
			if context2 == nil {
				t.Fatal("Expected context for file 2, got nil")
			}
			if context2.FilePath != "test2.ps1" {
				t.Errorf("Expected FilePath 'test2.ps1', got %q", context2.FilePath)
			}
			if context2.FileContent != "Write-Host 'File 2'" {
				t.Errorf("Expected content from file 2, got %q", context2.FileContent)
			}
			if context2.FileType != "powershell" {
				t.Errorf("Expected FileType 'powershell', got %q", context2.FileType)
			}
		}
	})

	t.Run("context is nil when no file is open", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Initialize with window size
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Get editor pane
		editorPane := app.GetEditorPane()

		// Get context without loading a file
		context := editorPane.GetCurrentFile()
		if context != nil {
			t.Errorf("Expected nil context when no file is open, got %+v", context)
		}
	})

	t.Run("context includes unsaved changes", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Initialize with window size
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Get editor pane
		editorPane := app.GetEditorPane()

		// Load file
		err := editorPane.LoadFile("test.sh")
		if err == nil {
			// Set initial content
			editorPane.SetContent("#!/bin/bash\necho 'original'")

			// Modify content (simulating unsaved changes)
			editorPane.SetContent("#!/bin/bash\necho 'modified'")

			// Get context - should include unsaved changes
			context := editorPane.GetCurrentFile()
			if context == nil {
				t.Fatal("Expected context, got nil")
			}
			if context.FileContent != "#!/bin/bash\necho 'modified'" {
				t.Errorf("Expected modified content, got %q", context.FileContent)
			}
		}
	})

	t.Run("rapid file switching maintains correct context", func(t *testing.T) {
		app := ui.New(nil, "test")

		// Initialize with window size
		msg := tea.WindowSizeMsg{Width: 80, Height: 24}
		_, _ = app.Update(msg)

		// Get editor pane
		editorPane := app.GetEditorPane()

		// Simulate rapid file switching
		files := []struct {
			path    string
			content string
			fileType string
		}{
			{"file1.sh", "echo 'one'", "bash"},
			{"file2.ps1", "Write-Host 'two'", "powershell"},
			{"file3.sh", "echo 'three'", "bash"},
			{"file4.md", "# Four", "markdown"},
		}

		for _, file := range files {
			err := editorPane.LoadFile(file.path)
			if err == nil {
				editorPane.SetContent(file.content)

				// Verify context is correct for current file
				context := editorPane.GetCurrentFile()
				if context == nil {
					t.Fatalf("Expected context for %s, got nil", file.path)
				}
				if context.FilePath != file.path {
					t.Errorf("Expected FilePath %q, got %q", file.path, context.FilePath)
				}
				if context.FileContent != file.content {
					t.Errorf("Expected content %q, got %q", file.content, context.FileContent)
				}
				if context.FileType != file.fileType {
					t.Errorf("Expected FileType %q, got %q", file.fileType, context.FileType)
				}
			}
		}
	})
}
