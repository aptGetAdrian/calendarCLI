package app

import (
	calendar "calendarCli/internal"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

func (m *model) buildStatusLine() string {
	authText := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("Authenticated ✓")
	calCount := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render(fmt.Sprintf("%d calendars", m.state.CalendarCount))
	selected := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("Selected: " + m.state.SelectedCalendar)
	selectedItem := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("Selected menu item: " + m.state.SelectedMenuItem)

	return fmt.Sprintf("%s • %s • %s • %s", authText, calCount, selected, selectedItem)
}

func (m *model) buildStatusBar() string {
	statusLine := m.buildStatusLine()

	return lipgloss.NewStyle().
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(m.list.Width()).
		Render(statusLine)
}

func (m *model) updateSelectedMenuItem(name string) {
	m.state.SelectedMenuItem = fmt.Sprintf("\"%s\"", name)
}

func setAppState(service *calendar.Service) AppState {
	if service != nil {
		return AppState{
			IsAuthenticated:  true,
			CalendarCount:    0, // TODO make a method in serivce to get calendar count
			EventCount:       0,
			SelectedCalendar: "Birthdays", // TODO make a method in serivce to get current selected calendar
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

func loadMenuItems(menuType string) ([]list.Item, error) {
	configPath := filepath.Join("ui", "menu.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	menuData, ok := config[menuType]
	if !ok {
		return nil, fmt.Errorf("menu type '%s' not found in config", menuType)
	}

	// Parse the specific menu items
	var menuItems []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Action      string `json:"action"`
	}

	if err := json.Unmarshal(menuData, &menuItems); err != nil {
		return nil, err
	}

	items := make([]list.Item, len(menuItems))
	for i, menuItem := range menuItems {
		items[i] = item{
			TitleValue: menuItem.Title,
			Desc:       menuItem.Description,
			Action:     menuItem.Action,
		}
	}

	return items, nil
}
