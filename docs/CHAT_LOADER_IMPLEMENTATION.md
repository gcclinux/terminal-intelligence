# Chat Loader Implementation Summary

## Overview
Implemented Ctrl+L functionality to load previously saved chat histories back into the AI pane, complementing the existing Ctrl+A save feature.

## Implementation Details

### 1. App Structure Changes (`internal/ui/app.go`)

Added new fields to the App struct:
- `showChatLoader bool` - Flag to show/hide the chat loader dialog
- `chatList []string` - List of available chat files for the picker

### 2. Ctrl+L Handler

The handler performs the following steps:
1. Checks if `.ti/` directory exists
2. Scans for all `chat-*.md` files
3. Sorts them newest first (reverse alphabetical order)
4. Opens the chat loader dialog with the list
5. Shows status message with count of found chats

### 3. Chat Loader Dialog

Similar to the backup picker, the chat loader provides:
- Scrollable list of saved chats (up to 15 visible at once)
- Navigation with `↑↓` or `k/j` keys
- Selection with `Enter` key
- Cancel with `Esc` key
- Centered modal dialog with rounded border

### 4. Chat History Parser (`loadChatHistory` method)

The parser:
1. Clears existing chat history
2. Parses the markdown file line by line
3. Identifies role lines (starting with "user " or "assistant ")
4. Extracts timestamps in HH:MM:SS format
5. Accumulates message content until next role line
6. Creates ChatMessage objects with proper timestamps
7. Adds messages to the AI pane's message list
8. Extracts code blocks from loaded messages
9. Scrolls to bottom of chat

### 5. File Format Support

The parser handles the format created by Ctrl+A:
```
role timestamp
content line 1
content line 2

role timestamp
content
```

### 6. UI Updates

Updated the AI pane title bar to show:
```
AI Responses (02) [Gemini] | Ctrl+Y: Code | Ctrl+A: Save | Ctrl+L: Load | ↑↓: Scroll | Ctrl+T: New
```

### 7. Help System Updates

Updated help text in:
- `internal/ui/help.go` - Main help dialog
- `internal/ui/app.go` - Inline help text
- `README.md` - Keyboard shortcuts table

## User Workflow

### Saving a Chat
1. User has conversation with AI
2. Presses `Ctrl+A`
3. Chat saved to `.ti/chat-YYYY-MM-DD-HH-MM-SS-query.md`
4. File opens in editor

### Loading a Chat
1. User presses `Ctrl+L` (works anytime, especially after `Ctrl+T` clear)
2. Dialog shows list of saved chats
3. User navigates with arrow keys
4. User presses `Enter` to load
5. Chat history loads into AI pane
6. Focus switches to AI pane
7. User can continue the conversation

## Benefits

- **Conversation Continuity**: Resume previous discussions seamlessly
- **Knowledge Base**: Build a searchable archive of AI interactions
- **Learning Tool**: Review past solutions and explanations
- **Workflow Efficiency**: Quick access to previous work
- **No Data Loss**: All conversations can be preserved and recalled

## Technical Considerations

### Timestamp Handling
- Saved timestamps are in HH:MM:SS format only
- On load, timestamps are reconstructed using today's date
- This is acceptable since the date is in the filename

### Error Handling
- Gracefully handles missing `.ti/` directory
- Reports errors if files can't be read
- Validates file format during parsing
- Falls back to current time if timestamp parsing fails

### Performance
- Efficient file scanning with `os.ReadDir`
- Minimal memory footprint (only loads selected chat)
- Fast parsing with string operations

## Future Enhancements

Potential improvements:
- Search/filter chats by content
- Delete chats from the loader dialog
- Export chats to other formats
- Merge multiple chats
- Chat statistics and analytics
- Tag/categorize chats
