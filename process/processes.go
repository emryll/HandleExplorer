package process

import (
	_ "HandleExplorer/app"
	"HandleExplorer/utils"
	"context"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

//?=======================================================================+
//?    This file is responsible for tracking details about processes      |
//?=======================================================================+

//*=========================[ Process Scanner ]=============================

//? Processes are scanned periodically and their details are cached,
//?  just so that this data is not constantly queried from the OS.

func NewProcessTable() *ProcessTable {
	return &ProcessTable{Table: make(map[uint32]*Process)}
}

// Main scanner routine for tracking active processes and their details.
func ProcessScanner(wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	if err := ScanProcesses(wg); err != nil {
		utils.PrintError("Failed to scan processes: %v\n", err)
	}

	refresh := time.NewTicker(time.Duration(PS_REFRESH_INTERVAL) * time.Second)
	defer refresh.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-refresh.C:
			if err := ScanProcesses(wg); err != nil {
				utils.PrintError("Failed to scan processes: %v\n", err)
			}
		}
	}
}

// Scan processes via th32 snapshot and add them to the process table. One time.
// This will also check for any dead processes (without callbacks).
func ScanProcesses(wg *sync.WaitGroup) error {
	//* get a process snapshot
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	processes := make(map[uint32]*windows.ProcessEntry32)
	var scanWg sync.WaitGroup

	//* walk the processes
	for {
		err = windows.Process32Next(handle, &entry)
		if err != nil {
			if err.Error() == "There are no more files." {
				break
			}
			return err
		}

		// entry can change since this loop
		// continues while goroutines launch
		entryCopy := entry
		processes[entry.ProcessID] = &entryCopy

		scanWg.Add(1)
		// launching a goroutine because it is
		// possible for WinVerifyTrustEx to stall
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			RegisterProcess(&entryCopy)
		}(&scanWg)
	}
	g_SessionStats.SetProcessCount(len(processes))
	// This tool doesn't have a driver for callbacks, so...
	ScanForDeadProcesses(processes)
	return nil
}

// Add process to process table and the correct graph.
// If the process already exists, any missing data is filled.
func RegisterProcess(entry *windows.ProcessEntry32) {
	name := windows.UTF16ToString(entry.ExeFile[:])
	if ps := PsTable.LookupProcess(entry.ProcessID); ps != nil {
		if filepath.Base(ps.Path) == name {
			return // technically could still be different...
		}
		// new process with same pid, clear old one
		HandleTable.Remove(entry.ProcessID)
		AccessRegistry.RemoveEntriesByProcess(entry.ProcessID)
	}
	process := CreateProcessEntry(entry)
	PsTable.AddProcess(process)
}

// Create an initial process entry with basic details.
// This does not add the process entry to the process table.
func CreateProcessEntry(pe32 *windows.ProcessEntry32) *Process {
	entry := Process{
		ProcessId: pe32.ProcessID,
		ParentPid: pe32.ParentProcessID,
	}

	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pe32.ProcessID)
	if err == nil {
		defer windows.CloseHandle(handle)
		path, err := GetProcessExecutable(handle)
		if err == nil {
			entry.Path = path
			// check signature only if full path is known,
			// otherwise this could be fooled by having
			// a different file of same name in working dir
			status, err := IsSignedWithTimeout(path)
			if err == nil {
				entry.SigStatus = status
			}
		}
		elevated, err := IsProcessElevated(handle)
		if err == nil {
			entry.Elevated = elevated
		}
	} /* else {
		PrintError("Failed to open process %d: %v\n", entry.ProcessId, err)
	}*/

	if entry.Path == "" {
		entry.Path = windows.UTF16ToString(pe32.ExeFile[:])
	}

	if entry.ParentPid == 0 || entry.ParentPid == 4 {
		return &entry
	}

	parentHandle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pe32.ParentProcessID)
	// parent basic info
	if err == nil {
		defer windows.CloseHandle(parentHandle)
		parentPath, err := GetProcessExecutable(parentHandle)
		if err == nil {
			entry.ParentPath = parentPath
		}
	}
	return &entry
}

func ScanForDeadProcesses(processes map[uint32]*windows.ProcessEntry32) {
	if len(processes) == 0 {
		return
	}
	if PsTable == nil || PsTable.Table == nil {
		utils.PrintWithRedLabel("[WARNING]", "Global process table not initialized!!")
	}

	PsTable.mu.RLock()
	var dead []uint32 // collect for shorter lock time
	//* make sure all processes are found in the process snapshot
	for pid := range PsTable.Table {
		if _, exists := processes[pid]; exists {
			continue
		}
		AccessRegistry.RemoveEntriesByProcess(pid)
		HandleTable.Remove(pid)
		dead = append(dead, pid)
	}
	PsTable.mu.RUnlock()

	PsTable.mu.Lock()
	defer PsTable.mu.Unlock()
	for _, pid := range dead {
		PsTable.RemoveProcess(pid)
	}
}

//*======================[ Process Table ]==============================

func (ps *ProcessTable) LookupProcess(pid uint32) *Process {
	if ps.Table == nil {
		return nil
	}
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	if process, exists := ps.Table[pid]; exists {
		return process
	}
	return nil
}

func (ps *ProcessTable) AddProcess(process *Process) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.Table == nil {
		ps.Table = make(map[uint32]*Process)
	}
	ps.Table[process.ProcessId] = process
}

// Does not lock the mutex
func (ps *ProcessTable) RemoveProcess(pid uint32) {
	if ps.Table == nil {
		return
	}
	delete(ps.Table, pid)
}

// Find all processes (IN THE OAR) with the
// specified executable name or full path.
func FindProcesses(name string) []uint32 {
	AccessRegistry.mu.RLock()
	defer AccessRegistry.mu.RUnlock()

	var pids []uint32
	for pid := range AccessRegistry.ProcessLookup {
		path := LookupProcessPath(pid)
		if name == path || name == filepath.Base(path) {
			pids = append(pids, pid)
		}
	}
	return pids
}

// Get the path of a processes source exe file.
// Looked up in the process table if it exists.
// If it does not, the process is looked up via win32.
func LookupProcessPath(pid uint32) string {
	if ps := PsTable.LookupProcess(pid); ps != nil {
		return ps.Path
	}

	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(handle)
	if path, err := GetProcessExecutable(handle); err == nil {
		return path
	}
	return ""
}

func LookupParent(pid uint32) (uint32, string) {
	ps := PsTable.LookupProcess(pid)
	if ps != nil {
		return ps.ParentPid, ps.ParentPath
	}

	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return 0, ""
	}
	defer windows.CloseHandle(handle)

	ppid, err := GetParentPid(handle)
	if err == nil {
		return ppid, ""
	}
	return 0, ""
}

//*=======================[ Stats ]=======================

// Update the active handle counts for each process listed in
// psCounts, where the key is the pid and value is the handle count.
func (pt *ProcessTable) UpdatePsHandleCount(psCounts map[uint32]int) {
	if len(psCounts) == 0 {
		return
	}
	pt.mu.Lock()
	defer pt.mu.Unlock()

	for pid, count := range psCounts {
		if ps, exists := pt.Table[pid]; exists {
			ps.HandleCount.Store(int64(count))
		}
	}
}

func (p *Process) GetHandleCount() int {
	return int(p.HandleCount.Load())
}

// Get the total count of active processes.
// This will read lock the process table.
func GetTotalProcessCount() int {
	PsTable.mu.RLock()
	defer PsTable.mu.RUnlock()
	return len(PsTable.Table)
}

// Get the total handle count of a process.
// This will read lock the process table.
func GetHandleCountPs(pid uint32) int {
	ps := PsTable.LookupProcess(pid)
	if ps == nil {
		return 0
	}
	return ps.GetHandleCount()
}
