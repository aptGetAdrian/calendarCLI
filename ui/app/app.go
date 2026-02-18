package app

import (
	calendar "calendarCli/internal"

	"calendarCli/ui/styles"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type item struct {
	title, desc string
}

type model struct {
	service  *calendar.Service
	list     list.Model
	docStyle lipgloss.Style
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func New(service *calendar.Service) tea.Model {
	items := []list.Item{
		item{title: "Choose calendar", desc: "Choose a calendar from your Google calendars.\nThe default one is the primary calendar."},
		item{title: "List events", desc: "List all events in your chosen calendar."},
		item{title: "Create event", desc: "Create a new event for your chosen calendar."},
		item{title: "Exit", desc: "Close the application"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Main menu"

	return model{
		service:  service,
		list:     l,
		docStyle: styles.DocStyle,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := m.docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			selectedItem, ok := m.list.SelectedItem().(item)
			if ok {
				switch selectedItem.title {
				case "Exit":
					return m, tea.Quit
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return m.docStyle.Render(m.list.View())
}
