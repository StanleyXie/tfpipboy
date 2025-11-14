# TUI Implementation - Backup Documentation

**Branch:** `backup/tui-implementation-v1`  
**Date:** 2025-01-14  
**Status:** Functional but replaced with CLI wrapper approach

## Overview

This document describes the Bubble Tea TUI implementation that was developed and tested. While functional, user feedback indicated that a CLI wrapper approach would provide better user experience for real-time output and native terminal interaction.

## Architecture

### Components

1. **Model** (`pkg/tui/model.go`)
   - Application state management
   - Viewport for scrollable output
   - Text input component for command entry
   - Auth and Terraform managers
   - Command history tracking

2. **Update** (`pkg/tui/update.go`)
   - Event handling and message routing
   - Command execution logic
   - Interactive command detection
   - Streaming command execution (attempted)

3. **View** (`pkg/tui/view.go`)
   - UI rendering with lipgloss styles
   - Status bar with Terraform context and auth status
   - Viewport display for command output
   - Help text with keyboard shortcuts

## Features Implemented

### ✅ Working Features

- **Viewport Scrolling**: PgUp/PgDn for page scrolling
- **Command History**: Arrow keys (↑↓) to navigate previous commands
- **Mouse Support**: Mouse wheel scrolling (with Shift+Select for text copy)
- **Interactive Commands**: Automatic TUI suspension for commands requiring user input
  - `terraform apply`, `terraform destroy`
  - `az login`, `gcloud auth login`
  - Text editors: `vim`, `nano`, etc.
- **Status Bar**: Real-time display of:
  - Terraform workspace, backend, module
  - Authentication status (Azure, GitHub)
  - Git branch (via environment)
- **Tab Completion**: Files and common commands
- **Command Output**: Full output display after command completes
- **Quit Key**: Ctrl+Q (doesn't interfere with 'q' in commands)

### ⚠️ Known Limitations

1. **No Real-Time Streaming**: Output appears only after command completes
2. **No Progress Indicators**: Terraform progress bars not visible during execution
3. **Complex Implementation**: Attempted streaming with channels was too complex
4. **Output Review**: Can't review output while command is running
5. **Carriage Returns**: Line-by-line scanner doesn't handle `\r` progress updates

## Key Implementation Decisions

### Interactive Command Detection

Commands that require user input or show progress bars are run with `tea.ExecProcess`:

```go
func isInteractiveCommand(command string) bool {
    interactivePatterns := []string{
        "terraform apply",
        "terraform destroy",
        "terraform console",
        "az login",
        "vim", "vi", "nano",
        // ... etc
    }
    // Check if command matches patterns
}
```

This suspends the TUI and gives full terminal control to the command.

### Command Execution Flow

```
User Input (Enter)
    ↓
Is Interactive? → Yes → tea.ExecProcess (suspend TUI)
    ↓ No                    ↓
executeCommandCmd       Run in native terminal
    ↓                       ↓
executeStreamingCommand Return to TUI with status
    ↓
Collect output
    ↓
Return commandResultMsg
    ↓
Display in viewport
```

### Viewport Management

- Viewport receives all keyboard events EXCEPT arrow keys (used for history)
- Arrow keys filtered before reaching viewport to prevent scroll interference
- Auto-scroll to bottom when new output arrives
- Manual scroll with PgUp/PgDn

## Research: Terragrunt's Approach

We researched how Terragrunt handles Terraform command execution:

### Key Findings

1. **Direct Stream Connection**: They connect `cmd.Stdout/Stderr` directly to `os.Stdout/Stderr`
2. **io.MultiWriter**: For simultaneous display and capture
3. **No Line-by-Line Scanning**: Avoids buffering delays
4. **PTY Support**: Uses pseudo-terminals for truly interactive commands
5. **No TUI**: Terragrunt is a CLI wrapper, not a TUI app

### Why TUI is Challenging

The fundamental issue: Bubble Tea runs in **alternate screen mode**. When commands write to `os.Stdout`, it goes to the hidden terminal underneath, not the TUI viewport.

To show output in TUI, we must:
- Capture output (introduces buffering)
- OR suspend TUI and lose the viewport

## Attempted Solutions

### Attempt 1: Real-Time Streaming with Channels

```go
// Tried to send messages from goroutine
go func() {
    scanner := bufio.NewScanner(stdoutPipe)
    for scanner.Scan() {
        outputChan <- scanner.Text()  // Send to TUI
    }
}()
```

**Problem**: Bubble Tea's `tea.Cmd` can only return ONE message, not multiple. Needed complex subscription patterns.

### Attempt 2: Streaming Message Types

```go
type streamingOutputMsg struct { line string }
type commandFinishedMsg struct { err error }
```

**Problem**: Managing the channel lifecycle and state was complex. Also, `bufio.Scanner` blocks until newline, so progress bars (using `\r`) don't work.

## Why We're Moving to CLI Wrapper

### User Requirements

1. **Real-time output**: See terraform progress as it happens
2. **Native interaction**: Interactive prompts should work perfectly
3. **Future orchestration**: Multi-module parallel execution
4. **Ghostty integration**: Use Ghostty's split/tabs for orchestration

### CLI Wrapper Advantages

- ✅ Direct terminal access (no buffering)
- ✅ Progress bars work
- ✅ Interactive prompts work
- ✅ Simpler implementation
- ✅ Can add status bar with ANSI escapes
- ✅ Easier to integrate with Ghostty for multi-session

### TUI Disadvantages for This Use Case

- ❌ Complex streaming implementation
- ❌ Alternate screen mode hides command output
- ❌ Can't show real-time progress without suspension
- ❌ When suspended, viewport is hidden anyway

## Code Reference

### Main Files

- `pkg/tui/model.go`: 70 lines - State management
- `pkg/tui/update.go`: 480 lines - Event handling, command execution
- `pkg/tui/view.go`: 140 lines - UI rendering
- `cmd/tfpipboy/main.go`: 25 lines - Entry point

### Key Functions

- `executeCommandCmd`: Wraps command execution in tea.Cmd
- `executeStreamingCommand`: Executes command and collects output
- `isInteractiveCommand`: Detects commands needing user input
- `runInteractiveCommand`: Suspends TUI via tea.ExecProcess
- `Update`: Main event loop with message routing

## Lessons Learned

1. **TUI is great for**: Dashboards, logs, status displays, data exploration
2. **TUI is challenging for**: Real-time command wrappers, interactive shells
3. **tea.ExecProcess**: Works well for interactive commands but hides the TUI
4. **Real-time in TUI**: Requires complex async patterns with channels
5. **User experience**: Sometimes native terminal is better than forced TUI

## Migration Path

### Phase 1: CLI Wrapper (Current)
- Remove Bubble Tea dependency
- Use ANSI escape codes for status bar
- Direct terminal access for all commands
- Sequential execution

### Phase 2: Ghostty Orchestration (Future)
- Detect Ghostty terminal
- Use Ghostty API for splits/tabs
- Parallel execution across panes
- Status aggregation

## References

- Bubble Tea: https://github.com/charmbracelet/bubbletea
- Terragrunt shell package: https://github.com/gruntwork-io/terragrunt/tree/main/shell
- Ghostty API discussion: https://github.com/ghostty-org/ghostty/discussions/2353
- Design docs: `design/04-ghostty-integration.md`

## How to Restore This Implementation

If you want to go back to the TUI approach:

```bash
git checkout backup/tui-implementation-v1
go build -o tfpipboy cmd/tfpipboy/main.go
./tfpipboy
```

The TUI will work, with all features listed above, but with the limitations noted.
