# Installation Progress Visibility

Terminal Intelligence provides full visibility into the language runtime installation process through the AI chat panel.

## What You See During Installation

### 1. Initial Prompt
When you save a Go file and Go is not installed, you see:

```
┌────────────────────────────────────────┐
│ ⚠  Go is not installed or not in PATH │
│                                        │
│ Would you like to install Go          │
│ automatically?                         │
│                                        │
│ [Y]es to install / [N]o to cancel     │
└────────────────────────────────────────┘
```

### 2. Installation Start (Immediate)
As soon as you press `Y`, the AI chat panel immediately shows:

```
🚀 Starting Go Installation

This may take a few minutes. Please wait...

Steps:
1. Fetching latest version
2. Detecting system architecture
3. Downloading Go (~140MB)
4. Removing old installation
5. Extracting files (requires sudo)
6. Updating shell configuration
7. Verifying installation

Installation in progress...
```

**Status Bar**: Shows "Installing Go..."

### 3. Installation Progress (During)
While the installation runs in the background:
- The AI chat panel shows the initial message
- The status bar shows "Installing Go..."
- The UI remains responsive (you can switch panes, scroll, etc.)
- On Linux, you'll be prompted for your sudo password in the terminal

### 4. Installation Complete (After)
When installation finishes successfully, the AI chat panel updates with the full detailed output:

```
✓ Go Installation Complete

Starting Go installation for Linux...

1. Fetching latest Go version...
   Latest version: go1.21.5

2. Detected architecture: amd64

3. Download URL: https://go.dev/dl/go1.21.5.linux-amd64.tar.gz

4. Downloading Go...
   Download complete!

5. Removing old Go installation (if exists)...
   No old installation found

6. Extracting Go to /usr/local...
   Extraction complete!

7. Updating PATH in shell configuration...
   Updated .bashrc
   Updated .profile

8. Cleaning up...
   Temporary files removed

9. Verifying installation...
   go version go1.21.5 linux/amd64

✓ Go installation complete!

IMPORTANT: Please restart Terminal Intelligence or run:
  source ~/.bashrc
to update your PATH.
```

**Status Bar**: Shows "Go installed successfully!"

### 5. Installation Failed (If Error)
If installation fails, you see:

```
✗ Go Installation Failed

Starting Go installation for Linux...

1. Fetching latest Go version...
   Latest version: go1.21.5

2. Detected architecture: amd64

3. Download URL: https://go.dev/dl/go1.21.5.linux-amd64.tar.gz

4. Downloading Go...
   [Error output here]

Error: download failed: connection timeout
```

**Status Bar**: Shows "Go installation failed: [error message]"

## Visibility Features

### Real-Time Status
- **Status Bar**: Always shows current operation
- **AI Chat Panel**: Shows installation steps and progress
- **Responsive UI**: Can interact with TI during installation

### Detailed Output
- **Step-by-Step**: Each installation step is logged
- **Version Info**: Shows exactly what version is being installed
- **Architecture**: Confirms correct binary for your system
- **File Locations**: Shows where files are being installed
- **Verification**: Confirms installation succeeded

### Error Reporting
- **Clear Messages**: Errors are shown with context
- **Full Output**: Complete error output for troubleshooting
- **Actionable**: Suggests next steps if installation fails

## Platform-Specific Visibility

### Windows
```
🚀 Starting Go Installation

Installation will:
• Use winget (Windows Package Manager)
• Install latest Go version
• Update system PATH automatically

Installation in progress...

[After completion]
✓ Go Installation Complete

Starting Go installation for Windows...

1. Checking for winget...
   winget is available

2. Installing Go via winget...
   [winget output]

✓ Go installation complete!

IMPORTANT: Please restart Terminal Intelligence to update your PATH.
```

### macOS
```
🚀 Starting Go Installation

Installation will:
• Use Homebrew
• Install latest Go version
• Update PATH automatically

Installation in progress...

[After completion]
✓ Go Installation Complete

Starting Go installation for macOS...

1. Checking for Homebrew...
   Homebrew is available

2. Updating Homebrew...
   Homebrew updated

3. Installing Go via Homebrew...
   [brew output]

✓ Go installation complete!

Go should now be available in your PATH.
```

### Linux
```
🚀 Starting Go Installation

Installation will:
• Download latest Go from golang.org
• Extract to /usr/local/go
• Update shell configuration
• Requires sudo password

Installation in progress...

[You'll be prompted for sudo password in terminal]

[After completion]
✓ Go Installation Complete

Starting Go installation for Linux...

1. Fetching latest Go version...
   Latest version: go1.21.5

2. Detected architecture: amd64

3. Download URL: https://go.dev/dl/go1.21.5.linux-amd64.tar.gz

4. Downloading Go...
   Download complete!

5. Removing old Go installation (if exists)...
   No old installation found

6. Extracting Go to /usr/local...
   Extraction complete!

7. Updating PATH in shell configuration...
   Updated .bashrc
   Updated .profile

8. Cleaning up...
   Temporary files removed

9. Verifying installation...
   go version go1.21.5 linux/amd64

✓ Go installation complete!

IMPORTANT: Please restart Terminal Intelligence or run:
  source ~/.bashrc
to update your PATH.
```

## UI Behavior During Installation

### What You Can Do
- ✓ Switch between editor and AI panes (Tab)
- ✓ Scroll through AI chat history
- ✓ View the installation progress
- ✓ Edit files in the editor
- ✓ Read the status bar

### What You Cannot Do
- ✗ Start another installation (blocked until current completes)
- ✗ Close the installation dialog (must wait or cancel)

### Cancellation
- Installation cannot be cancelled once started
- If you need to stop, you can quit TI (Ctrl+Q)
- Partial installations may need manual cleanup

## Troubleshooting Visibility

### "Installation seems stuck"
**Check**:
- Status bar shows "Installing Go..."
- AI chat shows "Installation in progress..."
- On Linux, check if sudo password prompt is waiting in terminal

**Action**:
- Wait for download to complete (can take 1-2 minutes)
- Check your internet connection
- On Linux, enter sudo password if prompted

### "No output in AI chat"
**Check**:
- Status bar for current operation
- AI chat panel is visible (not scrolled away)

**Action**:
- Switch to AI pane (Tab)
- Scroll to bottom of AI chat
- Wait for installation to complete

### "Installation completed but no message"
**Check**:
- Status bar shows success/failure
- Scroll to bottom of AI chat panel

**Action**:
- Press Tab to switch to AI pane
- Scroll down to see full output
- Check status bar for summary

## Logging

### Where Output Goes
1. **AI Chat Panel**: Full installation output
2. **Status Bar**: Current operation and final result
3. **Terminal**: Sudo password prompts (Linux only)

### What Gets Logged
- Every installation step
- Version information
- Download progress (start/complete)
- File operations
- Configuration updates
- Verification results
- Errors and warnings

### Output Retention
- Installation output remains in AI chat history
- Can scroll back to review
- Persists until you clear chat (Ctrl+T)
- Not saved to disk

## Best Practices

### For Users
1. **Watch the AI Chat**: Primary source of information
2. **Check Status Bar**: Quick status updates
3. **Be Patient**: Downloads can take time
4. **Read Output**: Contains important post-install instructions
5. **Restart TI**: Required after installation to update PATH

### For Troubleshooting
1. **Read Full Output**: Scroll through complete installation log
2. **Note Error Messages**: Copy exact error text
3. **Check Prerequisites**: Ensure sudo/winget/brew available
4. **Verify Network**: Ensure internet connection works
5. **Manual Install**: If auto-install fails, try manual method

## Future Enhancements

Planned improvements for visibility:
- **Real-time streaming**: Show each step as it happens (not just at end)
- **Progress bar**: Visual indicator for download progress
- **Percentage complete**: Show 1/9, 2/9, etc. for steps
- **Estimated time**: Show approximate time remaining
- **Cancellation**: Allow cancelling installation in progress
- **Retry button**: Quick retry if installation fails

## See Also

- [Automatic Language Installation](AUTO_INSTALL.md) - Overview of auto-install
- [Linux Go Installation](LINUX_GO_INSTALL.md) - Linux-specific details
- [Go Language Support](GO_SUPPORT.md) - Using Go in TI

---

[← Back to Auto-Install Documentation](AUTO_INSTALL.md)
