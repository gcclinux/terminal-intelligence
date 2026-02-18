# Help Dialog (Ctrl+H)

The help dialog provides a quick reference for all keyboard shortcuts and agent commands.

## How to Access

Press `Ctrl+H` at any time to open the help dialog.

You can also type `/help` in the AI chat prompt.

## Help Dialog Layout

```
┌────────────────────────────────────────────────────────────────┐
│                    ⌨  Keyboard Shortcuts                       │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│ ── File ──────────────────────────────────────────────────     │
│   Ctrl+O    Open file                                          │
│   Ctrl+N    New file                                           │
│   Ctrl+S    Save file                                          │
│   Ctrl+X    Close file                                         │
│   Ctrl+Q    Quit                                               │
│                                                                │
│ ── AI ─────────────────────────────────────────────────────    │
│   Ctrl+Y    List code blocks                                   │
│   Ctrl+P    Insert selected code into editor                   │
│   Ctrl+A    Insert full AI response into file                  │
│   Ctrl+T    Clear chat / New chat                              │
│                                                                │
│ ── Navigation ─────────────────────────────────────────────    │
│   Tab       Cycle: Editor → AI Input → AI Response            │
│   ↑↓        Scroll line by line                                │
│   PgUp/PgDn Scroll page                                        │
│   Home/End  Jump to top/bottom                                 │
│   Esc       Back                                               │
│                                                                │
│ ── Agent Commands ─────────────────────────────────────────    │
│   /fix       Force agentic mode (AI modifies code)            │
│   /ask       Force conversational mode (no changes)           │
│   /preview   Preview changes before applying                  │
│   /model     Show current agent and model info                │
│   /config    Edit configuration settings                      │
│   /help      Show this help message                           │
│                                                                │
│              Press Esc or Ctrl+H to close                      │
└────────────────────────────────────────────────────────────────┘
```

## Sections

### File
Commands for file management operations.

### AI
Commands for interacting with AI-generated code and responses.

### Navigation
Commands for moving around the interface and scrolling content.

### Agent Commands
Special commands you can type in the AI chat prompt to control the AI's behavior and access features.

## Closing the Help Dialog

- Press `Esc` to close
- Press `Ctrl+H` again to close

## Tips

- The help dialog is always available, even when other dialogs are open
- You can quickly reference shortcuts without leaving your workflow
- The Agent Commands section shows all available slash commands
- Use `/help` in the chat for a text-based version of this information
