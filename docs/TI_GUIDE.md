# Terminal Intelligence (TI) — The Complete Guide

## What is TI?

Terminal Intelligence is an AI-powered CLI IDE that lives entirely in your terminal. It combines a code editor, an AI assistant, Git integration, script execution, and project-wide agentic operations into a single split-window interface. Think of it as your coding companion that can read, understand, explain, fix, and generate code — all without leaving the command line.

TI supports three AI backends — Ollama (local), Google Gemini, and AWS Bedrock — so you can choose between running models on your own machine or leveraging cloud intelligence.

---

## Who Is TI For?

- Developers who live in the terminal and want AI assistance without switching to a browser or GUI IDE
- Teams working on remote servers or headless environments where a GUI isn't available
- Anyone who wants a lightweight, keyboard-driven coding workflow with built-in AI

---

## Getting Started

### Install

Download the binary for your platform and make it executable:

```bash
# Linux
chmod +x ti-linux-amd64 && ./ti-linux-amd64

# macOS (Apple Silicon)
chmod +x ti-darwin-arm64 && ./ti-darwin-arm64

# Windows
.\ti-windows-amd64.exe
```

Or build from source:

```bash
git clone https://github.com/user/terminal-intelligence.git
cd terminal-intelligence
go build -o ti cmd/ti/main.go
./ti
```

### First Launch

On first run TI creates `~/.ti/config.json` with defaults and sets up a workspace at `~/ti-workspace`. You can change the workspace anytime with `Ctrl+W`.

### Configure Your AI Provider

Edit `~/.ti/config.json` or type `/config` inside TI to use the interactive editor.

**Ollama (local, no internet needed):**
```json
{
  "agent": "ollama",
  "model": "qwen2.5-coder:3b",
  "ollama_url": "http://localhost:11434",
  "workspace": "~/ti-workspace"
}
```

**Google Gemini:**
```json
{
  "agent": "gemini",
  "gmodel": "gemini-3-flash-preview",
  "gemini_api": "your-api-key",
  "workspace": "~/ti-workspace"
}
```

**AWS Bedrock:**
```json
{
  "agent": "bedrock",
  "bedrock_model": "anthropic.claude-sonnet-4-6",
  "bedrock_api": "ACCESS_KEY_ID:SECRET_ACCESS_KEY",
  "bedrock_region": "us-east-1",
  "workspace": "~/ti-workspace"
}
```

You can also override any setting with command-line flags:
```bash
./ti -agent gemini -workspace /path/to/project
```

---

## The Interface

TI uses a split-window layout:

```
┌──────────────────────────┬──────────────────────────────────┐
│                          │  AI Input (ti> prompt)           │
│   EDITOR                 ├──────────────────────────────────┤
│   (your code)            │  AI Responses                    │
│                          │  (scrollable conversation)       │
├──────────────────────────┴──────────────────────────────────┤
│  Status Bar                                                 │
└─────────────────────────────────────────────────────────────┘
```

Press `Tab` to cycle focus between three areas: Editor → AI Input → AI Response. The active area has a blue border.

---

## AI Chat Commands — When to Use Each

This is the heart of TI. Each command serves a distinct purpose. Here's when and why you'd reach for each one.

### Just Talking: Plain Text or `/ask`

**What it does:** Sends your message to the AI in conversational mode. The AI answers your question but does not touch your code.

**When to use it:**
- You want to understand how something works
- You need an explanation, a concept, or advice
- You're exploring ideas before writing code

**Examples:**
```
What is a goroutine?
How do I handle errors in Python?
/ask explain the difference between let and const in JavaScript
```

**Tip:** Press `Ctrl+Enter` instead of `Enter` to send your message along with the contents of the currently open file. This gives the AI full context about your code.

---

### Fixing Code: `/fix`

**What it does:** Forces agentic mode. The AI reads your currently open file, generates a fix, and applies it directly to the editor.

**When to use it:**
- You have a bug and want the AI to patch it
- You want to refactor, rename, or restructure code in the open file
- You want the AI to add error handling, logging, or other improvements to existing code

**How it works:**
1. Open the file you want to modify
2. Type `/fix <what you want changed>`
3. AI reads the file, generates a patch, applies it
4. Your file is marked modified but not saved — you review first

**Examples:**
```
/fix add error handling to the HTTP request on line 42
/fix refactor this function to use list comprehension
/fix the off-by-one error in the loop
```

**Note:** TI also auto-detects fix intent from keywords like "fix", "change", "update", "modify", "correct" — so you don't always need the `/fix` prefix. But when in doubt, use it explicitly.

---

### Previewing Changes: `/preview`

**What it does:** Shows you what the AI would change without actually applying anything. Works for both single-file and project-wide changes.

**When to use it:**
- You want to see proposed changes before committing to them
- You're about to make a risky or large-scale modification
- You want to understand the scope of a change before applying it

**Examples:**
```
/preview add input validation to all form handlers
/preview rename the Config struct to AppConfig everywhere
```

After reviewing, type `/proceed` to apply the changes, or just move on if you don't like them.

---

### Applying Previewed Changes: `/proceed`

**What it does:** Applies the changes from the last `/preview` run.

**When to use it:** After running `/preview` and deciding the proposed changes look good.

---

### Project-Wide Changes: `/project`

**What it does:** Scans your entire workspace (up to 500 files), uses the AI to identify the most relevant files (up to 20), reads them, generates patches, and writes changes to disk.

**When to use it:**
- You need to make a change that spans multiple files
- You want to add logging, error handling, or a pattern across the codebase
- You're renaming something that appears in many places

**How it works:**
1. Scans workspace, skipping `.git`, `vendor`, `node_modules`, `.ti`, `build`
2. AI ranks files by relevance to your request
3. Reads each file (up to 2000 lines), generates SEARCH/REPLACE patches
4. Applies patches and shows a change report

**Examples:**
```
/project add error handling to all HTTP client calls
/project rename the Config struct to AppConfig everywhere
/project update all log.Printf calls to use structured logging
```

**Best practice:** Run `/preview <request>` first to see what would change, then `/proceed` to apply.

---

### Creating a New Application: `/create`

**What it does:** Generates a new application from a description. The AI scaffolds files, writes boilerplate, and sets up a project structure.

**When to use it:**
- You're starting from scratch and want the AI to bootstrap a project
- You want a quick prototype or skeleton to build on

**Examples:**
```
/create a REST API in Go with health check and user endpoints
/create a Python CLI tool that converts CSV to JSON
/create a bash script that monitors disk usage and sends alerts
```

---

### Generating Documentation: `/doc`

**What it does:** Analyzes your project and generates documentation. Can create user manuals, API references, installation guides, or module-specific docs.

**When to use it:**
- Your project needs documentation and you don't want to write it from scratch
- You want docs that reflect the actual code, not outdated assumptions

**Examples:**
```
/doc create user manual
/doc create API reference
/doc create installation guide
/doc for module internal
```

---

### Refreshing Project Context: `/rescan`

**What it does:** Re-scans your project files to pick up any changes made outside TI (e.g., new files added, dependencies updated).

**When to use it:**
- You've made changes outside TI (pulled from Git, ran a build, etc.)
- The AI seems to be working with stale context

---

### Utility Commands

| Command | What it does |
|---------|-------------|
| `/model` | Shows current AI provider, model name, and config |
| `/config` | Opens the interactive configuration editor |
| `/help` | Shows keyboard shortcuts and command reference |
| `/quit` | Exits TI |

---

## Working with Code Blocks

When the AI generates code in its response, you can interact with those blocks directly:

1. Press `Ctrl+Y` to enter code block selection mode
2. Use `Up/Down` to navigate between blocks
3. Press `Enter` to see options:
   - **[0] Execute CMD** — Run the code block in an inline terminal
   - **[1] / Ctrl+P Insert** — Insert the code into your editor
   - **[2] / Esc Return** — Go back to selection

This is great for quickly trying out AI-generated snippets or inserting them into your file.

---

## Running Scripts

Press `Ctrl+R` to run the currently open file. TI auto-detects the file type:

| Extension | Runs with |
|-----------|-----------|
| `.sh`, `.bash` | `bash` |
| `.ps1` | `powershell` |
| `.py` | `python` |
| `.go` | `go run` |
| `*_test.go` | `go test -v` |

Output appears in the right pane. Press `Ctrl+K` to kill a running process.

If a required runtime (Go, Python) isn't installed, TI offers to install it for you.

---

## Git Integration

Press `Ctrl+G` to open the Git panel. No external Git installation required — TI uses a pure Go implementation.

**Typical workflow:**
1. **Status** — See what's changed
2. **Stage** — Stage all modified files
3. **Commit** — Enter a commit message
4. **Push** — Push to remote

Also supports Clone, Pull, Fetch, and Restore (discard uncommitted changes).

Authentication works with GitHub Personal Access Tokens (recommended), username/password, or auto-detected credentials from `.git/config`.

---

## Keyboard Shortcuts — Quick Reference

### Files
| Shortcut | Action |
|----------|--------|
| `Ctrl+N` | New file |
| `Ctrl+O` | Open file |
| `Ctrl+S` | Save |
| `Ctrl+X` | Close file |
| `Ctrl+W` | Change workspace |
| `Ctrl+R` | Run script |
| `Ctrl+K` | Kill process |
| `Ctrl+B` | Backup picker |
| `Ctrl+Q` | Quit |

### AI
| Shortcut | Action |
|----------|--------|
| `Enter` | Send message |
| `Ctrl+Enter` | Send message with code context |
| `Ctrl+Y` | Code block selection |
| `Ctrl+P` | Insert code block into editor |
| `Ctrl+L` | Load saved chat |
| `Ctrl+T` | Clear chat / new session |

### Navigation
| Shortcut | Action |
|----------|--------|
| `Tab` | Cycle focus areas |
| `↑↓` | Scroll / move cursor |
| `PgUp/PgDn` | Page scroll |
| `Home/End` | Jump to top/bottom |
| `Esc` | Back / cancel |

### Editor
| Shortcut | Action |
|----------|--------|
| `Alt+D, D` or `Alt+L` | Delete line |
| `Alt+D, W` or `Alt+W` | Delete word |
| `Alt+D, 1-9` | Delete N lines |
| `Alt+U` | Undo |
| `Alt+R` | Redo |
| `Alt+H` | Jump to top of file |
| `Alt+G` | Jump to end of file |

### Git
| Shortcut | Action |
|----------|--------|
| `Ctrl+G` | Open Git panel |
| `Ctrl+H` | Toggle help dialog |

---

## Common Workflows

### "I want to understand some code"
```
1. Open the file (Ctrl+O)
2. Tab to AI Input
3. Type your question
4. Press Ctrl+Enter (sends with code context)
5. Read the response
```

### "I need to fix a bug"
```
1. Open the buggy file
2. Tab to AI Input
3. Type: /fix <describe the bug>
4. Review the applied changes
5. Ctrl+S to save if happy, Alt+U to undo if not
6. Ctrl+R to test
```

### "I want to make changes across the project"
```
1. /preview <describe the change>
2. Review the change report
3. /proceed to apply
```

### "I'm starting a new project"
```
1. Ctrl+W to set your workspace
2. /create <describe what you want>
3. Review generated files
4. Start iterating
```

### "I need docs for my project"
```
1. Navigate to your project workspace (Ctrl+W)
2. /doc create user manual
3. Review and edit the generated documentation
```

### "I want to commit and push my work"
```
1. Ctrl+G to open Git panel
2. Status → Stage → Commit (enter message) → Push
3. Esc to close
```

---

## Tips

- Use `Ctrl+Enter` liberally — the AI gives much better answers when it can see your code
- Start risky changes with `/preview` before `/proceed`
- `/ask` is your safe mode — guaranteed no code modifications
- Chat sessions auto-save to `~/.ti/` — load previous ones with `Ctrl+L`
- Keep requests specific: "add error handling to the HTTP call on line 42" beats "improve the code"
- For project-wide ops, scope your request: "update logging in internal/ui" is better than "fix everything"

---

## Tested AI Models

| Provider | Model | Best For |
|----------|-------|----------|
| Ollama | `qwen2.5-coder:3b` | General coding (local) |
| Ollama | `qwen2.5-coder:1.5b` | Fast responses (local) |
| Ollama | `deepseek-coder-v2:16b` | Complex tasks (local) |
| Gemini | `gemini-3-flash-preview` | Fast cloud responses |
| Gemini | `gemini-3.1-pro-preview` | Advanced reasoning |
| Gemini | `gemini-3.1-flash-lite` | Lightweight cloud |
| Bedrock | `anthropic.claude-sonnet-4-6` | Best coding performance |
| Bedrock | `anthropic.claude-haiku-4-6` | Fast and cost-effective |
| Bedrock | `anthropic.claude-opus-4-6` | Highest intelligence |

---

## Troubleshooting

**AI not responding?**
- Ollama: Check it's running (`curl http://localhost:11434/api/tags`)
- Gemini: Verify API key in config
- Bedrock: Check AWS credentials and region

**Can't run a script?**
- Ensure the runtime is installed (`go version`, `python --version`)
- TI will offer to auto-install Go or Python if missing

**Git push rejected?**
- Pull first, resolve conflicts, then push again

**Interface looks broken?**
- Use a terminal that supports UTF-8 and 256 colors
- Minimum 80 columns wide

For detailed troubleshooting, see the [Configuration Guide](CONFIG.md).
