package app

import (
	calendar "calendarCli/internal"
	"calendarCli/ui"
	"fmt"
	"os"

	"calendarCli/ui/models"
	"calendarCli/ui/styles"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AppState struct {
	IsAuthenticated  bool
	CalendarCount    int
	EventCount       int
	SelectedCalendar string
	SelectedMenuItem string
}

type model struct {
	service  *calendar.Service
	list     list.Model
	docStyle lipgloss.Style
	state    AppState
}

func New(service *calendar.Service) tea.Model {
	items, err := ui.LoadMenuItems("main_menu_items")
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	l := BuildList("Main menu", items, ui.MainMenu)

	l.SetShowStatusBar(false)

	appState := setAppState(service)

	model := model{
		service:  service,
		list:     l,
		docStyle: styles.DocStyle,
		state:    appState,
	}

	return model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := m.docStyle.GetFrameSize()
		statusBarHeight := lipgloss.Height(m.buildStatusBar())

		listHeight := msg.Height - v - statusBarHeight
		m.list.SetSize(msg.Width-h, listHeight)

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "down", "k", "j":
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)

			selectedItem, ok := m.list.SelectedItem().(models.MenuItem)
			if ok {
				m.updateSelectedMenuItem(selectedItem.TitleValue)
			}

			return m, cmd
		case "enter":
			selectedItem, ok := m.list.SelectedItem().(models.MenuItem)
			if ok {
				switch selectedItem.Action {
				case "EXIT_APP":
					m.updateSelectedMenuItem("Exit")
					return m, tea.Quit
				case "SELECT_CALENDAR":
					m.updateSelectedMenuItem("Select calendar")

					// selected := m.service.SelectCalendar("Work") // hypothetical
					// m.state.SelectedCalendar = selected.Name

					// TODO: at the bottom line to the "Create calendar" option
					// m.state.CalendarCount = len(m.service.Calendars)
				case "LIST_EVENTS":
					m.updateSelectedMenuItem("List events")
				case "CREATE_EVENT":
					m.updateSelectedMenuItem("Create event")
				}

				// TODO: add other items
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m model) View() string {
	return m.docStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.list.View(),
			m.buildStatusBar(),
		),
	)
}
