package tuimenu

import tea "github.com/charmbracelet/bubbletea"

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

	m := finalModel.(*ProcessFilterModel)
	return m.result
}
