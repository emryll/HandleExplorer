package tuimenu

import (
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
