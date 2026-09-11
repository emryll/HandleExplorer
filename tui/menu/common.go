package tmenu

import (
	"HandleExplorer/utils"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
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

func truncate(s string, width int) string {
	const suffix = "..."

	r := []rune(s)

	if len(r) <= width {
		return s
	}

	if width <= len(suffix) {
		if width < 0 {
			width = 0
		}

		return string(r[:width])
	}

	return string(r[:width-len(suffix)]) + suffix
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}

	return s
}

func formatList(values []string) string {
	if len(values) == 0 {
		return "any"
	}

	return strings.Join(values, ", ")
}

func padRight(s string, width int) string {
	current := lipgloss.Width(s)
	if current >= width {
		return s
	}
	return s + textStyle.Render(strings.Repeat(" ", width-current))
}

func joinColumns(left string, leftWidth int, right string, rightWidth int) string {
	left = truncate(left, leftWidth)
	right = truncate(right, rightWidth)

	left = padRight(
		left,
		leftWidth,
	)

	return left + right
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

// Take in raw string access mask input,
// normalize the string into a mask value.
func parseAccessField(strMask string) utils.Bitmask {
	// If it starts with 0x its surely a raw hex value
	if strings.HasPrefix(strMask, "0x") {
		if val, err := strconv.ParseUint(strMask[2:], 16, 64); err == nil {
			return (utils.Bitmask)(val)
		} else {
			return 0
		}
	}

	// Check if its a raw decimal value
	if val, err := strconv.ParseUint(strMask, 10, 64); err == nil {
		return (utils.Bitmask)(val)
	}
	// Check if its a raw hex value (without the 0x)
	if val, err := strconv.ParseUint(strMask, 16, 64); err == nil {
		return (utils.Bitmask)(val)
	}

	var mask utils.Bitmask
	for _, flag := range strings.Split(strMask, "|") {
		flag = strings.TrimSpace(flag)
		mask |= utils.GetEnumValue(flag)
	}
	return mask
}
