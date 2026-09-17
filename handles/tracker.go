package handles

import (
	"HandleExplorer/handles/registry"
	"HandleExplorer/process"
	"HandleExplorer/profiler"
	"HandleExplorer/stats"
	"sync"
	"time"
)

type AccessTracker struct {
	HandleTable    *HandleCache
	AccessRegistry *registry.ObjectAccessRegistry

	refreshMu sync.RWMutex // for state check/set
	// nil indicates ready-state
	// non-nil means a refresh is in progress.
	refreshing chan struct{}
}

//*============================[ Tracking ]=============================

// Search wrapper for CLI find commands.
func (at *AccessTracker) Search(f HandleFilter) []*registry.AccessEntry {
	at.RefreshIfStale()
	at.WaitReady()

	at.AccessRegistry.RLock()
	defer at.AccessRegistry.RUnlock()

	if len(f.Pids) != 0 {
		return at.AccessRegistry.FindByProcess(f.Pids, f.ObjType, f.Names, f.Access)
	} else if len(f.ObjType) != 0 {
		return at.AccessRegistry.FindByObject(f.ObjType, f.Access, f.Names...)
	} else {
		return nil
	}
}

// Get all pids that accessed a named object.
func (at *AccessTracker) GetObjectAccessPids(objType uint32, name string) []uint32 {
	at.AccessRegistry.Lock()
	defer at.AccessRegistry.Unlock()

	if len(at.AccessRegistry.ObjectLookup[objType]) == 0 {
		return nil
	}
	var (
		pids []uint32
		seen = make(map[uint32]bool)
	)
	for key := range at.AccessRegistry.ObjectLookup[objType] {
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
func (at *AccessTracker) GetObjectTypesAccessed(pid uint32) map[uint32]bool {
	at.AccessRegistry.RLock()
	defer at.AccessRegistry.RUnlock()

	if len(at.AccessRegistry.ProcessLookup[pid]) == 0 {
		return nil
	}

	accessed := make(map[uint32]bool)
	for key := range at.AccessRegistry.ProcessLookup[pid] {
		accessed[key.ObjType] = true
	}
	return accessed
}

// Get total count of handles currently cached.
// This will read lock the global handle cache.
func (at *AccessTracker) GetTotalCachedHandleCount() int {
	at.HandleTable.RLock()
	defer at.HandleTable.RUnlock()

	var total int
	for _, typeMap := range at.HandleTable.Cache {
		for _, handleMap := range typeMap {
			total += len(handleMap)
		}
	}
	return total
}

//*=======================[ Refresh data ]===========================

func (at *AccessTracker) Refresh() {
	cb := profiler.GetBenchmarker("CacheRefresh")
	if cb != nil {
		stop := cb.Benchmark()
		defer stop()
	}

	at.SetRefresh()
	defer at.SetReady()

	handleTable := GetGlobalHandleTable()
	at.HandleTable.rs.SetHandleCount(len(handleTable))

	at.HandleTable.Lock()
	defer at.HandleTable.Unlock()

	var (
		cache = make(map[uint32]map[uint32]map[uint32]*HandleEntry)
		// total active handle counts are
		// collected here, because it is
		// hard to collect anywhere else.
		psCounts = make(map[uint32]int)
	)

	at.AccessRegistry.Lock()
	for _, handle := range handleTable {
		if cache[handle.Pid] == nil {
			cache[handle.Pid] = make(map[uint32]map[uint32]*HandleEntry)
		}
		if cache[handle.Pid][handle.Type] == nil {
			cache[handle.Pid][handle.Type] = make(map[uint32]*HandleEntry)
		}

		cache[handle.Pid][handle.Type][handle.Handle] = &handle
		at.AccessRegistry.AddEntryRaw(handle.ConvertToAccessEntry())
		psCounts[handle.Pid]++
	}
	at.AccessRegistry.Unlock()
	at.HandleTable.Cache = cache
	at.HandleTable.TimeStamp = time.Now()

	at.HandleTable.pst.UpdatePsHandleCount(psCounts)
}

func (at *AccessTracker) RefreshIfStale() {
	if at.HandleTable.IsStale() {
		at.Refresh()
	}
}

//? The access tracker has waiting functionality
//? because the cache refresh is expensive (>1s), so
//? its done in the background, but object access data
//? should not be used until it is ready.

//? Call AccessTracker.WaitReady() before using the data,
//? in order to guarantee the data has not gone stale.

// Set access tracker state as being refreshed.
// Call CacheReady when the refresh has finished.
// Return value true indicates refresh was set,
// while false indicates refresh is in progress.
// Only refresh the cache if this returns true!

func (at *AccessTracker) SetRefresh() bool {
	at.refreshMu.Lock()
	defer at.refreshMu.Unlock()

	if at.refreshing == nil {
		at.refreshing = make(chan struct{})
		return true
	}
	return false
}

// Set the handle cache state as ready.
// This WILL write lock the handle cache mutex.
func (at *AccessTracker) SetReady() {
	at.refreshMu.Lock()
	defer at.refreshMu.Unlock()

	if at.refreshing != nil {
		close(at.refreshing)
		// a closed channel is non-nil,
		// but closing it causes panic
		at.refreshing = nil
	}
}

// Wait until the handle cache has finished refresh.
// If there is no refresh, it will immediately return.
// This WILL read-lock the handle cache mutex (quick)
func (at *AccessTracker) WaitReady() {
	at.refreshMu.RLock()
	refreshing := at.refreshing
	at.refreshMu.RUnlock()

	if refreshing == nil {
		return
	}

	// wait until signalled
	<-refreshing
}

//*==========================[ Helpers ]================================

func (f HandleFilter) Empty() bool {
	if len(f.ObjType) == 0 && len(f.Pids) == 0 && len(f.Names) == 0 {
		return true
	}
	return false
}

// this is for solving dependency cycles...
// HandleRemover cleanup interface used in process package
func (a *AccessTracker) Remove(pid uint32) {
	a.HandleTable.Remove(pid)
	a.AccessRegistry.RemoveEntriesByProcess(pid)
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
