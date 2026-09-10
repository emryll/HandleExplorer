package process

import (
	"HandleExplorer/utils"
	"context"
	"path/filepath"
	"strconv"
	"strings"
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
func ProcessScanner(wg *sync.WaitGroup, ctx context.Context, pt *ProcessTable, hr HandleRemover) {
	defer wg.Done()
	if err := ScanProcesses(wg, pt, hr); err != nil {
		utils.PrintError("Failed to scan processes: %v\n", err)
	}

	refresh := time.NewTicker(time.Duration(PS_REFRESH_INTERVAL) * time.Second)
	defer refresh.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-refresh.C:
			if err := ScanProcesses(wg, pt, hr); err != nil {
				utils.PrintError("Failed to scan processes: %v\n", err)
			}
		}
	}
}

// Scan processes via th32 snapshot and add them to the process table. One time.
// This will also check for any dead processes (without callbacks).
func ScanProcesses(wg *sync.WaitGroup, pt *ProcessTable, hr HandleRemover) error {
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
			if pt != nil {
				pt.RegisterProcess(&entryCopy, hr)
			}
		}(&scanWg)
	}
	pt.RtStats.SetProcessCount(len(processes))
	// This tool doesn't have a driver for callbacks, so...
	if pt != nil {
		pt.ScanForDeadProcesses(processes, hr)
	}
	return nil
}

// interface for process cleanup,
// avoiding dependency cycles...
// AccessTracker implements this.
type HandleRemover interface {
	Remove(pid uint32)
}

// Add process to process table and the correct graph.
// If the process already exists, any missing data is filled.
func (pt *ProcessTable) RegisterProcess(entry *windows.ProcessEntry32, hr HandleRemover) {
	name := windows.UTF16ToString(entry.ExeFile[:])
	if ps := pt.LookupProcess(entry.ProcessID); ps != nil {
		if filepath.Base(ps.Path) == name {
			return // technically could still be different...
		}
		// new process with same pid, clear old one
		hr.Remove(entry.ProcessID)
	}
	process := CreateProcessEntry(entry)
	pt.AddProcess(process)
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

func (pt *ProcessTable) ScanForDeadProcesses(processes map[uint32]*windows.ProcessEntry32, hr HandleRemover) {
	if len(processes) == 0 {
		return
	}
	if pt.Table == nil {
		utils.PrintWithRedLabel("[WARNING]", "Global process table not initialized!!")
	}

	pt.RLock()
	var dead []uint32 // collect for shorter lock time
	//* make sure all processes are found in the process snapshot
	for pid := range pt.Table {
		if _, exists := processes[pid]; exists {
			continue
		}
		hr.Remove(pid)
		dead = append(dead, pid)
	}
	pt.RUnlock()

	pt.Lock()
	defer pt.Unlock()
	for _, pid := range dead {
		pt.RemoveProcess(pid)
	}
}

//*======================[ Search filter (CLI) ]========================

func (pt *ProcessTable) Search(filter ProcessFilter) []*Process {
	pt.RLock()
	defer pt.RUnlock()
	var results []*Process

	//* quick lookup, used only when pid is provided
	if len(filter.Pids) > 0 {
		for pid := range filter.Pids {
			if ps, exists := pt.Table[pid]; exists && filter.Passes(ps) {
				results = append(results, ps)
			}
		}
		return results
	}
	//* regular lookup
	for _, ps := range pt.Table {
		if filter.Passes(ps) {
			results = append(results, ps)
		}
	}
	return results
}

func (f ProcessFilter) Passes(ps *Process) bool {
	//* Process Id
	if len(f.Pids) > 0 && !f.Pids[ps.ProcessId] {
		return false
	}

	//* Path / Name
	if f.Path != "" && ps.Path != f.Path &&
		f.Path != filepath.Base(ps.Path) &&
		filepath.Base(f.Path) != ps.Path {
		return false
	}

	//* Directory
	// does not qualify if it has no dir listed
	// while the directory filter has been set
	if f.Path == filepath.Base(f.Path) && len(f.DirFilter) > 0 {
		return false
	}

	var dirFound bool
	for _, dir := range f.DirFilter {
		if strings.HasPrefix(f.Path, dir) { // allow subdirs
			dirFound = true
			break
		}
	}
	if len(f.DirFilter) > 0 && !dirFound {
		return false
	}

	//* Parent
	var parentFound bool
	if f.Parent[ps.ParentPath] || f.Parent[filepath.Base(ps.ParentPath)] {
		parentFound = true
	}

	for parent := range f.Parent {
		if parent == "" {
			continue
		}
		pid, err := strconv.Atoi(parent)
		if err == nil && uint32(pid) == ps.ParentPid {
			parentFound = true
			break
		}
	}
	if len(f.Parent) > 0 && !parentFound {
		return false
	}

	//* Process elevation
	if f.Elevated && !ps.Elevated {
		return false
	}
	if f.NotElevated && ps.Elevated {
		return false
	}

	//* Signature status
	if len(f.SigStatus) > 0 && !f.SigStatus[ps.SigStatus] {
		return false
	}

	//* Accessed objects
	accessed := f.reg.GetObjectTypesAccessed(ps.ProcessId)
	for objType := range f.ObjTypes {
		if !accessed[objType] {
			return false
		}
	}
	return true
}

//*======================[ Process Table ]==============================

func (pt *ProcessTable) LookupProcess(pid uint32) *Process {
	if pt.Table == nil {
		return nil
	}
	pt.RLock()
	defer pt.RUnlock()
	if process, exists := pt.Table[pid]; exists {
		return process
	}
	return nil
}

func (pt *ProcessTable) AddProcess(process *Process) {
	pt.Lock()
	defer pt.Unlock()

	if pt.Table == nil {
		pt.Table = make(map[uint32]*Process)
	}
	pt.Table[process.ProcessId] = process
}

// Does not lock the mutex
func (pt *ProcessTable) RemoveProcess(pid uint32) {
	if pt.Table == nil {
		return
	}
	delete(pt.Table, pid)
}

// Find all processes with the
// specified executable name or full path.
func (pt *ProcessTable) FindProcesses(name string) []uint32 {
	pt.RLock()
	defer pt.RUnlock()

	var pids []uint32
	for pid, ps := range pt.Table {
		if ps.Path == name || filepath.Base(ps.Path) == name {
			pids = append(pids, pid)
		}
	}
	return pids
}

// Get the path of a processes source exe file.
// Looked up in the process table if it exists.
// If it does not, the process is looked up via win32.
func LookupProcessPath(pid uint32, pt *ProcessTable) string {
	if pt != nil {
		if ps := pt.LookupProcess(pid); ps != nil {
			return ps.Path
		}
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

func LookupParent(pid uint32, pt *ProcessTable) (uint32, string) {
	if pt != nil {
		if ps := pt.LookupProcess(pid); ps != nil {
			return ps.ParentPid, ps.ParentPath
		}
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
	pt.Lock()
	defer pt.Unlock()

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
func (pt *ProcessTable) GetTotalProcessCount() int {
	pt.RLock()
	defer pt.RUnlock()
	return len(pt.Table)
}

// Get the total handle count of a process.
// This will read lock the process table.
func (pt *ProcessTable) GetHandleCountPs(pid uint32) int {
	ps := pt.LookupProcess(pid)
	if ps == nil {
		return 0
	}
	return ps.GetHandleCount()
}
