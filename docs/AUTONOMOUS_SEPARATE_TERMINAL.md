# Autonomous Separate Terminal Window Feature

## Feature Overview

When the autonomous creation feature (`/create` command) completes and builds a web server application, it now launches the server in a separate terminal window. This provides a professional development experience with visible server logs and easy process management.

## Why Separate Terminal?

### Benefits

1. **Visible Logs**: User can see server output, errors, and requests in real-time
2. **Easy to Stop**: Just close the terminal window or press Ctrl+C
3. **Independent Lifecycle**: Server doesn't depend on TI staying open
4. **Professional Experience**: Matches standard development workflows
5. **Process Isolation**: Server runs in its own process space
6. **Clear Status**: Terminal window title shows what's running

### Previous Approach Issues

Running in background had problems:
- No visible logs
- Hard to know if server is actually running
- Difficult to stop (need to find process ID)
- Server dies when TI closes
- No way to see errors or debug issues

## Implementation

### Windows

Uses PowerShell's `Start-Process` command:

```go
if runtime.GOOS == "windows" {
    // For Go binaries
    runCmd = exec.Command("powershell", "-Command", 
        fmt.Sprintf("Start-Process -FilePath '%s' -WorkingDirectory '%s'", 
            binaryPath, c.ProjectDir))
    
    // For Python scripts
    runCmd = exec.Command("powershell", "-Command", 
        fmt.Sprintf("Start-Process -FilePath 'python' -ArgumentList '%s' -WorkingDirectory '%s'", 
            mainFile, c.ProjectDir))
}
```

This opens a new PowerShell window with the server running.

### Linux

Tries multiple terminal emulators in order:

```go
// Try gnome-terminal first
if _, err := exec.LookPath("gnome-terminal"); err == nil {
    runCmd = exec.Command("gnome-terminal", "--", binaryPath)
    runCmd.Dir = c.ProjectDir
}
// Fallback to xterm
else if _, err := exec.LookPath("xterm"); err == nil {
    runCmd = exec.Command("xterm", "-e", binaryPath)
    runCmd.Dir = c.ProjectDir
}
// Last resort: background process
else {
    runCmd = exec.Command(binaryPath)
    runCmd.Dir = c.ProjectDir
}
```

### macOS

Uses Terminal.app:

```go
if runtime.GOOS == "darwin" {
    // For Go binaries
    runCmd = exec.Command("open", "-a", "Terminal", binaryPath)
    
    // For Python scripts
    script := fmt.Sprintf("cd '%s' && python %s", c.ProjectDir, mainFile)
    runCmd = exec.Command("osascript", "-e", 
        fmt.Sprintf("tell application \"Terminal\" to do script \"%s\"", script))
}
```

### Fallback Strategy

If terminal window fails to open:

1. Show warning message
2. Try to start in background
3. If that fails, show manual start instructions

```go
if err := runCmd.Run(); err != nil {
    result.WriteString("Warning: Could not open terminal window\n")
    result.WriteString("Trying to start in background...\n")
    
    bgCmd := exec.Command(binaryPath)
    bgCmd.Dir = c.ProjectDir
    if err := bgCmd.Start(); err != nil {
        result.WriteString("Error: Could not start server\n")
        result.WriteString("To start manually: cd project && ./binary\n")
    }
}
```

## User Experience

### New Output

```
ai-assist 2026-03-12 12:32:49
Building Go application...
Build successful! Binary: autonomous-app.exe

Web server detected (port 8080)
🌐 Application URL: http://localhost:8080

Starting server in new terminal window...
✓ Server is now running in a new terminal window!

🌐 Application URL: http://localhost:8080
   Click the link above to open in your browser

Note: Check the new terminal window for server logs.
      Close the terminal window to stop the server.

App Creation complete!
```

### What Happens

1. TI builds the application
2. A new terminal window pops up
3. Server starts in that window
4. User sees server logs in the terminal
5. User clicks the link in TI to open browser
6. Application loads successfully
7. User can see requests in the terminal
8. To stop: close terminal or Ctrl+C

### Terminal Window Content

The new terminal shows:
```
C:\Users\user\Programming\autonomous-app> .\autonomous-app.exe
Server starting on port 8080...
Server is ready!
Listening on http://localhost:8080
```

User can see:
- Startup messages
- HTTP requests
- Errors and warnings
- Debug output
- Performance metrics

## Platform-Specific Behavior

### Windows (PowerShell)

- Opens new PowerShell window
- Window title shows the executable name
- Working directory is set to project folder
- Window stays open after server stops (if it crashes)
- User can scroll through logs

### Linux (GNOME Terminal)

- Opens new GNOME Terminal tab or window
- Falls back to xterm if GNOME Terminal not available
- Working directory is set to project folder
- Terminal closes when server stops

### macOS (Terminal.app)

- Opens new Terminal.app window
- Executes the command in that window
- Working directory is set to project folder
- Window stays open after command completes

## Error Handling

### Terminal Not Available

If no terminal emulator is found:
```
Warning: Could not open terminal window: exec: "gnome-terminal": executable file not found
Trying to start in background...
✓ Server started in background
```

Server still runs, just without visible terminal.

### Server Fails to Start

If server can't start:
```
Error: Could not start server: <error message>

To start manually: cd autonomous-app && ./autonomous-app.exe
```

User gets clear instructions for manual start.

## Benefits Over Background Execution

| Feature | Background | Separate Terminal |
|---------|-----------|-------------------|
| See logs | ❌ No | ✅ Yes |
| Easy to stop | ❌ Hard | ✅ Easy |
| Debug errors | ❌ Hard | ✅ Easy |
| See requests | ❌ No | ✅ Yes |
| Independent | ❌ No | ✅ Yes |
| Professional | ❌ No | ✅ Yes |

## Future Enhancements

Potential improvements:

1. **Terminal Preferences**
   - Let user choose terminal emulator
   - Save preference in config
   - Support more terminal types

2. **Window Customization**
   - Set window title
   - Set window size
   - Set window position
   - Custom color scheme

3. **Log Management**
   - Capture logs to file
   - View logs in TI
   - Search through logs
   - Export logs

4. **Process Management**
   - List running servers in TI
   - Stop servers from TI
   - Restart servers
   - View server status

5. **Multiple Servers**
   - Run multiple servers simultaneously
   - Each in its own terminal
   - Manage all from TI
   - Port conflict detection

## Technical Notes

### Why `Run()` Instead of `Start()`?

For opening terminal windows, we use `Run()` because:
- The terminal launcher command completes immediately
- The actual server runs in the new terminal process
- We don't need to track the launcher process
- The terminal manages the server process

### Process Lifecycle

```
TI → PowerShell → New Terminal → Server
     (Run())      (New Process)   (Runs until stopped)
     ↓
   Returns immediately
```

TI doesn't track the server process because:
- It runs in a separate terminal
- Terminal manages the process
- User controls it directly
- Independent of TI lifecycle

### Cross-Platform Compatibility

The code detects the platform and uses appropriate commands:
- Windows: PowerShell `Start-Process`
- Linux: `gnome-terminal` or `xterm`
- macOS: `open` with Terminal.app or `osascript`

All platforms fall back to background execution if terminal unavailable.
