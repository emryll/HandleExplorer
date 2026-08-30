package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

//*=================================[ Handle Table Cache ]=================================

var HandleTable HandleCache

// Since currently handles are retrieved
// via global handle table lookup,
// which is expensive; caching is used.
// You should only use the mutex if you
// directly access cache, never for methods.
type HandleCache struct {
	mu        sync.RWMutex
    // pid -> objType (NT name) -> handle raw value
	Cache     map[uint32]map[string]map[uint32]*HandleEntry // pid -> object type -> handle
	TimeStamp int64                                         // last updated

    refreshMu sync.RWMutex // for state check/set
    // nil indicates ready-state
    // non-nil means a refresh is in progress.
    refreshing chan struct{}
}

//? The handle cache has waiting functionality
//? because the cache refresh is expensive, so
//? it is done in the background, but object
//? access data should not be used until it is ready.

//? Call HandleTable.WaitReady() before using the OAR,
//? in order to guarantee the data has not gone stale.

// Set handle cache state as being refreshed.
// Call CacheReady when the refresh has finished.
// This WILL write lock the handle cache mutex.
func (c *HandleCache) SetRefresh() {
    c.refreshMu.Lock()
    defer c.refreshMu.Unlock()

    if c.refreshing == nil {
        c.refreshing = make(chan struct{})
    }
}

// Set the handle cache state as ready.
// This WILL write lock the handle cache mutex.
func (c *HandleCache) SetReady() {
    c.refreshMu.Lock()
    defer c.refreshMu.Unlock()

    if c.refreshing != nil {
        close(c.refreshing)
    }
}

// Wait until the handle cache has finished refresh.
// If there is no refresh, it will immediately return.
// This WILL read-lock the handle cache mutex (quick)
func (c *HandleCache) WaitReady() {
    c.refreshMu.RLock()
    refreshing := c.refreshing
    c.refreshMu.RUnlock()

    if refreshing == nil {
        return
    }

    // wait until signalled
    <-refreshing
}

// Initialize cache, refilling it. Mutex is handled internally. Concurrency safe method.
func (c *HandleCache) Init() {
    c.SetRefresh()

	start := time.Now() //dbg
	handleTable := GetGlobalHandleTable()

	c.mu.Lock()
	if c.Cache == nil {
		c.Cache = make(map[uint32]map[string]map[uint32]*HandleEntry)
	}

	var (
        newEntries []AccessEntry
        // total active handle counts are
        // collected here, because it is
        // hard to collect anywhere else.
        psCounts = make(map[uint32]int)
    )

	for _, handle := range handleTable {
        psCounts[handle.Pid]++
		if c.Cache[handle.Pid] == nil {
			c.Cache[handle.Pid] = make(map[string]map[uint32]*HandleEntry)
		}

		objectType := GetTypeName(handle.Type)
		if c.Cache[handle.Pid][objectType] == nil {
			c.Cache[handle.Pid][objectType] = make(map[uint32]*HandleEntry)
		}

		if entry, exists := c.Cache[handle.Pid][objectType][handle.Handle]; exists {
			entry.LastSeen = time.Now().UnixMilli()
			//TODO: if there is a new access flag then add that to OAC
		} else {
			c.Cache[handle.Pid][objectType][handle.Handle] = &handle
			entry := handle.ConvertToAccessEntry()
			newEntries = append(newEntries, entry)
		}
	}
	c.mu.Unlock()


	fmt.Printf("[dbg] found %d new entries\n", len(newEntries))
	//TODO: THESE LOCKS ARE CAUSING AN INDEFINITE STALL
	//TODO: FOR SOME REASON ON THE SECOND INIT, NOT FIRST
	g_ObjectAccessRegistry.mu.Lock()
	defer g_ObjectAccessRegistry.mu.Unlock()
	for _, entry := range newEntries {
		//fmt.Printf("[dbg] add entry raw")
		g_ObjectAccessRegistry.addEntryRaw(entry)
	}
	fmt.Printf("[dbg] Init took %dms\n", time.Since(start).Milliseconds())
	c.TimeStamp = time.Now().Unix()
    c.mu.SetReady()

    g_ProcessTable.UpdatePsHandleCount(psCounts)
}

// Is handle table cache ready for use. Mutex is handled internally
func (c *HandleCache) Valid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Cache == nil {
		return false
	}
	return c.TimeStamp >= (time.Now().Unix() - int64(HANDLE_CACHE_EXPIRATION))
}

// Add new handle entry into cache, or update existing
func (c *HandleCache) Add(handle *HandleEntry) {
	if handle.FirstSeen == 0 {
		handle.FirstSeen = time.Now().UnixMilli()
	}
	if handle.LastSeen == 0 {
		handle.LastSeen = handle.FirstSeen
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Cache[handle.Pid] == nil {
		c.Cache[handle.Pid] = make(map[string]map[uint32]*HandleEntry)
	}
	objectType := GetTypeName(handle.Type)
	if c.Cache[handle.Pid][objectType] == nil {
		c.Cache[handle.Pid][objectType] = make(map[uint32]*HandleEntry)
	}

	//* Add handle, or update existing
	if entry, exists := c.Cache[handle.Pid][objectType][handle.Handle]; exists {
		entry.LastSeen = time.Now().UnixMilli()
	} else {
		c.Cache[handle.Pid][objectType][handle.Handle] = handle
	}
}

// Remove all handle entries held by given process
func (c *HandleCache) Remove(pid uint32) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	delete(c.Cache, pid)
}

func (c *HandleCache) getPsHandleCountsByType(pid uint32) map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	results := make(map[string]int)
	for objType, typeMap := range c.Cache[pid] {
		if len(typeMap) > 0 {
			results[objType] = len(typeMap)
		}
	}
	return results
}

//*====================================[ Cache Cleanup ]=========================================

// Get the base priority score for the given object.
// Used to calculate priority for score based handle cache cleanup.
func (handle *HandleEntry) GetObjectScore() int {
	var tier int
	if val, exists := ObjectTypeTier[handle.Type]; exists {
		tier = val
	}
	switch tier {
	case 1:
		return HCC_OBJECT_TIER_1_SCORE
	case 2:
		return HCC_OBJECT_TIER_2_SCORE
	case 3:
		return HCC_OBJECT_TIER_3_SCORE
	case 4:
		return HCC_OBJECT_TIER_4_SCORE
	}
	return HCC_OBJECT_UNKNOWN_SCORE
}

// Calculate the final priority of a handle entry.
// Priority is based on age of handle and base priority of object.
// A higher priority means that it should be cleaned up sooner.
func (handle *HandleEntry) CalculatePriority(stamp ...int64) int {
	var now int64
	if len(stamp) == 0 {
		now = time.Now().UnixMilli()
	} else {
		now = stamp[0]
	}
	timeElapsed := int(now - handle.LastSeen)
	timeScore := int(math.Pow(float64(timeElapsed)/1000, HCC_TIME_POWER))
	timeScore *= HCC_TIME_MULTIPLIER

	objectScore := handle.GetObjectScore()
	objectScore *= HCC_OBJECT_MULTIPLIER

	// should you do multiplication or addition?
	return (timeScore + objectScore) * HCC_MULTIPLIER_CONST
}

func (c *HandleCache) Cleanup(quota ...int) {
	var cleanupQuota int
	if len(quota) == 0 {
		cleanupQuota = HCC_DEFAULT_QUOTA
	} else {
		cleanupQuota = quota[0]
	}

	c.mu.RLock()
	// helper struct to avoid recalculating priority
	type handleWithPriority struct {
		handle   *HandleEntry
		priority int
	}

	//* get cache as slice
	now := time.Now().UnixMilli()
	handles := make([]handleWithPriority, 0, 5000)
	for _, objectCache := range c.Cache {
		for _, entries := range objectCache {
			for _, handle := range entries {
				handles = append(handles, handleWithPriority{
					handle: handle, priority: handle.CalculatePriority(now)})
			}
		}
	}
	c.mu.RUnlock()
	//* sort handles in descending priority
	sort.Slice(handles, func(i, j int) bool {
		return handles[i].priority > handles[j].priority
	})
	c.mu.Lock()
	defer c.mu.Unlock()
	//* cleanup highest priority handles until quota is met
	for i := 0; i < cleanupQuota && i < len(handles); i++ {
		var (
			pid        = handles[i].handle.Pid
			handle     = handles[i].handle.Handle
			objectType = GetTypeName(handles[i].handle.Type)
		)
		// between mutex it couldve been deleted
		if _, exists := c.Cache[pid]; !exists {
			continue
		}
		if _, exists := c.Cache[pid][objectType]; !exists {
			continue
		}
		// delete the handle entry
		delete(c.Cache[pid][objectType], handle)
		if len(c.Cache[pid][objectType]) == 0 {
			delete(c.Cache[pid], objectType)
		}
		if len(c.Cache[pid]) == 0 {
			delete(c.Cache, pid)
		}
	}
}

func (c *HandleCache) CacheCleaner(wg *sync.WaitGroup, ctx context.Context) {
	//TODO:
}
