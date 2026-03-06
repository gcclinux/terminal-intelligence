package docgen

import (
	"fmt"
	"strings"
)

// FeedbackManager provides progress updates and completion summaries for documentation generation
type FeedbackManager struct {
	chatPane ChatPaneInterface
}

// ChatPaneInterface defines the interface for displaying messages in the chat pane
// This allows for testing without requiring a full AIChatPane instance
type ChatPaneInterface interface {
	DisplayNotification(notification string)
	OpenFileInEditor(filePath string) error
}

// NewFeedbackManager creates a new FeedbackManager with the given chat pane
func NewFeedbackManager(chatPane ChatPaneInterface) *FeedbackManager {
	return &FeedbackManager{
		chatPane: chatPane,
	}
}

// NotifyStart displays a message indicating documentation generation has begun
func (f *FeedbackManager) NotifyStart(docTypes []DocumentationType) {
	if f.chatPane == nil {
		return
	}

	typeNames := make([]string, len(docTypes))
	for i, dt := range docTypes {
		typeNames[i] = documentationTypeName(dt)
	}

	message := fmt.Sprintf("📝 Starting documentation generation: %s", strings.Join(typeNames, ", "))
	f.chatPane.DisplayNotification(message)
}

// NotifyProgress displays a progress update for a specific stage
func (f *FeedbackManager) NotifyProgress(stage string, details string) {
	if f.chatPane == nil {
		return
	}

	message := fmt.Sprintf("⏳ %s", stage)
	if details != "" {
		message += fmt.Sprintf(": %s", details)
	}
	f.chatPane.DisplayNotification(message)
}

// NotifyComplete displays a completion summary with file paths and generated content
func (f *FeedbackManager) NotifyComplete(results []*WriteResult, scopeFilters []string) {
	if f.chatPane == nil {
		return
	}

	var message strings.Builder
	message.WriteString("✅ Documentation generation complete!\n\n")

	// List generated files
	message.WriteString("Generated files:\n")
	for _, result := range results {
		if result.Written {
			status := "created"
			if result.Existed {
				status = "updated"
			}
			message.WriteString(fmt.Sprintf("  • %s (%s)\n", result.Path, status))
		}
	}

	// Report scope if filters were applied
	if len(scopeFilters) > 0 {
		message.WriteString(fmt.Sprintf("\nScope: %s\n", strings.Join(scopeFilters, ", ")))
	}

	f.chatPane.DisplayNotification(message.String())
}

// NotifyError displays an error message
func (f *FeedbackManager) NotifyError(err error) {
	if f.chatPane == nil {
		return
	}

	message := fmt.Sprintf("❌ Documentation generation failed: %v", err)
	f.chatPane.DisplayNotification(message)
}

// NotifyFileConflict displays a message about file conflicts
func (f *FeedbackManager) NotifyFileConflict(conflicts []*WriteResult) {
	if f.chatPane == nil {
		return
	}

	var message strings.Builder
	message.WriteString("⚠️  File conflicts detected:\n\n")

	for _, conflict := range conflicts {
		message.WriteString(fmt.Sprintf("  • %s already exists\n", conflict.Filename))
	}

	message.WriteString("\nUse overwrite option to replace existing files.")
	f.chatPane.DisplayNotification(message.String())
}

// NotifyFileCreated displays a message about a file being created and opens it in the editor
func (f *FeedbackManager) NotifyFileCreated(result *WriteResult) {
	if f.chatPane == nil {
		return
	}

	status := "created"
	if result.Existed {
		status = "updated"
	}

	message := fmt.Sprintf("📄 Document %s: %s", status, result.Filename)
	f.chatPane.DisplayNotification(message)

	// Open the file in the editor
	if err := f.chatPane.OpenFileInEditor(result.Path); err != nil {
		f.chatPane.DisplayNotification(fmt.Sprintf("⚠️  Could not open file in editor: %v", err))
	}
}

// documentationTypeName returns a human-readable name for a documentation type
func documentationTypeName(docType DocumentationType) string {
	switch docType {
	case DocTypeUserManual:
		return "User Manual"
	case DocTypeInstallation:
		return "Installation Guide"
	case DocTypeAPI:
		return "API Reference"
	case DocTypeTutorial:
		return "Tutorial"
	case DocTypeGeneral:
		return "General Documentation"
	default:
		return "Documentation"
	}
}
