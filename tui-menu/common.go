package tmenu

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	"github.com/charmbracelet/lipgloss"
)

//*=======================[ Styles ]======================

var (
	colorAccent = lipgloss.Color("#89B4FA")
	colorGood   = lipgloss.Color("#A6E3A1")
	colorText   = lipgloss.Color("#CDD6F4")
	colorMuted  = lipgloss.Color("#6C7086")
	colorBg     = lipgloss.Color("#181825")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Background(colorBg)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Background(colorBg)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorMuted).
			Background(colorBg)

	sectionStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			BorderBackground(colorBg).
			Background(colorBg).
			Padding(0, 2)

	focusedSectionStyle = sectionStyle.
				BorderForeground(colorAccent)

	textStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBg)

	cursorGlyphStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Background(colorBg).
				Bold(true)

	checkedStyle = lipgloss.NewStyle().
			Foreground(colorGood).
			Background(colorBg)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Background(colorBg).
			MarginBackground(colorBg).
			MarginTop(1)
)

//*=========================[ Generic helpers ]=========================

func renderInput(ti *textinput.Model, width int) string {
	value := ti.Value()

	text := value
	color := colorText

	if text == "" {
		text = ti.Placeholder
		color = colorMuted
	}

	showCursor := ti.Focused()

	maxLen := width
	if showCursor {
		maxLen--
	}
	if maxLen < 0 {
		maxLen = 0
	}

	text = truncate(text, maxLen)

	line := textStyle.
		Foreground(color).
		Render(text)

	if showCursor {
		line += cursorGlyphStyle.
			Background(colorBg).
			Render("_")
	}

	pad := width - lipgloss.Width(line)
	if pad < 0 {
		pad = 0
	}

	line += lipgloss.NewStyle().
		Background(colorBg).
		Render(strings.Repeat(" ", pad))

	return line
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func placeForm(width int, height int, form string) string {
	if width <= 0 || height <= 0 {
		return form
	}

	formHeight := lipgloss.Height(form)

	vertical := lipgloss.Center

	// center when it fits, top align when doesnt
	// this avoids having the top go off screen
	if formHeight >= height {
		vertical = lipgloss.Top
	}

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		vertical,
		form,
		lipgloss.WithWhitespaceBackground(colorBg),
	)
}
