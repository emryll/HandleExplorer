package tmenu

import (
	"strings"

	"charm.land/lipgloss/v2"
	tea "github.com/charmbracelet/bubbletea"
)

// Outer function for launching a TUI filter selection menu.
// The result is nil only if running the menu failed.
func ObjFilterSelectionMenu() *ObjectFilter {
	menu := tea.NewProgram(
		initialObjectFilterModel(),
		tea.WithAltScreen(),
	)
	finalModel, err := menu.Run()
	if err != nil {
		PrintError("Failed to launch filter selection menu: %v\n", err)
		return nil
	}
	m := finalModel.(*ObjectFilterModel)
	return m.result
}

//*==============================[ Model ]==============================

func (m *objectFilterModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(
		titleStyle.Render("HandleExplorer"),
	)
	b.WriteString("\n")

	b.WriteString(
		subtitleStyle.Render("Search Filters"),
	)
	b.WriteString("\n\n")

	b.WriteString(
		m.types.view(
			m.focus == objectFocusTypes,
		),
	)

	b.WriteString(
		m.renderTextSection(
			"Object Name",
			&m.objectName,
			objectFocusName,
		),
	)

	b.WriteString(
		m.renderTextSection(
			"Accessing Process",
			&m.process,
			objectFocusProcess,
		),
	)

	b.WriteString(
		m.renderTextSection(
			"Access Level",
			&m.accessLevel,
			objectFocusAccess,
		),
	)

	help := "type to filter types - tab switch field - arrows move - space toggle - enter search - esc cancel"

	if m.types.boxWidth > 4 {
		help = truncate(
			help,
			m.types.boxWidth-2,
		)
	}

	b.WriteString(
		helpStyle.Render(help),
	)

	form := placeForm(
		m.width,
		m.height,
		b.String(),
	)

	return lipgloss.NewStyle().
		Background(colorBg).
		Width(m.width).
		Height(m.height).
		Render(form)
}
