package main

import (
	calendar "calendarCli/internal"
	"calendarCli/internal/logger"
	"fmt"
	"log"
	"os"
	"path/filepath"

	ui "calendarCli/ui/app"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Setup logging
	logDir := "logs"
	logFile := filepath.Join(logDir, "app.log")

	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	logConfig := logger.Config{
		Level:      logger.Debug,
		FilePath:   logFile,
		UseConsole: false,
	}

	appLogger, err := logger.New(logConfig)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	defer appLogger.Close()

	appLogger.Info("Application starting")

	svc, err := calendar.NewService()
	if err != nil {
		fmt.Printf("%v", err)
	}

	p := tea.NewProgram(ui.New(svc, appLogger), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		appLogger.Fatal("Application failed: %v", err)
		os.Exit(1)
	}

	tea.ClearScreen()

}
