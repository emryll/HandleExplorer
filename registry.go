package main

//?==============================================================================+
//?     This file is responsible for managing and searching object access.       |
//?    It is done using a secondary structure from the raw handle table cache.   |
//?      This is done to optimize each structure for their own purposes.         |
//?==============================================================================+

// Find all objects accessed by several different processes.
// This method will read lock the object access registry.
func (reg *ObjectAccessRegistry) FindOverlapping(filter ClusterFilter) []Cluster {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	var (
		overlapping []Cluster
		accessed    = make(map[ProcessAccessKey][]uint32)
	)

	//TODO: also calculate the stats while youre at it
	//TODO: median, avg, directory distribution, exe distribution

	for pid, objs := range reg.ProcessLookup {
		for key := range objs {
			if filter.ObjType != 0 && key.ObjType != filter.ObjType {
				continue
			}
			if key.Name == "" {
				continue // cant track anon objects currently :(
			}
			if filter.Name != "" && key.Name != filter.Name {
				continue
			}
			accessed[key] = append(accessed[key], pid)
		}
	}

	for key, pids := range accessed {
		cluster := Cluster{
			ObjType: key.ObjType,
			ObjName: key.Name,
		}
		if filter.MinSize > 0 && len(pids) < filter.MinSize {
			continue
		}
		cluster.Members = append(cluster.Members, pids...)
		overlapping = append(overlapping, cluster)
	}

	return overlapping
}

// Add an interaction to the registry or update existing.
// This 'raw' version of the method does not lock the mutex.
func (reg *ObjectAccessRegistry) addEntryRaw(entry AccessEntry) {
	ab := GetBenchmarker("AddEntry")
	if ab != nil {
		stop := ab.Benchmark()
		defer stop()
	}

	// check that maps are initialized (avoid panic)
	if reg.ProcessLookup == nil {
		reg.ProcessLookup = make(map[uint32]map[ProcessAccessKey][]*AccessEntry)
	}
	if reg.ProcessLookup[entry.Pid] == nil {
		reg.ProcessLookup[entry.Pid] = make(map[ProcessAccessKey][]*AccessEntry)
	}
	if reg.ObjectLookup == nil {
		reg.ObjectLookup = make(map[uint32]map[ObjectAccessKey][]*AccessEntry)
	}
	if reg.ObjectLookup[entry.Object] == nil {
		reg.ObjectLookup[entry.Object] = make(map[ObjectAccessKey][]*AccessEntry)
	}

	objectKey := entry.CreateObjectKey()
	processKey := entry.CreateProcessKey()

	// if it already exists, update existing
	if entries, exists := reg.ProcessLookup[entry.Pid][processKey]; exists {
		for _, ent := range entries {
			if ent.Handle == entry.Handle {
				ent.Object |= entry.Object
				return
			}
		}
	}

	e := entry // just to be safe with uniqueness...
	reg.ProcessLookup[e.Pid][processKey] = append(reg.ProcessLookup[e.Pid][processKey], &e)
	reg.ObjectLookup[e.Object][objectKey] = append(reg.ObjectLookup[e.Object][objectKey], &e)
}

// Add an interaction to the registry or update existing.
// This version of the method will write lock the mutex.
func (reg *ObjectAccessRegistry) AddEntry(entry AccessEntry) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.addEntryRaw(entry)
}

// Delete all interaction entries under a certain process.
// This function should be called when a process exits, to cleanup.
func (reg *ObjectAccessRegistry) RemoveEntriesByProcess(pid uint32) {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	if len(reg.ProcessLookup[pid]) == 0 {
		return
	}

	// remove entries
	for psKey, entries := range reg.ProcessLookup[pid] {
		for _, entry := range entries {
			objKey := ObjectAccessKey{Name: psKey.Name, Pid: pid}
			if len(reg.ObjectLookup[uint32(entry.Object)]) > 0 {
				delete(reg.ObjectLookup[uint32(entry.Object)], objKey)
			}
		}
	}
	delete(reg.ProcessLookup, pid)
}

// Find all corresponding entries based on the acting process.
// Set an allowlist for accessing process with a list of pids.
// (optional) Set an allowlist for object type (internal enum OBJ_TYPE_*)
// (optional) Set an allowlist for object names (process path counts as name)
func (reg *ObjectAccessRegistry) FindByProcess(pids []uint32, objs []uint32, names []string, access Bitmask) []*AccessEntry {
	if len(pids) == 0 {
		return nil
	}

	fb := GetBenchmarker("FindByProcess")
	if fb != nil {
		stop := fb.Benchmark()
		defer stop()
	}

	var (
		entries    []*AccessEntry
		typeFilter = make(map[uint32]bool)
		nameFilter = make(map[string]bool)
		pidFilter  = make(map[uint32]bool)
	)

	for _, val := range pids {
		pidFilter[val] = true
	}
	for _, val := range objs {
		typeFilter[val] = true
	}
	for _, val := range names {
		nameFilter[val] = true
	}

	for _, pid := range pids {
		if len(reg.ProcessLookup[pid]) == 0 {
			continue
		}
		for objKey, accessEntries := range reg.ProcessLookup[pid] {
			if len(objs) > 0 && !typeFilter[objKey.ObjType] {
				continue
			}
			if len(names) > 0 && !nameFilter[objKey.Name] {
				continue
			}
			for _, entry := range accessEntries {
				if (Bitmask)(entry.Access).HasFlags(access) {
					entries = append(entries, entry)
				}
			}
		}
	}
	return entries
}

// Find all corresponding entries based on object description.
// Set an allowlist for object types (internal enum ids, OBJ_TYPE_*).
// (optional) Set a filter for required access level bitflags.
// (optional) Set an allowlist for object names. (process path counts as name)
func (reg *ObjectAccessRegistry) FindByObject(objectType []uint32, access Bitmask, names ...string) []*AccessEntry {
	fb := GetBenchmarker("FindByObject")
	if fb != nil {
		stop := fb.Benchmark()
		defer stop()
	}

	var objectMaps []map[ObjectAccessKey][]*AccessEntry
	for _, objType := range objectType {
		if len(reg.ObjectLookup[objType]) > 0 {
			objectMaps = append(objectMaps, reg.ObjectLookup[objType])
		}
	}
	if len(objectMaps) == 0 {
		return nil
	}

	var (
		result     []*AccessEntry
		nameFilter = make(map[string]bool)
	)
	for _, name := range names {
		nameFilter[name] = true
	}

	for _, typeMap := range objectMaps {
		for key, entries := range typeMap {
			if len(names) > 0 && !nameFilter[key.Name] {
				continue
			}
			for _, entry := range entries {
				if (Bitmask)(entry.Access).HasFlags(access) {
					result = append(result, entry)
				}
			}
		}
	}
	return result
}

func (entry *AccessEntry) CreateObjectKey() ObjectAccessKey {
	return ObjectAccessKey{Name: entry.Name, Pid: entry.Pid}
}

func (entry *AccessEntry) CreateProcessKey() ProcessAccessKey {
	return ProcessAccessKey{Name: entry.Name, ObjType: entry.Object}
}

/*
func (reg *ObjectAccessRegistry) PrintStatus() {
	//TODO: print how many entries there are
	//TODO: print whether you are keeping cache active.
}
*/
