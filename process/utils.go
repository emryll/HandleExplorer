package process

import (
	tlist "HandleExplorer/tui/list"
	"fmt"
	"path/filepath"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

//*======================[ Lookup Utils ]=======================

// TODO
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

// *===================[ Process Utils ]======================

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
// NOTE: DONT CALL THIS BECAUSE IT CAN HANG FOR EVER!! Call IsSignedWithTimeout()
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

// Check the status of a file's digital signature, returned as an enum (CERT_).
// If string conversion fails, or an unexpected status is received,
// then the corresponding error is returned. Otherwise the error is nil.
// This has a timeout to prevent WinVerifyTrustEx from freezing the program.
// You can provide your own timeout, or use default timeout of 5 seconds.
func IsSignedWithTimeout(path string, timeout ...time.Duration) (int, error) {
	type result struct {
		status int
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		status, err := IsSigned(path)
		ch <- result{status, err}
	}()

	var timeLimit time.Duration
	if len(timeout) > 0 {
		timeLimit = timeout[0]
	} else {
		timeLimit = time.Second * 3
	}

	select {
	case r := <-ch:
		return r.status, r.err
	case <-time.After(timeLimit):
		return 0, fmt.Errorf("signature check timed out for %s", path)
	}
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

//*===================[ List Picker (UI) ]===================

func (p *Process) Columns() []tlist.Column {
	return []tlist.Column{
		{Title: "Handles"},
		{Title: "PID", Highlight: true},
		{Title: "Path", Highlight: true},
		{Title: "Parent", Right: true},
	}
}

func (p *Process) Fields() []string {
	parent := fmt.Sprintf("PID %d", p.ParentPid)
	if p.ParentPath != "" {
		parent += fmt.Sprintf(" (%s)", filepath.Base(p.ParentPath))
	}

	return []string{
		fmt.Sprint(p.GetHandleCount()),
		fmt.Sprint(p.ProcessId),
		p.Path,
		parent,
	}
}

func (p *Process) Key() string { return fmt.Sprintf("%d", p.ProcessId) }

// RightStages implements StagedField for the Parent column: full
// "PID x (name)" if there's room, then "PID x (...)" to indicate a
// name exists without showing it, then just "PID x", then hidden.
func (p *Process) RightStages() []string {
	if p.ParentPid == 0 {
		return []string{""}
	}

	return []string{
		fmt.Sprintf("PID %d (%s)", p.ParentPid, p.ParentPath),
		fmt.Sprintf("PID %d (...)", p.ParentPid),
		fmt.Sprintf("PID %d", p.ParentPid),
	}
}

// Used for polymorphic list (newPickerModel)
func (p *Process) Title() string {
	return "Process"
}

// Used for polymorphic list (newPickerModel)
func (p *Process) Subtitle() string {
	return "Process search results"
}

// Used for polymorphic list (newPickerModel)
func (p *Process) Noun() string {
	return "processes"
}
