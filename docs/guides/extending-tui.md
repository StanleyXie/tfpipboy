# Guide: Extending the TUI

This guide explains how to extend the TUI (Terminal User Interface) in tfpipboy to add new features, views, and interactive components.

## Overview

tfpipboy uses the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework, which follows the Elm architecture:

- **Model**: Application state
- **Update**: Handles messages and updates state
- **View**: Renders the UI from state

## Architecture

```
pkg/tui/
├── model.go    # Application state
├── update.go   # Event handling and state updates
├── view.go     # UI rendering
└── styles.go   # Visual styling
```

## The Elm Architecture

### Flow

```
User Input → Message → Update → New Model → View → Display
     ↑                                                   ↓
     └──────────────── User sees output ────────────────┘
```

### Example Flow

1. User presses a key → `tea.KeyMsg`
2. `Update()` processes the message
3. Returns new `Model` with updated state
4. `View()` renders the new state
5. Terminal displays the result

## Core Concepts

### 1. Model (State)

The `Model` struct holds all application state:

```go
type Model struct {
	// UI state
	width        int
	height       int
	
	// Feature state
	authManager  *auth.Manager
	authStatus   map[string]*auth.Status
	
	// User input
	input        string
	output       []string
	
	// Metadata
	lastUpdate   time.Time
	err          error
}
```

### 2. Messages

Messages represent events:

```go
// System messages (from Bubble Tea)
tea.KeyMsg      // Keyboard input
tea.MouseMsg    // Mouse events
tea.WindowSizeMsg  // Terminal resize

// Custom messages
type tickMsg time.Time
type authStatusMsg map[string]*auth.Status
type tfContextMsg struct {
	workspace string
	backend   string
}
```

### 3. Commands

Commands produce messages asynchronously:

```go
func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func checkAuthCmd(manager *auth.Manager) tea.Cmd {
	return func() tea.Msg {
		return authStatusMsg(manager.CheckAll())
	}
}
```

## Step-by-Step Guide

### Adding a New Feature

Let's add Terraform context detection as an example.

#### Step 1: Update Model

Add state fields in `pkg/tui/model.go`:

```go
type Model struct {
	// ... existing fields ...
	
	// Terraform context
	tfContext    *terraform.Context
	tfWorkspace  string
	tfBackend    string
}

func NewModel() Model {
	return Model{
		// ... existing initialization ...
		tfContext: terraform.NewContext(),
	}
}
```

#### Step 2: Add Message Types

In `pkg/tui/update.go`, define message types:

```go
// tfContextMsg carries Terraform context information
type tfContextMsg struct {
	workspace string
	backend   string
	module    string
	error     error
}
```

#### Step 3: Create Command

Add command function in `pkg/tui/update.go`:

```go
// checkTfContextCmd checks Terraform context in background
func checkTfContextCmd(ctx *terraform.Context) tea.Cmd {
	return func() tea.Msg {
		workspace, err := ctx.GetWorkspace()
		if err != nil {
			return tfContextMsg{error: err}
		}
		
		backend, err := ctx.GetBackend()
		if err != nil {
			return tfContextMsg{error: err}
		}
		
		module, err := ctx.GetModule()
		if err != nil {
			return tfContextMsg{error: err}
		}
		
		return tfContextMsg{
			workspace: workspace,
			backend:   backend,
			module:    module,
		}
	}
}
```

#### Step 4: Handle Message in Update

In `pkg/tui/update.go`, handle the new message:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	
	// ... existing cases ...
	
	case tfContextMsg:
		if msg.error != nil {
			m.err = msg.error
			return m, nil
		}
		
		m.tfWorkspace = msg.workspace
		m.tfBackend = msg.backend
		// Update display
		return m, nil
	
	case tickMsg:
		m.lastUpdate = time.Time(msg)
		// Trigger both auth check and tf context check
		return m, tea.Batch(
			tickCmd(),
			checkAuthCmd(m.authManager),
			checkTfContextCmd(m.tfContext),  // Add this
		)
	}
	
	return m, nil
}
```

#### Step 5: Update View

In `pkg/tui/view.go`, render the new information:

```go
func (m Model) View() string {
	// Status bar with auth and TF context
	statusText := m.renderStatus()
	status := statusBarStyle.Width(m.width).Render(statusText)
	
	// ... rest of view ...
}

func (m Model) renderStatus() string {
	var parts []string
	
	// Authentication status
	authStatus := m.renderAuthStatus()
	if authStatus != "" {
		parts = append(parts, authStatus)
	}
	
	// Terraform context - NEW
	tfStatus := m.renderTfContext()
	if tfStatus != "" {
		parts = append(parts, tfStatus)
	}
	
	return strings.Join(parts, " | ")
}

func (m Model) renderTfContext() string {
	if m.tfWorkspace == "" {
		return "TF: Not detected"
	}
	
	return fmt.Sprintf("TF: %s @ %s", m.tfWorkspace, m.tfBackend)
}
```

## Common Patterns

### Background Tasks

Use commands for async operations:

```go
func fetchDataCmd(url string) tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(url)
		if err != nil {
			return errorMsg{err}
		}
		// Process response
		return dataMsg{data}
	}
}
```

### Batching Commands

Run multiple commands together:

```go
return m, tea.Batch(
	cmd1(),
	cmd2(),
	cmd3(),
)
```

### Periodic Updates

Use `tea.Tick` for periodic tasks:

```go
func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
```

### Keyboard Handling

```go
case tea.KeyMsg:
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
		
	case "r":
		// Refresh
		return m, refreshCmd()
		
	case "enter":
		// Execute command
		return m, executeCmd(m.input)
	}
```

### Window Resize

```go
case tea.WindowSizeMsg:
	m.width = msg.Width
	m.height = msg.Height
	// Adjust layout
	return m, nil
```

## Styling

### Using Lip Gloss

In `pkg/tui/styles.go`:

```go
var (
	statusBarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)
	
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170"))
	
	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))
	
	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))
)
```

### Using Styles

```go
// Simple styling
text := titleStyle.Render("Hello World")

// With width
text := statusBarStyle.Width(m.width).Render("Status")

// Conditional styling
style := successStyle
if hasError {
	style = errorStyle
}
text := style.Render("Result")
```

### Color Reference

Common ANSI colors:
- Green: `42`
- Red: `196`
- Blue: `62`
- Yellow: `226`
- Gray: `240`
- White: `230`

## Advanced Features

### Adding Interactive Components

Using Bubbles library:

```go
import "github.com/charmbracelet/bubbles/textinput"

type Model struct {
	// ... existing fields ...
	textInput textinput.Model
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.Focus()
	
	return Model{
		textInput: ti,
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.textInput.View()
}
```

### List Component

```go
import "github.com/charmbracelet/bubbles/list"

type item struct {
	title string
	desc  string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type Model struct {
	list list.Model
}

func NewModel() Model {
	items := []list.Item{
		item{title: "Item 1", desc: "Description 1"},
		item{title: "Item 2", desc: "Description 2"},
	}
	
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Items"
	
	return Model{list: l}
}
```

### Progress Bar

```go
import "github.com/charmbracelet/bubbles/progress"

type Model struct {
	progress progress.Model
	percent  float64
}

func NewModel() Model {
	return Model{
		progress: progress.New(progress.WithDefaultGradient()),
		percent:  0.0,
	}
}

func (m Model) View() string {
	return m.progress.ViewAs(m.percent)
}
```

## Layout Techniques

### Vertical Layout

```go
func (m Model) View() string {
	sections := []string{
		m.renderHeader(),
		m.renderContent(),
		m.renderFooter(),
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
```

### Horizontal Layout

```go
func (m Model) View() string {
	left := m.renderSidebar()
	right := m.renderMain()
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}
```

### Grid Layout

```go
func (m Model) View() string {
	row1 := lipgloss.JoinHorizontal(lipgloss.Top, cell1, cell2)
	row2 := lipgloss.JoinHorizontal(lipgloss.Top, cell3, cell4)
	return lipgloss.JoinVertical(lipgloss.Left, row1, row2)
}
```

## Testing

### Testing Update Logic

```go
func TestModel_Update_KeyPress(t *testing.T) {
	m := NewModel()
	
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	newModel, cmd := m.Update(msg)
	
	if cmd == nil {
		t.Error("Expected command to be returned")
	}
	
	// Verify model updated correctly
}
```

### Testing View Output

```go
func TestModel_View_StatusBar(t *testing.T) {
	m := NewModel()
	m.width = 80
	m.authStatus = map[string]*auth.Status{
		"github": {
			Provider:      "github",
			Authenticated: true,
			User:          "testuser",
		},
	}
	
	output := m.View()
	
	if !strings.Contains(output, "✓ GitHub") {
		t.Error("Expected GitHub status in output")
	}
}
```

## Debugging

### Print Debug Info

```go
func (m Model) View() string {
	debug := fmt.Sprintf("Width: %d, Height: %d\n", m.width, m.height)
	debug += fmt.Sprintf("Auth Status: %v\n", m.authStatus)
	
	// ... rest of view ...
	return debug + mainView
}
```

### Log Messages

```go
import "log"

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("Received message: %T - %v\n", msg, msg)
	// ... handle message ...
}
```

## Best Practices

### 1. Keep Model Immutable

```go
// Bad
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.someField = "value"  // Modifying receiver directly
	return m, nil
}

// Good
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel := m
	newModel.someField = "value"
	return newModel, nil
}
```

### 2. Use Commands for Side Effects

```go
// Bad - blocking operation in Update
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	data := fetchData()  // Blocking!
	m.data = data
	return m, nil
}

// Good - async with command
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, fetchDataCmd()
}
```

### 3. Handle Errors Gracefully

```go
case errorMsg:
	m.err = msg.error
	m.errorVisible = true
	return m, clearErrorCmd()  // Auto-clear after timeout
```

### 4. Responsive Design

```go
func (m Model) View() string {
	if m.width < 80 {
		return m.renderCompactView()
	}
	return m.renderFullView()
}
```

### 5. Separate Concerns

```go
// Separate rendering functions
func (m Model) renderHeader() string { ... }
func (m Model) renderContent() string { ... }
func (m Model) renderFooter() string { ... }

func (m Model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(),
		m.renderContent(),
		m.renderFooter(),
	)
}
```

## Example: Adding a Help View

Complete example of adding a help screen:

```go
// model.go
type Model struct {
	// ... existing fields ...
	showHelp bool
}

// update.go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "esc":
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
		}
	}
	return m, nil
}

// view.go
func (m Model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}
	return m.renderMain()
}

func (m Model) renderHelp() string {
	help := `
tfpipboy Help

Keyboard Shortcuts:
  ?        Toggle this help
  r        Refresh status
  q        Quit
  ESC      Close help

Press ESC to close this help.
`
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2)
	
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		style.Render(help),
	)
}
```

## Resources

- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss Documentation](https://github.com/charmbracelet/lipgloss)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Example Applications](https://github.com/charmbracelet/bubbletea/tree/master/examples)

## Next Steps

- Add more interactive components (lists, tables)
- Implement split pane views
- Add mouse support
- Create custom themes
