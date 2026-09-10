package stats

import (
	_ "HandleExplorer/app"
	"sort"
	"sync"
)

// Get a list of all active processes
// sorted in descending order of handle count.
func RankProcessHandleCount() []*ps.Process {
	var processes []*ps.Process

	PsTable.mu.RLock()
	defer PsTable.mu.RUnlock()

	for _, ps := range PsTable.Table {
		processes = append(processes, ps)
	}
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].GetHandleCount() > processes[j].GetHandleCount()
	})

	return processes
}

//TODO FindMostWideReaching

// Cached stats about current state.
type SessionStats struct {
	mu                   sync.RWMutex
	TotalActiveHandles   int
	TotalActiveProcesses int
}

// Get the total handle count (global).
// This will read lock the session stats.
func GetTotalHandleCount() int {
	RuntimeStats.mu.RLock()
	defer RuntimeStats.mu.RUnlock()
	return RuntimeStats.TotalActiveHandles
}
func (s *SessionStats) SetProcessCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalActiveProcesses = count
}

func (s *SessionStats) SetHandleCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalActiveHandles = count
}
