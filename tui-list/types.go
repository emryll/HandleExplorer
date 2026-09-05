package tlist

// Shared layout constants
const (
	minVisibleRows = 6
	sideMargin     = 2

	pickerHeaderRows = 5 // title, subtitle, blank, header, rule
	pickerHelpRows   = 2 // blank after picker + help
)

//*================[ List items ]=======================

type Column struct {
	Title     string
	Highlight bool
	Right     bool
}

type ListItem interface {
	Columns() []Column
	Fields() []string
	Key() string
}

type StagedField interface {
	// Returns progressively shorter versions of the right-pinned field.
	//
	// Example:
	//
	//	[]string{
	//	    "PID 501 (svchost.exe)",
	//	    "PID 501 (...)",
	//	    "PID 501",
	//	}
	//
	// The picker chooses the most detailed stage that fits.
	RightStages() []string
}

//*=======================[ Result Picker ]==========================

type pickerModel[T ListItem] struct {
	picker resultPicker[T]

	title    string
	subtitle string
	noun     string

	width  int
	height int

	quitting  bool
	submitted bool
	result    T
	hasResult bool
}

type resultPicker[T ListItem] struct {
	items     []T
	columns   []Column
	colWidths []int

	// rightIndex is -1 when there is no right-pinned column.
	rightIndex int

	// rightStaged says whether the items implement StagedField.
	rightStaged bool

	// rightStage is the selected staged representation.
	// -1 means the right column is hidden.
	rightStage int

	cursor       int
	scrollOffset int

	// boxWidth is the actual width available to table rows.
	// It does NOT include the section border.
	boxWidth int

	visibleRows int
}

const (
	minColumnWidth = 8
	columnGap      = 2
	rightColumnGap = 2
)
