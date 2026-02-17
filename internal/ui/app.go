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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/terminal-intelligence/internal/agentic"
	"github.com/user/terminal-intelligence/internal/ai"
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
	config               *types.AppConfig           // Application configuration
	editorPane           *EditorPane                // Left pane: code editor
	aiPane               *AIChatPane                // Right pane: AI chat
	fileManager          *filemanager.FileManager   // File system operations
	aiClient             ai.AIClient                // AI service client (Ollama or Gemini)
	agenticFixer         *agentic.AgenticCodeFixer  // Autonomous code fixing orchestrator
	activePane           types.PaneType             // Currently focused pane
	width                int                        // Terminal width
	height               int                        // Terminal height
	ready                bool                       // Whether initial sizing is complete
	showExitConfirmation bool                       // Whether exit confirmation dialog is showing
	showFilePrompt       bool                       // Whether file creation prompt is showing
	showFilePicker       bool                       // Whether file picker dialog is showing
	showHelp             bool                       // Whether help dialog is showing
	filePromptBuffer     string                     // Buffer for file name input
	fileList             []string                   // List of files for picker
	filePickerIndex      int                        // Selected index in file picker
	forceQuit            bool                       // Whether to quit without save confirmation
	statusMessage        string                     // Status bar message
	pendingCodeInsert    string                     // Code waiting to be inserted after file creation
}

// New creates a new application instance with the provided configuration.
// If config is nil, uses default configuration.
//
// Initialization process:
//   1. Creates FileManager for file system operations
//   2. Creates AI client based on provider (Ollama or Gemini)
//   3. Initializes AgenticCodeFixer with AI client and model
//   4. Creates EditorPane and AIChatPane components
//   5. Sets initial state (editor pane active, no dialogs showing)
//
// The returned App is ready to be run with Bubble Tea's tea.NewProgram().
//
// Parameters:
//   - config: Application configuration (provider, model, API keys, workspace directory)
//
// Returns:
//   - *App: Initialized application instance
func New(config *types.AppConfig) *App {
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
//   1. Custom messages (InsertCodeMsg, SendAIMessageMsg, AINotificationMsg)
//   2. Window sizing (tea.WindowSizeMsg)
//   3. Keyboard input (tea.KeyMsg)
//   4. Routing to active pane
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
					err := a.editorPane.LoadFile(selectedFile)
					if err != nil {
						a.statusMessage = "Error opening: " + err.Error()
					} else {
						a.statusMessage = "Opened: " + selectedFile
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
		case "ctrl+c", "ctrl+q":
			// Check for unsaved changes
			if a.editorPane.HasUnsavedChanges() && !a.forceQuit {
				a.showExitConfirmation = true
				return a, nil
			}
			// Clear AI history on normal exit
			a.aiPane.ClearHistory()
			return a, tea.Quit

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
			// Switch active pane (toggle between Editor and AI)
			if a.activePane == types.EditorPaneType {
				a.activePane = types.AIPaneType
				a.editorPane.focused = false
				a.aiPane.focused = true
			} else {
				a.activePane = types.EditorPaneType
				a.editorPane.focused = true
				a.aiPane.focused = false
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
func (a *App) renderHeader() string {
	// Binary code for "CLI" on the right side
	binaryStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	binary := binaryStyle.Render("01000011 01001100 01001001")

	// Create the TI title (centered)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	title := titleStyle.Render("TERMINAL INTELLIGENCE (TI)")

	// Calculate available width for content (accounting for border and padding)
	contentWidth := a.width - 8
	binaryLen := len("01000011 01001100 01001001")
	titleLen := len("TERMINAL INTELLIGENCE (TI)")

	// Calculate left padding to center the title
	leftPadding := (contentWidth - titleLen) / 2

	// Calculate right padding (space between title and binary)
	rightPadding := contentWidth - leftPadding - titleLen - binaryLen
	if rightPadding < 1 {
		rightPadding = 1
	}

	// Create spacing
	leftSpace := lipgloss.NewStyle().Width(leftPadding).Render("")
	rightSpace := lipgloss.NewStyle().Width(rightPadding).Render("")

	// Combine: left padding + title + right padding + binary
	headerContent := lipgloss.JoinHorizontal(lipgloss.Top, leftSpace, title, rightSpace, binary)

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
//   1. Initialization message (if not ready)
//   2. Help dialog (if showing)
//   3. File picker dialog (if showing)
//   4. File prompt dialog (if showing)
//   5. Exit confirmation dialog (if showing)
//   6. Main UI (header + editor title bar + split panes + status bar)
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

	// Show file picker dialog if needed
	if a.showFilePicker {
		pickerStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Width(60).
			Align(lipgloss.Left)

		// Build file list display
		var fileListDisplay string
		fileListDisplay = "Select a file to open:\n\n"

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
				fileListDisplay += "> " + a.fileList[i] + "\n"
			} else {
				fileListDisplay += "  " + a.fileList[i] + "\n"
			}
		}

		fileListDisplay += "\n[↑↓] Navigate | [Enter] Open | [Esc] Cancel"

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
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	// Simplified status bar - details available via Ctrl+H help
	statusText := "Ctrl+H: Help | Ctrl+O: Open | Ctrl+S: Save | Tab: Switch Pane | Ctrl+Q: Quit"

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
	// Step 1: Get file context from EditorPane using GetCurrentFile()
	fileContext := a.editorPane.GetCurrentFile()
	
	var fileContent, filePath, fileType string
	
	if fileContext != nil {
		fileContent = fileContext.FileContent
		filePath = fileContext.FilePath
		fileType = fileContext.FileType
	}
	
	// Step 2: Process message through AgenticCodeFixer
	result, err := a.agenticFixer.ProcessMessage(
		message,
		fileContent,
		filePath,
		fileType,
	)
	
	// Handle errors from ProcessMessage
	if err != nil {
		return func() tea.Msg {
			return AIResponseMsg{
				Content: "Error processing message: " + err.Error(),
				Done:    true,
			}
		}
	}
	
	// Step 3: Handle the FixResult appropriately
	if result.IsConversational {
		// This is a conversational message, not a fix request
		// Use the existing SendMessage method to handle it normally
		return a.aiPane.SendMessage(message, fileContent)
	}
	
	// Step 4: Add fix request to conversation history
	// This is a fix request, so add it to history with file context
	a.aiPane.AddFixRequest(message, filePath)
	
	// Step 5: Handle fix results
	if result.Success {
		// In preview mode, don't apply the fix to the editor
		// Just show the changes summary
		if !result.PreviewMode {
			// Apply the fix to the editor
			a.editorPane.SetContent(result.ModifiedContent)
		}
		
		// Return notification message with changes summary
		// The notification will be added to history by DisplayNotification
		return func() tea.Msg {
			return AINotificationMsg{
				Content: result.ChangesSummary,
			}
		}
	}
	
	// Handle fix failure
	// Add error notification to history
	return func() tea.Msg {
		return AINotificationMsg{
			Content: result.ErrorMessage,
		}
	}
}
