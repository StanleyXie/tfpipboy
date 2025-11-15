package tui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/StanleyXie/tfpipboy/pkg/auth"
	"github.com/StanleyXie/tfpipboy/pkg/terraform"
	tea "github.com/charmbracelet/bubbletea"
)

// authStatusMsg contains authentication status for all providers
type authStatusMsg map[string]*auth.Status

// checkAuthCmd performs authentication checks in background
func checkAuthCmd(manager *auth.Manager) tea.Cmd {
	return func() tea.Msg {
		return authStatusMsg(manager.CheckAll())
	}
}

// tfContextMsg contains Terraform context information
type tfContextMsg *terraform.Context

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isInteractiveCommand checks if a command requires interactive terminal
func isInteractiveCommand(command string) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	// List of interactive commands
	interactivePatterns := []string{
		// Auth commands
		"az login",
		"az account",
		"gcloud auth login",
		"gcloud auth",
		"aws configure",
		"aws sso login",
		"gh auth login",
		// Terraform commands requiring user input
		"terraform apply",
		"terraform destroy",
		"terraform console",
		// Editors
		"vim",
		"vi",
		"nano",
		"emacs",
		// Interactive tools
		"top",
		"htop",
		"less",
		"more",
		"ssh",
		"docker exec -it",
		"kubectl exec -it",
	}

	// Check command start
	cmdStart := strings.Join(parts[:min(3, len(parts))], " ")
	for _, pattern := range interactivePatterns {
		if strings.HasPrefix(cmdStart, pattern) {
			return true
		}
	}

	// Check first word
	for _, pattern := range interactivePatterns {
		if parts[0] == pattern {
			return true
		}
	}

	return false
}

// runInteractiveCommand suspends TUI and runs command with full terminal access
func runInteractiveCommand(command string, tfManager *terraform.Manager) tea.Cmd {
	// Use shell for execution
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell, "-c", command)
	cmd.Dir, _ = os.Getwd()
	cmd.Env = os.Environ()

	// Use tea.ExecProcess to suspend TUI and give full terminal access
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		output := fmt.Sprintf("[Interactive command completed]")
		if err != nil {
			output = fmt.Sprintf("[Interactive command completed with error: %v]", err)
		}

		return commandResultMsg{
			command:        command,
			output:         output,
			wasInteractive: true,
			err:            err,
		}
	})
}

// checkTfContextCmd performs Terraform context check in background
func checkTfContextCmd(manager *terraform.Manager) tea.Cmd {
	return func() tea.Msg {
		ctx, _ := manager.GetContext()
		return tfContextMsg(ctx)
	}
}

// commandResultMsg contains command execution result
type commandResultMsg struct {
	command        string
	output         string
	err            error
	changedDir     bool
	wasInteractive bool
}

// streamingOutputMsg contains a single line of output from a running command
type streamingOutputMsg struct {
	line string
}

// commandFinishedMsg signals that a command has completed
type commandFinishedMsg struct {
	command string
	err     error
}

// executeCommandCmd executes a command with streaming output
func executeCommandCmd(command string, tfManager *terraform.Manager) tea.Cmd {
	return func() tea.Msg {
		// Parse command into parts
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return commandResultMsg{
				command: command,
				output:  "",
				err:     fmt.Errorf("empty command"),
			}
		}

		// Handle built-in commands
		switch parts[0] {
		case "cd":
			// Handle cd command specially
			var dir string
			if len(parts) > 1 {
				dir = parts[1]
			} else {
				// cd with no args goes to home
				dir = os.Getenv("HOME")
			}

			// Expand ~
			if strings.HasPrefix(dir, "~") {
				dir = strings.Replace(dir, "~", os.Getenv("HOME"), 1)
			}

			// Change directory
			err := os.Chdir(dir)
			if err != nil {
				return commandResultMsg{
					command: command,
					output:  "",
					err:     fmt.Errorf("cd: %v", err),
				}
			}

			// Update terraform manager path
			newPath, _ := os.Getwd()
			tfManager.SetPath(newPath)

			return commandResultMsg{
				command:    command,
				output:     fmt.Sprintf("Changed directory to: %s", newPath),
				changedDir: true,
			}
		}

		// Execute external command with streaming output
		// Call the streaming function directly (not wrap it in another tea.Cmd)
		return executeStreamingCommand(command, tfManager)
	}
}

// executeStreamingCommand starts command and returns initial tea.Cmd for streaming
func executeStreamingCommand(command string, tfManager *terraform.Manager) tea.Msg {
	// Use shell for execution
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell, "-c", command)
	cmd.Dir, _ = os.Getwd()
	cmd.Env = os.Environ()

	// Create pipes
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return commandFinishedMsg{command: command, err: err}
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return commandFinishedMsg{command: command, err: err}
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return commandFinishedMsg{command: command, err: err}
	}

	// Create channel for streaming output
	outputChan := make(chan string, 100)

	// Read from both streams concurrently and merge into outputChan
	go func() {
		done := make(chan bool, 2)

		// Read stdout
		go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			buf := make([]byte, 0, 64*1024)
			scanner.Buffer(buf, 1024*1024)
			for scanner.Scan() {
				outputChan <- scanner.Text()
			}
			done <- true
		}()

		// Read stderr
		go func() {
			scanner := bufio.NewScanner(stderrPipe)
			buf := make([]byte, 0, 64*1024)
			scanner.Buffer(buf, 1024*1024)
			for scanner.Scan() {
				outputChan <- scanner.Text()
			}
			done <- true
		}()

		// Wait for both readers, then close channel and send finish message
		<-done
		<-done
		close(outputChan)

		// Wait for command to finish
		cmdErr := cmd.Wait()

		// Send a special signal to indicate completion
		// We'll use empty string as end marker
		if cmdErr != nil {
			outputChan <- fmt.Sprintf("[COMMAND_FINISHED_WITH_ERROR:%v]", cmdErr)
		} else {
			outputChan <- "[COMMAND_FINISHED_SUCCESS]"
		}
	}()

	// Return the first line (or wait for it)
	return waitForNextLine(outputChan, command)
}

// waitForNextLine creates a tea.Cmd that waits for the next line from channel
func waitForNextLine(outputChan chan string, command string) tea.Msg {
	line, ok := <-outputChan
	if !ok {
		// Channel closed, command finished
		return commandFinishedMsg{command: command, err: nil}
	}

	// Check for special finish markers
	if strings.HasPrefix(line, "[COMMAND_FINISHED_") {
		if strings.Contains(line, "ERROR") {
			errMsg := strings.TrimPrefix(line, "[COMMAND_FINISHED_WITH_ERROR:")
			errMsg = strings.TrimSuffix(errMsg, "]")
			return commandFinishedMsg{command: command, err: fmt.Errorf("%s", errMsg)}
		}
		return commandFinishedMsg{command: command, err: nil}
	}

	// Regular output line
	return streamingOutputMsg{line: line}
}

// waitForNextLineCmd wraps waitForNextLine as a tea.Cmd
func waitForNextLineCmd(outputChan chan string, command string) tea.Cmd {
	return func() tea.Msg {
		return waitForNextLine(outputChan, command)
	}
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+q":
			return m, tea.Quit

		case "enter":
			// Execute command
			command := m.textInput.Value()
			if command != "" {
				// Add command to history
				m.commandHistory = append(m.commandHistory, command)
				m.historyIndex = len(m.commandHistory)

				// Add command to output
				m.output = append(m.output, fmt.Sprintf("$ %s", command))

				// Clear input
				m.textInput.Reset()

				// Check if this is an interactive command
				m.viewport.SetContent(strings.Join(m.output, "\n"))
				m.viewport.GotoBottom()

				if isInteractiveCommand(command) { // Run interactive command by suspending TUI
					return m, runInteractiveCommand(command, m.tfManager)
				}

				// Execute regular command in background
				return m, executeCommandCmd(command, m.tfManager)
			}
			return m, nil

		case "up", "ctrl+p":
			// Navigate up in command history
			if len(m.commandHistory) > 0 && m.historyIndex > 0 {
				m.historyIndex--
				m.textInput.SetValue(m.commandHistory[m.historyIndex])
				m.textInput.CursorEnd()
			}
			return m, nil

		case "down", "ctrl+n":
			// Navigate down in command history
			if len(m.commandHistory) > 0 && m.historyIndex < len(m.commandHistory)-1 {
				m.historyIndex++
				m.textInput.SetValue(m.commandHistory[m.historyIndex])
				m.textInput.CursorEnd()
			} else if m.historyIndex == len(m.commandHistory)-1 {
				// At the end, clear input
				m.historyIndex = len(m.commandHistory)
				m.textInput.Reset()
			}
			return m, nil

		case "ctrl+l":
			// Clear screen
			m.output = []string{}
			m.viewport.SetContent("")
			return m, nil

		case "tab":
			// Tab completion
			currentValue := m.textInput.Value()
			if currentValue != "" {
				completed := attemptCompletion(currentValue)
				if completed != currentValue {
					m.textInput.SetValue(completed)
					m.textInput.CursorEnd()
				}
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = m.width - 3 // Account for prompt "> "

		// Calculate viewport height (total - title - status - input - help - margins)
		viewportHeight := m.height - 7
		if viewportHeight < 5 {
			viewportHeight = 5
		}

		m.viewport.Width = m.width
		m.viewport.Height = viewportHeight
		m.ready = true

	case tickMsg:
		// Refresh status - trigger auth and TF context check
		m.lastUpdate = time.Time(msg)
		return m, tea.Batch(
			tickCmd(),
			checkAuthCmd(m.authManager),
			checkTfContextCmd(m.tfManager),
		)

	case authStatusMsg:
		// Update auth status from background check
		m.authStatus = msg
		return m, nil

	case tfContextMsg:
		// Update Terraform context from background check
		m.tfContext = msg
		return m, nil

	case commandResultMsg:
		// Debug: Add marker to show we received the message

		// Add output lines first (preserve all output including whitespace)
		if msg.output != "" {
			// Don't trim - preserve exact formatting including leading/trailing newlines
			lines := strings.Split(msg.output, "\n")
			m.output = append(m.output, lines...)
		}

		// Handle command result errors
		if msg.err != nil {
			errorMsg := fmt.Sprintf("\n[ERROR] Exit code: %v", msg.err)

			// Provide helpful hint for common errors
			if strings.Contains(msg.err.Error(), "executable file not found") {
				parts := strings.Fields(msg.command)
				if len(parts) > 0 {
					cmdName := parts[0]
					errorMsg += fmt.Sprintf("\n[HINT] '%s' command not found in PATH. Install it first:", cmdName)

					// Provide installation hints for common tools
					switch cmdName {
					case "terraform":
						errorMsg += "\n  brew install terraform  (or download from https://terraform.io)"
					case "aws":
						errorMsg += "\n  brew install awscli"
					case "az":
						errorMsg += "\n  brew install azure-cli"
					case "gcloud":
						errorMsg += "\n  brew install google-cloud-sdk"
					case "kubectl":
						errorMsg += "\n  brew install kubectl"
					default:
						errorMsg += fmt.Sprintf("\n  brew install %s  (or check if it's installed)", cmdName)
					}
				}
			}

			m.output = append(m.output, errorMsg)
			m.output = append(m.output, "") // Add blank line after error
		}

		// Update viewport content with all output
		m.viewport.SetContent(strings.Join(m.output, "\n"))

		// Auto-scroll to bottom to show latest output
		m.viewport.GotoBottom()

		// Refresh context after command execution
		return m, tea.Batch(
			checkTfContextCmd(m.tfManager),
			checkAuthCmd(m.authManager),
		)

	case error:
		m.err = msg
		return m, nil
	}

	// Update viewport for scrolling (pgup/pgdn) but not arrow keys (command history)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		keyStr := keyMsg.String()
		switch keyStr {
		case "up", "down", "ctrl+p", "ctrl+n":
			// Don't pass arrow keys to viewport - they're for command history
			// These are already handled in the switch statement above
		default:
			// Pass all other keys (including pgup/pgdn) to viewport
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	} else {
		// Non-key messages can go to viewport
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update textinput component for all other messages
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// attemptCompletion attempts to complete the current input
func attemptCompletion(input string) string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return input
	}

	// Get the last part to complete
	lastPart := parts[len(parts)-1]

	// Try file/directory completion
	matches, _ := filepath.Glob(lastPart + "*")
	if len(matches) == 1 {
		// Single match - complete it
		parts[len(parts)-1] = matches[0]
		return strings.Join(parts, " ")
	} else if len(matches) > 1 {
		// Multiple matches - find common prefix
		prefix := longestCommonPrefix(matches)
		if len(prefix) > len(lastPart) {
			parts[len(parts)-1] = prefix
			return strings.Join(parts, " ")
		}
	}

	// Try command completion for first word
	if len(parts) == 1 {
		commonCommands := []string{
			"terraform", "cd", "ls", "pwd", "cat", "echo",
			"git", "make", "go", "docker", "kubectl",
		}

		var cmdMatches []string
		for _, cmd := range commonCommands {
			if strings.HasPrefix(cmd, lastPart) {
				cmdMatches = append(cmdMatches, cmd)
			}
		}

		if len(cmdMatches) == 1 {
			return cmdMatches[0]
		} else if len(cmdMatches) > 1 {
			prefix := longestCommonPrefix(cmdMatches)
			if len(prefix) > len(lastPart) {
				return prefix
			}
		}
	}

	return input
}

// longestCommonPrefix finds the longest common prefix of strings
func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}
