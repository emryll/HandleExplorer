package main

import (
	"sync"
	"unsafe"
)

type Bitmask uint32

type Parameter struct {
	Name   string
	Type   uint8
	Domain uint8
	Buffer []byte
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

var g_ObjectAccessRegistry ObjectAccessRegistry

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

//*============================[ Constants ]================================

const (
	OBJECT_TYPE_UNKNOWN        = 0
	OBJECT_TYPE_PROCESS        = 1
	OBJECT_TYPE_THREAD         = 2
	OBJECT_TYPE_FILE           = 3
	OBJECT_TYPE_SEMAPHORE      = 4
	OBJECT_TYPE_EVENT          = 5
	OBJECT_TYPE_MUTEX          = 6
	OBJECT_TYPE_SYMLINK        = 7
	OBJECT_TYPE_PIPE           = 8
	OBJECT_TYPE_TOKEN          = 9
	OBJECT_TYPE_DEVICE         = 10
	OBJECT_TYPE_DRIVER         = 11
	OBJECT_TYPE_DESKTOP        = 12
	OBJECT_TYPE_DBGOBJECT      = 13
	OBJECT_TYPE_DIRECTORY      = 14
	OBJECT_TYPE_WORKER_FACTORY = 15
	OBJECT_TYPE_SECTION        = 16
	OBJECT_TYPE_KEY            = 17
)

const (
	PARAMETER_ANSISTRING    = 1
	PARAMETER_ASTR_ARRAY    = 10
	PARAMETER_UINT32        = 2
	PARAMETER_UINT32_ARRAY  = 20
	PARAMETER_UINT64        = 3
	PARAMETER_UINT64_ARRAY  = 30
	PARAMETER_BOOLEAN       = 4
	PARAMETER_BOOLEAN_ARRAY = 40
	PARAMETER_POINTER       = 5
	PARAMETER_POINTER_ARRAY = 50
	PARAMETER_BYTES         = 7
)
