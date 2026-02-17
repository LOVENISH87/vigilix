package ui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"vigilix/internal/systemd"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shirou/gopsutil/v3/host"
)

// --- Color Scheme (Dracula-inspired) ---
var (
	background = lipgloss.Color("#282a36")
	current    = lipgloss.Color("#44475a")
	foreground = lipgloss.Color("#f8f8f2")
	comment    = lipgloss.Color("#6272a4")
	cyan       = lipgloss.Color("#8be9fd")
	green      = lipgloss.Color("#50fa7b")
	orange     = lipgloss.Color("#ffb86c")
	pink       = lipgloss.Color("#ff79c6")
	purple     = lipgloss.Color("#2d57ff")
	red        = lipgloss.Color("#ff5555")
	yellow     = lipgloss.Color("#f1fa8c")
)

// --- Styles ---
var (
	// Base
	baseStyle = lipgloss.NewStyle().Foreground(foreground)

	// Panel Borders
	panelBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	panelStyle = baseStyle.Copy().
			Border(panelBorder).
			BorderForeground(comment)

	focusedPanelStyle = panelStyle.Copy().
				BorderForeground(purple)

	// Tabs
	activeTabStyle = baseStyle.Copy().
			Bold(true).
			Foreground(background).
			Background(purple).
			Padding(0, 1)

	inactiveTabStyle = baseStyle.Copy().
				Foreground(comment).
				Padding(0, 1)

	// Titles
	titleStyle = baseStyle.Copy().
			Bold(true).
			Padding(0, 1).
			Foreground(cyan)
)

// --- Help Keys ---
type keyMap struct {
	Up, Down, Left, Right key.Binding
	Enter, Esc, Tab       key.Binding
	Start, Stop, Restart  key.Binding
	Config                key.Binding
	Quit                  key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Enter, k.Config, k.Start, k.Stop, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Esc, k.Tab},
		{k.Start, k.Stop, k.Restart, k.Config},
		{k.Quit},
	}
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Left:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left")),
	Right:   key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right")),
	Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Esc:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
	Start:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start")),
	Stop:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "stop")),
	Restart: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart")),
	Config:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "config")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// --- Model ---

const (
	PaneList = iota
	PaneContent
)

const (
	ModeDashboard = iota
	ModeList
	ModeLogs
	ModeConfig
)

type item struct {
	unit systemd.Unit
}

func (i item) Title() string {
	// Icon + Name
	icon := "📦"
	name := strings.ToLower(i.unit.Name)
	if strings.Contains(name, "docker") {
		icon = "🐳"
	}
	if strings.Contains(name, "mongo") {
		icon = "🍃"
	}
	if strings.Contains(name, "postgres") || strings.Contains(name, "psql") {
		icon = "🐘"
	}
	if strings.Contains(name, "mysql") || strings.Contains(name, "mariadb") {
		icon = "🐬"
	}
	if strings.Contains(name, "redis") {
		icon = "🔺"
	}
	if strings.Contains(name, "nginx") {
		icon = "🌐"
	}
	if strings.Contains(name, "apache") || strings.Contains(name, "httpd") {
		icon = "🪶"
	}
	if strings.Contains(name, "ssh") {
		icon = "🔒"
	}
	if strings.Contains(name, "node") || strings.Contains(name, "npm") {
		icon = "🟢"
	}
	if strings.Contains(name, "python") {
		icon = "🐍"
	}
	if strings.Contains(name, "go") {
		icon = "🐹"
	}

	return fmt.Sprintf("%s %s", icon, i.unit.Name)
}

func (i item) Description() string {
	// Status Dot + Load State + Description
	status := "⚪"
	if i.unit.ActiveState == "active" {
		status = "🟢"
	} else if i.unit.ActiveState == "failed" {
		status = "🔴"
	}

	return fmt.Sprintf("%s %s | %s", status, i.unit.ActiveState, i.unit.Description)
}

func (i item) FilterValue() string { return i.unit.Name }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 2 }
func (d itemDelegate) Spacing() int                              { return 1 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	// 2. Width Calculation
	totalWidth := m.Width()
	if totalWidth <= 0 {
		totalWidth = 40
	} // fallback

	// Styles
	titleStyle := baseStyle.Copy().Bold(true)
	descStyle := baseStyle.Copy().Foreground(comment)

	// Status Badge
	activeState := i.unit.ActiveState
	statusColor := comment
	statusFg := foreground

	switch activeState {
	case "active":
		statusColor = green
		statusFg = lipgloss.Color("#000000")
	case "failed":
		statusColor = red
	case "inactive":
		statusColor = lipgloss.Color("#44475a") // Dark gray
	}

	statusBadge := lipgloss.NewStyle().
		Background(statusColor).
		Foreground(statusFg).
		Padding(0, 1).
		Bold(true).
		Render(strings.ToUpper(activeState))

	// Selection Special Handling
	isSelected := index == m.Index()

	var itemStyle lipgloss.Style
	if isSelected {
		itemStyle = lipgloss.NewStyle().
			Background(current).
			PaddingLeft(1).
			Border(lipgloss.NormalBorder(), false, false, false, true). // Left border
			BorderForeground(purple)

		titleStyle = titleStyle.Foreground(purple)
		descStyle = descStyle.Foreground(foreground)
	} else {
		itemStyle = lipgloss.NewStyle().PaddingLeft(2) // Match the 1 padding + 1 border width of selected state
		titleStyle = titleStyle.Foreground(foreground)
	}

	// Width available for text inside the style
	innerWidth := totalWidth - itemStyle.GetHorizontalFrameSize()
	if innerWidth < 0 {
		innerWidth = 0
	}

	// 4. Layout Line 1 (Title + Badge)
	badgeWidth := lipgloss.Width(statusBadge)
	availableTitleWidth := innerWidth - badgeWidth - 2 // 2 chars gap

	titleStr := i.Title()
	if lipgloss.Width(titleStr) > availableTitleWidth {
		if availableTitleWidth > 3 {
			titleStr = titleStr[:availableTitleWidth-3] + "..."
		} else {
			titleStr = "" // Hide if too small
		}
	}

	left1 := titleStyle.Render(titleStr)
	gapSize := innerWidth - lipgloss.Width(left1) - badgeWidth
	if gapSize < 0 {
		gapSize = 0
	}
	gap := strings.Repeat(" ", gapSize)

	line1 := left1 + gap + statusBadge

	// 5. Layout Line 2 (Description)
	descStr := i.unit.Description
	if lipgloss.Width(descStr) > innerWidth {
		if innerWidth > 3 {
			descStr = descStr[:innerWidth-3] + "..."
		} else {
			descStr = ""
		}
	}
	line2 := descStyle.Render(descStr)

	// 6. Combine and Render
	content := fmt.Sprintf("%s\n%s", line1, line2)

	// Force the style to take full width so background fills properly
	fmt.Fprint(w, itemStyle.Width(innerWidth).Render(content))
}

type errMsg error
type actionResultMsg struct {
	err    error
	action string
}
type logLineMsg string
type configMsg string
type statsMsg struct {
	hostname string
	os       string
	uptime   uint64
	kernel   string
}

type model struct {
	// Bubbles
	list     list.Model
	viewport viewport.Model
	help     help.Model
	spinner  spinner.Model

	// State
	activePane int
	viewMode   int
	devMode    bool

	// Layout
	width, height int

	// Data
	allUnits      []systemd.Unit
	logLines      []string
	configContent string
	streamingUnit string
	stats         statsMsg

	// Async
	logCtx    context.Context
	logCancel context.CancelFunc
	logChan   chan string

	// Meta
	err           error
	statusMessage string
}

func NewModel() model {
	// 1. List - Custom Delegate
	delegate := itemDelegate{}

	l := list.New(nil, delegate, 0, 0)
	l.Title = "Units"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false) // Custom header used instead
	l.SetFilteringEnabled(true)
	l.SetShowPagination(false)
	l.Styles.Title = titleStyle
	l.DisableQuitKeybindings()

	// 2. Viewport
	vp := viewport.New(0, 0)

	// 3. Spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(pink)

	return model{
		list:          l,
		viewport:      vp,
		help:          help.New(),
		spinner:       s,
		activePane:    PaneList,
		viewMode:      ModeDashboard,
		devMode:       true,
		logLines:      []string{},
		statusMessage: "Ready",
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchUnits,
		m.spinner.Tick,
		fetchStats,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global Quit
		if key.Matches(msg, keys.Quit) {
			if m.logCancel != nil {
				m.logCancel()
			}
			return m, tea.Quit
		}

		// Dashboard Interaction
		if m.viewMode == ModeDashboard {
			switch msg.String() {
			case "enter", "space", "tab", "l", "right":
				m.viewMode = ModeList
				m.activePane = PaneList
				return m, nil
			}
			return m, nil
		}

		// Global Tab Navigation
		if key.Matches(msg, keys.Tab) {
			if m.activePane == PaneList {
				m.activePane = PaneContent
			} else {
				m.activePane = PaneList
			}
			return m, nil
		}

		// Filter Toggle (d)
		if msg.String() == "d" {
			m.devMode = !m.devMode
			m.updateListItems()
			m.statusMessage = fmt.Sprintf("Dev Mode: %v", m.devMode)
			return m, nil
		}

		// If filtering, list handles input
		if m.activePane == PaneList && m.list.SettingFilter() {
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		switch m.activePane {
		case PaneList:
			switch {
			case key.Matches(msg, keys.Enter):
				m.viewMode = ModeLogs
				m.activePane = PaneContent
				if i, ok := m.list.SelectedItem().(item); ok {
					m.startStreaming(i.unit.Name)
					cmds = append(cmds, waitForLogLine(m.logChan))
				}
			case key.Matches(msg, keys.Config):
				m.viewMode = ModeConfig
				m.activePane = PaneContent
				if i, ok := m.list.SelectedItem().(item); ok {
					cmds = append(cmds, fetchConfig(i.unit.Name))
				}
			case key.Matches(msg, keys.Start):
				if i, ok := m.list.SelectedItem().(item); ok {
					cmds = append(cmds, performAction(systemd.StartUnit, i.unit.Name, "Started"))
				}
			case key.Matches(msg, keys.Stop):
				if i, ok := m.list.SelectedItem().(item); ok {
					cmds = append(cmds, performAction(systemd.StopUnit, i.unit.Name, "Stopped"))
				}
			case key.Matches(msg, keys.Restart):
				if i, ok := m.list.SelectedItem().(item); ok {
					cmds = append(cmds, performAction(systemd.RestartUnit, i.unit.Name, "Restarted"))
				}
			}
			m.list, cmd = m.list.Update(msg)
			cmds = append(cmds, cmd)

		case PaneContent:
			if key.Matches(msg, keys.Esc) {
				m.activePane = PaneList
				return m, nil
			}
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

		contentHeight := m.height - 4
		contentWidth := m.width - 4

		sidebarWidth := int(float64(contentWidth) * 0.35)
		mainWidth := contentWidth - sidebarWidth

		// Adjust list size to account for header
		headerHeight := 2 // Text + Border
		m.list.SetSize(sidebarWidth-2, contentHeight-4-headerHeight)
		m.viewport.Width = mainWidth - 2
		m.viewport.Height = contentHeight - 4

	case []systemd.Unit:
		m.allUnits = msg    // Store source of truth
		m.updateListItems() // Apply filter
		cmds = append(cmds, cmd)

	case logLineMsg:
		if string(msg) != "" {
			m.logLines = append(m.logLines, string(msg))
			if len(m.logLines) > 1000 {
				m.logLines = m.logLines[len(m.logLines)-1000:]
			}
			if m.viewMode == ModeLogs {
				m.viewport.SetContent(strings.Join(m.logLines, "\n"))
				m.viewport.GotoBottom()
			}
		}
		cmds = append(cmds, waitForLogLine(m.logChan))

	case configMsg:
		m.configContent = string(msg)
		if m.viewMode == ModeConfig {
			m.viewport.SetContent(m.configContent)
			m.viewport.GotoTop()
		}

	case statsMsg:
		m.stats = msg

	case actionResultMsg:
		if msg.err != nil {
			m.statusMessage = "Error: " + msg.err.Error()
		} else {
			m.statusMessage = msg.action + " unit."
			cmds = append(cmds, fetchUnits)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) updateListItems() {
	var filtered []list.Item
	for _, unit := range m.allUnits {
		if m.devMode {
			name := strings.ToLower(unit.Name)
			isDev := false
			keywords := []string{"docker", "mongo", "postgres", "mysql", "redis", "nginx", "apache", "node", "python", "go", "java", "php", "ruby", "rust", "app", "api", "service", "web", "worker", "db"}

			for _, kw := range keywords {
				if strings.Contains(name, kw) {
					isDev = true
					break
				}
			}

			if isDev {
				filtered = append(filtered, item{unit: unit})
			}
		} else {
			filtered = append(filtered, item{unit: unit})
		}
	}
	m.list.SetItems(filtered)

	title := "System Units"
	if m.devMode {
		title = "Dev Services 🚀"
	}
	m.list.Title = title
}

func (m *model) startStreaming(name string) {
	if m.streamingUnit == name {
		return
	}
	if m.logCancel != nil {
		m.logCancel()
	}
	m.logLines = []string{}
	m.streamingUnit = name
	m.logCtx, m.logCancel = context.WithCancel(context.Background())
	m.logChan = make(chan string)

	go func() {
		systemd.StreamLogs(m.logCtx, name, m.logChan)
	}()
}

func (m model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// 1. DASHBOARD MODE (Keep Clean)
	if m.viewMode == ModeDashboard {
		logo := `
██╗   ██╗██╗ ██████╗ ██╗██╗     ██╗██╗  ██╗
██║   ██║██║██╔════╝ ██║██║     ██║╚██╗██╔╝
██║   ██║██║██║  ███╗██║██║     ██║ ╚███╔╝ 
╚██╗ ██╔╝██║██║   ██║██║██║     ██║ ██╔██╗ 
 ╚████╔╝ ██║╚██████╔╝██║███████╗██║██╔╝ ██╗
  ╚═══╝  ╚═╝ ╚═════╝ ╚═╝╚══════╝╚═╝╚═╝  ╚═╝
`
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center,
				lipgloss.NewStyle().Foreground(purple).Render(logo),
				lipgloss.NewStyle().Foreground(foreground).MarginTop(1).Render(fmt.Sprintf("Units: %d", len(m.allUnits))),
				lipgloss.NewStyle().Foreground(comment).MarginTop(2).Render("Press Enter to Start"),
			),
		)
	}

	// 2. MAIN APP
	contentHeight := m.height - 4
	contentWidth := m.width - 4
	sidebarWidth := int(float64(contentWidth) * 0.35)
	mainWidth := contentWidth - sidebarWidth

	// Sidebar
	sidebarStyle := panelStyle
	if m.activePane == PaneList {
		sidebarStyle = focusedPanelStyle
	}

	// Custom Header for List
	// We need to match the padding of itemDelegate
	// Delegate: PaddingLeft(2) for default, Selected has (1 + Border)
	// Badge: Right aligned.

	headerText := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#bd93f9")). // Purple
		PaddingLeft(2).
		Render("UNIT")

	statusText := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#bd93f9")).
		PaddingRight(1). // Match badge padding
		Render("STATUS")

	// Calculate spacer
	// sidebarWidth is total width of panel.
	// Content width inside panel is `sidebarWidth - 2`.
	listContentWidth := sidebarWidth - 2

	spacerWidth := listContentWidth - lipgloss.Width(headerText) - lipgloss.Width(statusText)
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	spacer := strings.Repeat(" ", spacerWidth)

	customHeader := fmt.Sprintf("%s%s%s", headerText, spacer, statusText)
	// Add a small separator line under header?
	// headerLine := strings.Repeat("─", listContentWidth)
	// Let's just keep it clean text for now, maybe underlined?
	customHeader = lipgloss.NewStyle().
		Width(listContentWidth).
		Border(lipgloss.Border{Bottom: "─"}, false, false, true, false).
		BorderForeground(comment).
		Render(customHeader)

	sidebarContent := lipgloss.JoinVertical(lipgloss.Left, customHeader, m.list.View())

	sidebar := sidebarStyle.
		Width(sidebarWidth).
		Height(contentHeight).
		Render(sidebarContent)

	// Main Panel Header
	logsTab := inactiveTabStyle.Render(" Logs ")
	configTab := inactiveTabStyle.Render(" Config ")

	if m.viewMode == ModeLogs {
		logsTab = activeTabStyle.Render(" Logs ")
	} else if m.viewMode == ModeConfig {
		configTab = activeTabStyle.Render(" Config ")
	}

	// Right Side Status
	headerInfo := ""
	if i, ok := m.list.SelectedItem().(item); ok {
		statusColor := comment
		statusFg := foreground
		if i.unit.ActiveState == "active" {
			statusColor = green
			statusFg = lipgloss.Color("#000000")
		} else if i.unit.ActiveState == "failed" {
			statusColor = red
		}

		statusStr := lipgloss.NewStyle().
			Background(statusColor).
			Foreground(statusFg).
			Padding(0, 1).
			Bold(true).
			Render(strings.ToUpper(i.unit.ActiveState))

		headerInfo = fmt.Sprintf(" %s ", statusStr)
	}

	if m.streamingUnit != "" && m.viewMode == ModeLogs {
		headerInfo += fmt.Sprintf(" %s", m.spinner.View())
	}

	// Separator line
	lineLen := mainWidth - lipgloss.Width(logsTab) - lipgloss.Width(configTab) - lipgloss.Width(headerInfo) - 4
	if lineLen < 0 {
		lineLen = 0
	}
	line := lipgloss.NewStyle().Foreground(comment).Render(strings.Repeat("─", lineLen))

	header := lipgloss.JoinHorizontal(lipgloss.Bottom,
		logsTab,
		configTab,
		line,
		headerInfo,
	)

	// Main Panel Content
	contentView := m.viewport.View()
	if m.viewport.TotalLineCount() == 0 {
		contentView = lipgloss.NewStyle().
			Foreground(comment).
			Align(lipgloss.Center).
			Width(mainWidth - 2).
			Render("No content loaded. Select a unit and press Enter.")
	}

	mainStyle := panelStyle
	if m.activePane == PaneContent {
		mainStyle = focusedPanelStyle
	}

	mainPanel := mainStyle.
		Width(mainWidth).
		Height(contentHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			header,
			contentView,
		))

	// Footer
	helpText := "Tab: Switch | d: Dev Mode | Enter: View | s/x/r: Control"
	statusView := lipgloss.NewStyle().Foreground(orange).Render(m.statusMessage)

	footer := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Foreground(comment).Render(helpText),
		lipgloss.NewStyle().PaddingLeft(2).Render("│ "+statusView),
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainPanel)

	return lipgloss.JoinVertical(lipgloss.Left, body, footer)
}

func fetchUnits() tea.Msg {
	units, err := systemd.ListUnits()
	if err != nil {
		return errMsg(err)
	}
	return units
}

func fetchConfig(name string) tea.Cmd {
	return func() tea.Msg {
		content, err := systemd.GetUnitFileContent(name)
		if err != nil {
			return configMsg("Error reading config: " + err.Error())
		}
		return configMsg(content)
	}
}

func fetchStats() tea.Msg {
	info, err := host.Info()
	if err != nil {
		return statsMsg{hostname: "Unknown", os: "Unknown"}
	}
	return statsMsg{
		hostname: info.Hostname,
		os:       info.Platform,
		uptime:   info.Uptime,
		kernel:   info.KernelVersion,
	}
}

func waitForLogLine(sub <-chan string) tea.Cmd {
	return func() tea.Msg {
		if sub == nil {
			return nil
		}
		line, ok := <-sub
		if !ok {
			return nil
		}
		return logLineMsg(line)
	}
}

func performAction(actionFunc func(string) error, name, actionName string) tea.Cmd {
	return func() tea.Msg {
		err := actionFunc(name)
		return actionResultMsg{err: err, action: actionName}
	}
}
