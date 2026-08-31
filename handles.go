package main

//#include "handles.h"
import "C"

import (
	"unsafe"
)

// Get global handle table. Note that this is heavy; in the ballpark of 1000ms+
func GetGlobalHandleTable() []HandleEntry {
	hb := GetBenchmarker("HandleTable")
	if hb != nil {
		stop := hb.Benchmark()
		defer stop()
	}

	var handleCount C.size_t
	cHandleEntries := C.GetGlobalHandleTable(&handleCount)
	cSlice := unsafe.Slice((*cHandleEntry)(unsafe.Pointer(cHandleEntries)), int(handleCount))

	handleTable := make([]HandleEntry, 0, int(handleCount))
	for _, v := range cSlice {
		handleTable = append(handleTable, v.GoEntry())
	}
	C.free(unsafe.Pointer(cHandleEntries))
    g_SessionStats.SetHandleCount(int(handleCount))
	return handleTable
}

func (h HandleEntry) ConvertToAccessEntry() AccessEntry {
	var entry AccessEntry
	entry.Object = h.Type
	entry.Handle = h.Handle
	entry.Pid = h.Pid
	entry.Params = h.Parameters

	if entry.Object == OBJ_TYPE_PROCESS {
		pathParam := h.GetParameter("ImagePath")
		if !pathParam.Empty() {
			entry.Name = h.Parameters["ImagePath"].GetValue().(string)
			entry.Params["ImagePath"] = pathParam
		}
	} else {
		nameParam := h.GetParameter("Name")
		if !nameParam.Empty() {
			entry.Name = GetAnsiValue(nameParam.Buffer)
		}
	}
	return entry
}

// Convert mirrored C HANDLE_ENTRY layout into go version
func (h cHandleEntry) GoEntry() HandleEntry {
	var entry HandleEntry
	entry.FirstSeen = h.FirstSeen
	entry.LastSeen = h.LastSeen
	entry.Handle = h.Handle
	entry.Access = h.Access
	entry.Type = h.Type
	entry.Pid = h.Pid

	if h.ParamsSize == 0 || h.Params == nil || uintptr(h.Params) == ^uintptr(0) {
		return entry
	}

	buf := C.GoBytes(unsafe.Pointer(h.Params), C.int(h.ParamsSize))
	params := ParseParameters(buf)
	entry.Parameters = make(map[string]Parameter)
	for _, param := range params {
		entry.Parameters[param.Name] = param
	}
	C.free(unsafe.Pointer(h.Params))

	return entry
}
