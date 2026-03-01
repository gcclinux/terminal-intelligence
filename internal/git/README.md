# Git Integration Package

This package provides Git functionality for the Terminal Intelligence IDE using the go-git library.

## Overview

The Git integration provides a complete Git workflow without requiring external Git installation. It includes a visual panel interface accessible via `Ctrl+G` that supports all common Git operations.

## Components

- `client.go` - GitClient for Git operations (clone, pull, push, fetch, stage, commit, status, restore)
- `credentials.go` - CredentialStore for secure credential management

## Features

### Git Operations

**Remote Operations**
- **Clone**: Clone repositories from remote URLs with authentication support
- **Pull**: Fetch and merge changes from remote repository
- **Fetch**: Download changes from remote without merging

**Local to Remote Workflow**
- **Stage**: Stage all modified and untracked files for commit
- **Commit**: Create commits with custom messages (validates non-empty messages)
- **Push**: Push local commits to remote repository with authentication

**Info and Undo**
- **Status**: Display repository status (modified, staged, untracked files)
- **Restore**: Discard uncommitted changes and restore to last commit

### Authentication

- GitHub Personal Access Tokens (ghp_...)
- Username/Password authentication
- Automatic credential detection from `.git/config`
- Secure credential storage in repository configuration

### Repository Detection

- Automatically detects if current directory is a Git repository
- Loads stored credentials from existing repositories
- Populates remote URL from repository configuration

## User Interface

The Git panel (`internal/ui/gitpane.go`) provides:
- Three input fields: URL, Username, Password/Token
- Eight operation buttons organized into logical groups
- Dynamic commit message input (appears only when Commit is selected)
- Real-time status and error messages
- Keyboard-driven navigation

## Dependencies

- github.com/go-git/go-git/v5 - Pure Go Git implementation
- github.com/go-git/go-git/v5/plumbing/object - Git object types
- github.com/go-git/go-git/v5/plumbing/transport/http - HTTP authentication

## Testing

Tests are organized in the following directories:
- `tests/unit/git/` - Unit tests for individual components
- `tests/property/git/` - Property-based tests for correctness properties
- `tests/integration/git/` - Integration tests for end-to-end workflows

## Usage Example

```go
// Create a Git client
client := git.NewClient("/path/to/workspace")

// Clone a repository
result, err := client.Clone("https://github.com/user/repo.git", "username", "token", "")

// Stage changes
result, err = client.Stage()

// Commit with message
result, err = client.Commit("Add new feature")

// Push to remote
result, err = client.Push("username", "token")
```

## Error Handling

All operations return `*OperationResult` with:
- `Success`: Boolean indicating operation success
- `Message`: Human-readable result message
- `Error`: Detailed error information if operation failed

Errors are categorized for better user feedback:
- Authentication errors
- Network errors
- Repository state errors
- File system errors
