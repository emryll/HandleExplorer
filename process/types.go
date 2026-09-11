package process

import (
	"HandleExplorer/handles/registry"
	"HandleExplorer/stats"
	"sync"
	"sync/atomic"
)

//*====================[ Processes ]========================

// This structure is used to cache
// details about the process.
// The fields should be set only
// once when the entry is created!
// Handle count is the only exception.
type Process struct {
	Path       string
	ProcessId  uint32
	ParentPid  uint32
	ParentPath string
	Elevated   bool
	SigStatus  int
	// This stat is cached, because
	// its hard to retrieve otherwise.
	HandleCount atomic.Int64 // active handles only
}

type ProcessFilter struct {
	Path        string
	DirFilter   []string
	Pids        map[uint32]bool
	Parent      map[string]bool // path or pid
	ObjTypes    map[uint32]bool // object types accessed
	SigStatus   map[int]bool
	Elevated    bool // do you want to include elevated ones
	NotElevated bool // do you want to include not-elevated ones
	// avoid dependency cycle and ugly function prototype
	OAR *registry.ObjectAccessRegistry
}

type ProcessTable struct {
	sync.RWMutex
	Table map[uint32]*Process
	// dependency injection
	RtStats *stats.SessionStats
}

//*======================[ Constants ]==========================

var (
	PS_REFRESH_INTERVAL = 10
)

const ( // Digital signature check status
	CERT_UNKNOWN        = 0
	CERT_VALID          = 1
	CERT_MISSING        = 2
	CERT_HASH_MISMATCH  = 3
	CERT_EXP_DISTRUST   = 4
	CERT_UNTRUSTED_CA   = 5
	CERT_UNTRUSTED_ROOT = 6
	CERT_REVOKED        = 7
	CERT_EXPIRED        = 8
)

//*=========================[ Windows ]================================

type TOKEN_ELEVATION struct {
	TokenIsElevated uint32
}
