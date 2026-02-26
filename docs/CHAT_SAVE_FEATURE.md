# Chat Save Feature

## Overview
The Ctrl+A shortcut saves the entire AI chat history to a markdown file in the `.ti/` folder. The Ctrl+L shortcut allows you to reload previously saved chats back into the AI pane.

## Save Chat (Ctrl+A)

### Usage

1. Have a conversation with the AI assistant
2. Press `Ctrl+A` at any time during or after the conversation
3. The entire chat history will be automatically saved to `.ti/chat-YYYY-MM-DD-HH-MM-SS-query.md`
4. The saved file will automatically open in the editor

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

The filename is generated automatically using:
- Current date: `YYYY-MM-DD`
- First user message timestamp: `HH-MM-SS`
- First few words of the user's initial query (sanitized for filename compatibility)

Example: `chat-2026-02-26-13-32-48-using-Go-language-create-a-program.md`

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
    chat-2026-02-26-13-32-48-query1.md
    chat-2026-02-26-14-15-30-query2.md
    20260219-160311_test.py  (existing backup files)
```

## Benefits

- Complete conversation history preserved
- Timestamped for easy reference
- Searchable markdown format
- Automatic organization in `.ti/` folder
- No manual filename prompts
- Easy to reload and continue conversations
- Instantly viewable in the editor
