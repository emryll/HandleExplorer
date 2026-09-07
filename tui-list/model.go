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
