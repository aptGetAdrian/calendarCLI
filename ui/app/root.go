package app

import (
	calendar "calendarCli/internal"
	"calendarCli/internal/logger"
	"calendarCli/ui"
	"calendarCli/ui/styles"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen string

const (
	screenMainMenu       ui.Screen = ui.MainMenuScreen
	screenSelectCalendar ui.Screen = ui.SelectCalendarScreen
	screenListEvents     ui.Screen = ui.ListEventsScreen
	screenCreateEvent    ui.Screen = ui.CreateEventScreen
)

type AppState struct {
	IsAuthenticated  bool
	CalendarCount    int
	EventCount       int
	SelectedCalendar string
	SelectedMenuItem string
}

type sizedMsg struct {
	width  int
	height int
}

type RootModel struct {
	service      *calendar.Service
	state        AppState
	activeScreen ui.Screen
	child        tea.Model
	termWidth    int
	termHeight   int
	docStyle     lipgloss.Style
}

func New(service *calendar.Service, logger *logger.Logger) tea.Model {
	state := setAppState(service)

	h, v := styles.DocStyle.GetFrameSize()

	child := newMainMenuModel(state, h, v)

	return &RootModel{
		service:      service,
		state:        state,
		activeScreen: screenMainMenu,
		child:        child,
		docStyle:     styles.DocStyle,
	}
}

func (m *RootModel) Init() tea.Cmd {
	return tea.Batch(tea.WindowSize(), m.child.Init())
}

func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		child, cmd := m.child.Update(sizedMsg{width: m.contentWidth(), height: m.contentHeight()})
		m.child = child
		return m, cmd

	case NavigateTo:
		return m.handleNavigation(msg)

	case calendarSelectedMsg:
		m.state.SelectedCalendar = msg.calendarName
		child := newMainMenuModel(m.state, m.contentWidth(), m.contentHeight())
		m.activeScreen = screenMainMenu
		m.child = child
		return m, child.Init()
	case eventCreatedMsg:
		// event saved — go back to main menu
		child := newMainMenuModel(m.state, m.contentWidth(), m.contentHeight())
		m.activeScreen = screenMainMenu
		m.child = child
		return m, child.Init()

	case menuItemHighlightedMsg:
		m.state.SelectedMenuItem = fmt.Sprintf("%q", msg.name)
		return m, nil
	}

	// delegate everything else to the child
	child, cmd := m.child.Update(msg)
	m.child = child
	return m, cmd
}

func (m *RootModel) View() string {
	statusBar := buildStatusBar(&m.state, m.contentWidth())

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.child.View(),
		statusBar,
	)

	return m.docStyle.Render(content)
}
