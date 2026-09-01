package main

import (
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
	if err := ScanProcesses(); err != nil {
		PrintError("Failed to scan processes: %v\n", err)
	}

	refresh := time.NewTicker(time.Duration(PS_REFRESH_INTERVAL) * time.Second)
	defer refresh.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-refresh.C:
			if err := ScanProcesses(); err != nil {
				PrintError("Failed to scan processes: %v\n", err)
			}
		}
	}
}

// Scan processes via th32 snapshot and add them to the process table. One time.
// This will also check for any dead processes (without callbacks).
func ScanProcesses() error {
	//* get a process snapshot
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	processes := make(map[uint32]*windows.ProcessEntry32)

	//* walk the processes
	for {
		err = windows.Process32Next(handle, &entry)
		if err != nil {
			if err.Error() == "There are no more files." {
				break
			}
			return err
		}

		processes[entry.ProcessID] = &entry
		RegisterProcess(&entry)
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
	if ps := g_ProcessTable.LookupProcess(entry.ProcessID); ps != nil {
		if filepath.Base(ps.Path) == name {
			return // technically could still be different...
		}
		// new process with same pid, clear old one
		HandleTable.Remove(entry.ProcessID)
		g_ObjectAccessRegistry.RemoveEntriesByProcess(entry.ProcessID)
	}
	process := CreateProcessEntry(entry)
	g_ProcessTable.AddProcess(process)
}

// Create an initial process entry with basic details.
// This does not add the process entry to the process table.
func CreateProcessEntry(pe32 *windows.ProcessEntry32) *Process {
	entry := Process{
		ProcessId: pe32.ProcessID,
		ParentPid: pe32.ParentProcessID,
	}

	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err == nil {
		defer windows.CloseHandle(handle)
		path, err := GetProcessExecutable(handle)
		if err == nil {
			entry.Path = path
			// check signature only if full path is known,
			// otherwise this could be fooled by having
			// a different file of same name in working dir
			status, err := IsSigned(path)
			if err == nil {
				entry.SigStatus = status
			}
		}
		elevated, err := IsProcessElevated(handle)
		if err == nil {
			entry.Elevated = elevated
		}
	} else {
		PrintError("Failed to open process %d: %v\n", entry.ProcessId, err)
	}

	if entry.Path == "" {
		entry.Path = windows.UTF16ToString(pe32.ExeFile[:])
	}

	if entry.ParentPid == 0 || entry.ParentPid == 4 {
		return &entry
	}

	parentHandle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, ppid)
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
	if processes == nil || len(processes) == 0 {
		return
	}
	if g_ProcessTable == nil || g_ProcessTable.Table == nil {
		PrintWithRedLabel("[WARNING]", "Global process table not initialized!!")
	}

	g_ProcessTable.mu.RLock()
	defer g_ProcessTable.mu.RUnlock()

	//* make sure all processes are found in the process snapshot
	for pid := range g_ProcessTable.Table {
		if _, exists := processes[pid]; exists {
			continue
		}
		g_ProcessTable.RemoveProcess(pid)
		g_ObjectAccessRegistry.RemoveEntriesByProcess(pid)
		HandleTable.Remove(pid)
	}
}

//*======================[ Process Table ]==============================

var g_ProcessTable = &ProcessTable{}

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
	if _, exists := ps.Table[pid]; exists {
		delete(ps.Table, pid)
	}
}

func LookupParent(pid uint32) (uint32, string) {
	ps := g_ProcessTable.LookupProcess(pid)
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

//*=============================[ Utilities ]================================

// Get the total count of active processes.
// This will read lock the process table.
func GetTotalProcessCount() int {
	g_ProcessTable.mu.RLock()
	defer g_ProcessTable.mu.RUnlock()
	return len(g_ProcessTable.Table)
}

// Get the total handle count of a process.
// This will read lock the process table.
func GetHandleCountPs(pid uint32) int {
	ps := g_ProcessTable.LookupProcess(pid)
	if ps == nil {
		return 0
	}
	return ps.GetHandleCount()
}

// Get the path of a processes source exe file.
func LookupProcessPath(pid uint32) string {
	if ps := g_ProcessTable.LookupProcess(pid); ps != nil {
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

// Get the path of a processes source exe file.
// Provided handle only needs PROCESS_QUERY_LIMITED_INFORMATION
func GetProcessExecutable(handle windows.Handle) (string, error) {
	var buf [windows.MAX_PATH]uint16
	size := uint32(len(buf))
	// flag 0 for win32 path format
	err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size)
	if err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf[:size]), nil
}

// Check the status of a file's digital signature, returned as an enum (CERT_).
// If string conversion fails, or an unexpected status is received,
// then the corresponding error is returned. Otherwise the error is nil.
func IsSigned(path string) (int, error) {
	utf16Path, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	fileInfo := &windows.WinTrustFileInfo{
		Size:     uint32(unsafe.Sizeof(windows.WinTrustFileInfo{})),
		FilePath: utf16Path,
	}

	winTrustData := windows.WinTrustData{
		Size:                            uint32(unsafe.Sizeof(windows.WinTrustData{})),
		UIChoice:                        windows.WTD_UI_NONE,
		UnionChoice:                     windows.WTD_CHOICE_FILE,
		StateAction:                     windows.WTD_STATEACTION_IGNORE,
		ProvFlags:                       windows.WTD_REVOCATION_CHECK_NONE,
		FileOrCatalogOrBlobOrSgnrOrCert: unsafe.Pointer(fileInfo),
	}

	guid := windows.WINTRUST_ACTION_GENERIC_VERIFY_V2
	ret := windows.WinVerifyTrustEx(0, &guid, &winTrustData)

	if ret == nil {
		return CERT_VALID, nil
	}

	if errno, ok := ret.(windows.Errno); ok {
		switch windows.Handle(errno) {
		case windows.TRUST_E_NOSIGNATURE:
			return CERT_MISSING, nil
		case windows.TRUST_E_BAD_DIGEST:
			return CERT_HASH_MISMATCH, nil
		case windows.TRUST_E_EXPLICIT_DISTRUST:
			return CERT_EXP_DISTRUST, nil
		case windows.CERT_E_UNTRUSTEDROOT:
			return CERT_UNTRUSTED_ROOT, nil
		case windows.CERT_E_UNTRUSTEDCA:
			return CERT_UNTRUSTED_CA, nil
		case windows.CERT_E_REVOKED:
			return CERT_REVOKED, nil
		case windows.CERT_E_EXPIRED:
			return CERT_EXPIRED, nil
		}
	}
	return 0, ret
}

// Check if a process has elevated access rights.
// Provided handle only needs PROCESS_QUERY_LIMITED_INFORMATION
func IsProcessElevated(hProcess windows.Handle) (bool, error) {
	var (
		hToken    windows.Token
		elevation TOKEN_ELEVATION
		size      uint32
	)
	err := windows.OpenProcessToken(hProcess, windows.TOKEN_QUERY, &hToken)
	if err != nil {
		return false, err
	}

	err = windows.GetTokenInformation(hToken, windows.TokenElevation, (*byte)(unsafe.Pointer(&elevation)), uint32(unsafe.Sizeof(elevation)), &size)
	if err != nil {
		return false, err
	}
	return elevation.TokenIsElevated != 0, nil
}

func GetParentPid(handle windows.Handle) (uint32, error) {
	var (
		pbi    windows.PROCESS_BASIC_INFORMATION
		retLen uint32
	)
	err := windows.NtQueryInformationProcess(
		handle,
		windows.ProcessBasicInformation,
		unsafe.Pointer(&pbi),
		uint32(unsafe.Sizeof(pbi)),
		&retLen,
	)
	if err != nil {
		return 0, err
	}
	return uint32(pbi.InheritedFromUniqueProcessId), nil
}

func findProcesses(name string) []uint32 {
	g_ObjectAccessRegistry.mu.RLock()
	defer g_ObjectAccessRegistry.mu.RLock()

	var pids []uint32
	for pid, _ := range g_ObjectAccessRegistry.ProcessLookup {
		path := LookupProcessPath(pid)
		if name == path || name == filepath.Base(path) {
			pids = append(pids, pid)
		}
	}
	return pids
}
