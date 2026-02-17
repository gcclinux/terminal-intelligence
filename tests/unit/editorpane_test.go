package unit

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/terminal-intelligence/internal/filemanager"
	"github.com/user/terminal-intelligence/internal/ui"
)

func TestEditorPane_NewEditorPane(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)

	editor := ui.NewEditorPane(fm)

	if editor == nil {
		t.Fatal("NewEditorPane() returned nil")
	}

	if editor.GetContent() != "" {
		t.Errorf("NewEditorPane() content = %q, want empty string", editor.GetContent())
	}

	if editor.HasUnsavedChanges() {
		t.Error("NewEditorPane() should not have unsaved changes")
	}
}

func TestEditorPane_SetContent(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)

	tests := []struct {
		name    string
		content string
	}{
		{"empty content", ""},
		{"simple text", "hello world"},
		{"multiline text", "line1\nline2\nline3"},
		{"text with special chars", "hello\tworld\n\ntest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor.SetContent(tt.content)

			got := editor.GetContent()
			if got != tt.content {
				t.Errorf("SetContent() then GetContent() = %q, want %q", got, tt.content)
			}
		})
	}
}

func TestEditorPane_LoadFile(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)

	// Create test files
	testContent := "#!/bin/bash\necho 'test'"
	err := fm.CreateFile("test.sh", testContent)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test loading file
	err = editor.LoadFile("test.sh")
	if err != nil {
		t.Errorf("LoadFile() error = %v, want nil", err)
	}

	got := editor.GetContent()
	if got != testContent {
		t.Errorf("LoadFile() content = %q, want %q", got, testContent)
	}

	if editor.HasUnsavedChanges() {
		t.Error("LoadFile() should not have unsaved changes")
	}
}

func TestEditorPane_LoadFile_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)

	err := editor.LoadFile("nonexistent.txt")
	if err == nil {
		t.Error("LoadFile() with non-existent file should return error")
	}
}

func TestEditorPane_SaveFile(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)

	// Create and load a file
	initialContent := "initial content"
	err := fm.CreateFile("test.txt", initialContent)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = editor.LoadFile("test.txt")
	if err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}

	// Modify content
	newContent := "modified content"
	editor.SetContent(newContent)

	if !editor.HasUnsavedChanges() {
		t.Error("SetContent() should mark file as modified")
	}

	// Save file
	err = editor.SaveFile()
	if err != nil {
		t.Errorf("SaveFile() error = %v, want nil", err)
	}

	if editor.HasUnsavedChanges() {
		t.Error("SaveFile() should clear unsaved changes flag")
	}

	// Verify file was saved
	savedContent, err := fm.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if savedContent != newContent {
		t.Errorf("SaveFile() saved content = %q, want %q", savedContent, newContent)
	}
}

func TestEditorPane_HasUnsavedChanges(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)

	// Create and load a file
	err := fm.CreateFile("test.txt", "original")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = editor.LoadFile("test.txt")
	if err != nil {
		t.Fatalf("Failed to load file: %v", err)
	}

	// Initially no unsaved changes
	if editor.HasUnsavedChanges() {
		t.Error("Newly loaded file should not have unsaved changes")
	}

	// Modify content
	editor.SetContent("modified")
	if !editor.HasUnsavedChanges() {
		t.Error("Modified content should have unsaved changes")
	}

	// Save and check again
	editor.SaveFile()
	if editor.HasUnsavedChanges() {
		t.Error("Saved file should not have unsaved changes")
	}

	// Modify again
	editor.SetContent("modified again")
	if !editor.HasUnsavedChanges() {
		t.Error("Modified content should have unsaved changes")
	}
}

func TestEditorPane_Update_WindowSize(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)
	editor.SetFocused(true)

	// Send window size message
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	editor.Update(msg)

	// The editor should handle the resize (no error expected)
	// We can't directly test the internal width/height, but we can verify View() doesn't panic
	view := editor.View()
	if view == "" {
		t.Error("View() should return non-empty string after resize")
	}
}

func TestEditorPane_View(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)
	editor.SetSize(50, 20)

	// Test with empty content
	view := editor.View()
	if view == "" {
		t.Error("View() should return non-empty string even with empty content")
	}

	// Test with content
	editor.SetContent("line1\nline2\nline3")
	view = editor.View()
	if view == "" {
		t.Error("View() should return non-empty string with content")
	}
}

func TestEditorPane_CursorMovement(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)
	editor.SetFocused(true)
	editor.SetSize(50, 20)
	editor.SetContent("line1\nline2\nline3")

	// Test arrow key movements
	tests := []struct {
		name string
		key  string
	}{
		{"move down", "down"},
		{"move up", "up"},
		{"move right", "right"},
		{"move left", "left"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			editor.Update(msg)
			// Just verify it doesn't panic
		})
	}
}

func TestEditorPane_TextEditing(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)
	editor.SetFocused(true)
	editor.SetSize(50, 20)

	// Test inserting characters
	editor.SetContent("")

	// Simulate typing "hello"
	for _, ch := range "hello" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
		editor.Update(msg)
	}

	content := editor.GetContent()
	if content != "hello" {
		t.Errorf("After typing 'hello', content = %q, want 'hello'", content)
	}
}

func TestEditorPane_FileTypeDetection(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)

	tests := []struct {
		filename     string
		content      string
		expectedType string
	}{
		{"test.sh", "#!/bin/bash", "bash"},
		{"test.bash", "echo test", "bash"},
		{"test.ps1", "Write-Host test", "powershell"},
		{"test.md", "# Markdown", "markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			err := fm.CreateFile(tt.filename, tt.content)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			err = editor.LoadFile(tt.filename)
			if err != nil {
				t.Fatalf("Failed to load file: %v", err)
			}

			// We can't directly access currentFile.FileType, but we can verify the file loaded
			if editor.GetContent() != tt.content {
				t.Errorf("LoadFile() content = %q, want %q", editor.GetContent(), tt.content)
			}
		})
	}
}

func TestEditorPane_CursorMovement_Comprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)
	editor.SetFocused(true)
	editor.SetSize(50, 20)

	// Test with multi-line content
	content := "line1\nline2\nline3\nline4"
	editor.SetContent(content)

	tests := []struct {
		name        string
		keys        []string
		description string
	}{
		{
			name:        "move down multiple times",
			keys:        []string{"down", "down", "down"},
			description: "cursor should move down through lines",
		},
		{
			name:        "move up from bottom",
			keys:        []string{"down", "down", "down", "up", "up"},
			description: "cursor should move up through lines",
		},
		{
			name:        "move right through line",
			keys:        []string{"right", "right", "right"},
			description: "cursor should move right through characters",
		},
		{
			name:        "move left through line",
			keys:        []string{"right", "right", "right", "left", "left"},
			description: "cursor should move left through characters",
		},
		{
			name:        "move right at end of line wraps to next line",
			keys:        []string{"right", "right", "right", "right", "right", "right"},
			description: "cursor should wrap to next line at end",
		},
		{
			name:        "move left at start of line wraps to previous line",
			keys:        []string{"down", "left"},
			description: "cursor should wrap to previous line at start",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset editor state
			editor.SetContent(content)

			// Simulate key presses
			for _, key := range tt.keys {
				msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
				editor.Update(msg)
			}

			// Verify editor doesn't panic and content is unchanged
			if editor.GetContent() != content {
				t.Errorf("Cursor movement should not modify content")
			}
		})
	}
}

func TestEditorPane_Scrolling(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)
	editor.SetFocused(true)
	editor.SetSize(50, 10) // Small height to trigger scrolling

	// Create content with many lines
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "line " + string(rune('0'+i%10))
	}
	content := ""
	for i, line := range lines {
		if i > 0 {
			content += "\n"
		}
		content += line
	}
	editor.SetContent(content)

	// Move cursor down many times to trigger scrolling
	for i := 0; i < 30; i++ {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")}
		editor.Update(msg)
	}

	// Verify view renders without panic
	view := editor.View()
	if view == "" {
		t.Error("View should render content even with scrolling")
	}

	// Move cursor up to trigger scroll up
	for i := 0; i < 20; i++ {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("up")}
		editor.Update(msg)
	}

	// Verify view still renders
	view = editor.View()
	if view == "" {
		t.Error("View should render content after scrolling up")
	}
}

func TestEditorPane_FileTypeDetection_AllTypes(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)

	tests := []struct {
		filename string
		content  string
	}{
		{"script.sh", "#!/bin/bash\necho 'bash script'"},
		{"script.bash", "#!/bin/bash\necho 'bash script'"},
		{"script.ps1", "Write-Host 'PowerShell script'"},
		{"readme.md", "# Markdown Document\n\nThis is markdown."},
		{"unknown.txt", "plain text file"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			err := fm.CreateFile(tt.filename, tt.content)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			err = editor.LoadFile(tt.filename)
			if err != nil {
				t.Fatalf("Failed to load file: %v", err)
			}

			// Verify content loaded correctly
			if editor.GetContent() != tt.content {
				t.Errorf("LoadFile() content = %q, want %q", editor.GetContent(), tt.content)
			}

			// Verify view renders (which uses file type for display)
			editor.SetSize(80, 20)
			view := editor.View()
			if view == "" {
				t.Error("View should render content for file type")
			}
		})
	}
}

func TestEditorPane_TextEditing_Comprehensive(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)
	editor.SetFocused(true)
	editor.SetSize(50, 20)

	tests := []struct {
		name           string
		initialContent string
		operations     []struct {
			key      string
			expected string
		}
	}{
		{
			name:           "insert characters at start",
			initialContent: "",
			operations: []struct {
				key      string
				expected string
			}{
				{"h", "h"},
				{"e", "he"},
				{"l", "hel"},
				{"l", "hell"},
				{"o", "hello"},
			},
		},
		{
			name:           "insert newline",
			initialContent: "hello",
			operations: []struct {
				key      string
				expected string
			}{
				{"enter", "hello\n"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor.SetContent(tt.initialContent)

			for i, op := range tt.operations {
				// Simulate key press
				var msg tea.Msg
				switch op.key {
				case "enter":
					msg = tea.KeyMsg{Type: tea.KeyEnter}
				case "backspace":
					msg = tea.KeyMsg{Type: tea.KeyBackspace}
				default:
					// For regular characters, use KeyRunes
					msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(op.key)}
				}

				editor.Update(msg)

				// Check content after operation
				got := editor.GetContent()
				if got != op.expected {
					t.Errorf("After operation %d (%s): content = %q, want %q", i, op.key, got, op.expected)
				}
			}
		})
	}
}

func TestEditorPane_TextEditing_MultiLine(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)
	editor.SetFocused(true)
	editor.SetSize(50, 20)

	// Start with empty content
	editor.SetContent("")

	// Type "line1", press enter, type "line2"
	keys := []string{"l", "i", "n", "e", "1", "enter", "l", "i", "n", "e", "2"}
	for _, key := range keys {
		var msg tea.Msg
		if key == "enter" {
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		} else {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}
		editor.Update(msg)
	}

	expected := "line1\nline2"
	got := editor.GetContent()
	if got != expected {
		t.Errorf("Multi-line editing: content = %q, want %q", got, expected)
	}
}

func TestEditorPane_TextEditing_BackspaceAcrossLines(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)
	editor := ui.NewEditorPane(fm)
	editor.SetFocused(true)
	editor.SetSize(50, 20)

	// Set content with two lines
	editor.SetContent("line1\nline2")

	// Move to start of second line
	msg := tea.KeyMsg{Type: tea.KeyDown}
	editor.Update(msg)

	// Press backspace to merge lines
	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	editor.Update(msg)

	expected := "line1line2"
	got := editor.GetContent()
	if got != expected {
		t.Errorf("Backspace across lines: content = %q, want %q", got, expected)
	}
}

func TestEditorPane_TextEditing_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	fm := filemanager.NewFileManager(tmpDir)

	tests := []struct {
		name           string
		initialContent string
		cursorMoves    []string // simple "down", "right" etc
		operations     []string // "delete"
		expected       string
	}{
		{
			name:           "delete char in middle",
			initialContent: "abc",
			cursorMoves:    []string{"right"}, // cursor at 'b'
			operations:     []string{"delete"},
			expected:       "ac",
		},
		{
			name:           "delete char at start",
			initialContent: "abc",
			cursorMoves:    []string{}, // cursor at 'a'
			operations:     []string{"delete"},
			expected:       "bc",
		},
		{
			name:           "delete char at end",
			initialContent: "abc",
			cursorMoves:    []string{"right", "right"}, // cursor at 'c'
			operations:     []string{"delete"},
			expected:       "ab",
		},
		{
			name:           "delete merge lines",
			initialContent: "abc\ndef",
			cursorMoves:    []string{"right", "right", "right"}, // cursor at end of line 1
			operations:     []string{"delete"},
			expected:       "abcdef",
		},
		{
			name:           "delete at end of file",
			initialContent: "abc",
			cursorMoves:    []string{"right", "right", "right"}, // cursor at end of line
			operations:     []string{"delete"},
			expected:       "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor := ui.NewEditorPane(fm)
			editor.SetSize(50, 20)
			editor.SetFocused(true)
			editor.SetContent(tt.initialContent)

			// Move cursor
			for _, move := range tt.cursorMoves {
				var msg tea.Msg
				if move == "right" {
					msg = tea.KeyMsg{Type: tea.KeyRight}
				} else if move == "down" {
					msg = tea.KeyMsg{Type: tea.KeyDown}
				} else {
					msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(move)}
				}
				editor.Update(msg)
			}

			// Perform deletes
			for _, op := range tt.operations {
				var msg tea.Msg
				if op == "delete" {
					msg = tea.KeyMsg{Type: tea.KeyDelete}
				}
				editor.Update(msg)
			}

			got := editor.GetContent()
			if got != tt.expected {
				t.Errorf("content = %q, want %q", got, tt.expected)
			}
		})
	}
}
