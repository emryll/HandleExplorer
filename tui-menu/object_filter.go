package tmenu

import (
	"fmt"
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

func (p *objectTypePicker) update(msg tea.KeyMsg) {
	visible := p.visibleIndices()

	if len(visible) == 0 {
		p.cursor = 0
	} else if p.cursor >= len(visible) {
		p.cursor = len(visible) - 1
	}

	switch msg.String() {
	case "up":
		if p.cursor-p.cols >= 0 {
			p.cursor -= p.cols
		}

	case "down":
		if p.cursor+p.cols < len(visible) {
			p.cursor += p.cols
		}

	case "left":
		if p.cursor%p.cols != 0 {
			p.cursor--
		}

	case "right":
		if p.cursor%p.cols != p.cols-1 &&
			p.cursor+1 < len(visible) {
			p.cursor++
		}

	case " ":
		if len(visible) > 0 {
			realIndex := visible[p.cursor]
			p.selected[realIndex] = !p.selected[realIndex]
		}

	case "backspace":
		if len(p.filter) > 0 {
			runes := []rune(p.filter)
			p.filter = string(runes[:len(runes)-1])
			p.cursor = 0
		}

	default:
		if msg.Type == tea.KeyRunes {
			p.filter += string(msg.Runes)
			p.cursor = 0
		}
	}
}

func (p *objectTypePicker) view(focused bool) string {
	style := sectionStyle

	if focused {
		style = focusedSectionStyle
	}

	style = style.Width(p.boxWidth)

	visible := p.visibleIndices()
	totalRows := 0

	if len(visible) > 0 {
		totalRows = (len(visible) + p.cols - 1) / p.cols
	}

	currentRow := 0

	if len(visible) > 0 {
		currentRow = p.cursor / p.cols
	}

	visibleRows := p.visibleRows
	scrollOffset := 0

	if totalRows > visibleRows {
		scrollOffset = currentRow - visibleRows/2

		if scrollOffset < 0 {
			scrollOffset = 0
		}
	}

	maxOffset := totalRows - visibleRows

	if maxOffset < 0 {
		maxOffset = 0
	}

	if scrollOffset > maxOffset {
		scrollOffset = maxOffset
	}

	if currentRow >= scrollOffset+visibleRows {
		scrollOffset =
			currentRow - visibleRows + 1
	}

	if scrollOffset < 0 {
		scrollOffset = 0
	}

	if scrollOffset > maxOffset {
		scrollOffset = maxOffset
	}

	startRow := scrollOffset
	endRow := scrollOffset + visibleRows

	if endRow > totalRows {
		endRow = totalRows
	}

	var b strings.Builder

	header :=
		labelStyle.Render("Object Type") +
			textStyle.Render("  ") +
			labelStyle.Render(
				fmt.Sprintf(
					"space toggle - %d selected",
					p.selectedCount(),
				),
			)

	b.WriteString(header)
	b.WriteString("\n")

	if len(visible) > 0 && focused {
		realIndex := visible[p.cursor]

		b.WriteString(labelStyle.Render(">> "))

		previewWidth := p.gridWidth - 3

		if previewWidth < 1 {
			previewWidth = 1
		}

		b.WriteString(
			textStyle.Render(
				truncate(
					p.objectTypes[realIndex],
					previewWidth,
				),
			),
		)
	}

	b.WriteString("\n\n")

	if len(visible) == 0 {
		b.WriteString(
			lipgloss.NewStyle().
				Foreground(colorMuted).
				Italic(true).
				Render(
					fmt.Sprintf(
						"  no types match %q",
						p.filter,
					),
				),
		)
		b.WriteString("\n")
	} else {
		for row := startRow; row < endRow; row++ {
			start := row * p.cols
			end := start + p.cols

			if end > len(visible) {
				end = len(visible)
			}

			var cells []string

			for pos := start; pos < end; pos++ {
				cells = append(
					cells,
					p.renderCell(
						visible[pos],
						pos,
					),
				)
			}

			b.WriteString(
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					cells...,
				),
			)

			b.WriteString("\n")
		}

		actualRows := endRow - startRow

		for i := actualRows; i < visibleRows; i++ {
			b.WriteString("\n")
		}
	}

	scrollStyle := lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	remaining := totalRows - endRow

	if scrollOffset > 0 {
		b.WriteString(
			scrollStyle.Render(
				fmt.Sprintf(
					"  ^ %d more above (up arrow)",
					scrollOffset*p.cols,
				),
			),
		)
	}
	b.WriteString("\n")

	if remaining > 0 {
		b.WriteString(
			scrollStyle.Render(
				fmt.Sprintf(
					"  v %d more below (down arrow)",
					remaining*p.cols,
				),
			),
		)
	}
	b.WriteString("\n")

	b.WriteString("\n")

	filterLine := labelStyle.Render("Type to filter: ")

	if p.filter != "" {
		filterLine += titleStyle.Render(p.filter)
		filterLine += titleStyle.Render("|")
	} else if focused {
		filterLine += titleStyle.Render("_")
	}

	b.WriteString(filterLine)

	return style.Render(
		strings.TrimRight(
			b.String(),
			"\n",
		),
	) + "\n\n"
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
