package store

import (
	"HandleExplorer/handles"
	"HandleExplorer/process"
	"HandleExplorer/stats"
)

const (
	MAJOR_VERSION = 0
	MINOR_VERSION = 1
)

var ( //* Shared global lookup structures
	RuntimeStats = &stats.SessionStats{}
	PsTable      = process.NewProcessTable(RuntimeStats)

	AccessTracker = handles.NewAccessTracker(PsTable, RuntimeStats)
)
