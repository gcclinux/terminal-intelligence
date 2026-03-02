## Architecture

The application follows a clean architecture with clear separation of concerns:

- **UI Layer**: Bubble Tea components for terminal interface
- **Business Logic**: File management, command execution, AI communication
- **Data Layer**: File system operations

### Key Components

- **App**: Main Bubble Tea application orchestrating all components
- **EditorPane**: Code editor with syntax highlighting and file editing
- **AIChatPane**: AI interaction pane with conversation history
- **FileManager**: Handles all file system operations
- **CommandExecutor**: Executes system commands and scripts
- **OllamaClient**: Communicates with Ollama service via REST API

## Configuration

The application can be configured via:

1. **Configuration file** at `~/.ti/config.json` (loaded automatically if present)
2. **Command-line flags** (override config file values)
3. **Default configuration** (used when no config file or flags provided)

### Configuration Priority

Command-line flags > Config file > Built-in defaults

### Default Configuration

- **Workspace Directory**: `~/ti-workspace`
- **Ollama URL**: `http://localhost:11434`
- **Default Model**: `llama2`
- **Tab Size**: 4 spaces
- **Auto Save**: Disabled

## Cross-Platform Support

The application is designed to run on:
- **Linux**: Tested on Ubuntu 20.04+
- **Windows**: Tested on Windows 10+
- **macOS**: Tested on macOS 11+ (Intel and Apple Silicon)

## Development

### Prerequisites

- Go 1.21 or higher
- Make (for build automation)
- golangci-lint (optional, for linting)

### Development Commands

```bash
make fmt        # Format code
make lint       # Run linter
make run        # Build and run
make clean      # Clean build artifacts
```

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass (`make test`)
2. Code is formatted (`make fmt`)
3. Linter passes (`make lint`)
4. New features include tests