package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/terminal-intelligence/internal/filemanager"
	"github.com/user/terminal-intelligence/internal/types"
)

// EditorPane manages the code editor pane with syntax highlighting and file editing.
// This component provides a full-featured text editor with:
//   - Cursor navigation (arrow keys, home, end)
//   - Text editing (insert, delete, backspace, newline)
//   - File operations (load, save, close)
//   - Unsaved changes tracking
//   - Line numbering
//   - Scrolling for large files
//   - Visual cursor indicator
//
// The editor supports multiple file types (bash, shell, powershell, markdown) and
// integrates with the AgenticCodeFixer for autonomous code modifications.
//
// File Type Detection:
// File types are determined by extension:
//   - .sh, .bash -> bash
//   - .ps1 -> powershell
//   - .md -> markdown
//   - default -> shell
//
// Unsaved Changes:
// The editor tracks modifications by comparing current content with originalContent.
// This enables the exit confirmation dialog and modified indicator (*) in the title bar.
type EditorPane struct {
	content         string                   // Current editor content
	originalContent string                   // Original content for unsaved changes detection
	cursorLine      int                      // Current cursor line (0-indexed)
	cursorCol       int                      // Current cursor column (0-indexed)
	scrollOffset    int                      // Vertical scroll offset
	currentFile     *types.FileMetadata      // Current file metadata (nil if no file open)
	fileManager     *filemanager.FileManager // File system operations
	width           int                      // Pane width
	height          int                      // Pane height
	focused         bool                     // Whether this pane is focused
	diffMarkers     map[int]string           // Tracks red/green line styling for diffs
	undoStack       []editorSnapshot         // Undo history
	redoStack       []editorSnapshot         // Redo history
	pendingAltD     bool                     // Waiting for second key after Alt+D
	suggestedName   string                   // AI-suggested filename for unsaved buffer
}

// editorSnapshot stores editor state for undo/redo
type editorSnapshot struct {
	content    string
	cursorLine int
	cursorCol  int
}

// NewEditorPane creates a new editor pane.
// Initializes an empty editor with cursor at position (0, 0).
//
// Parameters:
//   - fm: FileManager for file system operations
//
// Returns:
//   - *EditorPane: Initialized editor pane
func NewEditorPane(fm *filemanager.FileManager) *EditorPane {
	return &EditorPane{
		content:         "",
		originalContent: "",
		cursorLine:      0,
		cursorCol:       0,
		scrollOffset:    0,
		currentFile:     nil,
		fileManager:     fm,
		width:           0,
		height:          0,
		focused:         false,
		diffMarkers:     make(map[int]string),
	}
}

// LoadFile loads a file into the editor.
// Reads the file content, normalizes line endings (CRLF -> LF), and resets cursor position.
// Determines file type from extension and creates FileMetadata.
//
// Line ending normalization:
//   - \r\n (Windows) -> \n
//   - \r (old Mac) -> \n
//   - \n (Unix) -> \n (unchanged)
//
// Parameters:
//   - filepath: Path to the file to load
//
// Returns:
//   - error: Error if file cannot be read, nil on success
func (e *EditorPane) LoadFile(filepath string) error {
	content, err := e.fileManager.ReadFile(filepath)
	if err != nil {
		return err
	}

	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	e.content = content
	e.originalContent = content
	e.cursorLine = 0
	e.cursorCol = 0
	e.scrollOffset = 0
	e.diffMarkers = make(map[int]string)

	// Determine file type from extension
	fileType := determineFileType(filepath)

	e.currentFile = &types.FileMetadata{
		Filepath:   filepath,
		FileType:   fileType,
		IsModified: false,
	}

	return nil
}

// SaveFile saves current editor content to disk.
// Updates originalContent to match current content and clears the modified flag.
//
// Returns:
//   - error: Error if no file is loaded or write fails, nil on success
func (e *EditorPane) SaveFile() error {
	if e.currentFile == nil {
		return fmt.Errorf("no file loaded")
	}

	if len(e.diffMarkers) > 0 {
		lines := strings.Split(e.content, "\n")
		var cleaned []string
		for i, line := range lines {
			if color, ok := e.diffMarkers[i]; ok {
				if color == "red" {
					continue
				}
			}
			cleaned = append(cleaned, line)
		}
		e.content = strings.Join(cleaned, "\n")
		e.diffMarkers = make(map[int]string)

		if e.cursorLine >= len(cleaned) {
			e.cursorLine = len(cleaned) - 1
			if e.cursorLine < 0 {
				e.cursorLine = 0
			}
		}
	}

	err := e.fileManager.WriteFile(e.currentFile.Filepath, e.content)
	if err != nil {
		return err
	}

	e.originalContent = e.content
	e.currentFile.IsModified = false

	return nil
}

// CloseFile closes the current file and clears the editor
func (e *EditorPane) CloseFile() {
	e.content = ""
	e.originalContent = ""
	e.cursorLine = 0
	e.cursorCol = 0
	e.scrollOffset = 0
	e.currentFile = nil
	e.diffMarkers = nil
}

// GetContent returns current editor content.
// This includes any unsaved changes.
//
// Returns:
//   - string: Current editor content
func (e *EditorPane) GetContent() string {
	return e.content
}

// SetContent sets editor content.
// Updates the modified flag by comparing with originalContent.
// This method is used by AgenticCodeFixer to apply code fixes.
//
// Parameters:
//   - content: New content to set
func (e *EditorPane) SetContent(content string) {
	lines := strings.Split(content, "\n")
	var cleaned []string
	e.diffMarkers = make(map[int]string)

	for _, line := range lines {
		if strings.HasPrefix(line, "~DEL~") {
			e.diffMarkers[len(cleaned)] = "red"
			cleaned = append(cleaned, strings.TrimPrefix(line, "~DEL~"))
		} else if strings.HasPrefix(line, "~ADD~") {
			e.diffMarkers[len(cleaned)] = "green"
			cleaned = append(cleaned, strings.TrimPrefix(line, "~ADD~"))
		} else {
			cleaned = append(cleaned, line)
		}
	}

	e.content = strings.Join(cleaned, "\n")
	if e.currentFile != nil {
		e.currentFile.IsModified = (e.content != e.originalContent) || len(e.diffMarkers) > 0
	}
}

// SetContentUnsaved loads content into the editor without an associated file.
// If suggestedName is provided, it will be used as the default filename on save.
func (e *EditorPane) SetContentUnsaved(content string, suggestedName string) {
	e.content = content
	e.originalContent = ""
	e.cursorLine = 0
	e.cursorCol = 0
	e.scrollOffset = 0
	e.currentFile = nil
	e.diffMarkers = make(map[int]string)
	e.suggestedName = suggestedName
	e.undoStack = nil
	e.redoStack = nil
}

// HasUnsavedChanges checks if editor has unsaved changes.
// Compares current content with originalContent (content at last save/load).
//
// Returns:
//   - bool: True if content differs from originalContent, false otherwise
func (e *EditorPane) HasUnsavedChanges() bool {
	return e.content != e.originalContent || len(e.diffMarkers) > 0
}

// GetCurrentLine returns the text of the line currently under the cursor.
// Used for copying the current line to clipboard.
//
// Returns:
//   - string: The current line text
func (e *EditorPane) GetCurrentLine() string {
	lines := strings.Split(e.content, "\n")
	if e.cursorLine >= 0 && e.cursorLine < len(lines) {
		return lines[e.cursorLine]
	}
	return ""
}

// Update handles messages for the editor pane.
// Only processes messages when the pane is focused.
// Delegates keyboard input to handleKeyPress.
//
// Parameters:
//   - msg: The message to handle
//
// Returns:
//   - tea.Cmd: Command to execute (can be nil)
func (e *EditorPane) Update(msg tea.Msg) tea.Cmd {
	if !e.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return e.handleKeyPress(msg)
		// Removed WindowSizeMsg handling - size is set by app.go
	}

	return nil
}

// handleKeyPress handles keyboard input for the editor.
// Supports standard text editing operations:
//   - Arrow keys: Navigate cursor
//   - Enter: Insert newline
//   - Backspace: Delete character before cursor
//   - Delete: Delete character at cursor
//   - Printable characters: Insert at cursor
//
// Cursor behavior:
//   - Left/Right at line boundaries: Move to adjacent line
//   - Up/Down: Adjust column if new line is shorter
//   - Scrolling: Automatically adjusts to keep cursor visible
//
// Parameters:
//   - msg: The key message to handle
//
// Returns:
//   - tea.Cmd: Command to execute (currently always nil)
func (e *EditorPane) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	lines := strings.Split(e.content, "\n")
	keyStr := msg.String()

	// Handle pending Alt+D sequence (waiting for d/w/number)
	if e.pendingAltD {
		e.pendingAltD = false
		// Accept both plain and alt+ variants (user may still hold Alt)
		switch {
		case keyStr == "d" || keyStr == "D" || keyStr == "alt+d" || keyStr == "alt+D":
			e.deleteLine()
			return nil
		case keyStr == "w" || keyStr == "W" || keyStr == "alt+w" || keyStr == "alt+W":
			e.deleteWord()
			return nil
		case keyStr >= "1" && keyStr <= "9":
			n := int(keyStr[0] - '0')
			e.deleteLines(n)
			return nil
		default:
			if len(keyStr) == 5 && keyStr[:4] == "alt+" && keyStr[4] >= '1' && keyStr[4] <= '9' {
				n := int(keyStr[4] - '0')
				e.deleteLines(n)
				return nil
			}
			return nil
		}
	}

	switch keyStr {
	case "alt+d":
		// Start Alt+D sequence, wait for next key
		e.pendingAltD = true
		return nil
	// Direct Alt shortcuts for delete (single press alternatives)
	case "alt+l":
		// Alt+L = delete line (single-key alternative to Alt+D,D)
		e.deleteLine()
		return nil
	case "alt+w":
		// Alt+W = delete word (single-key alternative to Alt+D,W)
		e.deleteWord()
		return nil
	case "alt+u":
		e.undo()
		return nil
	case "alt+r":
		e.redo()
		return nil
	case "alt+g":
		// Go to end of file
		e.cursorLine = len(lines) - 1
		if e.cursorLine < 0 {
			e.cursorLine = 0
		}
		e.cursorCol = len(lines[e.cursorLine])
		e.adjustScroll()
		return nil
	case "alt+h":
		// Go to top of file
		e.cursorLine = 0
		e.cursorCol = 0
		e.adjustScroll()
		return nil
	case "up":
		if e.cursorLine > 0 {
			e.cursorLine--
			// Adjust cursor column if new line is shorter
			if e.cursorLine < len(lines) && e.cursorCol > len(lines[e.cursorLine]) {
				e.cursorCol = len(lines[e.cursorLine])
			}
			e.adjustScroll()
		}
	case "down":
		if e.cursorLine < len(lines)-1 {
			e.cursorLine++
			// Adjust cursor column if new line is shorter
			if e.cursorLine < len(lines) && e.cursorCol > len(lines[e.cursorLine]) {
				e.cursorCol = len(lines[e.cursorLine])
			}
			e.adjustScroll()
		}
	case "left":
		if e.cursorCol > 0 {
			e.cursorCol--
		} else if e.cursorLine > 0 {
			// Move to end of previous line
			e.cursorLine--
			if e.cursorLine < len(lines) {
				e.cursorCol = len(lines[e.cursorLine])
			}
			e.adjustScroll()
		}
	case "right":
		if e.cursorLine < len(lines) {
			if e.cursorCol < len(lines[e.cursorLine]) {
				e.cursorCol++
			} else if e.cursorLine < len(lines)-1 {
				// Move to start of next line
				e.cursorLine++
				e.cursorCol = 0
				e.adjustScroll()
			}
		}
	case "enter":
		e.insertNewline()
	case "backspace":
		e.deleteChar()
	case "delete":
		e.deleteNextChar()
	default:
		// Insert regular characters
		if msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				if r >= 32 || r == '\t' {
					e.insertChar(string(r))
				}
			}
		}
	}

	return nil
}

// shiftMarkers shifts diff markers when rows are inserted or deleted
func (e *EditorPane) shiftMarkers(fromLine int, amount int) {
	if len(e.diffMarkers) == 0 {
		return
	}
	newMarkers := make(map[int]string)
	for idx, color := range e.diffMarkers {
		if idx >= fromLine {
			if amount > 0 {
				newMarkers[idx+amount] = color
			} else {
				if idx < fromLine-amount {
					// dropped line
				} else {
					newMarkers[idx+amount] = color
				}
			}
		} else {
			newMarkers[idx] = color
		}
	}
	e.diffMarkers = newMarkers
}

// insertChar inserts a character at the cursor position.
// Updates the modified flag after insertion.
//
// Parameters:
//   - char: The character to insert
func (e *EditorPane) insertChar(char string) {
	e.saveSnapshot()
	lines := strings.Split(e.content, "\n")
	if e.cursorLine >= len(lines) {
		lines = append(lines, "")
	}

	line := lines[e.cursorLine]
	if e.cursorCol > len(line) {
		e.cursorCol = len(line)
	}

	newLine := line[:e.cursorCol] + char + line[e.cursorCol:]
	lines[e.cursorLine] = newLine
	e.cursorCol++

	e.content = strings.Join(lines, "\n")
	if e.currentFile != nil {
		e.currentFile.IsModified = (e.content != e.originalContent)
	}
}

// insertNewline inserts a newline at the cursor position.
// Splits the current line at the cursor, creating a new line below.
// Moves cursor to the beginning of the new line.
// Updates the modified flag and adjusts scroll.
func (e *EditorPane) insertNewline() {
	e.saveSnapshot()
	lines := strings.Split(e.content, "\n")
	if e.cursorLine >= len(lines) {
		lines = append(lines, "")
		e.cursorLine = len(lines) - 1
		e.cursorCol = 0
	} else {
		line := lines[e.cursorLine]
		if e.cursorCol > len(line) {
			e.cursorCol = len(line)
		}

		// Split the line at cursor position
		before := line[:e.cursorCol]
		after := line[e.cursorCol:]

		// Update current line and insert new line
		lines[e.cursorLine] = before
		newLines := make([]string, len(lines)+1)
		copy(newLines, lines[:e.cursorLine+1])
		newLines[e.cursorLine+1] = after
		copy(newLines[e.cursorLine+2:], lines[e.cursorLine+1:])
		lines = newLines

		e.cursorLine++
		e.cursorCol = 0
		e.shiftMarkers(e.cursorLine, 1)
	}

	e.content = strings.Join(lines, "\n")
	if e.currentFile != nil {
		e.currentFile.IsModified = (e.content != e.originalContent) || len(e.diffMarkers) > 0
	}
	e.adjustScroll()
}

// deleteChar deletes the character before the cursor (backspace behavior).
// If at the beginning of a line, merges with the previous line.
// Updates the modified flag and adjusts scroll.
func (e *EditorPane) deleteChar() {
	e.saveSnapshot()
	lines := strings.Split(e.content, "\n")
	if e.cursorLine >= len(lines) {
		return
	}

	if e.cursorCol > 0 {
		// Delete character in current line (backspace)
		line := lines[e.cursorLine]
		if e.cursorCol <= len(line) {
			newLine := line[:e.cursorCol-1] + line[e.cursorCol:]
			lines[e.cursorLine] = newLine
			e.cursorCol--
		}
	} else if e.cursorLine > 0 {
		// Merge with previous line
		prevLine := lines[e.cursorLine-1]
		currentLine := lines[e.cursorLine]
		lines[e.cursorLine-1] = prevLine + currentLine

		// Remove current line
		newLines := make([]string, len(lines)-1)
		copy(newLines, lines[:e.cursorLine])
		copy(newLines[e.cursorLine:], lines[e.cursorLine+1:])
		lines = newLines

		e.shiftMarkers(e.cursorLine, -1)
		e.cursorLine--
		e.cursorCol = len(prevLine)
		e.adjustScroll()
	}

	e.content = strings.Join(lines, "\n")
	if e.currentFile != nil {
		e.currentFile.IsModified = (e.content != e.originalContent) || len(e.diffMarkers) > 0
	}
}

// deleteNextChar deletes the character at the cursor position (delete key behavior).
// If at the end of a line, merges with the next line.
// Updates the modified flag.
func (e *EditorPane) deleteNextChar() {
	e.saveSnapshot()
	lines := strings.Split(e.content, "\n")
	if e.cursorLine >= len(lines) {
		return
	}

	line := lines[e.cursorLine]

	// If cursor within the line, delete character at cursor
	if e.cursorCol < len(line) {
		// e.g. "abc", cursor at 1 ('b'). New: "ac"
		newLine := line[:e.cursorCol] + line[e.cursorCol+1:]
		lines[e.cursorLine] = newLine
		// cursorCol stays the same
	} else if e.cursorCol == len(line) && e.cursorLine < len(lines)-1 {
		// At end of line, merge with next line
		nextLine := lines[e.cursorLine+1]
		lines[e.cursorLine] = line + nextLine

		// Remove next line
		newLines := make([]string, len(lines)-1)
		copy(newLines, lines[:e.cursorLine+1])
		copy(newLines[e.cursorLine+1:], lines[e.cursorLine+2:])
		lines = newLines
		e.shiftMarkers(e.cursorLine+1, -1)
		// cursorCol stays the same (at the join point)
	}

	e.content = strings.Join(lines, "\n")
	if e.currentFile != nil {
		e.currentFile.IsModified = (e.content != e.originalContent) || len(e.diffMarkers) > 0
	}
}

// saveSnapshot pushes current state onto the undo stack and clears redo
func (e *EditorPane) saveSnapshot() {
	e.undoStack = append(e.undoStack, editorSnapshot{
		content:    e.content,
		cursorLine: e.cursorLine,
		cursorCol:  e.cursorCol,
	})
	e.redoStack = nil
}

// undo restores the previous editor state
func (e *EditorPane) undo() {
	if len(e.undoStack) == 0 {
		return
	}
	// Push current state to redo
	e.redoStack = append(e.redoStack, editorSnapshot{
		content:    e.content,
		cursorLine: e.cursorLine,
		cursorCol:  e.cursorCol,
	})
	snap := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	e.content = snap.content
	e.cursorLine = snap.cursorLine
	e.cursorCol = snap.cursorCol
	if e.currentFile != nil {
		e.currentFile.IsModified = (e.content != e.originalContent) || len(e.diffMarkers) > 0
	}
	e.adjustScroll()
}

// redo restores the next editor state
func (e *EditorPane) redo() {
	if len(e.redoStack) == 0 {
		return
	}
	e.undoStack = append(e.undoStack, editorSnapshot{
		content:    e.content,
		cursorLine: e.cursorLine,
		cursorCol:  e.cursorCol,
	})
	snap := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	e.content = snap.content
	e.cursorLine = snap.cursorLine
	e.cursorCol = snap.cursorCol
	if e.currentFile != nil {
		e.currentFile.IsModified = (e.content != e.originalContent) || len(e.diffMarkers) > 0
	}
	e.adjustScroll()
}

// deleteLine deletes the current line
func (e *EditorPane) deleteLine() {
	e.saveSnapshot()
	lines := strings.Split(e.content, "\n")
	if len(lines) == 0 {
		return
	}
	if e.cursorLine >= len(lines) {
		e.cursorLine = len(lines) - 1
	}
	lines = append(lines[:e.cursorLine], lines[e.cursorLine+1:]...)
	if len(lines) == 0 {
		lines = []string{""}
	}
	e.content = strings.Join(lines, "\n")
	if e.cursorLine >= len(lines) {
		e.cursorLine = len(lines) - 1
	}
	if e.cursorLine < 0 {
		e.cursorLine = 0
	}
	if e.cursorCol > len(strings.Split(e.content, "\n")[e.cursorLine]) {
		e.cursorCol = len(strings.Split(e.content, "\n")[e.cursorLine])
	}
	if e.currentFile != nil {
		e.currentFile.IsModified = (e.content != e.originalContent) || len(e.diffMarkers) > 0
	}
	e.adjustScroll()
}

// deleteLines deletes n lines starting from the current line
func (e *EditorPane) deleteLines(n int) {
	e.saveSnapshot()
	lines := strings.Split(e.content, "\n")
	if len(lines) == 0 || n <= 0 {
		return
	}
	if e.cursorLine >= len(lines) {
		e.cursorLine = len(lines) - 1
	}
	end := e.cursorLine + n
	if end > len(lines) {
		end = len(lines)
	}
	lines = append(lines[:e.cursorLine], lines[end:]...)
	if len(lines) == 0 {
		lines = []string{""}
	}
	e.content = strings.Join(lines, "\n")
	if e.cursorLine >= len(lines) {
		e.cursorLine = len(lines) - 1
	}
	if e.cursorLine < 0 {
		e.cursorLine = 0
	}
	curLines := strings.Split(e.content, "\n")
	if e.cursorCol > len(curLines[e.cursorLine]) {
		e.cursorCol = len(curLines[e.cursorLine])
	}
	if e.currentFile != nil {
		e.currentFile.IsModified = (e.content != e.originalContent) || len(e.diffMarkers) > 0
	}
	e.adjustScroll()
}

// deleteWord deletes from cursor to end of current word (or next word boundary)
func (e *EditorPane) deleteWord() {
	e.saveSnapshot()
	lines := strings.Split(e.content, "\n")
	if e.cursorLine >= len(lines) {
		return
	}
	line := lines[e.cursorLine]
	if e.cursorCol >= len(line) {
		return
	}
	// Find end of word: skip non-spaces, then skip spaces
	pos := e.cursorCol
	// Skip current word characters
	for pos < len(line) && line[pos] != ' ' && line[pos] != '\t' {
		pos++
	}
	// Skip trailing whitespace
	for pos < len(line) && (line[pos] == ' ' || line[pos] == '\t') {
		pos++
	}
	lines[e.cursorLine] = line[:e.cursorCol] + line[pos:]
	e.content = strings.Join(lines, "\n")
	if e.currentFile != nil {
		e.currentFile.IsModified = (e.content != e.originalContent) || len(e.diffMarkers) > 0
	}
}

// adjustScroll adjusts the scroll offset to keep cursor visible.
// Scrolls down if cursor is below visible area.
// Scrolls up if cursor is above visible area.
// Ensures scroll offset is never negative.
func (e *EditorPane) adjustScroll() {
	visibleLines := e.height - 2 // Account for borders
	if visibleLines < 1 {
		visibleLines = 1
	}

	// Scroll down if cursor is below visible area
	if e.cursorLine >= e.scrollOffset+visibleLines {
		e.scrollOffset = e.cursorLine - visibleLines + 1
	}

	// Scroll up if cursor is above visible area
	if e.cursorLine < e.scrollOffset {
		e.scrollOffset = e.cursorLine
	}

	// Ensure scroll offset is not negative
	if e.scrollOffset < 0 {
		e.scrollOffset = 0
	}
}

// View renders the editor pane.
// Displays visible lines with line numbers, cursor indicator, and border.
//
// Rendering details:
//   - Line numbers: 3-digit format, gray color
//   - Cursor: Reverse video on current character (or space at end of line)
//   - Long lines: Truncated with "..." to prevent wrapping
//   - Empty lines below content: Shown as "~" (vim-style)
//   - Border: Blue when focused, gray when unfocused
//
// Returns:
//   - string: Rendered editor pane
func (e *EditorPane) View() string {
	lines := strings.Split(e.content, "\n")

	// Calculate how many lines fit in the panel
	// e.height is the total pane height set by app.go
	// Subtract 2 for the border (top + bottom)
	visibleLines := e.height - 2
	if visibleLines < 1 {
		visibleLines = 1
	}

	totalLines := len(lines)

	// Calculate max line width to prevent wrapping
	// Available space:
	// Width(e.width - 4) -> -4
	// Border -> -2 (left/right)
	// Padding(0, 0) -> 0
	// Total available content width = e.width - 6
	//
	// Content usage:
	// LineNum (3) + Space (1) + Content (X) = X + 4
	//
	// Constraint: X + 4 <= e.width - 6  =>  X <= e.width - 10
	// Using -12 to be safe and prevent any wrapping
	maxLineWidth := e.width - 12
	if maxLineWidth < 10 {
		maxLineWidth = 10
	}

	// Render exactly visibleLines lines
	var renderedLines []string
	for i := 0; i < visibleLines; i++ {
		fileLineIdx := e.scrollOffset + i

		if fileLineIdx < totalLines {
			lineNum := fileLineIdx + 1
			lineNumStr := fmt.Sprintf("%3d", lineNum)
			lineNumStyled := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render(lineNumStr)

			line := lines[fileLineIdx]

			// Truncate long lines to strictly prevent wrapping
			if len(line) > maxLineWidth {
				line = line[:maxLineWidth-3] + "..."
			}

			// Highlight cursor line
			if fileLineIdx == e.cursorLine && e.focused {
				cursorStyle := lipgloss.NewStyle().Reverse(true)
				if e.cursorCol < len(line) {
					line = line[:e.cursorCol] + cursorStyle.Render(string(line[e.cursorCol])) + line[e.cursorCol+1:]
				} else {
					line += cursorStyle.Render(" ")
				}
			}

			if color, ok := e.diffMarkers[fileLineIdx]; ok {
				if color == "red" {
					line = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(line)
				} else if color == "green" {
					line = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(line)
				}
			}

			renderedLines = append(renderedLines, lineNumStyled+" │ "+line)
		} else {
			renderedLines = append(renderedLines, "  ~")
		}
	}

	content := strings.Join(renderedLines, "\n")

	// Use strict Height and MaxWidth to enforce size
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 0).
		Width(e.width - 4).
		MaxWidth(e.width - 2). // Fix: total outer width is e.width - 2
		Height(e.height - 2)

	if e.focused {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("62"))
	} else {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("240"))
	}

	return borderStyle.Render(content)
}

// SetFocused sets the focus state of the editor pane.
// Affects border color and cursor visibility.
//
// Parameters:
//   - focused: Whether the pane should be focused
func (e *EditorPane) SetFocused(focused bool) {
	e.focused = focused
}

// SetSize sets the size of the editor pane.
// Called by App when window is resized.
//
// Parameters:
//   - width: New pane width
//   - height: New pane height
func (e *EditorPane) SetSize(width, height int) {
	e.width = width
	e.height = height
}

// determineFileType determines the file type from the file extension.
// Used for syntax validation and AI prompt construction.
//
// Supported extensions:
//   - .sh, .bash -> bash
//   - .ps1 -> powershell
//   - .md -> markdown
//   - default -> shell
//
// Parameters:
//   - filepath: Path to the file
//
// Returns:
//   - string: File type identifier
func determineFileType(filepath string) string {
	if strings.HasSuffix(filepath, ".sh") {
		return "bash"
	}
	if strings.HasSuffix(filepath, ".bash") {
		return "bash"
	}
	if strings.HasSuffix(filepath, ".ps1") {
		return "powershell"
	}
	if strings.HasSuffix(filepath, ".md") {
		return "markdown"
	}
	if strings.HasSuffix(filepath, ".py") {
		return "python"
	}
	if strings.HasSuffix(filepath, ".go") {
		return "go"
	}
	return "shell"
}

// GetWidth returns the width of the editor pane.
// Used for testing and layout calculations.
//
// Returns:
//   - int: Pane width
func (e *EditorPane) GetWidth() int {
	return e.width
}

// GetHeight returns the height of the editor pane.
// Used for testing and layout calculations.
//
// Returns:
//   - int: Pane height
func (e *EditorPane) GetHeight() int {
	return e.height
}

// GetFileManager returns the file manager (for testing).
// Allows tests to verify file operations.
//
// Returns:
//   - *filemanager.FileManager: The file manager instance
func (e *EditorPane) GetFileManager() *filemanager.FileManager {
	return e.fileManager
}

// FileContext holds the current file context for agentic operations.
// This structure is returned by GetCurrentFile() and provides all information
// needed by AgenticCodeFixer to process fix requests.
type FileContext struct {
	FilePath    string // Path to the current file
	FileContent string // Current editor content (includes unsaved changes)
	FileType    string // File type (bash, shell, powershell, markdown)
}

// GetCurrentFile returns the current file context including content, path, and type.
// This method is used by AgenticCodeFixer to retrieve file context for fix requests.
//
// The returned content includes any unsaved changes in the editor, ensuring that
// the AI always works with the most current version of the code.
//
// Returns:
//   - *FileContext: File context with path, content, and type (nil if no file is open)
func (e *EditorPane) GetCurrentFile() *FileContext {
	if e.currentFile == nil {
		return nil
	}

	return &FileContext{
		FilePath:    e.currentFile.Filepath,
		FileContent: e.content, // Use current editor content (includes unsaved changes)
		FileType:    e.currentFile.FileType,
	}
}

// GetSuggestedName returns the AI-suggested filename, if any.
func (e *EditorPane) GetSuggestedName() string {
	return e.suggestedName
}
