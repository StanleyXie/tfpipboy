package tui

import (
	"time"

	"github.com/StanleyXie/tf-pipboy/pkg/auth"
	"github.com/StanleyXie/tf-pipboy/pkg/terraform"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the application state
type Model struct {
	statusBar        string
	textInput        textinput.Model
	viewport         viewport.Model
	output           []string
	commandHistory   []string
	historyIndex     int
	width            int
	height           int
	ready            bool
	lastUpdate       time.Time
	err              error
	authManager      *auth.Manager
	authStatus       map[string]*auth.Status
	tfManager        *terraform.Manager
	tfContext        *terraform.Context
	streamingChan    chan string // Channel for receiving streaming output
	streamingCommand string      // Current command being streamed
}

// NewModel creates a new TUI model
func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 80
	ti.Prompt = "> "
	ti.PromptStyle = ti.PromptStyle.Foreground(ti.PromptStyle.GetForeground())

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to tf-pipboy!\n\nType any command and press Enter to execute.\nExamples:\n  terraform version\n  terraform init\n  ls -la")
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3

	return Model{
		statusBar:      "tf-pipboy - Loading...",
		textInput:      ti,
		viewport:       vp,
		output:         []string{},
		commandHistory: []string{},
		historyIndex:   0,
		ready:          false,
		lastUpdate:     time.Now(),
		authManager:    auth.NewManager(),
		authStatus:     make(map[string]*auth.Status),
		tfManager:      terraform.NewManager(),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		checkAuthCmd(m.authManager),    // Initial auth check
		checkTfContextCmd(m.tfManager), // Initial TF context check
	)
}

// tickMsg is sent every 5 seconds to refresh status
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
