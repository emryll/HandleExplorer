package handles

import (
	"HandleExplorer/handles/registry"
	"HandleExplorer/process"
	"HandleExplorer/stats"
)

type AccessTracker struct {
	HandleTable    *HandleCache
	AccessRegistry *registry.ObjectAccessRegistry
}

func NewAccessTracker(pst *process.ProcessTable, rs *stats.SessionStats) *AccessTracker {
	var (
		reg   = &registry.ObjectAccessRegistry{}
		cache = &HandleCache{
			pst: pst,
			reg: reg,
			rs:  rs,
		}
	)

	return &AccessTracker{
		HandleTable:    cache,
		AccessRegistry: reg,
	}
}

// Search wrapper for CLI find commands.
func (a *AccessTracker) Search(f HandleFilter) []*registry.AccessEntry {
	if !a.HandleTable.Valid() {
		a.HandleTable.Init()
	}
	a.HandleTable.WaitReady()

	a.AccessRegistry.RLock()
	defer a.AccessRegistry.RUnlock()

	if len(f.Pids) != 0 {
		return a.AccessRegistry.FindByProcess(f.Pids, f.ObjType, f.Names, f.Access)
	} else if len(f.ObjType) != 0 {
		return a.AccessRegistry.FindByObject(f.ObjType, f.Access, f.Names...)
	} else {
		return nil
	}
}

func (f HandleFilter) Empty() bool {
	if len(f.ObjType) == 0 && len(f.Pids) == 0 && len(f.Names) == 0 {
		return true
	}
	return false
}

// Get all pids that accessed a named object.
func (a *AccessTracker) GetObjectAccessPids(objType uint32, name string) []uint32 {
	a.AccessRegistry.Lock()
	defer a.AccessRegistry.Unlock()

	if len(a.AccessRegistry.ObjectLookup[objType]) == 0 {
		return nil
	}
	var (
		pids []uint32
		seen = make(map[uint32]bool)
	)
	for key := range a.AccessRegistry.ObjectLookup[objType] {
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
func (a *AccessTracker) GetObjectTypesAccessed(pid uint32) map[uint32]bool {
	a.AccessRegistry.RLock()
	defer a.AccessRegistry.RUnlock()

	if len(a.AccessRegistry.ProcessLookup[pid]) == 0 {
		return nil
	}

	accessed := make(map[uint32]bool)
	for key := range a.AccessRegistry.ProcessLookup[pid] {
		accessed[key.ObjType] = true
	}
	return accessed
}

// Get total count of handles currently cached.
// This will read lock the global handle cache.
func (a *AccessTracker) GetTotalCachedHandleCount() int {
	a.HandleTable.mu.RLock()
	defer a.HandleTable.mu.RUnlock()

	var total int
	for _, typeMap := range a.HandleTable.Cache {
		for _, handleMap := range typeMap {
			total += len(handleMap)
		}
	}
	return total
}

// this is for solving dependency cycles...
// HandleRemover cleanup interface used in process package
func (a *AccessTracker) Remove(pid uint32) {
	a.HandleTable.Remove(pid)
	a.AccessRegistry.RemoveEntriesByProcess(pid)
}
