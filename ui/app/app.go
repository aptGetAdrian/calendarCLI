package app

import (
	calendar "calendarCli/internal"
	"calendarCli/ui"

	"calendarCli/ui/styles"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type item struct {
	Title  string `json:"title"`
	Desc   string `json:"description"`
	Action string `json:"action"`
}

type AppState struct {
	IsAuthenticated  bool
	CalendarCount    int
	EventCount       int
	SelectedCalendar string
}

type model struct {
	service  *calendar.Service
	list     list.Model
	docStyle lipgloss.Style
	state    AppState
}

func (i item) TitleValue() string  { return i.Title }
func (i item) Description() string { return i.Desc }
func (i item) ActionValue() string { return i.Action }
func (i item) FilterValue() string { return i.Title }

func New(service *calendar.Service) tea.Model {
	items := []list.Item{
		item{Title: "Choose calendar", Desc: "Choose a calendar from your Google calendars.\nThe default one is the primary calendar.", Action: "SELECT_CALENDAR"},
		item{Title: "List events", Desc: "List all events in your chosen calendar.", Action: "LIST_EVENTS"},
		item{Title: "Create event", Desc: "Create a new event for your chosen calendar.", Action: "CREATE_EVENT"},
		item{Title: "Exit", Desc: "Close the application", Action: "EXIT_APP"},
	}

	l := styles.BuildList("Main menu", items, ui.MainMenu)

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
		case "enter":
			selectedItem, ok := m.list.SelectedItem().(item)
			if ok {
				switch selectedItem.Action {
				case "EXIT_APP":
					return m, tea.Quit
				case "SELECT_CALENDAR":

					// selected := m.service.SelectCalendar("Work") // hypothetical
					// m.state.SelectedCalendar = selected.Name

					// TODO: at the bottom line to the "Create calendar" option
					// m.state.CalendarCount = len(m.service.Calendars)
				case "LIST_EVENTS":
				case "CCREATE_EVENT":
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
