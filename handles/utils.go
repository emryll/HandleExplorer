package handles

import (
	"HandleExplorer/process"
	tlist "HandleExplorer/tui/list"
	"HandleExplorer/utils"
	"fmt"
	"path/filepath"
	"sort"
)

// Find all processes whose object access overlaps with that of the given process.
// Result is a map where key is pid and value is how many times it overlapped.
// If no processes with overlapping object access are found, the result is nil.
// The second return value is all the clusters this process is a part of.
func (reg *ObjectAccessRegistry) FindOverlappingWithPs(pid uint32) (map[uint32]int, []Cluster) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	if len(reg.ProcessLookup[pid]) == 0 {
		return nil, nil
	}

	var clusters []Cluster
	overlapping := make(map[uint32]int)
	for key := range reg.ProcessLookup[pid] {
		if key.Name == "" {
			continue // cant track anon objects currently :(
		}
		cluster := Cluster{
			ObjName: key.Name,
			ObjType: key.ObjType,
		}
		// add all other processes that accessed the named object
		for objKey := range reg.ObjectLookup[key.ObjType] {
			if objKey.Name == key.Name {
				if objKey.Pid != pid {
					overlapping[objKey.Pid]++
				}
				cluster.Members = append(cluster.Members, objKey.Pid)
			}
		}
		if len(cluster.Members) > 1 {
			clusters = append(clusters, cluster)
		}
	}
	return overlapping, clusters
}

// Find all objects accessed by several different processes.
// This method will read lock the object access registry.
func (reg *ObjectAccessRegistry) FindOverlapping(filter *ClusterFilter) ([]*Cluster, ClusterStats) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	if filter == nil {
		filter = &ClusterFilter{}
	}

	var (
		total       int // total size of clusters (for avg)
		overlapping []*Cluster
		accessed    = make(map[ProcessAccessKey][]uint32)
		stats       = ClusterStats{
			DirFrequency: make(map[string]int),
			ExeFrequency: make(map[string]int),
			ObjFrequency: make(map[uint32]int),
		}
	)

	for pid, objs := range reg.ProcessLookup {
		for key := range objs {
			if filter.ObjType != 0 && key.ObjType != filter.ObjType {
				continue
			}
			if key.Name == "" {
				continue // cant track anon objects currently :(
			}
			if filter.ObjName != "" && key.Name != filter.ObjName {
				continue
			}
			accessed[key] = append(accessed[key], pid)
		}
	}

	for key, pids := range accessed {
		if len(pids) < 2 {
			continue
		}
		if filter.MinSize > 0 && len(pids) < filter.MinSize {
			continue
		}

		cluster := Cluster{
			ObjType: key.ObjType,
			ObjName: key.Name,
		}
		cluster.Members = append(cluster.Members, pids...)
		overlapping = append(overlapping, &cluster)

		// collect data for cluster stats
		for _, pid := range pids {
			path := process.LookupProcessPath(pid)
			if path != "" {
				stats.ExeFrequency[path]++
			}
			if filepath.Base(path) != path {
				stats.DirFrequency[filepath.Dir(path)]++
			}
		}
		stats.ObjFrequency[key.ObjType]++
		total += len(cluster.Members)
	}

	// avoid out of bounds panic
	if len(overlapping) == 0 {
		return overlapping, stats
	}

	//* finish cluster stat calculations
	sort.Slice(overlapping, func(i, j int) bool {
		return len(overlapping[i].Members) > len(overlapping[j].Members)
	})

	if len(overlapping)%2 == 0 {
		upperMidIndex := len(overlapping) / 2
		totalMiddle := len(overlapping[upperMidIndex-1].Members)
		totalMiddle += len(overlapping[upperMidIndex].Members)
		stats.MedianSize = float32(totalMiddle) / 2
	} else {
		midIndex := len(overlapping) / 2
		stats.MedianSize = float32(len(overlapping[midIndex].Members))
	}
	stats.AvgSize = float32(total) / float32(len(overlapping))
	return overlapping, stats
}

// Get all pids that accessed a named object.
func GetObjectAccessPids(objType uint32, name string) []uint32 {
	AccessRegistry.mu.Lock()
	defer AccessRegistry.mu.Unlock()

	if len(AccessRegistry.ObjectLookup[objType]) == 0 {
		return nil
	}
	var (
		pids []uint32
		seen = make(map[uint32]bool)
	)
	for key := range AccessRegistry.ObjectLookup[objType] {
		if key.Name != name {
			continue
		}
		if seen[key.Pid] {
			continue
		}
		pids = append(pids, key.Pid)
		seen[key.Pid] = true
	}
	return pids
}

// Get all object types that a process has accessed
// This will read lock the object access registry.
func GetObjectTypesAccessed(pid uint32) map[uint32]bool {
	AccessRegistry.mu.RLock()
	defer AccessRegistry.mu.RUnlock()

	if len(AccessRegistry.ProcessLookup[pid]) == 0 {
		return nil
	}

	accessed := make(map[uint32]bool)
	for key := range AccessRegistry.ProcessLookup[pid] {
		accessed[key.ObjType] = true
	}
	return accessed
}

// Get total count of handles currently cached.
// This will read lock the global handle cache.
func GetTotalCachedHandleCount() int {
	HandleTable.mu.RLock()
	defer HandleTable.mu.RUnlock()

	var total int
	for _, typeMap := range HandleTable.Cache {
		for _, handleMap := range typeMap {
			total += len(handleMap)
		}
	}
	return total
}

func (h HandleEntry) GetParameter(name string) utils.Parameter {
	if param, exists := h.Parameters[name]; exists {
		return param
	}
	return utils.Parameter{}
}

//*=================[ Access mask util wrappers ]=================

// Get the access mask as a single printable value. No arrays
func (e *AccessEntry) GetAccessAsString() any {
	domain := GetDomainFromObject(e.Object)
	return utils.InterpretBitmaskValue(e.Access, domain)
}

// Get the access mask as a list of flags in human readable form.
func (e *AccessEntry) GetAccessFlagsAsString() []string {
	domain := GetDomainFromObject(e.Object)
	// last parameter as true guarantees []string return value
	return utils.InterpretBitmaskValue(e.Access, domain, true).([]string)
}

// Get the access mask as a single printable value. No arrays
func (h *HandleEntry) GetAccessAsString() any {
	domain := GetDomainFromObject(h.Type)
	return utils.InterpretBitmaskValue((utils.Bitmask)(h.Access), domain)
}

// Get the access mask as a list of flags in human readable form.
func (h *HandleEntry) GetAccessFlagsAsString() []string {
	domain := GetDomainFromObject(h.Type)
	// last parameter as true guarantees []string return value
	return utils.InterpretBitmaskValue((utils.Bitmask)(h.Access), domain, true).([]string)
}

//*===================[ ListItem (UI) interface methods ]===================

func (e *AccessEntry) Columns() []tlist.Column {
	return []tlist.Column{
		{Title: "Type", Highlight: true},
		{Title: "Name"},
		{Title: "Accessing Process", Highlight: true},
		{Title: "Access", Right: true},
	}
}

func (e *AccessEntry) Fields() []string {
	var accessingPs string
	/*if e.PsPath == "" {
		accessingPs = fmt.Sprintf("PID %d", e.Pid)
	} else {
		accessingPs = fmt.Sprintf("PID %d (%s)", e.Pid)
	}*/ //TODO
	accessingPs = fmt.Sprintf("PID %d", e.Pid)
	return []string{
		GetTypeName(e.Object),
		utils.OrDash(e.Name),
		accessingPs,
		fmt.Sprintf("%v", e.GetAccessAsString()),
	}
}

/*
func (e AccessEntry) RightStages() []string {
}*/

func (e *AccessEntry) Key() string {
	return fmt.Sprintf("%d:%d:%s:%X", e.Pid, e.Object, e.Name, e.Access)
}

// Used for polymorphic list (newPickerModel)
func (e *AccessEntry) Title() string {
	return "Handle"
}

// Used for polymorphic list (newPickerModel)
func (e *AccessEntry) Subtitle() string {
	return "Handle search results"
}

// Used for polymorphic list (newPickerModel)
func (e *AccessEntry) Noun() string {
	return "handles"
}

//*==================[ Cluster ]=====================

func (c Cluster) Columns() []tlist.Column {
	return []tlist.Column{
		{Title: "Type", Highlight: true},
		{Title: "Name"},
		{Title: "Size"},
	}
}

func (c Cluster) Fields() []string {
	return []string{
		GetTypeName(c.ObjType),
		utils.OrDash(c.ObjName),
		fmt.Sprintf("%d in cluster", len(c.Members)),
	}
}

func (c Cluster) Key() string {
	return fmt.Sprintf("%s:%s", c.ObjType, c.ObjName)
}

// Used for polymorphic list (newPickerModel)
func (c Cluster) Title() string {
	return "Cluster"
}

// Used for polymorphic list (newPickerModel)
func (c Cluster) Subtitle() string {
	return "Cluster search results"
}

// Used for polymorphic list (newPickerModel)
func (c Cluster) Noun() string {
	return "clusters"
}
