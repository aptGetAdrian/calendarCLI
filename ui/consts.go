package ui

//////////////////////////////////////
// menu type
//////////////////////////////////////

type Menu int

const (
	MainMenu Menu = iota
	SecondaryMenu
)

//////////////////////////////////////
// screen type
//////////////////////////////////////

type Screen int

const (
	MainMenuScreen Screen = iota
	ListEventsScreen
	CreateEventScreen
	AddBirthdayScreen
	TodoScreen
	NotebookScreen
)
