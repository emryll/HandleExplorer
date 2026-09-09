package tmenu

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//*======================[ Core ]============================

func newObjectTypePicker() objectTypePicker {
	return objectTypePicker{
		objectTypes: ntObjectTypes,
		selected:    make(map[int]bool),

		cols:        3,
		cellWidth:   typeMinCellWidth,
		nameMaxLen:  typeMinCellWidth - 6,
		visibleRows: minVisibleRows,

		boxWidth:  3 * typeMinCellWidth,
		gridWidth: 3 * typeMinCellWidth,
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
	currentRow := 0

	if len(visible) > 0 {
		totalRows = (len(visible) + p.cols - 1) / p.cols
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

//*======================[ Helpers ]=========================

func (p *objectTypePicker) visibleIndices() []int {
	if p.filter == "" {
		result := make([]int, len(p.objectTypes))

		for i := range p.objectTypes {
			result[i] = i
		}

		return result
	}

	query := strings.ToLower(p.filter)

	var result []int

	for i, objectType := range p.objectTypes {
		if strings.Contains(
			strings.ToLower(objectType),
			query,
		) {
			result = append(result, i)
		}
	}

	return result
}

func (p *objectTypePicker) selectedCount() int {
	count := 0

	for _, selected := range p.selected {
		if selected {
			count++
		}
	}

	return count
}

func (p *objectTypePicker) values() []string {
	var result []string

	for i, objectType := range p.objectTypes {
		if p.selected[i] {
			result = append(result, objectType)
		}
	}

	return result
}

func (p *objectTypePicker) recalcLayout(
	width int,
	height int,
	heightReserve int,
) {
	frameWidth := sectionStyle.GetHorizontalFrameSize()

	boxWidth := width - (formSideMargin * 2)

	if boxWidth < typeMinCellWidth+frameWidth {
		boxWidth = typeMinCellWidth + frameWidth
	}

	p.boxWidth = boxWidth
	p.gridWidth = boxWidth - frameWidth

	if p.gridWidth < typeMinCellWidth {
		p.gridWidth = typeMinCellWidth
	}

	cols := p.gridWidth / typeMinCellWidth

	if cols < 1 {
		cols = 1
	}

	if cols > 4 {
		cols = 4
	}

	p.cols = cols
	p.cellWidth = p.gridWidth / cols

	if p.cellWidth < 1 {
		p.cellWidth = 1
	}

	p.nameMaxLen = p.cellWidth - 6

	if p.nameMaxLen < 1 {
		p.nameMaxLen = 1
	}

	visibleRows := height - heightReserve

	if visibleRows < minVisibleRows {
		visibleRows = minVisibleRows
	}

	if visibleRows > maxVisibleRows {
		visibleRows = maxVisibleRows
	}

	p.visibleRows = visibleRows
}

func (p *objectTypePicker) renderCell(
	realIndex int,
	visiblePosition int,
) string {
	const (
		cursorWidth   = 2
		checkboxWidth = 3
		separator     = 1
	)

	nameWidth :=
		p.cellWidth -
			cursorWidth -
			checkboxWidth -
			separator

	if nameWidth < 1 {
		nameWidth = 1
	}

	name := truncate(
		p.objectTypes[realIndex],
		nameWidth,
	)

	checkbox := "[ ]"
	nameStyle := textStyle

	if p.selected[realIndex] {
		checkbox = "[x]"
		nameStyle = checkedStyle
	}

	prefix := textStyle.Render("  ")
	if visiblePosition == p.cursor {
		prefix = cursorGlyphStyle.Render("> ")
	}

	nameLen := lipgloss.Width(name)
	padding := nameWidth - nameLen

	if padding < 0 {
		padding = 0
	}

	return prefix +
		textStyle.Render(checkbox) +
		textStyle.Render(" ") +
		nameStyle.Render(name) +
		textStyle.Render(strings.Repeat(" ", padding))
}
