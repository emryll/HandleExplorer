package tmenu

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
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

func initialObjectFilterModel() *objectFilterModel {
	objectName := textinput.New()
	objectName.Placeholder =
		`e.g. \Sessions\1\BaseNamedObjects\MyMutex`
	objectName.CharLimit = 256
	objectName.Prompt = ""
	styleTextInput(&objectName)

	process := textinput.New()
	process.Placeholder =
		"e.g. explorer.exe or a PID"
	process.CharLimit = 128
	process.Prompt = ""
	styleTextInput(&process)

	accessLevel := textinput.New()
	accessLevel.Placeholder =
		"e.g. 0x1F0001 or GENERIC_READ"
	accessLevel.CharLimit = 64
	accessLevel.Prompt = ""
	styleTextInput(&accessLevel)

	m := &objectFilterModel{
		focus: objectFocusTypes,

		types: newObjectTypePicker(),

		objectName:  objectName,
		process:     process,
		accessLevel: accessLevel,
	}

	m.syncFocus()

	return m
}

func (m *objectFilterModel) Init() tea.Cmd {
	return textinput.Blink
}

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

func (m *objectFilterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.types.recalcLayout(
			msg.Width,
			msg.Height,
			30,
		)

		m.recalcInputs()

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "esc":
			if m.focus == objectFocusTypes && m.types.filter != "" {
				m.types.filter = ""
				m.types.cursor = 0
				return m, nil
			}

			m.quitting = true
			return m, tea.Quit

		case "tab":
			m.focus = (m.focus + 1) % objectFocusCount
			m.syncFocus()
			return m, nil

		case "shift+tab":
			m.focus =
				(m.focus - 1 + objectFocusCount) % objectFocusCount

			m.syncFocus()
			return m, nil

		case "enter":
			m.submitted = true
			m.result = m.buildFilter()
			m.quitting = true

			return m, tea.Quit
		}

		switch m.focus {
		case objectFocusTypes:
			m.types.update(msg)
			return m, nil

		case objectFocusName:
			var cmd tea.Cmd
			m.objectName, cmd = m.objectName.Update(msg)
			return m, cmd

		case objectFocusProcess:
			var cmd tea.Cmd
			m.process, cmd = m.process.Update(msg)
			return m, cmd

		case objectFocusAccess:
			var cmd tea.Cmd
			m.accessLevel, cmd = m.accessLevel.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

//*======================[ Rendering fields ]==========================

func (m *objectFilterModel) renderTextSection(label string, ti *textinput.Model, field objectFocusField) string {
	style := sectionStyle

	if m.focus == field {
		style = focusedSectionStyle
	}

	style = style.Width(m.types.boxWidth)

	content :=
		labelStyle.Render(label) + "\n" +
			renderInput(ti, m.types.gridWidth)

	return style.Render(content) + "\n\n"
}
