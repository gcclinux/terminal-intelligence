package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Client provides Git operations using the go-git library without requiring external Git executable.
// It wraps go-git functionality and provides a clean interface for common Git operations including
// clone, pull, push, fetch, stage, status, and restore.
type Client struct {
	workDir string // Current working directory where Git operations are performed
}

// OperationResult represents the outcome of a Git operation.
// It contains success status, a human-readable message, and any error that occurred.
type OperationResult struct {
	Success bool   // Whether the operation completed successfully
	Message string // Human-readable message describing the operation result
	Error   error  // Error details if the operation failed, nil on success
}

// RepositoryInfo contains information about a Git repository and its stored credentials.
// This is returned by repository detection operations to provide context about the current directory.
type RepositoryInfo struct {
	IsRepo      bool         // Whether the directory is a Git repository (contains .git/)
	RemoteURL   string       // URL of the remote repository, empty if not configured
	Credentials *Credentials // Stored credentials from .git/config, nil if not found
}

// NewClient creates a new Git client for the specified working directory.
// The workDir parameter should be the path where Git operations will be performed.
//
// Requirements: 4.1
func NewClient(workDir string) *Client {
	return &Client{
		workDir: workDir,
	}
}

// IsRepository checks if the specified directory is a Git repository by looking for a .git directory.
// It returns true if a .git directory exists, false otherwise.
//
// Requirements: 2.1
func (c *Client) IsRepository(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DetectRepository checks if the current working directory is a Git repository and retrieves
// repository information including the remote URL and stored credentials.
// It returns a RepositoryInfo struct containing:
// - IsRepo: whether a .git directory exists
// - RemoteURL: the URL of the remote repository (if configured)
// - Credentials: stored credentials from .git/config (if available)
//
// Requirements: 2.1, 2.2, 2.3
func (c *Client) DetectRepository() (*RepositoryInfo, error) {
	info := &RepositoryInfo{
		IsRepo:      false,
		RemoteURL:   "",
		Credentials: nil,
	}

	// Check if .git directory exists
	if !c.IsRepository(c.workDir) {
		return info, nil
	}

	info.IsRepo = true

	// Try to open the repository to get remote URL
	repo, err := git.PlainOpen(c.workDir)
	if err != nil {
		// Repository exists but couldn't be opened - return partial info
		return info, nil
	}

	// Get remote URL (typically "origin")
	remote, err := repo.Remote("origin")
	if err == nil && remote != nil {
		config := remote.Config()
		if len(config.URLs) > 0 {
			info.RemoteURL = config.URLs[0]
		}
	}

	// Try to load stored credentials
	store := NewStore(c.workDir)
	creds, err := store.Load()
	if err == nil {
		info.Credentials = creds
	}
	// Ignore credential load errors - they're optional

	return info, nil
}
// createAuth creates an HTTP basic authentication object for go-git operations.
// It detects GitHub Personal Access Tokens (PAT) by checking for the "ghp_" prefix.
// For GitHub PATs, the username can be any non-empty string (GitHub ignores it when using PATs).
// For regular username/password authentication, both values are used as provided.
//
// Parameters:
//   - username: The username for authentication (or any non-empty string for GitHub PATs)
//   - password: The password or GitHub Personal Access Token (tokens start with "ghp_")
//
// Returns:
//   - *http.BasicAuth: Authentication object for use with go-git operations
//
// Requirements: 5.1, 5.2, 5.3
// createAuth creates an HTTP basic authentication object for go-git operations.
// It detects GitHub Personal Access Tokens (PAT) by checking for the "ghp_" prefix.
// For GitHub PATs, the username can be any non-empty string (GitHub ignores it when using PATs).
// For regular username/password authentication, both values are used as provided.
//
// Parameters:
//   - username: The username for authentication (or any non-empty string for GitHub PATs)
//   - password: The password or GitHub Personal Access Token (tokens start with "ghp_")
//
// Returns:
//   - *http.BasicAuth: Authentication object for use with go-git operations
//
// Requirements: 5.1, 5.2, 5.3
func createAuth(username, password string) *http.BasicAuth {
	// GitHub PATs are recognized by the "ghp_" prefix
	// When using a PAT, the username can be anything (GitHub ignores it)
	// but we still need to provide a non-empty username for BasicAuth

	// If username is empty and we have a PAT, use a placeholder username
	if username == "" && len(password) >= 4 && password[:4] == "ghp_" {
		username = "oauth2"
	}

	return &http.BasicAuth{
		Username: username,
		Password: password,
	}
}
// Clone clones a Git repository from the specified URL to a target directory.
// If targetDir is empty, it derives the directory name from the repository URL.
// If the target directory already exists, it appends a numeric suffix to create a unique name.
// The operation uses the provided credentials for authentication.
//
// Parameters:
//   - url: The URL of the Git repository to clone
//   - username: The username for authentication (or any non-empty string for GitHub PATs)
//   - password: The password or GitHub Personal Access Token
//   - targetDir: The target directory path (if empty, derives from URL)
//
// Returns:
//   - *OperationResult: Contains success status, message, and the new directory path
//   - error: Any error that occurred during the operation
//
// The OperationResult.Message field contains the path to the cloned repository on success.
//
// Requirements: 7.1, 7.2, 7.3
func (c *Client) Clone(url, username, password, targetDir string) (*OperationResult, error) {
	// Determine the clone directory
	cloneDir := c.determineCloneDir(url, targetDir)

	// Create authentication
	auth := createAuth(username, password)

	// Perform the clone operation
	_, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
		URL:      url,
		Auth:     auth,
		Progress: os.Stdout, // Show progress to stdout
	})

	if err != nil {
		// Categorize and return the error
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Success - save credentials for future use
	store := NewStore(cloneDir)
	if err := store.Save(&Credentials{
		URL:      url,
		Username: username,
		Password: password,
	}); err != nil {
		// Log error but don't fail the operation
		// Credential save errors are non-critical
	}

	// Return the cloned directory path
	return &OperationResult{
		Success: true,
		Message: cloneDir,
		Error:   nil,
	}, nil
}
// Pull fetches changes from the remote repository and merges them into the current branch.
// It opens an existing repository in the working directory, gets the worktree, and performs
// a pull operation with the provided credentials.
//
// Parameters:
//   - username: The username for authentication (or any non-empty string for GitHub PATs)
//   - password: The password or GitHub Personal Access Token
//
// Returns:
//   - *OperationResult: Contains success status, message with commit count, and any error
//   - error: Any error that occurred during the operation
//
// The OperationResult.Message field contains information about the pull operation on success.
//
// Requirements: 8.1, 8.2
func (c *Client) Pull(username, password string) (*OperationResult, error) {
	// Open the existing repository
	repo, err := git.PlainOpen(c.workDir)
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Create authentication
	auth := createAuth(username, password)

	// Perform the pull operation
	err = worktree.Pull(&git.PullOptions{
		Auth:     auth,
		Progress: os.Stdout,
	})

	// Check for errors
	if err != nil {
		// "already up-to-date" is not an error condition
		if err == git.NoErrAlreadyUpToDate {
			return &OperationResult{
				Success: true,
				Message: "Already up-to-date",
				Error:   nil,
			}, nil
		}

		// Categorize and return the error
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Success - save credentials for future use
	store := NewStore(c.workDir)
	if err := store.Save(&Credentials{
		URL:      "", // Will be populated from remote URL
		Username: username,
		Password: password,
	}); err != nil {
		// Log error but don't fail the operation
		// Credential save errors are non-critical
	}

	// Get remote URL for credential storage
	remote, err := repo.Remote("origin")
	if err == nil && remote != nil {
		config := remote.Config()
		if len(config.URLs) > 0 {
			store.Save(&Credentials{
				URL:      config.URLs[0],
				Username: username,
				Password: password,
			})
		}
	}

	// Return success message
	return &OperationResult{
		Success: true,
		Message: "Pull completed successfully",
		Error:   nil,
	}, nil
}
// Push pushes local commits to the remote repository.
// It opens an existing repository in the working directory and performs a push operation
// with the provided credentials.
//
// Parameters:
//   - username: The username for authentication (or any non-empty string for GitHub PATs)
//   - password: The password or GitHub Personal Access Token
//
// Returns:
//   - *OperationResult: Contains success status, message, and any error
//   - error: Any error that occurred during the operation
//
// The OperationResult.Message field contains information about the push operation on success.
//
// Requirements: 9.1, 9.2
func (c *Client) Push(username, password string) (*OperationResult, error) {
	// Open the existing repository
	repo, err := git.PlainOpen(c.workDir)
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Create authentication
	auth := createAuth(username, password)

	// Perform the push operation
	err = repo.Push(&git.PushOptions{
		Auth:     auth,
		Progress: os.Stdout,
	})

	// Check for errors
	if err != nil {
		// "already up-to-date" is not an error condition
		if err == git.NoErrAlreadyUpToDate {
			return &OperationResult{
				Success: true,
				Message: "Already up-to-date",
				Error:   nil,
			}, nil
		}

		// Categorize and return the error
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Success - save credentials for future use
	store := NewStore(c.workDir)
	remote, err := repo.Remote("origin")
	if err == nil && remote != nil {
		config := remote.Config()
		if len(config.URLs) > 0 {
			store.Save(&Credentials{
				URL:      config.URLs[0],
				Username: username,
				Password: password,
			})
		}
	}

	// Return success message
	return &OperationResult{
		Success: true,
		Message: "Push completed successfully",
		Error:   nil,
	}, nil
}
// Fetch fetches changes from the remote repository without merging them.
// It opens an existing repository in the working directory and performs a fetch operation
// with the provided credentials. This allows reviewing updates before integrating them.
//
// Parameters:
//   - username: The username for authentication (or any non-empty string for GitHub PATs)
//   - password: The password or GitHub Personal Access Token
//
// Returns:
//   - *OperationResult: Contains success status, message, and any error
//   - error: Any error that occurred during the operation
//
// The OperationResult.Message field contains information about the fetch operation on success.
//
// Requirements: 10.1, 10.2
func (c *Client) Fetch(username, password string) (*OperationResult, error) {
	// Open the existing repository
	repo, err := git.PlainOpen(c.workDir)
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Create authentication
	auth := createAuth(username, password)

	// Perform the fetch operation
	err = repo.Fetch(&git.FetchOptions{
		Auth:     auth,
		Progress: os.Stdout,
	})

	// Check for errors
	if err != nil {
		// "already up-to-date" is not an error condition
		if err == git.NoErrAlreadyUpToDate {
			return &OperationResult{
				Success: true,
				Message: "Already up-to-date",
				Error:   nil,
			}, nil
		}

		// Categorize and return the error
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Success - save credentials for future use
	store := NewStore(c.workDir)
	remote, err := repo.Remote("origin")
	if err == nil && remote != nil {
		config := remote.Config()
		if len(config.URLs) > 0 {
			store.Save(&Credentials{
				URL:      config.URLs[0],
				Username: username,
				Password: password,
			})
		}
	}

	// Return success message
	return &OperationResult{
		Success: true,
		Message: "Fetch completed successfully",
		Error:   nil,
	}, nil
}

// determineCloneDir determines the target directory for a clone operation.
// If targetDir is provided, it uses that. Otherwise, it extracts the repository name
// from the URL and creates a directory in the working directory.
// If the directory already exists, it appends a numeric suffix to create a unique name.
//
// Parameters:
//   - url: The repository URL (e.g., "https://github.com/user/repo.git")
//   - targetDir: The desired target directory (empty string to auto-generate)
//
// Returns:
//   - string: The absolute path to the target directory for cloning
//
// Requirements: 7.3
func (c *Client) determineCloneDir(url, targetDir string) string {
	// If targetDir is provided, use it
	if targetDir != "" {
		return targetDir
	}

	// Extract repository name from URL
	repoName := extractRepoName(url)

	// Create base directory path
	baseDir := filepath.Join(c.workDir, repoName)

	// Check if directory exists
	if !dirExists(baseDir) {
		return baseDir
	}

	// Find unique name by appending numbers
	for i := 2; i < 100; i++ {
		dir := filepath.Join(c.workDir, fmt.Sprintf("%s-%d", repoName, i))
		if !dirExists(dir) {
			return dir
		}
	}

	// Fallback: use timestamp
	return filepath.Join(c.workDir, fmt.Sprintf("%s-%d", repoName, time.Now().Unix()))
}

// extractRepoName extracts the repository name from a Git URL.
// It handles various URL formats including HTTPS, SSH, and Git protocol.
//
// Examples:
//   - "https://github.com/user/repo.git" -> "repo"
//   - "https://github.com/user/repo" -> "repo"
//   - "git@github.com:user/repo.git" -> "repo"
//
// Parameters:
//   - url: The Git repository URL
//
// Returns:
//   - string: The extracted repository name
func extractRepoName(url string) string {
	// Remove trailing .git if present
	url = strings.TrimSuffix(url, ".git")

	// Find the last slash or colon (for SSH URLs)
	lastSlash := strings.LastIndex(url, "/")
	lastColon := strings.LastIndex(url, ":")

	// Use the position of the last separator
	lastSep := lastSlash
	if lastColon > lastSlash {
		lastSep = lastColon
	}

	// Extract the name after the last separator
	if lastSep >= 0 && lastSep < len(url)-1 {
		return url[lastSep+1:]
	}

	// Fallback: use the entire URL (sanitized)
	return "repository"
}

// dirExists checks if a directory exists at the specified path.
//
// Parameters:
//   - path: The directory path to check
//
// Returns:
//   - bool: true if the directory exists, false otherwise
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// categorizeError categorizes Git operation errors into authentication, network, or Git operation errors.
// It examines the error message to determine the category and returns a formatted error with helpful hints.
//
// Parameters:
//   - err: The error to categorize
//
// Returns:
//   - error: A categorized error with a descriptive message
//
// Requirements: 15.1, 15.2
func categorizeError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Check for authentication errors
	if strings.Contains(errMsg, "authentication") ||
	   strings.Contains(errMsg, "401") ||
	   strings.Contains(errMsg, "403") ||
	   strings.Contains(errMsg, "unauthorized") ||
	   strings.Contains(errMsg, "forbidden") {
		return &GitError{
			Category: "Authentication",
			Message:  "Authentication failed: " + errMsg,
			Hint:     "For private repositories, use a GitHub Personal Access Token (ghp_...)",
			Original: err,
		}
	}

	// Check for network errors
	if strings.Contains(errMsg, "timeout") ||
	   strings.Contains(errMsg, "connection") ||
	   strings.Contains(errMsg, "network") ||
	   strings.Contains(errMsg, "dial") ||
	   strings.Contains(errMsg, "DNS") {
		return &GitError{
			Category: "Network",
			Message:  "Network error: " + errMsg,
			Hint:     "Check your internet connection and try again",
			Original: err,
		}
	}

	// Check for repository not found
	if strings.Contains(errMsg, "404") ||
	   strings.Contains(errMsg, "not found") ||
	   strings.Contains(errMsg, "repository not found") {
		return &GitError{
			Category: "Git Operation",
			Message:  "Git operation failed: Repository not found (404)",
			Hint:     "Verify the repository URL is correct and you have access permissions",
			Original: err,
		}
	}

	// Default to Git operation error
	return &GitError{
		Category: "Git Operation",
		Message:  "Git operation failed: " + errMsg,
		Hint:     "",
		Original: err,
	}
}

// GitError represents a categorized Git operation error with helpful hints.
type GitError struct {
	Category string // "Authentication", "Network", or "Git Operation"
	Message  string // Formatted error message
	Hint     string // Helpful hint for resolving the error
	Original error  // The original error
}

// Error implements the error interface for GitError.
func (e *GitError) Error() string {
	if e.Hint != "" {
		return e.Message + "\nHint: " + e.Hint
	}
	return e.Message
}

// Unwrap returns the original error for error unwrapping.
func (e *GitError) Unwrap() error {
	return e.Original
}

// Stage stages all modified and untracked files in the repository.
// It opens an existing repository in the working directory, gets the worktree,
// and stages all changes to prepare them for commit.
//
// Returns:
//   - *OperationResult: Contains success status, message with file count, and any error
//   - error: Any error that occurred during the operation
//
// The OperationResult.Message field contains the number of files staged on success.
//
// Requirements: 11.1
func (c *Client) Stage() (*OperationResult, error) {
	// Open the existing repository
	repo, err := git.PlainOpen(c.workDir)
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Get the current status to count files
	status, err := worktree.Status()
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Count files that need staging (modified or untracked)
	fileCount := 0
	for _, fileStatus := range status {
		// Check if file is modified or untracked
		if fileStatus.Worktree != git.Unmodified && fileStatus.Worktree != git.Untracked {
			fileCount++
		} else if fileStatus.Worktree == git.Untracked {
			fileCount++
		}
	}

	// Stage all files using "." pattern (stages everything)
	err = worktree.AddWithOptions(&git.AddOptions{
		All: true,
	})
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Success - return success message with file count
	message := fmt.Sprintf("Staged %d file(s)", fileCount)
	return &OperationResult{
		Success: true,
		Message: message,
		Error:   nil,
	}, nil
}

// Commit creates a new commit with the staged changes.
// It opens an existing repository in the working directory, gets the worktree,
// and commits all staged changes with the provided commit message.
//
// Parameters:
//   - message: The commit message
//
// Returns:
//   - *OperationResult: Contains success status, commit hash, and any error
//   - error: Any error that occurred during the operation
//
// The OperationResult.Message field contains the commit hash on success.
func (c *Client) Commit(message string) (*OperationResult, error) {
	// Open the existing repository
	repo, err := git.PlainOpen(c.workDir)
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Check if there are staged changes
	status, err := worktree.Status()
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Count staged files
	stagedCount := 0
	for _, fileStatus := range status {
		if fileStatus.Staging != git.Unmodified && fileStatus.Staging != git.Untracked {
			stagedCount++
		}
	}

	if stagedCount == 0 {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   fmt.Errorf("no changes staged for commit"),
		}, fmt.Errorf("no changes staged for commit")
	}

	// Use default message if empty
	if message == "" {
		message = "Update files"
	}

	// Create the commit
	hash, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Terminal Intelligence User",
			Email: "user@terminal-intelligence.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Success - return commit hash
	resultMessage := fmt.Sprintf("Committed %d file(s): %s", stagedCount, hash.String()[:7])
	return &OperationResult{
		Success: true,
		Message: resultMessage,
		Error:   nil,
	}, nil
}

// Status retrieves the current status of the repository.
// It opens an existing repository in the working directory, gets the worktree status,
// and formats it into three lists: modified files, staged files, and untracked files.
//
// Returns:
//   - *OperationResult: Contains success status, formatted status message, and any error
//   - error: Any error that occurred during the operation
//
// The OperationResult.Message field contains the formatted status with three sections:
// - Modified: files that have been changed but not staged
// - Staged: files that have been staged for commit
// - Untracked: new files that are not tracked by Git
//
// Requirements: 12.1, 12.2, 12.3, 12.4
func (c *Client) Status() (*OperationResult, error) {
	// Open the existing repository
	repo, err := git.PlainOpen(c.workDir)
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Get the current status
	status, err := worktree.Status()
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Categorize files into three lists
	var modified []string
	var staged []string
	var untracked []string

	for filename, fileStatus := range status {
		// Each file should appear in exactly one list
		// Priority: untracked > staged > modified
		
		if fileStatus.Worktree == git.Untracked {
			// Untracked files (new files not added to staging)
			untracked = append(untracked, filename)
		} else if fileStatus.Staging != git.Unmodified {
			// Staged files (files in staging area)
			staged = append(staged, filename)
		} else if fileStatus.Worktree != git.Unmodified {
			// Modified files (changed but not staged)
			modified = append(modified, filename)
		}
	}

	// Format the status message
	var message strings.Builder
	message.WriteString("Repository Status:\n\n")

	// Modified files
	message.WriteString(fmt.Sprintf("Modified (%d):\n", len(modified)))
	if len(modified) > 0 {
		for _, file := range modified {
			message.WriteString(fmt.Sprintf("  • %s\n", file))
		}
	} else {
		message.WriteString("  (none)\n")
	}
	message.WriteString("\n")

	// Staged files
	message.WriteString(fmt.Sprintf("Staged (%d):\n", len(staged)))
	if len(staged) > 0 {
		for _, file := range staged {
			message.WriteString(fmt.Sprintf("  • %s\n", file))
		}
	} else {
		message.WriteString("  (none)\n")
	}
	message.WriteString("\n")

	// Untracked files
	message.WriteString(fmt.Sprintf("Untracked (%d):\n", len(untracked)))
	if len(untracked) > 0 {
		for _, file := range untracked {
			message.WriteString(fmt.Sprintf("  • %s\n", file))
		}
	} else {
		message.WriteString("  (none)\n")
	}

	// Success - return formatted status
	return &OperationResult{
		Success: true,
		Message: message.String(),
		Error:   nil,
	}, nil
}

// Restore restores all modified files to their last committed state.
// It opens an existing repository in the working directory, gets the worktree,
// and discards all changes to tracked files, reverting them to HEAD.
//
// Returns:
//   - *OperationResult: Contains success status, message with file count, and any error
//   - error: Any error that occurred during the operation
//
// The OperationResult.Message field contains the number of files restored on success.
//
// Requirements: 13.1
func (c *Client) Restore() (*OperationResult, error) {
	// Open the existing repository
	repo, err := git.PlainOpen(c.workDir)
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Get the current status to count modified files
	status, err := worktree.Status()
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Count modified files (files that will be restored)
	fileCount := 0
	for _, fileStatus := range status {
		// Count files that are modified in worktree (not untracked)
		if fileStatus.Worktree != git.Unmodified && fileStatus.Worktree != git.Untracked {
			fileCount++
		}
	}

	// Restore all modified files by checking out HEAD
	// This reverts all changes in the worktree
	err = worktree.Reset(&git.ResetOptions{
		Mode: git.HardReset,
	})
	if err != nil {
		return &OperationResult{
			Success: false,
			Message: "",
			Error:   categorizeError(err),
		}, categorizeError(err)
	}

	// Success - return success message with file count
	message := fmt.Sprintf("Restored %d file(s)", fileCount)
	return &OperationResult{
		Success: true,
		Message: message,
		Error:   nil,
	}, nil
}
