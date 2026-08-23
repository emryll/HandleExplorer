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
