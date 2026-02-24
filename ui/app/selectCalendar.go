package app

import (
	calendar "calendarCli/internal"
	"calendarCli/ui"
	"calendarCli/ui/models"
	"calendarCli/ui/styles"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type selectCalendarModel struct {
	service  *calendar.Service
	list     list.Model
	docStyle lipgloss.Style
	state    AppState
}

func NewSelectCalendarModel(service *calendar.Service, width, height int) tea.Model {
	calendars, err := service.GetAllCalendars()
	if err != nil {
		fmt.Printf("Couldn't retrieve calendar list: %s\n", err)
	}

	items := make([]list.Item, len(calendars.Items))
	for i, cal := range calendars.Items {
		items[i] = models.MenuItem{
			TitleValue: cal.Id,
			Desc:       cal.Summary,
			Action:     "CHOOSE_CALENDAR",
		}
	}

	l := BuildList("Select Calendar", items, ui.MainMenu)
	l.SetShowStatusBar(false)

	m := &selectCalendarModel{
		service:  service,
		list:     l,
		docStyle: styles.DocStyle,
		state:    setAppState(service),
	}

	if width > 0 && height > 0 {
		h, v := m.docStyle.GetFrameSize()
		statusBarHeight := lipgloss.Height(buildStatusBar(&m.state, width))
		listWidth := width - h
		listHeight := height - v - statusBarHeight
		m.list.SetSize(listWidth, listHeight)
	}

	return m
}
func (m *selectCalendarModel) Init() tea.Cmd {
	return tea.WindowSize()
}

func (m *selectCalendarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := m.docStyle.GetFrameSize()
		statusBarHeight := lipgloss.Height(buildStatusBar(&m.state, msg.Width))

		listWidth := msg.Width - h
		listHeight := msg.Height - v - statusBarHeight

		m.list.SetSize(listWidth, listHeight)

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {

		case "q":
			// go back to main menu
			return New(m.service), nil

		case "enter":
			selected, ok := m.list.SelectedItem().(models.MenuItem)
			if ok {
				selectedCalendar, err := m.service.SelectCalendar(selected.TitleValue)
				if err != nil {
					fmt.Printf("Couldn't retrieve calendar name: %s\n", err)
				}
				m.state.SelectedCalendar = selectedCalendar
				// go back to main menu
				newModel := New(m.service)
				return newModel, newModel.Init()
				// TODO: next up, add the option to also choose the birthdays "calendar"
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *selectCalendarModel) View() string {
	return m.docStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.list.View(),
			buildStatusBar(&m.state, m.list.Width()),
		),
	)
}
