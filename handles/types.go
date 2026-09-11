package handles

import (
	"HandleExplorer/utils"
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
