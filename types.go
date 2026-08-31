package main

import (
	"sync"
	"unsafe"
)

//*=========================[ Processes ]===========================

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

type ProcessTable struct {
	mu    sync.RWMutex
	Table map[uint32]*Process
}

//*==========================[ Handle Entry ]=========================

type cHandleEntry struct {
	FirstSeen  int64
	LastSeen   int64
	Params     unsafe.Pointer
	ParamsSize uint64
	Handle     uint32
	Access     uint32
	Type       uint32
	Pid        uint32
}

type HandleEntry struct {
	FirstSeen  int64
	LastSeen   int64
	Handle     uint32
	Access     uint32
	Type       uint32
	Pid        uint32
	Parameters map[string]Parameter
}

//*=====================[ Object Access Registry ]========================

type AccessEntry struct {
	Object uint32 // type enum
	Name   string // name of object
	Pid    uint32 // who accessed the object
	Handle uint32 // raw handle value used as id
	Access uint32
	Params map[string]Parameter // extended object info
}

// Lookup table for object interactions
// 500 000 entries would be around 32MB
type ObjectAccessRegistry struct {
	mu sync.RWMutex // used internally in methods
	// process -> object type -> name -> entry
	ProcessLookup map[uint32]map[ProcessAccessKey][]*AccessEntry // array is for anon objects
	// object type -> name -> process -> entry
	ObjectLookup map[uint32]map[ObjectAccessKey][]*AccessEntry
}

var g_ObjectAccessRegistry = &ObjectAccessRegistry{}

// With the triple nested map, amount of maps grows very quickly.
// To fix this issue, the structure is partially flattened.
// Instead of a triple map its a double map with a struct key,
// which has a very big effect on the amount of maps created.

// This key struct is made to flatten ProcessLookup
type ProcessAccessKey struct {
	ObjType uint32
	Name    string
}

// This key struct is made to flatten ObjectLookup
type ObjectAccessKey struct {
	Pid  uint32
	Name string
}

// A cluster describes overlapping
// access to a (named) object.
type Cluster struct {
	Members []uint32
	ObjType uint32
	ObjName string
	Params  map[string]Parameter
}

//*======================[ Search filters (CLI) ]===========================

type SearchFilter struct {
	ObjType []uint32
	Names   []string
	Pids    []uint32
	Access  Bitmask
}

type ProcessFilter struct {
	Path      string
	DirFilter []string
	Parent    []string // pid or path
	SigStatus int
	Elevated  bool
	ObjTypes  map[string]bool
	Pids      []uint32
}

type ClusterFilter struct {
	MinSize int
	ObjType uint32
	ObjName string
}

//*========================[ Util types ]===============================

var g_SessionStats = &SessionStats{}
// Cached stats about current state.
type SessionStats struct {
	mu                   sync.RWMutex
	TotalActiveHandles   int
	TotalActiveProcesses int
}

type ClusterStats struct {
    AvgSize float32
    MedianSize float32
    DirFrequency map[string]int
    ExeFrequency map[string]int
}

type Bitmask uint32

type Parameter struct {
	Name   string
	Type   uint8
	Domain uint8
	Buffer []byte
}

//*=========================[ Windows ]================================

type TOKEN_ELEVATION struct {
	TokenIsElevated uint32
}
