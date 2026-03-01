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
		Align(lipgloss.Center)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62"))

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	// Left column - General shortcuts
	var leftColumn string
	leftColumn += titleStyle.Width(60).Render("⌨  Keyboard Shortcuts") + "\n\n"

	// File section
	leftColumn += sectionStyle.Render("── File ──────────────────────────────────────") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+O") + descStyle.Render("    Open file") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+N") + descStyle.Render("    New file") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+S") + descStyle.Render("    Save file") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+X") + descStyle.Render("    Close file") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+R") + descStyle.Render("    Run current script") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+K") + descStyle.Render("    Kill running process (in terminal mode)") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+B") + descStyle.Render("    Backup Picker (Restore previous versions)") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+Q") + descStyle.Render("    Quit") + "\n"
	leftColumn += "\n"

	// AI section
	leftColumn += sectionStyle.Render("── AI ────────────────────────────────────────") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+Y") + descStyle.Render("    List code blocks (Execute/Insert/Return)") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+P") + descStyle.Render("    Insert selected code into editor") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+A") + descStyle.Render("    Save full chat history to .ti/ folder") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+L") + descStyle.Render("    Load saved chat from .ti/ folder") + "\n"
	leftColumn += keyStyle.Render("  Ctrl+T") + descStyle.Render("    Clear chat / New chat") + "\n"
	leftColumn += "\n"

	// Navigation section
	leftColumn += sectionStyle.Render("── Navigation ────────────────────────────────") + "\n"
	leftColumn += keyStyle.Render("  Tab") + descStyle.Render("       Cycle: Editor → AI Input → AI Response") + "\n"
	leftColumn += keyStyle.Render("  ↑↓") + descStyle.Render("        Scroll line by line") + "\n"
	leftColumn += keyStyle.Render("  PgUp/PgDn") + descStyle.Render(" Scroll page") + "\n"
	leftColumn += keyStyle.Render("  Home/End") + descStyle.Render("  Jump to top/bottom") + "\n"
	leftColumn += keyStyle.Render("  Esc") + descStyle.Render("       Back") + "\n"
	leftColumn += "\n"

	// Agent Commands section
	leftColumn += sectionStyle.Render("── Agent Commands ────────────────────────────") + "\n"
	leftColumn += keyStyle.Render("  /fix") + descStyle.Render("       Force agentic mode (AI modifies code)") + "\n"
	leftColumn += keyStyle.Render("  /ask") + descStyle.Render("       Force conversational mode (no changes)") + "\n"
	leftColumn += keyStyle.Render("  /preview") + descStyle.Render("   Preview changes before applying") + "\n"
	leftColumn += keyStyle.Render("  /model") + descStyle.Render("     Show current agent and model info") + "\n"
	leftColumn += keyStyle.Render("  /config") + descStyle.Render("    Edit configuration settings") + "\n"
	leftColumn += keyStyle.Render("  /help") + descStyle.Render("      Show this help message") + "\n"

	// Right column - Editor shortcuts
	var rightColumn string
	rightColumn += titleStyle.Width(60).Render("✏  Editor Shortcuts") + "\n\n"

	// Delete operations
	rightColumn += sectionStyle.Render("── Delete ────────────────────────────────────") + "\n"
	rightColumn += keyStyle.Render("  Alt+D, D") + descStyle.Render("      Delete current line") + "\n"
	rightColumn += keyStyle.Render("  Alt+L") + descStyle.Render("         Delete current line (single key)") + "\n"
	rightColumn += keyStyle.Render("  Alt+D, W") + descStyle.Render("      Delete word from cursor") + "\n"
	rightColumn += keyStyle.Render("  Alt+W") + descStyle.Render("         Delete word from cursor (single key)") + "\n"
	rightColumn += keyStyle.Render("  Alt+D, 1-9") + descStyle.Render("    Delete N lines from cursor") + "\n"
	rightColumn += "\n"

	// Undo / Redo
	rightColumn += sectionStyle.Render("── Undo / Redo ───────────────────────────────") + "\n"
	rightColumn += keyStyle.Render("  Alt+U") + descStyle.Render("         Undo last change") + "\n"
	rightColumn += keyStyle.Render("  Alt+R") + descStyle.Render("         Redo last undone change") + "\n"
	rightColumn += "\n"

	// Navigation
	rightColumn += sectionStyle.Render("── Navigation ────────────────────────────────") + "\n"
	rightColumn += keyStyle.Render("  Alt+G") + descStyle.Render("         Go to end of file") + "\n"
	rightColumn += keyStyle.Render("  Alt+H") + descStyle.Render("         Go to top of file") + "\n"
	rightColumn += keyStyle.Render("  ↑↓←→") + descStyle.Render("          Move cursor") + "\n"
	rightColumn += keyStyle.Render("  Home/End") + descStyle.Render("      Jump to line start/end") + "\n"
	rightColumn += "\n"

	// Editing
	rightColumn += sectionStyle.Render("── Editing ───────────────────────────────────") + "\n"
	rightColumn += keyStyle.Render("  Enter") + descStyle.Render("         Insert new line") + "\n"
	rightColumn += keyStyle.Render("  Backspace") + descStyle.Render("     Delete char before cursor") + "\n"
	rightColumn += keyStyle.Render("  Delete") + descStyle.Render("        Delete char at cursor") + "\n"

	// Style both columns
	columnStyle := lipgloss.NewStyle().
		Width(64).
		Padding(1, 2)

	leftBox := columnStyle.Render(leftColumn)
	rightBox := columnStyle.Render(rightColumn)

	// Join columns horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Align(lipgloss.Center)

	footer := footerStyle.Render("Press Esc or Ctrl+H to close")

	// Dialog box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)

	dialog := dialogStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, content, "", footer),
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
