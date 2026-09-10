package registry

import (
	"HandleExplorer/profiler"
	"HandleExplorer/utils"
)

//?==============================================================================+
//?     This file is responsible for managing and searching object access.       |
//?    It is done using a secondary structure from the raw handle table cache.   |
//?      This is done to optimize each structure for their own purposes.         |
//?==============================================================================+

// Add an interaction to the registry or update existing.
// This 'raw' version of the method does not lock the mutex.
func (reg *ObjectAccessRegistry) AddEntryRaw(entry AccessEntry) {
	ab := profiler.GetBenchmarker("AddEntry")
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
				ent.Access |= entry.Access
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
	reg.Lock()
	defer reg.Unlock()
	reg.AddEntryRaw(entry)
}

// Delete all interaction entries under a certain process.
// This function should be called when a process exits, to cleanup.
func (reg *ObjectAccessRegistry) RemoveEntriesByProcess(pid uint32) {
	reg.Lock()
	defer reg.Unlock()

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
func (reg *ObjectAccessRegistry) FindByProcess(pids []uint32, objs []uint32, names []string, access utils.Bitmask) []*AccessEntry {
	if len(pids) == 0 {
		return nil
	}

	fb := profiler.GetBenchmarker("FindByProcess")
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
				if entry.Access.HasFlags(access) {
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
func (reg *ObjectAccessRegistry) FindByObject(objectType []uint32, access utils.Bitmask, names ...string) []*AccessEntry {
	fb := profiler.GetBenchmarker("FindByObject")
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
	//TODO: add all instead if len(objectType) == 0
	if len(objectMaps) == 0 {
		return nil
	}

	var (
		result     []*AccessEntry
		nameFilter = make(map[string]bool)
	)
	for _, name := range names {
		if utils.IsEmptyName(name) {
			continue
		}
		nameFilter[name] = true
	}

	for _, typeMap := range objectMaps {
		for key, entries := range typeMap {
			if len(nameFilter) > 0 && !nameFilter[key.Name] {
				continue
			}
			for _, entry := range entries {
				if entry.Access.HasFlags(access) {
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
