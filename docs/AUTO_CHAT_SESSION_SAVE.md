# Automatic Chat Session Save Implementation

## Overview

The application now automatically saves all AI chat conversations to session files in the `.ti/` directory. This eliminates the need for manual save operations (like Ctrl+A) and ensures no conversation history is lost.

## How It Works

### Automatic Session Creation

1. When the first message is sent in a new chat session, a session file is automatically created
2. The file is named: `session_token_chat_YYYYMMDD_HHMMSS.md`
   - Example: `session_token_chat_20260305_143045.md`
3. The timestamp reflects when the session started

### Real-Time Message Saving

Every message (user and assistant) is immediately appended to the session file as it's sent or received:

- User messages are saved when sent
- Assistant responses are saved when received
- Fix requests are saved with context
- Notifications are saved

### Session Lifecycle

- **New Session**: Created on first message after app start or after clearing chat (Ctrl+T)
- **Active Session**: All subsequent messages append to the same file
- **Session End**: When chat is cleared (Ctrl+T), the session file is finalized and a new one will be created for the next conversation

## Implementation Details

### Core Function: `appendMessageToSessionLog`

Located in `internal/ui/aichat.go`:

```go
func (a *AIChatPane) appendMessageToSessionLog(msg types.ChatMessage) {
	if a.workspaceRoot == "" {
		return
	}

	tiDir := filepath.Join(a.workspaceRoot, ".ti")
	tiDirCreated := false
	if _, err := os.Stat(tiDir); os.IsNotExist(err) {
		os.MkdirAll(tiDir, 0755)
		tiDirCreated = true
	}

	// If we just created .ti/ directory, ensure it's in .gitignore
	if tiDirCreated {
		a.ensureTiDirInGitignore()
	}

	if a.sessionFile == "" {
		a.sessionFile = filepath.Join(tiDir, fmt.Sprintf("session_token_chat_%s.md", time.Now().Format("20060102_150405")))
	}

	f, err := os.OpenFile(a.sessionFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	var content strings.Builder
	content.WriteString(msg.Role)
	content.WriteString(" ")
	content.WriteString(msg.Timestamp.Format("15:04:05"))
	content.WriteString("\n")
	content.WriteString(msg.Content)
	content.WriteString("\n\n")

	f.WriteString(content.String())
}
```

### Automatic .gitignore Management: `ensureTiDirInGitignore`

Also located in `internal/ui/aichat.go`:

```go
func (a *AIChatPane) ensureTiDirInGitignore() {
	if a.workspaceRoot == "" {
		return
	}

	gitignorePath := filepath.Join(a.workspaceRoot, ".gitignore")
	
	// Read existing .gitignore if it exists
	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	contentStr := string(content)
	
	// Check if .ti/ is already in .gitignore
	lines := strings.Split(contentStr, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".ti/" || trimmed == ".ti" || trimmed == "/.ti/" {
			return // Already present
		}
	}

	// Need to add .ti/ to .gitignore
	var newContent string
	if contentStr == "" {
		newContent = ".ti/\n"
	} else {
		if !strings.HasSuffix(contentStr, "\n") {
			newContent = contentStr + "\n.ti/\n"
		} else {
			newContent = contentStr + ".ti/\n"
		}
	}

	os.WriteFile(gitignorePath, []byte(newContent), 0644)
}
```

### Integration Points

The function is called from:

1. `SendMessage()` - When user sends a message
2. `DisplayResponse()` - When AI responds
3. `DisplayNotification()` - When notifications are shown
4. `AddFixRequest()` - When fix requests are added

### File Format

Each message is saved in this format:

```markdown
role timestamp
content

role timestamp
content
```

Example:

```markdown
user 14:30:45
How do I create a web server in Go?

assistant 14:30:52
To create a web server in Go, you can use the net/http package...

user 14:32:10
Can you add routing?

assistant 14:32:18
Sure! Here's how to add routing...
```

## Loading Saved Sessions

Users can reload any saved session using Ctrl+L:

1. Press Ctrl+L to open the chat loader
2. Select from a list of saved sessions (newest first)
3. Press Enter to load the selected session
4. The entire conversation history is restored

The chat loader specifically looks for files matching the pattern: `session_token_chat_*.md`

## Benefits

1. **No Data Loss**: Conversations are saved in real-time
2. **No Manual Action**: Users don't need to remember to save
3. **Session Organization**: Each conversation gets its own file
4. **Easy Recovery**: Load any previous session with Ctrl+L
5. **Searchable**: Files are in markdown format for easy searching
6. **Timestamped**: Both filename and messages include timestamps
7. **Git-Safe**: Automatic `.gitignore` management prevents accidental commits

## Testing

Comprehensive tests are included in `internal/ui/session_log_test.go`:

### Session Logging Tests
- `TestAppendMessageToSessionLog`: Verifies messages are saved correctly
- `TestSessionFileReusedAcrossMessages`: Ensures same file is used for entire session
- `TestSessionFileResetOnClearHistory`: Confirms new session starts after clearing
- `TestNoSessionFileWithoutWorkspaceRoot`: Handles edge case of no workspace

### .gitignore Management Tests
- `TestEnsureTiDirInGitignore_NewGitignore`: Creates new .gitignore with .ti/ entry
- `TestEnsureTiDirInGitignore_ExistingGitignore`: Appends to existing .gitignore
- `TestEnsureTiDirInGitignore_AlreadyPresent`: Avoids duplicate entries
- `TestEnsureTiDirInGitignore_VariantFormats`: Handles .ti, .ti/, /.ti/ variants
- `TestEnsureTiDirInGitignore_NoWorkspaceRoot`: Gracefully handles missing workspace
- `TestEnsureTiDirInGitignore_OnlyOnFirstCreation`: Ensures .gitignore is only updated once

All tests pass successfully.

## Migration from Manual Save

Previously, the documentation mentioned Ctrl+A for manual saving. This has been removed because:

1. Automatic saving is more reliable
2. No user action required
3. Real-time saving prevents data loss
4. Simpler user experience

The documentation has been updated to reflect the automatic saving behavior.

## File Location

All session files are stored in:
```
<workspace-root>/.ti/session_token_chat_*.md
```

The `.ti` directory is automatically created if it doesn't exist.

## Edge Cases Handled

1. **No Workspace Root**: If `workspaceRoot` is empty, no session file is created (graceful degradation)
2. **Directory Creation**: The `.ti` directory is created automatically if needed
3. **File Errors**: File write errors are silently ignored to prevent disrupting the user experience
4. **Session Reset**: Clearing chat (Ctrl+T) resets the session file path, so a new file is created for the next conversation
5. **Duplicate .gitignore Entries**: Checks for `.ti/`, `.ti`, and `/.ti/` variants to avoid duplicates
6. **Existing .gitignore**: Preserves existing content when appending `.ti/`
7. **Missing .gitignore**: Creates new `.gitignore` file if it doesn't exist
8. **Multiple Messages**: `.gitignore` is only updated once when `.ti/` directory is first created

## Future Enhancements

Potential improvements:

1. Session file size limits (split large sessions)
2. Automatic cleanup of old sessions
3. Session search/filter functionality
4. Export sessions to different formats
5. Session tagging or categorization
