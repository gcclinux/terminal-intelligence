# Automatic .gitignore Update for .ti/ Directory

## Overview

When the `.ti/` directory is created for the first time (to store chat session files), the application automatically updates or creates a `.gitignore` file to exclude the `.ti/` directory from version control.

## Why This Matters

Chat session files contain:
- Conversation history with AI
- Potentially sensitive information
- Personal development notes
- Code snippets and experiments

These files are meant for local use only and should not be committed to Git repositories.

## How It Works

### Automatic Detection and Update

When the first chat message is sent:

1. The `.ti/` directory is created (if it doesn't exist)
2. The system detects this is the first creation
3. The `.gitignore` file is checked/updated automatically

### Three Scenarios

#### Scenario 1: No .gitignore File Exists (New Project)

```
Before:
<workspace-root>/
  (no .gitignore)

After:
<workspace-root>/
  .gitignore  (contains: .ti/)
  .ti/
    session_token_chat_20260305_143045.md
```

The system creates a new `.gitignore` file with `.ti/` as the first entry.

#### Scenario 2: .gitignore Exists (Cloned Project)

```
Before:
<workspace-root>/
  .gitignore  (contains: node_modules/, *.log)

After:
<workspace-root>/
  .gitignore  (contains: node_modules/, *.log, .ti/)
  .ti/
    session_token_chat_20260305_143045.md
```

The system appends `.ti/` to the existing `.gitignore` file, preserving all existing entries.

#### Scenario 3: .ti/ Already in .gitignore

```
Before:
<workspace-root>/
  .gitignore  (contains: node_modules/, .ti/, *.log)

After:
<workspace-root>/
  .gitignore  (unchanged)
  .ti/
    session_token_chat_20260305_143045.md
```

The system detects `.ti/` is already present and makes no changes.

## Implementation Details

### Function: `ensureTiDirInGitignore()`

Located in `internal/ui/aichat.go`:

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

### Integration Point

Called from `appendMessageToSessionLog()` when `.ti/` directory is first created:

```go
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
```

## Duplicate Prevention

The system checks for multiple variants to avoid duplicate entries:
- `.ti/` (standard format)
- `.ti` (without trailing slash)
- `/.ti/` (with leading slash)

If any of these variants are found, no changes are made.

## Edge Cases Handled

1. **No Workspace Root**: Gracefully skips if workspace root is not set
2. **File Read Errors**: Silently handles permission or I/O errors
3. **Empty .gitignore**: Creates proper content with newline
4. **Missing Trailing Newline**: Adds newline before appending `.ti/`
5. **Multiple Messages**: Only updates `.gitignore` once (on first `.ti/` creation)
6. **Existing Entries**: Preserves all existing `.gitignore` content

## Testing

Comprehensive tests in `internal/ui/session_log_test.go`:

### Test Coverage

- `TestEnsureTiDirInGitignore_NewGitignore`: Creates new .gitignore
- `TestEnsureTiDirInGitignore_ExistingGitignore`: Appends to existing file
- `TestEnsureTiDirInGitignore_AlreadyPresent`: Avoids duplicates
- `TestEnsureTiDirInGitignore_VariantFormats`: Handles .ti, .ti/, /.ti/
- `TestEnsureTiDirInGitignore_NoWorkspaceRoot`: Handles missing workspace
- `TestEnsureTiDirInGitignore_OnlyOnFirstCreation`: Updates only once

All tests pass successfully.

## Benefits

1. **Automatic**: No manual .gitignore editing required
2. **Safe**: Never commits chat sessions to Git
3. **Smart**: Detects existing entries to avoid duplicates
4. **Preserving**: Keeps all existing .gitignore content
5. **Universal**: Works for both new and cloned projects
6. **One-Time**: Only updates when .ti/ is first created

## User Experience

From the user's perspective:
1. Start using the application
2. Send first message to AI
3. `.ti/` directory is created automatically
4. `.gitignore` is updated automatically
5. Chat sessions are never committed to Git
6. No manual configuration needed

## Git Status Example

After the automatic update:

```bash
$ git status
On branch main
Changes not staged for commit:
  modified:   .gitignore

Untracked files:
  (nothing - .ti/ is ignored)
```

The `.gitignore` file itself will show as modified (if it existed) or untracked (if newly created), but the `.ti/` directory and its contents will never appear in `git status`.

## Best Practices

The implementation follows Git best practices:
- Uses `.ti/` format (directory with trailing slash)
- Adds proper newlines for readability
- Preserves existing file structure
- Avoids duplicate entries
- Handles edge cases gracefully

## Future Considerations

Potential enhancements:
- Add comment explaining why .ti/ is ignored
- Support for custom ignore patterns
- Integration with other VCS systems (SVN, Mercurial)
- User notification when .gitignore is updated
