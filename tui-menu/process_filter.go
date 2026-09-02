package tmenu

import (
	"charm.land/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

	m := finalModel.(*ProcessFilterModel)
	return m.result
}

//*=======================[ Initial Model ]===========================

func initialProcessFilterModel() *processModel {
	path := textinput.New()
	path.Placeholder = `e.g. C:\Windows\System32\explorer.exe`
	path.CharLimit = 512
	path.Prompt = ""
	styleTextInput(&path)

	allowlistInput := textinput.New()
	allowlistInput.Placeholder = `e.g. C:\Windows\ or %appdata%`
	allowlistInput.CharLimit = 512
	allowlistInput.Prompt = ""
	styleTextInput(&allowlistInput)

	parentInput := textinput.New()
	parentInput.Placeholder = `e.g. svchost.exe or a PID`
	parentInput.CharLimit = 512
	parentInput.Prompt = ""
	styleTextInput(&parentInput)

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
