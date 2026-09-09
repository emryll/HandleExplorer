package tlist

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func RenderList[T ListItem](items []T) T {
	var zero T
	if len(items) == 0 {
		return zero
	}

	var (
		item     = items[0]
		title    = item.Title()
		subtitle = item.Subtitle()
		noun     = item.Noun()
	)

	listPicker := tea.NewProgram(
		newPickerModel(
			items,
			title,
			subtitle,
			noun,
		),
		tea.WithAltScreen(),
	)

	finalModel, err := listPicker.Run()
	if err != nil {
		PrintError("Failed to launch list picker: %v\n", err)
		return zero
	}

	return finalModel.(*pickerModel[T]).Result()
}

//*======================[ Picker model ]=========================

func newResultPicker[T ListItem](items []T) resultPicker[T] {
	p := resultPicker[T]{
		items:       items,
		rightIndex:  -1,
		rightStage:  -1,
		boxWidth:    60,
		visibleRows: minVisibleRows,
	}

	if len(items) > 0 {
		p.columns = items[0].Columns()

		for i, col := range p.columns {
			if col.Right {
				p.rightIndex = i
				break
			}
		}

		if p.rightIndex >= 0 {
			_, p.rightStaged = any(items[0]).(StagedField)
		}
	}

	p.recalcColumnWidths()
	p.fitColumnsToWidth()
	p.recalcRightColumn()

	return p
}

func (p *resultPicker[T]) view(title, subtitle, noun string) string {
	style := sectionStyle.
		Width(p.boxWidth + sectionStyle.GetHorizontalFrameSize())

	var b strings.Builder

	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	b.WriteString(subtitleStyle.Render(subtitle))
	b.WriteString("\n\n")

	total := len(p.items)

	if total == 0 {
		b.WriteString(
			emptyStyle.Render(
				fmt.Sprintf("  no %s found", noun),
			),
		)
		b.WriteString("\n")

		return style.Render(
			strings.TrimRight(b.String(), "\n"),
		) + "\n\n"
	}

	b.WriteString(p.headerLine())
	b.WriteString("\n")

	b.WriteString(
		subtitleStyle.Render(
			strings.Repeat("─", p.boxWidth),
		),
	)
	b.WriteString("\n")

	// ---------------------------------------------------------------
	// Scrolling
	//
	// p.visibleRows is the TOTAL number of rows available for the
	// scrolling area.
	//
	// When scrolled, ONE of those rows is used by the "more above"
	// indicator. Do not subtract another row based on the cursor.
	// ---------------------------------------------------------------

	p.scrollOffset = clampScrollOffset(
		p.cursor,
		total,
		p.visibleRows,
	)

	scrolled := p.scrollOffset > 0

	itemRows := p.visibleRows
	if scrolled {
		itemRows--
	}

	if itemRows < 1 {
		itemRows = 1
	}

	endRow := p.scrollOffset + itemRows
	if endRow > total {
		endRow = total
	}

	hasBelow := endRow < total

	// Data rows
	for i := p.scrollOffset; i < endRow; i++ {
		b.WriteString(
			p.renderDataRow(
				p.items[i],
				i == p.cursor,
				i%2 == 1,
			),
		)
		b.WriteString("\n")
	}

	// More-above indicator
	if scrolled {
		b.WriteString(
			scrollStyle.Render(
				fmt.Sprintf(
					"  ^ %d more %s above",
					p.scrollOffset,
					noun,
				),
			),
		)
		b.WriteString("\n")
	}

	// More-below indicator
	if hasBelow {
		b.WriteString(
			scrollStyle.Render(
				fmt.Sprintf(
					"  v %d more %s below",
					total-endRow,
					noun,
				),
			),
		)
		b.WriteString("\n")
	}

	// ---------------------------------------------------------------
	// EXTRA LIST-PANEL ROW
	//
	// This is an actual row of cells, not just "\n".
	// It therefore receives the panel background and occupies one
	// complete row inside the bordered section.
	// ---------------------------------------------------------------

	blankRow := lipgloss.NewStyle().
		Background(colorPanel).
		Width(p.boxWidth).
		Render(strings.Repeat(" ", p.boxWidth))

	b.WriteString(blankRow)

	return style.Render(
		strings.TrimSuffix(b.String(), "\n"),
	) + "\n\n"
}

func (p *resultPicker[T]) update(msg tea.KeyMsg) {
	switch msg.String() {
	case "up":
		if p.cursor > 0 {
			p.cursor--
		}

	case "down":
		if p.cursor < len(p.items)-1 {
			p.cursor++
		}

	case "pgup":
		p.cursor -= p.visibleRows

		if p.cursor < 0 {
			p.cursor = 0
		}

	case "pgdown":
		p.cursor += p.visibleRows

		if p.cursor > len(p.items)-1 {
			p.cursor = len(p.items) - 1
		}

	case "home":
		p.cursor = 0

	case "end":
		if len(p.items) > 0 {
			p.cursor = len(p.items) - 1
		}
	}
}

func (p *resultPicker[T]) selected() (T, bool) {
	var zero T

	if len(p.items) == 0 {
		return zero, false
	}

	if p.cursor < 0 || p.cursor >= len(p.items) {
		return zero, false
	}

	return p.items[p.cursor], true
}

//*=====================[ Render ]======================

func (p *resultPicker[T]) rowCells(item T) []string {
	cells := item.Fields()

	if p.rightIndex < 0 || !p.rightStaged || p.rightStage < 0 {
		return cells
	}

	sf, ok := any(item).(StagedField)

	if !ok {
		return cells
	}

	stages := sf.RightStages()

	if p.rightStage >= len(stages) {
		return cells
	}

	out := append([]string(nil), cells...)
	out[p.rightIndex] = stages[p.rightStage]

	return out
}

// ---------------------------------------------------------------------
// Rendering helpers
// ---------------------------------------------------------------------

func (p *resultPicker[T]) renderCell(
	text string,
	width int,
	bg lipgloss.Color,
	style lipgloss.Style,
) string {
	if width <= 0 {
		return ""
	}

	text = truncate(text, width)

	return style.
		Background(bg).
		Render(padRight(text, width))
}

// renderLine guarantees that the returned line is exactly boxWidth
// cells wide.
//
// This is the important invariant:
//
//	lipgloss.Width(renderLine(...)) == p.boxWidth
//
// The section itself then adds its border around that exact content
// width.
func (p *resultPicker[T]) renderLine(
	cells []string,
	bg lipgloss.Color,
	styleFn func(colIndex int) lipgloss.Style,
) string {
	gapStyle := lipgloss.NewStyle().
		Background(bg)

	var leftParts []string

	rightText := ""
	rightWidth := 0
	rightIndex := -1

	for i, col := range p.columns {
		text := ""

		if i < len(cells) {
			text = cells[i]
		}

		if col.Right {
			rightIndex = i
			rightWidth = p.colWidths[i]
			rightText = text
			continue
		}

		if p.colWidths[i] <= 0 {
			continue
		}

		leftParts = append(
			leftParts,
			p.renderCell(
				text,
				p.colWidths[i],
				bg,
				styleFn(i),
			),
		)
	}

	line := strings.Join(
		leftParts,
		gapStyle.Render(strings.Repeat(" ", columnGap)),
	)

	// No visible right column.
	if rightIndex < 0 || rightWidth <= 0 {
		width := lipgloss.Width(line)

		if width < p.boxWidth {
			line += gapStyle.Render(
				strings.Repeat(" ", p.boxWidth-width),
			)
		}

		return line
	}

	leftWidth := lipgloss.Width(line)

	// Space between the left columns and right-pinned column.
	gap := p.boxWidth - leftWidth - rightWidth

	if gap < rightColumnGap {
		// Right column no longer fits.
		//
		// The sizing pass should normally prevent this, but this keeps
		// rendering safe if widths become inconsistent.
		if leftWidth < p.boxWidth {
			line += gapStyle.Render(
				strings.Repeat(" ", p.boxWidth-leftWidth),
			)
		}

		return line
	}

	line += gapStyle.Render(
		strings.Repeat(" ", gap),
	)

	line += p.renderCell(
		rightText,
		rightWidth,
		bg,
		styleFn(rightIndex),
	)

	// Final normalization.
	width := lipgloss.Width(line)

	if width < p.boxWidth {
		line += gapStyle.Render(
			strings.Repeat(" ", p.boxWidth-width),
		)
	}

	if width > p.boxWidth {
		line = truncate(line, p.boxWidth)
	}
	return line
}
func (p *resultPicker[T]) headerLine() string {
	titles := make([]string, len(p.columns))

	for i, col := range p.columns {
		titles[i] = strings.ToUpper(col.Title)
	}

	style := func(i int) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(columnColors[i%len(columnColors)]).
			Bold(true)
	}

	// Use the panel background for the header so it visually matches
	// the section instead of introducing another background.
	return p.renderLine(
		titles,
		colorPanel,
		style,
	)
}

func (p *resultPicker[T]) ruleLine() string {
	return subtitleStyle.Render(
		strings.Repeat("─", p.boxWidth),
	)
}

func (p *resultPicker[T]) renderDataRow(item T, isCursor bool, zebra bool) string {
	fields := p.rowCells(item)

	if isCursor {
		style := func(int) lipgloss.Style {
			return lipgloss.NewStyle().
				Foreground(colorSelectedFg).
				Bold(true)
		}

		return p.renderLine(
			fields,
			colorSelectedBg,
			style,
		)
	}

	bg := colorRowEven

	if zebra {
		bg = colorRowOdd
	}

	style := func(i int) lipgloss.Style {
		s := lipgloss.NewStyle().
			Foreground(columnColors[i%len(columnColors)])

		if i < len(p.columns) && p.columns[i].Highlight {
			s = s.Bold(true)
		}

		return s
	}

	return p.renderLine(
		fields,
		bg,
		style,
	)
}
