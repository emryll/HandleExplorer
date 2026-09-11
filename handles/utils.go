package handles

import (
	"HandleExplorer/nt"
	"HandleExplorer/utils"
)

func (h HandleEntry) GetParameter(name string) utils.Parameter {
	if param, exists := h.Parameters[name]; exists {
		return param
	}
	return utils.Parameter{}
}

//*=================[ Access mask util wrappers ]=================

// Get the access mask as a single printable value. No arrays
func (h *HandleEntry) GetAccessAsString() any {
	domain := nt.GetDomainFromObject(h.Type)
	return utils.InterpretBitmaskValue((utils.Bitmask)(h.Access), domain)
}

// Get the access mask as a list of flags in human readable form.
func (h *HandleEntry) GetAccessFlagsAsString() []string {
	domain := nt.GetDomainFromObject(h.Type)
	// last parameter as true guarantees []string return value
	return utils.InterpretBitmaskValue((utils.Bitmask)(h.Access), domain, true).([]string)
}
