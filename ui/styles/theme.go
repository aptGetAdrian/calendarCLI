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

	// Commodore 64 (blue) for third menu if needed
	ColorTertiaryBg     = "#00002a" // Deep blue-black
	ColorTertiaryFg     = "#5fafff" // Light blue
	ColorTertiaryBorder = "#1f3f7f" // Navy blue

	// Pure terminal black/white
	ColorBlack = "#000000"
	ColorWhite = "#ffffff"
)

var (
	SuccessText = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess))
	InfoText    = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorInfo))
	WarningText = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning))
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

var (
	// Normal list item
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

	SelectedListItemDesc = SelectedListItemTitle.Copy() // Same style or customize differently
)
