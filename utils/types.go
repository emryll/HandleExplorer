package utils

type Bitmask uint32

type Parameter struct {
	Name   string
	Type   uint8
	Domain uint8
	Buffer []byte
}

const (
	PARAMETER_ANSISTRING    = 1
	PARAMETER_ASTR_ARRAY    = 10
	PARAMETER_UINT32        = 2
	PARAMETER_UINT32_ARRAY  = 20
	PARAMETER_UINT64        = 3
	PARAMETER_UINT64_ARRAY  = 30
	PARAMETER_BOOLEAN       = 4
	PARAMETER_BOOLEAN_ARRAY = 40
	PARAMETER_POINTER       = 5
	PARAMETER_POINTER_ARRAY = 50
	PARAMETER_BYTES         = 7
)

type Enum struct {
	Value Bitmask
	// A domain is added, because the same
	// access flag value can mean different
	// things with a different object type.
	Domain uint8
}

type BitFlag struct {
	Name  string
	Value Bitmask
}
