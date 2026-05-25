package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var DocStyle = lipgloss.NewStyle().Margin(1, 2)

// Color palette
const (
	// Semantic colors - classic terminal
	ColorSuccess = "2" // Classic terminal green
	ColorInfo    = "4" // Classic terminal blue
	ColorWarning = "3" // Classic terminal yellow
	ColorAccent  = "5" // Classic terminal magenta
	ColorBorder  = "7" // Classic terminal white (dim)
	ColorError   = "1" // Classic terminal red

	// Main menu - Mellow amber (like aged paper/phosphor)
	ColorTitleBg     = "#fff4e4" // Warm cream
	ColorTitleFg     = "#c96e1c" // Warm amber (muted)
	ColorTitleBorder = "#e6c9a8" // Light tan

	ColorSecondaryBg     = "#e8f3e8" // Soft mint cream
	ColorSecondaryFg     = "#3b7a57" // Soft forest
	ColorSecondaryBorder = "#b8d9b8" // Light moss

	// List Events - cool periwinkle blue
	ColorListEventsBg     = "#e6eeff"
	ColorListEventsFg     = "#2a55b0"
	ColorListEventsBorder = "#a0b8e8"

	// Create Event - warm violet / plum
	ColorCreateEventBg     = "#f2eaff"
	ColorCreateEventFg     = "#6b28b8"
	ColorCreateEventBorder = "#c8a8e8"

	// Add Birthday - blush rose
	ColorBirthdayBg     = "#fff0f2"
	ColorBirthdayFg     = "#b83060"
	ColorBirthdayBorder = "#e8aac0"

	// Todo - seafoam teal
	ColorTodoBg     = "#e2f5f4"
	ColorTodoFg     = "#1d7a78"
	ColorTodoBorder = "#8ecece"

	// Pure terminal black/white
	ColorBlack = "#000000"
	ColorWhite = "#ffffff"
)

var (
	SuccessText = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess))
	InfoText    = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorInfo))
	AccentText  = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorAccent))

	StatusBarBorder = lipgloss.NewStyle().
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(ColorBorder))
)

func MainMenuTtitle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorTitleBorder)).
		Padding(0, 2).
		Bold(true).
		Foreground(lipgloss.Color(ColorTitleFg)).
		Background(lipgloss.Color(ColorTitleBg))
}

func SecondaryMenuTtitle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorSecondaryBorder)).
		Padding(0, 2).
		Bold(true).
		Foreground(lipgloss.Color(ColorSecondaryFg)).
		Background(lipgloss.Color(ColorSecondaryBg))
}

func ListEventsTtitle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorListEventsBorder)).
		Padding(0, 2).Bold(true).
		Foreground(lipgloss.Color(ColorListEventsFg)).
		Background(lipgloss.Color(ColorListEventsBg))
}

func CreateEventTtitle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorCreateEventBorder)).
		Padding(0, 2).Bold(true).
		Foreground(lipgloss.Color(ColorCreateEventFg)).
		Background(lipgloss.Color(ColorCreateEventBg))
}

func AddBirthdayTtitle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorBirthdayBorder)).
		Padding(0, 2).Bold(true).
		Foreground(lipgloss.Color(ColorBirthdayFg)).
		Background(lipgloss.Color(ColorBirthdayBg))
}

func TodoTtitle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorTodoBorder)).
		Padding(0, 2).Bold(true).
		Foreground(lipgloss.Color(ColorTodoFg)).
		Background(lipgloss.Color(ColorTodoBg))
}

var (
	ListItemTitle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color(ColorInfo))

	ListItemDesc = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color(ColorBorder))

	// Selected list item
	SelectedListItemTitle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(lipgloss.Color(ColorAccent)).
				Bold(true).
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color(ColorAccent))

	SelectedListItemDesc = SelectedListItemTitle
)
