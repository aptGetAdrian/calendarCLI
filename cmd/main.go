package main

import (
	calendar "calendarCli/internal"
	"fmt"
	"os"

	ui "calendarCli/ui/app"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	svc, err := calendar.NewService()
	if err != nil {
		fmt.Printf("%v", err)
	}

	p := tea.NewProgram(ui.New(svc), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	tea.ClearScreen()

}
