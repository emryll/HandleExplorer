package tlist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
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

//*=====================[ Rendering ]=====================

func placeForm(width, height int, form string) string {
	if width <= 0 || height <= 0 {
		return form
	}

	formHeight := lipgloss.Height(form)

	vertical := lipgloss.Center

	if formHeight >= height {
		vertical = lipgloss.Top
	}

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		vertical,
		form,
	)
}

func (p *resultPicker[T]) renderBlankRow(bg lipgloss.Color) string {
	return lipgloss.NewStyle().
		Background(bg).
		Render(strings.Repeat(" ", p.boxWidth))
}

//*========================[ Utils ]=========================

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

func padRight(s string, width int) string {
	current := lipgloss.Width(s)

	if current >= width {
		return s
	}

	return s + strings.Repeat(" ", width-current)
}
func clampScrollOffset(cursor, total, visible int) int {
	if total <= visible {
		return 0
	}

	offset := cursor - visible/2
	if offset < 0 {
		offset = 0
	}

	maxOffset := total - visible
	if offset > maxOffset {
		offset = maxOffset
	}

	if cursor < offset {
		offset = cursor
	}

	if cursor >= offset+visible {
		offset = cursor - visible + 1
	}

	if offset < 0 {
		offset = 0
	}

	if offset > maxOffset {
		offset = maxOffset
	}

	return offset
}

var red = color.New(color.FgHiRed, color.Bold)

func PrintError(format string, a ...any) {
	red.Printf("[*] ")
	fmt.Printf(format, a...)
}
