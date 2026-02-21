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
	helpText += keyStyle.Render("  Ctrl+B") + descStyle.Render("    Backup Picker (Restore previous versions)") + "\n"
	helpText += keyStyle.Render("  Ctrl+Q") + descStyle.Render("    Quit") + "\n"
	helpText += "\n"

	// AI section
	helpText += sectionStyle.Render("── AI ────────────────────────────────────────") + "\n"
	helpText += keyStyle.Render("  Ctrl+Y") + descStyle.Render("    List code blocks (Execute/Insert/Return)") + "\n"
	helpText += keyStyle.Render("  Ctrl+P") + descStyle.Render("    Insert selected code into editor") + "\n"
	helpText += keyStyle.Render("  Ctrl+A") + descStyle.Render("    Insert full AI response into file") + "\n"
	helpText += keyStyle.Render("  Ctrl+T") + descStyle.Render("    Clear chat / New chat") + "\n"
	helpText += "\n"

	// Navigation section
	helpText += sectionStyle.Render("── Navigation ────────────────────────────────") + "\n"
	helpText += keyStyle.Render("  Tab") + descStyle.Render("       Cycle: Editor → AI Input → AI Response") + "\n"
	helpText += keyStyle.Render("  ↑↓") + descStyle.Render("        Scroll line by line") + "\n"
	helpText += keyStyle.Render("  PgUp/PgDn") + descStyle.Render(" Scroll page") + "\n"
	helpText += keyStyle.Render("  Home/End") + descStyle.Render("  Jump to top/bottom") + "\n"
	helpText += keyStyle.Render("  Esc") + descStyle.Render("       Back") + "\n"
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
