package registry

import (
	"HandleExplorer/utils"
	"sync"
)

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
	sync.RWMutex // used internally in methods
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
