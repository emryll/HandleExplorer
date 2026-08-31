package main

import "sort"

// Get a list of all active processes
// sorted in descending order of handle count.
func RankProcessHandleCount() []*Process {
	var processes []*Process

	g_ProcessTable.mu.RLock()
	defer g_ProcessTable.mu.RUnlock()

	for _, ps := range g_ProcessTable.Table {
		processes = append(processes, ps)
	}
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].HandleCount > processes[j].HandleCount
	})

	return processes
}

//TODO FindMostWideReaching
