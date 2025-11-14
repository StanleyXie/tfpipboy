package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("63")).
			Bold(true).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("63")).
			Bold(true).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	outputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Bold(true)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)
)

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Title bar
	title := titleStyle.Width(m.width).Render("tf-pipboy v0.1.0")

	// Status bar with authentication and Terraform context
	statusText := m.renderStatus()
	status := statusBarStyle.Width(m.width).Render(statusText)

	// Output pane - use viewport for scrollable output
	output := m.viewport.View()

	// Input using textinput component
	inputPrompt := m.textInput.View()

	// Help text with all supported keybindings
	help := helpStyle.Render("Keys: Ctrl+Q=quit | ↑↓=history | PgUp/PgDn/MouseWheel=scroll | Ctrl+A/E=start/end | Ctrl+U/K=clear | Tab=complete | Ctrl+L=clear | Copy: Shift+Select")

	// Combine all sections
	return fmt.Sprintf("%s\n%s\n\n%s\n\n%s\n%s",
		title,
		status,
		output,
		inputPrompt,
		help,
	)
}

// renderStatus renders complete status line with auth and Terraform context
func (m Model) renderStatus() string {
	var sections []string

	// Terraform context section
	tfStatus := m.renderTfContext()
	if tfStatus != "" {
		sections = append(sections, tfStatus)
	}

	// Authentication section
	authStatus := m.renderAuthStatus()
	if authStatus != "" {
		sections = append(sections, authStatus)
	}

	if len(sections) == 0 {
		return "Initializing..."
	}

	return strings.Join(sections, " | ")
}

// renderTfContext renders Terraform context information
func (m Model) renderTfContext() string {
	if m.tfContext == nil {
		return "TF: Checking..."
	}

	// If not in a Terraform directory
	if m.tfContext.Workspace == "" && m.tfContext.Backend == "" && m.tfContext.Module == "" {
		return "TF: Not detected"
	}

	var parts []string

	// Workspace
	if m.tfContext.Workspace != "" {
		parts = append(parts, fmt.Sprintf("WS:%s", m.tfContext.Workspace))
	}

	// Backend
	if m.tfContext.Backend != "" {
		parts = append(parts, fmt.Sprintf("Backend:%s", m.tfContext.Backend))
	}

	// Module
	if m.tfContext.Module != "" {
		parts = append(parts, fmt.Sprintf("Module:%s", m.tfContext.Module))
	}

	// Variables count
	if len(m.tfContext.Variables) > 0 {
		parts = append(parts, fmt.Sprintf("Vars:%d", len(m.tfContext.Variables)))
	}

	if len(parts) == 0 {
		return "TF: Unknown"
	}

	return "TF[" + strings.Join(parts, " ") + "]"
}

// renderAuthStatus renders authentication status for all providers
func (m Model) renderAuthStatus() string {
	if len(m.authStatus) == 0 {
		return "Auth: Checking..."
	}

	var parts []string

	// Azure
	if azure, ok := m.authStatus["azure"]; ok {
		if azure.Authenticated {
			parts = append(parts, fmt.Sprintf("✓Az:%s", azure.User))
		} else {
			parts = append(parts, "✗Az")
		}
	}

	// GitHub
	if github, ok := m.authStatus["github"]; ok {
		if github.Authenticated {
			parts = append(parts, fmt.Sprintf("✓GH:%s", github.User))
		} else {
			parts = append(parts, "✗GH")
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "Auth[" + strings.Join(parts, " ") + "]"
}
