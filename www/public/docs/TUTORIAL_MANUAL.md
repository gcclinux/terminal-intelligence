# Terminal Intelligence (TI) - Complete Tutorial Manual

## Table of Contents

1. [Introduction](#introduction)
2. [What is Terminal Intelligence?](#what-is-terminal-intelligence)
3. [Installation](#installation)
4. [Getting Started](#getting-started)
5. [Interface Overview](#interface-overview)
6. [File Operations](#file-operations)
7. [Editor Features](#editor-features)
8. [AI Assistant](#ai-assistant)
9. [Agentic Code Fixing](#agentic-code-fixing)
10. [Project-Wide Operations](#project-wide-operations)
11. [Git Integration](#git-integration)
12. [Running Scripts](#running-scripts)
13. [Language Support](#language-support)
14. [Keyboard Shortcuts Reference](#keyboard-shortcuts-reference)
15. [Configuration](#configuration)
16. [Tips and Best Practices](#tips-and-best-practices)
17. [Troubleshooting](#troubleshooting)

---

## Introduction

Welcome to Terminal Intelligence (TI), a lightweight CLI-based IDE with integrated AI assistance. This manual will guide you through all features and help you become productive quickly.

Terminal Intelligence combines a powerful code editor with AI-powered assistance, all within your terminal. Whether you're writing shell scripts, Python programs, Go applications, or markdown documentation, TI provides an intuitive interface with intelligent code assistance.

## What is Terminal Intelligence?

Terminal Intelligence is a split-window terminal application that provides:

- **Code Editor** (Left Pane): Full-featured text editor with syntax awareness, line numbers, and file management
- **AI Assistant** (Right Pane): Context-aware AI that can answer questions, explain code, and autonomously fix issues
- **Integrated Git**: Built-in Git operations without requiring external Git installation
- **Multi-Language Support**: Run and edit Bash, PowerShell, Python, Go, and Markdown files
- **Agentic Capabilities**: AI can read, analyze, and modify code across your entire project


## Installation

### Pre-built Binaries

Download the appropriate binary for your platform:

**Linux (AMD64)**
```bash
# Download and make executable
chmod +x ti-linux-amd64
./ti-linux-amd64
```

**Linux (ARM64)**
```bash
chmod +x ti-linux-aarch64
./ti-linux-aarch64
```

**macOS (Intel)**
```bash
chmod +x ti-darwin-amd64
./ti-darwin-amd64
```

**macOS (Apple Silicon)**
```bash
chmod +x ti-darwin-arm64
./ti-darwin-arm64
```

**Windows**
```powershell
.\ti-windows-amd64.exe
```

### From Source

```bash
# Clone the repository
git clone https://github.com/user/terminal-intelligence.git
cd terminal-intelligence

# Build
go build -o ti cmd/ti/main.go

# Run
./ti
```

### First Launch

On first launch, TI will:
1. Create a workspace directory (default: `~/ti-workspace`)
2. Create a configuration directory (`~/.ti/`)
3. Check for AI provider availability (Ollama, Gemini, or Bedrock)


## Getting Started

### Quick Start Tutorial

Let's create your first script with TI:

1. **Launch TI**
   ```bash
   ./ti-linux-amd64
   ```

2. **Create a New File**
   - Press `Ctrl+N`
   - Type: `hello.sh`
   - Press `Enter`

3. **Write Your Code**
   ```bash
   #!/bin/bash
   echo "Hello from Terminal Intelligence!"
   ```

4. **Save the File**
   - Press `Ctrl+S`
   - You'll see "File saved" in the status bar

5. **Run Your Script**
   - Press `Ctrl+R`
   - Output appears in the right pane

6. **Ask the AI for Help**
   - Press `Tab` to switch to AI input
   - Type: `explain this code`
   - Press `Ctrl+Enter` to send with code context
   - Read the AI's explanation in the response area

Congratulations! You've just created, saved, and run your first script with AI assistance.


## Interface Overview

### Split-Window Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  Terminal Intelligence - Build #123                             │
├──────────────────────────┬──────────────────────────────────────┤
│                          │                                      │
│   EDITOR PANE (LEFT)     │   AI ASSISTANT PANE (RIGHT)          │
│                          │                                      │
│   - Code editing         │   Top: AI Input (ti> prompt)         │
│   - Line numbers         │   Bottom: AI Responses               │
│   - Syntax awareness     │                                      │
│   - File management      │                                      │
│                          │                                      │
├──────────────────────────┴──────────────────────────────────────┤
│  Status Bar: Shortcuts and Messages                             │
└─────────────────────────────────────────────────────────────────┘
```

### Active Pane Indicator

The active pane is highlighted with a blue border. Press `Tab` to cycle through:
1. **Editor Pane** - Edit your code
2. **AI Input** - Type messages to the AI
3. **AI Response** - Scroll through AI responses

### Status Bar

The bottom status bar shows:
- Current keyboard shortcuts
- File save confirmations
- Error messages
- Operation status


## File Operations

### Creating a New File

**Method 1: New File Prompt**
1. Press `Ctrl+N`
2. Type the filename (e.g., `script.py`, `notes.md`, `main.go`)
3. Press `Enter`

**Method 2: From AI Code Blocks**
1. Ask AI to generate code
2. Press `Ctrl+Y` to select a code block
3. Press `Ctrl+P` to insert into editor
4. Press `Ctrl+S` to save with a filename

### Opening an Existing File

1. Press `Ctrl+O`
2. Navigate with `Up/Down` arrow keys or `j/k`
3. Press `Enter` on a file to open it
4. Use `..` to navigate to parent directory
5. Folders are marked with `/` suffix
6. Press `Esc` to cancel

### Saving Files

**Save Current File**
- Press `Ctrl+S`
- If the file is new, you'll be prompted for a filename

**Auto-Save on Code Insertion**
- When inserting AI-generated code into an open file, it saves automatically

### Closing Files

- Press `Ctrl+X` to close the current file
- If there are unsaved changes, you'll be prompted to save

### Changing Workspace

1. Press `Ctrl+W`
2. Navigate through directories with `Up/Down`
3. Select `[ Select Current Directory ]` to set as workspace
4. Use `.. (Parent Directory)` to navigate up
5. Press `Enter` to confirm
6. Press `Esc` to cancel


## Editor Features

### Basic Editing

**Cursor Movement**
- `↑↓←→` - Move cursor one character/line
- `Home` - Jump to start of line
- `End` - Jump to end of line
- `Alt+H` - Go to top of file
- `Alt+G` - Go to end of file

**Text Input**
- Type to insert text at cursor
- `Enter` - Insert new line
- `Backspace` - Delete character before cursor
- `Delete` - Delete character at cursor

### Advanced Editing

**Delete Operations**
- `Alt+D, D` - Delete current line
- `Alt+L` - Delete current line (single key)
- `Alt+D, W` - Delete word from cursor
- `Alt+W` - Delete word from cursor (single key)
- `Alt+D, 1-9` - Delete N lines from cursor

**Undo/Redo**
- `Alt+U` - Undo last change
- `Alt+R` - Redo last undone change

### Scrolling

**In Editor Pane**
- `↑↓` - Scroll line by line
- `PgUp/PgDn` - Scroll page by page
- `Home/End` - Jump to top/bottom

**In AI Response Pane**
- Same scrolling keys work when AI Response is active
- Use `Tab` to switch to AI Response area first

### File Backup and Restore

**Backup Picker**
- Press `Ctrl+B` to open backup picker
- View previous versions of your files
- Navigate with `Up/Down`
- Press `Enter` to restore a backup
- Press `Esc` to cancel


## AI Assistant

### Conversational Mode

Ask questions and get explanations without modifying code:

**Examples:**
```
What is a closure in Python?
How do I handle errors in Go?
Explain the difference between let and const
What are the best practices for shell scripting?
```

**With Code Context:**
1. Open a file in the editor
2. Press `Tab` to switch to AI input
3. Type your question
4. Press `Ctrl+Enter` to send with code context
5. The AI sees your code and provides context-aware answers

### AI Commands

**Force Conversational Mode**
```
/ask <your question>
```
Guarantees the AI won't modify your code, only provide guidance.

**Display Model Information**
```
/model
```
Shows current AI provider, model, and configuration.

**Edit Configuration**
```
/config
```
Opens interactive configuration editor.

**Show Help**
```
/help
```
Displays keyboard shortcuts and commands (same as `Ctrl+H`).

**Clear Chat History**
- Press `Ctrl+T` to start a new chat session
- Previous chat is automatically saved to `.ti/` folder

### Chat History Management

**Auto-Save**
- Every chat session is automatically saved
- Saved to `.ti/session_token_chat_YYYYMMDD_HHMMSS.md`

**Load Previous Chat**
1. Press `Ctrl+L`
2. Navigate with `Up/Down`
3. Press `Enter` to load a chat
4. Press `Esc` to cancel


### Working with AI Code Blocks

When the AI generates code, you can interact with it directly:

**Enter Code Block Selection Mode**
1. Press `Ctrl+Y`
2. Use `Up/Down` to navigate between code blocks
3. Press `Enter` to view action options

**Code Block Actions**
- **[0] Execute CMD** - Run the code in an inline terminal
  - View output in the terminal pane
  - Use `Up/Down` or `PgUp/PgDn` to scroll output
  - Press `Ctrl+K` to kill running process
- **[1] / Ctrl+P Insert** - Insert code into editor
  - If file is open: appends to current file
  - If no file: loads as unsaved buffer
- **[2] / Esc Return** - Return to code block selection

**Example Workflow:**
```
You: "write a Python script to list files"
AI: [generates code block]
You: Press Ctrl+Y → Select code → Press Enter → Choose Insert
Result: Code appears in editor, ready to save
```


## Agentic Code Fixing

The AI can autonomously read, analyze, and fix code in your open files.

### How It Works

**Automatic Detection**
The AI detects when you want code modifications based on keywords:
- "fix", "change", "update", "modify", "correct", "refactor"

**Process:**
1. AI reads your currently open file (including unsaved changes)
2. Analyzes your request with code context
3. Generates a precise fix
4. Applies changes directly to the editor
5. Notifies you of modifications made

**Your file is marked as modified but NOT automatically saved** - you maintain full control.

### Agentic vs Conversational

**Agentic Mode** (AI modifies code)
```
fix the off-by-one error in the loop
update the function to handle empty strings
refactor this to use list comprehension
```

**Conversational Mode** (AI provides guidance)
```
how does this function work?
what's wrong with this code?
explain the algorithm
```

### Explicit Commands

**Force Agentic Mode**
```
/fix <your request>
```
Example: `/fix add error handling to this function`

**Preview Changes First**
```
/preview <your request>
```
Shows what would change without applying it.

**Apply Previewed Changes**
```
/proceed
```
Applies the changes from the last preview.


### Best Practices for Agentic Fixing

**1. Be Specific**
- ✅ Good: "fix the off-by-one error in the loop on line 15"
- ❌ Less clear: "fix the bug"

**2. One File at a Time**
- Open the file you want to modify before requesting fixes
- For multi-file changes, use `/project` command

**3. Review Before Saving**
- Always review AI-generated changes
- Use `Alt+U` to undo if needed
- Press `Ctrl+S` only when satisfied

**4. Test Your Code**
- After applying fixes, test with `Ctrl+R`
- Verify the changes work as expected

**5. Use Preview for Complex Changes**
- `/preview` shows changes without applying them
- Safer for large or risky modifications

### Example Workflows

**Fix a Bug**
```
1. Open file with bug
2. Type: "fix the null pointer exception on line 42"
3. AI applies fix automatically
4. Review changes
5. Press Ctrl+S to save
6. Press Ctrl+R to test
```

**Refactor Code**
```
1. Open file to refactor
2. Type: "/preview refactor this function to be more readable"
3. Review proposed changes
4. Type: "/proceed" to apply
5. Save and test
```


## Project-Wide Operations

The AI can operate across your entire project, finding and modifying multiple files at once.

### How It Works

**Three-Phase Pipeline:**

1. **Scanning** - Recursively walks workspace directory
   - Collects all text files (up to 500)
   - Skips: `.git`, `vendor`, `node_modules`, `.ti`, `build`

2. **Ranking** - AI analyzes file list
   - Identifies up to 20 most relevant files
   - Based on your request and file names/paths

3. **Modifying** - Reads and patches files
   - Reads each file (up to 2000 lines)
   - Generates SEARCH/REPLACE patches
   - Validates and applies changes
   - Writes modified files to disk

### Commands

**Run Project-Wide Change**
```
/project <request>
```

**Preview Without Writing**
```
/preview /project <request>
```

**Apply Last Preview**
```
/proceed
```

### Examples

**Add Error Handling**
```
/project add error handling to all HTTP client calls
```

**Rename Across Project**
```
/project rename the Config struct to AppConfig everywhere
```

**Update Logging**
```
/preview /project update all log.Printf calls to use structured logging
```

**Generate Documentation**
```
/project /doc create user manual
/project /doc create API reference
/project /doc create installation guide
/project /doc for module internal
```


### Change Report

After each operation, you'll see a detailed report:

```
Project-wide operation complete

Files scanned: 42
Files modified: 3
  - internal/ui/aichat.go (+5 -2)
  - internal/config/config.go (+1 -0)
  - README.md (+3 -1)

Patch failures: 1
  - internal/executor/executor.go: search text not found in file content
```

**Report Sections:**
- **Files scanned** - Total files examined
- **Files modified** - Successfully changed files with line counts
- **Patch failures** - Files where changes couldn't be applied
- **Hallucinated paths** - Non-existent files the AI suggested (ignored)

### Safety Features

**Workspace Boundaries**
- Only files inside workspace root are accessed
- Symlinks outside workspace are rejected
- Prevents accidental system file modifications

**Preview Mode**
- `/preview /project` never writes to disk
- Review all proposed changes first
- Use `/proceed` to apply when ready

**Supported File Types**
- `.go`, `.md`, `.sh`, `.bash`, `.ps1`, `.py`

**Soon tobe supported**
- `.ts`, `.js`, `.json`, `.yaml`, `.yml`
- `.toml`, `.txt`, `.html`, `.css`

### Best Practices

**1. Start with Preview**
```
/preview /project <request>
```
Always preview large or risky changes first.

**2. Be Specific**
- ✅ "add error handling to all HTTP client calls"
- ❌ "improve the code"

**3. Scope Your Request**
- Focus on specific modules or patterns
- Example: "update logging in internal/ui package"

**4. Review the Report**
- Check which files were modified
- Investigate patch failures
- Open modified files to verify changes


## Git Integration

TI includes a built-in Git client - no external Git installation required.

### Opening the Git Panel

Press `Ctrl+G` to open the Git Operations panel.

**Panel Features:**
- Three input fields (URL, username, password/token)
- Eight operation buttons
- Real-time status messages
- Automatic credential detection

### Git Operations

**Remote Operations**
- **Clone** - Clone a repository to your workspace
- **Pull** - Fetch and merge changes from remote
- **Fetch** - Download changes without merging

**Local to Remote Workflow**
- **Stage** - Stage all modified and untracked files
- **Commit** - Create a commit with staged changes
- **Push** - Push local commits to remote

**Info and Undo**
- **Status** - View repository status
- **Restore** - Discard uncommitted changes

### Git Workflow Example

**1. Check Status**
```
Press Ctrl+G
Navigate to "Status" button
Press Enter
View modified, staged, and untracked files
```

**2. Stage Changes**
```
Navigate to "Stage" button
Press Enter
All changes are staged
```

**3. Commit**
```
Navigate to "Commit" button
Press Enter
Type commit message in input field
Press Enter to commit
```

**4. Push**
```
Navigate to "Push" button
Press Enter
Changes pushed to remote
```


### Authentication

**GitHub Personal Access Token (Recommended)**
1. Generate token at: https://github.com/settings/tokens
2. In Git panel:
   - Username: Your GitHub username
   - Password: Token (starts with `ghp_...`)
3. Credentials are stored in `.git/config`

**Username/Password**
- Works with most Git hosting services
- Less secure than tokens

**Credential Auto-Detection**
- Opening panel in existing repo loads credentials automatically
- Stored credentials used for subsequent operations

### Cloning a Repository

**1. Open Git Panel**
```
Press Ctrl+G
```

**2. Enter Repository URL**
```
Navigate to URL field
Type: https://github.com/user/repo.git
Press Tab to move to next field
```

**3. Enter Credentials (if private)**
```
Username: your-username
Password: your-token-or-password
```

**4. Clone**
```
Navigate to "Clone" button
Press Enter
Wait for completion
Workspace automatically changes to cloned directory
```

### Commit Message Input

**When Commit button is selected:**
1. Press `Enter` to show commit message field
2. Type your message (cannot be empty)
3. Press `Enter` to create commit
4. Input field disappears after success

### Navigation in Git Panel

- `Up/Down` or `j/k` - Navigate options
- `Tab` - Move between input fields
- `Enter` - Select operation or submit input
- `Esc` - Close Git panel


## Running Scripts

### Execute Current File

Press `Ctrl+R` to run the currently open file.

**Auto-Detection:**
TI automatically detects file type and runs appropriately:

**Bash Scripts** (`.sh`, `.bash`)
```bash
#!/bin/bash
echo "Hello World"
```
Runs with: `bash filename.sh`

**PowerShell Scripts** (`.ps1`)
```powershell
Write-Host "Hello World"
```
Runs with: `powershell -File filename.ps1`

**Python Scripts** (`.py`)
```python
print("Hello World")
```
Runs with: `python filename.py`

**Go Programs** (`.go`)
```go
package main
import "fmt"
func main() {
    fmt.Println("Hello World")
}
```
Runs with: `go run filename.go`

**Go Tests** (`*_test.go`)
```go
func TestExample(t *testing.T) {
    // test code
}
```
Runs with: `go test -v`

### Viewing Output

**Output Location:**
- Execution output appears in the right pane (AI Response area)
- Replaces current AI conversation temporarily
- Scroll with `Up/Down` or `PgUp/PgDn`

**Killing Running Process:**
- Press `Ctrl+K` to terminate running process
- Only works when in terminal output mode

### Example: Running a Python Script

```
1. Create file: Ctrl+N → "test.py"
2. Write code:
   print("Hello from TI!")
   for i in range(5):
       print(f"Count: {i}")
3. Save: Ctrl+S
4. Run: Ctrl+R
5. View output in right pane
```


## Language Support

### Supported Languages

TI supports editing and running:
- **Shell/Bash** (`.sh`, `.bash`)
- **PowerShell** (`.ps1`)
- **Python** (`.py`)
- **Go** (`.go`)
- **Markdown** (`.md`)

### Auto-Install Detection

When you create or open a file, TI checks if the required runtime is installed.

**If Missing:**
1. TI detects the missing language
2. Shows installation prompt
3. You can choose to install automatically

**Supported Auto-Install:**
- **Go** - Downloads and installs latest version
- **Python** - Downloads and installs latest version

### Go Development

**Running Go Programs**
```go
// main.go
package main

import "fmt"

func main() {
    fmt.Println("Hello, Go!")
}
```
Press `Ctrl+R` → Runs with `go run main.go`

**Running Go Tests**
```go
// main_test.go
package main

import "testing"

func TestExample(t *testing.T) {
    result := 2 + 2
    if result != 4 {
        t.Errorf("Expected 4, got %d", result)
    }
}
```
Press `Ctrl+R` → Runs with `go test -v`

### Python Development

**Running Python Scripts**
```python
# script.py
def greet(name):
    return f"Hello, {name}!"

if __name__ == "__main__":
    print(greet("World"))
```
Press `Ctrl+R` → Runs with `python script.py`


### Shell Scripting

**Bash Scripts**
```bash
#!/bin/bash
# script.sh

echo "Starting backup..."
tar -czf backup.tar.gz /path/to/files
echo "Backup complete!"
```
Press `Ctrl+R` → Runs with `bash script.sh`

**PowerShell Scripts**
```powershell
# script.ps1

Write-Host "Starting process..."
Get-Process | Where-Object {$_.CPU -gt 100}
Write-Host "Process complete!"
```
Press `Ctrl+R` → Runs with `powershell -File script.ps1`

### Markdown Editing

**Markdown Files**
```markdown
# My Document

This is a **markdown** file with:
- Lists
- **Bold** and *italic*
- Code blocks
```

Markdown files can be edited but not "run" with `Ctrl+R`.

### Language Installation

**Manual Installation Check**
```bash
# Check if Go is installed
go version

# Check if Python is installed
python --version
python3 --version
```

**Auto-Install Process (Go Example)**
1. Create `test.go` file
2. TI detects Go is not installed
3. Shows prompt: "Go is not installed. Install now?"
4. Select "Yes"
5. TI downloads and installs Go
6. Installation output shown in AI pane
7. Ready to use immediately


## Keyboard Shortcuts Reference

### File Operations

| Shortcut | Action |
|----------|--------|
| `Ctrl+W` | Change Workspace / Open Folder |
| `Ctrl+O` | Open file |
| `Ctrl+N` | New file |
| `Ctrl+S` | Save file |
| `Ctrl+X` | Close file |
| `Ctrl+R` | Run current script |
| `Ctrl+K` | Kill running process (in terminal mode) |
| `Ctrl+B` | Backup Picker (Restore previous versions) |
| `Ctrl+Q` | Quit |

### AI Operations

| Shortcut | Action |
|----------|--------|
| `Ctrl+Y` | List code blocks (Execute/Insert/Return) |
| `Ctrl+P` | Insert selected code into editor |
| `Ctrl+L` | Load saved chat from .ti/ folder |
| `Ctrl+T` | Clear chat / New chat |
| `Ctrl+Enter` | Send message with editor context |

### Navigation

| Shortcut | Action |
|----------|--------|
| `Tab` | Cycle: Editor → AI Input → AI Response |
| `↑↓` | Scroll line by line |
| `PgUp/PgDn` | Scroll page |
| `Home/End` | Jump to top/bottom |
| `Esc` | Back / Cancel |

### Editor - Delete Operations

| Shortcut | Action |
|----------|--------|
| `Alt+D, D` | Delete current line |
| `Alt+L` | Delete current line (single key) |
| `Alt+D, W` | Delete word from cursor |
| `Alt+W` | Delete word from cursor (single key) |
| `Alt+D, 1-9` | Delete N lines from cursor |

### Editor - Undo/Redo

| Shortcut | Action |
|----------|--------|
| `Alt+U` | Undo last change |
| `Alt+R` | Redo last undone change |


### Editor - Navigation

| Shortcut | Action |
|----------|--------|
| `Alt+G` | Go to end of file |
| `Alt+H` | Go to top of file |
| `↑↓←→` | Move cursor |
| `Home/End` | Jump to line start/end |

### Editor - Basic Editing

| Shortcut | Action |
|----------|--------|
| `Enter` | Insert new line |
| `Backspace` | Delete char before cursor |
| `Delete` | Delete char at cursor |

### Git Operations

| Shortcut | Action |
|----------|--------|
| `Ctrl+G` | Open Git Panel |
| `↑↓` | Navigate options |
| `Enter` | Select operation |
| `Esc` | Close Git Panel |

### Help

| Shortcut | Action |
|----------|--------|
| `Ctrl+H` | Toggle help dialog |
| `/help` | Show help in AI pane |

### Agent Commands

| Command | Description |
|---------|-------------|
| `/fix <request>` | Force agentic mode (AI modifies code) |
| `/ask <question>` | Force conversational mode (no changes) |
| `/preview <request>` | Preview changes before applying |
| `/project <request>` | Project-wide change across all files |
| `/preview /project` | Dry-run project-wide change (no writes) |
| `/proceed` | Apply changes from last dry-run |
| `/model` | Show current agent and model info |
| `/config` | Edit configuration settings |
| `/help` | Show help message |
| `/quit` | Quit the program |


## Configuration

### Configuration File Location

**User Config:**
- Linux/macOS: `~/.ti/config.json`
- Windows: `%USERPROFILE%\.ti\config.json`

### Interactive Configuration

**Edit Config in TI:**
1. Type `/config` in AI input
2. Navigate through configuration fields
3. Edit values as needed
4. Save changes

**Configuration Fields:**
- `agent` - AI provider (ollama, gemini, bedrock)
- `model` - Default model name
- `gmodel` - Gemini-specific model
- `bedrock_model` - Bedrock-specific model
- `ollama_url` - Ollama service URL
- `gemini_api` - Gemini API key
- `bedrock_api` - Bedrock API key
- `bedrock_region` - AWS region for Bedrock
- `workspace` - Default workspace directory

### AI Provider Configuration

**Ollama (Local)**
```json
{
  "agent": "ollama",
  "model": "qwen2.5-coder:3b",
  "ollama_url": "http://localhost:11434",
  "workspace": "~/ti-workspace"
}
```

**Gemini (Google)**
```json
{
  "agent": "gemini",
  "gmodel": "gemini-3-flash-preview",
  "gemini_api": "your-api-key-here",
  "workspace": "~/ti-workspace"
}
```

**AWS Bedrock**
```json
{
  "agent": "bedrock",
  "bedrock_model": "anthropic.claude-sonnet-4-6",
  "bedrock_api": "your-aws-access-key",
  "bedrock_region": "us-east-1",
  "workspace": "~/ti-workspace"
}
```


### Tested AI Models

**Ollama Models (Local)**
- `qwen2.5-coder:3b` - Recommended for coding
- `qwen2.5-coder:1.5b` - Lightweight
- `deepseek-coder-v2:16b` - Advanced capabilities

**Gemini Models (Google)**
- `gemini-3-flash-preview` - Fast responses
- `gemini-3.1-pro-preview` - Advanced reasoning
- `gemini-3.1-flash-lite` - Lightweight

**AWS Bedrock Models**
- `anthropic.claude-sonnet-4-6` - Best coding performance
- `anthropic.claude-haiku-4-6` - Fast and cost-effective
- `anthropic.claude-opus-4-6` - Highest intelligence

### Command-Line Flags

Override config file settings:

```bash
# Show version
./ti -version

# Specify config file
./ti -config /path/to/config.json

# Override provider
./ti -agent gemini

# Override model
./ti -model qwen2.5-coder:3b

# Override workspace
./ti -workspace /path/to/workspace
```

### Configuration Priority

1. Command-line flags (highest priority)
2. Configuration file
3. Built-in defaults (lowest priority)

### Default Configuration

If no config file exists:
- **Workspace**: `~/ti-workspace`
- **Provider**: `ollama`
- **Ollama URL**: `http://localhost:11434`
- **Model**: `llama2`


## Tips and Best Practices

### General Tips

**1. Use Tab Efficiently**
- `Tab` cycles through Editor → AI Input → AI Response
- Learn this flow to navigate quickly

**2. Save Often**
- Press `Ctrl+S` frequently
- TI doesn't auto-save (except when inserting AI code into open files)

**3. Use Chat History**
- Press `Ctrl+L` to load previous conversations
- All chats are auto-saved to `.ti/` folder

**4. Leverage Code Context**
- Use `Ctrl+Enter` to send messages with code context
- AI provides better answers when it sees your code

**5. Preview Before Applying**
- Use `/preview` for single-file changes
- Use `/preview /project` for multi-file changes
- Review before using `/proceed`

### AI Interaction Tips

**1. Be Specific in Requests**
```
✅ "add error handling to the HTTP request on line 42"
❌ "make it better"
```

**2. Use Explicit Commands When Needed**
```
/ask - When you want explanations only
/fix - When you want code modifications
/preview - When you want to see changes first
```

**3. Break Down Complex Tasks**
```
Instead of: "refactor the entire application"
Try: "refactor the authentication module to use JWT"
Then: "update the user service to use the new auth"
```

**4. Provide Context**
```
"This Python script connects to a database. Add connection pooling."
Better than: "Add connection pooling"
```


### Project-Wide Operation Tips

**1. Start Small**
```
Test with /preview first
Apply to a subset of files
Gradually expand scope
```

**2. Use Specific Patterns**
```
✅ "/project add logging to all error handlers in internal/ui"
❌ "/project improve everything"
```

**3. Review the Change Report**
```
Check files modified
Investigate patch failures
Open files to verify changes
```

**4. Understand Limitations**
```
Max 500 files scanned
Max 20 files modified per run
Max 2000 lines per file
```

### Editor Tips

**1. Use Undo/Redo**
- `Alt+U` to undo mistakes
- `Alt+R` to redo
- Multiple levels of undo available

**2. Quick Line Deletion**
- `Alt+L` for single line
- `Alt+D, 5` to delete 5 lines

**3. Navigate Large Files**
- `Alt+H` to jump to top
- `Alt+G` to jump to bottom
- `PgUp/PgDn` for page scrolling

**4. Use Backup Picker**
- `Ctrl+B` to restore previous versions
- Useful if you accidentally delete content

### Git Workflow Tips

**1. Check Status First**
```
Ctrl+G → Status → Enter
See what changed before committing
```

**2. Write Good Commit Messages**
```
✅ "Add error handling to HTTP client"
❌ "fix"
```

**3. Pull Before Push**
```
Pull → Review changes → Stage → Commit → Push
Avoids merge conflicts
```


## Troubleshooting

### AI Connection Issues

**Problem: "AI service unavailable"**

**For Ollama:**
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Start Ollama
ollama serve

# Pull a model
ollama pull qwen2.5-coder:3b
```

**For Gemini:**
```
1. Check API key in config
2. Verify key at: https://makersuite.google.com/app/apikey
3. Ensure internet connection
4. Check API quota limits
```

**For Bedrock:**
```
1. Verify AWS credentials
2. Check region configuration
3. Ensure Bedrock access is enabled
4. Verify model availability in region
```

### File Operation Issues

**Problem: "Cannot open file"**
```
1. Check file permissions
2. Verify file path is correct
3. Ensure file exists
4. Try absolute path instead of relative
```

**Problem: "Cannot save file"**
```
1. Check write permissions
2. Verify disk space
3. Ensure directory exists
4. Check if file is locked by another process
```

### Script Execution Issues

**Problem: "Command not found" when running script**

**For Python:**
```bash
# Check Python installation
python --version
python3 --version

# Install Python if missing
# TI will prompt for auto-install
```

**For Go:**
```bash
# Check Go installation
go version

# Install Go if missing
# TI will prompt for auto-install
```

