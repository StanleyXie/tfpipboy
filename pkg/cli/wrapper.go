package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/StanleyXie/tfpipboy/pkg/auth"
	"github.com/StanleyXie/tfpipboy/pkg/terraform"
	"github.com/StanleyXie/tfpipboy/pkg/version"
	"github.com/ergochat/readline"
)

// ANSI color codes and control sequences
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
	colorBold    = "\033[1m"

	// ANSI cursor control
	saveCursor    = "\033[s"    // Save cursor position
	restoreCursor = "\033[u"    // Restore cursor position
	clearLine     = "\033[2K"   // Clear entire line
	hideCursor    = "\033[?25l" // Hide cursor
	showCursor    = "\033[?25h" // Show cursor

	// Security and performance settings
	maxHistoryLines      = 10000            // Maximum lines in history file to prevent unbounded growth
	historyFileMode      = 0600             // User read/write only for security
	defaultCommandTimeout = 30 * time.Minute // Default timeout for command execution
)

// Wrapper is the main CLI wrapper structure
// SECURITY NOTE: This wrapper executes user commands directly through the shell.
// By design, it provides full shell access to the authenticated user.
// - Commands are executed with the user's own permissions and credentials
// - All environment variables are inherited by child processes
// - Command history is stored locally (see history file security below)
// This is intended for interactive use by trusted operators, not for programmatic/automated use.
type Wrapper struct {
	authManager    *auth.Manager
	tfManager      *terraform.Manager
	commandHistory []string
	historyIndex   int
	historyFile    string
	termHeight     int
	termWidth      int
	commandTimeout time.Duration
}

// NewWrapper creates a new CLI wrapper instance
func NewWrapper() *Wrapper {
	return &Wrapper{
		authManager:    auth.NewManager(),
		tfManager:      terraform.NewManager(),
		commandHistory: []string{},
		historyIndex:   0,
		historyFile:    os.ExpandEnv("$HOME/.tfpipboy_history"),
		commandTimeout: defaultCommandTimeout,
	}
}

// setupHistoryFile ensures history file exists with proper permissions and size limits
// SECURITY: History file set to 0600 (user read/write only) to protect potentially sensitive commands
func (w *Wrapper) setupHistoryFile() error {
	// Ensure history file exists
	if _, err := os.Stat(w.historyFile); os.IsNotExist(err) {
		// Create empty file with secure permissions
		f, err := os.OpenFile(w.historyFile, os.O_CREATE|os.O_WRONLY, historyFileMode)
		if err != nil {
			return fmt.Errorf("failed to create history file: %w", err)
		}
		f.Close()
	} else {
		// File exists, ensure permissions are correct
		if err := os.Chmod(w.historyFile, historyFileMode); err != nil {
			return fmt.Errorf("failed to set history file permissions: %w", err)
		}
	}

	// Trim history file if too large
	return w.trimHistoryFile()
}

// trimHistoryFile limits history file to maxHistoryLines to prevent unbounded growth
func (w *Wrapper) trimHistoryFile() error {
	file, err := os.Open(w.historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file to trim
		}
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	// Read all lines
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read history file: %w", err)
	}

	// Trim if necessary
	if len(lines) > maxHistoryLines {
		lines = lines[len(lines)-maxHistoryLines:]

		// Write back trimmed history
		tmpFile := w.historyFile + ".tmp"
		f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, historyFileMode)
		if err != nil {
			return fmt.Errorf("failed to create temp history file: %w", err)
		}

		writer := bufio.NewWriter(f)
		for _, line := range lines {
			fmt.Fprintln(writer, line)
		}

		if err := writer.Flush(); err != nil {
			f.Close()
			return fmt.Errorf("failed to write history file: %w", err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("failed to close history file: %w", err)
		}

		// Atomic replace
		if err := os.Rename(tmpFile, w.historyFile); err != nil {
			return fmt.Errorf("failed to replace history file: %w", err)
		}
	}

	return nil
}

// Run starts the CLI wrapper main loop
func (w *Wrapper) Run() error {
	// Setup history file with security settings
	if err := w.setupHistoryFile(); err != nil {
		fmt.Printf("Warning: Failed to setup history file: %v\n", err)
		// Continue without history file
		w.historyFile = ""
	}

	// Print initial header
	w.printWelcome()
	w.printStatus()

	// Create readline instance with configuration
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          w.buildPrompt(),
		HistoryFile:     w.historyFile,
		AutoComplete:    w.buildCompleter(),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return fmt.Errorf("failed to create readline: %w", err)
	}
	defer rl.Close()

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		rl.Close()
		fmt.Println("\nGoodbye!")
		os.Exit(0)
	}()

	// Main command loop
	for {
		// Update prompt
		rl.SetPrompt(w.buildPrompt())

		// Read command with readline (supports backspace, arrows, paste, etc.)
		input, err := rl.Readline()
		if err != nil {
			// Check if it's EOF or interrupt
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("\nGoodbye!")
				break
			}

			// Other error - print it and exit
			fmt.Printf("\nError reading input: %v\n", err)
			break
		}

		command := strings.TrimSpace(input)

		// Skip empty commands
		if command == "" {
			continue
		}

		// Handle exit - support exit, quit
		if command == "exit" || command == "quit" {
			fmt.Println("\nGoodbye!")
			break
		}

		// Add to our history
		w.addToHistory(command)

		// Execute command
		w.executeCommand(command)

		// Clear auth cache to force fresh check after command execution
		// This ensures commands like 'az logout' or 'gh auth logout' are reflected immediately
		w.authManager.ClearCache()

		// Refresh status after command (will do fresh auth checks now)
		w.updateStatus()
	}

	return nil
}

// printWelcome prints the welcome header
func (w *Wrapper) printWelcome() {
	clearScreen()
	fmt.Printf("%s╭────────────────────────────────────────────────────────────────────╮%s\n", colorCyan, colorReset)
	fmt.Printf("%s│%s                        %stfpipboy v%s%s                            %s│%s\n",
		colorCyan, colorReset, colorBold+colorYellow, version.Version, colorReset, colorCyan, colorReset)
	fmt.Printf("%s│%s                    %sCLI Wrapper for Terraform%s                       %s│%s\n",
		colorCyan, colorReset, colorWhite, colorReset, colorCyan, colorReset)
	fmt.Printf("%s╰────────────────────────────────────────────────────────────────────╯%s\n", colorCyan, colorReset)
	fmt.Println()
}

// printStatus prints the initial status (now using bottom bar)
func (w *Wrapper) printStatus() {
	// Render at bottom of terminal
	w.renderBottomStatusBar()
	fmt.Println() // Add some spacing after welcome
}

// buildStatusLine constructs the status line string
func (w *Wrapper) buildStatusLine(tfContext *terraform.Context, authStatus map[string]*auth.Status) string {
	var lines []string

	// Terraform context
	if tfContext != nil {
		if tfContext.Workspace != "" {
			lines = append(lines, fmt.Sprintf("%s🏢 Workspace:%s %s%s%s",
				colorCyan, colorReset, colorBold, tfContext.Workspace, colorReset))
		}
		if tfContext.Backend != "" {
			lines = append(lines, fmt.Sprintf("%s💾 Backend:%s %s%s%s",
				colorBlue, colorReset, colorBold, tfContext.Backend, colorReset))
		}
		if tfContext.Module != "" {
			lines = append(lines, fmt.Sprintf("%s📦 Module:%s %s%s%s",
				colorMagenta, colorReset, colorBold, tfContext.Module, colorReset))
		}
	} else {
		lines = append(lines, fmt.Sprintf("%s⚠️  No TF context%s", colorYellow, colorReset))
	}

	// Auth status - each provider on separate line (show complete info)
	if az, ok := authStatus["azure"]; ok && az.Authenticated {
		lines = append(lines, fmt.Sprintf("%s🔐 Azure:%s %s%s%s",
			colorGreen, colorReset, colorBold, az.User, colorReset))
	}
	if gh, ok := authStatus["github"]; ok && gh.Authenticated {
		lines = append(lines, fmt.Sprintf("%s🔐 GitHub:%s %s%s%s",
			colorGreen, colorReset, colorBold, gh.User, colorReset))
	}

	// Format as multiple lines (one per status type)
	return strings.Join(lines, "\n")
}

// buildPrompt builds the command prompt string
func (w *Wrapper) buildPrompt() string {
	// Get current directory
	pwd, _ := os.Getwd()
	home := os.Getenv("HOME")

	// Replace home with ~
	if strings.HasPrefix(pwd, home) {
		pwd = "~" + strings.TrimPrefix(pwd, home)
	}

	// Return prompt with console icon and color
	// Using 💻 (laptop/console) icon to represent terminal/console
	return fmt.Sprintf("%s💻 %s%s%s >%s ", colorGreen, colorBold, pwd, colorReset, colorReset)
}

// executeCommand executes a user command
func (w *Wrapper) executeCommand(command string) {
	// Handle built-in commands
	if w.handleBuiltinCommand(command) {
		return
	}

	// Execute external command
	w.runExternalCommand(command)
}

// handleBuiltinCommand handles built-in commands like cd
func (w *Wrapper) handleBuiltinCommand(command string) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	switch parts[0] {
	case "cd":
		w.handleCD(parts)
		return true
	case "help":
		w.printHelp()
		return true
	default:
		return false
	}
}

// handleCD handles directory change with path validation
func (w *Wrapper) handleCD(parts []string) {
	var dir string

	if len(parts) > 1 {
		dir = parts[1]
	} else {
		dir = os.Getenv("HOME")
	}

	// Expand ~
	if strings.HasPrefix(dir, "~") {
		home := os.Getenv("HOME")
		if home == "" {
			fmt.Printf("\033[31mcd: HOME environment variable not set\033[0m\n")
			return
		}
		dir = strings.Replace(dir, "~", home, 1)
	}

	// Clean and validate path
	dir = filepath.Clean(dir)

	// Convert to absolute path if not already
	if !filepath.IsAbs(dir) {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("\033[31mcd: failed to get current directory: %v\033[0m\n", err)
			return
		}
		dir = filepath.Join(cwd, dir)
	}

	// Evaluate symlinks for canonical path
	absDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		// If symlink evaluation fails, try without it (might be valid path)
		absDir = dir
	}

	// Change directory
	if err := os.Chdir(absDir); err != nil {
		fmt.Printf("\033[31mcd: %v\033[0m\n", err)
		return
	}

	// Update terraform manager path with the actual directory we changed to
	newPath, err := os.Getwd()
	if err != nil {
		fmt.Printf("\033[33mWarning: failed to update terraform context: %v\033[0m\n", err)
		return
	}
	w.tfManager.SetPath(newPath)
}

// runExternalCommand runs an external command with full terminal access
// SECURITY: This function executes user-provided commands through the shell.
// - Commands run with the user's own permissions (not elevated)
// - Full environment is inherited (including credentials in env vars)
// - Timeout applied to prevent indefinite hangs
// - Interactive commands (terraform apply, etc.) work correctly
func (w *Wrapper) runExternalCommand(command string) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	// Create context with timeout to prevent indefinite hangs
	ctx, cancel := context.WithTimeout(context.Background(), w.commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, shell, "-c", command)
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("\n\033[31m[Error: failed to get working directory: %v]\033[0m\n", err)
		return
	}
	cmd.Dir = cwd
	cmd.Env = os.Environ()

	// Connect directly to terminal for real-time output and interactivity
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run and wait
	if err := cmd.Run(); err != nil {
		// Check if context deadline exceeded (timeout)
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Printf("\n\033[31m[Error: Command timed out after %v]\033[0m\n", w.commandTimeout)
		} else {
			// Regular command error
			fmt.Printf("\n\033[31m[Error: %v]\033[0m\n", err)
		}
	}

	// Add a newline after command output to separate from status bar
	fmt.Println()
}

// addToHistory adds a command to history
func (w *Wrapper) addToHistory(command string) {
	// Don't add duplicates if same as last command
	if len(w.commandHistory) > 0 && w.commandHistory[len(w.commandHistory)-1] == command {
		w.historyIndex = len(w.commandHistory)
		return
	}

	w.commandHistory = append(w.commandHistory, command)
	w.historyIndex = len(w.commandHistory)
}

// updateStatus refreshes the status bar
// Status is updated synchronously after each command to avoid display conflicts with readline
func (w *Wrapper) updateStatus() {
	// Render at bottom of terminal
	w.renderBottomStatusBar()
}

// printHelp prints help information
func (w *Wrapper) printHelp() {
	fmt.Println("\n╭─── tfpipboy Help ─────────────────────────────────────────────╮")
	fmt.Println("│                                                                │")
	fmt.Println("│  Built-in Commands:                                            │")
	fmt.Println("│    cd <dir>     Change directory                               │")
	fmt.Println("│    help         Show this help                                 │")
	fmt.Println("│    exit/quit    Exit tfpipboy                                 │")
	fmt.Println("│                                                                │")
	fmt.Println("│  All other commands are passed directly to your shell          │")
	fmt.Println("│  with full terminal access (real-time output, interactivity)   │")
	fmt.Println("│                                                                │")
	fmt.Println("│  Examples:                                                     │")
	fmt.Println("│    terraform plan                                              │")
	fmt.Println("│    terraform apply                                             │")
	fmt.Println("│    az login                                                    │")
	fmt.Println("│    ls -la                                                      │")
	fmt.Println("│                                                                │")
	fmt.Println("╰────────────────────────────────────────────────────────────────╯")
	fmt.Println()
}

// Helper functions

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// getTerminalSize returns the terminal dimensions
func (w *Wrapper) getTerminalSize() (width, height int) {
	// Use syscall to get terminal size
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	ws := &winsize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 || errno != 0 {
		// Default size if we can't detect
		return 80, 24
	}

	return int(ws.Col), int(ws.Row)
}

// renderBottomStatusBar renders the status bar at the bottom of the terminal
func (w *Wrapper) renderBottomStatusBar() {
	// Get current status
	authStatus := w.authManager.CheckAll()
	tfContext, _ := w.tfManager.GetContext()

	// Update terminal size
	w.termWidth, w.termHeight = w.getTerminalSize()

	// Build status lines
	var lines []string

	// Add separator line
	separatorLine := fmt.Sprintf("%s%s%s", colorCyan, strings.Repeat("─", w.termWidth), colorReset)
	lines = append(lines, separatorLine)

	// Terraform context
	if tfContext != nil {
		if tfContext.Workspace != "" {
			lines = append(lines, fmt.Sprintf("%s🏢 Workspace:%s %s%s%s",
				colorCyan, colorReset, colorBold, tfContext.Workspace, colorReset))
		}
		if tfContext.Backend != "" {
			lines = append(lines, fmt.Sprintf("%s💾 Backend:%s %s%s%s",
				colorBlue, colorReset, colorBold, tfContext.Backend, colorReset))
		}
		if tfContext.Module != "" {
			lines = append(lines, fmt.Sprintf("%s📦 Module:%s %s%s%s",
				colorMagenta, colorReset, colorBold, tfContext.Module, colorReset))
		}
	}

	// Auth status
	if az, ok := authStatus["azure"]; ok && az.Authenticated {
		lines = append(lines, fmt.Sprintf("%s🔐 Azure:%s %s%s%s",
			colorGreen, colorReset, colorBold, az.User, colorReset))
	}
	if gh, ok := authStatus["github"]; ok && gh.Authenticated {
		lines = append(lines, fmt.Sprintf("%s🔐 GitHub:%s %s%s%s",
			colorGreen, colorReset, colorBold, gh.User, colorReset))
	}

	// Print status lines inline (scrolls with output, no cursor positioning)
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println() // Add spacing after status bar
}

// buildCompleter creates an autocompleter for readline
func (w *Wrapper) buildCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("terraform",
			readline.PcItem("init"),
			readline.PcItem("plan"),
			readline.PcItem("apply"),
			readline.PcItem("destroy"),
			readline.PcItem("workspace",
				readline.PcItem("list"),
				readline.PcItem("select"),
				readline.PcItem("new"),
			),
			readline.PcItem("output"),
			readline.PcItem("validate"),
			readline.PcItem("fmt"),
		),
		readline.PcItem("cd"),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
		readline.PcItem("help"),
	)
}
