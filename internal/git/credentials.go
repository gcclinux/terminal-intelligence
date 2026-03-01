package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Credentials represents Git authentication credentials including URL, username, and password/token.
// The Password field can contain either a traditional password or a GitHub Personal Access Token (PAT).
type Credentials struct {
	URL      string // Repository URL (e.g., https://github.com/user/repo)
	Username string // Git username
	Password string // Password or GitHub PAT (tokens begin with ghp_)
}

// Store manages persistent storage and retrieval of Git credentials.
// Credentials are stored in the repository's .git/config file using standard Git configuration format.
type Store struct {
	repoPath string // Path to the Git repository root directory
}

// NewStore creates a new credential store for the specified repository path.
// The repoPath should point to the root directory of a Git repository (containing .git/).
func NewStore(repoPath string) *Store {
	return &Store{
		repoPath: repoPath,
	}
}

// Save writes credentials to the repository's .git/config file using standard Git configuration format.
// The credentials are stored in a [credential "URL"] section with username and password fields.
// For security, the .git/config file permissions are set to 0600 (owner read/write only).
// GitHub Personal Access Tokens (beginning with ghp_) are supported and stored as-is.
//
// Requirements: 14.1, 14.4, 5.2
func (s *Store) Save(creds *Credentials) error {
	if creds == nil {
		return fmt.Errorf("credentials cannot be nil")
	}

	configPath := filepath.Join(s.repoPath, ".git", "config")
	
	// Check if .git directory exists
	gitDir := filepath.Join(s.repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf(".git directory not found at %s", gitDir)
	}

	// Read existing config file
	content, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse existing config and remove old credential section for this URL
	lines := strings.Split(string(content), "\n")
	var newLines []string
	inCredentialSection := false
	credentialSectionHeader := fmt.Sprintf("[credential \"%s\"]", creds.URL)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check if we're entering a credential section for our URL
		if trimmed == credentialSectionHeader {
			inCredentialSection = true
			continue // Skip this line, we'll add new section later
		}
		
		// Check if we're entering a different section
		if strings.HasPrefix(trimmed, "[") && trimmed != credentialSectionHeader {
			inCredentialSection = false
		}
		
		// Skip lines within the credential section we're replacing
		if inCredentialSection {
			continue
		}
		
		newLines = append(newLines, line)
	}

	// Add new credential section
	newLines = append(newLines, credentialSectionHeader)
	newLines = append(newLines, fmt.Sprintf("\tusername = %s", creds.Username))
	newLines = append(newLines, fmt.Sprintf("\tpassword = %s", creds.Password))

	// Write updated config
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(configPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Explicitly set file permissions to 0600 for security (owner read/write only)
	if err := os.Chmod(configPath, 0600); err != nil {
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}

	return nil
}

// Load retrieves credentials from the repository's .git/config file.
// It parses the Git configuration format and returns the credentials for the repository's remote URL.
// Returns an error if the .git/config file doesn't exist or if no credentials are found.
//
// Requirements: 14.2
func (s *Store) Load() (*Credentials, error) {
	configPath := filepath.Join(s.repoPath, ".git", "config")
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at %s", configPath)
	}

	// Read config file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse config to find credential sections
	lines := strings.Split(string(content), "\n")
	var currentURL string
	var username, password string
	inCredentialSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check for credential section header: [credential "URL"]
		if strings.HasPrefix(trimmed, "[credential \"") && strings.HasSuffix(trimmed, "\"]") {
			inCredentialSection = true
			// Extract URL from [credential "URL"]
			start := strings.Index(trimmed, "\"") + 1
			end := strings.LastIndex(trimmed, "\"")
			if start > 0 && end > start {
				currentURL = trimmed[start:end]
			}
			continue
		}
		
		// Check if we're entering a different section
		if strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "[credential") {
			inCredentialSection = false
			continue
		}
		
		// Parse username and password within credential section
		if inCredentialSection {
			if strings.Contains(trimmed, "username = ") {
				parts := strings.SplitN(trimmed, "username = ", 2)
				if len(parts) == 2 {
					username = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(trimmed, "password = ") {
				parts := strings.SplitN(trimmed, "password = ", 2)
				if len(parts) == 2 {
					password = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	// Check if we found credentials
	if currentURL == "" || username == "" || password == "" {
		return nil, fmt.Errorf("no credentials found in config file")
	}

	return &Credentials{
		URL:      currentURL,
		Username: username,
		Password: password,
	}, nil
}

// Clear removes all credential sections from the repository's .git/config file.
// This method gracefully handles cases where the .git/config file doesn't exist.
// It preserves all other configuration sections while removing only credential sections.
//
// Requirements: 14.3
func (s *Store) Clear() error {
	configPath := filepath.Join(s.repoPath, ".git", "config")
	
	// Check if config file exists - if not, nothing to clear
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // Gracefully handle missing file
	}

	// Read existing config file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse config and remove all credential sections
	lines := strings.Split(string(content), "\n")
	var newLines []string
	inCredentialSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check if we're entering a credential section
		if strings.HasPrefix(trimmed, "[credential") {
			inCredentialSection = true
			continue // Skip credential section header
		}
		
		// Check if we're entering a different section
		if strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "[credential") {
			inCredentialSection = false
		}
		
		// Skip lines within credential sections
		if inCredentialSection {
			continue
		}
		
		newLines = append(newLines, line)
	}

	// Write updated config
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(configPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
