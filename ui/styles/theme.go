package styles

import (
	"calendarCli/ui"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var DocStyle = lipgloss.NewStyle().Margin(1, 2)

func MainMenuTtitle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0, 2).
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4"))
}

func SecondaryMenuTtitle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#b8b0d2")).
		Padding(0, 2).
		Bold(true).
		Foreground(lipgloss.Color("#caa1a1")).
		Background(lipgloss.Color("#9079d4"))
}

func BuildList(title string, items []list.Item, menu ui.Menu) list.Model {

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)

	l.Title = title

	switch menu {
	case ui.MainMenu:
		l.Styles.Title = MainMenuTtitle()
	case ui.SecondaryMenu:
		l.Styles.Title = SecondaryMenuTtitle()
	}

	return l
}
