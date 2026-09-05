package tlist

import (
	"github.com/charmbracelet/lipgloss"
)

//*======================[ Styles ]=========================

var (
	colorBackground = lipgloss.Color("#11111B")
	colorPanel      = lipgloss.Color("#181825")

	colorAccent = lipgloss.Color("#89B4FA")
	colorText   = lipgloss.Color("#CDD6F4")
	colorMuted  = lipgloss.Color("#6C7086")

	colorNone = lipgloss.Color("")

	colorRowEven = lipgloss.Color("#1E1E2E")
	colorRowOdd  = lipgloss.Color("#252537")
	colorRowFg   = lipgloss.Color("#CDD6F4")

	colorSelectedBg = lipgloss.Color("#89B4FA")
	colorSelectedFg = lipgloss.Color("#1E1E2E")

	columnColors = []lipgloss.Color{
		lipgloss.Color("#89B4FA"),
		lipgloss.Color("#A6E3A1"),
		lipgloss.Color("#94E2D5"),
		lipgloss.Color("#B4BEFE"),
		lipgloss.Color("#F5C2E7"),
	}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// IMPORTANT:
	//
	// Do not put horizontal padding on the section.
	//
	// The table renderer below owns the one-cell inset. That means
	// selected/zebra backgrounds can paint through the inset instead
	// of stopping one cell short of the border.
	sectionStyle = lipgloss.NewStyle().
			Background(colorPanel).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			BorderBackground(colorPanel)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	scrollStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	emptyStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)
)
