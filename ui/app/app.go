package app

import (
	"calendarCli/ui"
	"calendarCli/ui/models"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type mainMenuModel struct {
	state AppState
	list  list.Model
}

func newMainMenuModel(state AppState) *mainMenuModel {
	items, err := ui.LoadMenuItems("main_menu_items")
	if err != nil {
		fmt.Println("Error loading menu items:", err)
		os.Exit(1)
	}

	l := BuildList("Main menu", items, ui.MainMenu)
	l.SetShowStatusBar(false)

	return &mainMenuModel{
		state: state,
		list:  l,
	}
}

func (m *mainMenuModel) Init() tea.Cmd { return nil }

func (m *mainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case sizedMsg:
		m.list.SetSize(msg.width, msg.height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "enter":
			selected, ok := m.list.SelectedItem().(models.MenuItem)
			if !ok {
				break
			}
			switch selected.Action {
			case "EXIT_APP":
				return m, tea.Quit
			case "SELECT_CALENDAR":
				return m, func() tea.Msg { return NavigateTo{Screen: ui.SelectCalendarScreen} }
			case "LIST_EVENTS":
				return m, func() tea.Msg { return NavigateTo{Screen: ui.ListEventsScreen} }
			case "CREATE_EVENT":
				return m, func() tea.Msg { return NavigateTo{Screen: ui.CreateEventScreen} }
			}
			return m, nil

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

// View renders ONLY the list — no status bar, no docStyle wrapping
// The root model handles that.
func (m *mainMenuModel) View() string {
	return m.list.View()
}
