package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// renderHelpDialog renders the help menu popup
func (a *App) renderHelpDialog() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Padding(0, 1).
		Width(60).
		Align(lipgloss.Center)

	titleBar := titleStyle.Render("⌨  Keyboard Shortcuts")

	// Help content with sections
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62"))

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	var helpText string

	// File section
	helpText += sectionStyle.Render("── File ──────────────────────────────────────") + "\n"
	helpText += keyStyle.Render("  Ctrl+O") + descStyle.Render("    Open file") + "\n"
	helpText += keyStyle.Render("  Ctrl+N") + descStyle.Render("    New file") + "\n"
	helpText += keyStyle.Render("  Ctrl+S") + descStyle.Render("    Save file") + "\n"
	helpText += keyStyle.Render("  Ctrl+X") + descStyle.Render("    Close file") + "\n"
	helpText += keyStyle.Render("  Ctrl+R") + descStyle.Render("    Run current script") + "\n"
	helpText += keyStyle.Render("  Ctrl+K") + descStyle.Render("    Kill running process (in terminal mode)") + "\n"
	helpText += keyStyle.Render("  Ctrl+B") + descStyle.Render("    Backup Picker (Restore previous versions)") + "\n"
	helpText += keyStyle.Render("  Ctrl+Q") + descStyle.Render("    Quit") + "\n"
	helpText += "\n"

	// AI section
	helpText += sectionStyle.Render("── AI ────────────────────────────────────────") + "\n"
	helpText += keyStyle.Render("  Ctrl+Y") + descStyle.Render("    List code blocks (Execute/Insert/Return)") + "\n"
	helpText += keyStyle.Render("  Ctrl+P") + descStyle.Render("    Insert selected code into editor") + "\n"
	helpText += keyStyle.Render("  Ctrl+A") + descStyle.Render("    Save full chat history to .ti/ folder") + "\n"
	helpText += keyStyle.Render("  Ctrl+L") + descStyle.Render("    Load saved chat from .ti/ folder") + "\n"
	helpText += keyStyle.Render("  Ctrl+T") + descStyle.Render("    Clear chat / New chat") + "\n"
	helpText += "\n"

	// Navigation section
	helpText += sectionStyle.Render("── Navigation ────────────────────────────────") + "\n"
	helpText += keyStyle.Render("  Tab") + descStyle.Render("       Cycle: Editor → AI Input → AI Response") + "\n"
	helpText += keyStyle.Render("  ↑↓") + descStyle.Render("        Scroll line by line") + "\n"
	helpText += keyStyle.Render("  PgUp/PgDn") + descStyle.Render(" Scroll page") + "\n"
	helpText += keyStyle.Render("  Home/End") + descStyle.Render("  Jump to top/bottom") + "\n"
	helpText += keyStyle.Render("  Esc") + descStyle.Render("       Back") + "\n"
	helpText += keyStyle.Render("  Ctrl+E") + descStyle.Render("    Show Editor Shortcuts") + "\n"
	helpText += "\n"

	// Agent Commands section
	helpText += sectionStyle.Render("── Agent Commands ────────────────────────────") + "\n"
	helpText += keyStyle.Render("  /fix") + descStyle.Render("       Force agentic mode (AI modifies code)") + "\n"
	helpText += keyStyle.Render("  /ask") + descStyle.Render("       Force conversational mode (no changes)") + "\n"
	helpText += keyStyle.Render("  /preview") + descStyle.Render("   Preview changes before applying") + "\n"
	helpText += keyStyle.Render("  /model") + descStyle.Render("     Show current agent and model info") + "\n"
	helpText += keyStyle.Render("  /config") + descStyle.Render("    Edit configuration settings") + "\n"
	helpText += keyStyle.Render("  /help") + descStyle.Render("      Show this help message") + "\n"
	helpText += "\n"

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Align(lipgloss.Center).
		Width(60)

	footer := footerStyle.Render("Press Esc or Ctrl+H to close")

	// Dialog box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(64)

	dialog := dialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, titleBar, "", helpText, footer),
	)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog)
}

// renderEditorHelpDialog renders the editor shortcuts popup
func (a *App) renderEditorHelpDialog() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Padding(0, 1).
		Width(60).
		Align(lipgloss.Center)

	titleBar := titleStyle.Render("✏  Editor Shortcuts")

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62"))

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	var helpText string

	// Delete operations
	helpText += sectionStyle.Render("── Delete ────────────────────────────────────") + "\n"
	helpText += keyStyle.Render("  Alt+D, D") + descStyle.Render("      Delete current line") + "\n"
	helpText += keyStyle.Render("  Alt+L") + descStyle.Render("         Delete current line (single key)") + "\n"
	helpText += keyStyle.Render("  Alt+D, W") + descStyle.Render("      Delete word from cursor") + "\n"
	helpText += keyStyle.Render("  Alt+W") + descStyle.Render("         Delete word from cursor (single key)") + "\n"
	helpText += keyStyle.Render("  Alt+D, 1-9") + descStyle.Render("    Delete N lines from cursor") + "\n"
	helpText += "\n"

	// Undo / Redo
	helpText += sectionStyle.Render("── Undo / Redo ───────────────────────────────") + "\n"
	helpText += keyStyle.Render("  Alt+U") + descStyle.Render("         Undo last change") + "\n"
	helpText += keyStyle.Render("  Alt+R") + descStyle.Render("         Redo last undone change") + "\n"
	helpText += "\n"

	// Navigation
	helpText += sectionStyle.Render("── Navigation ────────────────────────────────") + "\n"
	helpText += keyStyle.Render("  Alt+G") + descStyle.Render("         Go to end of file") + "\n"
	helpText += keyStyle.Render("  Alt+H") + descStyle.Render("         Go to top of file") + "\n"
	helpText += keyStyle.Render("  ↑↓←→") + descStyle.Render("          Move cursor") + "\n"
	helpText += keyStyle.Render("  Home/End") + descStyle.Render("      Jump to line start/end") + "\n"
	helpText += "\n"

	// Editing
	helpText += sectionStyle.Render("── Editing ───────────────────────────────────") + "\n"
	helpText += keyStyle.Render("  Enter") + descStyle.Render("         Insert new line") + "\n"
	helpText += keyStyle.Render("  Backspace") + descStyle.Render("     Delete char before cursor") + "\n"
	helpText += keyStyle.Render("  Delete") + descStyle.Render("        Delete char at cursor") + "\n"
	helpText += "\n"

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Align(lipgloss.Center).
		Width(60)

	footer := footerStyle.Render("Press Esc or Ctrl+E to close")

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(64)

	dialog := dialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, titleBar, "", helpText, footer),
	)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, dialog)
}
