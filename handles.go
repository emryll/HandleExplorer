package main

import (
	"unsafe"
)

// Get global handle table. Note that this is heavy; in the ballpark of 1000ms+
func GetGlobalHandleTable() []HandleEntry {
	//TODO: add silent benchmark
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

func (h HandleEntry) ConvertToAccessEntry() AccessEntry {
	var entry AccessEntry
	entry.Object = h.Type
	entry.Handle = h.Handle
	entry.Pid = h.Pid
	entry.Params = h.Parameters

	if entry.Object == OBJECT_TYPE_PROCESS {
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
