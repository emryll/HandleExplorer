package tuilist

import tea "github.com/charmbracelet/bubbletea"

func RenderList[T ListItem](items []T) *T {
	var (
		// placeholder for getting type specific info
		item T

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
		return nil
	}

	return finalModel.Result()
}
