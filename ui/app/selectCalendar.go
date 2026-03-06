package app

import (
	calendar "calendarCli/internal"
	"calendarCli/internal/logger"
	"calendarCli/ui"
	"calendarCli/ui/models"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type selectCalendarModel struct {
	service *calendar.Service
	list    list.Model
	logger  *logger.Logger
}

func newSelectCalendarModel(service *calendar.Service, state AppState, width, height int, logger *logger.Logger) *selectCalendarModel {
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

	if birthdayEvents, err := service.ExistBirthday(); err != nil {
		fmt.Println("Error checking birthdays:", err)
	} else if birthdayEvents {
		items = append(items, models.MenuItem{
			TitleValue: "Birthdays",
			Desc:       "Displays birthdays, anniversaries and other significant dates",
			Action:     "CHOOSE_CALENDAR",
		})
	}

	l := BuildList("Select Calendar", items, ui.MainMenu, width, height)
	l.SetShowStatusBar(false)

	return &selectCalendarModel{
		service: service,
		list:    l,
		logger:  logger,
	}
}

func (m *selectCalendarModel) Init() tea.Cmd { return nil }

func (m *selectCalendarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case sizedMsg:
		m.list.SetSize(msg.width, msg.height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {

		case "q":
			return m, func() tea.Msg { return NavigateTo{Screen: ui.MainMenuScreen} }

		case "enter":
			selected, ok := m.list.SelectedItem().(models.MenuItem)
			if !ok {
				return m, nil
			}

			var calendarName string

			if selected.TitleValue == "Birthdays" {
				calendarName = "Birthdays"
			} else {
				name, err := m.service.SelectCalendar(selected.TitleValue)
				if err != nil {
					fmt.Printf("Couldn't retrieve calendar name: %s\n", err)
					return m, nil
				}
				calendarName = name
			}

			// fire the message upward — root handles state update + navigation
			return m, func() tea.Msg {
				return calendarSelectedMsg{calendarName: calendarName}
			}

		case "up", "down", "k", "j":
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)

			if selected, ok := m.list.SelectedItem().(models.MenuItem); ok {
				return m, func() tea.Msg { return menuItemHighlightedMsg{name: selected.TitleValue} }
			}
			return m, cmd

		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *selectCalendarModel) View() string {
	return m.list.View()
}
