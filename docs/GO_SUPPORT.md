# Go Language Support in Terminal Intelligence

Terminal Intelligence now includes full support for editing, running, and testing Go source files.

## Features

### File Type Detection
- Automatically detects `.go` files and sets the file type to "go"
- Provides appropriate syntax context for AI assistance

### Execution with Ctrl+R

When you press `Ctrl+R` on a Go file, Terminal Intelligence automatically determines the best execution method:

#### Regular Go Files
For standard Go source files (e.g., `main.go`, `hello.go`):
```bash
go run filename.go
```

This compiles and runs the Go program in one step, displaying output in the AI pane.

#### Test Files
For Go test files (files ending with `_test.go`):
```bash
go test -v filename_test.go
```

This runs the tests with verbose output, showing detailed test results.

## Usage Examples

### Example 1: Running a Simple Go Program

1. Create or open a Go file (e.g., `hello.go`):
```go
package main

import "fmt"

func main() {
    fmt.Println("Hello from Terminal Intelligence!")
}
```

2. Press `Ctrl+S` to save
3. Press `Ctrl+R` to run
4. See the output in the AI pane

### Example 2: Running Go Tests

1. Create or open a test file (e.g., `math_test.go`):
```go
package main

import "testing"

func TestAdd(t *testing.T) {
    result := add(2, 3)
    if result != 5 {
        t.Errorf("Expected 5, got %d", result)
    }
}

func add(a, b int) int {
    return a + b
}
```

2. Press `Ctrl+S` to save
3. Press `Ctrl+R` to run tests
4. See test results in the AI pane

### Example 3: AI-Assisted Go Development

You can ask the AI to help with Go code:

**Conversational Mode:**
```
You: /ask how do I create a goroutine in Go?
```

**Agentic Mode (AI modifies code):**
```
You: /fix add error handling to this function
```

**Preview Mode:**
```
You: /preview refactor this to use interfaces
```

## Workflow

### Creating a New Go Program

1. Press `Ctrl+N` to create a new file
2. Enter filename with `.go` extension (e.g., `myapp.go`)
3. Write your Go code in the editor
4. Press `Ctrl+S` to save
5. Press `Ctrl+R` to run

### Editing Existing Go Files

1. Press `Ctrl+O` to open file picker
2. Select your `.go` file
3. Edit as needed
4. Press `Ctrl+S` to save changes
5. Press `Ctrl+R` to run

### Testing Go Code

1. Create a test file with `_test.go` suffix
2. Write your test functions (must start with `Test`)
3. Press `Ctrl+S` to save
4. Press `Ctrl+R` to run tests
5. View test results in the AI pane

## AI Integration

The AI assistant understands Go syntax and can help with:

- **Code Generation**: Generate Go functions, structs, interfaces
- **Error Handling**: Add proper error handling patterns
- **Refactoring**: Improve code structure and organization
- **Testing**: Generate test cases and table-driven tests
- **Concurrency**: Help with goroutines and channels
- **Best Practices**: Apply Go idioms and conventions

### Example AI Interactions

**Generate a struct:**
```
You: create a User struct with name, email, and age fields
```

**Add error handling:**
```
You: /fix add error handling to the file reading code
```

**Generate tests:**
```
You: create table-driven tests for the calculate function
```

**Refactor code:**
```
You: /preview refactor this to use the io.Reader interface
```

## Requirements

- Go must be installed on your system
- `go` command must be available in your PATH
- For test files, the Go testing package is used automatically

## Supported Go Features

- Standard Go programs with `package main`
- Library packages
- Test files with `_test.go` suffix
- All Go standard library packages
- Third-party packages (if already installed via `go get`)

## Limitations

- Multi-file Go programs: Only the current file is executed with `go run`
  - For multi-file programs, consider using `go build` manually or creating a build script
- Module dependencies: Ensure `go.mod` is properly configured for external dependencies
- Build tags and conditional compilation: Not automatically detected

## Tips

1. **Auto-save before run**: The editor automatically saves before running (Ctrl+R)
2. **Test naming**: Test files must end with `_test.go` to be recognized
3. **Package context**: For library packages, create a test file to verify functionality
4. **Error messages**: Go compiler errors appear in the terminal output pane
5. **AI context**: The AI can read your Go code and provide context-aware suggestions

## Keyboard Shortcuts for Go Development

| Shortcut | Action |
|----------|--------|
| `Ctrl+N` | Create new Go file |
| `Ctrl+O` | Open existing Go file |
| `Ctrl+S` | Save Go file |
| `Ctrl+R` | Run Go program or tests |
| `Ctrl+Enter` | Ask AI with current file context |
| `Tab` | Switch between editor and AI pane |

## Examples

See the test files in the project root:
- `test_go_support.go` - Example Go program
- `test_go_support_test.go` - Example Go tests

You can open these files in Terminal Intelligence to see Go support in action!

## Troubleshooting

**Problem**: "go: command not found"
- **Solution**: Install Go from https://golang.org/dl/ and ensure it's in your PATH

**Problem**: "package X is not in GOROOT"
- **Solution**: Run `go get X` to install the missing package

**Problem**: Tests don't run
- **Solution**: Ensure filename ends with `_test.go` and test functions start with `Test`

**Problem**: Multi-file program doesn't compile
- **Solution**: Use `go build` in terminal or create all files in the same package

## Future Enhancements

Potential future improvements for Go support:
- Automatic `go build` for multi-file programs
- Integration with `go fmt` for automatic formatting
- Support for `go mod` commands
- Debugging support with Delve
- Code completion and IntelliSense

---

[← Back to README](../README.md)
