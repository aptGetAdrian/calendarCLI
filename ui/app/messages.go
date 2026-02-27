package app

import "calendarCli/ui"

type NavigateTo struct {
	Screen ui.Screen
}

type calendarSelectedMsg struct {
	calendarName string
}

type menuItemHighlightedMsg struct {
	name string
}
