package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestSave(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Create initial config file
	configPath := filepath.Join(gitDir, "config")
	initialConfig := `[core]
	repositoryformatversion = 0
	filemode = true
[remote "origin"]
	url = https://github.com/test/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create initial config: %v", err)
	}

	// Create store and save credentials
	store := NewStore(tempDir)
	creds := &Credentials{
		URL:      "https://github.com/test/repo.git",
		Username: "testuser",
		Password: "ghp_testtoken123",
	}

	if err := store.Save(creds); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file permissions are 0600
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", info.Mode().Perm())
	}

	// Read and verify config content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	configStr := string(content)

	// Verify credential section exists
	expectedSection := `[credential "https://github.com/test/repo.git"]`
	if !strings.Contains(configStr, expectedSection) {
		t.Errorf("Config missing credential section. Got:\n%s", configStr)
	}

	// Verify username
	if !strings.Contains(configStr, "username = testuser") {
		t.Errorf("Config missing username. Got:\n%s", configStr)
	}

	// Verify password (GitHub PAT)
	if !strings.Contains(configStr, "password = ghp_testtoken123") {
		t.Errorf("Config missing password. Got:\n%s", configStr)
	}

	// Verify original config sections are preserved
	if !strings.Contains(configStr, "[core]") {
		t.Errorf("Original [core] section missing")
	}
	if !strings.Contains(configStr, "[remote \"origin\"]") {
		t.Errorf("Original [remote] section missing")
	}
}

func TestSave_UpdateExistingCredentials(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Create config with existing credentials
	configPath := filepath.Join(gitDir, "config")
	initialConfig := `[core]
	repositoryformatversion = 0
[credential "https://github.com/test/repo.git"]
	username = olduser
	password = oldpass
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create initial config: %v", err)
	}

	// Save new credentials
	store := NewStore(tempDir)
	creds := &Credentials{
		URL:      "https://github.com/test/repo.git",
		Username: "newuser",
		Password: "ghp_newtoken456",
	}

	if err := store.Save(creds); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Read and verify config content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	configStr := string(content)

	// Verify new credentials
	if !strings.Contains(configStr, "username = newuser") {
		t.Errorf("Config should have new username. Got:\n%s", configStr)
	}
	if !strings.Contains(configStr, "password = ghp_newtoken456") {
		t.Errorf("Config should have new password. Got:\n%s", configStr)
	}

	// Verify old credentials are gone
	if strings.Contains(configStr, "olduser") {
		t.Errorf("Config should not contain old username. Got:\n%s", configStr)
	}
	if strings.Contains(configStr, "oldpass") {
		t.Errorf("Config should not contain old password. Got:\n%s", configStr)
	}
}

func TestSave_NoGitDirectory(t *testing.T) {
	// Create a temporary directory without .git
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := NewStore(tempDir)
	creds := &Credentials{
		URL:      "https://github.com/test/repo.git",
		Username: "testuser",
		Password: "testpass",
	}

	err = store.Save(creds)
	if err == nil {
		t.Error("Expected error when .git directory doesn't exist")
	}
	if !strings.Contains(err.Error(), ".git directory not found") {
		t.Errorf("Expected '.git directory not found' error, got: %v", err)
	}
}

func TestSave_NilCredentials(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := NewStore(tempDir)
	err = store.Save(nil)
	if err == nil {
		t.Error("Expected error when credentials are nil")
	}
	if !strings.Contains(err.Error(), "credentials cannot be nil") {
		t.Errorf("Expected 'credentials cannot be nil' error, got: %v", err)
	}
}

func TestLoad(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Create config file with credentials
	configPath := filepath.Join(gitDir, "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
[remote "origin"]
	url = https://github.com/test/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
[credential "https://github.com/test/repo.git"]
	username = testuser
	password = ghp_testtoken123
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Load credentials
	store := NewStore(tempDir)
	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify loaded credentials
	if creds.URL != "https://github.com/test/repo.git" {
		t.Errorf("Expected URL 'https://github.com/test/repo.git', got '%s'", creds.URL)
	}
	if creds.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", creds.Username)
	}
	if creds.Password != "ghp_testtoken123" {
		t.Errorf("Expected password 'ghp_testtoken123', got '%s'", creds.Password)
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	// Create a temporary directory without .git/config
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory but no config file
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	store := NewStore(tempDir)
	_, err = store.Load()
	if err == nil {
		t.Error("Expected error when config file doesn't exist")
	}
	if !strings.Contains(err.Error(), "config file not found") {
		t.Errorf("Expected 'config file not found' error, got: %v", err)
	}
}

func TestLoad_NoCredentials(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Create config file without credentials
	configPath := filepath.Join(gitDir, "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
[remote "origin"]
	url = https://github.com/test/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	store := NewStore(tempDir)
	_, err = store.Load()
	if err == nil {
		t.Error("Expected error when no credentials found")
	}
	if !strings.Contains(err.Error(), "no credentials found") {
		t.Errorf("Expected 'no credentials found' error, got: %v", err)
	}
}

func TestLoad_EmptyConfigFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Create empty config file
	configPath := filepath.Join(gitDir, "config")
	if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	store := NewStore(tempDir)
	_, err = store.Load()
	if err == nil {
		t.Error("Expected error when config file is empty")
	}
	if !strings.Contains(err.Error(), "no credentials found") {
		t.Errorf("Expected 'no credentials found' error, got: %v", err)
	}
}

func TestLoad_MultipleCredentialSections(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Create config file with multiple credential sections
	configPath := filepath.Join(gitDir, "config")
	configContent := `[core]
	repositoryformatversion = 0
[credential "https://github.com/other/repo.git"]
	username = otheruser
	password = otherpass
[credential "https://github.com/test/repo.git"]
	username = testuser
	password = ghp_testtoken123
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Load credentials - should get the last one
	store := NewStore(tempDir)
	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify we got the last credential section
	if creds.URL != "https://github.com/test/repo.git" {
		t.Errorf("Expected URL 'https://github.com/test/repo.git', got '%s'", creds.URL)
	}
	if creds.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", creds.Username)
	}
	if creds.Password != "ghp_testtoken123" {
		t.Errorf("Expected password 'ghp_testtoken123', got '%s'", creds.Password)
	}
}

func TestClear(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Create config file with credentials
	configPath := filepath.Join(gitDir, "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
[remote "origin"]
	url = https://github.com/test/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
[credential "https://github.com/test/repo.git"]
	username = testuser
	password = ghp_testtoken123
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Clear credentials
	store := NewStore(tempDir)
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Read and verify config content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	configStr := string(content)

	// Verify credential section is removed
	if strings.Contains(configStr, "[credential") {
		t.Errorf("Config should not contain credential section. Got:\n%s", configStr)
	}
	if strings.Contains(configStr, "testuser") {
		t.Errorf("Config should not contain username. Got:\n%s", configStr)
	}
	if strings.Contains(configStr, "ghp_testtoken123") {
		t.Errorf("Config should not contain password. Got:\n%s", configStr)
	}

	// Verify other sections are preserved
	if !strings.Contains(configStr, "[core]") {
		t.Errorf("Original [core] section should be preserved")
	}
	if !strings.Contains(configStr, "[remote \"origin\"]") {
		t.Errorf("Original [remote] section should be preserved")
	}
}

func TestClear_MissingConfigFile(t *testing.T) {
	// Create a temporary directory without .git/config
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory but no config file
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Clear should handle missing file gracefully
	store := NewStore(tempDir)
	err = store.Clear()
	if err != nil {
		t.Errorf("Clear should handle missing file gracefully, got error: %v", err)
	}
}

func TestClear_MultipleCredentialSections(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Create config file with multiple credential sections
	configPath := filepath.Join(gitDir, "config")
	configContent := `[core]
	repositoryformatversion = 0
[credential "https://github.com/repo1.git"]
	username = user1
	password = pass1
[remote "origin"]
	url = https://github.com/test/repo.git
[credential "https://github.com/repo2.git"]
	username = user2
	password = pass2
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Clear all credentials
	store := NewStore(tempDir)
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Read and verify config content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	configStr := string(content)

	// Verify all credential sections are removed
	if strings.Contains(configStr, "[credential") {
		t.Errorf("Config should not contain any credential sections. Got:\n%s", configStr)
	}
	if strings.Contains(configStr, "user1") || strings.Contains(configStr, "user2") {
		t.Errorf("Config should not contain any usernames. Got:\n%s", configStr)
	}
	if strings.Contains(configStr, "pass1") || strings.Contains(configStr, "pass2") {
		t.Errorf("Config should not contain any passwords. Got:\n%s", configStr)
	}

	// Verify other sections are preserved
	if !strings.Contains(configStr, "[core]") {
		t.Errorf("Original [core] section should be preserved")
	}
	if !strings.Contains(configStr, "[remote \"origin\"]") {
		t.Errorf("Original [remote] section should be preserved")
	}
}

func TestClear_NoGitDirectory(t *testing.T) {
	// Create a temporary directory without .git
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Clear should handle missing .git directory gracefully
	store := NewStore(tempDir)
	err = store.Clear()
	if err != nil {
		t.Errorf("Clear should handle missing .git directory gracefully, got error: %v", err)
	}
}

// Property-Based Tests

// TestProperty4_CredentialRoundTripPersistence tests Property 4: Credential Round-Trip Persistence
// **Validates: Requirements 14.2**
//
// For any valid credentials (URL, username, password), saving them to the credential store
// and then loading them should return equivalent credential values.
func TestProperty4_CredentialRoundTripPersistence(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("saving and loading credentials returns identical values", prop.ForAll(
		func(url, username, password string) bool {
			// Create a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "git-creds-prop-test-*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			// Create .git directory
			gitDir := filepath.Join(tempDir, ".git")
			if err := os.Mkdir(gitDir, 0755); err != nil {
				t.Logf("Failed to create .git dir: %v", err)
				return false
			}

			// Create initial config file (simulating existing repo)
			configPath := filepath.Join(gitDir, "config")
			initialConfig := `[core]
	repositoryformatversion = 0
	filemode = true
`
			if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
				t.Logf("Failed to create initial config: %v", err)
				return false
			}

			// Create store and save credentials
			store := NewStore(tempDir)
			originalCreds := &Credentials{
				URL:      url,
				Username: username,
				Password: password,
			}

			if err := store.Save(originalCreds); err != nil {
				t.Logf("Save failed: %v", err)
				return false
			}

			// Load credentials back
			loadedCreds, err := store.Load()
			if err != nil {
				t.Logf("Load failed: %v", err)
				return false
			}

			// Verify round-trip: loaded credentials should match original
			if loadedCreds.URL != originalCreds.URL {
				t.Logf("URL mismatch: expected %q, got %q", originalCreds.URL, loadedCreds.URL)
				return false
			}
			if loadedCreds.Username != originalCreds.Username {
				t.Logf("Username mismatch: expected %q, got %q", originalCreds.Username, loadedCreds.Username)
				return false
			}
			if loadedCreds.Password != originalCreds.Password {
				t.Logf("Password mismatch: expected %q, got %q", originalCreds.Password, loadedCreds.Password)
				return false
			}

			return true
		},
		genGitURL(),
		genUsername(),
		genPassword(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty18_CredentialStorageSecurity tests Property 18: Credential Storage Security
// **Validates: Requirements 14.3**
//
// For any credentials saved to the credential store, the storage file (.git/config)
// should have file permissions set to 0600 (owner read/write only).
func TestProperty18_CredentialStorageSecurity(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("saved credentials have secure file permissions (0600)", prop.ForAll(
		func(url, username, password string) bool {
			// Create a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "git-creds-security-test-*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tempDir)

			// Create .git directory
			gitDir := filepath.Join(tempDir, ".git")
			if err := os.Mkdir(gitDir, 0755); err != nil {
				t.Logf("Failed to create .git dir: %v", err)
				return false
			}

			// Create initial config file with different permissions (0644)
			configPath := filepath.Join(gitDir, "config")
			initialConfig := `[core]
	repositoryformatversion = 0
	filemode = true
`
			if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
				t.Logf("Failed to create initial config: %v", err)
				return false
			}

			// Verify initial permissions are NOT 0600
			initialInfo, err := os.Stat(configPath)
			if err != nil {
				t.Logf("Failed to stat config file before save: %v", err)
				return false
			}
			if initialInfo.Mode().Perm() == 0600 {
				t.Logf("Initial permissions should not be 0600 for this test")
				// Continue anyway - the important part is checking after Save
			}

			// Create store and save credentials
			store := NewStore(tempDir)
			creds := &Credentials{
				URL:      url,
				Username: username,
				Password: password,
			}

			if err := store.Save(creds); err != nil {
				t.Logf("Save failed: %v", err)
				return false
			}

			// Verify file permissions are 0600 after save
			info, err := os.Stat(configPath)
			if err != nil {
				t.Logf("Failed to stat config file after save: %v", err)
				return false
			}

			actualPerms := info.Mode().Perm()
			if actualPerms != 0600 {
				t.Logf("Expected file permissions 0600, got %o (octal) for credentials: URL=%q, Username=%q, Password type=%s",
					actualPerms, url, username, getPasswordType(password))
				return false
			}

			return true
		},
		genGitURL(),
		genUsername(),
		genPassword(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// getPasswordType returns a description of the password type for logging
func getPasswordType(password string) string {
	if strings.HasPrefix(password, "ghp_") {
		return "GitHub PAT"
	}
	if strings.Contains(password, " ") {
		return "password with spaces"
	}
	if len(password) > 50 {
		return "long password"
	}
	return "regular password"
}

// Generator functions for property-based testing

// genGitURL generates various Git repository URL formats
func genGitURL() gopter.Gen {
	return gen.OneGenOf(
		// HTTPS URLs
		gen.Const("https://github.com/user/repo.git"),
		gen.Const("https://github.com/org/project.git"),
		gen.Const("https://gitlab.com/user/repo.git"),
		gen.Const("https://bitbucket.org/user/repo.git"),
		// HTTPS URLs without .git suffix
		gen.Const("https://github.com/user/repo"),
		gen.Const("https://github.com/org/project"),
		// URLs with special characters in repo names
		gen.Const("https://github.com/user/my-repo.git"),
		gen.Const("https://github.com/user/my_repo.git"),
		gen.Const("https://github.com/user/repo-123.git"),
		// URLs with subgroups (GitLab style)
		gen.Const("https://gitlab.com/group/subgroup/repo.git"),
		// URLs with ports
		gen.Const("https://git.example.com:8443/user/repo.git"),
		// Self-hosted Git servers
		gen.Const("https://git.company.com/team/project.git"),
	)
}

// genUsername generates various username formats
func genUsername() gopter.Gen {
	return gen.OneGenOf(
		// Simple usernames
		gen.Const("user"),
		gen.Const("testuser"),
		gen.Const("developer"),
		// Usernames with numbers
		gen.Const("user123"),
		gen.Const("dev42"),
		// Usernames with hyphens and underscores
		gen.Const("test-user"),
		gen.Const("test_user"),
		gen.Const("my-dev-account"),
		// Email-style usernames
		gen.Const("user@example.com"),
		gen.Const("dev@company.org"),
		// Mixed case usernames
		gen.Const("TestUser"),
		gen.Const("DevAccount"),
		// Long usernames
		gen.Const("very-long-username-for-testing"),
	)
}

// genPassword generates various password and token formats
func genPassword() gopter.Gen {
	return gen.OneGenOf(
		// GitHub Personal Access Tokens (ghp_ prefix)
		gen.Const("ghp_1234567890abcdefghijklmnopqrstuvwxyz"),
		gen.Const("ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"),
		gen.Const("ghp_mixedCASE123token456TEST"),
		gen.Const("ghp_shorttoken"),
		gen.Const("ghp_verylongtokenwithlotsofrandomcharacters1234567890abcdefghijklmnopqrstuvwxyz"),
		// Traditional passwords
		gen.Const("password123"),
		gen.Const("MySecureP@ssw0rd"),
		gen.Const("simple"),
		// Passwords with special characters
		gen.Const("p@ssw0rd!"),
		gen.Const("test#pass$123"),
		gen.Const("complex&Pass*456"),
		// Long passwords
		gen.Const("this-is-a-very-long-password-with-many-characters-1234567890"),
		// Passwords with spaces (edge case)
		gen.Const("pass word"),
		gen.Const("my secret token"),
	)
}

// Edge Case Tests for Task 2.7

// TestSave_SpecialCharactersInURL tests saving credentials with special characters in the URL
func TestSave_SpecialCharactersInURL(t *testing.T) {
	testCases := []struct {
		name string
		url  string
	}{
		{"URL with port", "https://git.example.com:8443/user/repo.git"},
		{"URL with hyphen", "https://github.com/user/my-repo.git"},
		{"URL with underscore", "https://github.com/user/my_repo.git"},
		{"URL with numbers", "https://github.com/user/repo-123.git"},
		{"URL with subgroups", "https://gitlab.com/group/subgroup/repo.git"},
		{"URL with dots", "https://git.company.com/user/repo.v2.git"},
		{"URL without .git suffix", "https://github.com/user/repo"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "git-creds-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create .git directory
			gitDir := filepath.Join(tempDir, ".git")
			if err := os.Mkdir(gitDir, 0755); err != nil {
				t.Fatalf("Failed to create .git dir: %v", err)
			}

			// Create initial config file
			configPath := filepath.Join(gitDir, "config")
			initialConfig := `[core]
	repositoryformatversion = 0
`
			if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
				t.Fatalf("Failed to create initial config: %v", err)
			}

			// Create store and save credentials
			store := NewStore(tempDir)
			creds := &Credentials{
				URL:      tc.url,
				Username: "testuser",
				Password: "testpass",
			}

			if err := store.Save(creds); err != nil {
				t.Fatalf("Save failed for URL %q: %v", tc.url, err)
			}

			// Load and verify
			loadedCreds, err := store.Load()
			if err != nil {
				t.Fatalf("Load failed: %v", err)
			}

			if loadedCreds.URL != tc.url {
				t.Errorf("URL mismatch: expected %q, got %q", tc.url, loadedCreds.URL)
			}
		})
	}
}

// TestSave_SpecialCharactersInUsername tests saving credentials with special characters in username
func TestSave_SpecialCharactersInUsername(t *testing.T) {
	testCases := []struct {
		name     string
		username string
	}{
		{"Username with hyphen", "test-user"},
		{"Username with underscore", "test_user"},
		{"Username with numbers", "user123"},
		{"Username with dots", "user.name"},
		{"Email as username", "user@example.com"},
		{"Username with plus", "user+tag@example.com"},
		{"Mixed case username", "TestUser"},
		{"Username with multiple hyphens", "my-test-user-account"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "git-creds-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create .git directory
			gitDir := filepath.Join(tempDir, ".git")
			if err := os.Mkdir(gitDir, 0755); err != nil {
				t.Fatalf("Failed to create .git dir: %v", err)
			}

			// Create initial config file
			configPath := filepath.Join(gitDir, "config")
			initialConfig := `[core]
	repositoryformatversion = 0
`
			if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
				t.Fatalf("Failed to create initial config: %v", err)
			}

			// Create store and save credentials
			store := NewStore(tempDir)
			creds := &Credentials{
				URL:      "https://github.com/user/repo.git",
				Username: tc.username,
				Password: "testpass",
			}

			if err := store.Save(creds); err != nil {
				t.Fatalf("Save failed for username %q: %v", tc.username, err)
			}

			// Load and verify
			loadedCreds, err := store.Load()
			if err != nil {
				t.Fatalf("Load failed: %v", err)
			}

			if loadedCreds.Username != tc.username {
				t.Errorf("Username mismatch: expected %q, got %q", tc.username, loadedCreds.Username)
			}
		})
	}
}

// TestSave_SpecialCharactersInPassword tests saving credentials with special characters in password
func TestSave_SpecialCharactersInPassword(t *testing.T) {
	testCases := []struct {
		name     string
		password string
	}{
		{"Password with special chars", "p@ssw0rd!"},
		{"Password with hash", "test#pass$123"},
		{"Password with ampersand", "complex&Pass*456"},
		{"Password with spaces", "pass word with spaces"},
		{"Password with equals", "pass=word"},
		{"Password with quotes", "pass\"word"},
		{"Password with backslash", "pass\\word"},
		// Note: Newlines in passwords would break the config file format, so we skip that test
		{"Password with tab escaped", "pass\tword"},
		{"GitHub PAT with special chars", "ghp_1234567890abcdefGHIJKLMNOP"},
		{"Very long password", "this-is-a-very-long-password-with-many-characters-1234567890-abcdefghijklmnopqrstuvwxyz-ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{"Password with brackets", "pass[word]"},
		{"Password with braces", "pass{word}"},
		{"Password with parentheses", "pass(word)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "git-creds-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create .git directory
			gitDir := filepath.Join(tempDir, ".git")
			if err := os.Mkdir(gitDir, 0755); err != nil {
				t.Fatalf("Failed to create .git dir: %v", err)
			}

			// Create initial config file
			configPath := filepath.Join(gitDir, "config")
			initialConfig := `[core]
	repositoryformatversion = 0
`
			if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
				t.Fatalf("Failed to create initial config: %v", err)
			}

			// Create store and save credentials
			store := NewStore(tempDir)
			creds := &Credentials{
				URL:      "https://github.com/user/repo.git",
				Username: "testuser",
				Password: tc.password,
			}

			if err := store.Save(creds); err != nil {
				t.Fatalf("Save failed for password %q: %v", tc.password, err)
			}

			// Load and verify
			loadedCreds, err := store.Load()
			if err != nil {
				t.Fatalf("Load failed: %v", err)
			}

			if loadedCreds.Password != tc.password {
				t.Errorf("Password mismatch: expected %q, got %q", tc.password, loadedCreds.Password)
			}
		})
	}
}

// TestSave_EmptyCredentialFields tests saving credentials with empty fields
// Note: The Load method requires all fields to be non-empty, so we only test that Save succeeds
func TestSave_EmptyCredentialFields(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		username string
		password string
	}{
		{"Empty URL", "", "user", "pass"},
		{"Empty username", "https://github.com/user/repo.git", "", "pass"},
		{"Empty password", "https://github.com/user/repo.git", "user", ""},
		{"All empty", "", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "git-creds-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create .git directory
			gitDir := filepath.Join(tempDir, ".git")
			if err := os.Mkdir(gitDir, 0755); err != nil {
				t.Fatalf("Failed to create .git dir: %v", err)
			}

			// Create initial config file
			configPath := filepath.Join(gitDir, "config")
			initialConfig := `[core]
	repositoryformatversion = 0
`
			if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
				t.Fatalf("Failed to create initial config: %v", err)
			}

			// Create store and save credentials (should succeed even with empty fields)
			store := NewStore(tempDir)
			creds := &Credentials{
				URL:      tc.url,
				Username: tc.username,
				Password: tc.password,
			}

			if err := store.Save(creds); err != nil {
				t.Fatalf("Save failed: %v", err)
			}

			// Verify the config file was written
			content, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("Failed to read config file: %v", err)
			}

			configStr := string(content)
			
			// Verify credential section exists (even if URL is empty)
			if !strings.Contains(configStr, "[credential") {
				t.Errorf("Config should contain credential section. Got:\n%s", configStr)
			}
			
			// Note: Load will fail for empty fields, which is expected behavior
			// The Load method requires all fields to be non-empty to return credentials
		})
	}
}

// TestLoad_CorruptedConfigFile tests loading from a corrupted config file
func TestLoad_CorruptedConfigFile(t *testing.T) {
	testCases := []struct {
		name          string
		configContent string
	}{
		{
			"Malformed credential section",
			`[core]
	repositoryformatversion = 0
[credential "https://github.com/user/repo.git"
	username = testuser
	password = testpass
`,
		},
		{
			"Missing closing bracket",
			`[core]
	repositoryformatversion = 0
[credential "https://github.com/user/repo.git"
	username = testuser
`,
		},
		{
			"Credential section with only username",
			`[core]
	repositoryformatversion = 0
[credential "https://github.com/user/repo.git"]
	username = testuser
`,
		},
		{
			"Credential section with only password",
			`[core]
	repositoryformatversion = 0
[credential "https://github.com/user/repo.git"]
	password = testpass
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "git-creds-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create .git directory
			gitDir := filepath.Join(tempDir, ".git")
			if err := os.Mkdir(gitDir, 0755); err != nil {
				t.Fatalf("Failed to create .git dir: %v", err)
			}

			// Create corrupted config file
			configPath := filepath.Join(gitDir, "config")
			if err := os.WriteFile(configPath, []byte(tc.configContent), 0600); err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			// Load should fail gracefully
			store := NewStore(tempDir)
			_, err = store.Load()
			if err == nil {
				t.Error("Expected error when loading from corrupted config file")
			}
			if !strings.Contains(err.Error(), "no credentials found") {
				t.Errorf("Expected 'no credentials found' error, got: %v", err)
			}
		})
	}
}

// TestClear_GracefulHandling tests that Clear handles edge cases gracefully
func TestClear_GracefulHandling(t *testing.T) {
	testCases := []struct {
		name      string
		setupFunc func(string) error
	}{
		{
			"No .git directory",
			func(tempDir string) error {
				// Don't create .git directory
				return nil
			},
		},
		{
			"Empty .git directory",
			func(tempDir string) error {
				gitDir := filepath.Join(tempDir, ".git")
				return os.Mkdir(gitDir, 0755)
			},
		},
		// Note: .git as a file (worktree reference) is a valid Git scenario
		// but our implementation expects a directory, so we skip that edge case
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir, err := os.MkdirTemp("", "git-creds-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test scenario
			if err := tc.setupFunc(tempDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Clear should handle gracefully without error
			store := NewStore(tempDir)
			err = store.Clear()
			if err != nil {
				t.Errorf("Clear should handle %s gracefully, got error: %v", tc.name, err)
			}
		})
	}
}

// TestSave_ConcurrentAccess tests that saving credentials is safe (basic test)
func TestSave_ConcurrentAccess(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-creds-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Create initial config file
	configPath := filepath.Join(gitDir, "config")
	initialConfig := `[core]
	repositoryformatversion = 0
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create initial config: %v", err)
	}

	// Save multiple credentials sequentially (simulating rapid updates)
	store := NewStore(tempDir)
	for i := 0; i < 5; i++ {
		creds := &Credentials{
			URL:      "https://github.com/user/repo.git",
			Username: fmt.Sprintf("user%d", i),
			Password: fmt.Sprintf("pass%d", i),
		}

		if err := store.Save(creds); err != nil {
			t.Fatalf("Save %d failed: %v", i, err)
		}
	}

	// Load and verify we got the last saved credentials
	loadedCreds, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loadedCreds.Username != "user4" {
		t.Errorf("Expected last saved username 'user4', got %q", loadedCreds.Username)
	}
	if loadedCreds.Password != "pass4" {
		t.Errorf("Expected last saved password 'pass4', got %q", loadedCreds.Password)
	}
}
