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

	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = styles.ListItemTitle
	delegate.Styles.NormalDesc = styles.ListItemDesc
	delegate.Styles.SelectedTitle = styles.SelectedListItemTitle
	delegate.Styles.SelectedDesc = styles.SelectedListItemDesc

	l := list.New(items, delegate, 0, 0)

	l.Title = title

	switch menu {
	case ui.MainMenu:
		l.Styles.Title = styles.MainMenuTtitle()
	case ui.SecondaryMenu:
		l.Styles.Title = styles.SecondaryMenuTtitle()
	}

	return l
}

func buildStatusLine(state *AppState) string {
	calCount := ""
	authText := styles.SuccessText.Render("Authenticated ✓")

	if state.CalendarCount == 1 {
		calCount = styles.InfoText.Render(fmt.Sprintf("%d calendar", state.CalendarCount))
	} else {
		calCount = styles.InfoText.Render(fmt.Sprintf("%d calendars", state.CalendarCount))
	}

	selected := styles.WarningText.Render("Selected: " + state.SelectedCalendar)
	selectedItem := styles.AccentText.Render("Selected menu item: " + state.SelectedMenuItem)

	return fmt.Sprintf("%s • %s • %s • %s", authText, calCount, selected, selectedItem)
}

func buildStatusBar(state *AppState, width int) string {
	statusLine := buildStatusLine(state)

	return styles.StatusBarBorder.
		Width(width).
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
