# Quick Start: Using /config Command

## What is it?

The `/config` command lets you edit your Terminal Intelligence configuration settings directly from within the application, without needing to open external text editors or restart the app.

## How to use it

### Step 1: Open the config editor

In the AI chat prompt (TI>), type:
```
/config
```

Press Enter.

### Step 2: Navigate to the field you want to edit

Use the arrow keys (↑/↓) or vim-style keys (k/j) to move between fields:
- `agent` - Choose between "ollama" or "gemini"
- `model` - Model name for Ollama (e.g., "llama2", "qwen2.5-coder:3b")
- `gmodel` - Model name for Gemini (e.g., "gemini-2.5-flash-lite")
- `ollama_url` - Ollama server URL (default: "http://localhost:11434")
- `gemini_api` - Your Gemini API key
- `workspace` - Your workspace directory path

### Step 3: Edit the field

Press Enter when you're on the field you want to change.

Use the arrow keys (←/→) to position the cursor where you want to edit.

Use Backspace to delete characters before the cursor, or Delete to remove the character at the cursor.

Type new characters - they'll be inserted at the cursor position.

Press Enter again to save the value.

### Step 4: Save and exit

Press Esc to save all your changes and exit the config editor.

You'll see a confirmation message in the status bar.

## Example: Switching to a different Ollama model

```
1. Type: /config
2. Press: ↓ (to move to "model" field)
3. Press: Enter
4. Type: qwen2.5-coder:3b
5. Press: Enter
6. Press: Esc
```

Done! The app is now using the new model.

## Example: Switching from Ollama to Gemini

```
1. Type: /config
2. Press: Enter (on "agent" field)
3. Type: gemini
4. Press: Enter
5. Press: ↓ ↓ ↓ ↓ (to move to "gemini_api" field)
6. Press: Enter
7. Type: your-api-key-here
8. Press: Enter
9. Press: ↓ ↓ (to move to "gmodel" field - optional)
10. Press: Enter
11. Type: gemini-2.5-flash-lite
12. Press: Enter
13. Press: Esc
```

Done! The app is now using Gemini.

## Tips

- You can press Esc while editing a field to cancel without saving that field
- All changes are saved to `~/.ti/config.json`
- Changes take effect immediately - no restart needed
- If you make a mistake, just run `/config` again and fix it

## Common Issues

**"Config validation error: invalid agent"**
- Make sure you typed "ollama" or "gemini" exactly (lowercase)

**"Config validation error: gemini_api is required"**
- You need to provide a Gemini API key when using the Gemini agent

**Can't find the config file**
- The config file is at `~/.ti/config.json` on Mac/Linux
- On Windows it's at `%USERPROFILE%\.ti\config.json`

## Need more help?

- Type `/help` in the AI prompt to see all available commands
- See [docs/CONFIG_EDITOR.md](docs/CONFIG_EDITOR.md) for detailed documentation
- See [docs/CONFIG_FLOW.md](docs/CONFIG_FLOW.md) for technical details
