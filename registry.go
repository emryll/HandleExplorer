package main

import "fmt"

func (reg *ObjectAccessRegistry) PrintOverlapping() {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	accessed := make(map[ProcessAccessKey][]uint32)
	for pid, objs := range reg.ProcessLookup {
		for key := range objs {
			accessed[key] = append(accessed[key], pid)
		}
	}

	var (
		types          = make(map[uint32]int)
		overlap_counts []int
		total_counts   int
		overlapping    int
		largest        int
	)

	// Print overlapping
	for key, pids := range accessed {
		if len(pids) < 2 {
			continue
		}
		overlapping++
		types[key.ObjType]++
		total_counts += len(pids)
		overlap_counts = append(overlap_counts, len(pids))
		if len(pids) > largest {
			largest = len(pids)
		}

		fmt.Printf("\n%s\n", stars)
		fmt.Printf("\nOverlapping access to %s\n", GetTypeName(key.ObjType))
		fmt.Printf("\tName: %s\n", key.Name)
		fmt.Printf("\nPids: ")
		for _, pid := range pids {
			fmt.Printf("%d ", pid)
		}
		fmt.Println()
	}
	fmt.Printf("\n%s\n", stars)

	var (
		median float32
		avg    float32
	)

	avg = float32(total_counts) / float32(overlapping)

	if overlapping%2 == 0 {
		upperMidIndex := overlapping / 2
		totalMiddle := overlap_counts[upperMidIndex-1]
		totalMiddle += overlap_counts[upperMidIndex]
		median = float32(totalMiddle) / 2
	} else {
		midIndex := overlapping / 2
		median = float32(overlap_counts[midIndex])
	}

	fmt.Printf("\n\t[ Total of %d overlapping ; avg size %.1f ; median size %.1f ]\n", overlapping, avg, median)
	fmt.Printf("\n%s\n", stars)
	fmt.Println("\tObject type statistics:\n")
	for objType, count := range types {
		fmt.Printf("\t* %s : %d instances\n", GetTypeName(objType), count)
	}
	fmt.Printf("\n%s\n", stars)
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
