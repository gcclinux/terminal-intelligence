package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/terminal-intelligence/internal/git"
)

// GitPane is a bubbletea Model that renders as a popup overlay for Git operations.
// It provides a Terminal UI interface for common Git operations including clone, pull,
// push, fetch, stage, status, and restore. The pane manages three text input fields
// (URL, USER, PASS), seven action buttons, and displays operation status/error messages.
//
// The GitPane integrates with the existing App in internal/ui/app.go and uses the
// GitClient from internal/git/client.go for Git operations.
//
// Requirements: 3.1, 3.2, 3.3
type GitPane struct {
	// UI state
	visible bool // Whether the GitPane is currently visible
	width   int  // Width of the terminal window
	height  int  // Height of the terminal window

	// Input focus management
	// focusedInput: 0=URL, 1=USER, 2=PASS, 3=buttons
	focusedInput int

	// Input fields (using textinput from bubbletea)
	urlInput  textinput.Model // Input field for Git repository URL
	userInput textinput.Model // Input field for Git username
	passInput textinput.Model // Input field for Git password or GitHub PAT (ghp_*)

	// Button state
	// selectedButton: 0=Clone, 1=Pull, 2=Fetch, 3=Stage, 4=Commit, 5=Push, 6=Status, 7=Restore
	selectedButton int

	// Status display
	statusMessage string // Success message displayed after successful operations
	errorMessage  string // Error message displayed after failed operations
	isProcessing  bool   // Whether a Git operation is currently in progress

	// Dependencies
	gitClient *git.Client // GitClient for executing Git operations
	workDir   string      // Current working directory where Git operations are performed
}

// NewGitPane creates a new GitPane instance with the provided GitClient and working directory.
// The GitPane is initially hidden and will be shown when Toggle() is called.
//
// Parameters:
//   - gitClient: GitClient instance for executing Git operations
//   - workDir: Current working directory where Git operations are performed
//
// Returns:
//   - *GitPane: Initialized GitPane instance
func NewGitPane(gitClient *git.Client, workDir string) *GitPane {
	g := &GitPane{
		gitClient: gitClient,
		workDir:   workDir,
		visible:   false,
	}
	g.Init()
	return g
}

// Init initializes the GitPane component when it's first created.
// This method is part of the tea.Model interface from bubbletea.
// It sets up the three text input fields (URL, USER, PASS) with appropriate
// labels and configures the PASS field to hide password input.
//
// Requirements: 3.1, 3.2, 3.3, 3.4
func (g *GitPane) Init() tea.Cmd {
	// Initialize URL input field
	g.urlInput = textinput.New()
	g.urlInput.Placeholder = "https://github.com/user/repo"
	g.urlInput.Prompt = "Git URL: "
	g.urlInput.CharLimit = 500
	g.urlInput.Width = 60

	// Initialize USER input field
	g.userInput = textinput.New()
	g.userInput.Placeholder = "username"
	g.userInput.Prompt = "Git USER: "
	g.userInput.CharLimit = 100
	g.userInput.Width = 60

	// Initialize PASS input field with password mode
	g.passInput = textinput.New()
	g.passInput.Placeholder = "password or token"
	g.passInput.Prompt = "Git PASS (ghp_): "
	g.passInput.CharLimit = 200
	g.passInput.Width = 60
	g.passInput.EchoMode = textinput.EchoPassword // Hide password input
	g.passInput.EchoCharacter = '•'

	// Set initial focus to URL input
	g.urlInput.Focus()
	g.focusedInput = 0

	// Return nil command (no initial command to execute)
	return nil
}

// GitRepositoryDetectedMsg is a message sent when repository detection completes.
// It contains information about whether the current directory is a Git repository
// and any stored credentials that were found.
type GitRepositoryDetectedMsg struct {
	IsRepo      bool             // Whether the directory is a Git repository
	Credentials *git.Credentials // Stored credentials from .git/config, nil if not found
	RemoteURL   string           // URL of the remote repository, empty if not configured
}

// Git operation messages - sent when user activates a button

// GitCloneMsg is sent when the user activates the Clone button.
type GitCloneMsg struct {
	URL      string
	Username string
	Password string
}

// GitPullMsg is sent when the user activates the Pull button.
type GitPullMsg struct {
	Username string
	Password string
}

// GitPushMsg is sent when the user activates the Push button.
type GitPushMsg struct {
	Username string
	Password string
}

// GitFetchMsg is sent when the user activates the Fetch button.
type GitFetchMsg struct {
	Username string
	Password string
}

// GitStageMsg is sent when the user activates the Stage button.
type GitStageMsg struct{}

// GitCommitMsg is sent when the user activates the Commit button.
type GitCommitMsg struct {
	Message string
}

// GitStatusMsg is sent when the user activates the Status button.
type GitStatusMsg struct{}

// GitRestoreMsg is sent when the user activates the Restore button.
type GitRestoreMsg struct{}

// GitOperationCompleteMsg is sent when a Git operation completes (successfully or with an error).
// It contains the operation type, success status, result message, any error, and for clone operations,
// the path to the newly cloned directory.
type GitOperationCompleteMsg struct {
	Operation string // The operation type: "clone", "pull", "push", "fetch", "stage", "status", "restore"
	Success   bool   // Whether the operation completed successfully
	Message   string // Human-readable message describing the result
	Error     error  // Error details if the operation failed, nil on success
	NewDir    string // For clone operations: the path to the newly cloned directory
}

// Toggle toggles the visibility of the GitPane.
// When opening, it triggers repository detection to check if the current directory
// is a Git repository and populate credentials if found.
// When closing, it clears any status or error messages.
//
// Requirements: 1.1, 1.2
func (g *GitPane) Toggle() tea.Cmd {
	g.visible = !g.visible

	if g.visible {
		// Opening the pane - trigger repository detection
		return g.detectRepository()
	}

	// Closing the pane - clear status messages
	g.statusMessage = ""
	g.errorMessage = ""
	return nil
}

// IsVisible returns whether the GitPane is currently visible.
func (g *GitPane) IsVisible() bool {
	return g.visible
}

// SetWorkDir updates the working directory for the GitPane and its GitClient.
// This should be called when the IDE changes directories (e.g., after a clone operation).
//
// Parameters:
//   - dir: The new working directory path
//
// Returns:
//   - tea.Cmd: A command to trigger repository detection in the new directory
func (g *GitPane) SetWorkDir(dir string) tea.Cmd {
	g.workDir = dir
	if g.gitClient != nil {
		g.gitClient = git.NewClient(dir)
	}
	// If the pane is visible, re-detect repository in the new directory
	if g.visible {
		return g.detectRepository()
	}
	return nil
}

// detectRepository is a helper method that triggers repository detection.
// It calls the GitClient to check if the current directory is a Git repository
// and returns a command that will send a GitRepositoryDetectedMsg when complete.
//
// Requirements: 2.1, 2.2, 2.3
func (g *GitPane) detectRepository() tea.Cmd {
	return func() tea.Msg {
		// If gitClient is nil, return empty result
		if g.gitClient == nil {
			return GitRepositoryDetectedMsg{
				IsRepo:      false,
				Credentials: nil,
				RemoteURL:   "",
			}
		}

		// Call GitClient to detect repository
		info, err := g.gitClient.DetectRepository()
		if err != nil {
			// If detection fails, return empty result
			return GitRepositoryDetectedMsg{
				IsRepo:      false,
				Credentials: nil,
				RemoteURL:   "",
			}
		}

		// Return the detection result
		return GitRepositoryDetectedMsg{
			IsRepo:      info.IsRepo,
			Credentials: info.Credentials,
			RemoteURL:   info.RemoteURL,
		}
	}
}

// executeClone triggers an async clone operation.
// It sets the isProcessing flag and returns a command that will execute the clone
// and send a GitCloneMsg when complete.
//
// Requirements: 7.1
func (g *GitPane) executeClone(url, username, password string) tea.Cmd {
	g.isProcessing = true
	
	return func() tea.Msg {
		return GitCloneMsg{
			URL:      url,
			Username: username,
			Password: password,
		}
	}
}

// executePull triggers an async pull operation.
// It sets the isProcessing flag and returns a command that will execute the pull
// and send a GitPullMsg when complete.
//
// Requirements: 8.1
func (g *GitPane) executePull(username, password string) tea.Cmd {
	g.isProcessing = true
	
	return func() tea.Msg {
		return GitPullMsg{
			Username: username,
			Password: password,
		}
	}
}

// executePush triggers an async push operation.
// It sets the isProcessing flag and returns a command that will execute the push
// and send a GitPushMsg when complete.
//
// Requirements: 9.1
func (g *GitPane) executePush(username, password string) tea.Cmd {
	g.isProcessing = true
	
	return func() tea.Msg {
		return GitPushMsg{
			Username: username,
			Password: password,
		}
	}
}

// executeFetch triggers an async fetch operation.
// It sets the isProcessing flag and returns a command that will execute the fetch
// and send a GitFetchMsg when complete.
//
// Requirements: 10.1
func (g *GitPane) executeFetch(username, password string) tea.Cmd {
	g.isProcessing = true
	
	return func() tea.Msg {
		return GitFetchMsg{
			Username: username,
			Password: password,
		}
	}
}

// executeStage triggers an async stage operation.
// It sets the isProcessing flag and returns a command that will execute the stage
// and send a GitStageMsg when complete.
//
// Requirements: 11.1
func (g *GitPane) executeStage() tea.Cmd {
	g.isProcessing = true
	
	return func() tea.Msg {
		return GitStageMsg{}
	}
}

// executeCommit triggers an async commit operation.
// It sets the isProcessing flag and returns a command that will execute the commit
// and send a GitCommitMsg when complete.
func (g *GitPane) executeCommit(message string) tea.Cmd {
	g.isProcessing = true
	
	return func() tea.Msg {
		return GitCommitMsg{
			Message: message,
		}
	}
}

// executeStatus triggers an async status operation.
// It sets the isProcessing flag and returns a command that will execute the status
// and send a GitStatusMsg when complete.
//
// Requirements: 12.1
func (g *GitPane) executeStatus() tea.Cmd {
	g.isProcessing = true
	
	return func() tea.Msg {
		return GitStatusMsg{}
	}
}

// executeRestore triggers an async restore operation.
// It sets the isProcessing flag and returns a command that will execute the restore
// and send a GitRestoreMsg when complete.
//
// Requirements: 13.1
func (g *GitPane) executeRestore() tea.Cmd {
	g.isProcessing = true
	
	return func() tea.Msg {
		return GitRestoreMsg{}
	}
}

// Update handles incoming messages and updates the GitPane state accordingly.
// This method is part of the tea.Model interface from bubbletea.
// It handles GitRepositoryDetectedMsg to populate input fields with detected credentials
// and keyboard input for navigation and interaction.
//
// Requirements: 2.1, 2.2, 2.3, 3.1, 3.2, 3.3, 3.4, 3.5
func (g *GitPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case GitRepositoryDetectedMsg:
		// Handle repository detection result
		if msg.IsRepo && msg.Credentials != nil {
			// Repository detected with credentials - populate input fields
			g.urlInput.SetValue(msg.Credentials.URL)
			g.userInput.SetValue(msg.Credentials.Username)
			g.passInput.SetValue(msg.Credentials.Password)
		} else if msg.IsRepo && msg.RemoteURL != "" {
			// Repository detected but no stored credentials - populate URL only, clear others
			g.urlInput.SetValue(msg.RemoteURL)
			g.userInput.SetValue("")
			g.passInput.SetValue("")
		} else {
			// Not a repository - leave fields empty
			g.urlInput.SetValue("")
			g.userInput.SetValue("")
			g.passInput.SetValue("")
		}

	case GitOperationCompleteMsg:
		// Handle Git operation completion result
		// Clear the isProcessing flag
		g.isProcessing = false

		if msg.Success {
			// Operation succeeded - update statusMessage and clear errorMessage
			g.statusMessage = msg.Message
			g.errorMessage = ""

			// For successful clone operations, prepare directory change message
			// The parent App will handle the actual directory change
			if msg.Operation == "clone" && msg.NewDir != "" {
				// The App will receive this message and change the working directory
				// For now, we just update the status message
				g.statusMessage = "Clone completed successfully: " + msg.NewDir
			}
		} else {
			// Operation failed - update errorMessage and clear statusMessage
			if msg.Error != nil {
				g.errorMessage = msg.Error.Error()
			} else {
				g.errorMessage = msg.Message
			}
			g.statusMessage = ""
		}

	case GitCloneMsg:
		// Handle clone operation - execute async via GitClient
		return g, func() tea.Msg {
			result, _ := g.gitClient.Clone(msg.URL, msg.Username, msg.Password, "")
			return GitOperationCompleteMsg{
				Operation: "clone",
				Success:   result.Success,
				Message:   result.Message,
				Error:     result.Error,
				NewDir:    result.Message, // For clone, Message contains the cloned directory path
			}
		}

	case GitPullMsg:
		// Handle pull operation - execute async via GitClient
		return g, func() tea.Msg {
			result, _ := g.gitClient.Pull(msg.Username, msg.Password)
			return GitOperationCompleteMsg{
				Operation: "pull",
				Success:   result.Success,
				Message:   result.Message,
				Error:     result.Error,
			}
		}

	case GitPushMsg:
		// Handle push operation - execute async via GitClient
		return g, func() tea.Msg {
			result, _ := g.gitClient.Push(msg.Username, msg.Password)
			return GitOperationCompleteMsg{
				Operation: "push",
				Success:   result.Success,
				Message:   result.Message,
				Error:     result.Error,
			}
		}

	case GitFetchMsg:
		// Handle fetch operation - execute async via GitClient
		return g, func() tea.Msg {
			result, _ := g.gitClient.Fetch(msg.Username, msg.Password)
			return GitOperationCompleteMsg{
				Operation: "fetch",
				Success:   result.Success,
				Message:   result.Message,
				Error:     result.Error,
			}
		}

	case GitStageMsg:
		// Handle stage operation - execute async via GitClient
		return g, func() tea.Msg {
			result, _ := g.gitClient.Stage()
			return GitOperationCompleteMsg{
				Operation: "stage",
				Success:   result.Success,
				Message:   result.Message,
				Error:     result.Error,
			}
		}

	case GitCommitMsg:
		// Handle commit operation - execute async via GitClient
		return g, func() tea.Msg {
			result, _ := g.gitClient.Commit(msg.Message)
			return GitOperationCompleteMsg{
				Operation: "commit",
				Success:   result.Success,
				Message:   result.Message,
				Error:     result.Error,
			}
		}

	case GitStatusMsg:
		// Handle status operation - execute async via GitClient
		return g, func() tea.Msg {
			result, _ := g.gitClient.Status()
			return GitOperationCompleteMsg{
				Operation: "status",
				Success:   result.Success,
				Message:   result.Message,
				Error:     result.Error,
			}
		}

	case GitRestoreMsg:
		// Handle restore operation - execute async via GitClient
		return g, func() tea.Msg {
			result, _ := g.gitClient.Restore()
			return GitOperationCompleteMsg{
				Operation: "restore",
				Success:   result.Success,
				Message:   result.Message,
				Error:     result.Error,
			}
		}

	case tea.KeyMsg:
		// Handle keyboard input for navigation and interaction
		switch msg.String() {
		case "esc":
			// Esc closes the popup
			g.visible = false
			g.statusMessage = ""
			g.errorMessage = ""
			return g, nil

		case "tab", "down":
			// Tab or Down moves focus forward: URL → USER → PASS → buttons
			g.focusedInput++
			if g.focusedInput > 3 {
				g.focusedInput = 0
			}
			g.updateFocus()

		case "shift+tab", "up":
			// Shift+Tab or Up moves focus backward: buttons → PASS → USER → URL
			g.focusedInput--
			if g.focusedInput < 0 {
				g.focusedInput = 3
			}
			g.updateFocus()

		case "enter":
			// Enter only activates buttons, does not move between fields
			if g.focusedInput == 3 {
				// Focused on buttons - activate selected button
				// Clear previous error message before starting new operation
				g.errorMessage = ""
				g.statusMessage = ""
				
				// Get input values
				url := g.urlInput.Value()
				username := g.userInput.Value()
				password := g.passInput.Value()
				
				// Trigger the appropriate Git operation based on selected button
				switch g.selectedButton {
				case 0: // Clone
					return g, g.executeClone(url, username, password)
				case 1: // Pull
					return g, g.executePull(username, password)
				case 2: // Fetch
					return g, g.executeFetch(username, password)
				case 3: // Stage
					return g, g.executeStage()
				case 4: // Commit
					return g, g.executeCommit("Update files")
				case 5: // Push
					return g, g.executePush(username, password)
				case 6: // Status
					return g, g.executeStatus()
				case 7: // Restore
					return g, g.executeRestore()
				}
				return g, nil
			}
			// If focused on input field, Enter does nothing (use Tab/Down to move)

		case "left", "right":
			// Arrow keys navigate between buttons when focused on buttons
			if g.focusedInput == 3 {
				if msg.String() == "left" {
					g.selectedButton--
					if g.selectedButton < 0 {
						g.selectedButton = 7 // Wrap to last button (Restore)
					}
				} else { // "right"
					g.selectedButton++
					if g.selectedButton > 7 {
						g.selectedButton = 0 // Wrap to first button (Clone)
					}
				}
			}
		}
	}

	// Update the focused input field with keyboard input
	// Only update if we're focused on an input field (not buttons)
	if g.focusedInput < 3 {
		switch g.focusedInput {
		case 0:
			g.urlInput, cmd = g.urlInput.Update(msg)
		case 1:
			g.userInput, cmd = g.userInput.Update(msg)
		case 2:
			g.passInput, cmd = g.passInput.Update(msg)
		}
	}

	return g, cmd
}

// updateFocus updates the focus state of input fields based on focusedInput.
// It ensures only the currently focused input field is active.
func (g *GitPane) updateFocus() {
	// Blur all input fields first
	g.urlInput.Blur()
	g.userInput.Blur()
	g.passInput.Blur()

	// Focus the appropriate input field
	switch g.focusedInput {
	case 0:
		g.urlInput.Focus()
	case 1:
		g.userInput.Focus()
	case 2:
		g.passInput.Focus()
	case 3:
		// Focused on buttons - no input field should be focused
	}
}

// View renders the GitPane as a popup overlay.
// This method is part of the tea.Model interface from bubbletea.
// It returns an empty string when the pane is not visible.
//
// Requirements: 1.3, 3.1, 3.2, 3.3, 3.4, 3.5
func (g *GitPane) View() string {
	if !g.visible {
		return ""
	}

	// Define styles
	popupStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(84)

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("63")).
		Padding(0, 1)

	buttonSelectedStyle := buttonStyle.Copy().
		Background(lipgloss.Color("205")).
		Bold(true)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	processingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Bold(true)

	// Build the content
	var content strings.Builder

	// Title
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Git Operations"))
	content.WriteString("\n\n")

	// Input fields
	content.WriteString(g.urlInput.View())
	content.WriteString("\n")
	content.WriteString(g.userInput.View())
	content.WriteString("\n")
	content.WriteString(g.passInput.View())
	content.WriteString("\n\n")

	// Buttons - reordered and grouped: Clone Pull Fetch | Stage Commit Push | Status Restore
	buttonNames := []string{"Clone", "Pull", "Fetch", "Stage", "Commit", "Push", "Status", "Restore"}
	var buttons []string
	for i, name := range buttonNames {
		if g.focusedInput == 3 && g.selectedButton == i {
			buttons = append(buttons, buttonSelectedStyle.Render(name))
		} else {
			buttons = append(buttons, buttonStyle.Render(name))
		}
	}
	
	// Create button row with spacing and separators
	// Group 1: Clone Pull Fetch (remote operations)
	buttonRow := buttons[0] + "  " + buttons[1] + "  " + buttons[2]
	// Separator
	buttonRow += "  |  "
	// Group 2: Stage Commit Push (local to remote workflow)
	buttonRow += buttons[3] + "  " + buttons[4] + "  " + buttons[5]
	// Separator
	buttonRow += "  |  "
	// Group 3: Status Restore (info and undo)
	buttonRow += buttons[6] + "  " + buttons[7]
	
	content.WriteString(buttonRow)
	content.WriteString("\n\n")

	// Status/Error messages
	if g.isProcessing {
		content.WriteString(processingStyle.Render("Processing..."))
		content.WriteString("\n")
	} else if g.errorMessage != "" {
		content.WriteString(errorStyle.Render("Error: " + g.errorMessage))
		content.WriteString("\n")
	} else if g.statusMessage != "" {
		content.WriteString(successStyle.Render(g.statusMessage))
		content.WriteString("\n")
	}

	// Render the popup with the content
	popup := popupStyle.Render(content.String())

	// Center the popup on screen
	if g.width > 0 && g.height > 0 {
		return lipgloss.Place(
			g.width,
			g.height,
			lipgloss.Center,
			lipgloss.Center,
			popup,
		)
	}

	return popup
}
