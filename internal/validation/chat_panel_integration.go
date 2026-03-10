package validation

import (
	"fmt"
	"strings"
)

// ChatPanelIntegration handles displaying validation status and results in the AI chat interface
type ChatPanelIntegration struct {
	messages []string // Store messages for testing purposes
}

// NewChatPanelIntegration creates a new ChatPanelIntegration instance
func NewChatPanelIntegration() *ChatPanelIntegration {
	return &ChatPanelIntegration{
		messages: make([]string, 0),
	}
}

// ShowValidationStart displays a message indicating validation has started
func (cpi *ChatPanelIntegration) ShowValidationStart(files []string, language Language) {
	var message strings.Builder

	languageName := string(language)
	if language == LanguageGo {
		languageName = "Go"
	} else if language == LanguagePython {
		languageName = "Python"
	}

	message.WriteString(fmt.Sprintf("🔍 Validating %s code in %d file", languageName, len(files)))
	if len(files) != 1 {
		message.WriteString("s")
	}
	message.WriteString("...\n")

	for _, file := range files {
		message.WriteString(fmt.Sprintf("- %s\n", file))
	}

	cpi.addMessage(message.String())
}

// ShowValidationProgress displays validation progress for long-running validations
func (cpi *ChatPanelIntegration) ShowValidationProgress(language Language, duration float64) {
	languageName := string(language)
	if language == LanguageGo {
		languageName = "Go"
	} else if language == LanguagePython {
		languageName = "Python"
	}

	message := fmt.Sprintf("⏳ Compiling %s package... (%.1fs)", languageName, duration)
	cpi.addMessage(message)
}

// ShowValidationSuccess displays a success message with duration
func (cpi *ChatPanelIntegration) ShowValidationSuccess(result ValidationResult) {
	languageName := string(result.Language)
	if result.Language == LanguageGo {
		languageName = "Go"
	} else if result.Language == LanguagePython {
		languageName = "Python"
	}

	durationSeconds := result.Duration.Seconds()
	message := fmt.Sprintf("✅ %s compilation successful (%.1fs)", languageName, durationSeconds)
	cpi.addMessage(message)
}

// ShowValidationFailure displays validation failure with error messages
func (cpi *ChatPanelIntegration) ShowValidationFailure(result ValidationResult) {
	var message strings.Builder

	languageName := string(result.Language)
	if result.Language == LanguageGo {
		languageName = "Go"
	} else if result.Language == LanguagePython {
		languageName = "Python"
	}

	durationSeconds := result.Duration.Seconds()
	message.WriteString(fmt.Sprintf("❌ %s compilation failed (%.1fs)\n\n", languageName, durationSeconds))

	// Display each error with file path, line number, and message
	for _, err := range result.Errors {
		if err.Column > 0 {
			message.WriteString(fmt.Sprintf("%s:%d:%d: %s\n", err.File, err.Line, err.Column, err.Message))
		} else {
			message.WriteString(fmt.Sprintf("%s:%d: %s\n", err.File, err.Line, err.Message))
		}
	}

	// Preserve original error format in output if errors weren't fully parsed
	if result.Output != "" && len(result.Errors) == 0 {
		message.WriteString("\nRaw output:\n")
		message.WriteString(result.Output)
	}

	cpi.addMessage(message.String())
}

// ShowUnsupportedLanguage displays a notification for unsupported files
func (cpi *ChatPanelIntegration) ShowUnsupportedLanguage(files []string, supportedLanguages []Language) {
	var message strings.Builder

	message.WriteString("ℹ️ Skipped validation for unsupported files:\n")
	for _, file := range files {
		message.WriteString(fmt.Sprintf("- %s\n", file))
	}

	message.WriteString("\nSupported languages: ")
	var langNames []string
	for _, lang := range supportedLanguages {
		if lang == LanguageGo {
			langNames = append(langNames, "Go")
		} else if lang == LanguagePython {
			langNames = append(langNames, "Python")
		} else if lang != LanguageUnsupported {
			langNames = append(langNames, string(lang))
		}
	}
	message.WriteString(strings.Join(langNames, ", "))

	cpi.addMessage(message.String())
}

// GetMessages returns all messages (for testing purposes)
func (cpi *ChatPanelIntegration) GetMessages() []string {
	return cpi.messages
}

// ClearMessages clears all messages (for testing purposes)
func (cpi *ChatPanelIntegration) ClearMessages() {
	cpi.messages = make([]string, 0)
}

// addMessage adds a message to the internal list and would display it in the chat panel
func (cpi *ChatPanelIntegration) addMessage(message string) {
	cpi.messages = append(cpi.messages, message)
	// In a real implementation, this would send the message to the chat panel UI
	// For now, we just store it for testing
}

// GetLastMessage returns the most recent message (for testing purposes)
func (cpi *ChatPanelIntegration) GetLastMessage() string {
	if len(cpi.messages) == 0 {
		return ""
	}
	return cpi.messages[len(cpi.messages)-1]
}
