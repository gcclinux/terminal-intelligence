# Chat Save Feature

## Overview
Chat conversations are automatically saved to markdown files in the `.ti/` folder as you interact with the AI. The Ctrl+L shortcut allows you to reload previously saved chats back into the AI pane.

## Automatic Chat Saving

### How It Works

1. When you start a conversation with the AI assistant, a new session file is automatically created
2. Every message (both user and assistant) is immediately appended to the session file
3. The session file is named `.ti/session_token_chat_YYYYMMDD_HHMMSS.md` based on when the session started
4. All messages in the current session are saved to the same file
5. When you clear chat history (Ctrl+T), a new session file will be created for the next conversation

### Automatic .gitignore Management

When the `.ti/` directory is created for the first time:
- If `.gitignore` exists: `.ti/` is automatically appended to it
- If `.gitignore` doesn't exist: A new `.gitignore` file is created with `.ti/` entry
- The system checks for existing `.ti/`, `.ti`, or `/.ti/` entries to avoid duplicates
- This ensures your chat sessions are never accidentally committed to version control

### File Format

The saved chat file follows this format:

```markdown
user 13:32:48
using Go language create a program to display cpu, memory, root volume used and capacity

assistant 13:32:59
To create a system monitor in Go, the most reliable and cross-platform way is to use the `gopsutil` library. It is the Go equivalent of Python's `psutil`.

user 13:35:12
Can you add network statistics too?

assistant 13:35:20
Sure! I'll add network statistics to the program...
```

### Filename Convention

The filename is generated automatically when the first message is sent:
- Format: `session_token_chat_YYYYMMDD_HHMMSS.md`
- Date: `YYYYMMDD` (e.g., 20260305)
- Time: `HHMMSS` (e.g., 143045)

Example: `session_token_chat_20260305_143045.md`

### Benefits of Automatic Saving

- No manual action required - conversations are preserved automatically
- Real-time saving - messages are saved immediately as they're sent/received
- No risk of losing conversation history
- Eliminates the need for manual Ctrl+A save operations
- Session-based organization - each conversation session gets its own file

## Load Chat (Ctrl+L)

### Usage

1. Press `Ctrl+L` to open the chat loader dialog
2. A list of all saved chats from the `.ti/` folder will be displayed (newest first)
3. Use `↑` and `↓` arrow keys (or `k` and `j`) to navigate through the list
4. Press `Enter` to load the selected chat into the AI pane
5. Press `Esc` to cancel and close the dialog

### Features

- Loads complete conversation history with timestamps
- Preserves message roles (user/assistant)
- Extracts code blocks from loaded messages
- Automatically scrolls to the bottom of the loaded chat
- Switches focus to the AI pane after loading

### When to Use

- After clearing chat with `Ctrl+T` to start fresh
- When opening TI with an empty chat
- To review or continue previous conversations
- To reference past solutions or discussions

## Directory Structure

All chat files are saved in the `.ti/` directory at the workspace root:
```
<workspace-root>/
  .ti/
    session_token_chat_20260305_143045.md
    session_token_chat_20260305_151230.md
    20260219-160311_test.py  (existing backup files)
  .gitignore  (automatically updated to include .ti/)
```

The `.ti/` directory is automatically added to `.gitignore` to prevent chat sessions from being committed to version control.

## Overall Benefits

- Complete conversation history preserved automatically
- Real-time saving - no data loss
- Timestamped for easy reference
- Searchable markdown format
- Automatic organization in `.ti/` folder
- No manual save operations required
- Easy to reload and continue conversations
- Session-based file organization
- Automatic `.gitignore` management - chat sessions never committed to Git
