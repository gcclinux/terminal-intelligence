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
