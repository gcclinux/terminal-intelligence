package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/terminal-intelligence/internal/agentic"
)

// WindowSizeMsg is exposed for testing purposes
type WindowSizeMsg = tea.WindowSizeMsg

// AgenticFixResultMsg is sent when the agentic code fixer completes processing.
type AgenticFixResultMsg struct {
	Result *agentic.FixResult
}

// SearchCompleteMsg is sent when a workspace search completes.
type SearchCompleteMsg struct {
	SearchTerm   string
	ExactResults []string
	AltResults   []string
}

// ProjectCompleteMsg is sent when a /project or /proceed operation finishes.
// It carries the ChangeReport so the Update handler can open modified files.
type ProjectCompleteMsg struct {
	Report             *agentic.ChangeReport
	Formatted          string
	LastPreviewRequest string // non-empty when this was a preview run
}

// ProjectFileOpenMsg drives sequential file loading into the editor panel.
// Each message opens Paths[0] and queues a new message for Paths[1:].
type ProjectFileOpenMsg struct {
	Paths []string // remaining absolute paths to open; open [0], queue rest
}
