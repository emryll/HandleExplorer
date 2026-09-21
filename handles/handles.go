package handles

//#include "handles.h"
import "C"

import (
	"HandleExplorer/handles/registry"
	"HandleExplorer/nt"
	"HandleExplorer/profiler"
	"HandleExplorer/utils"
	"fmt"
	"path/filepath"
	"unsafe"
)

// Get global handle table. Note that this is heavy; in the ballpark of 1000ms+
func GetGlobalHandleTable() []HandleEntry {
	hb := profiler.GetBenchmarker("HandleTable")
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
	return handleTable
}

func (h HandleEntry) ConvertToAccessEntry() registry.AccessEntry {
	var entry registry.AccessEntry
	entry.Object = h.Type
	entry.Handle = h.Handle
	entry.Pid = h.Pid
	entry.Params = h.Parameters
	entry.Access = (utils.Bitmask)(h.Access)
	entry.Address = h.Address

	switch entry.Object {
	case nt.OBJ_TYPE_PROCESS:
		pathParam := h.GetParameter("ImagePath")
		if !pathParam.Empty() {
			entry.Name = h.Parameters["ImagePath"].GetValue().(string)
			entry.Params["ImagePath"] = pathParam
		}
	case nt.OBJ_TYPE_THREAD:
		var name string
		tidParam := h.GetParameter("Tid")
		if !tidParam.Empty() {
			name = fmt.Sprintf("TID %v", h.Parameters["Tid"].GetValue())
		}

		if pathParam := h.GetParameter("Path"); !pathParam.Empty() {
			processPath := fmt.Sprintf("%v", h.Parameters["Path"].GetValue())
			if !utils.IsEmptyName(processPath) {
				name += fmt.Sprintf(" (%s)", filepath.Base(processPath))
			}
		}

		entry.Name = name
	default:
		nameParam := h.GetParameter("Name")
		if !nameParam.Empty() {
			entry.Name = utils.GetAnsiValue(nameParam.Buffer)
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
	entry.Address = h.Address
	entry.Type = h.Type
	entry.Pid = h.Pid

	if h.ParamsSize == 0 || h.Params == nil ||
		uintptr(h.Params) == ^uintptr(0) || h.ParamsSize > MAX_PARAMS_SIZE {

		return entry
	}

	buf := C.GoBytes(unsafe.Pointer(h.Params), C.int(h.ParamsSize))
	params := utils.ParseParameters(buf)
	entry.Parameters = make(map[string]utils.Parameter)
	for _, param := range params {
		entry.Parameters[param.Name] = param
	}
	C.free(unsafe.Pointer(h.Params))

	return entry
}
