package registry

import (
	"HandleExplorer/nt"
	tlist "HandleExplorer/tui/list"
	"HandleExplorer/utils"
	"fmt"
	"path/filepath"
	"sort"
)

// Get all object types that a process has accessed
// This will read lock the object access registry.
func (reg *ObjectAccessRegistry) GetObjectTypesAccessed(pid uint32) map[uint32]bool {
	reg.RLock()
	defer reg.RUnlock()

	if len(reg.ProcessLookup[pid]) == 0 {
		return nil
	}

	accessed := make(map[uint32]bool)
	for key := range reg.ProcessLookup[pid] {
		accessed[key.ObjType] = true
	}
	return accessed
}

// Find all processes whose object access overlaps with that of the given process.
// Result is a map where key is pid and value is how many times it overlapped.
// If no processes with overlapping object access are found, the result is nil.
// The second return value is all the clusters this process is a part of.
func (reg *ObjectAccessRegistry) FindOverlappingWithPs(pid uint32) (map[uint32]int, []Cluster) {
	reg.RLock()
	defer reg.RUnlock()

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
func (reg *ObjectAccessRegistry) FindOverlapping(filter *ClusterFilter, pathLookup func(pid uint32) string) ([]*Cluster, ClusterStats) {
	reg.RLock()
	defer reg.RUnlock()

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
			path := pathLookup(pid)
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

//*=================[ Access mask util wrappers ]=================

// Get the access mask as a single printable value. No arrays
func (e *AccessEntry) GetAccessAsString() any {
	domain := nt.GetDomainFromObject(e.Object)
	return utils.InterpretBitmaskValue(e.Access, domain)
}

// Get the access mask as a list of flags in human readable form.
func (e *AccessEntry) GetAccessFlagsAsString() []string {
	domain := nt.GetDomainFromObject(e.Object)
	// last parameter as true guarantees []string return value
	return utils.InterpretBitmaskValue(e.Access, domain, true).([]string)
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
		nt.GetTypeName(e.Object),
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

func (c Cluster) Columns() []tlist.Column {
	return []tlist.Column{
		{Title: "Type", Highlight: true},
		{Title: "Name"},
		{Title: "Size"},
	}
}

func (c Cluster) Fields() []string {
	return []string{
		nt.GetTypeName(c.ObjType),
		utils.OrDash(c.ObjName),
		fmt.Sprintf("%d in cluster", len(c.Members)),
	}
}

func (c Cluster) Key() string {
	return fmt.Sprintf("%d:%s", c.ObjType, c.ObjName)
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
