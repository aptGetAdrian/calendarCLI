package ui

import (
	"calendarCli/ui/models"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
)

// LoadMenuItems loads menu items from the JSON file based on menu type
func LoadMenuItems(menuType string) ([]list.Item, error) {
	// Get the path to menu.json (it's in the same ui directory)
	configPath := filepath.Join("ui", "menu.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("could not read menu file at %s: %w", configPath, err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("could not parse JSON: %w", err)
	}

	menuData, ok := config[menuType]
	if !ok {
		return nil, fmt.Errorf("menu type '%s' not found in config", menuType)
	}

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
		items[i] = models.MenuItem{
			TitleValue: menuItem.Title,
			Desc:       menuItem.Description,
			Action:     menuItem.Action,
		}
	}

	return items, nil
}
