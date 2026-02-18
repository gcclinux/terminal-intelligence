# Config Editor - Cursor Navigation Feature

## Overview

The config editor now supports full cursor navigation, allowing you to position the cursor anywhere within a field value and make precise edits without retyping the entire line.

## New Capabilities

### Before (Previous Version)
- Press Enter to edit a field
- Type characters - they append to the end
- Use Backspace to delete from the end only
- Had to delete entire value and retype to change beginning or middle

### After (Current Version)
- Press Enter to edit a field (cursor starts at end)
- Use ←/→ arrow keys to move cursor anywhere in the text
- Use Home/End to jump to start/end
- Use Backspace to delete character before cursor
- Use Delete to delete character at cursor
- Type characters - they insert at cursor position
- Make precise edits anywhere in the value

## Keyboard Controls (When Editing)

| Key | Action | Example |
|-----|--------|---------|
| ← (Left Arrow) | Move cursor one position left | `llama█2` → `llam█a2` |
| → (Right Arrow) | Move cursor one position right | `llam█a2` → `llama█2` |
| Home | Jump cursor to start of field | `llama2█` → `█llama2` |
| End | Jump cursor to end of field | `█llama2` → `llama2█` |
| Backspace | Delete character before cursor | `llam█a2` → `lla█a2` |
| Delete | Delete character at cursor | `llam█a2` → `llam█2` |
| Any character | Insert at cursor position | `llam█2` → `llama█2` |
| Enter | Save the edited value | - |
| Esc | Cancel editing (discard changes) | - |

## Usage Examples

### Example 1: Fixing a Typo in the Middle

**Scenario:** You typed `http://localhots:11434` instead of `http://localhost:11434`

**Steps:**
1. Navigate to `ollama_url` field
2. Press Enter to edit
3. Press Home to go to start
4. Press → repeatedly or hold → to move to the typo: `http://localhots█:11434`
5. Press Backspace to delete 's': `http://localhost█:11434`
6. Type 's': `http://localhost█:11434`
7. Wait, that's wrong. Press ← to go back: `http://localhos█t:11434`
8. Press Delete to remove 't': `http://localhos█:11434`
9. Type 't': `http://localhost█:11434`
10. Press Enter to save

### Example 2: Changing Port Number

**Scenario:** Change `http://localhost:11434` to `http://localhost:8080`

**Steps:**
1. Navigate to `ollama_url` field
2. Press Enter to edit
3. Press End to ensure cursor is at end (it starts there by default)
4. Press ← four times to position before port: `http://localhost:█11434`
5. Press Delete five times to remove `11434`: `http://localhost:█`
6. Type `8080`: `http://localhost:8080█`
7. Press Enter to save

### Example 3: Adding to the Beginning

**Scenario:** Change `llama2` to `qwen2.5-coder:3b`

**Steps:**
1. Navigate to `model` field
2. Press Enter to edit
3. Press Home to go to start: `█llama2`
4. Press Delete six times to clear: `█`
5. Type `qwen2.5-coder:3b`: `qwen2.5-coder:3b█`
6. Press Enter to save

Or more efficiently:
1. Navigate to `model` field
2. Press Enter to edit
3. Press Ctrl+A (select all) - wait, that's not implemented yet
4. Just use Home, then Delete repeatedly, or Backspace from end

### Example 4: Inserting in the Middle

**Scenario:** Change `/home/user/workspace` to `/home/user/my-workspace`

**Steps:**
1. Navigate to `workspace` field
2. Press Enter to edit
3. Use → to position cursor: `/home/user/█workspace`
4. Type `my-`: `/home/user/my-█workspace`
5. Press Enter to save

## Visual Representation

The cursor is shown as a block character (█) at the current position:

```
Not Editing:
> model: llama2

Editing (cursor at end):
> model: llama2█

Editing (cursor in middle):
> model: lla█ma2

Editing (cursor at start):
> model: █llama2
```

## Tips

1. **Start Position**: When you press Enter to edit, the cursor starts at the end of the current value
2. **Quick Navigation**: Use Home/End to jump to start/end instead of holding arrow keys
3. **Precise Edits**: Position cursor exactly where you need to make changes
4. **Cancel Anytime**: Press Esc to cancel editing and restore the original value
5. **Visual Feedback**: The cursor (█) always shows your current position

## Technical Details

### Cursor Position Tracking

The config editor maintains:
- `editBuffer`: The current text being edited
- `editCursorPos`: Integer position (0 = start, len(editBuffer) = end)

### Character Operations

**Insert:**
```go
editBuffer = editBuffer[:cursorPos] + newChar + editBuffer[cursorPos:]
cursorPos++
```

**Delete (Backspace):**
```go
editBuffer = editBuffer[:cursorPos-1] + editBuffer[cursorPos:]
cursorPos--
```

**Delete (Delete key):**
```go
editBuffer = editBuffer[:cursorPos] + editBuffer[cursorPos+1:]
// cursorPos stays the same
```

### Rendering

The cursor is rendered by splitting the buffer:
```go
beforeCursor := editBuffer[:cursorPos]
afterCursor := editBuffer[cursorPos:]
display := prefix + beforeCursor + "█" + afterCursor
```

## Comparison with Standard Text Editors

| Feature | Config Editor | Vi/Nano | Notes |
|---------|--------------|---------|-------|
| Left/Right navigation | ✓ | ✓ | Same |
| Home/End | ✓ | ✓ | Same |
| Backspace | ✓ | ✓ | Same |
| Delete | ✓ | ✓ | Same |
| Insert at cursor | ✓ | ✓ | Same |
| Word jump (Ctrl+←/→) | ✗ | ✓ | Future enhancement |
| Select text | ✗ | ✓ | Future enhancement |
| Copy/Paste | ✗ | ✓ | Future enhancement |
| Undo/Redo | ✗ | ✓ | Future enhancement |

## Future Enhancements

Possible improvements:
1. **Word Navigation**: Ctrl+← and Ctrl+→ to jump by words
2. **Select Text**: Shift+arrows to select text
3. **Copy/Paste**: Ctrl+C/Ctrl+V for clipboard operations
4. **Undo/Redo**: Ctrl+Z/Ctrl+Y for undo/redo
5. **Clear Line**: Ctrl+U to clear entire line
6. **Kill to End**: Ctrl+K to delete from cursor to end

## Troubleshooting

**Cursor not moving:**
- Make sure you're in edit mode (press Enter on a field first)
- Check that you're using arrow keys, not vim keys (h/j/k/l don't work in edit mode)

**Character appearing in wrong place:**
- Check cursor position (look for the █ block)
- Use arrow keys to position cursor before typing

**Can't delete character:**
- Backspace deletes before cursor (cursor must not be at start)
- Delete deletes at cursor (cursor must not be at end)

**Lost my changes:**
- Press Enter to save before pressing Esc
- Esc cancels the current edit and restores original value
