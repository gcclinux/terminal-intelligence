package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/terminal-intelligence/internal/ai"
	"github.com/user/terminal-intelligence/internal/types"
)

// AIChatPane manages the AI interaction pane with conversation history.
// This component provides a chat interface for interacting with AI models (Ollama or Gemini).
//
// Features:
//   - Streaming AI responses
//   - Conversation history with timestamps
//   - Code block extraction and viewing
//   - Code insertion into editor
//   - Scrolling for long conversations
//   - Context indicators (when file context is included)
//   - Distinct formatting for notifications (fix results)
//
// Modes:
//   - Normal mode: Type messages and view responses
//   - Copy mode (Ctrl+Y): Select and view code blocks
//   - View mode: View full code block with scrolling
//
// The pane has two areas:
//   - Input area (top): Type AI messages
//   - Response area (bottom): View conversation history
//
// Integration with AgenticCodeFixer:
//   - AddFixRequest: Adds fix requests to history with file context
//   - DisplayNotification: Shows fix results with distinct styling
//   - GetLastAssistantResponse: Retrieves last AI response for insertion
type AIChatPane struct {
	messages         []types.ChatMessage // Conversation history
	inputBuffer      string              // Current input being typed
	aiClient         ai.AIClient         // AI service client
	model            string              // AI model to use
	provider         string              // "ollama" or "gemini"
	scrollOffset     int                 // Vertical scroll offset for responses
	width            int                 // Pane width
	height           int                 // Pane height
	focused          bool                // Whether this pane is focused
	streaming        bool                // Whether AI is currently generating
	copyMode         bool                // Whether in code block selection mode
	viewMode         bool                // Whether viewing a code block
	codeBlocks       []string            // Extracted code blocks from responses
	selectedBlock    int                 // Currently selected code block index
	lastSelectedCode string              // Last selected code block content
	lastKeyPressed   string              // Last key pressed (for debugging)
	activeArea       int                 // 0: Input, 1: Response
	viewModeScroll   int                 // Scroll offset for code block view mode
	viewModeScrollX  int                 // Horizontal scroll offset for code block view mode
	configMode       bool                // Whether in config editor mode
	configFields     []string            // Config field names
	configValues     []string            // Config field values
	selectedField    int                 // Currently selected config field
	editingField     bool                // Whether currently editing a field
	editBuffer       string              // Buffer for editing field value
	editCursorPos    int                 // Cursor position within edit buffer
	terminalMode     bool                // Whether terminal execution is active
	cmdRunning       bool                // Whether the command is currently running
	terminalOutput   []string            // Output lines from terminal execution
	stdinWriter      io.WriteCloser      // Stdin pipe for sending input to running command
	terminalInput    string              // Current input line being typed in terminal mode
	aiAvailable      bool                // Whether the AI service is reachable
	aiChecked        bool                // Whether the availability check has completed
	suggestedFile    string              // Filename suggested by AI for the current code block
	workingDir       string              // Directory from which to execute commands
}

// AIResponseMsg is sent when AI response chunk is received.
// Used for streaming AI responses from the AI client.
type AIResponseMsg struct {
	Content string // Response content
	Done    bool   // Whether generation is complete
}

// InsertCodeMsg is sent when user wants to insert code into editor.
// Triggered by Ctrl+P in view mode.
type InsertCodeMsg struct{}

// SendAIMessageMsg is sent when user wants to send a message to AI.
// Triggered by Enter in input area.
type SendAIMessageMsg struct {
	Message string // The message to send
}

// AINotificationMsg is sent when a change notification should be displayed.
// Used by AgenticCodeFixer to show fix results.
type AINotificationMsg struct {
	Content string // Notification content
}

// SaveConfigMsg is sent when user wants to save config changes.
// Triggered by Esc in config mode.
type SaveConfigMsg struct {
	Fields []string // Config field names
	Values []string // Config field values
}

// TerminalOutputMsg is sent when a line of output is received from a running command.
type TerminalOutputMsg struct {
	Line   string
	Output chan tea.Msg
}

// TerminalDoneMsg is sent when command execution finishes.
type TerminalDoneMsg struct {
	ExitCode int
	Err      error
}

// AIAvailabilityMsg is sent when the AI availability check completes.
type AIAvailabilityMsg struct {
	Available bool
}

// LanguageCheckMsg is sent when a language runtime check is needed.
type LanguageCheckMsg struct {
	FileType     string // The file type being checked (e.g., "go", "python")
	LanguageName string // Human-readable language name
}

// LanguageInstallPromptMsg is sent to show the installation prompt dialog.
type LanguageInstallPromptMsg struct {
	LanguageName string // Language to install (e.g., "Go", "Python")
	FileType     string // File type that triggered the check
}

// LanguageInstallMsg is sent to start the installation process.
type LanguageInstallMsg struct {
	LanguageName string // Language to install
}

// LanguageInstallProgressMsg is sent during installation to show progress.
type LanguageInstallProgressMsg struct {
	Message string // Progress message to display
}

// LanguageInstallResultMsg is sent when installation completes.
type LanguageInstallResultMsg struct {
	Success bool
	Output  string
	Error   error
}

// NewAIChatPane creates a new AI chat pane.
// Initializes an empty conversation with the specified AI client and model.
// Defaults to "llama2" model if none is specified.
//
// Parameters:
//   - client: AI service client (Ollama or Gemini)
//   - model: AI model identifier
//   - provider: Provider name ("ollama" or "gemini")
//
// Returns:
//   - *AIChatPane: Initialized AI chat pane
func NewAIChatPane(client ai.AIClient, model string, provider string) *AIChatPane {
	if model == "" {
		model = "llama2"
	}
	cwd, _ := os.Getwd()
	return &AIChatPane{
		messages:       []types.ChatMessage{},
		inputBuffer:    "",
		aiClient:       client,
		model:          model,
		provider:       provider,
		scrollOffset:   0,
		width:          0,
		height:         0,
		focused:        false,
		streaming:      false,
		activeArea:     0, // 0: Input, 1: Response
		terminalOutput: []string{},
		workingDir:     cwd,
	}
}

// SetWorkingDir sets the directory from which commands should execute.
func (a *AIChatPane) SetWorkingDir(dir string) {
	if dir != "" {
		a.workingDir = dir
	}
}

// CheckAIAvailability returns a tea.Cmd that checks if the configured AI
// provider and model are reachable, sending an AIAvailabilityMsg with the result.
func (a *AIChatPane) CheckAIAvailability() tea.Cmd {
	client := a.aiClient
	model := a.model
	provider := a.provider

	return func() tea.Msg {
		if client == nil {
			return AIAvailabilityMsg{Available: false}
		}

		// Check if the service is reachable
		available, err := client.IsAvailable()
		if err != nil || !available {
			return AIAvailabilityMsg{Available: false}
		}

		// For Ollama, also verify the specific model exists
		if provider == "ollama" {
			models, err := client.ListModels()
			if err != nil {
				return AIAvailabilityMsg{Available: false}
			}
			found := false
			for _, m := range models {
				if m == model || m == model+":latest" {
					found = true
					break
				}
			}
			if !found {
				return AIAvailabilityMsg{Available: false}
			}
		}

		// For Gemini, try listing models to confirm API key + model access
		if provider == "gemini" {
			// IsAvailable already checks API key; do a lightweight generate test
			// by just confirming the client was created with a key.
			// The real validation happens on first request.
		}

		return AIAvailabilityMsg{Available: true}
	}
}

// SetActiveArea sets the active area (0: Input, 1: Response).
// Used to switch focus between input and response areas.
//
// Parameters:
//   - area: Area index (0 for input, 1 for response)
func (a *AIChatPane) SetActiveArea(area int) {
	a.activeArea = area
}

// GetActiveArea returns the current active area (0: Input, 1: Response).
func (a *AIChatPane) GetActiveArea() int {
	return a.activeArea
}

// SendMessage sends a message to the AI with optional code context.
// Adds the user message to history and initiates streaming AI generation.
// If context is provided, it's included in the prompt as a code block.
//
// Parameters:
//   - message: User's message
//   - context: Optional code context (empty string if none)
//
// Returns:
//   - tea.Cmd: Command that streams AI response
func (a *AIChatPane) SendMessage(message string, context string) tea.Cmd {
	// Add user message to history
	userMsg := types.ChatMessage{
		Role:            "user",
		Content:         message,
		Timestamp:       time.Now(),
		ContextIncluded: context != "",
	}
	a.messages = append(a.messages, userMsg)

	// Build prompt with context if provided
	prompt := message
	if context != "" {
		prompt = "Here is the current code:\n\n```\n" + context + "\n```\n\n" + message
	}

	a.streaming = true

	// Return command to start streaming
	return func() tea.Msg {
		responseChan, err := a.aiClient.Generate(prompt, a.model, nil)
		if err != nil {
			return AIResponseMsg{
				Content: "Error: " + err.Error(),
				Done:    true,
			}
		}

		// Collect streaming responses
		var fullResponse strings.Builder
		for chunk := range responseChan {
			fullResponse.WriteString(chunk)
		}

		// Return complete response as message
		return AIResponseMsg{
			Content: fullResponse.String(),
			Done:    true,
		}
	}
}

// DisplayResponse displays AI response in the chat pane.
// Adds the response to conversation history, extracts code blocks, and scrolls to bottom.
//
// Parameters:
//   - response: AI response content
func (a *AIChatPane) DisplayResponse(response string) {
	assistantMsg := types.ChatMessage{
		Role:            "assistant",
		Content:         response,
		Timestamp:       time.Now(),
		ContextIncluded: false,
	}
	a.messages = append(a.messages, assistantMsg)
	a.streaming = false

	// Extract code blocks from response
	a.extractCodeBlocks()

	// Auto-scroll to bottom
	a.scrollToBottom()
}

// DisplayNotification displays a change notification in the chat pane with distinct formatting.
// Used by AgenticCodeFixer to show fix results with cyan color and notification label.
// Adds the notification to conversation history and scrolls to bottom.
//
// Parameters:
//   - notification: Notification content (typically fix result summary)
func (a *AIChatPane) DisplayNotification(notification string) {
	notificationMsg := types.ChatMessage{
		Role:            "assistant",
		Content:         notification,
		Timestamp:       time.Now(),
		ContextIncluded: false,
		IsNotification:  true,
	}
	a.messages = append(a.messages, notificationMsg)
	a.streaming = false

	// Auto-scroll to bottom
	a.scrollToBottom()
}

// AddFixRequest adds a fix request to conversation history as a user message.
// Includes context indicators like file path to distinguish fix requests from regular messages.
// Used by AgenticCodeFixer when processing fix requests.
//
// Parameters:
//   - message: User's fix request message
//   - filePath: Path to the file being fixed
func (a *AIChatPane) AddFixRequest(message string, filePath string) {
	fixRequestMsg := types.ChatMessage{
		Role:            "user",
		Content:         message,
		Timestamp:       time.Now(),
		ContextIncluded: true,
		IsFixRequest:    true,
		FilePath:        filePath,
	}
	a.messages = append(a.messages, fixRequestMsg)

	// Auto-scroll to bottom
	a.scrollToBottom()
}

// extractCodeBlocks extracts all code blocks from messages.
// Scans all assistant messages in history and extracts markdown code blocks.
// Updates the codeBlocks slice for use in copy mode.
func (a *AIChatPane) extractCodeBlocks() {
	a.codeBlocks = []string{}
	a.suggestedFile = ""
	for _, msg := range a.messages {
		if msg.Role == "assistant" {
			blocks := extractCodeFromMarkdown(msg.Content)
			a.codeBlocks = append(a.codeBlocks, blocks...)
			// Extract suggested filename from the last assistant message that has one
			if name := extractSuggestedFilename(msg.Content); name != "" {
				a.suggestedFile = name
			}
		}
	}
}

// extractCodeFromMarkdown extracts code blocks from markdown.
// Parses markdown-formatted text and extracts content between ``` markers.
//
// Parameters:
//   - content: Markdown-formatted text
//
// Returns:
//   - []string: Slice of extracted code blocks
func extractCodeFromMarkdown(content string) []string {
	var blocks []string
	lines := strings.Split(content, "\n")
	var currentBlock strings.Builder
	inBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				// End of code block
				blocks = append(blocks, currentBlock.String())
				currentBlock.Reset()
				inBlock = false
			} else {
				// Start of code block
				inBlock = true
			}
		} else if inBlock {
			currentBlock.WriteString(line)
			currentBlock.WriteString("\n")
		}
	}

	return blocks
}

// extractSuggestedFilename looks for a filename suggested by the AI in the response text.
// It searches for patterns like `filename.sh`, "filename.sh", or filename.ext near
// keywords like "save it to", "save it as", "for example", "called", "named".
var suggestedFileRe = regexp.MustCompile("(?i)(?:save (?:it )?(?:to|as)(?: a file)?|for example|called|named|create(?: a file)?)[^`\"]{0,30}[`\"]([\\w/-]+\\.[a-z0-9]{1,4})[`\"]")

func extractSuggestedFilename(content string) string {
	matches := suggestedFileRe.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// ClearHistory clears the conversation history.
// Resets messages and scroll offset. Used for "New Chat" functionality (Ctrl+T).
func (a *AIChatPane) ClearHistory() {
	a.messages = []types.ChatMessage{}
	a.scrollOffset = 0
}

// GetHistory returns conversation history.
// Used for testing and debugging.
//
// Returns:
//   - []types.ChatMessage: Slice of all messages in history
func (a *AIChatPane) GetHistory() []types.ChatMessage {
	return a.messages
}

// Update handles messages for the AI pane.
// Processes keyboard input when focused and handles AI response messages.
//
// Parameters:
//   - msg: The message to handle
//
// Returns:
//   - tea.Cmd: Command to execute (can be nil)
func (a *AIChatPane) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if a.focused {
			return a.handleKeyPress(msg)
		}
	// Removed WindowSizeMsg handling - size is set by app.go
	case AIResponseMsg:
		if msg.Done {
			a.DisplayResponse(msg.Content)
		}
	case AINotificationMsg:
		a.DisplayNotification(msg.Content)
	case TerminalOutputMsg:
		a.terminalOutput = append(a.terminalOutput, msg.Line)
		if a.terminalMode {
			a.viewModeScroll = len(a.terminalOutput)
		}
		return func() tea.Msg {
			return <-msg.Output
		}
	case TerminalDoneMsg:
		a.cmdRunning = false
		a.stdinWriter = nil
		a.terminalInput = ""
		a.terminalOutput = append(a.terminalOutput, "")
		if msg.Err != nil {
			a.terminalOutput = append(a.terminalOutput, fmt.Sprintf("[Process failed: %v]", msg.Err))
		} else {
			a.terminalOutput = append(a.terminalOutput, fmt.Sprintf("[Process exited with code %d]", msg.ExitCode))
		}
		if a.terminalMode {
			a.viewModeScroll = len(a.terminalOutput)
		}
		return nil
	case AIAvailabilityMsg:
		a.aiChecked = true
		a.aiAvailable = msg.Available
		return nil
	}

	return nil
}

// executeCommand executes a script and streams output to Bubble Tea messages.
func (a *AIChatPane) executeCommand(script string) tea.Cmd {
	outChan := make(chan tea.Msg)

	go func() {
		// Preliminary Go initialization check
		if strings.Contains(script, "go get") || strings.Contains(script, "go run") || strings.Contains(script, "go build") || strings.Contains(script, "go test") || strings.Contains(script, "go install") {
			cmdCheck := exec.Command("go", "env", "GOMOD")
			if a.workingDir != "" {
				cmdCheck.Dir = a.workingDir
			}
			out, err := cmdCheck.Output()
			outStr := strings.TrimSpace(string(out))
			if err != nil || outStr == "" || outStr == os.DevNull || outStr == "NUL" {
				baseName := filepath.Base(a.workingDir)
				if baseName == "." || baseName == "" || baseName == string(filepath.Separator) {
					baseName = "ti-project"
				}
				baseName = strings.ReplaceAll(baseName, " ", "-")
				baseName = strings.ToLower(baseName)

				initCmd := exec.Command("go", "mod", "init", baseName)
				if a.workingDir != "" {
					initCmd.Dir = a.workingDir
				}
				initCmd.Run()

				tidyCmd := exec.Command("go", "mod", "tidy")
				if a.workingDir != "" {
					tidyCmd.Dir = a.workingDir
				}
				tidyCmd.Run()
			}
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("powershell", "-NoProfile", "-Command", script)
		} else {
			cmd = exec.Command("sh", "-c", script)
		}

		if a.workingDir != "" {
			cmd.Dir = a.workingDir
		}

		stdin, _ := cmd.StdinPipe()
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		a.stdinWriter = stdin

		if err := cmd.Start(); err != nil {
			outChan <- TerminalDoneMsg{Err: err}
			return
		}

		textChan := make(chan string)
		doneCount := 0

		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				textChan <- scanner.Text()
			}
			textChan <- "\x00DONE\x00"
		}()

		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				textChan <- scanner.Text()
			}
			textChan <- "\x00DONE\x00"
		}()

		for doneCount < 2 {
			text := <-textChan
			if text == "\x00DONE\x00" {
				doneCount++
			} else {
				outChan <- TerminalOutputMsg{Line: text, Output: outChan}
			}
		}

		err := cmd.Wait()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}
		a.stdinWriter = nil
		outChan <- TerminalDoneMsg{ExitCode: exitCode, Err: err}
	}()

	return func() tea.Msg {
		return <-outChan
	}
}

// handleKeyPress handles keyboard input for the AI pane.
// Supports different key bindings based on current mode:
//
// Config mode (editing configuration):
//   - Up/Down/K/J: Navigate fields
//   - Enter: Edit selected field or save changes
//   - Esc/Q: Exit config mode or cancel editing
//   - Backspace: Delete character when editing
//   - Printable characters: Add to edit buffer when editing
//
// View mode (viewing code block):
//   - Esc/Q: Exit view mode
//   - Ctrl+P: Insert code into editor
//   - Up/Down/K/J: Scroll
//   - PgUp/PgDown: Page scroll
//   - Home/End: Jump to top/bottom
//
// Copy mode (selecting code block):
//   - Up/Down/K/J: Navigate blocks
//   - Enter: View selected block
//   - Esc/Q: Exit copy mode
//
// Normal mode - Response area active:
//   - Up/Down/K/J: Scroll responses
//   - PgUp/PgDown: Page scroll
//   - Home/End: Jump to top/bottom
//
// Normal mode - Input area active:
//   - Enter: Send message
//   - Backspace: Delete character
//   - Printable characters: Add to input buffer
//
// Global shortcuts:
//   - Ctrl+Y: Enter copy mode (if code blocks exist)
//
// Parameters:
//   - msg: The key message to handle
//
// Returns:
//   - tea.Cmd: Command to execute (can be nil or custom message)
func (a *AIChatPane) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	// Handle config mode
	if a.configMode {
		if a.editingField {
			// Editing a field value
			switch msg.String() {
			case "enter":
				// Save the edited value
				a.configValues[a.selectedField] = a.editBuffer
				a.editingField = false
				a.editBuffer = ""
				a.editCursorPos = 0
			case "esc":
				// Cancel editing
				a.editingField = false
				a.editBuffer = ""
				a.editCursorPos = 0
			case "left":
				// Move cursor left
				if a.editCursorPos > 0 {
					a.editCursorPos--
				}
			case "right":
				// Move cursor right
				if a.editCursorPos < len(a.editBuffer) {
					a.editCursorPos++
				}
			case "home":
				// Move cursor to start
				a.editCursorPos = 0
			case "end":
				// Move cursor to end
				a.editCursorPos = len(a.editBuffer)
			case "backspace":
				// Delete character before cursor
				if a.editCursorPos > 0 {
					a.editBuffer = a.editBuffer[:a.editCursorPos-1] + a.editBuffer[a.editCursorPos:]
					a.editCursorPos--
				}
			case "delete":
				// Delete character at cursor
				if a.editCursorPos < len(a.editBuffer) {
					a.editBuffer = a.editBuffer[:a.editCursorPos] + a.editBuffer[a.editCursorPos+1:]
				}
			default:
				// Insert character at cursor position
				if len(msg.String()) == 1 {
					r := []rune(msg.String())[0]
					if r >= 32 {
						// Insert character at cursor position
						a.editBuffer = a.editBuffer[:a.editCursorPos] + msg.String() + a.editBuffer[a.editCursorPos:]
						a.editCursorPos++
					}
				}
			}
		} else {
			// Navigating config fields
			switch msg.String() {
			case "up", "k":
				if a.selectedField > 0 {
					a.selectedField--
				}
			case "down", "j":
				if a.selectedField < len(a.configFields)-1 {
					a.selectedField++
				}
			case "enter":
				// Start editing the selected field
				a.editingField = true
				a.editBuffer = a.configValues[a.selectedField]
				a.editCursorPos = len(a.editBuffer) // Start cursor at end
			case "esc", "q":
				// Exit config mode and save changes
				a.configMode = false
				// Return a custom message to trigger config save
				return func() tea.Msg {
					return SaveConfigMsg{
						Fields: a.configFields,
						Values: a.configValues,
					}
				}
			}
		}
		return nil
	}
	// Handle view mode (viewing full code block)
	if a.viewMode {
		keyStr := msg.String()

		// When a command is running in terminal mode, forward input to the process
		if a.terminalMode && a.cmdRunning && a.stdinWriter != nil {
			switch keyStr {
			case "esc":
				// Esc always exits — close stdin and let the process finish
				a.stdinWriter.Close()
				return nil
			case "enter":
				// Send the current input line + newline to the process
				line := a.terminalInput + "\n"
				a.terminalOutput = append(a.terminalOutput, "> "+a.terminalInput)
				a.terminalInput = ""
				a.viewModeScroll = len(a.terminalOutput)
				a.stdinWriter.Write([]byte(line))
				return nil
			case "backspace":
				if len(a.terminalInput) > 0 {
					a.terminalInput = a.terminalInput[:len(a.terminalInput)-1]
				}
				return nil
			case "up", "down", "pgup", "pgdown":
				// Allow scrolling even while command is running
				switch keyStr {
				case "up":
					if a.viewModeScroll > 0 {
						a.viewModeScroll--
					}
				case "down":
					a.viewModeScroll++
				case "pgup":
					a.viewModeScroll -= 10
					if a.viewModeScroll < 0 {
						a.viewModeScroll = 0
					}
				case "pgdown":
					a.viewModeScroll += 10
				}
				return nil
			default:
				// Accumulate printable characters into the terminal input buffer
				if len(keyStr) == 1 {
					r := []rune(keyStr)[0]
					if r >= 32 {
						a.terminalInput += keyStr
					}
				}
				return nil
			}
		}

		switch keyStr {
		case "esc", "q", "2":
			a.viewMode = false
			a.copyMode = true
			a.viewModeScroll = 0
			a.terminalMode = false
		case "ctrl+p", "1":
			// Signal to insert code (will be handled by app)
			a.viewMode = false
			a.copyMode = false
			a.viewModeScroll = 0
			a.terminalMode = false
			// Return a custom message to trigger insert
			return func() tea.Msg {
				return InsertCodeMsg{}
			}
		case "0":
			if !a.cmdRunning {
				a.terminalMode = true
				a.cmdRunning = true

				var initOutput []string
				for _, line := range strings.Split(a.codeBlocks[a.selectedBlock], "\n") {
					initOutput = append(initOutput, "> "+line)
				}
				initOutput = append(initOutput, "")

				a.terminalOutput = initOutput
				a.viewModeScroll = 0
				return a.executeCommand(a.codeBlocks[a.selectedBlock])
			}
		case "up", "k":
			if a.viewModeScroll > 0 {
				a.viewModeScroll--
			}
		case "down", "j":
			a.viewModeScroll++
		case "pgup":
			a.viewModeScroll -= 10
			if a.viewModeScroll < 0 {
				a.viewModeScroll = 0
			}
		case "pgdown":
			a.viewModeScroll += 10
		case "home":
			a.viewModeScroll = 0
			a.viewModeScrollX = 0
		case "end":
			// Will be clamped in renderViewMode
			a.viewModeScroll = 999999
		case "left", "h":
			if a.viewModeScrollX > 0 {
				a.viewModeScrollX -= 5
				if a.viewModeScrollX < 0 {
					a.viewModeScrollX = 0
				}
			}
		case "right", "l":
			a.viewModeScrollX += 5
		default:
			// Store the key pressed for debugging
			a.lastKeyPressed = keyStr
		}
		return nil
	}

	// Handle copy mode
	if a.copyMode {
		switch msg.String() {
		case "up", "k":
			if a.selectedBlock > 0 {
				a.selectedBlock--
			}
		case "down", "j":
			if a.selectedBlock < len(a.codeBlocks)-1 {
				a.selectedBlock++
			}
		case "enter":
			// View selected code block and store it
			if len(a.codeBlocks) > 0 && a.selectedBlock < len(a.codeBlocks) {
				a.lastSelectedCode = a.codeBlocks[a.selectedBlock]
				a.viewMode = true
				a.viewModeScroll = 0
			}
		case "esc", "q":
			a.copyMode = false
		}
		return nil
	}

	// Global shortcuts
	if msg.String() == "ctrl+y" {
		// Enter copy mode if there are code blocks
		if len(a.codeBlocks) > 0 {
			a.copyMode = true
			a.selectedBlock = 0
		}
		return nil
	}

	// Handle input based on active area
	if a.activeArea == 1 {
		// Response area active - handle scrolling
		switch msg.String() {
		case "up", "k":
			if a.scrollOffset > 0 {
				a.scrollOffset--
			}
		case "down", "j":
			maxScroll := a.getMaxScroll()
			if a.scrollOffset < maxScroll {
				a.scrollOffset++
			}
		case "pgup":
			a.scrollOffset -= 10
			if a.scrollOffset < 0 {
				a.scrollOffset = 0
			}
		case "pgdown":
			a.scrollOffset += 10
			maxScroll := a.getMaxScroll()
			if a.scrollOffset > maxScroll {
				a.scrollOffset = maxScroll
			}
		case "home":
			a.scrollOffset = 0
		case "end":
			a.scrollToBottom()
		}
		return nil
	} else {
		// Input area active - handle typing
		switch msg.String() {
		case "enter":
			// Send message when Enter is pressed
			if a.inputBuffer != "" {
				message := a.inputBuffer
				a.inputBuffer = ""
				// Return a custom message to trigger AI message handling in App
				return func() tea.Msg {
					return SendAIMessageMsg{Message: message}
				}
			}
		case "backspace":
			// Delete last character from input buffer
			if len(a.inputBuffer) > 0 {
				a.inputBuffer = a.inputBuffer[:len(a.inputBuffer)-1]
			}
		default:
			// Add character to input buffer
			// Check for printable characters to avoid control chars
			if len(msg.String()) == 1 {
				r := []rune(msg.String())[0]
				if r >= 32 {
					a.inputBuffer += msg.String()
				}
			}
		}
	}

	return nil
}

// getMaxScroll calculates the maximum scroll offset.
// Determines how far the user can scroll based on total content lines and visible area.
//
// Returns:
//   - int: Maximum scroll offset (0 if all content fits in visible area)
func (a *AIChatPane) getMaxScroll() int {
	totalLines := 0
	for _, msg := range a.messages {
		totalLines += a.countMessageLines(msg)
	}

	// Calculate visible lines the same way as in View()
	// Input area: 1 prompt line + 1 status line = 2 content lines + 2 border = 4
	inputHeight := 4
	responseHeight := a.height - inputHeight
	if responseHeight < 5 {
		responseHeight = 5
	}
	visibleLines := responseHeight - 5 // Account for title and borders
	if visibleLines < 1 {
		visibleLines = 1
	}

	maxScroll := totalLines - visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}

	return maxScroll
}

// scrollToBottom scrolls to the bottom of the conversation.
// Sets scroll offset to maximum value, showing the most recent messages.
func (a *AIChatPane) scrollToBottom() {
	a.scrollOffset = a.getMaxScroll()
}

// countMessageLines counts how many lines a message will take.
// Accounts for header, content wrapping, and blank lines.
//
// Parameters:
//   - msg: The message to count lines for
//
// Returns:
//   - int: Number of lines the message will occupy
func (a *AIChatPane) countMessageLines(msg types.ChatMessage) int {
	contentWidth := a.width - 10
	if contentWidth < 20 {
		contentWidth = 20
	}

	lines := 2 // Header line + blank line after message

	// Pre-process tabs which cause visual sizing bugs
	msgContent := strings.ReplaceAll(msg.Content, "\t", "    ")
	contentLines := strings.Split(msgContent, "\n")

	for _, line := range contentLines {
		if len(line) == 0 {
			lines += 1
			continue
		}

		// Each content line can wrap to multiple lines
		wrapped := wrapTextFast(line, contentWidth)
		lines += len(wrapped)
	}

	return lines
}

// renderMessage renders a single message.
// Formats the message with role, timestamp, context indicators, and content.
//
// Styling:
//   - User messages: Blue role label
//   - Assistant messages: Purple role label
//   - Notifications: Cyan role label and content
//   - Context indicator: Cyan "[with context]" tag
//   - Fix request indicator: Yellow "[file: path]" tag
//
// Content wrapping:
// Long lines are wrapped to fit the pane width, accounting for borders and scrollbar.
//
// Parameters:
//   - msg: The message to render
//
// Returns:
//   - []string: Slice of rendered lines (including header, content, and blank line)
func (a *AIChatPane) renderMessage(msg types.ChatMessage) []string {
	var lines []string

	// Header line with role and timestamp
	roleStyle := lipgloss.NewStyle().Bold(true)

	// Use distinct styling for notifications
	if msg.IsNotification {
		roleStyle = roleStyle.Foreground(lipgloss.Color("34")) // Cyan for notifications
	} else if msg.Role == "user" {
		roleStyle = roleStyle.Foreground(lipgloss.Color("39"))
	} else {
		roleStyle = roleStyle.Foreground(lipgloss.Color("170"))
	}

	timeStr := msg.Timestamp.Format("15:04:05")

	// Add notification prefix
	roleText := msg.Role
	if msg.IsNotification {
		roleText = "notification"
	}

	header := roleStyle.Render(roleText) + " " +
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(timeStr)

	if msg.ContextIncluded {
		header += lipgloss.NewStyle().
			Foreground(lipgloss.Color("34")).
			Render(" [with context]")
	}

	// Add file path indicator for fix requests
	if msg.IsFixRequest && msg.FilePath != "" {
		header += lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Render(" [file: " + msg.FilePath + "]")
	}

	lines = append(lines, header)

	contentWidth := a.width - 10
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Apply distinct styling to notification content
	contentStyle := lipgloss.NewStyle()
	if msg.IsNotification {
		contentStyle = contentStyle.Foreground(lipgloss.Color("34")) // Cyan for notification content
	}

	msgContent := strings.ReplaceAll(msg.Content, "\t", "    ")
	contentLines := strings.Split(msgContent, "\n")

	for _, line := range contentLines {
		if len(line) == 0 {
			lines = append(lines, "")
			continue
		}

		// Wrap long lines to prevent them from hiding or breaking layout
		wrapped := wrapTextFast(line, contentWidth)
		for _, wline := range wrapped {
			if msg.IsNotification {
				lines = append(lines, contentStyle.Render(wline))
			} else {
				lines = append(lines, wline)
			}
		}
	}

	// Blank line after message
	lines = append(lines, "")

	return lines
}

// View renders the AI pane with split layout (input top, responses bottom).
// Displays different views based on current mode:
//   - Config mode: Configuration editor with field navigation
//   - View mode: Full code block viewer with scrolling
//   - Copy mode: Code block selection dialog
//   - Normal mode: Input area + response area with conversation history
//
// Normal mode layout:
//   - Input area (top, 3 lines): Prompt with cursor
//   - Response area (bottom): Scrollable conversation history with scrollbar
//
// Scrollbar:
// Displayed when content exceeds visible area. Shows thumb position and size
// to indicate current scroll position and visible portion.
//
// Returns:
//   - string: Rendered AI pane
func (a *AIChatPane) View() string {
	// Show config mode if active
	if a.configMode {
		return a.renderConfigMode()
	}
	// Show view mode if active
	if a.viewMode {
		return a.renderViewMode()
	}

	// Show copy mode dialog if active
	if a.copyMode {
		return a.renderCopyMode()
	}

	// Calculate heights - input gets 2 lines (prompt + status), rest for responses
	maxInputLines := 3
	inputWidth := a.width - 15 // Account for border, padding, and "ai-assist> " prefix
	if inputWidth < 10 {
		inputWidth = 10
	}

	// Calculate how many lines the input will actually take
	promptText := "ai-assist> " + a.inputBuffer
	if a.focused && a.activeArea == 0 {
		promptText += "█" // Cursor
	}

	// Wrap text to fit width and count lines
	wrappedLines := wrapText(promptText, inputWidth)
	actualInputLines := len(wrappedLines)

	// Cap at max input lines and truncate if needed
	if actualInputLines > maxInputLines {
		wrappedLines = wrappedLines[:maxInputLines]
		actualInputLines = maxInputLines
	}

	// Ensure at least 1 line
	if actualInputLines < 1 {
		actualInputLines = 1
	}

	// Build status line to show inside the input box
	statusText := "⚡ " + a.provider + "/" + a.model
	if !a.aiChecked {
		statusText += "  … checking"
	} else if a.aiAvailable {
		statusText += "  ✓ ready"
	} else {
		statusText += "  ✗ No AI Accessible"
	}
	statusColor := "15" // white
	if a.aiChecked && !a.aiAvailable {
		statusColor = "196" // red for unavailable
	}
	if a.streaming {
		statusText = "⚡ " + a.provider + "/" + a.model + "  ⏳ generating..."
		statusColor = "15"
	}
	statusLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor)).
		Render(statusText)

	// Add status as an extra line inside the input box
	wrappedLines = append(wrappedLines, statusLine)
	actualInputLines++

	// Input height includes border (2 lines) + content lines
	inputHeight := actualInputLines + 2
	responseHeight := a.height - inputHeight

	if responseHeight < 5 {
		responseHeight = 5
	}

	// Render input area with border and STRICT width enforcement
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(a.width - 4).
		MaxWidth(a.width).       // Fix: total outer width is a.width
		Height(actualInputLines) // Exact height for content

	if a.focused && a.activeArea == 0 {
		inputStyle = inputStyle.BorderForeground(lipgloss.Color("62"))
	} else {
		inputStyle = inputStyle.BorderForeground(lipgloss.Color("240"))
	}

	inputArea := inputStyle.Render(strings.Join(wrappedLines, "\n"))

	// Render response area
	visibleLines := responseHeight - 5 // Account for title and borders (increased subtraction to match editor height)
	if visibleLines < 1 {
		visibleLines = 1
	}

	// Calculate total lines to determine scrollbar
	totalLines := 0
	for _, msg := range a.messages {
		totalLines += a.countMessageLines(msg)
	}
	// Add expected lines for "AI is thinking..."
	if a.streaming {
		totalLines++
	}

	showScrollbar := totalLines > visibleLines
	var scrollbarThumbStart, scrollbarThumbEnd int

	if showScrollbar {
		scrollbarHeight := float64(visibleLines)
		contentRatio := float64(visibleLines) / float64(totalLines)
		thumbHeight := int(scrollbarHeight * contentRatio)
		if thumbHeight < 1 {
			thumbHeight = 1
		}

		scrollProgress := float64(a.scrollOffset) / float64(totalLines)
		thumbStart := int(scrollbarHeight * scrollProgress)

		// Adjust if thumb goes out of bounds
		if thumbStart+thumbHeight > visibleLines {
			thumbStart = visibleLines - thumbHeight
		}

		scrollbarThumbStart = thumbStart
		scrollbarThumbEnd = thumbStart + thumbHeight
	}

	var renderedLines []string
	currentLine := 0

	for _, msg := range a.messages {
		msgLines := a.renderMessage(msg)
		for _, line := range msgLines {
			if currentLine >= a.scrollOffset && len(renderedLines) < visibleLines {
				renderedLines = append(renderedLines, line)
			}
			currentLine++
		}
	}

	// Add streaming indicator if streaming
	if a.streaming {
		if len(renderedLines) < visibleLines {
			renderedLines = append(renderedLines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render("AI is thinking..."))
		}
	}

	// Content width must match renderMessage() — see comment there for derivation
	contentWidth := a.width - 10

	// Apply scrollbar to existing lines and fill empty lines
	finalLines := make([]string, 0, visibleLines)

	for i := 0; i < visibleLines; i++ {
		// Determine scroll char
		scrollChar := " "
		if showScrollbar {
			if i >= scrollbarThumbStart && i < scrollbarThumbEnd {
				scrollChar = "█"
			} else {
				scrollChar = "│"
			}
		}

		if i < len(renderedLines) {
			line := renderedLines[i]
			lineWidth := lipgloss.Width(line)

			// CRITICAL: Truncate line if it exceeds contentWidth
			if lineWidth > contentWidth {
				// We use wrapTextFast to get strictly bounded slices visually
				// Usually this won't be hit because renderMessage already wraps,
				// but as a fallback it prevents layout crashes.
				wrapped := wrapTextFast(line, contentWidth)
				if len(wrapped) > 0 {
					line = wrapped[0]
				}
				lineWidth = lipgloss.Width(line)
			}

			paddingNeed := contentWidth - lineWidth
			if paddingNeed < 0 {
				paddingNeed = 0
			}

			finalLines = append(finalLines, line+strings.Repeat(" ", paddingNeed)+" "+scrollChar)
		} else {
			// Empty line
			finalLines = append(finalLines, strings.Repeat(" ", contentWidth)+" "+scrollChar)
		}
	}

	// Add title bar for responses
	title := "AI Responses"
	if len(a.messages) > 0 {
		title += " (" + string(rune('0'+len(a.messages)/10%10)) +
			string(rune('0'+len(a.messages)%10)) + ")"
	}

	// Add provider indicator
	if a.provider == "gemini" {
		title += " [Gemini]"
	} else {
		title += " [Ollama]"
	}

	// Add instructions to title
	title += " | Ctrl+Y: Code | ↑↓: Scroll | Ctrl+T: New Chat"

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)

	if a.focused {
		titleStyle = titleStyle.
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62"))
	} else {
		titleStyle = titleStyle.
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("235"))
	}

	// Title bar: no border, just background. MarginLeft(1) adds 1 to total.
	// Response border renders at a.width total. Title should match: Width + margin = a.width
	titleBar := titleStyle.Width(a.width - 5).MaxWidth(a.width - 1).MarginLeft(1).Render(title)

	// Create border style for responses with STRICT width enforcement
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(a.width - 4).
		MaxWidth(a.width). // Fix: total outer width is a.width
		Height(responseHeight - 3)

	if a.focused && a.activeArea == 1 {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("62"))
	} else {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("240"))
	}

	responseContent := strings.Join(finalLines, "\n")

	responseArea := lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		borderStyle.Render(responseContent),
	)

	// CRITICAL: Wrap entire pane in strict width and height container
	paneContainer := lipgloss.NewStyle().
		Width(a.width).
		MaxWidth(a.width).
		Height(a.height).
		MaxHeight(a.height).
		Render(lipgloss.JoinVertical(lipgloss.Left, inputArea, responseArea))

	return paneContainer
}

// renderCopyMode renders the copy mode dialog.
// Displays a list of code blocks with navigation instructions.
// Shows first line of each block as a preview.
//
// Returns:
//   - string: Rendered copy mode dialog
func (a *AIChatPane) renderCopyMode() string {
	var content strings.Builder
	content.WriteString("Select code block to view:\n\n")

	for i, block := range a.codeBlocks {
		prefix := "  "
		if i == a.selectedBlock {
			prefix = "> "
		}

		// Show first line of code block as preview
		lines := strings.Split(strings.TrimSpace(block), "\n")
		preview := lines[0]
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		content.WriteString(prefix + "Block " + string(rune('0'+i+1)) + ": " + preview + "\n")
	}

	content.WriteString("\n[↑↓] Navigate | [Enter] View | [Esc] Cancel")

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(a.width - 8)

	result := dialogStyle.Render(content.String())

	// Wrap in strict container to match pane dimensions
	return lipgloss.NewStyle().
		Width(a.width).MaxWidth(a.width).
		Height(a.height).MaxHeight(a.height).
		Render(result)
}

// renderViewMode renders the full code block view constrained to pane height.
// Displays the selected code block with:
//   - Title bar showing block number
//   - Instructions for navigation and insertion
//   - Scrollable code content with scrollbar
//   - Line truncation to prevent wrapping
//
// The view is sized to match the normal pane height for consistent layout.
//
// Returns:
//   - string: Rendered view mode display
func (a *AIChatPane) renderViewMode() string {
	// In terminal mode, we don't need a code block
	if a.terminalMode {
		// Skip code block validation for terminal mode
	} else if a.selectedBlock >= len(a.codeBlocks) {
		return "Error: Invalid code block"
	}

	var codeBlock string
	if !a.terminalMode {
		codeBlock = a.codeBlocks[a.selectedBlock]
	}

	// Create title - match the width of the normal AI response title bar
	var title string
	if a.terminalMode {
		title = "Terminal Output"
	} else {
		title = "Code Block " + string(rune('0'+a.selectedBlock+1)) + " of " +
			string(rune('0'+len(a.codeBlocks)))
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Padding(0, 1).
		Width(a.width - 4) // Match the code border total width

	titleBar := titleStyle.Render(title)

	var instructions string
	var displayLinesSource []string

	if a.terminalMode {
		displayLinesSource = a.terminalOutput
		// Show input prompt at the bottom if command is still running
		if a.cmdRunning {
			displayLinesSource = append(append([]string{}, a.terminalOutput...), "$ "+a.terminalInput+"█")
			instructions = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("238")).
				Bold(true).
				MarginTop(1).
				MarginBottom(1).
				Padding(0, 1).
				Width(a.width - 4).
				Render(" ⚙  [Enter] Send Input  |  [Esc] Close Stdin  |  [↑↓/PgUp/PgDn] Scroll ")
		} else {
			instructions = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("238")).
				Bold(true).
				MarginTop(1).
				MarginBottom(1).
				Padding(0, 1).
				Width(a.width - 4).
				Render(" ⚙  [2/Esc] Return to Code  |  [↑↓/PgUp/PgDn] Scroll Terminal ")
		}
	} else {
		displayLinesSource = strings.Split(codeBlock, "\n")
		instructions = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("238")).
			Bold(true).
			MarginTop(1).
			MarginBottom(1).
			Padding(0, 1).
			Width(a.width - 4).
			Render(" ⚡ [0] Execute  |  [1/Ctrl+P] Insert  |  [Esc] Return  |  [↑↓/←→] Scroll ")
	}

	// Calculate available height for code content
	// a.height is the total pane height
	// Total must equal a.height to match normal View():
	// titleBar(1) + margin gap(1) + instructions(1) + margin gap(1) + border(2) + codeAreaHeight = a.height
	codeAreaHeight := a.height - 6
	if codeAreaHeight < 3 {
		codeAreaHeight = 3
	}

	totalLines := len(displayLinesSource)

	// Truncate content width to prevent wrapping (account for scrollbar + border + padding)
	contentWidth := a.width - 10
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Clamp scroll offset
	maxScroll := totalLines - codeAreaHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if a.viewModeScroll > maxScroll {
		a.viewModeScroll = maxScroll
	}
	if a.viewModeScroll < 0 {
		a.viewModeScroll = 0
	}

	// Determine scrollbar
	showScrollbar := totalLines > codeAreaHeight
	var scrollbarThumbStart, scrollbarThumbEnd int

	if showScrollbar {
		scrollbarH := float64(codeAreaHeight)
		contentRatio := float64(codeAreaHeight) / float64(totalLines)
		thumbHeight := int(scrollbarH * contentRatio)
		if thumbHeight < 1 {
			thumbHeight = 1
		}

		var scrollProgress float64
		if maxScroll > 0 {
			scrollProgress = float64(a.viewModeScroll) / float64(maxScroll)
		}
		thumbStart := int(float64(codeAreaHeight-thumbHeight) * scrollProgress)
		if thumbStart+thumbHeight > codeAreaHeight {
			thumbStart = codeAreaHeight - thumbHeight
		}
		if thumbStart < 0 {
			thumbStart = 0
		}

		scrollbarThumbStart = thumbStart
		scrollbarThumbEnd = thumbStart + thumbHeight
	}

	// Build display lines with scrollbar
	var displayLines []string
	for i := 0; i < codeAreaHeight; i++ {
		lineIdx := a.viewModeScroll + i

		var line string
		if lineIdx < totalLines {
			line = displayLinesSource[lineIdx]

			// Replace tabs for reliable visual scrolling
			line = strings.ReplaceAll(line, "\t", "    ")

			runes := []rune(line)

			// Apply horizontal scroll
			if a.viewModeScrollX > len(runes) {
				line = ""
			} else if a.viewModeScrollX > 0 {
				line = string(runes[a.viewModeScrollX:])
				runes = []rune(line)
			}

			// Try to truncate accurately based on visual width
			if lipgloss.Width(line) > contentWidth {
				cutAt := contentWidth
				if cutAt > len(runes) {
					cutAt = len(runes)
				}
				for cutAt > 0 && lipgloss.Width(string(runes[:cutAt])) > contentWidth {
					cutAt--
				}
				line = string(runes[:cutAt])
			}
		}

		// Pad line to content width
		lineWidth := lipgloss.Width(line)
		paddingNeed := contentWidth - lineWidth
		if paddingNeed < 0 {
			paddingNeed = 0
		}

		// Add scrollbar character
		scrollChar := " "
		if showScrollbar {
			if i >= scrollbarThumbStart && i < scrollbarThumbEnd {
				scrollChar = "█"
			} else {
				scrollChar = "│"
			}
		}

		displayLines = append(displayLines, line+strings.Repeat(" ", paddingNeed)+" "+scrollChar)
	}

	codeContent := strings.Join(displayLines, "\n")

	// Create code display with strict height - match normal pane width
	codeStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(a.width - 4).
		MaxWidth(a.width). // Cap total rendered width
		Height(codeAreaHeight).
		Foreground(lipgloss.Color("15"))

	result := lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		instructions,
		codeStyle.Render(codeContent),
	)

	// Wrap in strict container to match pane dimensions
	return lipgloss.NewStyle().
		Width(a.width).MaxWidth(a.width).
		Height(a.height).MaxHeight(a.height).
		Render(result)
}

// formatInt converts an integer to a string without fmt.Sprintf.
// Simple utility function for integer to string conversion.
//
// Parameters:
//   - n: Integer to convert
//
// Returns:
//   - string: String representation of the integer
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}

// SetFocused sets the focus state of the AI pane.
// Affects border color and input cursor visibility.
//
// Parameters:
//   - focused: Whether the pane should be focused
func (a *AIChatPane) SetFocused(focused bool) {
	a.focused = focused
}

// SetSize sets the size of the AI pane.
// Called by App when window is resized.
//
// Parameters:
//   - width: New pane width
//   - height: New pane height
func (a *AIChatPane) SetSize(width, height int) {
	a.width = width
	a.height = height
}

// GetWidth returns the width of the AI pane.
// Used for testing and layout calculations.
//
// Returns:
//   - int: Pane width
func (a *AIChatPane) GetWidth() int {
	return a.width
}

// GetHeight returns the height of the AI pane.
// Used for testing and layout calculations.
//
// Returns:
//   - int: Pane height
func (a *AIChatPane) GetHeight() int {
	return a.height
}

// GetCodeBlocks returns the list of code blocks.
// Used for testing and debugging.
//
// Returns:
//   - []string: Slice of extracted code blocks
func (a *AIChatPane) GetCodeBlocks() []string {
	return a.codeBlocks
}

// GetSelectedCodeBlock returns the currently selected code block.
// Returns the last code block that was viewed in view mode.
// Used by App to insert code into editor.
//
// Returns:
//   - string: Selected code block content (empty if none selected)
func (a *AIChatPane) GetSelectedCodeBlock() string {
	return a.lastSelectedCode
}

// GetSuggestedFilename returns the filename the AI suggested for the code, if any.
func (a *AIChatPane) GetSuggestedFilename() string {
	return a.suggestedFile
}

// IsInViewMode returns whether the pane is in view mode.
// Used by App to show appropriate status bar instructions.
//
// Returns:
//   - bool: True if in view mode, false otherwise
func (a *AIChatPane) IsInViewMode() bool {
	return a.viewMode
}

// GetLastAssistantResponse returns the content of the most recent assistant message.
// Searches conversation history in reverse to find the last assistant message.
// Used for Ctrl+A functionality (insert full response into editor).
//
// Returns:
//   - string: Last assistant message content (empty if no assistant messages exist)
func (a *AIChatPane) GetLastAssistantResponse() string {
	// Iterate in reverse to find the last assistant message
	for i := len(a.messages) - 1; i >= 0; i-- {
		if a.messages[i].Role == "assistant" {
			return a.messages[i].Content
		}
	}
	return ""
}

// RunScript enters terminal mode and executes the given command, streaming
// output into the AI chat pane. Returns a tea.Cmd to start the execution.
func (a *AIChatPane) RunScript(command string, label string) tea.Cmd {
	if a.cmdRunning {
		return nil
	}

	// Don't require code blocks for direct file execution
	// This allows running scripts opened with Ctrl+O
	a.viewMode = true
	a.terminalMode = true
	a.cmdRunning = true
	a.copyMode = false

	a.terminalOutput = []string{
		"▶ Running: " + label,
		"> " + command,
		"",
	}
	a.viewModeScroll = 0

	return a.executeCommand(command)
}

// EnterConfigMode enters the configuration editor mode.
// Loads current config values and displays them for editing.
//
// Parameters:
//   - fields: Config field names
//   - values: Config field values
func (a *AIChatPane) EnterConfigMode(fields []string, values []string) {
	a.configMode = true
	a.configFields = fields
	a.configValues = values
	a.selectedField = 0
	a.editingField = false
	a.editBuffer = ""
	a.editCursorPos = 0
}

// renderConfigMode renders the configuration editor.
// Displays config fields with their values and allows editing.
// Shows instructions for navigation and saving.
//
// Returns:
//   - string: Rendered config editor
func (a *AIChatPane) renderConfigMode() string {
	// Create title
	title := "Configuration Editor"

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Padding(0, 1).
		Width(a.width - 3)

	titleBar := titleStyle.Render(title)

	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Padding(0, 1).
		Width(a.width - 4).
		Render("[↑↓] Navigate | [Enter] Edit/Save | [←→] Move Cursor | [Esc] Save & Exit")

	// Calculate available height for config content
	contentHeight := a.height - 4
	if contentHeight < 3 {
		contentHeight = 3
	}

	// Build config fields display
	var displayLines []string

	for i, field := range a.configFields {
		var line string

		if i == a.selectedField {
			if a.editingField {
				// Show edit buffer with cursor at correct position
				prefix := "> " + field + ": "
				beforeCursor := a.editBuffer[:a.editCursorPos]
				afterCursor := a.editBuffer[a.editCursorPos:]

				line = lipgloss.NewStyle().
					Foreground(lipgloss.Color("15")).
					Background(lipgloss.Color("62")).
					Bold(true).
					Render(prefix + beforeCursor + "█" + afterCursor)
			} else {
				// Highlight selected field
				line = lipgloss.NewStyle().
					Foreground(lipgloss.Color("15")).
					Background(lipgloss.Color("62")).
					Bold(true).
					Render("> " + field + ": " + a.configValues[i])
			}
		} else {
			// Normal field display
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Render("  " + field + ": " + a.configValues[i])
		}

		displayLines = append(displayLines, line)
	}

	// Add empty lines to fill content height
	for len(displayLines) < contentHeight {
		displayLines = append(displayLines, "")
	}

	// Truncate if too many lines
	if len(displayLines) > contentHeight {
		displayLines = displayLines[:contentHeight]
	}

	// Pad each line to content width
	contentWidth := a.width - 6
	if contentWidth < 10 {
		contentWidth = 10
	}

	for i, line := range displayLines {
		lineWidth := lipgloss.Width(line)
		paddingNeed := contentWidth - lineWidth
		if paddingNeed < 0 {
			paddingNeed = 0
		}
		displayLines[i] = line + strings.Repeat(" ", paddingNeed)
	}

	configContent := strings.Join(displayLines, "\n")

	// Create config display with border
	configStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(a.width - 4).
		Height(contentHeight).
		Foreground(lipgloss.Color("15"))

	result := lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		instructions,
		configStyle.Render(configContent),
	)

	// Wrap in strict container to match pane dimensions
	return lipgloss.NewStyle().
		Width(a.width).MaxWidth(a.width).
		Height(a.height).MaxHeight(a.height).
		Render(result)
}

// wrapText wraps text to fit within the specified width.
// Returns a slice of lines, each fitting within the width.
func wrapText(text string, width int) []string {
	if width < 1 {
		width = 1
	}

	var lines []string
	var currentLine string

	for _, r := range text {
		if r == '\n' {
			lines = append(lines, currentLine)
			currentLine = ""
			continue
		}

		if lipgloss.Width(currentLine+string(r)) > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = string(r)
			} else {
				// Single character exceeds width, add it anyway
				lines = append(lines, string(r))
			}
		} else {
			currentLine += string(r)
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	if len(lines) == 0 {
		lines = []string{""}
	}

	return lines
}

// wrapTextFast quickly wraps text into lines of maximum width.
// Optimized for mostly ASCII text.
func wrapTextFast(text string, width int) []string {
	if width < 1 {
		width = 1
	}

	runes := []rune(text)

	// Quick path block: Check string width vs byte length
	if len(runes) <= width {
		// Verify there are no exceptionally wide characters hiding
		if lipgloss.Width(text) <= width {
			return []string{text}
		}
	}

	var lines []string
	var currentLine []rune
	currentWidth := 0

	for _, r := range runes {
		w := lipgloss.Width(string(r))
		if currentWidth+w > width {
			if len(currentLine) > 0 {
				lines = append(lines, string(currentLine))
				currentLine = []rune{r}
				currentWidth = w
			} else {
				// Single rune alone exceeds width
				lines = append(lines, string(r))
				currentWidth = 0
			}
		} else {
			currentLine = append(currentLine, r)
			currentWidth += w
		}
	}

	if len(currentLine) > 0 {
		lines = append(lines, string(currentLine))
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}
