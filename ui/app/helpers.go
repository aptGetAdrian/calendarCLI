package app

import (
	calendar "calendarCli/internal"
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m model) buildStatusLine() string {
	authText := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("Authenticated ✓")
	calCount := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render(fmt.Sprintf("%d calendars", m.state.CalendarCount))
	selected := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("Selected: " + m.state.SelectedCalendar)

	return fmt.Sprintf("%s • %s • %s", authText, calCount, selected)
}

func (m model) buildStatusBar() string {
	statusLine := m.buildStatusLine()

	return lipgloss.NewStyle().
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(m.list.Width()).
		Render(statusLine)
}

func setAppState(service *calendar.Service) AppState {
	if service != nil {
		return AppState{
			IsAuthenticated:  true,
			CalendarCount:    0, // TODO make a method in serivce to get calendar count
			EventCount:       0,
			SelectedCalendar: "Birthdays", // TODO make a method in serivce to get current selected calendar
		}
	} else {
		return AppState{
			IsAuthenticated:  false,
			CalendarCount:    0,
			EventCount:       0,
			SelectedCalendar: "None calendar selected",
		}
	}
}
