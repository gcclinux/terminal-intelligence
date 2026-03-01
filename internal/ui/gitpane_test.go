package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/user/terminal-intelligence/internal/git"
)

// TestToggleVisibility verifies that the Toggle method correctly toggles the visible state
// and clears messages when closing.
//
// Requirements: 1.1, 1.2
func TestToggleVisibility(t *testing.T) {
	// Create a GitPane instance
	pane := &GitPane{
		visible:       false,
		statusMessage: "some status",
		errorMessage:  "some error",
	}

	// Toggle to open (false -> true)
	cmd := pane.Toggle()
	if !pane.visible {
		t.Errorf("expected visible to be true after toggle, got false")
	}
	// When opening, should return a command for repository detection
	if cmd == nil {
		t.Errorf("expected non-nil command when opening pane")
	}

	// Toggle to close (true -> false)
	cmd = pane.Toggle()
	if pane.visible {
		t.Errorf("expected visible to be false after toggle, got true")
	}
	// When closing, should clear status messages
	if pane.statusMessage != "" {
		t.Errorf("expected statusMessage to be cleared when closing, got %q", pane.statusMessage)
	}
	if pane.errorMessage != "" {
		t.Errorf("expected errorMessage to be cleared when closing, got %q", pane.errorMessage)
	}
	// When closing, should return nil command
	if cmd != nil {
		t.Errorf("expected nil command when closing pane, got non-nil")
	}
}

// TestIsVisible verifies that the IsVisible method returns the correct visibility state.
func TestIsVisible(t *testing.T) {
	pane := &GitPane{visible: false}
	if pane.IsVisible() {
		t.Errorf("expected IsVisible to return false, got true")
	}

	pane.visible = true
	if !pane.IsVisible() {
		t.Errorf("expected IsVisible to return true, got false")
	}
}

// TestDetectRepositoryWithNilClient verifies that detectRepository handles nil gitClient gracefully.
//
// Requirements: 2.1, 2.2, 2.3
func TestDetectRepositoryWithNilClient(t *testing.T) {
	pane := &GitPane{
		gitClient: nil,
	}

	cmd := pane.detectRepository()
	if cmd == nil {
		t.Fatalf("expected non-nil command from detectRepository")
	}

	// Execute the command to get the message
	msg := cmd()
	detectedMsg, ok := msg.(GitRepositoryDetectedMsg)
	if !ok {
		t.Fatalf("expected GitRepositoryDetectedMsg, got %T", msg)
	}

	// Should return empty result when gitClient is nil
	if detectedMsg.IsRepo {
		t.Errorf("expected IsRepo to be false with nil client, got true")
	}
	if detectedMsg.Credentials != nil {
		t.Errorf("expected Credentials to be nil with nil client, got %v", detectedMsg.Credentials)
	}
	if detectedMsg.RemoteURL != "" {
		t.Errorf("expected RemoteURL to be empty with nil client, got %q", detectedMsg.RemoteURL)
	}
}

// TestDetectRepositoryWithClient verifies that detectRepository calls GitClient.DetectRepository
// and returns the correct message.
//
// Requirements: 2.1, 2.2, 2.3
func TestDetectRepositoryWithClient(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a GitClient for the temp directory
	client := git.NewClient(tmpDir)

	pane := &GitPane{
		gitClient: client,
		workDir:   tmpDir,
	}

	cmd := pane.detectRepository()
	if cmd == nil {
		t.Fatalf("expected non-nil command from detectRepository")
	}

	// Execute the command to get the message
	msg := cmd()
	detectedMsg, ok := msg.(GitRepositoryDetectedMsg)
	if !ok {
		t.Fatalf("expected GitRepositoryDetectedMsg, got %T", msg)
	}

	// Since tmpDir is not a git repository, IsRepo should be false
	if detectedMsg.IsRepo {
		t.Errorf("expected IsRepo to be false for non-repo directory, got true")
	}
	if detectedMsg.Credentials != nil {
		t.Errorf("expected Credentials to be nil for non-repo directory, got %v", detectedMsg.Credentials)
	}
	if detectedMsg.RemoteURL != "" {
		t.Errorf("expected RemoteURL to be empty for non-repo directory, got %q", detectedMsg.RemoteURL)
	}
}

// Feature: git-integration, Property 1: UI Toggle Behavior
// **Validates: Requirements 1.1, 1.2**
//
// For any UI state (visible or hidden), pressing Ctrl+G should toggle the Git UI
// to the opposite state.
func TestProperty1_UIToggleBehavior(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("toggling UI state produces opposite visibility", prop.ForAll(
		func(initialVisible bool) bool {
			// Create a GitPane with the initial visibility state
			pane := &GitPane{
				visible:       initialVisible,
				statusMessage: "test status",
				errorMessage:  "test error",
			}

			// Toggle the pane
			_ = pane.Toggle()

			// Verify the visibility is now the opposite of the initial state
			if pane.visible != !initialVisible {
				t.Logf("Expected visible=%v after toggle, got visible=%v", !initialVisible, pane.visible)
				return false
			}

			// When closing (initial was true), messages should be cleared
			if initialVisible && (pane.statusMessage != "" || pane.errorMessage != "") {
				t.Logf("Expected messages to be cleared when closing, got status=%q, error=%q",
					pane.statusMessage, pane.errorMessage)
				return false
			}

			return true
		},
		gen.Bool(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestUpdateHandlesRepositoryDetectedMsg verifies that the Update method correctly
// handles GitRepositoryDetectedMsg and populates input fields with detected credentials.
//
// Requirements: 2.1, 2.2, 2.3
func TestUpdateHandlesRepositoryDetectedMsg(t *testing.T) {
	tests := []struct {
		name        string
		msg         GitRepositoryDetectedMsg
		expectURL   string
		expectUser  string
		expectPass  string
	}{
		{
			name: "repository with credentials",
			msg: GitRepositoryDetectedMsg{
				IsRepo: true,
				Credentials: &git.Credentials{
					URL:      "https://github.com/user/repo",
					Username: "testuser",
					Password: "ghp_testtoken123",
				},
				RemoteURL: "https://github.com/user/repo",
			},
			expectURL:  "https://github.com/user/repo",
			expectUser: "testuser",
			expectPass: "ghp_testtoken123",
		},
		{
			name: "repository without credentials",
			msg: GitRepositoryDetectedMsg{
				IsRepo:      true,
				Credentials: nil,
				RemoteURL:   "https://github.com/user/repo",
			},
			expectURL:  "https://github.com/user/repo",
			expectUser: "",
			expectPass: "",
		},
		{
			name: "not a repository",
			msg: GitRepositoryDetectedMsg{
				IsRepo:      false,
				Credentials: nil,
				RemoteURL:   "",
			},
			expectURL:  "",
			expectUser: "",
			expectPass: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a GitPane and initialize it
			pane := &GitPane{}
			pane.Init()

			// Set some initial values to verify they get cleared/updated
			pane.urlInput.SetValue("old-url")
			pane.userInput.SetValue("old-user")
			pane.passInput.SetValue("old-pass")

			// Call Update with the GitRepositoryDetectedMsg
			_, _ = pane.Update(tt.msg)

			// Verify the input fields were updated correctly
			if pane.urlInput.Value() != tt.expectURL {
				t.Errorf("expected URL %q, got %q", tt.expectURL, pane.urlInput.Value())
			}
			if pane.userInput.Value() != tt.expectUser {
				t.Errorf("expected User %q, got %q", tt.expectUser, pane.userInput.Value())
			}
			if pane.passInput.Value() != tt.expectPass {
				t.Errorf("expected Pass %q, got %q", tt.expectPass, pane.passInput.Value())
			}
		})
	}
}

// Feature: git-integration, Property 3: Repository Detection on Open
// **Validates: Requirements 2.1, 2.2, 2.3**
//
// For any directory, when the Git UI opens, the system should check for a .git subdirectory
// and populate credentials if found, or leave fields empty if not found.
func TestProperty3_RepositoryDetectionOnOpen(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("opening UI in non-repo directory leaves fields empty", prop.ForAll(
		func() bool {
			// Create a temporary non-repo directory
			tmpDir := t.TempDir()

			// Create a GitClient for the temp directory
			client := git.NewClient(tmpDir)

			// Create a GitPane and initialize it
			pane := &GitPane{
				gitClient: client,
				workDir:   tmpDir,
			}
			pane.Init()

			// Set some initial values to verify they get cleared
			pane.urlInput.SetValue("old-url")
			pane.userInput.SetValue("old-user")
			pane.passInput.SetValue("old-pass")

			// Trigger repository detection (simulating opening the UI)
			cmd := pane.detectRepository()
			if cmd == nil {
				t.Logf("Expected non-nil command from detectRepository")
				return false
			}

			// Execute the command to get the message
			msg := cmd()
			detectedMsg, ok := msg.(GitRepositoryDetectedMsg)
			if !ok {
				t.Logf("Expected GitRepositoryDetectedMsg, got %T", msg)
				return false
			}

			// Process the message through Update
			_, _ = pane.Update(detectedMsg)

			// Verify fields are empty for non-repo directory
			if pane.urlInput.Value() != "" {
				t.Logf("Expected empty URL for non-repo, got %q", pane.urlInput.Value())
				return false
			}
			if pane.userInput.Value() != "" {
				t.Logf("Expected empty User for non-repo, got %q", pane.userInput.Value())
				return false
			}
			if pane.passInput.Value() != "" {
				t.Logf("Expected empty Pass for non-repo, got %q", pane.passInput.Value())
				return false
			}

			return true
		},
	))

	properties.Property("opening UI in repo with credentials populates all fields", prop.ForAll(
		func(repoNum uint8, userNum uint8, passNum uint8) bool {
			// Create valid values from numbers to ensure non-empty strings
			urlSuffix := fmt.Sprintf("repo%d", repoNum)
			username := fmt.Sprintf("user%d", userNum)
			password := fmt.Sprintf("pass%d", passNum)
			url := "https://github.com/user/" + urlSuffix

			// Create a temporary directory and initialize a git repo
			tmpDir := t.TempDir()

			// Initialize a git repository
			repo, err := gogit.PlainInit(tmpDir, false)
			if err != nil {
				t.Logf("Failed to initialize git repo: %v", err)
				return false
			}

			// Create a remote configuration
			_, err = repo.CreateRemote(&config.RemoteConfig{
				Name: "origin",
				URLs: []string{url},
			})
			if err != nil {
				t.Logf("Failed to create remote: %v", err)
				return false
			}

			// Save credentials to the repository
			store := git.NewStore(tmpDir)
			err = store.Save(&git.Credentials{
				URL:      url,
				Username: username,
				Password: password,
			})
			if err != nil {
				t.Logf("Failed to save credentials: %v", err)
				return false
			}

			// Create a GitClient for the temp directory
			client := git.NewClient(tmpDir)

			// Create a GitPane and initialize it
			pane := &GitPane{
				gitClient: client,
				workDir:   tmpDir,
			}
			pane.Init()

			// Trigger repository detection (simulating opening the UI)
			cmd := pane.detectRepository()
			if cmd == nil {
				t.Logf("Expected non-nil command from detectRepository")
				return false
			}

			// Execute the command to get the message
			msg := cmd()
			detectedMsg, ok := msg.(GitRepositoryDetectedMsg)
			if !ok {
				t.Logf("Expected GitRepositoryDetectedMsg, got %T", msg)
				return false
			}

			// Process the message through Update
			_, _ = pane.Update(detectedMsg)

			// Verify fields are populated with the saved credentials
			if pane.urlInput.Value() != url {
				t.Logf("Expected URL %q, got %q", url, pane.urlInput.Value())
				return false
			}
			if pane.userInput.Value() != username {
				t.Logf("Expected User %q, got %q", username, pane.userInput.Value())
				return false
			}
			if pane.passInput.Value() != password {
				t.Logf("Expected Pass %q, got %q", password, pane.passInput.Value())
				return false
			}

			return true
		},
		gen.UInt8(),
		gen.UInt8(),
		gen.UInt8(),
	))

	properties.Property("opening UI in repo without credentials populates only URL", prop.ForAll(
		func(repoNum uint8) bool {
			// Create valid URL from number
			urlSuffix := fmt.Sprintf("repo%d", repoNum)
			url := "https://github.com/user/" + urlSuffix

			// Create a temporary directory and initialize a git repo
			tmpDir := t.TempDir()

			// Initialize a git repository
			repo, err := gogit.PlainInit(tmpDir, false)
			if err != nil {
				t.Logf("Failed to initialize git repo: %v", err)
				return false
			}

			// Create a remote configuration
			_, err = repo.CreateRemote(&config.RemoteConfig{
				Name: "origin",
				URLs: []string{url},
			})
			if err != nil {
				t.Logf("Failed to create remote: %v", err)
				return false
			}

			// Do NOT save credentials - this tests the case where repo exists but no credentials

			// Create a GitClient for the temp directory
			client := git.NewClient(tmpDir)

			// Create a GitPane and initialize it
			pane := &GitPane{
				gitClient: client,
				workDir:   tmpDir,
			}
			pane.Init()

			// Set some initial values to verify they get cleared
			pane.userInput.SetValue("old-user")
			pane.passInput.SetValue("old-pass")

			// Trigger repository detection (simulating opening the UI)
			cmd := pane.detectRepository()
			if cmd == nil {
				t.Logf("Expected non-nil command from detectRepository")
				return false
			}

			// Execute the command to get the message
			msg := cmd()
			detectedMsg, ok := msg.(GitRepositoryDetectedMsg)
			if !ok {
				t.Logf("Expected GitRepositoryDetectedMsg, got %T", msg)
				return false
			}

			// Process the message through Update
			_, _ = pane.Update(detectedMsg)

			// Verify URL is populated but credentials are empty
			if pane.urlInput.Value() != url {
				t.Logf("Expected URL %q, got %q", url, pane.urlInput.Value())
				return false
			}
			if pane.userInput.Value() != "" {
				t.Logf("Expected empty User for repo without credentials, got %q", pane.userInput.Value())
				return false
			}
			if pane.passInput.Value() != "" {
				t.Logf("Expected empty Pass for repo without credentials, got %q", pane.passInput.Value())
				return false
			}

			return true
		},
		gen.UInt8(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestKeyboardNavigation_Tab verifies that Tab key cycles through input fields and buttons.
//
// Requirements: 3.1, 3.2, 3.3, 3.4, 3.5
func TestKeyboardNavigation_Tab(t *testing.T) {
	// Create and initialize a GitPane
	pane := &GitPane{}
	pane.Init()

	// Initial focus should be on URL input (focusedInput = 0)
	if pane.focusedInput != 0 {
		t.Errorf("expected initial focusedInput to be 0, got %d", pane.focusedInput)
	}
	if !pane.urlInput.Focused() {
		t.Errorf("expected URL input to be focused initially")
	}

	// Press Tab - should move to USER input (focusedInput = 1)
	pane.Update(tea.KeyMsg{Type: tea.KeyTab})
	if pane.focusedInput != 1 {
		t.Errorf("expected focusedInput to be 1 after Tab, got %d", pane.focusedInput)
	}
	if !pane.userInput.Focused() {
		t.Errorf("expected USER input to be focused after Tab")
	}

	// Press Tab - should move to PASS input (focusedInput = 2)
	pane.Update(tea.KeyMsg{Type: tea.KeyTab})
	if pane.focusedInput != 2 {
		t.Errorf("expected focusedInput to be 2 after second Tab, got %d", pane.focusedInput)
	}
	if !pane.passInput.Focused() {
		t.Errorf("expected PASS input to be focused after second Tab")
	}

	// Press Tab - should move to buttons (focusedInput = 3)
	pane.Update(tea.KeyMsg{Type: tea.KeyTab})
	if pane.focusedInput != 3 {
		t.Errorf("expected focusedInput to be 3 after third Tab, got %d", pane.focusedInput)
	}
	// All inputs should be blurred when focused on buttons
	if pane.urlInput.Focused() || pane.userInput.Focused() || pane.passInput.Focused() {
		t.Errorf("expected all inputs to be blurred when focused on buttons")
	}

	// Press Tab - should wrap back to URL input (focusedInput = 0)
	pane.Update(tea.KeyMsg{Type: tea.KeyTab})
	if pane.focusedInput != 0 {
		t.Errorf("expected focusedInput to wrap to 0 after fourth Tab, got %d", pane.focusedInput)
	}
	if !pane.urlInput.Focused() {
		t.Errorf("expected URL input to be focused after wrapping")
	}
}

// TestKeyboardNavigation_UpDown verifies that Up/Down arrow keys navigate between fields.
//
// Requirements: 3.1, 3.2, 3.3, 3.4, 3.5
func TestKeyboardNavigation_UpDown(t *testing.T) {
	// Create and initialize a GitPane
	pane := &GitPane{}
	pane.Init()

	// Initial focus should be on URL input (focusedInput = 0)
	if pane.focusedInput != 0 {
		t.Errorf("expected initial focusedInput to be 0, got %d", pane.focusedInput)
	}

	// Press Down - should move to USER input (focusedInput = 1)
	pane.Update(tea.KeyMsg{Type: tea.KeyDown})
	if pane.focusedInput != 1 {
		t.Errorf("expected focusedInput to be 1 after Down, got %d", pane.focusedInput)
	}
	if !pane.userInput.Focused() {
		t.Errorf("expected USER input to be focused after Down")
	}

	// Press Down - should move to PASS input (focusedInput = 2)
	pane.Update(tea.KeyMsg{Type: tea.KeyDown})
	if pane.focusedInput != 2 {
		t.Errorf("expected focusedInput to be 2 after second Down, got %d", pane.focusedInput)
	}
	if !pane.passInput.Focused() {
		t.Errorf("expected PASS input to be focused after second Down")
	}

	// Press Down - should move to buttons (focusedInput = 3)
	pane.Update(tea.KeyMsg{Type: tea.KeyDown})
	if pane.focusedInput != 3 {
		t.Errorf("expected focusedInput to be 3 after third Down, got %d", pane.focusedInput)
	}

	// Press Up - should move back to PASS input (focusedInput = 2)
	pane.Update(tea.KeyMsg{Type: tea.KeyUp})
	if pane.focusedInput != 2 {
		t.Errorf("expected focusedInput to be 2 after Up, got %d", pane.focusedInput)
	}
	if !pane.passInput.Focused() {
		t.Errorf("expected PASS input to be focused after Up")
	}

	// Press Up - should move to USER input (focusedInput = 1)
	pane.Update(tea.KeyMsg{Type: tea.KeyUp})
	if pane.focusedInput != 1 {
		t.Errorf("expected focusedInput to be 1 after second Up, got %d", pane.focusedInput)
	}
	if !pane.userInput.Focused() {
		t.Errorf("expected USER input to be focused after second Up")
	}

	// Press Up - should move to URL input (focusedInput = 0)
	pane.Update(tea.KeyMsg{Type: tea.KeyUp})
	if pane.focusedInput != 0 {
		t.Errorf("expected focusedInput to be 0 after third Up, got %d", pane.focusedInput)
	}
	if !pane.urlInput.Focused() {
		t.Errorf("expected URL input to be focused after third Up")
	}

	// Press Up - should wrap to buttons (focusedInput = 3)
	pane.Update(tea.KeyMsg{Type: tea.KeyUp})
	if pane.focusedInput != 3 {
		t.Errorf("expected focusedInput to wrap to 3 after Up from URL, got %d", pane.focusedInput)
	}
}

// TestKeyboardNavigation_ShiftTab verifies that Shift+Tab cycles backward through fields.
//
// Requirements: 3.1, 3.2, 3.3, 3.4, 3.5
func TestKeyboardNavigation_ShiftTab(t *testing.T) {
	// Create and initialize a GitPane
	pane := &GitPane{}
	pane.Init()

	// Initial focus should be on URL input (focusedInput = 0)
	if pane.focusedInput != 0 {
		t.Errorf("expected initial focusedInput to be 0, got %d", pane.focusedInput)
	}

	// Press Shift+Tab - should wrap to buttons (focusedInput = 3)
	pane.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if pane.focusedInput != 3 {
		t.Errorf("expected focusedInput to wrap to 3 after Shift+Tab, got %d", pane.focusedInput)
	}

	// Press Shift+Tab - should move to PASS input (focusedInput = 2)
	pane.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if pane.focusedInput != 2 {
		t.Errorf("expected focusedInput to be 2 after second Shift+Tab, got %d", pane.focusedInput)
	}
	if !pane.passInput.Focused() {
		t.Errorf("expected PASS input to be focused after second Shift+Tab")
	}

	// Press Shift+Tab - should move to USER input (focusedInput = 1)
	pane.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if pane.focusedInput != 1 {
		t.Errorf("expected focusedInput to be 1 after third Shift+Tab, got %d", pane.focusedInput)
	}
	if !pane.userInput.Focused() {
		t.Errorf("expected USER input to be focused after third Shift+Tab")
	}

	// Press Shift+Tab - should move to URL input (focusedInput = 0)
	pane.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if pane.focusedInput != 0 {
		t.Errorf("expected focusedInput to be 0 after fourth Shift+Tab, got %d", pane.focusedInput)
	}
	if !pane.urlInput.Focused() {
		t.Errorf("expected URL input to be focused after fourth Shift+Tab")
	}
}

// TestKeyboardNavigation_Enter verifies that Enter key only activates buttons, not moves between fields.
//
// Requirements: 3.1, 3.2, 3.3, 3.4, 3.5
func TestKeyboardNavigation_Enter(t *testing.T) {
	// Create and initialize a GitPane
	pane := &GitPane{}
	pane.Init()

	// Initial focus should be on URL input (focusedInput = 0)
	if pane.focusedInput != 0 {
		t.Errorf("expected initial focusedInput to be 0, got %d", pane.focusedInput)
	}

	// Press Enter on URL input - should NOT move to next field
	pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if pane.focusedInput != 0 {
		t.Errorf("expected focusedInput to remain 0 after Enter on URL, got %d", pane.focusedInput)
	}
	if !pane.urlInput.Focused() {
		t.Errorf("expected URL input to remain focused after Enter")
	}

	// Use Tab to move to buttons
	pane.Update(tea.KeyMsg{Type: tea.KeyTab}) // URL -> USER
	pane.Update(tea.KeyMsg{Type: tea.KeyTab}) // USER -> PASS
	pane.Update(tea.KeyMsg{Type: tea.KeyTab}) // PASS -> buttons

	if pane.focusedInput != 3 {
		t.Errorf("expected focusedInput to be 3 after navigating to buttons, got %d", pane.focusedInput)
	}

	// Press Enter on buttons - should trigger button action (tested in other tests)
	// For this test, we just verify the focus doesn't change
	initialFocus := pane.focusedInput
	pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if pane.focusedInput != initialFocus {
		t.Errorf("expected focusedInput to remain %d after Enter on buttons, got %d", initialFocus, pane.focusedInput)
	}
}

// TestKeyboardNavigation_Esc verifies that Esc key closes the popup.
//
// Requirements: 3.1, 3.2, 3.3, 3.4, 3.5
func TestKeyboardNavigation_Esc(t *testing.T) {
	// Create and initialize a GitPane
	pane := &GitPane{
		visible:       true,
		statusMessage: "test status",
		errorMessage:  "test error",
	}
	pane.Init()

	// Press Esc - should close the popup and clear messages
	pane.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if pane.visible {
		t.Errorf("expected visible to be false after Esc, got true")
	}
	if pane.statusMessage != "" {
		t.Errorf("expected statusMessage to be cleared after Esc, got %q", pane.statusMessage)
	}
	if pane.errorMessage != "" {
		t.Errorf("expected errorMessage to be cleared after Esc, got %q", pane.errorMessage)
	}
}

// TestKeyboardNavigation_ArrowKeys verifies that arrow keys navigate between buttons.
//
// Requirements: 3.1, 3.2, 3.3, 3.4, 3.5
func TestKeyboardNavigation_ArrowKeys(t *testing.T) {
	// Create and initialize a GitPane
	pane := &GitPane{}
	pane.Init()

	// Move focus to buttons (focusedInput = 3)
	pane.focusedInput = 3
	pane.updateFocus()

	// Initial selectedButton should be 0 (Clone)
	if pane.selectedButton != 0 {
		t.Errorf("expected initial selectedButton to be 0, got %d", pane.selectedButton)
	}

	// Press Right arrow - should move to button 1 (Pull)
	pane.Update(tea.KeyMsg{Type: tea.KeyRight})
	if pane.selectedButton != 1 {
		t.Errorf("expected selectedButton to be 1 after Right, got %d", pane.selectedButton)
	}

	// Press Right arrow - should move to button 2 (Push)
	pane.Update(tea.KeyMsg{Type: tea.KeyRight})
	if pane.selectedButton != 2 {
		t.Errorf("expected selectedButton to be 2 after second Right, got %d", pane.selectedButton)
	}

	// Press Left arrow - should move back to button 1 (Pull)
	pane.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if pane.selectedButton != 1 {
		t.Errorf("expected selectedButton to be 1 after Left, got %d", pane.selectedButton)
	}

	// Move to last button (Restore = 6)
	pane.selectedButton = 6

	// Press Right arrow - should wrap to button 0 (Clone)
	pane.Update(tea.KeyMsg{Type: tea.KeyRight})
	if pane.selectedButton != 0 {
		t.Errorf("expected selectedButton to wrap to 0 after Right on last button, got %d", pane.selectedButton)
	}

	// Press Left arrow - should wrap to button 6 (Restore)
	pane.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if pane.selectedButton != 6 {
		t.Errorf("expected selectedButton to wrap to 6 after Left on first button, got %d", pane.selectedButton)
	}
}

// TestKeyboardNavigation_ArrowKeysOnlyWorkOnButtons verifies that arrow keys
// only navigate buttons when focused on buttons, not when focused on input fields.
//
// Requirements: 3.1, 3.2, 3.3, 3.4, 3.5
func TestKeyboardNavigation_ArrowKeysOnlyWorkOnButtons(t *testing.T) {
	// Create and initialize a GitPane
	pane := &GitPane{}
	pane.Init()

	// Focus should be on URL input (focusedInput = 0)
	if pane.focusedInput != 0 {
		t.Errorf("expected initial focusedInput to be 0, got %d", pane.focusedInput)
	}

	// Set selectedButton to 3 to verify it doesn't change
	pane.selectedButton = 3

	// Press Right arrow while focused on input - should NOT change selectedButton
	pane.Update(tea.KeyMsg{Type: tea.KeyRight})
	if pane.selectedButton != 3 {
		t.Errorf("expected selectedButton to remain 3 when focused on input, got %d", pane.selectedButton)
	}

	// Press Left arrow while focused on input - should NOT change selectedButton
	pane.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if pane.selectedButton != 3 {
		t.Errorf("expected selectedButton to remain 3 when focused on input, got %d", pane.selectedButton)
	}

	// Move focus to buttons
	pane.focusedInput = 3
	pane.updateFocus()

	// Now arrow keys should work
	pane.Update(tea.KeyMsg{Type: tea.KeyRight})
	if pane.selectedButton != 4 {
		t.Errorf("expected selectedButton to be 4 after Right on buttons, got %d", pane.selectedButton)
	}
}

// TestUpdateFocus verifies that updateFocus correctly manages input field focus states.
//
// Requirements: 3.1, 3.2, 3.3
func TestUpdateFocus(t *testing.T) {
	// Create and initialize a GitPane
	pane := &GitPane{}
	pane.Init()

	// Test focusing URL input (focusedInput = 0)
	pane.focusedInput = 0
	pane.updateFocus()
	if !pane.urlInput.Focused() {
		t.Errorf("expected URL input to be focused when focusedInput=0")
	}
	if pane.userInput.Focused() || pane.passInput.Focused() {
		t.Errorf("expected USER and PASS inputs to be blurred when focusedInput=0")
	}

	// Test focusing USER input (focusedInput = 1)
	pane.focusedInput = 1
	pane.updateFocus()
	if !pane.userInput.Focused() {
		t.Errorf("expected USER input to be focused when focusedInput=1")
	}
	if pane.urlInput.Focused() || pane.passInput.Focused() {
		t.Errorf("expected URL and PASS inputs to be blurred when focusedInput=1")
	}

	// Test focusing PASS input (focusedInput = 2)
	pane.focusedInput = 2
	pane.updateFocus()
	if !pane.passInput.Focused() {
		t.Errorf("expected PASS input to be focused when focusedInput=2")
	}
	if pane.urlInput.Focused() || pane.userInput.Focused() {
		t.Errorf("expected URL and USER inputs to be blurred when focusedInput=2")
	}

	// Test focusing buttons (focusedInput = 3)
	pane.focusedInput = 3
	pane.updateFocus()
	if pane.urlInput.Focused() || pane.userInput.Focused() || pane.passInput.Focused() {
		t.Errorf("expected all inputs to be blurred when focusedInput=3")
	}
}

// TestButtonActivation_SendsCorrectMessages verifies that pressing Enter on each button
// sends the correct Git operation message.
//
// Requirements: 7.1, 8.1, 9.1, 10.1, 11.1, 12.1, 13.1
func TestButtonActivation_SendsCorrectMessages(t *testing.T) {
	tests := []struct {
		name           string
		selectedButton int
		buttonName     string
		expectedMsgType string
	}{
		{name: "Clone button", selectedButton: 0, buttonName: "Clone", expectedMsgType: "GitCloneMsg"},
		{name: "Pull button", selectedButton: 1, buttonName: "Pull", expectedMsgType: "GitPullMsg"},
		{name: "Push button", selectedButton: 2, buttonName: "Push", expectedMsgType: "GitPushMsg"},
		{name: "Fetch button", selectedButton: 3, buttonName: "Fetch", expectedMsgType: "GitFetchMsg"},
		{name: "Stage button", selectedButton: 4, buttonName: "Stage", expectedMsgType: "GitStageMsg"},
		{name: "Status button", selectedButton: 5, buttonName: "Status", expectedMsgType: "GitStatusMsg"},
		{name: "Restore button", selectedButton: 6, buttonName: "Restore", expectedMsgType: "GitRestoreMsg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create and initialize a GitPane with a GitClient
			tmpDir := t.TempDir()
			client := git.NewClient(tmpDir)
			
			pane := &GitPane{
				gitClient: client,
				workDir:   tmpDir,
			}
			pane.Init()

			// Set input values
			pane.urlInput.SetValue("https://github.com/user/repo")
			pane.userInput.SetValue("testuser")
			pane.passInput.SetValue("testpass")

			// Focus on buttons and select the button to test
			pane.focusedInput = 3
			pane.selectedButton = tt.selectedButton
			pane.updateFocus()

			// Clear any previous messages
			pane.errorMessage = ""
			pane.statusMessage = ""

			// Press Enter to activate the button
			_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})

			// Verify that a command was returned
			if cmd == nil {
				t.Fatalf("expected non-nil command after pressing Enter on %s button", tt.buttonName)
			}

			// Verify that error and status messages were cleared
			if pane.errorMessage != "" {
				t.Errorf("expected errorMessage to be cleared, got %q", pane.errorMessage)
			}
			if pane.statusMessage != "" {
				t.Errorf("expected statusMessage to be cleared, got %q", pane.statusMessage)
			}

			// Verify that isProcessing flag was set
			if !pane.isProcessing {
				t.Errorf("expected isProcessing to be true after activating %s button", tt.buttonName)
			}

			// Execute the command to get the message
			msg := cmd()

			// Verify the message type
			switch tt.expectedMsgType {
			case "GitCloneMsg":
				cloneMsg, ok := msg.(GitCloneMsg)
				if !ok {
					t.Fatalf("expected GitCloneMsg, got %T", msg)
				}
				if cloneMsg.URL != "https://github.com/user/repo" {
					t.Errorf("expected URL to be passed to GitCloneMsg, got %q", cloneMsg.URL)
				}
				if cloneMsg.Username != "testuser" {
					t.Errorf("expected Username to be passed to GitCloneMsg, got %q", cloneMsg.Username)
				}
				if cloneMsg.Password != "testpass" {
					t.Errorf("expected Password to be passed to GitCloneMsg, got %q", cloneMsg.Password)
				}

			case "GitPullMsg":
				pullMsg, ok := msg.(GitPullMsg)
				if !ok {
					t.Fatalf("expected GitPullMsg, got %T", msg)
				}
				if pullMsg.Username != "testuser" {
					t.Errorf("expected Username to be passed to GitPullMsg, got %q", pullMsg.Username)
				}
				if pullMsg.Password != "testpass" {
					t.Errorf("expected Password to be passed to GitPullMsg, got %q", pullMsg.Password)
				}

			case "GitPushMsg":
				pushMsg, ok := msg.(GitPushMsg)
				if !ok {
					t.Fatalf("expected GitPushMsg, got %T", msg)
				}
				if pushMsg.Username != "testuser" {
					t.Errorf("expected Username to be passed to GitPushMsg, got %q", pushMsg.Username)
				}
				if pushMsg.Password != "testpass" {
					t.Errorf("expected Password to be passed to GitPushMsg, got %q", pushMsg.Password)
				}

			case "GitFetchMsg":
				fetchMsg, ok := msg.(GitFetchMsg)
				if !ok {
					t.Fatalf("expected GitFetchMsg, got %T", msg)
				}
				if fetchMsg.Username != "testuser" {
					t.Errorf("expected Username to be passed to GitFetchMsg, got %q", fetchMsg.Username)
				}
				if fetchMsg.Password != "testpass" {
					t.Errorf("expected Password to be passed to GitFetchMsg, got %q", fetchMsg.Password)
				}

			case "GitStageMsg":
				_, ok := msg.(GitStageMsg)
				if !ok {
					t.Fatalf("expected GitStageMsg, got %T", msg)
				}

			case "GitStatusMsg":
				_, ok := msg.(GitStatusMsg)
				if !ok {
					t.Fatalf("expected GitStatusMsg, got %T", msg)
				}

			case "GitRestoreMsg":
				_, ok := msg.(GitRestoreMsg)
				if !ok {
					t.Fatalf("expected GitRestoreMsg, got %T", msg)
				}

			default:
				t.Fatalf("unknown expected message type: %s", tt.expectedMsgType)
			}
		})
	}
}

// TestGitOperationMessages_TriggerAsyncOperations verifies that Git operation messages
// trigger async operations via GitClient and set isProcessing flag.
//
// Requirements: 7.1, 8.1, 9.1, 10.1, 11.1, 12.1, 13.1
func TestGitOperationMessages_TriggerAsyncOperations(t *testing.T) {
	// Create a temporary directory and initialize a git repo for testing
	tmpDir := t.TempDir()
	
	// Initialize a git repository
	_, err := gogit.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Create a GitClient for the temp directory
	client := git.NewClient(tmpDir)

	// Create and initialize a GitPane
	pane := &GitPane{
		gitClient: client,
		workDir:   tmpDir,
	}
	pane.Init()

	// Test GitStageMsg (simplest operation that doesn't require credentials)
	t.Run("GitStageMsg triggers Stage operation", func(t *testing.T) {
		pane.isProcessing = false
		
		// Send GitStageMsg
		_, cmd := pane.Update(GitStageMsg{})
		
		// Verify command was returned
		if cmd == nil {
			t.Fatalf("expected non-nil command from GitStageMsg")
		}
		
		// Execute the command to trigger the async operation
		result := cmd()
		
		// Verify the result is a GitOperationCompleteMsg
		completeMsg, ok := result.(GitOperationCompleteMsg)
		if !ok {
			t.Fatalf("expected GitOperationCompleteMsg, got %T", result)
		}
		
		// Verify the operation completed (success or failure doesn't matter for this test)
		if completeMsg.Operation != "stage" {
			t.Fatalf("expected operation 'stage', got %q", completeMsg.Operation)
		}
	})

	// Test GitStatusMsg
	t.Run("GitStatusMsg triggers Status operation", func(t *testing.T) {
		pane.isProcessing = false
		
		// Send GitStatusMsg
		_, cmd := pane.Update(GitStatusMsg{})
		
		// Verify command was returned
		if cmd == nil {
			t.Fatalf("expected non-nil command from GitStatusMsg")
		}
		
		// Execute the command to trigger the async operation
		result := cmd()
		
		// Verify the result is a GitOperationCompleteMsg
		completeMsg, ok := result.(GitOperationCompleteMsg)
		if !ok {
			t.Fatalf("expected GitOperationCompleteMsg, got %T", result)
		}
		
		// Verify the operation completed
		if completeMsg.Operation != "status" {
			t.Fatalf("expected operation 'status', got %q", completeMsg.Operation)
		}
		
		// Status should succeed on an initialized repo
		if !completeMsg.Success {
			t.Errorf("expected Status operation to succeed, got error: %v", completeMsg.Error)
		}
	})
}

// TestGitOperationCompleteMsg_Success verifies that GitOperationCompleteMsg with success=true
// updates statusMessage, clears errorMessage, and clears isProcessing flag.
//
// Requirements: 7.5, 8.3, 9.3, 10.3, 11.2, 13.2
func TestGitOperationCompleteMsg_Success(t *testing.T) {
	tests := []struct {
		name            string
		operation       string
		message         string
		newDir          string
		expectedStatus  string
	}{
		{
			name:           "Pull success",
			operation:      "pull",
			message:        "Pull completed successfully",
			newDir:         "",
			expectedStatus: "Pull completed successfully",
		},
		{
			name:           "Push success",
			operation:      "push",
			message:        "Push completed successfully",
			newDir:         "",
			expectedStatus: "Push completed successfully",
		},
		{
			name:           "Fetch success",
			operation:      "fetch",
			message:        "Fetch completed successfully",
			newDir:         "",
			expectedStatus: "Fetch completed successfully",
		},
		{
			name:           "Stage success",
			operation:      "stage",
			message:        "Staged 3 file(s)",
			newDir:         "",
			expectedStatus: "Staged 3 file(s)",
		},
		{
			name:           "Restore success",
			operation:      "restore",
			message:        "Restored 2 file(s)",
			newDir:         "",
			expectedStatus: "Restored 2 file(s)",
		},
		{
			name:           "Clone success",
			operation:      "clone",
			message:        "/tmp/myrepo",
			newDir:         "/tmp/myrepo",
			expectedStatus: "Clone completed successfully: /tmp/myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create and initialize a GitPane
			pane := &GitPane{}
			pane.Init()

			// Set initial state with error message and isProcessing flag
			pane.errorMessage = "previous error"
			pane.statusMessage = ""
			pane.isProcessing = true

			// Send GitOperationCompleteMsg with success
			msg := GitOperationCompleteMsg{
				Operation: tt.operation,
				Success:   true,
				Message:   tt.message,
				Error:     nil,
				NewDir:    tt.newDir,
			}

			_, _ = pane.Update(msg)

			// Verify statusMessage was updated
			if pane.statusMessage != tt.expectedStatus {
				t.Errorf("expected statusMessage %q, got %q", tt.expectedStatus, pane.statusMessage)
			}

			// Verify errorMessage was cleared
			if pane.errorMessage != "" {
				t.Errorf("expected errorMessage to be cleared, got %q", pane.errorMessage)
			}

			// Verify isProcessing flag was cleared
			if pane.isProcessing {
				t.Errorf("expected isProcessing to be false, got true")
			}
		})
	}
}

// TestGitOperationCompleteMsg_Failure verifies that GitOperationCompleteMsg with success=false
// updates errorMessage, clears statusMessage, and clears isProcessing flag.
//
// Requirements: 7.6, 8.4, 9.4, 10.4, 11.3, 12.5, 13.3
func TestGitOperationCompleteMsg_Failure(t *testing.T) {
	tests := []struct {
		name          string
		operation     string
		errorMsg      string
	}{
		{
			name:      "Pull failure",
			operation: "pull",
			errorMsg:  "Authentication failed: invalid credentials",
		},
		{
			name:      "Push failure",
			operation: "push",
			errorMsg:  "Network error: connection timeout",
		},
		{
			name:      "Fetch failure",
			operation: "fetch",
			errorMsg:  "Git operation failed: repository not found",
		},
		{
			name:      "Clone failure",
			operation: "clone",
			errorMsg:  "Authentication failed: invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create and initialize a GitPane
			pane := &GitPane{}
			pane.Init()

			// Set initial state with status message and isProcessing flag
			pane.statusMessage = "previous status"
			pane.errorMessage = ""
			pane.isProcessing = true

			// Send GitOperationCompleteMsg with failure
			msg := GitOperationCompleteMsg{
				Operation: tt.operation,
				Success:   false,
				Message:   "",
				Error:     fmt.Errorf("%s", tt.errorMsg),
				NewDir:    "",
			}

			_, _ = pane.Update(msg)

			// Verify errorMessage was updated
			if pane.errorMessage != tt.errorMsg {
				t.Errorf("expected errorMessage %q, got %q", tt.errorMsg, pane.errorMessage)
			}

			// Verify statusMessage was cleared
			if pane.statusMessage != "" {
				t.Errorf("expected statusMessage to be cleared, got %q", pane.statusMessage)
			}

			// Verify isProcessing flag was cleared
			if pane.isProcessing {
				t.Errorf("expected isProcessing to be false, got true")
			}
		})
	}
}

// TestGitOperationMessages_ReturnGitOperationCompleteMsg verifies that Git operation messages
// return GitOperationCompleteMsg instead of OperationResult.
//
// Requirements: 7.1, 8.1, 9.1, 10.1, 11.1, 12.1, 13.1
func TestGitOperationMessages_ReturnGitOperationCompleteMsg(t *testing.T) {
	// Create a temporary directory and initialize a git repo for testing
	tmpDir := t.TempDir()
	
	// Initialize a git repository
	_, err := gogit.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Create a GitClient for the temp directory
	client := git.NewClient(tmpDir)

	// Create and initialize a GitPane
	pane := &GitPane{
		gitClient: client,
		workDir:   tmpDir,
	}
	pane.Init()

	// Test GitStageMsg returns GitOperationCompleteMsg
	t.Run("GitStageMsg returns GitOperationCompleteMsg", func(t *testing.T) {
		// Send GitStageMsg
		_, cmd := pane.Update(GitStageMsg{})
		
		// Verify command was returned
		if cmd == nil {
			t.Fatalf("expected non-nil command from GitStageMsg")
		}
		
		// Execute the command to trigger the async operation
		result := cmd()
		
		// Verify the result is a GitOperationCompleteMsg
		completeMsg, ok := result.(GitOperationCompleteMsg)
		if !ok {
			t.Fatalf("expected GitOperationCompleteMsg, got %T", result)
		}
		
		// Verify the operation field is set correctly
		if completeMsg.Operation != "stage" {
			t.Errorf("expected Operation to be 'stage', got %q", completeMsg.Operation)
		}
		
		// Verify the message has Success field
		if !completeMsg.Success {
			t.Logf("Stage operation failed (expected for empty repo): %v", completeMsg.Error)
		}
	})

	// Test GitStatusMsg returns GitOperationCompleteMsg
	t.Run("GitStatusMsg returns GitOperationCompleteMsg", func(t *testing.T) {
		// Send GitStatusMsg
		_, cmd := pane.Update(GitStatusMsg{})
		
		// Verify command was returned
		if cmd == nil {
			t.Fatalf("expected non-nil command from GitStatusMsg")
		}
		
		// Execute the command to trigger the async operation
		result := cmd()
		
		// Verify the result is a GitOperationCompleteMsg
		completeMsg, ok := result.(GitOperationCompleteMsg)
		if !ok {
			t.Fatalf("expected GitOperationCompleteMsg, got %T", result)
		}
		
		// Verify the operation field is set correctly
		if completeMsg.Operation != "status" {
			t.Errorf("expected Operation to be 'status', got %q", completeMsg.Operation)
		}
		
		// Status should succeed on an initialized repo
		if !completeMsg.Success {
			t.Errorf("expected Status operation to succeed, got error: %v", completeMsg.Error)
		}
	})

	// Test GitRestoreMsg returns GitOperationCompleteMsg
	t.Run("GitRestoreMsg returns GitOperationCompleteMsg", func(t *testing.T) {
		// Send GitRestoreMsg
		_, cmd := pane.Update(GitRestoreMsg{})
		
		// Verify command was returned
		if cmd == nil {
			t.Fatalf("expected non-nil command from GitRestoreMsg")
		}
		
		// Execute the command to trigger the async operation
		result := cmd()
		
		// Verify the result is a GitOperationCompleteMsg
		completeMsg, ok := result.(GitOperationCompleteMsg)
		if !ok {
			t.Fatalf("expected GitOperationCompleteMsg, got %T", result)
		}
		
		// Verify the operation field is set correctly
		if completeMsg.Operation != "restore" {
			t.Errorf("expected Operation to be 'restore', got %q", completeMsg.Operation)
		}
	})
}

// Feature: git-integration, Property 13: Operation Success Messages
// **Validates: Requirements 8.3, 9.3, 10.3, 11.2, 13.2**
//
// For any Git operation that completes successfully, the Git UI should display a success message
// and clear any previous error messages.
func TestProperty13_OperationSuccessMessages(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("successful operations display success message and clear errors", prop.ForAll(
		func(operationNum uint8, hasExistingError bool) bool {
			// Map operationNum to one of the operations that can succeed
			// We'll test: pull, push, fetch, stage, restore (5 operations)
			operations := []string{"pull", "push", "fetch", "stage", "restore"}
			operation := operations[int(operationNum)%len(operations)]

			// Create a GitPane and initialize it
			pane := &GitPane{}
			pane.Init()

			// Set initial state with potential error message
			if hasExistingError {
				pane.errorMessage = "previous error message"
			}
			pane.statusMessage = ""
			pane.isProcessing = true

			// Create a success message
			successMessage := fmt.Sprintf("%s completed successfully", operation)

			// Send GitOperationCompleteMsg with success
			msg := GitOperationCompleteMsg{
				Operation: operation,
				Success:   true,
				Message:   successMessage,
				Error:     nil,
				NewDir:    "",
			}

			_, _ = pane.Update(msg)

			// Verify statusMessage was set
			if pane.statusMessage == "" {
				t.Logf("Expected statusMessage to be set for successful %s operation, got empty", operation)
				return false
			}

			// Verify errorMessage was cleared
			if pane.errorMessage != "" {
				t.Logf("Expected errorMessage to be cleared for successful %s operation, got %q", operation, pane.errorMessage)
				return false
			}

			// Verify isProcessing flag was cleared
			if pane.isProcessing {
				t.Logf("Expected isProcessing to be false after successful %s operation", operation)
				return false
			}

			return true
		},
		gen.UInt8(),
		gen.Bool(),
	))

	properties.Property("clone success displays special message with directory", prop.ForAll(
		func(hasExistingError bool) bool {
			// Create a GitPane and initialize it
			pane := &GitPane{}
			pane.Init()

			// Set initial state with potential error message
			if hasExistingError {
				pane.errorMessage = "previous error message"
			}
			pane.statusMessage = ""
			pane.isProcessing = true

			// Send GitOperationCompleteMsg with clone success
			newDir := "/tmp/test-repo"
			msg := GitOperationCompleteMsg{
				Operation: "clone",
				Success:   true,
				Message:   newDir,
				Error:     nil,
				NewDir:    newDir,
			}

			_, _ = pane.Update(msg)

			// Verify statusMessage contains the directory
			if pane.statusMessage == "" {
				t.Logf("Expected statusMessage to be set for successful clone operation, got empty")
				return false
			}
			if pane.statusMessage != "Clone completed successfully: "+newDir {
				t.Logf("Expected statusMessage to contain directory path, got %q", pane.statusMessage)
				return false
			}

			// Verify errorMessage was cleared
			if pane.errorMessage != "" {
				t.Logf("Expected errorMessage to be cleared for successful clone operation, got %q", pane.errorMessage)
				return false
			}

			// Verify isProcessing flag was cleared
			if pane.isProcessing {
				t.Logf("Expected isProcessing to be false after successful clone operation")
				return false
			}

			return true
		},
		gen.Bool(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: git-integration, Property 14: Operation Failure Keeps UI Open
// **Validates: Requirements 6.2, 6.3, 7.6, 8.4, 9.4, 10.4, 11.3, 12.5, 13.3, 15.3**
//
// For any Git operation that fails, the Git UI should remain visible with the current input values
// preserved and display an error message.
func TestProperty14_OperationFailureKeepsUIOpen(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("failed operations keep UI visible and preserve input values", prop.ForAll(
		func(operationNum uint8, urlSuffix uint8, userNum uint8, passNum uint8, hasExistingStatus bool) bool {
			// Map operationNum to one of the operations
			operations := []string{"clone", "pull", "push", "fetch", "stage", "status", "restore"}
			operation := operations[int(operationNum)%len(operations)]

			// Create input values from numbers to ensure non-empty strings
			url := fmt.Sprintf("https://github.com/user/repo%d", urlSuffix)
			username := fmt.Sprintf("user%d", userNum)
			password := fmt.Sprintf("pass%d", passNum)

			// Create a GitPane and initialize it
			pane := &GitPane{
				visible: true, // UI is open
			}
			pane.Init()

			// Set input values
			pane.urlInput.SetValue(url)
			pane.userInput.SetValue(username)
			pane.passInput.SetValue(password)

			// Set initial state with potential status message
			if hasExistingStatus {
				pane.statusMessage = "previous status message"
			}
			pane.errorMessage = ""
			pane.isProcessing = true

			// Create an error message
			errorMessage := fmt.Sprintf("%s operation failed: test error", operation)

			// Send GitOperationCompleteMsg with failure
			msg := GitOperationCompleteMsg{
				Operation: operation,
				Success:   false,
				Message:   "",
				Error:     fmt.Errorf("%s", errorMessage),
				NewDir:    "",
			}

			_, _ = pane.Update(msg)

			// Verify UI remains visible
			if !pane.visible {
				t.Logf("Expected UI to remain visible after failed %s operation", operation)
				return false
			}

			// Verify input values are preserved
			if pane.urlInput.Value() != url {
				t.Logf("Expected URL to be preserved after failed %s operation, got %q", operation, pane.urlInput.Value())
				return false
			}
			if pane.userInput.Value() != username {
				t.Logf("Expected username to be preserved after failed %s operation, got %q", operation, pane.userInput.Value())
				return false
			}
			if pane.passInput.Value() != password {
				t.Logf("Expected password to be preserved after failed %s operation, got %q", operation, pane.passInput.Value())
				return false
			}

			// Verify errorMessage was set
			if pane.errorMessage == "" {
				t.Logf("Expected errorMessage to be set for failed %s operation, got empty", operation)
				return false
			}

			// Verify statusMessage was cleared
			if pane.statusMessage != "" {
				t.Logf("Expected statusMessage to be cleared for failed %s operation, got %q", operation, pane.statusMessage)
				return false
			}

			// Verify isProcessing flag was cleared
			if pane.isProcessing {
				t.Logf("Expected isProcessing to be false after failed %s operation", operation)
				return false
			}

			return true
		},
		gen.UInt8(),
		gen.UInt8(),
		gen.UInt8(),
		gen.UInt8(),
		gen.Bool(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: git-integration, Property 15: Error Message Categorization
// **Validates: Requirements 15.2**
//
// For any Git operation failure, the error message should indicate the failure category
// (authentication error, network error, or Git operation error) to help users diagnose the issue.
func TestProperty15_ErrorMessageCategorization(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("authentication errors are categorized correctly", prop.ForAll(
		func(operationNum uint8) bool {
			// Map operationNum to one of the operations
			operations := []string{"clone", "pull", "push", "fetch"}
			operation := operations[int(operationNum)%len(operations)]

			// Create a GitPane and initialize it
			pane := &GitPane{}
			pane.Init()

			// Create an authentication error using GitError type
			authError := &git.GitError{
				Category: "Authentication",
				Message:  "Authentication failed: invalid credentials",
				Hint:     "For private repositories, use a GitHub Personal Access Token (ghp_...)",
				Original: fmt.Errorf("authentication required"),
			}

			// Send GitOperationCompleteMsg with authentication failure
			msg := GitOperationCompleteMsg{
				Operation: operation,
				Success:   false,
				Message:   "",
				Error:     authError,
				NewDir:    "",
			}

			_, _ = pane.Update(msg)

			// Verify errorMessage contains "Authentication failed"
			if !strings.Contains(pane.errorMessage, "Authentication failed") {
				t.Logf("Expected errorMessage to contain 'Authentication failed' for auth error, got %q", pane.errorMessage)
				return false
			}

			return true
		},
		gen.UInt8(),
	))

	properties.Property("network errors are categorized correctly", prop.ForAll(
		func(operationNum uint8) bool {
			// Map operationNum to one of the operations
			operations := []string{"clone", "pull", "push", "fetch"}
			operation := operations[int(operationNum)%len(operations)]

			// Create a GitPane and initialize it
			pane := &GitPane{}
			pane.Init()

			// Create a network error using GitError type
			networkError := &git.GitError{
				Category: "Network",
				Message:  "Network error: connection timeout",
				Hint:     "Check your internet connection and try again",
				Original: fmt.Errorf("dial tcp: timeout"),
			}

			// Send GitOperationCompleteMsg with network failure
			msg := GitOperationCompleteMsg{
				Operation: operation,
				Success:   false,
				Message:   "",
				Error:     networkError,
				NewDir:    "",
			}

			_, _ = pane.Update(msg)

			// Verify errorMessage contains "Network error"
			if !strings.Contains(pane.errorMessage, "Network error") {
				t.Logf("Expected errorMessage to contain 'Network error' for network error, got %q", pane.errorMessage)
				return false
			}

			return true
		},
		gen.UInt8(),
	))

	properties.Property("git operation errors are categorized correctly", prop.ForAll(
		func(operationNum uint8) bool {
			// Map operationNum to one of the operations
			operations := []string{"clone", "pull", "push", "fetch", "stage", "status", "restore"}
			operation := operations[int(operationNum)%len(operations)]

			// Create a GitPane and initialize it
			pane := &GitPane{}
			pane.Init()

			// Create a Git operation error using GitError type
			gitError := &git.GitError{
				Category: "Git Operation",
				Message:  "Git operation failed: repository not found",
				Hint:     "Verify the repository URL is correct and you have access permissions",
				Original: fmt.Errorf("repository not found"),
			}

			// Send GitOperationCompleteMsg with Git operation failure
			msg := GitOperationCompleteMsg{
				Operation: operation,
				Success:   false,
				Message:   "",
				Error:     gitError,
				NewDir:    "",
			}

			_, _ = pane.Update(msg)

			// Verify errorMessage contains "Git operation failed"
			if !strings.Contains(pane.errorMessage, "Git operation failed") {
				t.Logf("Expected errorMessage to contain 'Git operation failed' for git error, got %q", pane.errorMessage)
				return false
			}

			return true
		},
		gen.UInt8(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: git-integration, Property 16: Error Message Clearing
// **Validates: Requirements 15.4**
//
// For any Git UI state with an existing error message, initiating a new operation should clear
// the previous error message before executing.
func TestProperty16_ErrorMessageClearing(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("initiating new operation clears previous error message", prop.ForAll(
		func(buttonNum uint8, errorNum uint8) bool {
			// Map buttonNum to one of the 7 buttons
			// 0=Clone, 1=Pull, 2=Push, 3=Fetch, 4=Stage, 5=Status, 6=Restore
			selectedButton := int(buttonNum) % 7

			// Create a GitPane and initialize it
			tmpDir := t.TempDir()
			client := git.NewClient(tmpDir)
			
			pane := &GitPane{
				gitClient: client,
				workDir:   tmpDir,
			}
			pane.Init()

			// Set input values
			pane.urlInput.SetValue("https://github.com/user/repo")
			pane.userInput.SetValue("testuser")
			pane.passInput.SetValue("testpass")

			// Set an existing error message
			previousError := fmt.Sprintf("previous error %d", errorNum)
			pane.errorMessage = previousError
			pane.statusMessage = ""

			// Focus on buttons and select the button to test
			pane.focusedInput = 3
			pane.selectedButton = selectedButton
			pane.updateFocus()

			// Press Enter to activate the button (which should clear the error message)
			_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})

			// Verify that error message was cleared before the operation started
			if pane.errorMessage != "" {
				t.Logf("Expected errorMessage to be cleared when initiating new operation, got %q", pane.errorMessage)
				return false
			}

			// Verify that status message was also cleared
			if pane.statusMessage != "" {
				t.Logf("Expected statusMessage to be cleared when initiating new operation, got %q", pane.statusMessage)
				return false
			}

			// Verify that a command was returned (operation was initiated)
			if cmd == nil {
				t.Logf("Expected non-nil command when activating button")
				return false
			}

			return true
		},
		gen.UInt8(),
		gen.UInt8(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: git-integration, Property 2: UI Overlay Rendering
// **Validates: Requirements 1.3**
//
// For any application state when the Git UI is visible, the rendered view should contain
// both the main IDE interface and the Git UI popup overlay.
func TestProperty2_UIOverlayRendering(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("visible UI renders non-empty content", prop.ForAll(
		func(width uint16, height uint16) bool {
			// Ensure reasonable dimensions (at least 20x10)
			w := int(width%200) + 80  // 80-279
			h := int(height%100) + 20 // 20-119

			// Create a GitPane and initialize it
			pane := &GitPane{
				visible: true,
				width:   w,
				height:  h,
			}
			pane.Init()

			// Render the view
			view := pane.View()

			// Verify the view is not empty when visible
			if view == "" {
				t.Logf("Expected non-empty view when visible, got empty string")
				return false
			}

			// Verify the view contains key elements
			if !strings.Contains(view, "Git Operations") {
				t.Logf("Expected view to contain 'Git Operations' title")
				return false
			}

			return true
		},
		gen.UInt16(),
		gen.UInt16(),
	))

	properties.Property("hidden UI renders empty content", prop.ForAll(
		func(width uint16, height uint16) bool {
			// Ensure reasonable dimensions
			w := int(width%200) + 80
			h := int(height%100) + 20

			// Create a GitPane and initialize it
			pane := &GitPane{
				visible: false,
				width:   w,
				height:  h,
			}
			pane.Init()

			// Render the view
			view := pane.View()

			// Verify the view is empty when not visible
			if view != "" {
				t.Logf("Expected empty view when not visible, got %q", view)
				return false
			}

			return true
		},
		gen.UInt16(),
		gen.UInt16(),
	))

	properties.Property("UI renders all seven buttons", prop.ForAll(
		func() bool {
			// Create a GitPane and initialize it
			pane := &GitPane{
				visible: true,
				width:   100,
				height:  30,
			}
			pane.Init()

			// Render the view
			view := pane.View()

			// Verify all seven button names are present
			buttonNames := []string{"Clone", "Pull", "Push", "Fetch", "Stage", "Status", "Restore"}
			for _, name := range buttonNames {
				if !strings.Contains(view, name) {
					t.Logf("Expected view to contain button %q", name)
					return false
				}
			}

			return true
		},
	))

	properties.Property("UI renders input field labels", prop.ForAll(
		func() bool {
			// Create a GitPane and initialize it
			pane := &GitPane{
				visible: true,
				width:   100,
				height:  30,
			}
			pane.Init()

			// Render the view
			view := pane.View()

			// Verify input field labels are present
			if !strings.Contains(view, "Git URL") {
				t.Logf("Expected view to contain 'Git URL' label")
				return false
			}
			if !strings.Contains(view, "Git USER") {
				t.Logf("Expected view to contain 'Git USER' label")
				return false
			}
			if !strings.Contains(view, "Git PASS") {
				t.Logf("Expected view to contain 'Git PASS' label")
				return false
			}

			return true
		},
	))

	properties.Property("UI renders error messages when present", prop.ForAll(
		func(errorNum uint8) bool {
			// Create a GitPane and initialize it
			pane := &GitPane{
				visible: true,
				width:   100,
				height:  30,
			}
			pane.Init()

			// Set an error message
			errorMsg := fmt.Sprintf("Test error %d", errorNum)
			pane.errorMessage = errorMsg

			// Render the view
			view := pane.View()

			// Verify error message is present in the view
			if !strings.Contains(view, errorMsg) {
				t.Logf("Expected view to contain error message %q", errorMsg)
				return false
			}

			return true
		},
		gen.UInt8(),
	))

	properties.Property("UI renders status messages when present", prop.ForAll(
		func(statusNum uint8) bool {
			// Create a GitPane and initialize it
			pane := &GitPane{
				visible: true,
				width:   100,
				height:  30,
			}
			pane.Init()

			// Set a status message
			statusMsg := fmt.Sprintf("Test status %d", statusNum)
			pane.statusMessage = statusMsg

			// Render the view
			view := pane.View()

			// Verify status message is present in the view
			if !strings.Contains(view, statusMsg) {
				t.Logf("Expected view to contain status message %q", statusMsg)
				return false
			}

			return true
		},
		gen.UInt8(),
	))

	properties.Property("UI renders processing indicator when processing", prop.ForAll(
		func() bool {
			// Create a GitPane and initialize it
			pane := &GitPane{
				visible:      true,
				width:        100,
				height:       30,
				isProcessing: true,
			}
			pane.Init()

			// Render the view
			view := pane.View()

			// Verify processing indicator is present
			if !strings.Contains(view, "Processing") {
				t.Logf("Expected view to contain 'Processing' indicator")
				return false
			}

			return true
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestUIRendering_AllSevenButtonsVisible verifies that all seven buttons are visible in the rendered view.
//
// Requirements: 3.4, 3.5
func TestUIRendering_AllSevenButtonsVisible(t *testing.T) {
	// Create a GitPane and initialize it
	pane := &GitPane{
		visible: true,
		width:   100,
		height:  30,
	}
	pane.Init()

	// Render the view
	view := pane.View()

	// Verify all seven button names are present
	buttonNames := []string{"Clone", "Pull", "Push", "Fetch", "Stage", "Status", "Restore"}
	for _, name := range buttonNames {
		if !strings.Contains(view, name) {
			t.Errorf("Expected view to contain button %q, but it was not found", name)
		}
	}
}

// TestUIRendering_InputFieldLabels verifies that input field labels match requirements.
//
// Requirements: 3.1, 3.2, 3.3
func TestUIRendering_InputFieldLabels(t *testing.T) {
	// Create a GitPane and initialize it
	pane := &GitPane{
		visible: true,
		width:   100,
		height:  30,
	}
	pane.Init()

	// Render the view
	view := pane.View()

	// Verify input field labels are present and match requirements
	if !strings.Contains(view, "Git URL") {
		t.Errorf("Expected view to contain 'Git URL' label")
	}
	if !strings.Contains(view, "Git USER") {
		t.Errorf("Expected view to contain 'Git USER' label")
	}
	if !strings.Contains(view, "Git PASS") {
		t.Errorf("Expected view to contain 'Git PASS' label")
	}
	// Verify the PASS field has the (ghp_) hint
	if !strings.Contains(view, "ghp_") {
		t.Errorf("Expected view to contain 'ghp_' hint in PASS field label")
	}
}

// TestUIRendering_StatusMessageFormatting verifies that status messages are formatted correctly.
//
// Requirements: 8.3, 9.3, 10.3, 11.2, 13.2
func TestUIRendering_StatusMessageFormatting(t *testing.T) {
	tests := []struct {
		name          string
		statusMessage string
	}{
		{
			name:          "Pull success message",
			statusMessage: "Pull completed successfully",
		},
		{
			name:          "Push success message",
			statusMessage: "Push completed successfully",
		},
		{
			name:          "Fetch success message",
			statusMessage: "Fetch completed successfully",
		},
		{
			name:          "Stage success message with count",
			statusMessage: "Staged 3 file(s)",
		},
		{
			name:          "Restore success message with count",
			statusMessage: "Restored 2 file(s)",
		},
		{
			name:          "Clone success message with directory",
			statusMessage: "Clone completed successfully: /tmp/myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a GitPane and initialize it
			pane := &GitPane{
				visible:       true,
				width:         100,
				height:        30,
				statusMessage: tt.statusMessage,
			}
			pane.Init()

			// Render the view
			view := pane.View()

			// Verify status message is present in the view
			if !strings.Contains(view, tt.statusMessage) {
				t.Errorf("Expected view to contain status message %q, but it was not found", tt.statusMessage)
			}

			// Verify no error message is shown when status message is present
			if strings.Contains(view, "Error:") {
				t.Errorf("Expected view to not contain 'Error:' when status message is present")
			}
		})
	}
}

// TestUIRendering_ErrorMessageStyling verifies that error messages are styled correctly.
//
// Requirements: 15.1, 15.2, 15.3
func TestUIRendering_ErrorMessageStyling(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
	}{
		{
			name:         "Authentication error",
			errorMessage: "Authentication failed: invalid credentials",
		},
		{
			name:         "Network error",
			errorMessage: "Network error: connection timeout",
		},
		{
			name:         "Git operation error",
			errorMessage: "Git operation failed: repository not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a GitPane and initialize it
			pane := &GitPane{
				visible:      true,
				width:        100,
				height:       30,
				errorMessage: tt.errorMessage,
			}
			pane.Init()

			// Render the view
			view := pane.View()

			// Verify error message is present in the view with "Error:" prefix
			if !strings.Contains(view, "Error:") {
				t.Errorf("Expected view to contain 'Error:' prefix for error messages")
			}
			if !strings.Contains(view, tt.errorMessage) {
				t.Errorf("Expected view to contain error message %q, but it was not found", tt.errorMessage)
			}

			// Verify no status message is shown when error message is present
			if pane.statusMessage != "" && strings.Contains(view, pane.statusMessage) {
				t.Errorf("Expected view to not show status message when error message is present")
			}
		})
	}
}

// TestUIRendering_ProcessingIndicatorDisplay verifies that the processing indicator is displayed correctly.
//
// Requirements: 7.1, 8.1, 9.1, 10.1, 11.1, 12.1, 13.1
func TestUIRendering_ProcessingIndicatorDisplay(t *testing.T) {
	// Create a GitPane and initialize it with isProcessing flag set
	pane := &GitPane{
		visible:      true,
		width:        100,
		height:       30,
		isProcessing: true,
	}
	pane.Init()

	// Render the view
	view := pane.View()

	// Verify processing indicator is present
	if !strings.Contains(view, "Processing") {
		t.Errorf("Expected view to contain 'Processing' indicator when isProcessing is true")
	}

	// Verify no error or status messages are shown when processing
	if strings.Contains(view, "Error:") {
		t.Errorf("Expected view to not show error message when processing")
	}

	// Now test with isProcessing false
	pane.isProcessing = false
	view = pane.View()

	// Verify processing indicator is not present
	if strings.Contains(view, "Processing") {
		t.Errorf("Expected view to not contain 'Processing' indicator when isProcessing is false")
	}
}

// TestUIRendering_HiddenUIReturnsEmpty verifies that the View method returns empty string when not visible.
//
// Requirements: 1.3
func TestUIRendering_HiddenUIReturnsEmpty(t *testing.T) {
	// Create a GitPane and initialize it with visible=false
	pane := &GitPane{
		visible: false,
		width:   100,
		height:  30,
	}
	pane.Init()

	// Render the view
	view := pane.View()

	// Verify the view is empty
	if view != "" {
		t.Errorf("Expected empty view when visible=false, got %q", view)
	}
}

// TestUIRendering_SelectedButtonHighlighting verifies that the selected button is highlighted.
//
// Requirements: 3.4, 3.5
func TestUIRendering_SelectedButtonHighlighting(t *testing.T) {
	// Test each button selection
	buttonNames := []string{"Clone", "Pull", "Push", "Fetch", "Stage", "Status", "Restore"}
	
	for i, name := range buttonNames {
		t.Run("Selected button: "+name, func(t *testing.T) {
			// Create a GitPane and initialize it
			pane := &GitPane{
				visible:        true,
				width:          100,
				height:         30,
				focusedInput:   3, // Focused on buttons
				selectedButton: i,
			}
			pane.Init()

			// Render the view
			view := pane.View()

			// Verify the button name is present
			if !strings.Contains(view, name) {
				t.Errorf("Expected view to contain button %q", name)
			}

			// Note: We can't easily verify the styling (bold, different color) in a unit test
			// without parsing ANSI codes, but we can verify the button is present
		})
	}
}

// TestGitPane_KeyboardNavigation verifies that keyboard input is properly handled
// when GitPane is visible, including Tab navigation between fields and buttons.
//
// This test addresses the bug where GitPane opened with Ctrl+G but keyboard input
// didn't work because App.Update wasn't routing messages to GitPane.
func TestGitPane_KeyboardNavigation(t *testing.T) {
	// Create a GitPane instance
	gitClient := git.NewClient(t.TempDir())
	pane := NewGitPane(gitClient, t.TempDir())
	pane.visible = true

	tests := []struct {
		name           string
		key            string
		initialFocus   int
		expectedFocus  int
		checkURLFocus  bool
		checkUserFocus bool
		checkPassFocus bool
	}{
		{
			name:          "Tab moves from URL to USER",
			key:           "tab",
			initialFocus:  0,
			expectedFocus: 1,
			checkURLFocus: false,
		},
		{
			name:           "Tab moves from USER to PASS",
			key:            "tab",
			initialFocus:   1,
			expectedFocus:  2,
			checkUserFocus: false,
		},
		{
			name:           "Tab moves from PASS to buttons",
			key:            "tab",
			initialFocus:   2,
			expectedFocus:  3,
			checkPassFocus: false,
		},
		{
			name:          "Tab wraps from buttons to URL",
			key:           "tab",
			initialFocus:  3,
			expectedFocus: 0,
			checkURLFocus: true,
		},
		{
			name:          "Shift+Tab moves from URL to buttons",
			key:           "shift+tab",
			initialFocus:  0,
			expectedFocus: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set initial focus
			pane.focusedInput = tt.initialFocus
			pane.updateFocus()

			// Send keyboard message
			keyMsg := tea.KeyMsg{Type: tea.KeyTab}
			if tt.key == "shift+tab" {
				keyMsg = tea.KeyMsg{Type: tea.KeyShiftTab}
			}

			// Update the pane
			_, _ = pane.Update(keyMsg)

			// Verify focus changed
			if pane.focusedInput != tt.expectedFocus {
				t.Errorf("expected focusedInput to be %d, got %d", tt.expectedFocus, pane.focusedInput)
			}

			// Verify the correct input field is focused
			if tt.checkURLFocus && pane.urlInput.Focused() == false {
				t.Errorf("expected URL input to be focused")
			}
			if tt.checkUserFocus && pane.userInput.Focused() == false {
				t.Errorf("expected USER input to be focused")
			}
			if tt.checkPassFocus && pane.passInput.Focused() == false {
				t.Errorf("expected PASS input to be focused")
			}
		})
	}
}

// TestGitPane_TextInput verifies that text input works in the input fields
// when GitPane is visible.
func TestGitPane_TextInput(t *testing.T) {
	// Create a GitPane instance
	gitClient := git.NewClient(t.TempDir())
	pane := NewGitPane(gitClient, t.TempDir())
	pane.visible = true

	// Focus on URL input
	pane.focusedInput = 0
	pane.updateFocus()

	// Type some text
	testURL := "https://github.com/user/repo"
	for _, char := range testURL {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		_, _ = pane.Update(keyMsg)
	}

	// Verify the URL was entered
	if pane.urlInput.Value() != testURL {
		t.Errorf("expected URL input to be %q, got %q", testURL, pane.urlInput.Value())
	}

	// Move to USER input
	keyMsg := tea.KeyMsg{Type: tea.KeyTab}
	_, _ = pane.Update(keyMsg)

	// Type username
	testUser := "testuser"
	for _, char := range testUser {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		_, _ = pane.Update(keyMsg)
	}

	// Verify the username was entered
	if pane.userInput.Value() != testUser {
		t.Errorf("expected USER input to be %q, got %q", testUser, pane.userInput.Value())
	}
}

// TestGitPane_EscapeClosesPane verifies that pressing Esc closes the GitPane.
func TestGitPane_EscapeClosesPane(t *testing.T) {
	// Create a GitPane instance
	gitClient := git.NewClient(t.TempDir())
	pane := NewGitPane(gitClient, t.TempDir())
	pane.visible = true

	// Send Esc key
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	_, _ = pane.Update(keyMsg)

	// Verify the pane is now hidden
	if pane.visible {
		t.Errorf("expected GitPane to be hidden after Esc, but it's still visible")
	}
}
