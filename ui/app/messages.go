package app

import "calendarCli/ui"

type NavigateTo struct {
	Screen ui.Screen
}

type eventCreatedMsg struct{}

type menuItemHighlightedMsg struct {
	name string
}
