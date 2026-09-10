package stats

import (
	"sync"
)

//TODO FindMostWideReaching

// Cached stats about current state.
type SessionStats struct {
	sync.RWMutex
	TotalActiveHandles   int
	TotalActiveProcesses int
}

// Get the total handle count (global).
// This will read lock the session stats.
func GetTotalHandleCount(rtStats *SessionStats) int {
	rtStats.RLock()
	defer rtStats.RUnlock()
	return rtStats.TotalActiveHandles
}
func (s *SessionStats) SetProcessCount(count int) {
	s.Lock()
	defer s.Unlock()
	s.TotalActiveProcesses = count
}

func (s *SessionStats) SetHandleCount(count int) {
	s.Lock()
	defer s.Unlock()
	s.TotalActiveHandles = count
}
