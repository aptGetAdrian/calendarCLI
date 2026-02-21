package app

import (
	calendar "calendarCli/internal"
	"calendarCli/ui"
	"calendarCli/ui/styles"
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/list"
)

func BuildList(title string, items []list.Item, menu ui.Menu) list.Model {

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)

	l.Title = title

	switch menu {
	case ui.MainMenu:
		l.Styles.Title = styles.MainMenuTtitle()
	case ui.SecondaryMenu:
		l.Styles.Title = styles.SecondaryMenuTtitle()
	}

	return l
}

func (m *model) buildStatusLine() string {
	calCount := ""
	authText := styles.SuccessText.Render("Authenticated ✓")

	if m.state.CalendarCount == 1 {
		calCount = styles.InfoText.Render(fmt.Sprintf("%d calendar", m.state.CalendarCount))
	} else {
		calCount = styles.InfoText.Render(fmt.Sprintf("%d calendars", m.state.CalendarCount))
	}

	selected := styles.WarningText.Render("Selected: " + m.state.SelectedCalendar)
	selectedItem := styles.AccentText.Render("Selected menu item: " + m.state.SelectedMenuItem)

	return fmt.Sprintf("%s • %s • %s • %s", authText, calCount, selected, selectedItem)
}

func (m *model) buildStatusBar() string {
	statusLine := m.buildStatusLine()

	return styles.StatusBarBorder.
		Width(m.list.Width()).
		Render(statusLine)
}

func (m *model) updateSelectedMenuItem(name string) {
	m.state.SelectedMenuItem = fmt.Sprintf("\"%s\"", name)
}

func setAppState(service *calendar.Service) AppState {
	calendarCount, err := service.GetNumCalendars()
	if err != nil {
		log.Printf("Failed to get list of calendars: %v", err)
		calendarCount = 0
	}

	selectedCalendar, err := service.GetPrimaryCalendar()
	if err != nil {
		log.Printf("Failed to get primary calendar: %v", err)
		selectedCalendar = ""
	}

	if service != nil {
		return AppState{
			IsAuthenticated:  true,
			CalendarCount:    calendarCount,
			EventCount:       0,
			SelectedCalendar: selectedCalendar,
			SelectedMenuItem: "\"Select calendar\"",
		}
	} else {
		return AppState{
			IsAuthenticated:  false,
			CalendarCount:    0,
			EventCount:       0,
			SelectedCalendar: "None calendar selected",
			SelectedMenuItem: "\"Select calendar\"",
		}
	}
}
