package app

import "calendarCli/ui"

type NavigateTo struct {
	Screen ui.Screen
}

type calendarSelectedMsg struct {
	calendarName string
}

type eventCreatedMsg struct{}

type menuItemHighlightedMsg struct {
	name string
}
