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

	// ---------------------------------------------------------------
	// Data rows
	// ---------------------------------------------------------------

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

	// ---------------------------------------------------------------
	// More-above indicator
	// ---------------------------------------------------------------

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

	// ---------------------------------------------------------------
	// More-below indicator
	// ---------------------------------------------------------------

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
