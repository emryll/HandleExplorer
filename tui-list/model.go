package tlist

import (
	tea "github.com/charmbracelet/bubbletea"
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
func (m *pickerModel[T]) Result() *T {
	return &m.result
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
