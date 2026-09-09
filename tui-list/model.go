package tlist

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func newPickerModel[T ListItem](items []T, title string, subtitle string, noun string) *pickerModel[T] {
	return &pickerModel[T]{
		picker:   newResultPicker(items),
		title:    title,
		subtitle: subtitle,
		noun:     noun,
	}
}

func (m *pickerModel[T]) Init() tea.Cmd {
	return nil
}

// Submitted reports whether the user picked
// an item rather than cancelling with esc/q/ctrl+c.
func (m *pickerModel[T]) Submitted() bool {
	return m.submitted && m.hasResult
}

// Result returns the selected item.
// Only meaningful if Submitted() is true.
func (m *pickerModel[T]) Result() T {
	return m.result
}

func (m *pickerModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.picker.recalcLayout(
			msg.Width,
			msg.Height,
			13,
		)

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if item, ok := m.picker.selected(); ok {
				m.result = item
				m.hasResult = true
				m.submitted = true
			}

			m.quitting = true
			return m, tea.Quit
		}

		m.picker.update(msg)

		return m, nil
	}

	return m, nil
}

func (m *pickerModel[T]) View() string {
	if m.quitting {
		return ""
	}

	var help string
	if len(m.picker.items) == 0 {
		help = "esc cancel"
	} else {
		help = "up/down move - pgup/pgdown/home/end jump - enter select - esc cancel"
	}

	body := m.picker.view(
		"HandleExplorer",
		m.subtitle,
		m.noun,
	)

	helpWidth := m.picker.boxWidth +
		sectionStyle.GetHorizontalFrameSize()

	helpText := truncate(help, helpWidth)

	helpRow := lipgloss.NewStyle().
		Background(colorPanel).
		Foreground(colorMuted).
		Width(helpWidth).
		Align(lipgloss.Center).
		Render(padRight(helpText, helpWidth))

	body += helpRow
	body += "\n"

	content := placeForm(
		m.width+1,
		m.height,
		body,
	)

	return lipgloss.NewStyle().
		Background(colorPanel).
		Width(m.width).
		Height(m.height).
		Render(content)
}
