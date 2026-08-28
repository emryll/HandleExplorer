package main

import (
	"fmt"
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
	Cache     map[uint32]map[string]map[uint32]*HandleEntry // pid -> object type -> handle
	TimeStamp int64                                         // last updated
}

// Initialize cache, refilling it. Mutex is handled internally. Concurrency safe method.
func (c *HandleCache) Init() {
	start := time.Now()
	handleTable := GetGlobalHandleTable()
	fmt.Println("[dbg] got handle table")
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Cache == nil {
		c.Cache = make(map[uint32]map[string]map[uint32]*HandleEntry)
	}
	var newEntries []AccessEntry
	for _, handle := range handleTable {
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
