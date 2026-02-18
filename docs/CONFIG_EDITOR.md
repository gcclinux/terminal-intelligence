# Configuration Editor

The `/config` command provides an interactive way to edit your Terminal Intelligence configuration without leaving the application or using external text editors.

## Usage

1. In the AI chat prompt, type `/config` and press Enter
2. The configuration editor will open in the AI response panel
3. Navigate through configuration fields using arrow keys or vim-style keys (k/j)
4. Press Enter to edit a field
5. Type the new value and press Enter to save
6. Press Esc to exit config mode and save all changes

## Configuration Fields

The following fields can be edited:

- **agent**: AI provider (`ollama` or `gemini`)
- **model**: Model name for Ollama (e.g., `llama2`, `qwen2.5-coder:3b`)
- **gmodel**: Model name for Gemini (e.g., `gemini-2.5-flash-lite`, `gemini-3-pro-preview`)
- **ollama_url**: Ollama server URL (default: `http://localhost:11434`)
- **gemini_api**: Gemini API key (required when using Gemini)
- **workspace**: Workspace directory path (absolute path)

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| ↑/↓ or K/J | Navigate between fields |
| Enter | Edit selected field / Save edited value |
| ←/→ | Move cursor left/right within field (when editing) |
| Home | Move cursor to start of field (when editing) |
| End | Move cursor to end of field (when editing) |
| Backspace | Delete character before cursor (when editing) |
| Delete | Delete character at cursor (when editing) |
| Esc | Cancel edit / Save all changes and exit config mode |

## Features

### Real-time Validation

The configuration is validated before saving. If you enter invalid values (e.g., an unsupported agent name), you'll see an error message and the changes won't be saved.

### Immediate Application

Changes are applied immediately after saving:
- The AI client is reinitialized with new settings
- The model is updated
- The workspace directory is changed
- No application restart required

### Persistent Storage

Configuration changes are saved to `~/.ti/config.json` (or `%USERPROFILE%\.ti\config.json` on Windows) and will persist across application restarts.

## Example Workflow

### Switching from Ollama to Gemini

1. Type `/config` in the AI prompt
2. Navigate to the `agent` field
3. Press Enter to edit
4. Type `gemini` and press Enter
5. Navigate to the `gemini_api` field
6. Press Enter to edit
7. Paste your Gemini API key and press Enter
8. Navigate to the `gmodel` field
9. Press Enter to edit
10. Type `gemini-2.5-flash-lite` and press Enter
11. Press Esc to save and exit

The application will immediately switch to using Gemini with your specified model.

### Changing the Workspace Directory

1. Type `/config` in the AI prompt
2. Navigate to the `workspace` field
3. Press Enter to edit
4. Type the new absolute path (e.g., `/home/user/my-projects`)
5. Press Enter to save
6. Press Esc to exit

The file manager will now use the new workspace directory for all file operations.

## Tips

- You can press Esc while editing a field to cancel the edit without saving
- The config editor shows all fields even if they're empty (useful for seeing what's available)
- Changes to the `agent` field require appropriate values in `model` (for Ollama) or `gmodel` (for Gemini)
- The `workspace` directory must exist or be creatable by the application

## Troubleshooting

### "Config validation error"

This means one of your values is invalid. Common issues:
- `agent` must be either `ollama` or `gemini`
- `gemini_api` is required when `agent` is `gemini`
- Paths should be absolute (start with `/` on Unix or drive letter on Windows)

### "Error saving config"

This usually means:
- Permission issues with the config file
- The `~/.ti` directory doesn't exist (it should be created automatically)
- Disk space issues

### Changes not taking effect

If you edit the config but don't see changes:
- Make sure you pressed Esc to save and exit config mode
- Check the status bar for error messages
- Verify the config file was updated: `cat ~/.ti/config.json`

## See Also

- [Configuration Documentation](../README.md#configuration)
- [Usage Guide](USAGE.md)
- [Debugging Guide](DEBUGGING.md)
