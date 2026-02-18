# Config Editor Enhancement - Cursor Navigation

## Summary

Added full cursor navigation support to the config editor, allowing users to position the cursor anywhere within a field value and make precise edits without retyping entire lines.

## What Changed

### New Features

1. **Left/Right Arrow Navigation**: Move cursor character by character
2. **Home/End Keys**: Jump to start or end of field
3. **Delete Key**: Delete character at cursor position
4. **Improved Backspace**: Delete character before cursor (works at any position)
5. **Insert at Cursor**: Characters are inserted at cursor position, not just appended
6. **Visual Cursor**: Block cursor (█) shows current position in real-time

### User Experience Improvement

**Before:**
```
1. Press Enter to edit
2. Type characters (append only)
3. Use Backspace to delete from end
4. To fix a typo in the middle: Delete everything and retype
```

**After:**
```
1. Press Enter to edit
2. Use ←/→ to position cursor anywhere
3. Type to insert at cursor position
4. Use Backspace/Delete to remove specific characters
5. Make precise edits without retyping
```

## Code Changes

### File: internal/ui/aichat.go

#### 1. Added Cursor Position Field

```go
type AIChatPane struct {
    // ... existing fields ...
    editCursorPos    int    // Cursor position within edit buffer
}
```

#### 2. Enhanced Key Handling

Added support for:
- `left` - Move cursor left
- `right` - Move cursor right
- `home` - Jump to start
- `end` - Jump to end
- `delete` - Delete character at cursor
- Updated `backspace` - Delete character before cursor
- Updated character insertion - Insert at cursor position

#### 3. Updated Rendering

```go
// Before
Render(prefix + editBuffer + "█")

// After
beforeCursor := editBuffer[:cursorPos]
afterCursor := editBuffer[cursorPos:]
Render(prefix + beforeCursor + "█" + afterCursor)
```

#### 4. Updated Instructions

```
Before: [↑↓] Navigate | [Enter] Edit/Save | [Esc] Save & Exit
After:  [↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor | [Esc] Save & Exit
```

## Documentation Updates

### Updated Files

1. **docs/CONFIG_EDITOR.md**
   - Added cursor navigation keys to keyboard shortcuts table
   - Added Home/End/Delete key descriptions

2. **docs/QUICKSTART_CONFIG.md**
   - Updated Step 3 with cursor navigation instructions
   - Added examples of using arrow keys

3. **docs/CONFIG_CURSOR_NAVIGATION.md** (NEW)
   - Comprehensive guide to cursor navigation
   - Usage examples with step-by-step instructions
   - Visual representations
   - Tips and troubleshooting

## Usage Examples

### Example 1: Fix Typo in Middle

Change `http://localhots:11434` to `http://localhost:11434`

```
1. Navigate to ollama_url field
2. Press Enter
3. Use → to position: http://localhots█:11434
4. Press Backspace: http://localhost█:11434
5. Type 's': http://localhost█:11434
6. Press Enter to save
```

### Example 2: Change Port

Change `http://localhost:11434` to `http://localhost:8080`

```
1. Navigate to ollama_url field
2. Press Enter
3. Press ← four times: http://localhost:█11434
4. Press Delete five times: http://localhost:█
5. Type '8080': http://localhost:8080█
6. Press Enter to save
```

### Example 3: Insert in Middle

Change `/home/user/workspace` to `/home/user/my-workspace`

```
1. Navigate to workspace field
2. Press Enter
3. Use → to position: /home/user/█workspace
4. Type 'my-': /home/user/my-█workspace
5. Press Enter to save
```

## Technical Implementation

### Cursor Position Management

- **Initial Position**: Cursor starts at end when entering edit mode
- **Bounds Checking**: Cursor cannot move before start (0) or after end (len)
- **Character Operations**: All operations respect cursor position

### String Manipulation

**Insert Character:**
```go
editBuffer = editBuffer[:cursorPos] + char + editBuffer[cursorPos:]
cursorPos++
```

**Delete Before Cursor (Backspace):**
```go
if cursorPos > 0 {
    editBuffer = editBuffer[:cursorPos-1] + editBuffer[cursorPos:]
    cursorPos--
}
```

**Delete At Cursor (Delete):**
```go
if cursorPos < len(editBuffer) {
    editBuffer = editBuffer[:cursorPos] + editBuffer[cursorPos+1:]
}
```

## Testing

### Build Status
```bash
go build -o build/ti
✓ Build successful
```

### Test Results
```bash
go test ./internal/ui/... -v
PASS
ok      github.com/user/terminal-intelligence/internal/ui       0.543s
```

All existing tests pass. The cursor navigation is purely an enhancement to the editing experience and doesn't affect other functionality.

## Keyboard Reference

### When Editing a Field

| Key | Action |
|-----|--------|
| ← | Move cursor left |
| → | Move cursor right |
| Home | Jump to start |
| End | Jump to end |
| Backspace | Delete before cursor |
| Delete | Delete at cursor |
| Any char | Insert at cursor |
| Enter | Save changes |
| Esc | Cancel editing |

### When Navigating Fields

| Key | Action |
|-----|--------|
| ↑/↓ or K/J | Move between fields |
| Enter | Start editing |
| Esc/Q | Save and exit config mode |

## Benefits

1. **Precision**: Edit exactly where you need to
2. **Efficiency**: No need to retype entire values
3. **Familiarity**: Works like standard text editors
4. **Visual Feedback**: Cursor shows current position
5. **Flexibility**: Insert, delete, or replace at any position

## Future Enhancements

Potential improvements for future versions:

1. **Word Navigation**: Ctrl+← and Ctrl+→ to jump by words
2. **Select Text**: Shift+arrows for text selection
3. **Copy/Paste**: Clipboard operations
4. **Undo/Redo**: Multi-level undo/redo
5. **Clear Line**: Ctrl+U to clear entire line
6. **Kill to End**: Ctrl+K to delete from cursor to end
7. **Search/Replace**: Find and replace within field values

## Conclusion

The cursor navigation enhancement makes the config editor much more user-friendly and efficient. Users can now make precise edits to configuration values without the frustration of retyping entire lines, bringing the editing experience closer to what they expect from modern text editors.
