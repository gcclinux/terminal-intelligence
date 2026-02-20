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
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/terminal-intelligence/internal/agentic"
	"github.com/user/terminal-intelligence/internal/ai"
	"github.com/user/terminal-intelligence/internal/config"
	"github.com/user/terminal-intelligence/internal/filemanager"
	"github.com/user/terminal-intelligence/internal/gemini"
	"github.com/user/terminal-intelligence/internal/ollama"
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
	config               *types.AppConfig          // Application configuration
	editorPane           *EditorPane               // Left pane: code editor
	aiPane               *AIChatPane               // Right pane: AI chat
	fileManager          *filemanager.FileManager  // File system operations
	aiClient             ai.AIClient               // AI service client (Ollama or Gemini)
	agenticFixer         *agentic.AgenticCodeFixer // Autonomous code fixing orchestrator
	activePane           types.PaneType            // Currently focused pane
	width                int                       // Terminal width
	height               int                       // Terminal height
	ready                bool                      // Whether initial sizing is complete
	showExitConfirmation bool                      // Whether exit confirmation dialog is showing
	showFilePrompt       bool                      // Whether file creation prompt is showing
	showFilePicker       bool                      // Whether file picker dialog is showing
	showBackupPicker     bool                      // Whether backup picker dialog is showing
	showHelp             bool                      // Whether help dialog is showing
	filePromptBuffer     string                    // Buffer for file name input
	fileList             []string                  // List of files for picker
	backupList           []string                  // List of backups for picker
	filePickerIndex      int                       // Selected index in file picker
	forceQuit            bool                      // Whether to quit without save confirmation
	statusMessage        string                    // Status bar message
	pendingCodeInsert    string                    // Code waiting to be inserted after file creation
	buildNumber          string                    // Build number from git commits
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
	} else {
		aiClient = ollama.NewOllamaClient(config.OllamaURL)
	}

	// Initialize AgenticCodeFixer
	agenticFixer := agentic.NewAgenticCodeFixer(aiClient, config.DefaultModel)

	return &App{
		config:               config,
		fileManager:          fm,
		aiClient:             aiClient,
		agenticFixer:         agenticFixer,
		editorPane:           NewEditorPane(fm),
		aiPane:               NewAIChatPane(aiClient, config.DefaultModel, config.Provider),
		activePane:           types.EditorPaneType,
		ready:                false,
		showExitConfirmation: false,
		forceQuit:            false,
		buildNumber:          buildNumber,
	}
}

// Init initializes the application and returns initial command.
// This is part of the Bubble Tea Model interface.
// Currently returns nil as no initial commands are needed.
func (a *App) Init() tea.Cmd {
	return nil
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
//   - Ctrl+A: Insert full AI response into editor
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
			case "ollama_url":
				jcfg.OllamaURL = msg.Values[i]
			case "gemini_api":
				jcfg.GeminiAPI = msg.Values[i]
			case "workspace":
				jcfg.Workspace = msg.Values[i]
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
		} else {
			a.aiClient = ollama.NewOllamaClient(a.config.OllamaURL)
		}

		// Update AI pane with new client and model
		a.aiPane.aiClient = a.aiClient
		a.aiPane.model = a.config.DefaultModel
		a.aiPane.provider = a.config.Provider

		// Update agentic fixer
		a.agenticFixer = agentic.NewAgenticCodeFixer(a.aiClient, a.config.DefaultModel)

		a.statusMessage = "Configuration saved successfully to " + configPath
		return a, nil

	case InsertCodeMsg:
		// Handle code insertion from AI pane
		selectedCode := a.aiPane.GetSelectedCodeBlock()
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
				// No file open, prompt for new file
				a.pendingCodeInsert = selectedCode
				a.showFilePrompt = true
				a.filePromptBuffer = ""
				a.statusMessage = "Enter filename to insert code"
			}
		} else {
			a.statusMessage = "No code block selected"
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

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true

		// Account for header (3 lines), editor title bar (3 lines), status bar (1 line)
		// Give editor pane slightly more width to account for borders
		editorWidth := (msg.Width / 2) + 4 // Add 2 extra chars to editor
		aiWidth := (msg.Width / 2) - 1     // Subtract 2 from AI pane
		paneHeight := msg.Height - 7       // -3 for header, -3 for editor title bar, -1 for status bar

		// Update editor pane size
		a.editorPane.width = editorWidth
		a.editorPane.height = paneHeight

		// Update AI pane size
		a.aiPane.width = aiWidth
		a.aiPane.height = paneHeight

		return a, nil

	case tea.KeyMsg:
		// Handle help dialog
		if a.showHelp {
			switch msg.String() {
			case "esc", "ctrl+h", "q":
				a.showHelp = false
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
				// Open selected file
				if len(a.fileList) > 0 && a.filePickerIndex < len(a.fileList) {
					selectedFile := a.fileList[a.filePickerIndex]

					// Debug: Log the file being opened
					a.statusMessage = "Opening: " + selectedFile

					err := a.editorPane.LoadFile(selectedFile)
					if err != nil {
						a.statusMessage = "Error opening file: " + err.Error()
					} else {
						a.statusMessage = "Successfully opened: " + selectedFile
						// Switch to editor pane after opening file
						a.activePane = types.EditorPaneType
						a.editorPane.focused = true
						a.aiPane.focused = false
					}
				}
				a.showFilePicker = false
				a.fileList = nil
				a.filePickerIndex = 0
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
			// Open file picker with list of existing files
			files, err := a.fileManager.ListFiles()
			if err != nil {
				a.statusMessage = "Error listing files: " + err.Error()
				return a, nil
			}
			if len(files) == 0 {
				a.statusMessage = "No files found in workspace"
				return a, nil
			}
			a.fileList = files
			a.filePickerIndex = 0
			a.showFilePicker = true
			a.statusMessage = fmt.Sprintf("Found %d files", len(files))
			return a, nil

		case "ctrl+n":
			// New file prompt
			a.showFilePrompt = true
			a.filePromptBuffer = ""
			return a, nil

		case "ctrl+t":
			// Clear AI chat history (New Chat)
			a.aiPane.ClearHistory()
			a.statusMessage = "AI chat history cleared"
			return a, nil

		case "ctrl+h":
			// Toggle help menu
			a.showHelp = !a.showHelp
			return a, nil

		case "ctrl+a":
			// Insert entire last assistant response into editor file
			response := a.aiPane.GetLastAssistantResponse()
			if response == "" {
				a.statusMessage = "No AI response to insert"
				return a, nil
			}
			if a.editorPane.currentFile != nil {
				// Append response to current file
				currentContent := a.editorPane.GetContent()
				if currentContent != "" {
					a.editorPane.SetContent(currentContent + "\n" + response)
				} else {
					a.editorPane.SetContent(response)
				}

				// Save the file automatically
				err := a.editorPane.SaveFile()
				if err != nil {
					a.statusMessage = "Response inserted but save failed: " + err.Error()
				} else {
					a.statusMessage = "Full response inserted and saved to " + a.editorPane.currentFile.Filepath
				}

				// Switch to editor pane to show the result
				a.activePane = types.EditorPaneType
				a.editorPane.focused = true
				a.aiPane.focused = false
			} else {
				// No file open, prompt for new file
				a.pendingCodeInsert = response
				a.showFilePrompt = true
				a.filePromptBuffer = ""
				a.statusMessage = "Enter filename to insert response"
			}
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
				err := a.editorPane.SaveFile()
				if err != nil {
					a.statusMessage = "Error saving: " + err.Error()
				} else {
					a.statusMessage = "File saved"
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
			// Execute script (placeholder for now)
			// TODO: Implement script execution
			a.statusMessage = "Script execution not yet implemented"
			return a, nil

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
func (a *App) View() string {
	if !a.ready {
		return "Initializing..."
	}

	// Show help dialog if needed
	if a.showHelp {
		return a.renderHelpDialog()
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

		var listDisplay string
		listDisplay = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Render("Select a backup to restore (creates new backup of current):") + "\n\n"

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
			Foreground(lipgloss.Color("240")).
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
			Width(60).
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
			Render("Select a file to open:") + "\n\n"

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
			Foreground(lipgloss.Color("240")).
			Render("[↑↓] Navigate | [Enter] Open | [Esc] Cancel")

		dialog := pickerStyle.Render(fileListDisplay)

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

		promptText := "Enter filename to create/open:\n\n" + a.filePromptBuffer + "█\n\n[Enter] to confirm, [Esc] to cancel"

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

	// Define styles for focused and unfocused panes (removed - panes handle their own borders)

	// Render panes without additional borders (they have their own)
	editorContent := a.editorPane.View()
	aiContent := a.aiPane.View()

	// Create status bar
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	// Simplified status bar - details available via Ctrl+H help
	statusText := "Ctrl+H: Help | Ctrl+O: Open | Ctrl+S: Save | Tab: Cycle Areas | Ctrl+Q: Quit"

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

	// Join panes side by side
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, editorContent, aiContent)

	// Combine all sections vertically
	return lipgloss.JoinVertical(lipgloss.Left, header, editorTitleBar, mainView, statusBar)
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

		// Prepare config fields and values
		fields := []string{"agent", "model", "gmodel", "ollama_url", "gemini_api", "workspace"}
		values := []string{
			jcfg.Agent,
			jcfg.Model,
			jcfg.GModel,
			jcfg.OllamaURL,
			jcfg.GeminiAPI,
			jcfg.Workspace,
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

	// Handle /help command
	if trimmedMsg == "/help" {
		helpText := "Keyboard Shortcuts\n"
		helpText += "==================\n\n"
		helpText += "File\n"
		helpText += "----\n"
		helpText += "  Ctrl+O    Open file\n"
		helpText += "  Ctrl+N    New file\n"
		helpText += "  Ctrl+S    Save file\n"
		helpText += "  Ctrl+X    Close file\n"
		helpText += "  Ctrl+B    Backup Picker (Restore previous versions)\n"
		helpText += "  Ctrl+Q    Quit\n\n"
		helpText += "AI\n"
		helpText += "--\n"
		helpText += "  Ctrl+Y    List code blocks\n"
		helpText += "  Ctrl+P    Insert selected code into editor\n"
		helpText += "  Ctrl+A    Insert full AI response into file\n"
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
		helpText += "  /ask      Force conversational mode (no code changes)\n"
		helpText += "  /preview  Preview changes before applying\n"
		helpText += "  /model    Show current agent and model info\n"
		helpText += "  /config   Edit configuration settings\n"
		helpText += "  /help     Show this help message\n\n"
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

	// Step 1: Get file context from EditorPane using GetCurrentFile()
	fileContext := a.editorPane.GetCurrentFile()

	var fileContent, filePath, fileType string

	if fileContext != nil {
		fileContent = fileContext.FileContent
		filePath = fileContext.FilePath
		fileType = fileContext.FileType
	}

	// Step 2: Determine if this is a fix request upfront
	cleanMessage := message
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(message)), "/preview") {
		cleanMessage = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(message), "/preview"))
	}

	isFixDetection := a.agenticFixer.IsFixRequest(cleanMessage)

	// Step 3: Handle conversational mode immediately
	if !isFixDetection.IsFixRequest {
		return a.aiPane.SendMessage(message, fileContent)
	}

	// Step 4: Handle fix request
	// Display the message immediately as a fix request
	a.aiPane.AddFixRequest(message, filePath)

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
