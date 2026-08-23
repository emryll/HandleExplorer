package main

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

//TODO: set up lightweight process lookup for names etc

type ProcessTable struct {
	mu    sync.RWMutex
	Table map[uint32]*Process
}

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

func (ps *ProcessTable) RemoveProcess(pid uint32) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.Table == nil {
		return
	}
	if _, exists := ps.Table[pid]; exists {
		delete(ps.Table, pid)
	}
}

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
