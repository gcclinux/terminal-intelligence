package ui

import (
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
	return &AIChatPane{
		messages:     []types.ChatMessage{},
		inputBuffer:  "",
		aiClient:     client,
		model:        model,
		provider:     provider,
		scrollOffset: 0,
		width:        0,
		height:       0,
		focused:      false,
		streaming:    false,
		activeArea:   0, // 0: Input, 1: Response
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
	for _, msg := range a.messages {
		if msg.Role == "assistant" {
			blocks := extractCodeFromMarkdown(msg.Content)
			a.codeBlocks = append(a.codeBlocks, blocks...)
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
		if strings.HasPrefix(line, "```") {
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
	}

	return nil
}

// handleKeyPress handles keyboard input for the AI pane.
// Supports different key bindings based on current mode:
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
	// Handle view mode (viewing full code block)
	if a.viewMode {
		keyStr := msg.String()
		switch keyStr {
		case "esc", "q":
			a.viewMode = false
			a.copyMode = true
			a.viewModeScroll = 0
		case "ctrl+p":
			// Signal to insert code (will be handled by app)
			a.viewMode = false
			a.copyMode = false
			a.viewModeScroll = 0
			// Return a custom message to trigger insert
			return func() tea.Msg {
				return InsertCodeMsg{}
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
		case "end":
			// Will be clamped in renderViewMode
			a.viewModeScroll = 999999
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
	inputHeight := 3
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
	contentWidth := a.width - 8 // Account for padding and borders and scrollbar
	if contentWidth < 10 {
		contentWidth = 10
	}

	lines := 2 // Header line + blank line
	contentLines := strings.Split(msg.Content, "\n")
	for _, line := range contentLines {
		if len(line) == 0 {
			lines++
		} else {
			lines += (len(line) + contentWidth - 1) / contentWidth
		}
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

	// Content lines with wrapping
	contentWidth := a.width - 8 // Account for padding, borders and scrollbar
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Apply distinct styling to notification content
	contentStyle := lipgloss.NewStyle()
	if msg.IsNotification {
		contentStyle = contentStyle.Foreground(lipgloss.Color("34")) // Cyan for notification content
	}

	contentLines := strings.Split(msg.Content, "\n")
	for _, line := range contentLines {
		if len(line) == 0 {
			lines = append(lines, "")
		} else {
			// Wrap long lines
			for len(line) > 0 {
				if len(line) <= contentWidth {
					if msg.IsNotification {
						lines = append(lines, contentStyle.Render(line))
					} else {
						lines = append(lines, line)
					}
					break
				}
				if msg.IsNotification {
					lines = append(lines, contentStyle.Render(line[:contentWidth]))
				} else {
					lines = append(lines, line[:contentWidth])
				}
				line = line[contentWidth:]
			}
		}
	}

	// Blank line after message
	lines = append(lines, "")

	return lines
}

// View renders the AI pane with split layout (input top, responses bottom).
// Displays different views based on current mode:
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
	// Show view mode if active
	if a.viewMode {
		return a.renderViewMode()
	}

	// Show copy mode dialog if active
	if a.copyMode {
		return a.renderCopyMode()
	}

	// Calculate heights - input gets fixed 3 lines, rest for responses
	inputHeight := 3
	responseHeight := a.height - inputHeight // Remove spacing to match editor height

	if responseHeight < 5 {
		responseHeight = 5
	}

	// Render input area with border
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(a.width - 4).
		Height(inputHeight - 2) // Account for border

	if a.focused && a.activeArea == 0 {
		inputStyle = inputStyle.BorderForeground(lipgloss.Color("62"))
	} else {
		inputStyle = inputStyle.BorderForeground(lipgloss.Color("240"))
	}

	promptText := "TI> " + a.inputBuffer
	if a.focused && a.activeArea == 0 {
		promptText += "█" // Cursor
	}
	inputArea := inputStyle.Render(promptText)

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

	// Add padding and scrollbar to each line
	contentWidth := a.width - 8 // Content width used in renderMessage

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

	renderedLines = finalLines

	responseContent := strings.Join(renderedLines, "\n")

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
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("235"))
	}

	// AI Responses [Gemini] | Ctrl+Y: Code | ↑↓: Scroll | Ctrl+T: New Chat titlebar
	titleBar := titleStyle.Width(a.width - 4).MarginLeft(1).Render(title)

	// Create border style for responses
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(a.width - 4).
		Height(responseHeight - 3)

	if a.focused && a.activeArea == 1 {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("62"))
	} else {
		borderStyle = borderStyle.BorderForeground(lipgloss.Color("240"))
	}

	responseArea := lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		borderStyle.Render(responseContent),
	)

	// Join input and response areas vertically (removed instruction bar)
	return lipgloss.JoinVertical(lipgloss.Left, inputArea, responseArea)
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

	return dialogStyle.Render(content.String())
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
	if a.selectedBlock >= len(a.codeBlocks) {
		return "Error: Invalid code block"
	}

	codeBlock := a.codeBlocks[a.selectedBlock]

	// Create title - match the width of the normal AI response title bar
	title := "Code Block " + string(rune('0'+a.selectedBlock+1)) + " of " +
		string(rune('0'+len(a.codeBlocks)))

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Padding(0, 1).
		Width(a.width - 3) // Match the normal title bar width

	titleBar := titleStyle.Render(title)

	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(a.width - 4). // Match the normal pane width
		Render("[Ctrl+P] Insert into Editor | [↑↓/PgUp/PgDn] Scroll | [Esc] Back")

	// Calculate available height for code content
	// a.height is the total pane height
	// Total must equal a.height to match normal View():
	// titleBar(1) + instructions(1) + border(2) + codeAreaHeight = a.height
	codeAreaHeight := a.height - 4
	if codeAreaHeight < 3 {
		codeAreaHeight = 3
	}

	// Split code block into lines
	codeLines := strings.Split(codeBlock, "\n")
	totalCodeLines := len(codeLines)

	// Truncate content width to prevent wrapping (account for scrollbar + border + padding)
	contentWidth := a.width - 10
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Clamp scroll offset
	maxScroll := totalCodeLines - codeAreaHeight
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
	showScrollbar := totalCodeLines > codeAreaHeight
	var scrollbarThumbStart, scrollbarThumbEnd int

	if showScrollbar {
		scrollbarH := float64(codeAreaHeight)
		contentRatio := float64(codeAreaHeight) / float64(totalCodeLines)
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
		if lineIdx < totalCodeLines {
			line = codeLines[lineIdx]
			if len(line) > contentWidth {
				line = line[:contentWidth-3] + "..."
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
		Width(a.width - 4). // Match the normal pane width
		Height(codeAreaHeight).
		Foreground(lipgloss.Color("15"))

	return lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		instructions,
		codeStyle.Render(codeContent),
	)
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
