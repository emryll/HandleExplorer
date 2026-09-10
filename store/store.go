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

var (
	PsTable        = process.NewProcessTable()
	AccessRegistry = &handles.ObjectAccessRegistry{}
	RuntimeStats   = &stats.SessionStats{}
	HandleTable    handles.HandleCache
)
