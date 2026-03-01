# Git Integration Package

This package provides Git functionality for the Terminal Intelligence IDE using the go-git library.

## Components

- `client.go` - GitClient for Git operations (clone, pull, push, fetch, stage, status, restore)
- `credentials.go` - CredentialStore for secure credential management
- Additional components will be added as implementation progresses

## Dependencies

- github.com/go-git/go-git/v5 - Pure Go Git implementation
- github.com/leanovate/gopter - Property-based testing library

## Testing

Tests are organized in the following directories:
- `tests/unit/git/` - Unit tests for individual components
- `tests/property/git/` - Property-based tests for correctness properties
- `tests/integration/git/` - Integration tests for end-to-end workflows
