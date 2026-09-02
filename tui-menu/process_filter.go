package tmenu

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	tea "github.com/charmbracelet/bubbletea"
)

// Outer function for launching a TUI filter selection menu.
// The result is nil only if running the menu failed.
func PsFilterSelectionMenu() *ProcessFilter {
	menu := tea.NewProgram(
		initialProcessFilterModel(),
		tea.WithAltScreen(),
	)

	finalModel, err := menu.Run()
	if err != nil {
		PrintError("Failed to launch process filter selection menu: %v\n", err)
		return nil
	}

	m := finalModel.(*ProcessFilterModel)
	return m.result
}

//*=======================[ Initial Model ]===========================

func initialProcessFilterModel() *processModel {
	path := textinput.New()
	path.Placeholder = `e.g. C:\Windows\System32\explorer.exe`
	path.CharLimit = 512
	path.Prompt = ""
	styleTextInput(&path)

	allowlistInput := textinput.New()
	allowlistInput.Placeholder = `e.g. C:\Windows\ or %appdata%`
	allowlistInput.CharLimit = 512
	allowlistInput.Prompt = ""
	styleTextInput(&allowlistInput)

	parentInput := textinput.New()
	parentInput.Placeholder = `e.g. svchost.exe or a PID`
	parentInput.CharLimit = 512
	parentInput.Prompt = ""
	styleTextInput(&parentInput)

	m := &processModel{
		focus: processFocusPath,

		path:           &path,
		allowlistInput: &allowlistInput,
		parentInput:    &parentInput,

		objectTypes:   ntObjectTypes,
		selectedTypes: make(map[int]bool),

		typeCols:        3,
		typeCellWidth:   processTypeMinCellWidth,
		typeVisibleRows: processMinVisibleRows,
		typeBoxWidth:    3 * processTypeMinCellWidth,
		typeGridWidth:   3 * processTypeMinCellWidth,
	}

	m.syncFocus()

	return m
}

func (m *processModel) Init() tea.Cmd {
	return textinput.Blink
}

//*========================[ Model View ]=======================

func (m *processModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	b.WriteString(
		titleStyle.Render("HandleExplorer"),
	)

	b.WriteString("\n")

	b.WriteString(
		subtitleStyle.Render("Process Search Filters"),
	)

	b.WriteString("\n\n")

	b.WriteString(
		m.renderProcessTextSection(
			"Path",
			m.path,
			processFocusPath,
		),
	)

	b.WriteString(
		m.renderListSection(
			"Directory Filter",
			m.allowlistInput,
			len(m.allowlist),
			processFocusAllowlist,
		),
	)

	b.WriteString(
		m.renderListSection(
			"Parent Process",
			m.parentInput,
			len(m.parent),
			processFocusParent,
		),
	)

	b.WriteString(
		m.renderPropertiesSection(),
	)

	b.WriteString(
		m.renderObjectTypeSection(),
	)

	help :=
		"tab switch field - arrows move - space toggle - enter add/search - esc cancel"

	helpWidth := m.typeBoxWidth

	if helpWidth > 4 {
		help = truncate(
			help,
			helpWidth-2,
		)
	}

	b.WriteString(
		helpStyle.Render(help),
	)

	form := b.String()
	form = placeForm(
		m.width,
		m.height,
		form,
	)

	return lipgloss.NewStyle().
		Background(colorBg).
		Width(m.width).
		Height(m.height).
		Render(form)
}
