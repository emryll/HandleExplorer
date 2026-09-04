package tmenu

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	m := finalModel.(*processModel)
	return m.result
}

//*=======================[ Initial Model ]===========================

func initialProcessFilterModel() *processModel {
	path := textinput.New()
	path.Placeholder = `e.g. C:\Windows\System32\explorer.exe`
	path.CharLimit = 512
	path.Prompt = ""

	allowlistInput := textinput.New()
	allowlistInput.Placeholder = `e.g. C:\Windows\ or %appdata%`
	allowlistInput.CharLimit = 512
	allowlistInput.Prompt = ""

	parentInput := textinput.New()
	parentInput.Placeholder = `e.g. svchost.exe or a PID`
	parentInput.CharLimit = 512
	parentInput.Prompt = ""

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

//*=========================[ Model Update ]=========================

func (m *processModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.recalcLayout()

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "esc":
			if m.focus == processFocusObjectTypes && m.typeFilter != "" {
				m.typeFilter = ""
				m.typeCursor = 0
				return m, nil
			}

			m.quitting = true
			return m, tea.Quit

		case "tab":
			m.focus = (m.focus + 1) % processFocusFieldCount
			m.syncFocus()
			return m, nil

		case "shift+tab":
			m.focus =
				(m.focus - 1 + processFocusFieldCount) %
					processFocusFieldCount

			m.syncFocus()

			return m, nil

		case "enter":
			switch m.focus {
			case processFocusAllowlist:
				m.addAllowlistEntry()
				return m, nil

			case processFocusParent:
				m.addParentEntry()
				return m, nil

			default:
				m.submitted = true
				m.result = m.buildFilter()
				m.quitting = true

				return m, tea.Quit
			}
		}

		switch m.focus {
		case processFocusProperties:
			m.updateProperties(&msg)
			return m, nil

		case processFocusObjectTypes:
			m.updateObjectTypes(&msg)
			return m, nil

		case processFocusPath:
			var (
				cmd     tea.Cmd
				updated textinput.Model
			)

			updated, cmd = m.path.Update(msg)
			m.path = &updated
			return m, cmd

		case processFocusAllowlist:
			var (
				cmd     tea.Cmd
				updated textinput.Model
			)

			updated, cmd =
				m.allowlistInput.Update(msg)
			m.allowlistInput = &updated
			return m, cmd

		case processFocusParent:
			var (
				cmd     tea.Cmd
				updated textinput.Model
			)

			updated, cmd =
				m.parentInput.Update(msg)
			m.parentInput = &updated

			return m, cmd
		}
	}

	return m, nil
}

//*===========================[ Render fields ]==================================

func (m *processModel) renderObjectTypeSection() string {
	style := sectionStyle

	if m.focus == processFocusObjectTypes {
		style = focusedSectionStyle
	}

	style = style.Width(m.typeBoxWidth)

	selectedCount := 0

	for _, selected := range m.selectedTypes {
		if selected {
			selectedCount++
		}
	}

	header :=
		labelStyle.Render("Object Types Accessed") +
			textStyle.Render("  ") +
			subtitleStyle.Render(
				fmt.Sprintf(
					"%d selected",
					selectedCount,
				),
			)

	filterLine := labelStyle.Render("Type to filter: ")

	if m.typeFilter != "" {
		filterWidth := m.typeGridWidth - 16

		if filterWidth < 1 {
			filterWidth = 1
		}

		filterText :=
			truncate(
				m.typeFilter,
				filterWidth,
			)

		filterLine +=
			titleStyle.Render(filterText)

		filterLine +=
			titleStyle.Render("|")
	} else if m.focus == processFocusObjectTypes {
		filterLine +=
			titleStyle.Render("_")
	}

	visible := m.visibleTypeIndices()
	cols := m.typeCols

	if cols < 1 {
		cols = 1
	}

	totalRows := 0

	if len(visible) > 0 {
		totalRows =
			(len(visible) + cols - 1) / cols
	}

	visibleRows := m.typeVisibleRows
	if visibleRows < 1 {
		visibleRows = 1
	}

	currentRow := 0
	if len(visible) > 0 {
		currentRow = m.typeCursor / cols
	}

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

	if currentRow < scrollOffset {
		scrollOffset = currentRow
	}

	if currentRow >= scrollOffset+visibleRows {
		scrollOffset = currentRow - visibleRows + 1
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

	var grid strings.Builder
	grid.WriteString(header)
	grid.WriteString("\n")

	if m.focus == processFocusObjectTypes && len(visible) > 0 {
		realIdx := visible[m.typeCursor]

		previewWidth := m.typeGridWidth - 3
		if previewWidth < 1 {
			previewWidth = 1
		}

		preview := truncate(m.objectTypes[realIdx], previewWidth)

		grid.WriteString(
			labelStyle.Render(">> "),
		)
		grid.WriteString(
			textStyle.Render(preview),
		)
	}

	grid.WriteString("\n\n")

	if len(visible) == 0 {
		grid.WriteString(
			lipgloss.NewStyle().
				Foreground(colorMuted).
				Italic(true).
				Render(
					`  no types match "` +
						m.typeFilter +
						`"`,
				),
		)

		grid.WriteString("\n")
	} else {
		for row := startRow; row < endRow; row++ {
			start := row * cols
			end := start + cols

			if end > len(visible) {
				end = len(visible)
			}

			var cells []string
			for pos := start; pos < end; pos++ {
				cells = append(
					cells,
					m.renderProcessTypeCell(
						visible[pos],
						pos,
					),
				)
			}

			grid.WriteString(
				lipgloss.JoinHorizontal(
					lipgloss.Top,
					cells...,
				),
			)
			grid.WriteString("\n")
		}

		actualRows := endRow - startRow
		for i := actualRows; i < visibleRows; i++ {
			grid.WriteString("\n")
		}
	}

	scrollStyle :=
		lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	remainingRows := totalRows - endRow

	if scrollOffset > 0 {
		grid.WriteString(
			scrollStyle.Render(
				fmt.Sprintf(
					"  ^ %d more above",
					scrollOffset*cols,
				),
			),
		)
	}
	grid.WriteString("\n")

	if remainingRows > 0 {
		grid.WriteString(
			scrollStyle.Render(
				fmt.Sprintf(
					"  v %d more below",
					remainingRows*cols,
				),
			),
		)
	}
	grid.WriteString("\n")
	grid.WriteString("\n")
	grid.WriteString(filterLine)

	return style.Render(
		strings.TrimRight(grid.String(), "\n"),
	) + "\n\n"
}

func (m *processModel) renderProcessTypeCell(realIdx int, visiblePos int) string {
	const (
		cursorWidth   = 2
		checkboxWidth = 3
		separator     = 1
	)

	nameWidth :=
		m.typeCellWidth -
			cursorWidth -
			checkboxWidth -
			separator

	if nameWidth < 1 {
		nameWidth = 1
	}

	name :=
		truncate(
			m.objectTypes[realIdx],
			nameWidth,
		)

	checkbox := "[ ]"
	nameStyle := textStyle

	if m.selectedTypes[realIdx] {
		checkbox = "[x]"
		nameStyle = checkedStyle
	}

	prefix := textStyle.Render("  ")
	if m.focus == processFocusObjectTypes && visiblePos == m.typeCursor {
		prefix = cursorGlyphStyle.Render("> ")
	}

	nameLen := lipgloss.Width(name)
	padding := nameWidth - nameLen

	if padding < 0 {
		padding = 0
	}

	return prefix +
		nameStyle.Render(checkbox+" "+name) +
		textStyle.Render(strings.Repeat(" ", padding))
}

func (m *processModel) renderPropertiesSection() string {
	style := sectionStyle

	if m.focus == processFocusProperties {
		style = focusedSectionStyle
	}

	style = style.Width(m.typeBoxWidth)

	const (
		// Width of the left sub-column ("Signed" / "Not signed").
		signatureLeftSubWidth = 17

		// Minimum width needed to fit the widest signature row
		// ("Signed" + gap + "Hash mismatch") without truncation.
		signatureContentMinWidth = 36

		// Width reserved for the elevation column so its two
		// options ("Elevated" / "Not elevated") stay aligned.
		elevationColumnWidth = 18
	)

	gridWidth := m.typeGridWidth

	if gridWidth < signatureContentMinWidth+elevationColumnWidth {
		gridWidth = signatureContentMinWidth + elevationColumnWidth
	}

	leftColumnWidth := gridWidth - elevationColumnWidth

	if leftColumnWidth < signatureContentMinWidth {
		leftColumnWidth = signatureContentMinWidth
	}

	var b strings.Builder

	signatureTitle := labelStyle.Render("Signature Status")
	elevationTitle := labelStyle.Render("Elevation")

	header := padRight(signatureTitle, leftColumnWidth) + elevationTitle

	b.WriteString(header)
	b.WriteString("\n")

	signatureSigned := m.renderPropertyChoice(
		"Signed",
		m.signatureStatus[signatureSigned],
		processPropertyCursor == 0,
	)

	signatureHashMismatch := m.renderPropertyChoice(
		"Hash mismatch",
		m.signatureStatus[signatureHashMismatch],
		processPropertyCursor == 1,
	)

	signatureRow1 := padRight(
		padRight(signatureSigned, signatureLeftSubWidth)+signatureHashMismatch,
		leftColumnWidth,
	)

	elevationElevated := m.renderPropertyChoice(
		"Elevated",
		m.elevation[elevationElevated],
		processPropertyCursor == 4,
	)

	b.WriteString(signatureRow1 + elevationElevated)
	b.WriteString("\n")

	signatureNotSigned := m.renderPropertyChoice(
		"Not signed",
		m.signatureStatus[signatureNotSigned],
		processPropertyCursor == 2,
	)

	signatureOther := m.renderPropertyChoice(
		"Other",
		m.signatureStatus[signatureOther],
		processPropertyCursor == 3,
	)

	signatureRow2 := padRight(
		padRight(signatureNotSigned, signatureLeftSubWidth)+signatureOther,
		leftColumnWidth,
	)

	elevationNotElevated := m.renderPropertyChoice(
		"Not elevated",
		m.elevation[elevationNotElevated],
		processPropertyCursor == 5,
	)

	b.WriteString(signatureRow2 + elevationNotElevated)

	return style.Render(
		strings.TrimRight(b.String(), "\n"),
	) + "\n\n"
}

func (m *processModel) renderPropertyChoice(label string, checked bool, cursor bool) string {
	checkbox := "[ ]"

	if checked {
		checkbox = "[x]"
	}

	prefix := textStyle.Render("  ")
	if cursor && m.focus == processFocusProperties {
		prefix = cursorGlyphStyle.Render("> ")
	}

	nameStyle := textStyle

	if checked {
		nameStyle = checkedStyle
	}

	return prefix + nameStyle.Render(checkbox+" "+label)
}
