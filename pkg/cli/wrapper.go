package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/StanleyXie/tfpipboy/pkg/auth"
	"github.com/StanleyXie/tfpipboy/pkg/terraform"
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
)

// Wrapper is the main CLI wrapper structure
type Wrapper struct {
	authManager    *auth.Manager
	tfManager      *terraform.Manager
	commandHistory []string
	historyIndex   int
	termHeight     int
	termWidth      int
}

// NewWrapper creates a new CLI wrapper instance
func NewWrapper() *Wrapper {
	return &Wrapper{
		authManager:    auth.NewManager(),
		tfManager:      terraform.NewManager(),
		commandHistory: []string{},
		historyIndex:   0,
	}
}

// Run starts the CLI wrapper main loop
func (w *Wrapper) Run() error {
	// Print initial header
	w.printWelcome()
	w.printStatus()

	// Create readline instance with configuration
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          w.buildPrompt(),
		HistoryFile:     os.ExpandEnv("$HOME/.tfpipboy_history"),
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
	fmt.Printf("%s│%s                        %stfpipboy v0.2.0%s                            %s│%s\n",
		colorCyan, colorReset, colorBold+colorYellow, colorReset, colorCyan, colorReset)
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

// handleCD handles directory change
func (w *Wrapper) handleCD(parts []string) {
	var dir string

	if len(parts) > 1 {
		dir = parts[1]
	} else {
		dir = os.Getenv("HOME")
	}

	// Expand ~
	if strings.HasPrefix(dir, "~") {
		dir = strings.Replace(dir, "~", os.Getenv("HOME"), 1)
	}

	// Change directory
	if err := os.Chdir(dir); err != nil {
		fmt.Printf("\033[31mcd: %v\033[0m\n", err)
		return
	}

	// Update terraform manager path
	// Status will be updated by main loop after this returns
	newPath, _ := os.Getwd()
	w.tfManager.SetPath(newPath)
}

// runExternalCommand runs an external command with full terminal access
func (w *Wrapper) runExternalCommand(command string) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell, "-c", command)
	cmd.Dir, _ = os.Getwd()
	cmd.Env = os.Environ()

	// Connect directly to terminal for real-time output
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run and wait
	if err := cmd.Run(); err != nil {
		fmt.Printf("\n\033[31m[Error: %v]\033[0m\n", err)
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

// statusUpdateLoop runs in background to update status
// NOTE: This function is currently disabled because background updates
// interfere with liner's display. Status is updated after each command instead.
func (w *Wrapper) statusUpdateLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.updateStatus()
		}
	}
}

// updateStatus refreshes the status bar
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
