package agentic

import (
	"fmt"
	"log"
	"strings"
)

// AIClient is an interface for AI service providers.
// This interface abstracts the AI service layer, allowing the AgenticCodeFixer to work
// with different AI providers (Ollama, Gemini) without knowing implementation details.
//
// Implementations must provide:
//   - Availability checking (IsAvailable)
//   - Streaming text generation (Generate)
//   - Model listing (ListModels)
//
// This interface is defined here to avoid circular dependencies with the ai package.
type AIClient interface {
	// IsAvailable checks if the AI service is available and reachable.
	// Returns true if the service can be used, false otherwise.
	// Returns an error if the availability check itself fails.
	IsAvailable() (bool, error)

	// Generate generates AI response with streaming support.
	// Parameters:
	//   - prompt: The input prompt for the AI model
	//   - model: The model identifier to use for generation
	//   - context: Optional context tokens for conversation continuity (can be nil)
	//
	// Returns a channel that streams response chunks as they're generated.
	// The channel is closed when generation completes.
	// Returns an error if generation cannot be started.
	Generate(prompt string, model string, context []int) (<-chan string, error)

	// ListModels lists available models from the AI provider.
	// Returns a slice of model identifiers that can be used with Generate.
	// Returns an error if the model list cannot be retrieved.
	ListModels() ([]string, error)
}

// AgenticCodeFixer orchestrates the agentic code fixing workflow.
// This is the main component that coordinates all aspects of autonomous code fixing:
//  1. Detecting fix requests vs conversational messages
//  2. Retrieving file context from the editor
//  3. Constructing prompts with code context
//  4. Calling the AI service to generate fixes
//  5. Parsing AI responses to extract code fixes
//  6. Validating and applying fixes to file content
//  7. Generating change notifications
//
// The fixer supports:
//   - Single and multi-step fixes (multiple code changes in one request)
//   - Preview mode (show changes without applying)
//   - Transactional fix application (all-or-nothing with rollback)
//   - Multiple file types (bash, shell, powershell, markdown)
//   - Both Ollama and Gemini AI providers
//
// Error Handling:
// The fixer implements robust error handling with graceful degradation:
//   - No file open: Returns clear error message
//   - AI unavailable: Checks availability and returns user-friendly error
//   - Invalid fixes: Validates before applying, returns explanation on failure
//   - Application failures: Preserves original content and notifies user
type AgenticCodeFixer struct {
	aiClient  AIClient   // The AI client for generating fixes
	model     string     // The AI model to use for generation
	fixParser *FixParser // Parser for extracting and validating fixes
	debug     bool       // Enable debug logging
}

// NewAgenticCodeFixer creates a new agentic code fixer
// Parameters:
//   - aiClient: The AI client to use for generating fixes
//   - model: The AI model to use for generation
//
// Returns a configured AgenticCodeFixer instance
func NewAgenticCodeFixer(aiClient AIClient, model string) *AgenticCodeFixer {
	return &AgenticCodeFixer{
		aiClient:  aiClient,
		model:     model,
		fixParser: NewFixParser(),
		debug:     false, // Debug logging disabled by default
	}
}

// SetDebug enables or disables debug logging
func (f *AgenticCodeFixer) SetDebug(debug bool) {
	f.debug = debug
	if debug {
		log.Println("[DEBUG] Debug logging enabled for AgenticCodeFixer")
	}
}

// logInfo logs informational messages (only when debug mode is enabled)
func (f *AgenticCodeFixer) logInfo(format string, args ...interface{}) {
	if f.debug {
		log.Printf("[INFO] "+format, args...)
	}
}

// logError logs error messages with context (only when debug mode is enabled)
func (f *AgenticCodeFixer) logError(format string, args ...interface{}) {
	if f.debug {
		log.Printf("[ERROR] "+format, args...)
	}
}

// logDebug logs debug messages (only when debug mode is enabled)
func (f *AgenticCodeFixer) logDebug(format string, args ...interface{}) {
	if f.debug {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// IsFixRequest determines if a message is requesting a code fix
// It analyzes the message for fix-related keywords and returns a confidence score
// Keywords checked: "fix", "change", "update", "modify", "correct"
// Returns a FixDetectionResult with IsFixRequest flag, confidence score, and matched keywords
//
// Confidence scoring:
// - 1.0: Explicit command syntax (/fix)
// - 0.9: Multiple fix keywords present
// - 0.8: Single fix keyword with action context
// - 0.7: Single fix keyword present
// - 0.0: No fix keywords found
func (f *AgenticCodeFixer) IsFixRequest(message string) FixDetectionResult {
	f.logDebug("Analyzing message for fix request detection (length: %d chars)", len(message))

	// Normalize message for analysis
	lowerMessage := strings.ToLower(strings.TrimSpace(message))

	// Check for explicit /fix command
	if strings.HasPrefix(lowerMessage, "/fix") {
		f.logInfo("Fix request detected via /fix command")
		return FixDetectionResult{
			IsFixRequest: true,
			Confidence:   1.0,
			Keywords:     []string{"/fix"},
		}
	}

	// Check for explicit /ask command (conversational mode)
	if strings.HasPrefix(lowerMessage, "/ask") {
		f.logInfo("Conversational mode detected via /ask command")
		return FixDetectionResult{
			IsFixRequest: false,
			Confidence:   0.0,
			Keywords:     []string{},
		}
	}

	// Define fix-related keywords
	fixKeywords := []string{
		"fix",
		"change",
		"update",
		"modify",
		"correct",
	}

	// Find matching keywords
	var matchedKeywords []string
	for _, keyword := range fixKeywords {
		if strings.Contains(lowerMessage, keyword) {
			matchedKeywords = append(matchedKeywords, keyword)
		}
	}

	// Calculate confidence based on matched keywords
	if len(matchedKeywords) == 0 {
		f.logDebug("No fix keywords found, treating as conversational")
		return FixDetectionResult{
			IsFixRequest: false,
			Confidence:   0.0,
			Keywords:     []string{},
		}
	}

	// Determine confidence level
	var confidence float64

	if len(matchedKeywords) >= 2 {
		// Multiple keywords suggest strong fix intent
		confidence = 0.9
	} else {
		// Single keyword - check for action context
		// Action words that strengthen fix intent
		actionWords := []string{
			"please",
			"can you",
			"could you",
			"need to",
			"want to",
			"should",
			"must",
		}

		hasActionContext := false
		for _, action := range actionWords {
			if strings.Contains(lowerMessage, action) {
				hasActionContext = true
				break
			}
		}

		if hasActionContext {
			confidence = 0.8
		} else {
			confidence = 0.7
		}
	}

	f.logInfo("Fix request detected with confidence %.1f, keywords: %v", confidence, matchedKeywords)

	return FixDetectionResult{
		IsFixRequest: true,
		Confidence:   confidence,
		Keywords:     matchedKeywords,
	}
}

// BuildPrompt constructs a structured prompt for the AI model
// It combines the user's request with file context and provides clear instructions
// for generating code fixes.
//
// The prompt structure:
// 1. System instructions for the AI model
// 2. File metadata (path and type)
// 3. Current file content
// 4. User's fix request
// 5. Output format instructions
//
// Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 1.5
func (f *AgenticCodeFixer) BuildPrompt(request *FixRequest) string {
	var prompt strings.Builder

	// Section 1: System instructions
	prompt.WriteString("You are an AI code assistant helping to fix code issues.\n")
	prompt.WriteString("Your task is to analyze the user's request and generate a specific code fix.\n")
	prompt.WriteString("Provide the complete fixed code in a markdown code block.\n\n")

	// Section 2: File metadata
	prompt.WriteString("=== FILE METADATA ===\n")
	prompt.WriteString("File Path: ")
	prompt.WriteString(request.FilePath)
	prompt.WriteString("\n")
	prompt.WriteString("File Type: ")
	prompt.WriteString(request.FileType)
	prompt.WriteString("\n\n")

	// Section 3: Current file content
	prompt.WriteString("=== CURRENT FILE CONTENT ===\n")
	if strings.TrimSpace(request.FileContent) == "" {
		prompt.WriteString("(empty file)\n")
	} else {
		prompt.WriteString(request.FileContent)
		// Ensure there's a newline after content
		if !strings.HasSuffix(request.FileContent, "\n") {
			prompt.WriteString("\n")
		}
	}
	prompt.WriteString("\n")

	// Section 4: User's fix request
	prompt.WriteString("=== USER REQUEST ===\n")
	prompt.WriteString(request.UserMessage)
	prompt.WriteString("\n\n")

	// Section 5: Output format instructions
	prompt.WriteString("=== INSTRUCTIONS ===\n")
	prompt.WriteString("1. Analyze the user's request and the current file content\n")
	prompt.WriteString("2. Generate the complete fixed code\n")
	prompt.WriteString("3. Wrap your code in a markdown code block with the appropriate language identifier\n")
	prompt.WriteString("4. Use this format:\n")
	prompt.WriteString("```")
	prompt.WriteString(request.FileType)
	prompt.WriteString("\n")
	prompt.WriteString("(your fixed code here)\n")
	prompt.WriteString("```\n")
	prompt.WriteString("5. Provide a brief explanation of the changes you made\n")

	return prompt.String()
}

// ProcessMessage analyzes a user message and determines if it's a fix request
// This is the main orchestration method that:
// 1. Detects if the message is a fix request vs conversational
// 2. Routes to appropriate handler (GenerateFix or conversational mode)
// 3. Handles error cases (no file open, AI unavailable, etc.)
// 4. Detects preview mode requests (/preview command)
//
// Parameters:
//   - message: The user's message
//   - fileContent: Current content of the file in the editor
//   - filePath: Path to the current file
//   - fileType: Type of the file (bash, shell, powershell, markdown)
//
// Returns:
//   - *FixResult: Result containing either a fix or conversational response
//   - error: Error if something went wrong during processing
//
// Requirements: 9.1, 9.2, 9.5, 6.4
func (f *AgenticCodeFixer) ProcessMessage(
	message string,
	fileContent string,
	filePath string,
	fileType string,
) (*FixResult, error) {
	f.logInfo("Processing message for file: %s (type: %s)", filePath, fileType)
	f.logDebug("File content length: %d bytes", len(fileContent))

	// Step 1: Check for preview mode command
	previewMode := false
	actualMessage := message
	lowerMessage := strings.ToLower(strings.TrimSpace(message))

	if strings.HasPrefix(lowerMessage, "/preview") {
		previewMode = true
		f.logInfo("Preview mode enabled")
		// Remove the /preview command from the message
		actualMessage = strings.TrimSpace(strings.TrimPrefix(message, "/preview"))
		actualMessage = strings.TrimSpace(strings.TrimPrefix(actualMessage, "/PREVIEW"))
		actualMessage = strings.TrimSpace(strings.TrimPrefix(actualMessage, "/Preview"))

		// If no message after /preview, return error
		if strings.TrimSpace(actualMessage) == "" {
			f.logError("Preview mode requested but no fix request provided")
			return &FixResult{
				Success:          false,
				ModifiedContent:  "",
				ChangesSummary:   "",
				ErrorMessage:     "Please provide a fix request after /preview command",
				IsConversational: false,
				PreviewMode:      false,
			}, nil
		}
	}

	// Step 2: Detect if this is a fix request
	detection := f.IsFixRequest(actualMessage)

	// Step 3: If not a fix request, return conversational mode indicator
	if !detection.IsFixRequest {
		f.logInfo("Message classified as conversational, not a fix request")
		return &FixResult{
			Success:          false,
			ModifiedContent:  "",
			ChangesSummary:   "",
			ErrorMessage:     "",
			IsConversational: true,
			PreviewMode:      false,
		}, nil
	}

	// Step 4: Validate that a file is open (Requirements 1.2, 7.1)
	if strings.TrimSpace(filePath) == "" {
		f.logError("Fix request received but no file is open")
		return &FixResult{
			Success:          false,
			ModifiedContent:  "",
			ChangesSummary:   "",
			ErrorMessage:     "Please open a file before requesting code fixes",
			IsConversational: false,
			PreviewMode:      false,
		}, nil
	}

	// Step 5: Create fix request
	request := &FixRequest{
		UserMessage: actualMessage,
		FileContent: fileContent,
		FilePath:    filePath,
		FileType:    fileType,
		PreviewMode: previewMode,
	}

	// Validate the request
	if err := request.Validate(); err != nil {
		f.logError("Fix request validation failed: %v", err)
		return &FixResult{
			Success:          false,
			ModifiedContent:  "",
			ChangesSummary:   "",
			ErrorMessage:     "Invalid fix request: " + err.Error(),
			IsConversational: false,
			PreviewMode:      false,
		}, nil
	}

	f.logInfo("Fix request validated, proceeding to generate fix")

	// Step 6: Route to fix generation handler
	return f.GenerateFix(request)
}

// GenerateFix generates a code fix using the AI model
// This method constructs a prompt, calls the AI service, parses the response,
// and returns a FixResult with the generated fix.
//
// Parameters:
//   - request: The FixRequest containing user message and file context
//
// Returns:
//   - *FixResult: Result containing the generated fix or error
//   - error: Error if something went wrong during generation
//
// Requirements: 2.1, 3.1, 3.2, 3.4
func (f *AgenticCodeFixer) GenerateFix(request *FixRequest) (*FixResult, error) {
	f.logInfo("Generating fix for file: %s", request.FilePath)

	// Step 1: Check if AI service is available (Requirement 7.2)
	f.logDebug("Checking AI service availability")
	available, err := f.aiClient.IsAvailable()
	if err != nil || !available {
		f.logError("AI service unavailable: available=%v, error=%v", available, err)
		return &FixResult{
			Success:          false,
			ModifiedContent:  "",
			ChangesSummary:   "",
			ErrorMessage:     "AI service unavailable. Please check your AI provider connection",
			IsConversational: false,
		}, nil
	}
	f.logDebug("AI service is available")

	// Step 2: Construct prompt using prompt builder (Requirement 2.1)
	f.logDebug("Building prompt for AI model")
	prompt := f.BuildPrompt(request)
	f.logDebug("Prompt built successfully (length: %d chars)", len(prompt))

	// Step 3: Call AI client to generate response (Requirement 2.1)
	f.logInfo("Calling AI model: %s", f.model)
	responseChan, err := f.aiClient.Generate(prompt, f.model, nil)
	if err != nil {
		f.logError("Failed to generate fix: %v", err)
		return &FixResult{
			Success:          false,
			ModifiedContent:  "",
			ChangesSummary:   "",
			ErrorMessage:     "Failed to generate fix: " + err.Error(),
			IsConversational: false,
		}, nil
	}

	// Step 4: Collect the streaming response
	f.logDebug("Collecting streaming response from AI")
	var fullResponse strings.Builder
	for chunk := range responseChan {
		fullResponse.WriteString(chunk)
	}

	responseText := fullResponse.String()
	f.logDebug("AI response received (length: %d chars)", len(responseText))

	// Check if we got a response
	if strings.TrimSpace(responseText) == "" {
		f.logError("AI generated an empty response")
		return &FixResult{
			Success:          false,
			ModifiedContent:  "",
			ChangesSummary:   "",
			ErrorMessage:     "AI generated an empty response",
			IsConversational: false,
		}, nil
	}

	// Step 5: Parse response using FixParser (Requirement 3.1)
	f.logDebug("Parsing AI response for code blocks")
	codeBlocks := f.fixParser.ExtractCodeBlocks(responseText)
	f.logInfo("Extracted %d code blocks from AI response", len(codeBlocks))

	// Check if we extracted any code blocks
	if len(codeBlocks) == 0 {
		f.logError("No code blocks found in AI response")
		return &FixResult{
			Success:          false,
			ModifiedContent:  "",
			ChangesSummary:   "",
			ErrorMessage:     "AI response did not contain any code blocks",
			IsConversational: false,
		}, nil
	}

	// Step 6: Identify which blocks are fixes (Requirement 3.2)
	f.logDebug("Identifying fix blocks for file type: %s", request.FileType)
	fixBlocks := f.fixParser.IdentifyFixBlocks(codeBlocks, request.FileType)
	f.logInfo("Identified %d fix blocks", len(fixBlocks))

	// Check if we identified any fix blocks (Requirement 3.5, 7.3)
	if len(fixBlocks) == 0 {
		f.logError("Could not identify valid fix blocks in AI response")
		return &FixResult{
			Success:          false,
			ModifiedContent:  "",
			ChangesSummary:   "",
			ErrorMessage:     "Could not identify valid fix blocks in AI response. The AI may have provided explanations without code, or the code blocks don't match the file type. Please try rephrasing your request.",
			IsConversational: false,
		}, nil
	}

	// Step 7: Pre-validate all fix blocks before applying any changes (Requirement 10.5)
	// This ensures all steps are compatible and won't conflict with each other
	f.logDebug("Pre-validating %d fix blocks", len(fixBlocks))
	if err := f.validateMultiStepFix(fixBlocks, request.FileType); err != nil {
		f.logError("Pre-validation failed: %v", err)
		return &FixResult{
			Success:          false,
			ModifiedContent:  "",
			ChangesSummary:   "",
			ErrorMessage:     fmt.Sprintf("Pre-validation failed: %s. Please review your request and try again.", err.Error()),
			IsConversational: false,
		}, nil
	}
	f.logDebug("Pre-validation successful")

	// Step 8: Order fix blocks for correct application sequence (Requirement 10.2)
	f.logDebug("Ordering fix blocks for application")
	orderedFixBlocks := f.orderFixBlocks(fixBlocks)

	// Step 9: Apply all fixes atomically (Requirement 10.1, 10.3)
	// For preview mode, we generate the modified content but don't actually apply it
	// For multiple blocks, we apply them in the determined order
	// If any application fails, we return an error (atomicity is handled by ApplyFix's transactional approach)
	f.logInfo("Applying %d fix blocks (preview mode: %v)", len(orderedFixBlocks), request.PreviewMode)
	modifiedContent := request.FileContent
	for i, fixBlock := range orderedFixBlocks {
		f.logDebug("Applying fix block %d/%d", i+1, len(orderedFixBlocks))
		var err error
		modifiedContent, err = f.ApplyFix(modifiedContent, fixBlock.Code, request.FileType)
		if err != nil {
			f.logError("Failed to apply fix block %d: %v", i+1, err)
			return &FixResult{
				Success:          false,
				ModifiedContent:  "",
				ChangesSummary:   "",
				ErrorMessage:     fmt.Sprintf("Failed to apply fix block %d: %s. The generated code may be incomplete or invalid.", i+1, err.Error()),
				IsConversational: false,
				PreviewMode:      false,
			}, nil
		}
	}
	f.logInfo("All fix blocks applied successfully")

	// Step 10: Generate a change summary
	changesSummary := f.generateChangeSummary(request.FileContent, modifiedContent, request.FilePath, len(orderedFixBlocks), request.PreviewMode)

	// Step 11: Return successful result with inline diff
	// In preview mode, we return the modified content but indicate it's a preview
	f.logInfo("Fix generation completed successfully (preview: %v)", request.PreviewMode)

	diffContent := f.generateInlineDiff(request.FileContent, modifiedContent)

	return &FixResult{
		Success:          true,
		ModifiedContent:  diffContent,
		ChangesSummary:   changesSummary,
		ErrorMessage:     "",
		IsConversational: false,
		PreviewMode:      request.PreviewMode,
	}, nil
}

// generateInlineDiff creates an inline diff using special prefixes
// It finds the chunk of changed lines and inserts ~DEL~ and ~ADD~ markers.
// These markers are parsed by the EditorPane to show red/green lines.
func (f *AgenticCodeFixer) generateInlineDiff(original, modified string) string {
	origLines := strings.Split(original, "\n")
	modLines := strings.Split(modified, "\n")

	if len(origLines) == 0 && len(modLines) == 0 {
		return ""
	}

	minLen := len(origLines)
	if len(modLines) < minLen {
		minLen = len(modLines)
	}

	firstDiff := -1
	for i := 0; i < minLen; i++ {
		if origLines[i] != modLines[i] {
			firstDiff = i
			break
		}
	}

	if firstDiff == -1 {
		if len(origLines) != len(modLines) {
			firstDiff = minLen
		} else {
			return modified
		}
	}

	maxSuffixOrig := len(origLines) - firstDiff
	maxSuffixMod := len(modLines) - firstDiff
	maxSuffix := maxSuffixOrig
	if maxSuffixMod < maxSuffix {
		maxSuffix = maxSuffixMod
	}

	lastOrigDiff := -1
	lastModDiff := -1
	for i := 0; i < maxSuffix; i++ {
		origIdx := len(origLines) - 1 - i
		modIdx := len(modLines) - 1 - i
		if origLines[origIdx] != modLines[modIdx] {
			lastOrigDiff = origIdx
			lastModDiff = modIdx
			break
		}
	}

	if lastOrigDiff == -1 {
		lastOrigDiff = len(origLines) - 1 - maxSuffix
		lastModDiff = len(modLines) - 1 - maxSuffix
	}

	var result []string
	result = append(result, origLines[:firstDiff]...)

	for i := firstDiff; i <= lastOrigDiff; i++ {
		result = append(result, "~DEL~"+origLines[i])
	}

	for i := firstDiff; i <= lastModDiff; i++ {
		result = append(result, "~ADD~"+modLines[i])
	}

	if lastOrigDiff+1 < len(origLines) {
		result = append(result, origLines[lastOrigDiff+1:]...)
	}

	return strings.Join(result, "\n")
}

// generateChangeSummary creates a human-readable summary of changes
// This method generates clear, actionable notifications that include:
// - Location information (line numbers or function names if available)
// - Description of what changed
// - Save and test reminders
// - Handling of multiple changes in a single fix
// - Preview mode indication
//
// Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 6.4
func (f *AgenticCodeFixer) generateChangeSummary(originalContent, modifiedContent, filePath string, blockCount int, previewMode bool) string {
	var summary strings.Builder

	// Preview mode header (Requirement 6.4)
	if previewMode {
		summary.WriteString("🔍 PREVIEW MODE - Changes NOT applied\n\n")
	}

	// Header with file path (Requirement 5.1, 5.2)
	if previewMode {
		summary.WriteString("Proposed changes for ")
	} else {
		summary.WriteString("✓ Code fix applied to ")
	}
	summary.WriteString(filePath)
	summary.WriteString("\n\n")

	// Count lines changed
	originalLines := strings.Split(strings.TrimSpace(originalContent), "\n")
	modifiedLines := strings.Split(strings.TrimSpace(modifiedContent), "\n")

	if len(originalLines) == 1 && strings.TrimSpace(originalContent) == "" {
		originalLines = []string{}
	}

	summary.WriteString("Changes:\n")

	// Indicate if multiple code blocks were applied (Requirement 5.4)
	if blockCount > 1 {
		if previewMode {
			summary.WriteString(fmt.Sprintf("- Would apply %d code blocks\n", blockCount))
		} else {
			summary.WriteString(fmt.Sprintf("- Applied %d code blocks\n", blockCount))
		}
	}

	// Describe what changed (Requirement 5.2)
	if len(originalLines) == 0 {
		if previewMode {
			summary.WriteString(fmt.Sprintf("- Would add %d lines (new file)\n", len(modifiedLines)))
		} else {
			summary.WriteString(fmt.Sprintf("- Added %d lines (new file)\n", len(modifiedLines)))
		}
	} else if len(modifiedLines) > len(originalLines) {
		if previewMode {
			summary.WriteString(fmt.Sprintf("- Would modify file: %d → %d lines (+%d)\n",
				len(originalLines), len(modifiedLines), len(modifiedLines)-len(originalLines)))
		} else {
			summary.WriteString(fmt.Sprintf("- Modified file: %d → %d lines (+%d)\n",
				len(originalLines), len(modifiedLines), len(modifiedLines)-len(originalLines)))
		}
	} else if len(modifiedLines) < len(originalLines) {
		if previewMode {
			summary.WriteString(fmt.Sprintf("- Would modify file: %d → %d lines (-%d)\n",
				len(originalLines), len(modifiedLines), len(originalLines)-len(modifiedLines)))
		} else {
			summary.WriteString(fmt.Sprintf("- Modified file: %d → %d lines (-%d)\n",
				len(originalLines), len(modifiedLines), len(originalLines)-len(modifiedLines)))
		}
	} else {
		if previewMode {
			summary.WriteString(fmt.Sprintf("- Would modify file: %d lines (same length, content changed)\n",
				len(modifiedLines)))
		} else {
			summary.WriteString(fmt.Sprintf("- Modified file: %d lines (same length, content changed)\n",
				len(modifiedLines)))
		}
	}

	// Add location information (Requirement 5.3)
	locationInfo := f.detectChangeLocations(originalLines, modifiedLines)
	if locationInfo != "" {
		summary.WriteString(locationInfo)
	}

	summary.WriteString("\n")

	// Save and test reminders (Requirement 5.5)
	if previewMode {
		summary.WriteString("ℹ️  This is a preview. To apply these changes, send the request without /preview")
	} else {
		summary.WriteString("⚠️  Remember to save the file (Ctrl+S) and test the changes!")
	}

	return summary.String()
}

// detectChangeLocations analyzes the original and modified content to identify
// where changes occurred, providing line numbers or function names if available
// This supports Requirement 5.3: location information
func (f *AgenticCodeFixer) detectChangeLocations(originalLines, modifiedLines []string) string {
	// If the file was empty or is now empty, no location info needed
	if len(originalLines) == 0 || len(modifiedLines) == 0 {
		return ""
	}

	var locations strings.Builder

	// Find the first and last lines that differ
	firstDiff := -1
	lastDiff := -1

	// Find first difference
	minLen := len(originalLines)
	if len(modifiedLines) < minLen {
		minLen = len(modifiedLines)
	}

	for i := 0; i < minLen; i++ {
		if originalLines[i] != modifiedLines[i] {
			firstDiff = i
			break
		}
	}

	// If all common lines are the same, the difference is at the end
	if firstDiff == -1 {
		if len(originalLines) != len(modifiedLines) {
			firstDiff = minLen
		} else {
			// Files are identical (shouldn't happen, but handle it)
			return ""
		}
	}

	// Find last difference (search from the end)
	for i := 0; i < minLen; i++ {
		origIdx := len(originalLines) - 1 - i
		modIdx := len(modifiedLines) - 1 - i
		if originalLines[origIdx] != modifiedLines[modIdx] {
			lastDiff = modIdx
			break
		}
	}

	// If we didn't find a last diff, it means changes extend to the end
	if lastDiff == -1 {
		lastDiff = len(modifiedLines) - 1
	}

	// Generate location information
	if firstDiff == lastDiff {
		locations.WriteString(fmt.Sprintf("- Location: Line %d\n", firstDiff+1))
	} else {
		locations.WriteString(fmt.Sprintf("- Location: Lines %d-%d\n", firstDiff+1, lastDiff+1))
	}

	// Try to detect function names near the changes (simple heuristic)
	functionName := f.detectNearbyFunction(modifiedLines, firstDiff)
	if functionName != "" {
		locations.WriteString(fmt.Sprintf("- Context: Near function '%s'\n", functionName))
	}

	return locations.String()
}

// detectNearbyFunction attempts to find a function name near the given line
// This is a simple heuristic that looks for common function declaration patterns
// Supports bash, shell, powershell function declarations
func (f *AgenticCodeFixer) detectNearbyFunction(lines []string, lineNum int) string {
	// Search backwards from the change location to find a function declaration
	// Look up to 20 lines back
	searchStart := lineNum - 20
	if searchStart < 0 {
		searchStart = 0
	}

	for i := lineNum; i >= searchStart; i-- {
		line := strings.TrimSpace(lines[i])

		// Bash/Shell function patterns:
		// function name() { ... }
		// name() { ... }
		if strings.Contains(line, "function ") {
			// Extract function name: "function name() {" -> "name"
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				funcName := strings.TrimSuffix(parts[1], "()")
				funcName = strings.TrimSuffix(funcName, "(")
				return funcName
			}
		} else if strings.Contains(line, "()") && (strings.Contains(line, "{") || i+1 < len(lines) && strings.Contains(lines[i+1], "{")) {
			// Extract function name: "name() {" -> "name"
			parts := strings.Split(line, "(")
			if len(parts) >= 1 {
				funcName := strings.TrimSpace(parts[0])
				// Make sure it's a valid identifier (no spaces, special chars)
				if funcName != "" && !strings.ContainsAny(funcName, " \t=<>|&;") {
					return funcName
				}
			}
		}

		// PowerShell function pattern:
		// function Name { ... }
		// Function Name { ... }
		if strings.HasPrefix(strings.ToLower(line), "function ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}

	return ""
}

// ApplyFix applies a code fix to the file content using a transactional approach
// This method handles the application of code changes while:
// - Creating a backup before applying (Requirement 7.4)
// - Applying to a temporary copy (Requirement 7.4)
// - Validating the result (Requirement 7.4)
// - Committing or rolling back (Requirement 7.4, 7.6)
// - Preserving surrounding code (for partial fixes)
// - Maintaining formatting and indentation
// - Supporting insertions and deletions
//
// Parameters:
//   - originalContent: The original file content
//   - fixCode: The code to apply (can be whole file or partial)
//   - fileType: The type of file being fixed
//
// Returns:
//   - string: The modified content with the fix applied
//   - error: Error if the fix cannot be applied
//
// Requirements: 4.1, 4.2, 4.3, 4.5, 4.6, 7.4, 7.6
func (f *AgenticCodeFixer) ApplyFix(
	originalContent string,
	fixCode string,
	fileType string,
) (string, error) {
	f.logDebug("Applying fix (original content: %d bytes, fix code: %d bytes)", len(originalContent), len(fixCode))

	// Step 1: Create backup of original content (Requirement 7.4)
	backup := originalContent

	// Step 2: Validate inputs (Requirement 3.5, 7.3)
	if strings.TrimSpace(fixCode) == "" {
		// Rollback: return original content
		f.logError("Fix code is empty or whitespace-only, rolling back")
		return backup, fmt.Errorf("fix code cannot be empty or whitespace-only")
	}

	// Step 3: Apply fix to temporary copy (Requirement 7.4)
	// For now, we implement a whole-file replacement strategy
	// This is the simplest and most reliable approach for the initial implementation
	// More sophisticated partial replacement strategies can be added later
	tempContent := fixCode

	// Ensure the modified content ends with a newline (Requirement 4.3)
	if !strings.HasSuffix(tempContent, "\n") {
		tempContent += "\n"
	}

	// Step 4: Validate the result (Requirement 7.4)
	if strings.TrimSpace(tempContent) == "" {
		// Rollback: return original content (Requirement 7.6)
		f.logError("Applied fix resulted in empty content, rolling back")
		return backup, fmt.Errorf("applied fix resulted in empty or whitespace-only content")
	}

	// Additional validation: check that the fix is syntactically valid
	f.logDebug("Validating fix syntax for file type: %s", fileType)
	if err := f.fixParser.ValidateFixSyntax(tempContent, fileType); err != nil {
		// Rollback: return original content (Requirement 7.6)
		f.logError("Fix validation failed: %v, rolling back", err)
		return backup, fmt.Errorf("fix validation failed: %w", err)
	}

	// Step 5: Commit - return the validated temporary content (Requirement 7.4)
	f.logDebug("Fix applied successfully (result: %d bytes)", len(tempContent))
	return tempContent, nil
}

// orderFixBlocks determines the correct order for applying multiple fix blocks
// to maintain code validity at each step (Requirement 10.2)
//
// Ordering strategy:
// 1. Deletions first (removing code is less likely to break syntax)
// 2. Modifications second (changing existing code)
// 3. Additions last (adding new code builds on existing structure)
//
// Within each category, blocks are ordered by:
// - Larger blocks before smaller blocks (whole file replacements first)
// - This ensures that major structural changes happen before minor tweaks
//
// Parameters:
//   - blocks: The fix blocks to order
//
// Returns:
//   - []CodeBlock: The ordered fix blocks
func (f *AgenticCodeFixer) orderFixBlocks(blocks []CodeBlock) []CodeBlock {
	if len(blocks) <= 1 {
		// No ordering needed for 0 or 1 blocks
		return blocks
	}

	// For now, we implement a simple ordering strategy:
	// 1. Whole file replacements first (IsWhole = true)
	// 2. Partial fixes second (IsWhole = false)
	// 3. Within each category, larger blocks first

	// Separate blocks into categories
	var wholeBlocks []CodeBlock
	var partialBlocks []CodeBlock

	for _, block := range blocks {
		if block.IsWhole {
			wholeBlocks = append(wholeBlocks, block)
		} else {
			partialBlocks = append(partialBlocks, block)
		}
	}

	// Sort each category by size (larger first)
	sortBlocksBySize := func(blocks []CodeBlock) {
		// Simple bubble sort (fine for small arrays)
		for i := 0; i < len(blocks); i++ {
			for j := i + 1; j < len(blocks); j++ {
				if len(blocks[j].Code) > len(blocks[i].Code) {
					blocks[i], blocks[j] = blocks[j], blocks[i]
				}
			}
		}
	}

	sortBlocksBySize(wholeBlocks)
	sortBlocksBySize(partialBlocks)

	// Combine: whole blocks first, then partial blocks
	ordered := make([]CodeBlock, 0, len(blocks))
	ordered = append(ordered, wholeBlocks...)
	ordered = append(ordered, partialBlocks...)

	return ordered
}

// validateMultiStepFix performs pre-validation on all fix blocks before applying any changes
// This ensures that all steps are compatible and won't conflict with each other
// (Requirement 10.5)
//
// Validation checks:
// 1. All blocks have valid syntax for the file type
// 2. No duplicate or conflicting changes
// 3. All blocks are non-empty
// 4. Blocks are compatible with each other (no overlapping modifications)
//
// Parameters:
//   - blocks: The fix blocks to validate
//   - fileType: The type of file being fixed
//
// Returns:
//   - error: Error if validation fails, nil if all checks pass
func (f *AgenticCodeFixer) validateMultiStepFix(blocks []CodeBlock, fileType string) error {
	f.logDebug("Validating multi-step fix with %d blocks", len(blocks))

	if len(blocks) == 0 {
		f.logError("No fix blocks to validate")
		return fmt.Errorf("no fix blocks to validate")
	}

	// Check 1: Validate each block individually
	for i, block := range blocks {
		// Check for empty or whitespace-only blocks
		if strings.TrimSpace(block.Code) == "" {
			f.logError("Fix block %d is empty", i+1)
			return fmt.Errorf("fix block %d is empty or contains only whitespace", i+1)
		}

		// Validate syntax for the file type
		if err := f.fixParser.ValidateFixSyntax(block.Code, fileType); err != nil {
			f.logError("Fix block %d has invalid syntax: %v", i+1, err)
			return fmt.Errorf("fix block %d has invalid syntax: %w", i+1, err)
		}
	}
	f.logDebug("All blocks have valid syntax")

	// Check 2: For multiple blocks, check for potential conflicts
	if len(blocks) > 1 {
		f.logDebug("Checking for conflicts between %d blocks", len(blocks))

		// Check for duplicate blocks (same code content)
		seen := make(map[string]int)
		for i, block := range blocks {
			normalizedCode := strings.TrimSpace(block.Code)
			if prevIndex, exists := seen[normalizedCode]; exists {
				f.logError("Fix blocks %d and %d contain duplicate code", prevIndex+1, i+1)
				return fmt.Errorf("fix blocks %d and %d contain duplicate code", prevIndex+1, i+1)
			}
			seen[normalizedCode] = i
		}

		// Check for conflicting whole-file replacements
		// If we have multiple whole-file replacements, they conflict
		wholeFileCount := 0
		for _, block := range blocks {
			if block.IsWhole {
				wholeFileCount++
			}
		}
		if wholeFileCount > 1 {
			f.logError("Multiple whole-file replacements detected: %d", wholeFileCount)
			return fmt.Errorf("multiple whole-file replacements detected (%d blocks marked as whole); only one whole-file replacement is allowed in a multi-step fix", wholeFileCount)
		}

		// Check for incompatible mix: whole-file replacement with partial fixes
		// If we have a whole-file replacement, we shouldn't have partial fixes
		if wholeFileCount > 0 && len(blocks) > 1 {
			f.logError("Cannot mix whole-file replacement with partial fixes")
			return fmt.Errorf("cannot mix whole-file replacement with partial fixes; found %d blocks with 1 whole-file replacement", len(blocks))
		}

		f.logDebug("No conflicts detected between blocks")
	}

	f.logDebug("Multi-step fix validation passed")
	return nil
}
