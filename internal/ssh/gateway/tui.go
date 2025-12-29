package gateway

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Ensure Terminal, Agent, Snippet implement list.Item
var (
	_ list.Item = Terminal{}
	_ list.Item = Agent{}
	_ list.Item = Snippet{}
)

// Colors
var (
	primaryColor   = lipgloss.Color("#00D4FF")
	secondaryColor = lipgloss.Color("#7C3AED")
	successColor   = lipgloss.Color("#10B981")
	warningColor   = lipgloss.Color("#F59E0B")
	errorColor     = lipgloss.Color("#EF4444")
	dimColor       = lipgloss.Color("#6B7280")
	bgColor        = lipgloss.Color("#0F172A")
	cardBgColor    = lipgloss.Color("#1E293B")
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Padding(0, 1)

	menuItemStyle = lipgloss.NewStyle().
			Padding(0, 2)

	selectedMenuStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				Padding(0, 2)

	dimStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dimColor).
			Padding(1, 2)

	highlightBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(1, 2)
)

// View represents the current screen
type View int

const (
	ViewDashboard View = iota
	ViewTerminals
	ViewAgents
	ViewSnippets
	ViewLinkAccount
	ViewTerminal
)

// ModelConfig holds configuration for the TUI model
type ModelConfig struct {
	SessionID   string
	UserID      string
	Username    string
	Email       string
	Token       string
	IsGuest     bool
	Fingerprint string
	APIURL      string
	Width       int
	Height      int
}

// Model is the Bubble Tea model for the SSH TUI
type Model struct {
	config    ModelConfig
	apiClient *APIClient
	view      View
	width     int
	height    int

	// Dashboard
	menuItems    []string
	selectedItem int

	// Terminal list
	terminalList list.Model
	terminals    []Terminal

	// Agent list
	agentList list.Model
	agents    []Agent

	// Snippet list
	snippetList list.Model
	snippets    []Snippet

	// Link account
	linkInput textinput.Model
	linkError string

	// UI state
	loading bool
	spinner spinner.Model
	err     error
}

// Terminal represents a container terminal
type Terminal struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	Status    string `json:"status"`
	MFALocked bool   `json:"mfa_locked"`
}

func (t Terminal) Title() string       { return t.Name }
func (t Terminal) Description() string { return fmt.Sprintf("%s • %s", t.Image, t.Status) }
func (t Terminal) FilterValue() string { return t.Name }

// Agent represents a remote agent
type Agent struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Status   string `json:"status"`
}

func (a Agent) Title() string       { return a.Name }
func (a Agent) Description() string { return fmt.Sprintf("%s • %s", a.Hostname, a.Status) }
func (a Agent) FilterValue() string { return a.Name }

// Snippet represents a saved snippet
type Snippet struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Command  string `json:"command"`
	Language string `json:"language"`
}

func (s Snippet) Title() string       { return s.Name }
func (s Snippet) Description() string { return s.Command }
func (s Snippet) FilterValue() string { return s.Name }

// Messages
type terminalsMsg []Terminal
type agentsMsg []Agent
type snippetsMsg []Snippet
type errMsg struct{ err error }
type linkSuccessMsg struct{}
type linkFailMsg struct{ err string }

// NewModel creates a new TUI model
func NewModel(cfg ModelConfig) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	// Link input
	linkInput := textinput.New()
	linkInput.Placeholder = "Enter your email or API token"
	linkInput.Focus()
	linkInput.CharLimit = 256
	linkInput.Width = 40

	// Menu items depend on guest status
	var menuItems []string
	if cfg.IsGuest {
		menuItems = []string{
			"🎮 Try Demo Terminal",
			"👀 Explore Features",
			"🔗 Link Account",
			"📝 Sign Up",
			"🚪 Quit",
		}
	} else {
		menuItems = []string{
			"📦 Terminals",
			"🤖 Agents",
			"📝 Snippets",
			"➕ Create Terminal",
			"⚙️  Settings",
			"🚪 Quit",
		}
	}

	// Terminal list
	terminalDelegate := list.NewDefaultDelegate()
	terminalDelegate.Styles.SelectedTitle = terminalDelegate.Styles.SelectedTitle.Foreground(primaryColor)
	terminalList := list.New([]list.Item{}, terminalDelegate, cfg.Width-4, cfg.Height-10)
	terminalList.Title = "Terminals"
	terminalList.SetShowHelp(false)

	// Agent list
	agentDelegate := list.NewDefaultDelegate()
	agentDelegate.Styles.SelectedTitle = agentDelegate.Styles.SelectedTitle.Foreground(primaryColor)
	agentList := list.New([]list.Item{}, agentDelegate, cfg.Width-4, cfg.Height-10)
	agentList.Title = "Agents"
	agentList.SetShowHelp(false)

	// Snippet list
	snippetDelegate := list.NewDefaultDelegate()
	snippetDelegate.Styles.SelectedTitle = snippetDelegate.Styles.SelectedTitle.Foreground(primaryColor)
	snippetList := list.New([]list.Item{}, snippetDelegate, cfg.Width-4, cfg.Height-10)
	snippetList.Title = "Snippets"
	snippetList.SetShowHelp(false)

	// Create API client
	var apiClient *APIClient
	if cfg.Token != "" {
		apiClient = NewAPIClient(cfg.APIURL, cfg.Token)
	}

	return Model{
		config:       cfg,
		apiClient:    apiClient,
		view:         ViewDashboard,
		width:        cfg.Width,
		height:       cfg.Height,
		menuItems:    menuItems,
		selectedItem: 0,
		terminalList: terminalList,
		agentList:    agentList,
		snippetList:  snippetList,
		linkInput:    linkInput,
		spinner:      s,
		loading:      !cfg.IsGuest && apiClient != nil,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	if m.config.IsGuest || m.apiClient == nil {
		return m.spinner.Tick
	}
	// Fetch data for authenticated users
	return tea.Batch(
		m.spinner.Tick,
		CreateTerminalCmd(m.apiClient),
		CreateAgentCmd(m.apiClient),
		CreateSnippetCmd(m.apiClient),
	)
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.terminalList.SetSize(msg.Width-4, msg.Height-10)
		m.agentList.SetSize(msg.Width-4, msg.Height-10)
		m.snippetList.SetSize(msg.Width-4, msg.Height-10)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case terminalsMsg:
		m.loading = false
		m.terminals = msg
		items := make([]list.Item, len(msg))
		for i, t := range msg {
			items[i] = t
		}
		m.terminalList.SetItems(items)

	case agentsMsg:
		m.agents = msg
		items := make([]list.Item, len(msg))
		for i, a := range msg {
			items[i] = a
		}
		m.agentList.SetItems(items)

	case snippetsMsg:
		m.snippets = msg
		items := make([]list.Item, len(msg))
		for i, s := range msg {
			items[i] = s
		}
		m.snippetList.SetItems(items)

	case errMsg:
		m.loading = false
		m.err = msg.err

	case linkSuccessMsg:
		m.config.IsGuest = false
		m.view = ViewDashboard
		// Refresh menu for authenticated user
		m.menuItems = []string{
			"📦 Terminals",
			"🤖 Agents",
			"📝 Snippets",
			"➕ Create Terminal",
			"⚙️  Settings",
			"🚪 Quit",
		}
		if m.apiClient != nil {
			return m, tea.Batch(
				CreateTerminalCmd(m.apiClient),
				CreateAgentCmd(m.apiClient),
				CreateSnippetCmd(m.apiClient),
			)
		}
		return m, nil

	case linkFailMsg:
		m.linkError = msg.err
	}

	// Update sub-models based on view
	switch m.view {
	case ViewTerminals:
		var cmd tea.Cmd
		m.terminalList, cmd = m.terminalList.Update(msg)
		cmds = append(cmds, cmd)
	case ViewAgents:
		var cmd tea.Cmd
		m.agentList, cmd = m.agentList.Update(msg)
		cmds = append(cmds, cmd)
	case ViewSnippets:
		var cmd tea.Cmd
		m.snippetList, cmd = m.snippetList.Update(msg)
		cmds = append(cmds, cmd)
	case ViewLinkAccount:
		var cmd tea.Cmd
		m.linkInput, cmd = m.linkInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		if m.view == ViewDashboard {
			return m, tea.Quit
		}
		m.view = ViewDashboard
		return m, nil
	case "esc":
		if m.view != ViewDashboard {
			m.view = ViewDashboard
			m.linkError = ""
		}
		return m, nil
	}

	// View-specific keys
	switch m.view {
	case ViewDashboard:
		return m.handleDashboardKeys(msg)
	case ViewTerminals:
		return m.handleTerminalListKeys(msg)
	case ViewAgents:
		return m.handleAgentListKeys(msg)
	case ViewSnippets:
		return m.handleSnippetListKeys(msg)
	case ViewLinkAccount:
		return m.handleLinkAccountKeys(msg)
	}

	return m, nil
}

func (m Model) handleDashboardKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedItem > 0 {
			m.selectedItem--
		}
	case "down", "j":
		if m.selectedItem < len(m.menuItems)-1 {
			m.selectedItem++
		}
	case "enter":
		return m.selectMenuItem()
	case "1", "2", "3", "4", "5", "6":
		idx := int(msg.String()[0] - '1')
		if idx < len(m.menuItems) {
			m.selectedItem = idx
			return m.selectMenuItem()
		}
	case "l", "L":
		if m.config.IsGuest {
			m.view = ViewLinkAccount
			m.linkInput.Focus()
		}
	}
	return m, nil
}

func (m Model) selectMenuItem() (tea.Model, tea.Cmd) {
	if m.config.IsGuest {
		switch m.selectedItem {
		case 0: // Try Demo Terminal
			// TODO: Launch demo terminal
			return m, nil
		case 1: // Explore Features
			// TODO: Show features
			return m, nil
		case 2: // Link Account
			m.view = ViewLinkAccount
			m.linkInput.Focus()
			return m, nil
		case 3: // Sign Up
			// TODO: Show sign up info
			return m, nil
		case 4: // Quit
			return m, tea.Quit
		}
	} else {
		switch m.selectedItem {
		case 0: // Terminals
			m.view = ViewTerminals
		case 1: // Agents
			m.view = ViewAgents
		case 2: // Snippets
			m.view = ViewSnippets
		case 3: // Create Terminal
			// TODO: Create terminal flow
		case 4: // Settings
			// TODO: Settings view
		case 5: // Quit
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) handleTerminalListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if item, ok := m.terminalList.SelectedItem().(Terminal); ok {
			// TODO: Connect to terminal
			_ = item
		}
	case "r":
		if m.apiClient != nil {
			m.loading = true
			return m, CreateTerminalCmd(m.apiClient)
		}
	}
	return m, nil
}

func (m Model) handleAgentListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if item, ok := m.agentList.SelectedItem().(Agent); ok {
			// TODO: Connect to agent
			_ = item
		}
	case "r":
		if m.apiClient != nil {
			return m, CreateAgentCmd(m.apiClient)
		}
	}
	return m, nil
}

func (m Model) handleSnippetListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if item, ok := m.snippetList.SelectedItem().(Snippet); ok {
			// TODO: Run snippet
			_ = item
		}
	case "r":
		if m.apiClient != nil {
			return m, CreateSnippetCmd(m.apiClient)
		}
	}
	return m, nil
}

func (m Model) handleLinkAccountKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		value := m.linkInput.Value()
		if value != "" {
			// Create temp client to validate and link
			tempClient := NewAPIClient(m.config.APIURL, value)
			return m, CreateLinkAccountCmd(tempClient, m.config.Fingerprint, value)
		}
	}
	return m, nil
}

// View implements tea.Model
func (m Model) View() string {
	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Content based on view
	switch m.view {
	case ViewDashboard:
		b.WriteString(m.renderDashboard())
	case ViewTerminals:
		b.WriteString(m.renderTerminalList())
	case ViewAgents:
		b.WriteString(m.renderAgentList())
	case ViewSnippets:
		b.WriteString(m.renderSnippetList())
	case ViewLinkAccount:
		b.WriteString(m.renderLinkAccount())
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m Model) renderHeader() string {
	logo := `
██████╗ ███████╗██╗  ██╗███████╗ ██████╗
██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝
██████╔╝█████╗   ╚███╔╝ █████╗  ██║
██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║
██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗
╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝`

	logoStyled := titleStyle.Render(logo)

	// User info
	var userInfo string
	if m.config.IsGuest {
		fp := m.config.Fingerprint
		if len(fp) > 20 {
			fp = fp[:20] + "..."
		}
		userInfo = dimStyle.Render(fmt.Sprintf("👋 Guest • SSH Key: %s", fp))
	} else {
		userInfo = successStyle.Render(fmt.Sprintf("✓ %s <%s>", m.config.Username, m.config.Email))
	}

	return lipgloss.JoinVertical(lipgloss.Left, logoStyled, "", userInfo)
}

func (m Model) renderDashboard() string {
	var b strings.Builder

	if m.config.IsGuest {
		// Guest banner
		banner := highlightBoxStyle.Render(
			"🔗 Link your account for full access to terminals, agents, and snippets.\n" +
				"   Press [L] to link now, or explore as a guest.",
		)
		b.WriteString("\n")
		b.WriteString(banner)
		b.WriteString("\n\n")
	}

	// Menu items
	for i, item := range m.menuItems {
		prefix := "  "
		style := menuItemStyle

		if i == m.selectedItem {
			prefix = "▸ "
			style = selectedMenuStyle
		}

		line := fmt.Sprintf("[%d] %s%s", i+1, prefix, item)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	// Stats for authenticated users
	if !m.config.IsGuest && !m.loading {
		b.WriteString("\n")
		stats := dimStyle.Render(fmt.Sprintf(
			"📦 %d terminals • 🤖 %d agents • 📝 %d snippets",
			len(m.terminals), len(m.agents), len(m.snippets),
		))
		b.WriteString(stats)
	}

	if m.loading {
		b.WriteString("\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading...")
	}

	return b.String()
}

func (m Model) renderTerminalList() string {
	if m.loading {
		return m.spinner.View() + " Loading terminals..."
	}
	if len(m.terminals) == 0 {
		return dimStyle.Render("\nNo terminals found. Press 'c' to create one.")
	}
	return m.terminalList.View()
}

func (m Model) renderAgentList() string {
	if len(m.agents) == 0 {
		return dimStyle.Render("\nNo agents connected. Install an agent on your servers to get started.")
	}
	return m.agentList.View()
}

func (m Model) renderSnippetList() string {
	if len(m.snippets) == 0 {
		return dimStyle.Render("\nNo snippets saved. Create snippets in the web UI.")
	}
	return m.snippetList.View()
}

func (m Model) renderLinkAccount() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("🔗 Link Your Account"))
	b.WriteString("\n\n")

	b.WriteString("Enter your email address or API token to link this SSH key\n")
	b.WriteString("to your Rexec account.\n\n")

	b.WriteString(boxStyle.Render(m.linkInput.View()))
	b.WriteString("\n\n")

	if m.linkError != "" {
		b.WriteString(errorStyle.Render("Error: " + m.linkError))
		b.WriteString("\n")
	}

	b.WriteString(dimStyle.Render("Your SSH key fingerprint: " + m.config.Fingerprint))

	return b.String()
}

func (m Model) renderHelp() string {
	var keys []string

	switch m.view {
	case ViewDashboard:
		keys = []string{"↑/↓ navigate", "enter select", "1-6 quick select"}
		if m.config.IsGuest {
			keys = append(keys, "L link account")
		}
		keys = append(keys, "q quit")
	case ViewTerminals, ViewAgents, ViewSnippets:
		keys = []string{"↑/↓ navigate", "enter select", "r refresh", "esc back", "q quit"}
	case ViewLinkAccount:
		keys = []string{"enter submit", "esc cancel"}
	}

	help := strings.Join(keys, " • ")
	return dimStyle.Render(help)
}

// Keybindings
type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Back   key.Binding
	Quit   key.Binding
	Link   key.Binding
	Reload key.Binding
}

var defaultKeyMap = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Link: key.NewBinding(
		key.WithKeys("l", "L"),
		key.WithHelp("L", "link account"),
	),
	Reload: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
}
