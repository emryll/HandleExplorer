package handles

import (
	"HandleExplorer/utils"
	"sync"
	"unsafe"
)

//*=========================[ Handles ]===========================

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
	Parameters map[string]utils.Parameter
}

type HandleFilter struct {
	ObjType []uint32
	Names   []string
	Pids    []uint32
	Access  utils.Bitmask
}

//*========================[ Handle Cache ]========================

const (
	HANDLE_REFRESH_INTERVAL = 30
	HANDLE_CACHE_EXPIRATION = 10
	// This indicates a desired maximum,
	// not an actual hard limit on the capacity.
	// After this count, cache gets cleanup.
	HANDLE_CACHE_MAX_COUNT = 1000000 // around 50MB
)

var ( //* Handle Cache Cleanup modifiers
	HCC_MULTIPLIER_CONST = 1
	// these two just define the ratio
	// between object type and age of entry
	HCC_OBJECT_MULTIPLIER = 3
	HCC_TIME_MULTIPLIER   = 2
	// this one dictates how
	// the elapsed time scales
	HCC_TIME_POWER = 1.3

	HCC_DEFAULT_QUOTA        = 200
	HCC_TOP_PRIORITY_BONUS   = 5
	HCC_OBJECT_TIER_1_SCORE  = 2
	HCC_OBJECT_TIER_2_SCORE  = 15
	HCC_OBJECT_TIER_3_SCORE  = 40
	HCC_OBJECT_TIER_4_SCORE  = 60
	HCC_OBJECT_UNKNOWN_SCORE = 70
)

//*=====================[ Object Access Registry ]========================

type AccessEntry struct {
	Object uint32 // type enum
	Name   string // name of object
	Pid    uint32 // who accessed the object
	Handle uint32 // raw handle value used as id
	Access utils.Bitmask
	Params map[string]utils.Parameter // extended object info
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

//*============================[ Clusters ]=============================

// A cluster describes overlapping
// access to a (named) object.
type Cluster struct {
	Members []uint32
	ObjType uint32
	ObjName string
	Params  map[string]utils.Parameter
}

type ClusterStats struct {
	TotalCount   int
	AvgSize      float32
	MedianSize   float32
	DirFrequency map[string]int
	ExeFrequency map[string]int
	ObjFrequency map[uint32]int
}

type ClusterFilter struct {
	MinSize int
	ObjType uint32
	ObjName string
}
