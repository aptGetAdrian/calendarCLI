package app

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) updateMainMenuStatus() {
	authText := "Not authenticated ✗"

	if m.service != nil {
		authText = "Authenticated ✓"
	}

	// TODO: make a method in serivce to get calendar count
	calendarCount := 4

	// TODO: make a method in serivce to get current selected calendar
	selectedCalendar := "Personal"

	status := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")). // green
		Render(authText) +
		" • " +
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")). // blue
			Render(fmt.Sprintf("%d calendars", calendarCount)) +
		" • " +
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")). // cyan
			Render("Selected: "+selectedCalendar)

	m.list.NewStatusMessage(status)

}
