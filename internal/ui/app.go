// Package ui provides the terminal user interface components for the Terminal Intelligence (TI) application.
//
// This package implements a split-window CLI interface using the Bubble Tea framework:
//   - Left pane: Code editor with syntax highlighting and file editing
//   - Right pane: AI chat interface with conversation history
//
// Key Components:
//   - App: Main orchestrator that coordinates all UI components and handles application state
//   - EditorPane: File editor with cursor navigation, editing, and file management
//   - AIChatPane: AI interaction pane with streaming responses and code block extraction
//
// The UI supports both conversational AI assistance and agentic code fixing, where the AI
// can autonomously read, analyze, and modify files based on user requests.
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/user/terminal-intelligence/internal/agentic"
	"github.com/user/terminal-intelligence/internal/ai"
	"github.com/user/terminal-intelligence/internal/bedrock"
	"github.com/user/terminal-intelligence/internal/config"
	"github.com/user/terminal-intelligence/internal/filemanager"
	"github.com/user/terminal-intelligence/internal/gemini"
	"github.com/user/terminal-intelligence/internal/git"
	"github.com/user/terminal-intelligence/internal/installer"
	"github.com/user/terminal-intelligence/internal/ollama"
	"github.com/user/terminal-intelligence/internal/projectctx"
	"github.com/user/terminal-intelligence/internal/types"
)

// App is the main Bubble Tea application that orchestrates all components.
// It manages the overall application state, routes messages between components,
// and handles user interactions across the entire interface.
//
// The App coordinates:
//   - EditorPane: File editing and display
//   - AIChatPane: AI interactions and conversation history
//   - FileManager: File system operations
//   - AgenticCodeFixer: Autonomous code fixing workflow
//
// Key responsibilities:
//   - Window sizing and layout management
//   - Keyboard shortcut handling (Ctrl+S save, Ctrl+O open, Tab switch panes, etc.)
//   - Dialog management (file picker, exit confirmation, help)
//   - Message routing between panes
//   - Integration of agentic code fixing with editor and AI pane
//
// The App implements the Bubble Tea Model interface (Init, Update, View).
type App struct {
	config                    *types.AppConfig             // Application configuration
	editorPane                *EditorPane                  // Left pane: code editor
	aiPane                    *AIChatPane                  // Right pane: AI chat
	gitPane                   *GitPane                     // Git operations popup overlay
	fileManager               *filemanager.FileManager     // File system operations
	aiClient                  ai.AIClient                  // AI service client (Ollama or Gemini)
	agenticFixer              *agentic.AgenticCodeFixer    // Autonomous code fixing orchestrator
	projectFixer              *agentic.ProjectFixer        // Project-wide agentic fixer
	agenticProjectFixer       *agentic.AgenticProjectFixer // Project-wide agentic fixer with retry loop
	autonomousCreator         *agentic.AutonomousCreator   // Autonomous application builder
	activePane                types.PaneType               // Currently focused pane
	width                     int                          // Terminal width
	height                    int                          // Terminal height
	ready                     bool                         // Whether initial sizing is complete
	showExitConfirmation      bool                         // Whether exit confirmation dialog is showing
	showFilePrompt            bool                         // Whether file creation prompt is showing
	showFilePicker            bool                         // Whether file picker dialog is showing
	showFolderPicker          bool                         // Whether folder picker dialog is showing
	showFolderCreatePrompt    bool                         // Whether folder creation prompt is showing
	showBackupPicker          bool                         // Whether backup picker dialog is showing
	showChatLoader            bool                         // Whether chat loader dialog is showing
	showHelp                  bool                         // Whether help dialog is showing
	showLanguageInstallPrompt bool                         // Whether language install prompt is showing
	languageToInstall         string                       // Language name for installation prompt
	fileTypeForInstall        string                       // File type that triggered install check
	filePromptBuffer          string                       // Buffer for file name input
	folderCreateBuffer        string                       // Buffer for new folder name input
	fileList                  []string                     // List of files for picker
	folderList                []string                     // List of folders for picker
	backupList                []string                     // List of backups for picker
	chatList                  []string                     // List of saved chats for loader
	filePickerIndex           int                          // Selected index in file picker
	filePickerPath            string                       // Current path being browsed in file picker
	folderPickerIndex         int                          // Selected index in folder picker
	folderPickerPath          string                       // Current path being browsed in folder picker
	forceQuit                 bool                         // Whether to quit without save confirmation
	statusMessage             string                       // Status bar message
	pendingCodeInsert         string                       // Code waiting to be inserted after file creation
	buildNumber               string                       // Build number from git commits
	searchResults             []string                     // Files found in last search
	searchResultIndex         int                          // Current index in searchResults
	searchTerms               []string                     // Last search terms used
	lastPreviewRequest        string                       // Original /project request from the last preview run
	autonomousFileToOpen      string                       // File path to open after autonomous creation step
	projectCtxCache           *projectctx.ContextCache     // Cache for project context metadata
}

// New creates a new application instance with the provided configuration.
// If config is nil, uses default configuration.
//
// Initialization process:
//  1. Creates FileManager for file system operations
//  2. Creates AI client based on provider (Ollama or Gemini)
//  3. Initializes AgenticCodeFixer with AI client and model
//  4. Creates EditorPane and AIChatPane components
//  5. Sets initial state (editor pane active, no dialogs showing)
//
// The returned App is ready to be run with Bubble Tea's tea.NewProgram().
//
// Parameters:
//   - config: Application configuration (provider, model, API keys, workspace directory)
//
// Returns:
//   - *App: Initialized application instance
//
// New creates a new application instance with the provided configuration.
// If config is nil, uses default configuration.
//
// Initialization process:
//  1. Creates FileManager for file system operations
//  2. Creates AI client based on provider (Ollama or Gemini)
//  3. Initializes AgenticCodeFixer with AI client and model
//  4. Creates EditorPane and AIChatPane components
//  5. Sets initial state (editor pane active, no dialogs showing)
//
// The returned App is ready to be run with Bubble Tea's tea.NewProgram().
//
// Parameters:
//   - config: Application configuration (provider, model, API keys, workspace directory)
//   - buildNumber: Build number from git commits
//
// Returns:
//   - *App: Initialized application instance
func New(config *types.AppConfig, buildNumber string) *App {
	if config == nil {
		config = types.DefaultConfig()
	}

	// Initialize components
	fm := filemanager.NewFileManager(config.WorkspaceDir)

	// Create AI client based on provider
	var aiClient ai.AIClient
	if config.Provider == "gemini" {
		aiClient = gemini.NewGeminiClient(config.GeminiAPIKey)
	} else if config.Provider == "bedrock" {
		client, err := bedrock.NewBedrockClient(config.BedrockAPIKey, config.BedrockRegion)
		if err != nil {
			// Log error but continue with nil client - will be caught by availability check
			fmt.Fprintf(os.Stderr, "Failed to initialize Bedrock client: %v\n", err)
		}
		aiClient = client
	} else {
		aiClient = ollama.NewOllamaClient(config.OllamaURL)
	}

	// Initialize AgenticCodeFixer
	agenticFixer := agentic.NewAgenticCodeFixer(aiClient, config.DefaultModel)

	// Initialize ProjectFixer
	projectFixer := agentic.NewProjectFixer(aiClient, config.DefaultModel)

	// Initialize AgenticProjectFixer with a placeholder logger.
	// The real notify function is wired up after the App struct is created (see below).
	var fixNotify func(string)
	fixLogger := agentic.NewActionLogger(func(msg string) {
		if fixNotify != nil {
			fixNotify(msg)
		}
	})
	agenticProjectFixer := agentic.NewAgenticProjectFixer(aiClient, config.DefaultModel, fixLogger)

	// Initialize GitClient and GitPane
	gitClient := git.NewClient(config.WorkspaceDir)
	gitPane := NewGitPane(gitClient, config.WorkspaceDir)

	app := &App{
		config:               config,
		fileManager:          fm,
		aiClient:             aiClient,
		agenticFixer:         agenticFixer,
		projectFixer:         projectFixer,
		agenticProjectFixer:  agenticProjectFixer,
		editorPane:           NewEditorPane(fm),
		aiPane:               NewAIChatPane(aiClient, config.DefaultModel, config.Provider, config.WorkspaceDir),
		gitPane:              gitPane,
		autonomousCreator:    nil,
		activePane:           types.EditorPaneType,
		ready:                false,
		showExitConfirmation: false,
		forceQuit:            false,
		buildNumber:          buildNumber,
		searchResults:        []string{},
		searchResultIndex:    0,
		searchTerms:          []string{},
		projectCtxCache:      projectctx.NewContextCache(),
	}

	// Wire up the fix logger now that the App (and its aiPane) exist.
	fixNotify = func(msg string) {
		app.aiPane.DisplayNotification(msg)
	}

	return app
}

// Init initializes the application and returns initial command.
// This is part of the Bubble Tea Model interface.
// Currently returns nil as no initial commands are needed.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.aiPane.CheckAIAvailability(),
		tea.EnableBracketedPaste,
		func() tea.Msg {
			return OpenWorkspacePickerMsg{}
		},
	)
}

// Update handles messages and updates application state.
// This is the main message handler for the Bubble Tea Model interface.
//
// Message handling priority:
//  1. Custom messages (InsertCodeMsg, SendAIMessageMsg, AINotificationMsg)
//  2. Window sizing (tea.WindowSizeMsg)
//  3. Keyboard input (tea.KeyMsg)
//  4. Routing to active pane
//
// Dialog handling:
//   - Help dialog: Ctrl+H to toggle, Esc/Q to close
//   - File picker: Up/Down to navigate, Enter to open, Esc to cancel
//   - File prompt: Type filename, Enter to create/open, Esc to cancel
//   - Exit confirmation: Y to quit, N/Esc to cancel
//
// Global keyboard shortcuts:
//   - Ctrl+C/Ctrl+Q: Quit (with unsaved changes confirmation)
//   - Ctrl+O: Open file picker
//   - Ctrl+N: New file prompt
//   - Ctrl+T: Clear AI chat history
//   - Ctrl+H: Toggle help
//   - Tab: Switch between editor and AI panes
//   - Ctrl+S: Save file
//   - Ctrl+X: Close file
//   - Ctrl+Enter: Send AI message with editor context
//
// Parameters:
//   - msg: The message to handle (can be any type)
//
// Returns:
//   - tea.Model: Updated model (always returns *App)
//   - tea.Cmd: Command to execute (can be nil or batched commands)
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case OpenWorkspacePickerMsg:
		startDir := a.config.WorkspaceDir
		if startDir == "" {
			home, _ := os.UserHomeDir()
			startDir = home
		}

		dirs, err := a.fileManager.ListDirectories(startDir)
		if err != nil {
			a.statusMessage = "Error listing directories: " + err.Error()
			return a, nil
		}

		a.folderPickerPath = startDir
		a.folderList = append([]string{"[ Select Current Directory ]", ".. (Parent Directory)", "[ Create New Folder ]"}, dirs...)
		a.folderPickerIndex = 0
		a.showFolderPicker = true
		a.statusMessage = "Select a folder to set as workspace"
		return a, nil

	case SaveConfigMsg:
		// Handle config save
		configPath, err := config.ConfigFilePath()
		if err != nil {
			a.statusMessage = "Error: Unable to locate config file: " + err.Error()
			return a, nil
		}

		// Build JSONConfig from fields and values
		jcfg := &config.JSONConfig{}
		for i, field := range msg.Fields {
			switch field {
			case "agent":
				jcfg.Agent = msg.Values[i]
			case "model":
				jcfg.Model = msg.Values[i]
			case "gmodel":
				jcfg.GModel = msg.Values[i]
			case "bedrock_model":
				jcfg.BedrockModel = msg.Values[i]
			case "ollama_url":
				jcfg.OllamaURL = msg.Values[i]
			case "gemini_api":
				jcfg.GeminiAPI = msg.Values[i]
			case "bedrock_api":
				jcfg.BedrockAPI = msg.Values[i]
			case "bedrock_region":
				jcfg.BedrockRegion = msg.Values[i]
			case "workspace":
				jcfg.Workspace = msg.Values[i]
			case "autonomous":
				jcfg.Autonomous = msg.Values[i]
			}
		}

		// Validate config
		if err := config.Validate(jcfg); err != nil {
			a.statusMessage = "Config validation error: " + err.Error()
			return a, nil
		}

		// Save to file
		data, err := config.ToJSON(jcfg)
		if err != nil {
			a.statusMessage = "Error serializing config: " + err.Error()
			return a, nil
		}

		err = os.WriteFile(configPath, data, 0644)
		if err != nil {
			a.statusMessage = "Error saving config: " + err.Error()
			return a, nil
		}

		// Apply to current app config
		config.ApplyToAppConfig(jcfg, a.config)

		// Reinitialize AI client if provider or settings changed
		if a.config.Provider == "gemini" {
			a.aiClient = gemini.NewGeminiClient(a.config.GeminiAPIKey)
		} else if a.config.Provider == "bedrock" {
			client, err := bedrock.NewBedrockClient(a.config.BedrockAPIKey, a.config.BedrockRegion)
			if err != nil {
				a.statusMessage = "Failed to initialize Bedrock client: " + err.Error()
				return a, nil
			}
			a.aiClient = client
		} else {
			a.aiClient = ollama.NewOllamaClient(a.config.OllamaURL)
		}

		// Update AI pane with new client and model
		a.aiPane.aiClient = a.aiClient
		a.aiPane.model = a.config.DefaultModel
		a.aiPane.provider = a.config.Provider

		// Update agentic fixer
		a.agenticFixer = agentic.NewAgenticCodeFixer(a.aiClient, a.config.DefaultModel)

		// Update agentic project fixer
		fixLogger := agentic.NewActionLogger(func(msg string) {})
		a.agenticProjectFixer = agentic.NewAgenticProjectFixer(a.aiClient, a.config.DefaultModel, fixLogger)

		// Re-check AI availability with the new config
		a.aiPane.aiChecked = false
		a.aiPane.aiAvailable = false

		a.statusMessage = "Configuration saved successfully to " + configPath
		return a, a.aiPane.CheckAIAvailability()

	case LanguageCheckMsg:
		// Check if the required language runtime is installed
		langInstaller := installer.NewLanguageInstaller()
		installed, version := langInstaller.CheckLanguageForFile(msg.FileType)

		if installed {
			if version != "" {
				a.statusMessage = fmt.Sprintf("%s is installed: %s", msg.LanguageName, version)
			}
			return a, nil
		}

		// Language not installed, show prompt
		a.showLanguageInstallPrompt = true
		a.languageToInstall = msg.LanguageName
		a.fileTypeForInstall = msg.FileType
		return a, nil

	case LanguageInstallMsg:
		// Start installation process
		a.showLanguageInstallPrompt = false
		a.statusMessage = fmt.Sprintf("Installing %s...", msg.LanguageName)

		// Show initial message in AI pane immediately
		initialMsg := fmt.Sprintf("🚀 Starting %s Installation\n\n", msg.LanguageName)
		initialMsg += "This may take a few minutes. Please wait...\n\n"
		initialMsg += "Steps:\n"
		initialMsg += "1. Fetching latest version\n"
		initialMsg += "2. Detecting system architecture\n"
		initialMsg += "3. Downloading Go (~140MB)\n"
		initialMsg += "4. Removing old installation\n"
		initialMsg += "5. Extracting files (requires sudo)\n"
		initialMsg += "6. Updating shell configuration\n"
		initialMsg += "7. Verifying installation\n\n"
		initialMsg += "Installation in progress..."
		a.aiPane.DisplayNotification(initialMsg)

		// Run installation in background
		return a, func() tea.Msg {
			langInstaller := installer.NewLanguageInstaller()

			var output string
			var err error

			switch msg.LanguageName {
			case "Go":
				output, err = langInstaller.InstallGo()
			case "Python":
				output, err = langInstaller.InstallPython()
			default:
				err = fmt.Errorf("unsupported language: %s", msg.LanguageName)
			}

			return LanguageInstallResultMsg{
				Success: err == nil,
				Output:  output,
				Error:   err,
			}
		}

	case LanguageInstallResultMsg:
		// Handle installation result
		if msg.Success {
			a.statusMessage = fmt.Sprintf("%s installed successfully!", a.languageToInstall)

			// Display installation output in AI pane
			notification := fmt.Sprintf("✓ %s Installation Complete\n\n%s", a.languageToInstall, msg.Output)
			a.aiPane.DisplayNotification(notification)
		} else {
			a.statusMessage = fmt.Sprintf("%s installation failed: %s", a.languageToInstall, msg.Error.Error())

			// Display error in AI pane
			notification := fmt.Sprintf("✗ %s Installation Failed\n\n%s\n\nError: %s",
				a.languageToInstall, msg.Output, msg.Error.Error())
			a.aiPane.DisplayNotification(notification)
		}

		a.languageToInstall = ""
		a.fileTypeForInstall = ""
		return a, nil

	case InsertCodeMsg:
		// Handle code insertion from AI pane
		selectedCode := a.aiPane.GetSelectedCodeBlock()
		effectiveDir := msg.EffectiveDir
		a.statusMessage = "InsertCodeMsg received, code length: " + string(rune('0'+len(selectedCode)/100%10)) + string(rune('0'+len(selectedCode)/10%10)) + string(rune('0'+len(selectedCode)%10))
		if selectedCode != "" {
			// Check if a file is open
			if a.editorPane.currentFile != nil {
				// Append code to current file
				currentContent := a.editorPane.GetContent()
				if currentContent != "" {
					a.editorPane.SetContent(currentContent + "\n" + selectedCode)
				} else {
					a.editorPane.SetContent(selectedCode)
				}

				// Save the file automatically
				err := a.editorPane.SaveFile()
				if err != nil {
					a.statusMessage = "Code inserted but save failed: " + err.Error()
				} else {
					a.statusMessage = "Code inserted and saved to " + a.editorPane.currentFile.Filepath
				}

				// Switch to editor pane to show the result
				a.activePane = types.EditorPaneType
				a.editorPane.focused = true
				a.aiPane.focused = false
			} else {
				// No file open — load code into editor as unsaved buffer
				suggestedName := a.aiPane.GetSuggestedFilename()

				// Prepend effective dir to suggested filename when it differs from workspace root
				if suggestedName != "" && effectiveDir != "" && effectiveDir != a.config.WorkspaceDir {
					relDir, err := filepath.Rel(a.config.WorkspaceDir, effectiveDir)
					if err == nil && relDir != "." {
						suggestedName = filepath.Join(relDir, suggestedName)
					}
				}

				a.editorPane.SetContentUnsaved(selectedCode, suggestedName)

				// Switch to editor pane
				a.activePane = types.EditorPaneType
				a.editorPane.focused = true
				a.aiPane.focused = false

				if suggestedName != "" {
					// Show resolved save path when effective dir differs from workspace root
					if effectiveDir != "" && effectiveDir != a.config.WorkspaceDir {
						a.statusMessage = "Code loaded (save path: " + suggestedName + ") — Ctrl+S to save"
					} else {
						a.statusMessage = "Code loaded (suggested name: " + suggestedName + ") — Ctrl+S to save"
					}
				} else {
					a.statusMessage = "Code loaded — Ctrl+S to save with a filename"
				}
			}
		} else {
			a.statusMessage = "No code block selected"
		}
		return a, nil

	case OpenFileInEditorMsg:
		// Handle file opening from documentation generation
		err := a.editorPane.LoadFile(msg.FilePath)
		if err != nil {
			a.statusMessage = "Error opening file: " + err.Error()
		} else {
			a.statusMessage = "Opened: " + msg.FilePath
			// Switch to editor pane to show the file
			a.activePane = types.EditorPaneType
			a.editorPane.focused = true
			a.aiPane.focused = false
		}
		return a, nil

	case AgenticFixResultMsg:
		a.aiPane.streaming = false
		result := msg.Result

		// Handle fix results
		if result.IsConversational {
			// Should not happen if detected correctly, but handle gracefully
			return a, a.aiPane.SendMessage(result.ChangesSummary, "")
			// Wait, ChangesSummary might be empty/wrong if conversational.
			// Actually ProcessMessage returns conversational result with empty content?
			// Let's just log or ignore if it happens unexpectedly for now, or display error.
		}

		if result.Success {
			// In preview mode, don't apply the fix to the editor
			// Just show the changes summary
			if !result.PreviewMode {
				// Apply the fix to the editor
				a.editorPane.SetContent(result.ModifiedContent)
			}

			// Show notification in chat
			a.aiPane.DisplayNotification(result.ChangesSummary)
		} else {
			// Handle fix failure
			a.aiPane.DisplayNotification(result.ErrorMessage)
		}
		return a, nil

	case SendAIMessageMsg:
		// Handle AI message through the new handleAIMessage method
		cmd := a.handleAIMessage(msg.Message)
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)

	case GitOperationCompleteMsg:
		// Handle Git operation completion
		if msg.Operation == "clone" && msg.Success && msg.NewDir != "" {
			// Change working directory to the newly cloned repository
			a.projectCtxCache.Invalidate(a.config.WorkspaceDir) // Invalidate cache for old workspace (Req 2.3)
			a.config.WorkspaceDir = msg.NewDir

			// Actually change the process working directory
			if err := os.Chdir(msg.NewDir); err != nil {
				a.statusMessage = "Cloned successfully but failed to change directory: " + err.Error()
			} else {
				// Update FileManager workspace directory
				a.fileManager.SetWorkspaceDir(msg.NewDir)

				// Update GitPane working directory
				cmd := a.gitPane.SetWorkDir(msg.NewDir)
				cmds = append(cmds, cmd)

				// Update AIChatPane workspace root
				a.aiPane.SetWorkspaceRoot(msg.NewDir)

				// Save workspace to config file
				if err := config.UpdateWorkspace(msg.NewDir); err != nil {
					a.statusMessage = "Cloned successfully. Changed directory to: " + msg.NewDir + " (config update failed: " + err.Error() + ")"
				} else {
					a.statusMessage = "Cloned successfully. Changed directory to: " + msg.NewDir
				}

				// Close the Git UI after successful clone
				a.gitPane.Toggle()
			}
		}

		// Forward the message to GitPane for status display
		_, cmd := a.gitPane.Update(msg)
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true

		// Account for header (3 lines), editor title bar (3 lines), status bar (1 line)
		paneHeight := msg.Height - 7

		// Width budget:
		// Editor View() uses Border + Width(w-4) → rendered width = w - 4 (content) + 2 (border) = w - 2
		// AI pane View() wraps everything in a container with Width(w) → rendered width = w
		// Total must equal msg.Width: (editorW - 2) + aiW = msg.Width
		// So: editorW + aiW = msg.Width + 2
		halfWidth := msg.Width / 2
		editorWidth := halfWidth + 2           // renders as halfWidth wide
		aiWidth := msg.Width + 2 - editorWidth // renders as msg.Width - halfWidth wide

		// Update editor pane size
		a.editorPane.width = editorWidth
		a.editorPane.height = paneHeight

		// Update AI pane size
		a.aiPane.width = aiWidth
		a.aiPane.height = paneHeight

		// Update GitPane size for proper centering
		a.gitPane.width = msg.Width
		a.gitPane.height = msg.Height

		return a, nil

	case AutonomousTickMsg:
		if a.autonomousCreator == nil {
			return a, nil
		}

		status, err := a.autonomousCreator.Step()
		if err != nil {
			a.aiPane.DisplayNotification("Autonomous Creation Error: " + err.Error())
			a.autonomousCreator = nil // Reset state on error
			return a, nil
		}

		if status != "" {
			a.aiPane.DisplayNotification(status)
		}

		// Check if there's a file to open (e.g., SUMMARY.md) and open it immediately
		if a.autonomousFileToOpen != "" {
			filePath := a.autonomousFileToOpen
			a.autonomousFileToOpen = "" // Clear it after capturing

			// Open the file directly in the editor pane
			err := a.editorPane.LoadFile(filePath)
			if err != nil {
				a.statusMessage = "Error opening file: " + err.Error()
			} else {
				a.statusMessage = "Opened: " + filePath
				// Switch to editor pane to show the file
				a.activePane = types.EditorPaneType
				a.editorPane.focused = true
				a.aiPane.focused = false
			}
		}

		// If the state is not waiting for user or done, queue the next tick to keep it going.
		// For waiting states, we pause the loop until the user proceeds.
		if a.autonomousCreator.State != agentic.StateWaitingApproval && a.autonomousCreator.State != agentic.StateDone {
			tickCmd := func() tea.Msg {
				// yield to the UI event loop momentarily to redraw
				return AutonomousTickMsg{}
			}
			return a, tickCmd
		}

		if a.autonomousCreator.State == agentic.StateDone {
			a.autonomousCreator = nil // Process complete, reset
		}

		return a, nil

	case ProjectCompleteMsg:
		a.aiPane.streaming = false
		// Store the bare request for /proceed if this was a preview run.
		if msg.LastPreviewRequest != "" {
			a.lastPreviewRequest = msg.LastPreviewRequest
		}
		a.aiPane.DisplayNotification(msg.Formatted)

		// Open modified files sequentially into the editor panel.
		// Preview-mode runs don't write files, so skip loading.
		if msg.Report != nil && !msg.Report.PreviewMode && len(msg.Report.FilesModified) > 0 {
			paths := make([]string, 0, len(msg.Report.FilesModified))
			for _, f := range msg.Report.FilesModified {
				if f.Path != "" {
					paths = append(paths, f.Path)
				}
			}
			if len(paths) > 0 {
				// Load the first file immediately.
				if err := a.editorPane.LoadFile(paths[0]); err == nil {
					a.activePane = types.EditorPaneType
					a.editorPane.focused = true
					a.aiPane.focused = false
				}
				// Queue the rest.
				if len(paths) > 1 {
					cmds = append(cmds, func() tea.Msg {
						return ProjectFileOpenMsg{Paths: paths[1:]}
					})
				}
			}
		}
		return a, tea.Batch(cmds...)

	case FixSessionCompleteMsg:
		a.aiPane.streaming = false

		if msg.Error != nil {
			a.aiPane.DisplayNotification("Fix session error: " + msg.Error.Error())
			return a, nil
		}

		result := msg.Result
		if result == nil {
			a.aiPane.DisplayNotification("Fix session returned no result.")
			return a, nil
		}

		// Build a summary to display
		if result.Success && result.FinalReport != nil {
			formatted := agentic.FormatChangeReport(result.FinalReport)
			summary := fmt.Sprintf("✅ Fix successful after %d attempt(s) across %d cycle(s).\n\n%s",
				result.TotalAttempts, result.TotalCycles, formatted)
			a.aiPane.DisplayNotification(summary)

			// Open modified files in the editor
			if len(result.FinalReport.FilesModified) > 0 {
				paths := make([]string, 0, len(result.FinalReport.FilesModified))
				for _, f := range result.FinalReport.FilesModified {
					if f.Path != "" {
						paths = append(paths, f.Path)
					}
				}
				if len(paths) > 0 {
					if err := a.editorPane.LoadFile(paths[0]); err == nil {
						a.activePane = types.EditorPaneType
						a.editorPane.focused = true
						a.aiPane.focused = false
					}
					if len(paths) > 1 {
						cmds = append(cmds, func() tea.Msg {
							return ProjectFileOpenMsg{Paths: paths[1:]}
						})
					}
				}
			}
		} else {
			errMsg := result.ErrorMessage
			if errMsg == "" {
				errMsg = "Fix session completed without success."
			}
			summary := fmt.Sprintf("❌ %s\nAttempts: %d, Cycles: %d",
				errMsg, result.TotalAttempts, result.TotalCycles)
			a.aiPane.DisplayNotification(summary)
		}

		return a, tea.Batch(cmds...)

	case ProjectFileOpenMsg:
		if len(msg.Paths) == 0 {
			return a, nil
		}
		// Open the next file.
		if err := a.editorPane.LoadFile(msg.Paths[0]); err == nil {
			a.activePane = types.EditorPaneType
			a.editorPane.focused = true
			a.aiPane.focused = false
		}
		// Queue remaining files (last file stays open when list is exhausted).
		if len(msg.Paths) > 1 {
			remaining := msg.Paths[1:]
			cmds = append(cmds, func() tea.Msg {
				return ProjectFileOpenMsg{Paths: remaining}
			})
		}
		return a, tea.Batch(cmds...)

	case SearchCompleteMsg:
		a.aiPane.streaming = false
		totalResults := len(msg.ExactResults) + len(msg.AltResults)
		if totalResults > 0 {
			a.searchResults = append(msg.ExactResults, msg.AltResults...)
			a.searchTerms = strings.Split(msg.SearchTerm, ", ")
			a.searchResultIndex = 0

			a.openSearchResult()

			var chatMsg strings.Builder
			chatMsg.WriteString(fmt.Sprintf("🔍 Found '%s' in %d file(s):\n", msg.SearchTerm, totalResults))

			if len(msg.ExactResults) > 0 {
				chatMsg.WriteString("Exact search pattern:\n")
				for i, res := range msg.ExactResults {
					if i >= 3 {
						chatMsg.WriteString(fmt.Sprintf("- ... and %d more exact matches\n", len(msg.ExactResults)-3))
						break
					}
					chatMsg.WriteString("- " + res + "\n")
				}
			}

			if len(msg.AltResults) > 0 {
				chatMsg.WriteString("\nAlternative variations:\n")
				for i, res := range msg.AltResults {
					if i >= 3 {
						chatMsg.WriteString(fmt.Sprintf("- ... and %d more\n", len(msg.AltResults)-3))
						break
					}
					chatMsg.WriteString("- " + res + "\n")
				}
			}

			matchType := "exact match"
			if len(msg.ExactResults) == 0 {
				matchType = "alternative variation"
			}
			chatMsg.WriteString(fmt.Sprintf("\nOpening the first match: %s (%s)", a.searchResults[0], matchType))
			chatMsg.WriteString("\n\n*Tip: Use `Alt+N` (Next) and `Alt+P` (Previous) to jump between these files.*")
			a.aiPane.DisplayNotification(chatMsg.String())
		}
		return a, tea.Batch(cmds...)

	case TerminalOutputMsg, TerminalDoneMsg, AIResponseMsg, AINotificationMsg, AIAvailabilityMsg, ClearStatusMsg, DocPipelineMsg:
		// Handle ClearStatusMsg
		if _, ok := msg.(ClearStatusMsg); ok {
			a.statusMessage = ""
		}
		cmd := a.aiPane.Update(msg)
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Handle help dialog
		if a.showHelp {
			switch msg.String() {
			case "esc", "ctrl+h", "q":
				a.showHelp = false
			}
			return a, nil
		}

		// Handle folder creation prompt dialog
		if a.showFolderCreatePrompt {
			switch msg.String() {
			case "enter":
				// Create new folder
				if a.folderCreateBuffer != "" {
					newFolderPath := filepath.Join(a.folderPickerPath, a.folderCreateBuffer)

					// Check if folder already exists
					if _, err := os.Stat(newFolderPath); err == nil {
						a.statusMessage = "Folder already exists: " + a.folderCreateBuffer
					} else {
						// Create the folder
						if err := a.fileManager.CreateDirectory(newFolderPath); err != nil {
							a.statusMessage = "Error creating folder: " + err.Error()
						} else {
							a.statusMessage = "Created folder: " + a.folderCreateBuffer

							// Refresh folder list to show the new folder
							dirs, err := a.fileManager.ListDirectories(a.folderPickerPath)
							if err != nil {
								a.statusMessage = "Error refreshing folder list: " + err.Error()
							} else {
								a.folderList = append([]string{"[ Select Current Directory ]", ".. (Parent Directory)", "[ Create New Folder ]"}, dirs...)
								// Find and select the newly created folder
								for i, dir := range a.folderList {
									if dir == a.folderCreateBuffer {
										a.folderPickerIndex = i
										break
									}
								}
							}
						}
					}
				}
				a.showFolderCreatePrompt = false
				a.showFolderPicker = true
				a.folderCreateBuffer = ""
				return a, nil
			case "esc":
				// Cancel folder creation and return to folder picker
				a.showFolderCreatePrompt = false
				a.showFolderPicker = true
				a.folderCreateBuffer = ""
				a.statusMessage = "Folder creation cancelled"
				return a, nil
			case "backspace":
				if len(a.folderCreateBuffer) > 0 {
					a.folderCreateBuffer = a.folderCreateBuffer[:len(a.folderCreateBuffer)-1]
				}
				return a, nil
			default:
				// Add character to buffer
				if len(msg.String()) == 1 {
					a.folderCreateBuffer += msg.String()
				}
				return a, nil
			}
		}

		// Handle folder picker dialog
		if a.showFolderPicker {
			switch msg.String() {
			case "up", "k":
				if a.folderPickerIndex > 0 {
					a.folderPickerIndex--
				}
				return a, nil
			case "down", "j":
				if a.folderPickerIndex < len(a.folderList)-1 {
					a.folderPickerIndex++
				}
				return a, nil
			case "enter":
				if len(a.folderList) > 0 && a.folderPickerIndex < len(a.folderList) {
					selected := a.folderList[a.folderPickerIndex]

					if selected == "[ Select Current Directory ]" {
						// Set new workspace directory
						newDir := a.folderPickerPath
						a.projectCtxCache.Invalidate(a.config.WorkspaceDir) // Invalidate cache for old workspace (Req 2.3)
						a.config.WorkspaceDir = newDir

						if err := os.Chdir(newDir); err != nil {
							a.statusMessage = "Failed to change directory: " + err.Error()
						} else {
							a.fileManager.SetWorkspaceDir(newDir)
							a.gitPane.SetWorkDir(newDir)
							a.aiPane.SetWorkspaceRoot(newDir)
							a.editorPane.CloseFile() // Close current file as it's outside new workspace

							// Save workspace to config file
							if err := config.UpdateWorkspace(newDir); err != nil {
								a.statusMessage = "Workspace changed to: " + newDir + " (config update failed: " + err.Error() + ")"
							} else {
								a.statusMessage = "Workspace changed to: " + newDir
							}
						}
						a.showFolderPicker = false
						a.folderList = nil
						return a, nil
					}

					if selected == "[ Create New Folder ]" {
						// Switch to folder creation mode
						a.showFolderPicker = false
						a.showFolderCreatePrompt = true
						a.folderCreateBuffer = ""
						a.statusMessage = "Enter name for new folder (Esc to cancel)"
						return a, nil
					}

					var nextDir string
					if selected == ".. (Parent Directory)" {
						nextDir = filepath.Dir(a.folderPickerPath)
					} else {
						nextDir = filepath.Join(a.folderPickerPath, selected)
					}

					// Refresh folder list for the new path
					dirs, err := a.fileManager.ListDirectories(nextDir)
					if err != nil {
						a.statusMessage = "Error: " + err.Error()
						return a, nil
					}

					a.folderPickerPath = nextDir
					a.folderList = append([]string{"[ Select Current Directory ]", ".. (Parent Directory)", "[ Create New Folder ]"}, dirs...)
					a.folderPickerIndex = 0
					return a, nil
				}
			case "esc":
				a.showFolderPicker = false
				a.folderList = nil
				return a, nil
			}
			return a, nil
		}

		// Handle file picker dialog
		if a.showFilePicker {
			switch msg.String() {
			case "up", "k":
				if a.filePickerIndex > 0 {
					a.filePickerIndex--
				}
				return a, nil
			case "down", "j":
				if a.filePickerIndex < len(a.fileList)-1 {
					a.filePickerIndex++
				}
				return a, nil
			case "enter":
				// Open selected file or folder
				if len(a.fileList) > 0 && a.filePickerIndex < len(a.fileList) {
					selected := a.fileList[a.filePickerIndex]

					// If it ends with '/' or is '..', it's a directory
					if strings.HasSuffix(selected, "/") || selected == ".." {
						var nextDir string
						if selected == ".." {
							nextDir = filepath.Dir(a.filePickerPath)
						} else {
							nextDir = filepath.Join(a.filePickerPath, strings.TrimSuffix(selected, "/"))
						}

						// Refresh file list for the new path
						dirs, files, err := a.fileManager.ListEntries(nextDir)
						if err != nil {
							a.statusMessage = "Error: " + err.Error()
							return a, nil
						}

						var newList []string
						newList = append(newList, "..")
						for _, d := range dirs {
							newList = append(newList, d+"/")
						}
						newList = append(newList, files...)

						a.fileList = newList
						a.filePickerPath = nextDir
						a.filePickerIndex = 0
						return a, nil
					}

					// It's a file, open it
					fullPath := filepath.Join(a.filePickerPath, selected)

					// Debug: Log the file being opened
					a.statusMessage = "Opening: " + selected

					err := a.editorPane.LoadFile(fullPath)
					if err != nil {
						a.statusMessage = "Error opening file: " + err.Error()
					} else {
						a.statusMessage = "Successfully opened: " + selected
						a.aiPane.SetWorkingDir(filepath.Dir(a.editorPane.currentFile.Filepath))
						// Switch to editor pane after opening file
						a.activePane = types.EditorPaneType
						a.editorPane.focused = true
						a.aiPane.focused = false
						a.showFilePicker = false
						a.fileList = nil
					}
				}
				return a, nil
			case "esc":
				// Cancel file picker
				a.showFilePicker = false
				a.fileList = nil
				a.filePickerIndex = 0
				return a, nil
			}
			return a, nil
		}

		// Handle chat loader dialog
		if a.showChatLoader {
			switch msg.String() {
			case "up", "k":
				if a.filePickerIndex > 0 {
					a.filePickerIndex--
				}
				return a, nil
			case "down", "j":
				if a.filePickerIndex < len(a.chatList)-1 {
					a.filePickerIndex++
				}
				return a, nil
			case "enter":
				// Load selected chat
				if len(a.chatList) > 0 && a.filePickerIndex < len(a.chatList) {
					selectedChat := a.chatList[a.filePickerIndex]
					chatPath := filepath.Join(a.config.WorkspaceDir, ".ti", selectedChat)

					content, err := os.ReadFile(chatPath)
					if err != nil {
						a.statusMessage = "Error reading chat: " + err.Error()
					} else {
						// Parse and load chat into AI pane
						err = a.loadChatHistory(string(content))
						if err != nil {
							a.statusMessage = "Error loading chat: " + err.Error()
						} else {
							a.statusMessage = "Loaded chat: " + selectedChat
							// Switch to AI pane
							a.activePane = types.AIPaneType
							a.editorPane.focused = false
							a.aiPane.focused = true
						}
					}
				}
				a.showChatLoader = false
				a.chatList = nil
				a.filePickerIndex = 0
				return a, nil
			case "esc":
				// Cancel chat loader
				a.showChatLoader = false
				a.chatList = nil
				a.filePickerIndex = 0
				return a, nil
			}
			return a, nil
		}

		// Handle backup picker dialog
		if a.showBackupPicker {
			switch msg.String() {
			case "up", "k":
				if a.filePickerIndex > 0 {
					a.filePickerIndex--
				}
				return a, nil
			case "down", "j":
				if a.filePickerIndex < len(a.backupList)-1 {
					a.filePickerIndex++
				}
				return a, nil
			case "enter":
				// Open selected backup
				if len(a.backupList) > 0 && a.filePickerIndex < len(a.backupList) {
					// Restore backup content to editor (keep original file path)
					selectedBackup := a.backupList[a.filePickerIndex]
					backupPath := filepath.Join(a.config.WorkspaceDir, ".ti", selectedBackup)

					content, err := os.ReadFile(backupPath)
					if err != nil {
						a.statusMessage = "Error reading backup: " + err.Error()
					} else {
						a.editorPane.SetContent(string(content))
						a.statusMessage = "Restored backup: " + selectedBackup + " (unsaved)"
						// Switch to editor pane
						a.activePane = types.EditorPaneType
						a.editorPane.focused = true
						a.aiPane.focused = false
					}
				}
				a.showBackupPicker = false
				a.backupList = nil
				a.filePickerIndex = 0
				return a, nil
			case "esc":
				// Cancel backup picker
				a.showBackupPicker = false
				a.backupList = nil
				a.filePickerIndex = 0
				return a, nil
			}
			return a, nil
		}

		// Handle file prompt dialog
		if a.showFilePrompt {
			switch msg.String() {
			case "enter":
				// Create/open file
				if a.filePromptBuffer != "" {
					err := a.editorPane.LoadFile(a.filePromptBuffer)
					if err != nil {
						// File doesn't exist, create it
						err = a.fileManager.CreateFile(a.filePromptBuffer, "")
						if err != nil {
							a.statusMessage = "Error: " + err.Error()
						} else {
							a.editorPane.LoadFile(a.filePromptBuffer)
							a.statusMessage = "Created: " + a.filePromptBuffer
							a.aiPane.SetWorkingDir(filepath.Dir(a.editorPane.currentFile.Filepath))

							// Insert pending code if any
							if a.pendingCodeInsert != "" {
								currentContent := a.editorPane.GetContent()
								if currentContent != "" {
									a.editorPane.SetContent(currentContent + "\n" + a.pendingCodeInsert)
								} else {
									a.editorPane.SetContent(a.pendingCodeInsert)
								}
								a.statusMessage = "Created file and inserted code"
								a.pendingCodeInsert = ""
							}
						}
					} else {
						a.statusMessage = "Opened: " + a.filePromptBuffer
						a.aiPane.SetWorkingDir(filepath.Dir(a.editorPane.currentFile.Filepath))

						// Insert pending code if any
						if a.pendingCodeInsert != "" {
							currentContent := a.editorPane.GetContent()
							if currentContent != "" {
								a.editorPane.SetContent(currentContent + "\n" + a.pendingCodeInsert)
							} else {
								a.editorPane.SetContent(a.pendingCodeInsert)
							}
							a.statusMessage = "Opened file and inserted code"
							a.pendingCodeInsert = ""
						}
					}
				}
				a.showFilePrompt = false
				a.filePromptBuffer = ""
				return a, nil
			case "esc":
				// Cancel file prompt
				a.showFilePrompt = false
				a.filePromptBuffer = ""
				a.pendingCodeInsert = "" // Clear pending code on cancel
				return a, nil
			case "backspace":
				if len(a.filePromptBuffer) > 0 {
					a.filePromptBuffer = a.filePromptBuffer[:len(a.filePromptBuffer)-1]
				}
				return a, nil
			default:
				// Add character to buffer
				if len(msg.String()) == 1 {
					a.filePromptBuffer += msg.String()
				}
				return a, nil
			}
		}

		// Handle language install prompt dialog
		if a.showLanguageInstallPrompt {
			switch msg.String() {
			case "y", "Y":
				// Confirm installation
				return a, func() tea.Msg {
					return LanguageInstallMsg{LanguageName: a.languageToInstall}
				}
			case "n", "N", "esc":
				// Cancel installation
				a.showLanguageInstallPrompt = false
				a.languageToInstall = ""
				a.fileTypeForInstall = ""
				a.statusMessage = "Installation cancelled"
				return a, nil
			}
			return a, nil
		}

		// Handle exit confirmation dialog
		if a.showExitConfirmation {
			switch msg.String() {
			case "y", "Y":
				// Confirm exit without saving
				a.forceQuit = true
				a.aiPane.ClearHistory()
				return a, tea.Quit
			case "n", "N", "esc":
				// Cancel exit
				a.showExitConfirmation = false
				return a, nil
			}
			return a, nil
		}

		switch msg.String() {
		case "ctrl+q":
			// Check for unsaved changes
			if a.editorPane.HasUnsavedChanges() && !a.forceQuit {
				a.showExitConfirmation = true
				return a, nil
			}
			// Clear AI history on normal exit
			a.aiPane.ClearHistory()
			return a, tea.Quit

		case "ctrl+c":
			var textToCopy string

			if a.activePane == types.EditorPaneType {
				textToCopy = a.editorPane.GetCurrentLine()
			} else if a.activePane == types.AIPaneType || a.activePane == types.AIResponsePaneType {
				textToCopy = a.aiPane.GetSelectedCodeBlock()
			}

			if textToCopy != "" {
				err := clipboard.WriteAll(textToCopy)
				if err != nil {
					a.statusMessage = "Error copying to clipboard: " + err.Error()
				} else {
					a.statusMessage = "Copied to clipboard"
				}
			} else {
				a.statusMessage = "Nothing to copy"
			}
			return a, nil

		case "ctrl+b":
			// Open backup picker
			if a.editorPane.currentFile == nil {
				a.statusMessage = "No file open to list backups for"
				return a, nil
			}

			backups, err := a.fileManager.ListBackups(a.editorPane.currentFile.Filepath)
			if err != nil {
				a.statusMessage = "Error listing backups: " + err.Error()
				return a, nil
			}
			if len(backups) == 0 {
				a.statusMessage = "No backups found for this file"
				return a, nil
			}

			// Sort backups: newest first (reverse order)
			// Assuming timestamps YYYYMMDD-HHMMSS sort correctly lexicographically
			for i, j := 0, len(backups)-1; i < j; i, j = i+1, j-1 {
				backups[i], backups[j] = backups[j], backups[i]
			}

			a.backupList = backups
			a.filePickerIndex = 0
			a.showBackupPicker = true
			a.statusMessage = fmt.Sprintf("Found %d backups (Newest first)", len(backups))
			return a, nil

		case "ctrl+o":
			// Open file picker starting from workspace root
			startDir := a.config.WorkspaceDir
			if startDir == "" {
				home, _ := os.UserHomeDir()
				startDir = home
			}

			dirs, files, err := a.fileManager.ListEntries(startDir)
			if err != nil {
				a.statusMessage = "Error listing entries: " + err.Error()
				return a, nil
			}

			var fileList []string
			// Add folders first (with / suffix)
			fileList = append(fileList, "..")
			for _, d := range dirs {
				fileList = append(fileList, d+"/")
			}
			fileList = append(fileList, files...)

			if len(fileList) == 1 && fileList[0] == ".." { // Only parent dir found?
				a.statusMessage = "No files found in workspace"
				// Still allow it if they want to navigate up?
			}

			a.fileList = fileList
			a.filePickerPath = startDir
			a.filePickerIndex = 0
			a.showFilePicker = true
			a.statusMessage = "Select a file to open or folder to browse"
			return a, nil

		case "ctrl+w":
			// Open folder picker via message
			return a, func() tea.Msg {
				return OpenWorkspacePickerMsg{}
			}

		case "ctrl+n":
			// New file prompt
			a.showFilePrompt = true
			a.filePromptBuffer = ""
			return a, nil

		case "ctrl+t":
			// Clear AI chat history (New Chat)
			a.aiPane.ClearHistory()
			a.statusMessage = "AI chat history cleared"
			a.searchResults = []string{}
			return a, nil

		case "alt+n":
			if len(a.searchResults) > 0 {
				a.searchResultIndex++
				if a.searchResultIndex >= len(a.searchResults) {
					a.searchResultIndex = 0 // loop back to start
				}
				a.openSearchResult()
			} else {
				a.statusMessage = "No search results to jump to"
			}
			return a, nil

		case "alt+p":
			if len(a.searchResults) > 0 {
				a.searchResultIndex--
				if a.searchResultIndex < 0 {
					a.searchResultIndex = len(a.searchResults) - 1 // loop back to end
				}
				a.openSearchResult()
			} else {
				a.statusMessage = "No search results to jump to"
			}
			return a, nil

		case "ctrl+h":
			// Toggle help menu
			a.showHelp = !a.showHelp
			return a, nil

		case "ctrl+g":
			// Toggle Git UI
			cmd := a.gitPane.Toggle()
			return a, cmd

		case "ctrl+l":
			// Open chat loader to reload saved chats
			tiDir := filepath.Join(a.config.WorkspaceDir, ".ti")

			// Check if .ti directory exists
			if _, err := os.Stat(tiDir); os.IsNotExist(err) {
				a.statusMessage = "No saved chats found (.ti directory doesn't exist)"
				return a, nil
			}

			// List all chat-*.md files in .ti directory
			entries, err := os.ReadDir(tiDir)
			if err != nil {
				a.statusMessage = "Error reading .ti directory: " + err.Error()
				return a, nil
			}

			var chatFiles []string
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasPrefix(entry.Name(), "session_token_chat_") && strings.HasSuffix(entry.Name(), ".md") {
					chatFiles = append(chatFiles, entry.Name())
				}
			}

			if len(chatFiles) == 0 {
				a.statusMessage = "No saved chats found in .ti directory"
				return a, nil
			}

			// Sort chats: newest first (reverse order)
			for i, j := 0, len(chatFiles)-1; i < j; i, j = i+1, j-1 {
				chatFiles[i], chatFiles[j] = chatFiles[j], chatFiles[i]
			}

			a.chatList = chatFiles
			a.filePickerIndex = 0
			a.showChatLoader = true
			a.statusMessage = fmt.Sprintf("Found %d saved chats (Newest first)", len(chatFiles))
			return a, nil

		case "tab":
			// Cycle through: Editor → AI Input → AI Response → Editor
			if a.activePane == types.EditorPaneType {
				// Switch from Editor to AI Input
				a.activePane = types.AIPaneType
				a.editorPane.focused = false
				a.aiPane.focused = true
				a.aiPane.SetActiveArea(0) // Set to Input area
			} else if a.activePane == types.AIPaneType {
				// Check which AI area is active
				if a.aiPane.GetActiveArea() == 0 {
					// Switch from AI Input to AI Response
					a.aiPane.SetActiveArea(1) // Set to Response area
				} else {
					// Switch from AI Response back to Editor
					a.activePane = types.EditorPaneType
					a.editorPane.focused = true
					a.aiPane.focused = false
					a.aiPane.SetActiveArea(0) // Reset to Input area for next time
				}
			}
			return a, nil

		case "ctrl+s":
			// Save file in editor
			if a.activePane == types.EditorPaneType {
				if a.editorPane.currentFile == nil && a.editorPane.GetContent() != "" {
					// No file yet — check for AI-suggested name
					suggested := a.editorPane.GetSuggestedName()
					if suggested != "" {
						// Use the suggested name directly (may already include effective dir from InsertCodeMsg)
						filePath := suggested

						// If the suggested name is a plain filename (no directory component),
						// check if the AI pane has an effective directory to use
						if filepath.Dir(filePath) == "." {
							effectiveDir := a.aiPane.GetSelectedBlockDir()
							if effectiveDir != "" && effectiveDir != a.config.WorkspaceDir {
								relDir, err := filepath.Rel(a.config.WorkspaceDir, effectiveDir)
								if err == nil && relDir != "." {
									filePath = filepath.Join(relDir, filePath)
								}
							}
						}

						err := a.fileManager.CreateFile(filePath, a.editorPane.GetContent())
						if err != nil {
							a.statusMessage = "Error creating file: " + err.Error()
						} else {
							err = a.editorPane.LoadFile(filePath)
							if err != nil {
								a.statusMessage = "File created but load failed: " + err.Error()
							} else {
								a.statusMessage = "Saved as " + filePath
								a.aiPane.SetWorkingDir(filepath.Dir(a.editorPane.currentFile.Filepath))

								// Check if language runtime is installed for this file type
								fileType := a.editorPane.currentFile.FileType
								if fileType == "go" {
									a.ensureGoModule(filepath.Dir(a.editorPane.currentFile.Filepath))
								}
								if fileType == "python" {
									a.ensurePythonVenv(filepath.Dir(a.editorPane.currentFile.Filepath))
								}
								if fileType == "go" || fileType == "python" {
									return a, func() tea.Msg {
										langName := "Go"
										if fileType == "python" {
											langName = "Python"
										}
										return LanguageCheckMsg{
											FileType:     fileType,
											LanguageName: langName,
										}
									}
								}
							}
						}
					} else {
						// No suggestion — prompt for filename
						a.pendingCodeInsert = a.editorPane.GetContent()
						a.showFilePrompt = true
						a.filePromptBuffer = ""
						a.statusMessage = "Enter filename to save"
					}
				} else {
					err := a.editorPane.SaveFile()
					if err != nil {
						a.statusMessage = "Error saving: " + err.Error()
					} else {
						a.statusMessage = "File saved"

						// Check if language runtime is installed for this file type
						if a.editorPane.currentFile != nil {
							a.aiPane.SetWorkingDir(filepath.Dir(a.editorPane.currentFile.Filepath))
							fileType := a.editorPane.currentFile.FileType
							if fileType == "go" {
								a.ensureGoModule(filepath.Dir(a.editorPane.currentFile.Filepath))
							}
							if fileType == "python" {
								a.ensurePythonVenv(filepath.Dir(a.editorPane.currentFile.Filepath))
							}
							if fileType == "go" || fileType == "python" {
								return a, func() tea.Msg {
									langName := "Go"
									if fileType == "python" {
										langName = "Python"
									}
									return LanguageCheckMsg{
										FileType:     fileType,
										LanguageName: langName,
									}
								}
							}
						}
					}
				}
			}
			return a, nil

		case "ctrl+x":
			// Close file in editor
			if a.activePane == types.EditorPaneType {
				if a.editorPane.currentFile != nil {
					a.editorPane.CloseFile()
					a.statusMessage = "File closed"
				}
			}
			return a, nil

		case "ctrl+r":
			// Execute current script from editor
			if a.editorPane.currentFile == nil {
				a.statusMessage = "No file open to run"
				return a, nil
			}

			filePath := a.editorPane.currentFile.Filepath
			fileType := a.editorPane.currentFile.FileType

			// Auto-save before running
			if a.editorPane.HasUnsavedChanges() {
				if err := a.editorPane.SaveFile(); err != nil {
					a.statusMessage = "Save failed: " + err.Error()
					return a, nil
				}
			}

			// Build the execution command based on file type
			var runCmd string
			switch fileType {
			case "bash":
				runCmd = "bash " + filePath
			case "powershell":
				if runtime.GOOS == "windows" {
					runCmd = "powershell -NoProfile -File " + filePath
				} else {
					runCmd = "pwsh -NoProfile -File " + filePath
				}
			case "python":
				// Use venv Python if available
				runCmd = a.getPythonRunCommand(filePath)
			case "go":
				// For Go files, check if it's a test file or regular file
				if strings.HasSuffix(filePath, "_test.go") {
					// Run tests for test files
					runCmd = "go test -v " + filePath
				} else {
					// Run the Go file directly
					runCmd = "go run " + filePath
				}
			default:
				// Default: try to run as shell script
				if runtime.GOOS == "windows" {
					runCmd = "powershell -NoProfile -File " + filePath
				} else {
					runCmd = "sh " + filePath
				}
			}

			// Switch focus to AI pane and run
			a.activePane = types.AIPaneType
			a.editorPane.focused = false
			a.aiPane.focused = true

			fileName := filepath.Base(filePath)
			fileDir := filepath.Dir(filePath)
			a.statusMessage = "Running " + fileName + "..."

			cmd := a.aiPane.RunScript(runCmd, fileName, fileDir)
			cmds = append(cmds, cmd)
			return a, tea.Batch(cmds...)

		case "ctrl+enter":
			// Send AI message with context from editor
			if a.activePane == types.AIPaneType {
				// Use handleAIMessage which will get context automatically
				if a.aiPane.inputBuffer != "" {
					message := a.aiPane.inputBuffer
					a.aiPane.inputBuffer = ""
					cmd := a.handleAIMessage(message)
					cmds = append(cmds, cmd)
				}
			}
			return a, tea.Batch(cmds...)
		}
	}

	// Route messages to GitPane first if it's visible
	if a.gitPane.IsVisible() {
		var gitCmd tea.Cmd
		_, gitCmd = a.gitPane.Update(msg)
		if gitCmd != nil {
			cmds = append(cmds, gitCmd)
		}
		// When GitPane is visible, don't route to other panes
		return a, tea.Batch(cmds...)
	}

	// Route messages to active pane
	if a.activePane == types.EditorPaneType {
		cmd := a.editorPane.Update(msg)
		cmds = append(cmds, cmd)
	} else if a.activePane == types.AIPaneType || a.activePane == types.AIResponsePaneType {
		cmd := a.aiPane.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

// renderHeader renders the application header with logo.
// Displays "MINICLICODER" centered with binary code "01000011 01001100 01001001" (CLI) on the right.
// The header is wrapped in a blue rounded border.
//
// Returns:
//   - string: Rendered header with styling and border
//
// renderHeader renders the application header with logo.
// Displays binary "TI" on the left, "TERMINAL INTELLIGENCE (TI)" centered, and "Build: XXX" on the right.
// The header is wrapped in a blue rounded border.
//
// Layout:
//
//	Left: 01010100 01001001 (binary for "TI")
//	Center: TERMINAL INTELLIGENCE (TI)
//	Right: Build: XXX (git commit count)
func (a *App) renderHeader() string {
	// Binary code for "TI" on the left side
	// T = 84 (0x54) = 01010100
	// I = 73 (0x49) = 01001001
	binaryStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	binaryTI := binaryStyle.Render("01010100 01001001")

	// Create the TI title (centered)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	title := titleStyle.Render("TERMINAL INTELLIGENCE (TI)")

	// Create the build number (right side)
	buildText := fmt.Sprintf("Build: %03s", a.buildNumber)
	buildStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	build := buildStyle.Render(buildText)

	// Calculate available width for content (accounting for border and padding)
	contentWidth := a.width - 8
	binaryLen := len("01010100 01001001")
	titleLen := len("TERMINAL INTELLIGENCE (TI)")
	buildLen := len(buildText)

	// Calculate spacing
	// Left: binary + space
	// Center: title
	// Right: space + build
	totalContentLen := binaryLen + titleLen + buildLen
	availableSpace := contentWidth - totalContentLen

	// Distribute space: half before title, half after title
	leftSpace := availableSpace / 2
	rightSpace := availableSpace - leftSpace

	if leftSpace < 1 {
		leftSpace = 1
	}
	if rightSpace < 1 {
		rightSpace = 1
	}

	// Create spacing
	leftSpaceStr := lipgloss.NewStyle().Width(leftSpace).Render("")
	rightSpaceStr := lipgloss.NewStyle().Width(rightSpace).Render("")

	// Combine: binary + left space + title + right space + build
	headerContent := lipgloss.JoinHorizontal(lipgloss.Top, binaryTI, leftSpaceStr, title, rightSpaceStr, build)

	// Create blue border around header
	headerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(a.width - 4)

	return headerStyle.Render(headerContent)
}

// renderEditorTitleBar renders the full-width editor title bar.
// Displays "Editor: <filepath>" with an asterisk (*) if the file has unsaved changes.
// Shows "<no file>" if no file is currently open.
// The title bar has a blue background when a file is open.
//
// Returns:
//   - string: Rendered title bar with styling and border
func (a *App) renderEditorTitleBar() string {
	// Build title text
	title := "Editor: "
	if a.editorPane.currentFile != nil {
		title += a.editorPane.currentFile.Filepath
		if a.editorPane.HasUnsavedChanges() {
			title += " *"
		}
	} else {
		title += "<no file>"
	}

	// Create title bar style
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Width(a.width - 4)

	// Create border around title
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(a.width - 4)

	return borderStyle.Render(titleStyle.Render(title))
}

// View renders the application UI.
// This is part of the Bubble Tea Model interface.
//
// Rendering priority:
//  1. Initialization message (if not ready)
//  2. Help dialog (if showing)
//  3. File picker dialog (if showing)
//  4. File prompt dialog (if showing)
//  5. Exit confirmation dialog (if showing)
//  6. Main UI (header + editor title bar + split panes + status bar)
//
// Main UI layout:
//   - Header: Application logo and title
//   - Editor title bar: Current file name and modified indicator
//   - Split panes: Editor (left) and AI chat (right) side by side
//   - Status bar: Keyboard shortcuts and status messages
//
// Returns:
//   - string: Rendered UI as a string

// truncateToWidth ensures no line in the given string exceeds the specified width.
// Uses ANSI-aware truncation to correctly handle styled terminal output.
func truncateToWidth(content string, maxWidth int) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) > maxWidth {
			lines[i] = ansi.Truncate(line, maxWidth, "")
		}
	}
	return strings.Join(lines, "\n")
}

// enforceWidth ensures every line in content is exactly maxWidth visual columns wide.
// Lines shorter than maxWidth are padded with spaces; longer lines are truncated.
// This makes panels truly independent — JoinHorizontal will produce exact results.
func enforceWidth(content string, maxWidth int) string {
	if maxWidth <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w > maxWidth {
			lines[i] = ansi.Truncate(line, maxWidth, "")
		} else if w < maxWidth {
			lines[i] = line + strings.Repeat(" ", maxWidth-w)
		}
	}
	return strings.Join(lines, "\n")
}

func (a *App) View() string {
	if !a.ready {
		return "Initializing..."
	}

	// Show help dialog if needed
	if a.showHelp {
		return a.renderHelpDialog()
	}

	// Show chat loader dialog if needed
	if a.showChatLoader {
		pickerStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Width(80).
			Align(lipgloss.Left)

		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Bold(true)

		normalStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

		// Get the current project folder name
		projectFolder := filepath.Base(a.config.WorkspaceDir)

		var listDisplay string
		listDisplay = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Render("Select a chat to load:") + "\n"

		// Display project folder name
		listDisplay += lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Render("Project: "+projectFolder) + "\n\n"

		maxDisplay := 15
		startIdx := a.filePickerIndex - maxDisplay/2
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := startIdx + maxDisplay
		if endIdx > len(a.chatList) {
			endIdx = len(a.chatList)
			startIdx = endIdx - maxDisplay
			if startIdx < 0 {
				startIdx = 0
			}
		}

		for i := startIdx; i < endIdx; i++ {
			displayName := a.chatList[i]
			if i == a.filePickerIndex {
				listDisplay += selectedStyle.Render("> "+displayName) + "\n"
			} else {
				listDisplay += normalStyle.Render("  "+displayName) + "\n"
			}
		}

		listDisplay += "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Render("[↑↓] Navigate | [Enter] Load | [Esc] Cancel")

		dialog := pickerStyle.Render(listDisplay)
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	// Show folder picker dialog if needed
	if a.showFolderPicker {
		pickerStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Width(80).
			Align(lipgloss.Left)

		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Bold(true)

		normalStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

		var listDisplay string
		listDisplay = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Render("Choose a directory for your workspace:") + "\n"

		listDisplay += lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Render("Current: "+a.folderPickerPath) + "\n\n"

		maxDisplay := 15
		startIdx := a.folderPickerIndex - maxDisplay/2
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := startIdx + maxDisplay
		if endIdx > len(a.folderList) {
			endIdx = len(a.folderList)
			startIdx = endIdx - maxDisplay
			if startIdx < 0 {
				startIdx = 0
			}
		}

		for i := startIdx; i < endIdx; i++ {
			displayName := a.folderList[i]
			if i == a.folderPickerIndex {
				listDisplay += selectedStyle.Render("> "+displayName) + "\n"
			} else {
				listDisplay += normalStyle.Render("  "+displayName) + "\n"
			}
		}

		listDisplay += "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Render("[↑↓] Navigate | [Enter] Open/Select | [Esc] Cancel")

		dialog := pickerStyle.Render(listDisplay)
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	// Show backup picker dialog if needed
	if a.showBackupPicker {
		pickerStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Width(80).
			Align(lipgloss.Left)

		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Bold(true)

		normalStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

		// Get the current project folder name
		projectFolder := filepath.Base(a.config.WorkspaceDir)

		var listDisplay string
		listDisplay = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Render("Select a backup to restore (creates new backup of current):") + "\n"

		// Display project folder name
		listDisplay += lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Render("Project: "+projectFolder) + "\n\n"

		maxDisplay := 15
		startIdx := a.filePickerIndex - maxDisplay/2
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := startIdx + maxDisplay
		if endIdx > len(a.backupList) {
			endIdx = len(a.backupList)
			startIdx = endIdx - maxDisplay
			if startIdx < 0 {
				startIdx = 0
			}
		}

		for i := startIdx; i < endIdx; i++ {
			displayName := a.backupList[i]
			// Maybe shorten name for display if it has long path?
			// But user needs to distinguish versions. Timestamp is first, so it's good.
			if i == a.filePickerIndex {
				listDisplay += selectedStyle.Render("> "+displayName) + "\n"
			} else {
				listDisplay += normalStyle.Render("  "+displayName) + "\n"
			}
		}

		listDisplay += "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Render("[↑↓] Navigate | [Enter] Restore | [Esc] Cancel")

		dialog := pickerStyle.Render(listDisplay)
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	// Show file picker dialog if needed
	if a.showFilePicker {
		pickerStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Width(80).
			Align(lipgloss.Left)

		// Styles for file list
		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Bold(true)

		normalStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

		// Build file list display
		var fileListDisplay string
		fileListDisplay = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Render("Select a file to open or folder to browse:") + "\n"

		// Display current folder path
		fileListDisplay += lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Render("Current Path: "+a.filePickerPath) + "\n\n"

		maxDisplay := 15 // Maximum files to display at once
		startIdx := a.filePickerIndex - maxDisplay/2
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := startIdx + maxDisplay
		if endIdx > len(a.fileList) {
			endIdx = len(a.fileList)
			startIdx = endIdx - maxDisplay
			if startIdx < 0 {
				startIdx = 0
			}
		}

		for i := startIdx; i < endIdx; i++ {
			if i == a.filePickerIndex {
				fileListDisplay += selectedStyle.Render("> "+a.fileList[i]) + "\n"
			} else {
				fileListDisplay += normalStyle.Render("  "+a.fileList[i]) + "\n"
			}
		}

		fileListDisplay += "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Render("[↑↓] Navigate | [Enter] Open | [Esc] Cancel")

		dialog := pickerStyle.Render(fileListDisplay)

		// Center the dialog
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	// Show folder creation prompt dialog if needed
	if a.showFolderCreatePrompt {
		promptStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Width(60).
			Align(lipgloss.Center)

		// Get the current folder path
		currentPath := a.folderPickerPath
		if len(currentPath) > 40 {
			currentPath = "..." + currentPath[len(currentPath)-37:]
		}

		promptText := "Enter name for new folder:\n"
		promptText += lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Render("Location: "+currentPath) + "\n\n"
		promptText += a.folderCreateBuffer + "█\n\n[Enter] to create, [Esc] to cancel"

		dialog := promptStyle.Render(promptText)

		// Center the dialog
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	// Show file prompt dialog if needed
	if a.showFilePrompt {
		promptStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Width(60).
			Align(lipgloss.Center)

		// Get the current project folder name
		projectFolder := filepath.Base(a.config.WorkspaceDir)

		promptText := "Enter filename to create/open:\n"
		promptText += lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Render("Project: "+projectFolder) + "\n\n"
		promptText += a.filePromptBuffer + "█\n\n[Enter] to confirm, [Esc] to cancel"

		dialog := promptStyle.Render(promptText)

		// Center the dialog
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	// Show exit confirmation dialog if needed
	if a.showExitConfirmation {
		confirmStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(1, 2).
			Width(60).
			Align(lipgloss.Center)

		confirmText := "You have unsaved changes.\nAre you sure you want to quit without saving?\n\n[Y]es / [N]o"

		dialog := confirmStyle.Render(confirmText)

		// Center the dialog
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	// Show language install prompt dialog if needed
	if a.showLanguageInstallPrompt {
		promptStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")).
			Padding(1, 2).
			Width(70).
			Align(lipgloss.Center)

		var promptText string
		if a.languageToInstall == "Go" {
			langInstaller := installer.NewLanguageInstaller()
			installCmd, cmdText, _ := langInstaller.GetGoInstallCommand()

			if installCmd == "direct" {
				// Linux - direct download and install
				promptText = fmt.Sprintf("⚠  %s is not installed or not in PATH\n\n", a.languageToInstall)
				promptText += "Would you like to install Go automatically?\n\n"
				promptText += "Installation will:\n"
				promptText += "• Download latest Go from golang.org\n"
				promptText += "• Extract to /usr/local/go\n"
				promptText += "• Update your shell configuration\n"
				promptText += "• Requires sudo password\n\n"
				promptText += "[Y]es to install / [N]o to cancel"
			} else if installCmd == "manual" {
				promptText = fmt.Sprintf("⚠  %s is not installed or not in PATH\n\n", a.languageToInstall)
				promptText += "Manual installation required:\n"
				promptText += cmdText + "\n\n"
				promptText += "After installation, restart Terminal Intelligence.\n\n"
				promptText += "[N]o / [Esc] to cancel"
			} else {
				promptText = fmt.Sprintf("⚠  %s is not installed or not in PATH\n\n", a.languageToInstall)
				promptText += fmt.Sprintf("Would you like to install %s automatically?\n\n", a.languageToInstall)
				promptText += fmt.Sprintf("Installation command: %s\n\n", cmdText)
				promptText += "[Y]es to install / [N]o to cancel"
			}
		} else if a.languageToInstall == "Python" {
			langInstaller := installer.NewLanguageInstaller()
			installCmd, cmdText, _ := langInstaller.GetPythonInstallCommand()

			if installCmd == "manual" {
				promptText = fmt.Sprintf("⚠  %s is not installed or not in PATH\n\n", a.languageToInstall)
				promptText += "Manual installation required:\n"
				promptText += cmdText + "\n\n"
				promptText += "After installation, restart Terminal Intelligence.\n\n"
				promptText += "[N]o / [Esc] to cancel"
			} else {
				promptText = fmt.Sprintf("⚠  %s is not installed or not in PATH\n\n", a.languageToInstall)
				promptText += fmt.Sprintf("Would you like to install %s automatically?\n\n", a.languageToInstall)
				promptText += fmt.Sprintf("Installation command: %s\n\n", cmdText)
				promptText += "[Y]es to install / [N]o to cancel"
			}
		} else {
			promptText = fmt.Sprintf("⚠  %s is not installed or not in PATH\n\n", a.languageToInstall)
			promptText += fmt.Sprintf("Please install %s manually to run .%s files.\n\n", a.languageToInstall, a.fileTypeForInstall)
			promptText += "[N]o / [Esc] to close"
		}

		dialog := promptStyle.Render(promptText)

		// Center the dialog
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	// Render panes - each pane handles its own width internally
	editorContent := a.editorPane.View()
	aiContent := a.aiPane.View()

	// Enforce strict per-panel width by truncating each line to its allocated width.
	// This prevents either panel from overflowing into the other when joined.
	// Editor renders at editorPane.width - 2 (border takes 2), AI renders at aiPane.width.
	editorContent = enforceWidth(editorContent, a.editorPane.width-2)
	aiContent = enforceWidth(aiContent, a.aiPane.width)

	// Create status bar
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	// Simplified status bar - details available via Ctrl+H help
	statusText := "Ctrl+H: Help | Ctrl+W: Workspace | Ctrl+O: Open | Ctrl+S: Save | Tab: Cycle Areas | Ctrl+Q: Quit"

	// Add AI-specific instructions when AI pane is focused
	if a.activePane == types.AIPaneType || a.activePane == types.AIResponsePaneType {
		if a.aiPane.IsInViewMode() {
			// In view mode, show insert instruction
			statusText += " | Ctrl+P: Insert Code | Esc: Back"
		}
	}

	if a.statusMessage != "" {
		statusText += " | " + a.statusMessage
	}

	statusBar := statusStyle.Width(a.width - 2).Render(statusText)

	// Render header
	header := a.renderHeader()

	// Render full-width editor title bar
	editorTitleBar := a.renderEditorTitleBar()

	mainView := lipgloss.JoinHorizontal(lipgloss.Top, editorContent, aiContent)

	// Combine all sections vertically
	baseView := lipgloss.JoinVertical(lipgloss.Left, header, editorTitleBar, mainView, statusBar)

	// If GitPane is visible, render it as an overlay on top of the base view
	if a.gitPane.IsVisible() {
		gitPaneView := a.gitPane.View()
		// Center the GitPane popup on the screen
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, gitPaneView, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}))
	}

	return baseView
}

// GetEditorPane returns the editor pane for testing
func (a *App) GetEditorPane() *EditorPane {
	return a.editorPane
}

// GetAIPane returns the AI pane for testing
func (a *App) GetAIPane() *AIChatPane {
	return a.aiPane
}

// GetActivePane returns the currently active pane
func (a *App) GetActivePane() types.PaneType {
	return a.activePane
}

// SetActivePane sets the active pane (for testing)
func (a *App) SetActivePane(pane types.PaneType) {
	a.activePane = pane
	if pane == types.EditorPaneType {
		a.editorPane.focused = true
		a.aiPane.focused = false
	} else {
		a.editorPane.focused = false
		a.aiPane.focused = true
	}
}

// GetWidth returns the current width
func (a *App) GetWidth() int {
	return a.width
}

// GetHeight returns the current height
func (a *App) GetHeight() int {
	return a.height
}

// IsShowingExitConfirmation returns whether the exit confirmation dialog is showing
func (a *App) IsShowingExitConfirmation() bool {
	return a.showExitConfirmation
}

// SetForceQuit sets the force quit flag (for testing)
func (a *App) SetForceQuit(force bool) {
	a.forceQuit = force
}

// handleAIMessage processes an AI message through the AgenticCodeFixer
// It retrieves the current file context, processes the message, and applies fixes or returns conversational responses
// Requirements: 1.1, 4.1, 9.1, 9.2, 8.5
func (a *App) handleAIMessage(message string) tea.Cmd {
	// Check for special commands
	trimmedMsg := strings.TrimSpace(strings.ToLower(message))

	// Handle /config command
	if trimmedMsg == "/config" {
		// Load current config and enter config mode
		configPath, err := config.ConfigFilePath()
		if err != nil {
			return func() tea.Msg {
				return AIResponseMsg{
					Content: "Error: Unable to locate config file: " + err.Error(),
					Done:    true,
				}
			}
		}

		// Read current config
		jcfg, err := config.LoadFromFile(configPath)
		if err != nil {
			return func() tea.Msg {
				return AIResponseMsg{
					Content: "Error: Unable to load config file: " + err.Error(),
					Done:    true,
				}
			}
		}

		// Ensure all fields are populated with defaults if missing
		config.EnsureAllFields(jcfg)

		// Prepare config fields and values
		fields := []string{"agent", "model", "gmodel", "bedrock_model", "ollama_url", "gemini_api", "bedrock_api", "bedrock_region", "workspace", "autonomous"}
		values := []string{
			jcfg.Agent,
			jcfg.Model,
			jcfg.GModel,
			jcfg.BedrockModel,
			jcfg.OllamaURL,
			jcfg.GeminiAPI,
			jcfg.BedrockAPI,
			jcfg.BedrockRegion,
			jcfg.Workspace,
			jcfg.Autonomous,
		}

		// Enter config mode
		a.aiPane.EnterConfigMode(fields, values)

		return nil
	}

	// Handle /model command
	if trimmedMsg == "/model" {
		// Return current agent and model information
		modelInfo := fmt.Sprintf("Agent: %s\nModel: %s", a.config.Provider, a.config.DefaultModel)

		// If agent is Gemini, also show the API key
		if a.config.Provider == "gemini" && a.config.GeminiAPIKey != "" {
			modelInfo += fmt.Sprintf("\nAPI Key: %s", a.config.GeminiAPIKey)
		}

		return func() tea.Msg {
			return AIResponseMsg{
				Content: modelInfo,
				Done:    true,
			}
		}
	}

	// Handle /quit command
	if trimmedMsg == "/quit" {
		// Check for unsaved changes
		if a.editorPane.HasUnsavedChanges() && !a.forceQuit {
			a.showExitConfirmation = true
			return nil
		}
		// Clear AI history on normal exit
		a.aiPane.ClearHistory()
		return tea.Quit
	}

	// Handle /help command
	if trimmedMsg == "/help" {
		helpText := "Keyboard Shortcuts\n"
		helpText += "==================\n\n"
		helpText += "File\n"
		helpText += "----\n"
		helpText += "  Ctrl+W    Change Workspace / Open Folder\n"
		helpText += "  Ctrl+O    Open file\n"
		helpText += "  Ctrl+N    New file\n"
		helpText += "  Ctrl+S    Save file\n"
		helpText += "  Ctrl+X    Close file\n"
		helpText += "  Ctrl+R    Run current script\n"
		helpText += "  Ctrl+K    Kill running process (in terminal mode)\n"
		helpText += "  Ctrl+B    Backup Picker (Restore previous versions)\n"
		helpText += "  Ctrl+Q    Quit\n\n"
		helpText += "AI\n"
		helpText += "--\n"
		helpText += "  Ctrl+Y    List code blocks (Execute/Insert/Return)\n"
		helpText += "  Ctrl+P    Insert selected code into editor\n"
		helpText += "  Ctrl+L    Load saved chat from .ti/ folder\n"
		helpText += "  Ctrl+T    Clear chat / New chat\n\n"
		helpText += "Navigation\n"
		helpText += "----------\n"
		helpText += "  Tab       Switch between Editor, AI Input, and AI Response\n"
		helpText += "  Up/Down   Scroll line by line\n"
		helpText += "  PgUp/PgDn Scroll page\n"
		helpText += "  Home/End  Jump to top/bottom\n"
		helpText += "  Esc       Back\n\n"
		helpText += "Agent Commands\n"
		helpText += "--------------\n"
		helpText += "  /fix      Force agentic mode (AI modifies code)\n"
		helpText += "  /ask      Project-aware conversational mode\n"
		helpText += "  /doc      Generate project documentation\n"
		helpText += "  /preview  Preview changes without applying\n"
		helpText += "  /project  Run a project-wide change across all files\n"
		helpText += "  /proceed  Apply the last previewed change\n"
		helpText += "  /create   Autonomously build an app from scratch\n"
		helpText += "  /rescan   Rescan project files for fresh context\n"
		helpText += "  /model    Show current agent and model info\n"
		helpText += "  /config   Edit configuration settings\n"
		helpText += "  /help     Show this help message\n"
		helpText += "  /quit     Quit the program\n\n"
		helpText += "Smart Project Query\n"
		helpText += "-------------------\n"
		helpText += "  Ask project-level questions and get context-aware answers.\n"
		helpText += "  The AI auto-detects project questions (e.g. \"how do I build this?\").\n"
		helpText += "  Use /ask to force project context injection on any message.\n"
		helpText += "  Use /rescan after major project changes to refresh context.\n\n"
		helpText += "Fix Keywords\n"
		helpText += "------------\n"
		helpText += "  fix       Request code fix\n"
		helpText += "  change    Request code modification\n"
		helpText += "  update    Request code update\n"
		helpText += "  modify    Request code modification\n"
		helpText += "  correct   Request code correction\n\n"
		helpText += "Use these keywords in your message to trigger agentic mode."

		return func() tea.Msg {
			return AIResponseMsg{
				Content: helpText,
				Done:    true,
			}
		}
	}

	// Handle /project command (Req 9.2, 9.3, 1.4, 7.5)
	// Check for /project prefix (with optional /preview prefix before it)
	// BUT: Skip if this is a documentation generation command (/project /doc)
	trimmedForProject := strings.TrimSpace(strings.ToLower(message))
	isDocGenCmd := strings.Contains(trimmedForProject, "/doc")
	isProjectCmd := !isDocGenCmd && (strings.HasPrefix(trimmedForProject, "/project") ||
		strings.HasPrefix(trimmedForProject, "/preview /project") ||
		strings.HasPrefix(trimmedForProject, "/preview/project"))

	// Handle /cancel for AutonomousCreator
	if a.autonomousCreator != nil && a.autonomousCreator.State != agentic.StateDone && strings.TrimSpace(strings.ToLower(message)) == "/cancel" {
		a.autonomousCreator = nil
		return func() tea.Msg {
			return AINotificationMsg{Content: "Autonomous creation task aborted."}
		}
	}

	// Handle /proceed — re-run the last preview request without preview mode.
	if trimmedForProject == "/proceed" {
		if a.lastPreviewRequest == "" && (a.autonomousCreator == nil || a.autonomousCreator.State != agentic.StateWaitingApproval) {
			return func() tea.Msg {
				return AINotificationMsg{Content: "Nothing to proceed with."}
			}
		}

		// Check if we are proceeding with an AutonomousCreator plan
		if a.autonomousCreator != nil && a.autonomousCreator.State == agentic.StateWaitingApproval {
			a.autonomousCreator.State = agentic.StateSetup
			return func() tea.Msg {
				return AutonomousTickMsg{}
			}
		}

		// Re-issue as a real run using the stored request text (without /preview)
		// If it was a project-wide preview, re-run as project-wide
		message = a.lastPreviewRequest
		a.lastPreviewRequest = ""
		isProjectCmd = true
	}

	trimmedForCreate := strings.TrimSpace(strings.ToLower(message))
	isCreateCmd := strings.HasPrefix(trimmedForCreate, "/create")
	if isCreateCmd {
		if !a.config.Autonomous {
			return func() tea.Msg {
				return AINotificationMsg{Content: "Autonomous mode is currently disabled. Enable it via `/config` to use `/create`."}
			}
		}

		// Proceed with /create logic if not already running
		if a.autonomousCreator != nil && a.autonomousCreator.State != agentic.StateDone {
			return func() tea.Msg {
				return AINotificationMsg{Content: "An autonomous creation task is already in progress. Type /cancel to abort it first."}
			}
		}

		description := strings.TrimSpace(message[len("/create"):])
		if description == "" {
			return func() tea.Msg {
				return AINotificationMsg{Content: "Please provide a description for the application. Usage: `/create A simple text editor`"}
			}
		}

		// Add user message to chat history so it can be restored later
		a.aiPane.AddFixRequest(message, "", "")

		// Show immediate feedback that AI is working
		a.aiPane.DisplayNotification("🤖 AI is thinking and generating implementation plan...")

		createLogger := agentic.NewActionLogger(func(msg string) {
			a.aiPane.DisplayNotification(msg)
		})
		a.autonomousCreator = agentic.NewAutonomousCreator(
			a.aiClient, a.config.DefaultModel, a.config.WorkspaceDir, description,
			a.agenticProjectFixer, createLogger,
		)

		// Set callback to open SUMMARY.md in editor when it's created
		a.autonomousCreator.OpenFileCallback = func(filePath string) error {
			// Store the file path to be opened in the next tick
			a.autonomousFileToOpen = filePath
			return nil
		}

		// Return a command to tick the autonomous creator immediately to start planning
		return func() tea.Msg {
			return AutonomousTickMsg{}
		}
	}

	if isProjectCmd {
		// Req 9.3: Add user request to conversation history with filePath=projectRoot
		a.aiPane.AddFixRequest(message, a.aiPane.workspaceRoot, "")
		// Req 1.4: Show streaming indicator while operation is in progress
		a.aiPane.streaming = true

		projectRoot := a.aiPane.workspaceRoot
		msgCopy := message

		// Detect whether this is a preview run so we can store the request for /proceed.
		isPreview := strings.HasPrefix(trimmedForProject, "/preview")

		return func() tea.Msg {
			// statusUpdate is intentionally nil here: calling DisplayNotification from a
			// goroutine is not safe in Bubble Tea. The streaming indicator (set above)
			// already satisfies Req 1.4 by showing the pane is busy.
			report, err := a.projectFixer.ProcessProjectMessage(msgCopy, projectRoot, nil)
			if err != nil {
				return AINotificationMsg{Content: err.Error()}
			}

			formatted := agentic.FormatChangeReport(report)

			var bareRequest string
			// If this was a preview, remind the user they can /proceed.
			if isPreview {
				// Extract the bare request text (strip /preview and /project prefixes).
				bare := strings.TrimSpace(msgCopy)
				lower := strings.ToLower(bare)
				if strings.HasPrefix(lower, "/preview") {
					bare = strings.TrimSpace(bare[len("/preview"):])
					lower = strings.ToLower(bare)
				}
				if strings.HasPrefix(lower, "/project") {
					bare = strings.TrimSpace(bare[len("/project"):])
				}
				// Store the bare request without any command prefixes
				bareRequest = bare
				formatted += "\nType /proceed to apply these changes."
			}

			// Return ProjectCompleteMsg so the Update handler can open modified files.
			return ProjectCompleteMsg{Report: report, Formatted: formatted, LastPreviewRequest: bareRequest}
		}
	}

	// Step 1: Get file context from EditorPane using GetCurrentFile()
	fileContext := a.editorPane.GetCurrentFile()

	var fileContent, filePath, fileType string

	if fileContext != nil {
		fileContent = fileContext.FileContent
		filePath = fileContext.FilePath
		fileType = fileContext.FileType
	}

	// Handle /fix command — route to project-wide agentic fixer (Req 1.2, 1.3, 6.7)
	if strings.HasPrefix(trimmedMsg, "/fix") {
		fixMessage := strings.TrimSpace(message[len("/fix"):])
		if fixMessage == "" {
			return func() tea.Msg {
				return AINotificationMsg{Content: "Please provide a fix description. Usage: /fix <description of the issue>"}
			}
		}

		// Get open file path if available (Req 1.3: include as priority candidate)
		openFilePath := ""
		if fileContext != nil {
			openFilePath = fileContext.FilePath
		}

		request := &agentic.FixSessionRequest{
			Message:      fixMessage,
			ProjectRoot:  a.aiPane.workspaceRoot,
			OpenFilePath: openFilePath,
			MaxAttempts:  9,
			MaxCycles:    3,
		}

		a.aiPane.AddFixRequest(message, openFilePath, "")
		a.aiPane.streaming = true

		return func() tea.Msg {
			statusCallback := func(phase string) {
				a.aiPane.DisplayNotification(fmt.Sprintf("🔧 Fix phase: %s", phase))
			}
			result, err := a.agenticProjectFixer.ProcessFixCommand(request, statusCallback)
			return FixSessionCompleteMsg{Result: result, Error: err}
		}
	}

	// Handle /rescan command — invalidate cache and rebuild project context (Req 7.1, 7.2, 7.3)
	if trimmedMsg == "/rescan" {
		a.projectCtxCache.Invalidate(a.config.WorkspaceDir)
		builder := projectctx.NewContextBuilder()
		meta, err := builder.Build(a.config.WorkspaceDir)
		if err != nil {
			a.aiPane.DisplayNotification("⚠️ Rescan failed: " + err.Error())
			return nil
		}
		a.projectCtxCache.Put(a.config.WorkspaceDir, meta)
		notification := fmt.Sprintf("🔄 Project rescan complete: %d files discovered, %d key project files found.",
			meta.TotalFiles, len(meta.KeyFiles))
		a.aiPane.DisplayNotification(notification)
		return nil
	}

	// Step 2: Determine if this is a fix request upfront
	cleanMessage := message
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(message)), "/preview") {
		cleanMessage = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(message), "/preview"))
	}

	// Check if this is a search request
	isSearch := a.agenticFixer.IsSearchRequest(cleanMessage)
	if isSearch {
		a.aiPane.AddFixRequest(message, filePath, "")
		a.aiPane.streaming = true

		return func() tea.Msg {
			// Extract search terms using AI
			terms, err := a.agenticFixer.ExtractSearchTerms(cleanMessage)
			if err != nil || len(terms) == 0 {
				return AgenticFixResultMsg{
					Result: &agentic.FixResult{
						Success:          false,
						ErrorMessage:     "Could not extract search terms.",
						IsConversational: false,
					},
				}
			}

			exactResults, altResults, err := a.fileManager.SearchFilesContent(terms)
			if err != nil {
				return AgenticFixResultMsg{
					Result: &agentic.FixResult{
						Success:          false,
						ErrorMessage:     "Search failed: " + err.Error(),
						IsConversational: false,
					},
				}
			}

			if len(exactResults) == 0 && len(altResults) == 0 {
				return AgenticFixResultMsg{
					Result: &agentic.FixResult{
						Success:          false,
						ErrorMessage:     "No files found containing variations of: " + terms[0],
						IsConversational: false,
					},
				}
			}

			return SearchCompleteMsg{
				SearchTerm:   strings.Join(terms, ", "),
				ExactResults: exactResults,
				AltResults:   altResults,
			}
		}
	}

	// Project query classification — inject project context for project-level questions
	// (Req 3.1, 3.4, 3.5, 4.1, 4.2, 4.3, 4.4, 4.5, 5.1, 5.2, 5.3)
	classifier := projectctx.NewQueryClassifier()
	classification := classifier.Classify(cleanMessage)

	if classification.NeedsProjectContext {
		// Get or build project metadata from cache
		meta := a.projectCtxCache.Get(a.config.WorkspaceDir)
		if meta == nil {
			builder := projectctx.NewContextBuilder()
			var buildErr error
			meta, buildErr = builder.Build(a.config.WorkspaceDir)
			if buildErr != nil {
				// Fall through to existing conversational path on error
				a.aiPane.DisplayNotification("⚠️ Project context build failed: " + buildErr.Error())
				return a.aiPane.SendMessage(message, fileContent)
			}
			a.projectCtxCache.Put(a.config.WorkspaceDir, meta)
		}

		// Optional: search integration for search-like questions
		var searchResults []string
		if len(classification.SearchTerms) > 0 {
			exactResults, _, _ := a.fileManager.SearchFilesContent(classification.SearchTerms)
			searchResults = exactResults
		}

		// Build augmented prompt with project context
		promptBuilder := projectctx.NewPromptBuilder()
		augmentedPrompt := promptBuilder.Build(meta, message, searchResults, fileContent)

		// Send through existing streaming path — display user's original message,
		// but send the augmented prompt to the AI (Req 4.5: don't expose injected context)
		return a.aiPane.SendMessage(augmentedPrompt, "")
	}

	isFixDetection := a.agenticFixer.IsFixRequest(cleanMessage)

	// Step 3: Handle conversational mode immediately
	if !isFixDetection.IsFixRequest {
		return a.aiPane.SendMessage(message, fileContent)
	}

	// Step 4: Handle fix request
	// Display the message immediately as a fix request
	a.aiPane.AddFixRequest(message, filePath, fileContent)

	// Set AI pane to thinking state (streaming)
	a.aiPane.streaming = true

	// Step 5: Process the fix request asynchronously
	return func() tea.Msg {
		result, err := a.agenticFixer.ProcessMessage(
			message,
			fileContent,
			filePath,
			fileType,
		)

		if err != nil {
			// Create a fake failed result to carry the error
			return AgenticFixResultMsg{
				Result: &agentic.FixResult{
					Success:          false,
					ErrorMessage:     "Error processing message: " + err.Error(),
					IsConversational: false,
				},
			}
		}

		return AgenticFixResultMsg{Result: result}
	}
}

// openSearchResult opens the file at the current search result index and jumps to the matched term
func (a *App) openSearchResult() {
	if len(a.searchResults) == 0 || a.searchResultIndex < 0 || a.searchResultIndex >= len(a.searchResults) {
		return
	}

	res := a.searchResults[a.searchResultIndex]
	fullPath := filepath.Join(a.config.WorkspaceDir, res)

	err := a.editorPane.LoadFile(fullPath)
	if err != nil {
		a.statusMessage = fmt.Sprintf("Error opening file %d/%d: %s", a.searchResultIndex+1, len(a.searchResults), err.Error())
		return
	}

	a.statusMessage = fmt.Sprintf("Opened search match %d of %d: %s", a.searchResultIndex+1, len(a.searchResults), res)
	a.aiPane.SetWorkingDir(filepath.Dir(a.editorPane.currentFile.Filepath))

	// Search and jump to the matched term inside the editor
	if len(a.searchTerms) > 0 {
		a.editorPane.SearchAndJump(a.searchTerms)
	}

	// Switch to editor pane
	a.activePane = types.EditorPaneType
	a.editorPane.focused = true
	a.aiPane.focused = false
}

// ensureGoModule checks if a go.mod exists in the given directory and runs
// go mod init and go mod tidy if it doesn't, to automate Go project creation.
func (a *App) ensureGoModule(dir string) {
	go func() {
		cmdCheck := exec.Command("go", "env", "GOMOD")
		cmdCheck.Dir = dir
		out, err := cmdCheck.Output()
		outStr := strings.TrimSpace(string(out))
		if err != nil || outStr == "" || outStr == os.DevNull || outStr == "NUL" {
			baseName := filepath.Base(dir)
			if baseName == "." || baseName == "" || baseName == string(filepath.Separator) {
				baseName = "ti-project"
			}
			baseName = strings.ReplaceAll(baseName, " ", "-")
			baseName = strings.ToLower(baseName)
			initCmd := exec.Command("go", "mod", "init", baseName)
			initCmd.Dir = dir
			initCmd.Run()
		}

		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = dir
		tidyCmd.Run()
	}()
}

// ensurePythonVenv checks if a Python virtual environment exists in the given
// directory and creates one if it doesn't. It also installs dependencies from
// requirements.txt if present.
func (a *App) ensurePythonVenv(dir string) {
	go func() {
		venvDir := filepath.Join(dir, "venv")

		// Check if venv already exists
		if _, err := os.Stat(venvDir); err == nil {
			// venv exists — check if requirements.txt changed and re-install if needed
			reqFile := filepath.Join(dir, "requirements.txt")
			if _, err := os.Stat(reqFile); err == nil {
				pipCmd := a.getPipCommand(dir)
				installCmd := exec.Command(pipCmd, "install", "-r", reqFile)
				installCmd.Dir = dir
				installCmd.Run()
			}
			return
		}

		// Determine python command
		pythonCmd := getPythonCommand()
		if pythonCmd == "" {
			return // Python not installed
		}

		// Create venv
		createCmd := exec.Command(pythonCmd, "-m", "venv", "venv")
		createCmd.Dir = dir
		if err := createCmd.Run(); err != nil {
			return // Failed to create venv
		}

		// Install requirements.txt if it exists
		reqFile := filepath.Join(dir, "requirements.txt")
		if _, err := os.Stat(reqFile); err == nil {
			pipCmd := a.getPipCommand(dir)
			installCmd := exec.Command(pipCmd, "install", "-r", reqFile)
			installCmd.Dir = dir
			installCmd.Run()
		}
	}()
}

// getPythonCommand returns the Python command available on the system.
// Prefers python3 over python.
func getPythonCommand() string {
	if runtime.GOOS == "windows" {
		// On Windows, try python first (common), then python3
		if _, err := exec.LookPath("python"); err == nil {
			return "python"
		}
		if _, err := exec.LookPath("python3"); err == nil {
			return "python3"
		}
	} else {
		// On Unix, prefer python3
		if _, err := exec.LookPath("python3"); err == nil {
			return "python3"
		}
		if _, err := exec.LookPath("python"); err == nil {
			return "python"
		}
	}
	return ""
}

// getVenvPython returns the path to the Python interpreter inside a venv.
// Returns empty string if no venv exists in the directory.
func getVenvPython(dir string) string {
	var venvPython string
	if runtime.GOOS == "windows" {
		venvPython = filepath.Join(dir, "venv", "Scripts", "python.exe")
	} else {
		venvPython = filepath.Join(dir, "venv", "bin", "python")
	}
	if _, err := os.Stat(venvPython); err == nil {
		return venvPython
	}
	return ""
}

// getPipCommand returns the pip command to use, preferring the venv pip.
func (a *App) getPipCommand(dir string) string {
	var venvPip string
	if runtime.GOOS == "windows" {
		venvPip = filepath.Join(dir, "venv", "Scripts", "pip.exe")
	} else {
		venvPip = filepath.Join(dir, "venv", "bin", "pip")
	}
	if _, err := os.Stat(venvPip); err == nil {
		return venvPip
	}
	// Fallback to system pip
	if runtime.GOOS == "windows" {
		return "pip"
	}
	return "pip3"
}

// getPythonRunCommand returns the command to run a Python file, using the venv
// Python if available, otherwise falling back to the system Python.
func (a *App) getPythonRunCommand(filePath string) string {
	dir := filepath.Dir(filePath)
	venvPython := getVenvPython(dir)
	if venvPython != "" {
		return venvPython + " " + filePath
	}
	// Fallback to system python
	pythonCmd := getPythonCommand()
	if pythonCmd == "" {
		pythonCmd = "python3" // default fallback
	}
	return pythonCmd + " " + filePath
}

// loadChatHistory parses a saved chat file and loads it into the AI pane
// Format expected: "role timestamp\ncontent\n\n"
func (a *App) loadChatHistory(content string) error {
	// Clear existing chat history
	a.aiPane.ClearHistory()

	// Parse the chat file
	lines := strings.Split(content, "\n")
	var currentRole string
	var currentTimestamp time.Time
	var currentContent strings.Builder

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check if this is a role line (starts with "user ", "assistant ", or "notification ")
		if strings.HasPrefix(line, "user ") || strings.HasPrefix(line, "assistant ") || strings.HasPrefix(line, "notification ") {
			// Save previous message if exists
			if currentRole != "" && currentContent.Len() > 0 {
				msg := types.ChatMessage{
					Role:           currentRole,
					Content:        strings.TrimSpace(currentContent.String()),
					Timestamp:      currentTimestamp,
					IsNotification: currentRole == "notification",
				}
				a.aiPane.messages = append(a.aiPane.messages, msg)
			}

			// Parse new message header
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				currentRole = parts[0]
				// Parse timestamp (format: HH:MM:SS)
				timeStr := strings.TrimSpace(parts[1])
				parsedTime, err := time.Parse("15:04:05", timeStr)
				if err != nil {
					// If parsing fails, use current time
					currentTimestamp = time.Now()
				} else {
					// Use today's date with the parsed time
					now := time.Now()
					currentTimestamp = time.Date(now.Year(), now.Month(), now.Day(),
						parsedTime.Hour(), parsedTime.Minute(), parsedTime.Second(), 0, now.Location())
				}
				currentContent.Reset()
			}
		} else if line == "" && currentContent.Len() > 0 {
			// Empty line might indicate end of message, but continue accumulating
			// in case there are blank lines within the content
			currentContent.WriteString("\n")
		} else if currentRole != "" {
			// Accumulate content
			if currentContent.Len() > 0 {
				currentContent.WriteString("\n")
			}
			currentContent.WriteString(line)
		}
	}

	// Save last message if exists
	if currentRole != "" && currentContent.Len() > 0 {
		msg := types.ChatMessage{
			Role:           currentRole,
			Content:        strings.TrimSpace(currentContent.String()),
			Timestamp:      currentTimestamp,
			IsNotification: currentRole == "notification",
		}
		a.aiPane.messages = append(a.aiPane.messages, msg)
	}

	// Extract code blocks from loaded messages
	a.aiPane.extractCodeBlocks()

	// Scroll to bottom
	a.aiPane.scrollToBottom()

	// Restore autonomous creator state if the last message is waiting for approval
	a.restoreAutonomousState()

	return nil
}

// restoreAutonomousState checks if the loaded chat contains an autonomous creation
// plan waiting for approval and restores the AutonomousCreator state.
// restoreAutonomousState checks if the loaded chat contains an autonomous creation
// plan waiting for approval and restores the AutonomousCreator state.
func (a *App) restoreAutonomousState() {
	if len(a.aiPane.messages) == 0 {
		return
	}

	// Check the last assistant message
	lastMsg := a.aiPane.messages[len(a.aiPane.messages)-1]
	if lastMsg.Role != "assistant" {
		return
	}

	// Check if it contains the proceed prompt
	if !strings.Contains(lastMsg.Content, "Do you want to proceed? Type /proceed to continue or /cancel to abort.") {
		return
	}

	// Extract the plan from the message
	// Format: "ai-assist YYYY-MM-DD HH:MM:SS\nPlan generated:\n\n<plan>\n\nDo you want to proceed?"
	// or: "ai-assist HH:MM:SS\nPlan generated:\n\n<plan>\n\nDo you want to proceed?"
	// or just: "Plan generated:\n\n<plan>\n\nDo you want to proceed?"

	content := lastMsg.Content

	// Remove the "ai-assist ..." prefix if present (can be with date or just time)
	if strings.HasPrefix(content, "ai-assist ") {
		// Find the first newline after "ai-assist"
		newlineIdx := strings.Index(content, "\n")
		if newlineIdx != -1 {
			content = content[newlineIdx+1:]
		}
	}

	planStart := strings.Index(content, "Plan generated:")
	if planStart == -1 {
		return
	}

	planContent := content[planStart+len("Plan generated:"):]
	proceedPrompt := "\n\nDo you want to proceed? Type /proceed to continue or /cancel to abort."
	planEnd := strings.Index(planContent, proceedPrompt)
	if planEnd == -1 {
		return
	}

	plan := strings.TrimSpace(planContent[:planEnd])

	// Find the original /create command from user messages
	var description string
	for i := len(a.aiPane.messages) - 1; i >= 0; i-- {
		msg := a.aiPane.messages[i]
		if msg.Role == "user" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(msg.Content)), "/create") {
			description = strings.TrimSpace(msg.Content[len("/create"):])
			break
		}
	}

	if description == "" {
		// Can't restore without the original description
		return
	}

	// Extract project name from the plan
	projectName := extractProjectNameFromPlan(plan)
	if projectName == "" {
		projectName = "ti-autonomous-app"
	}

	// Reconstruct the AutonomousCreator in StateWaitingApproval
	a.autonomousCreator = &agentic.AutonomousCreator{
		AIClient:    a.aiClient,
		Model:       a.config.DefaultModel,
		Workspace:   a.config.WorkspaceDir,
		Description: description,
		Plan:        plan,
		ProjectName: projectName,
		ProjectDir:  filepath.Join(a.config.WorkspaceDir, projectName),
		State:       agentic.StateWaitingApproval,
	}
}

// extractProjectNameFromPlan attempts to extract the project name from the plan text.
// Looks for patterns like "project name: xyz" or "Project Name: xyz"
func extractProjectNameFromPlan(plan string) string {
	lines := strings.Split(plan, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "project name") {
			// Try to extract the name after the colon
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[1])
				// Remove common formatting characters
				name = strings.Trim(name, "` *-\"'")
				// Take only the first word if multiple
				words := strings.Fields(name)
				if len(words) > 0 {
					return words[0]
				}
			}
		}
	}
	return ""
}
