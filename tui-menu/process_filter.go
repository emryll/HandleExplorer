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

func (m *processModel) renderProcessTextSection(
	label string,
	ti *textinput.Model,
	field processFocusField,
) string {
	style := sectionStyle

	if m.focus == field {
		style = focusedSectionStyle
	}

	style = style.Width(m.typeBoxWidth)

	content :=
		labelStyle.Render(label) + "\n" +
			renderInput(ti, m.typeGridWidth)

	return style.Render(content) + "\n\n"
}

func (m *processModel) renderListSection(
	label string,
	ti *textinput.Model,
	count int,
	field processFocusField) string {
	style := sectionStyle

	if m.focus == field {
		style = focusedSectionStyle
	}

	style = style.Width(m.typeBoxWidth)

	countText :=
		fmt.Sprintf(
			"%d entries",
			count,
		)

	content :=
		labelStyle.Render(label) +
			textStyle.Render("  ") +
			subtitleStyle.Render(countText) +
			"\n" +
			renderInput(ti, m.typeGridWidth)

	return style.Render(content) + "\n\n"
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

//*=========================[ Helpers ]===============================

func (m *processModel) syncFocus() {
	m.path.Blur()
	m.allowlistInput.Blur()
	m.parentInput.Blur()

	switch m.focus {
	case processFocusPath:
		m.path.Focus()

	case processFocusAllowlist:
		m.allowlistInput.Focus()

	case processFocusParent:
		m.parentInput.Focus()
	}
}

func (m *processModel) recalcLayout() {
	frameWidth := sectionStyle.GetHorizontalFrameSize()
	boxWidth := m.width - (processFormSideMargin * 2)

	if boxWidth < processTypeMinCellWidth+frameWidth {
		boxWidth = processTypeMinCellWidth + frameWidth
	}

	m.typeBoxWidth = boxWidth
	m.typeGridWidth = boxWidth - frameWidth

	if m.typeGridWidth < processTypeMinCellWidth {
		m.typeGridWidth = processTypeMinCellWidth
	}

	cols := m.typeGridWidth / processTypeMinCellWidth

	if cols < 1 {
		cols = 1
	}
	if cols > 4 {
		cols = 4
	}

	m.typeCols = cols

	m.typeCellWidth = m.typeGridWidth / cols
	if m.typeCellWidth < 1 {
		m.typeCellWidth = 1
	}

	//* Calculate the amount of vertical space
	//* that the fixed parts of the form consume.

	//? The object type grid is the only section whose
	//? number of rows should change with terminal height.

	//
	// Header:
	//
	//   title
	//   subtitle
	//   blank
	//
	headerHeight := 3

	//
	// Three text/list sections:
	//
	//   border top
	//   label
	//   input
	//   border bottom
	//   blank line
	//
	textSectionHeight := 5

	fixedHeight := headerHeight + (textSectionHeight * 3)

	//
	// Properties:
	//
	//   border top
	//   title row
	//   signature/elevation row
	//   signature/elevation row
	//   border bottom
	//   blank line
	//
	propertiesHeight := 6

	//
	// Object type section overhead, excluding its actual object type rows:
	//   border top
	//   header
	//   preview or blank
	//   blank line
	//   grid rows (handled separately)
	//   scroll info (1 or 2 rows)
	//   blank line
	objectTypeOverhead := 10

	//
	// Help line.
	// NOTE: this must be 2, not 1. helpStyle has MarginTop(1), which
	// lipgloss renders as an extra blank line before the text, so
	// the help line actually costs 2 rendered lines.
	helpHeight := 2

	fixedHeight +=
		propertiesHeight +
			objectTypeOverhead +
			helpHeight

	availableRows := m.height - fixedHeight
	if availableRows < processMinVisibleRows {
		availableRows = processMinVisibleRows
	}
	if availableRows > processMaxVisibleRows {
		availableRows = processMaxVisibleRows
	}

	m.typeVisibleRows = availableRows

	//
	// Text inputs must fit inside the section's content area.
	//
	tiWidth := m.typeGridWidth

	if tiWidth < 10 {
		tiWidth = 10
	}

	m.path.Width = tiWidth
	m.allowlistInput.Width = tiWidth
	m.parentInput.Width = tiWidth
}

func (m *processModel) addAllowlistEntry() {
	value := strings.TrimSpace(
		m.allowlistInput.Value(),
	)

	if value == "" {
		return
	}

	m.allowlist = append(
		m.allowlist,
		value,
	)

	m.allowlistInput.Reset()
}

func (m *processModel) addParentEntry() {
	value := strings.TrimSpace(
		m.parentInput.Value(),
	)

	if value == "" {
		return
	}

	m.parent = append(
		m.parent,
		value,
	)

	m.parentInput.Reset()
}

//*==============================[ Properties ]========================

// processPropertyCursor:
//
//	0 = Signed
//	1 = Hash mismatch
//	2 = Not signed
//	3 = Other
//	4 = Elevated
//	5 = Not elevated
//
// The signature layout is:
//
//	0             1
//	Signed        Hash mismatch
//
//	2             3
//	Not signed    Other
//
// Elevation is:
//
//	4
//	Elevated
//
//	5
//	Not elevated
//
// This is package-level because the existing model in this test program
// already uses this cursor design.
var processPropertyCursor int

func (m *processModel) updateProperties(msg *tea.KeyMsg) {
	switch msg.String() {
	case "left":
		switch processPropertyCursor {
		case 1:
			processPropertyCursor = 0

		case 3:
			processPropertyCursor = 2

		case 4:
			processPropertyCursor = 1

		case 5:
			processPropertyCursor = 3
		}

	case "right":
		switch processPropertyCursor {
		case 0:
			processPropertyCursor = 1

		case 2:
			processPropertyCursor = 3

		case 1:
			processPropertyCursor = 4

		case 3:
			processPropertyCursor = 5
		}

	case "up":
		switch processPropertyCursor {
		case 2:
			processPropertyCursor = 0

		case 3:
			processPropertyCursor = 1

		case 5:
			processPropertyCursor = 4
		}

	case "down":
		switch processPropertyCursor {
		case 0:
			processPropertyCursor = 2

		case 1:
			processPropertyCursor = 3

		case 4:
			processPropertyCursor = 5
		}

	case " ":
		m.toggleCurrentProperty()
	}
}

func (m *processModel) signatureFocus() bool {
	return processPropertyCursor < 4
}

func (m *processModel) toggleCurrentProperty() {
	switch processPropertyCursor {
	case 0:
		m.signatureStatus[signatureSigned] =
			!m.signatureStatus[signatureSigned]

	case 1:
		m.signatureStatus[signatureHashMismatch] =
			!m.signatureStatus[signatureHashMismatch]

	case 2:
		m.signatureStatus[signatureNotSigned] =
			!m.signatureStatus[signatureNotSigned]

	case 3:
		m.signatureStatus[signatureOther] =
			!m.signatureStatus[signatureOther]

	case 4:
		m.elevation[elevationElevated] =
			!m.elevation[elevationElevated]

	case 5:
		m.elevation[elevationNotElevated] =
			!m.elevation[elevationNotElevated]
	}
}

func (m *processModel) visibleTypeIndices() []int {
	if m.typeFilter == "" {
		idxs := make([]int, len(m.objectTypes))

		for i := range m.objectTypes {
			idxs[i] = i
		}

		return idxs
	}

	q := strings.ToLower(m.typeFilter)

	var idxs []int

	for i, t := range m.objectTypes {
		if strings.Contains(
			strings.ToLower(t),
			q,
		) {
			idxs = append(idxs, i)
		}
	}

	return idxs
}

func (m *processModel) updateObjectTypes(msg *tea.KeyMsg) {
	visible := m.visibleTypeIndices()

	n := len(visible)

	if n == 0 {
		m.typeCursor = 0
	}

	cols := m.typeCols

	if cols < 1 {
		cols = 1
	}

	switch msg.String() {
	case "up":
		if m.typeCursor-cols >= 0 {
			m.typeCursor -= cols
		}

	case "down":
		if m.typeCursor+cols < n {
			m.typeCursor += cols
		}

	case "left":
		if m.typeCursor%cols != 0 {
			m.typeCursor--
		}

	case "right":
		if m.typeCursor%cols != cols-1 &&
			m.typeCursor+1 < n {
			m.typeCursor++
		}

	case " ":
		if n > 0 {
			realIdx := visible[m.typeCursor]

			m.selectedTypes[realIdx] =
				!m.selectedTypes[realIdx]
		}

	case "backspace":
		if len(m.typeFilter) > 0 {
			r := []rune(m.typeFilter)

			m.typeFilter = string(
				r[:len(r)-1],
			)

			m.typeCursor = 0
		}

	default:
		if msg.Type == tea.KeyRunes {
			m.typeFilter += string(msg.Runes)
			m.typeCursor = 0
		}
	}

	visible = m.visibleTypeIndices()

	if len(visible) == 0 {
		m.typeCursor = 0
	} else if m.typeCursor >= len(visible) {
		m.typeCursor = len(visible) - 1
	}
}

func (m *processModel) buildFilter() *ProcessFilter {
	var types []string

	for i, t := range m.objectTypes {
		if m.selectedTypes[i] {
			types = append(types, t)
		}
	}

	var signatures []string

	if m.signatureStatus[signatureSigned] {
		signatures = append(signatures, "Signed")
	}

	if m.signatureStatus[signatureNotSigned] {
		signatures = append(signatures, "Not signed")
	}

	if m.signatureStatus[signatureHashMismatch] {
		signatures = append(signatures, "Hash mismatch")
	}

	if m.signatureStatus[signatureOther] {
		signatures = append(signatures, "Other")
	}

	var elevations []string

	if m.elevation[elevationElevated] {
		elevations = append(
			elevations,
			"Elevated",
		)
	}

	if m.elevation[elevationNotElevated] {
		elevations = append(
			elevations,
			"Not elevated",
		)
	}

	return &ProcessFilter{
		Path: m.path.Value(),

		DirectoryAllowlist: append(
			[]string(nil),
			m.allowlist...,
		),

		ParentProcess: append(
			[]string(nil),
			m.parent...,
		),

		SignatureStatus: signatures,
		Elevation:       elevations,
		ObjectTypes:     types,
	}
}
