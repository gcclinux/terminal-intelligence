# Debugging Guide

## Agentic Code Fixing Logs

By default, the agentic code fixing feature runs silently to avoid cluttering the terminal UI. However, if you need to debug issues with code fixing, you can enable detailed logging.

### Enabling Debug Logs

Debug logging is currently disabled by default. To enable it, you would need to modify the code:

1. Open `internal/ui/app.go`
2. Find the `New()` function where `AgenticCodeFixer` is initialized
3. Add the following line after creating the `agenticFixer`:

```go
agenticFixer.SetDebug(true)
```

This will enable INFO, ERROR, and DEBUG level logs that show:
- Fix request detection and confidence scores
- AI model calls and responses
- Code block extraction and validation
- Fix application steps
- Error details with context

### Log Levels

When debug mode is enabled, you'll see three types of logs:

- **[INFO]**: Major workflow steps (fix requests, AI calls, results)
- **[ERROR]**: Failures with context (validation errors, AI unavailable)
- **[DEBUG]**: Detailed execution flow (only when debug mode is enabled)

### Security Note

Logs are designed to protect sensitive information:
- File contents are NOT logged (only metadata like file paths and content lengths)
- API keys are NOT logged
- Only error messages and workflow information are logged

### Disabling Logs

To disable logs again, simply remove the `SetDebug(true)` line or set it to `false`:

```go
agenticFixer.SetDebug(false)
```

## Future Enhancement

In a future version, debug logging could be controlled via:
- Command-line flag: `--debug` or `-d`
- Environment variable: `MINI_CLI_CODER_DEBUG=1`
- Configuration file setting: `debug: true`
