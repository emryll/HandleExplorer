package tmenu

import "github.com/charmbracelet/lipgloss"

//*=========================[ Generic helpers ]=========================

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
