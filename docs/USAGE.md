# Terminal Intelligence (TI) Usage Guide

### Basic Usage

```bash
# Run with config file (if ~/.ti/config.json exists)
./ti-linux-amd64

# Show version
./ti-linux-amd64 -version


## Starting the Application

```bash
./ti-linux-amd6
```

```powershell
.\build\ti-windows-amd6.exe
``` 


## Key Features

### 1. Creating/Opening Files

- Press **Ctrl+O** or **Ctrl+N** to open the file dialog
- Type the filename (e.g., `test.sh`, `script.py`, `notes.md`)
- Press **Enter** to create/open the file
- Press **Esc** to cancel

### 2. Editing Files (Left Pane)

- Use arrow keys to move cursor
- Type to insert text
- Press **Enter** for new line
- Press **Backspace** to delete
- Press **Ctrl+S** to save

### 3. AI Assistant (Right Pane)

The right pane is now split horizontally:

**Top Section (Input):**
- Type your question or prompt after `ti>`
- Press **Enter** to send message to AI
- Press **Backspace** to delete characters

**Bottom Section (Responses):**
- View AI responses here
- Scroll with **Up/Down** arrows
- Use **PgUp/PgDown** for faster scrolling

**AI Commands:**
- `/fix <request>` - Force agentic mode (AI will modify code)
- `/ask <question>` - Force conversational mode (AI won't modify code)
- `/preview <request>` - Preview changes before applying them (dry-run for single-file or project-wide)
- `/project <request>` - Run a project-wide change across all files in the workspace
- `/proceed` - Apply the changes from the last `/preview` dry-run
- `/model` - Display current agent, model, and API key (for Gemini)
- `/help` - Display keyboard shortcuts and agent commands

### 4. Switching Between Areas

- Press **Tab** to cycle through three areas:
  1. **Editor** (left pane) - Edit your code
  2. **AI Input** (right pane, top) - Type messages to AI
  3. **AI Response** (right pane, bottom) - Read AI responses and scroll
- The active area has a blue border
- When in AI Response area, use arrow keys to scroll through messages

### 5. Sending Code to AI

- When in the AI pane, press **Ctrl+Enter** to send your message WITH the current editor content as context
- The AI will see your code and can help you with it

### 6. Interacting with AI Code Blocks

- When the AI generates a code block, you can interact with it directly.
- Press **Ctrl+Y** to enter Code Block Selection Mode.
- Use **Up/Down** arrows to navigate between generated code blocks.
- Press **Enter** on the selected code block to view action options:
  - **[0] Execute CMD**: Execute the code block in an inline terminal. (Use **Up/Down** or **PgUp/PgDn** to scroll output)
  - **[1] / [Ctrl+P] Insert**: Insert the code block directly into the editor pane.
  - **[2] / [Esc] Return**: Return to the code block selection view.

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| **Ctrl+O** | Open file |
| **Ctrl+N** | New file |
| **Ctrl+S** | Save current file |
| **Tab** | Cycle between Editor, AI Input, and AI Response |
| **Ctrl+Enter** | Send AI message with code context |
| **Ctrl+Y** | Enter AI Code Block Selection |
| **Ctrl+Q** | Quit (will prompt if unsaved changes) |
| **Esc** | Cancel dialogs / Return |

## Example Workflow

1. Start the app
2. Press **Ctrl+N** to create a new file
3. Type filename: `hello.sh`
4. Press **Enter**
5. Type your code in the editor
6. Press **Tab** to switch to AI pane
7. Type: `explain this code`
8. Press **Ctrl+Enter** to send with code context
9. View AI response in bottom section
10. Press **Tab** to go back to editor
11. Press **Ctrl+S** to save

## Status Bar

At the bottom of the screen, you'll see:
- Available keyboard shortcuts
- Status messages (file saved, errors, etc.)

## Tips

- The editor shows line numbers on the left
- Unsaved files show a `*` in the title
- AI responses show timestamps
- Messages with code context are marked `[with context]`
- `/project` works best with a focused, specific request — the AI picks up to 20 files to modify per run
- Use `/preview` first on large or risky changes to see what would be affected before committing
