package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var DocStyle = lipgloss.NewStyle().Margin(1, 2)

// Color palette
const (
	ColorSuccess = "10"  // Green
	ColorInfo    = "12"  // Blue
	ColorWarning = "14"  // Cyan
	ColorAccent  = "86"  // Bright cyan/mint
	ColorBorder  = "240" // Gray
	ColorError   = "9"   // Red
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
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0, 2).
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4"))
}

func SecondaryMenuTtitle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#b8b0d2")).
		Padding(0, 2).
		Bold(true).
		Foreground(lipgloss.Color("#caa1a1")).
		Background(lipgloss.Color("#9079d4"))
}
