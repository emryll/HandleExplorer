package main

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// Outer function that will parse every parameter in provided data buffer.
// Name must NOT be null-terminated. Size must be size in bytes. Header portion must be null-terminated.
func ParseParameters(data []byte) []Parameter {
	var params []Parameter
	cursor := 0
	for cursor < len(data) {
		parameter := GetAnsiValue(data[cursor:])
		cursor += len(parameter) + 1 // +1 because null-terminator occupies one byte after

		// -1 so you wont get out of bounds error, but prevent off-by-one error.
		param, err := ParseSingleParameter(parameter, data[cursor-1:])
		if err != nil || param.Buffer == nil {
			if err != nil {
				color.Red("\n[!] Failed to parse parameter: %v", err)
			}
			if len(parameter) == 0 {
				break // prevent infinite loop
			}
			continue
		}
		cursor += len(param.Buffer)
		params = append(params, param)
	}
	return params
}

// This function will parse a single parameter.
// Before calling this, retrieve the header by reading an ansi string,
// and then you can pass all the bytes following it, as the data argument.
// An empty or invalid parameter header is not considered an error, it returns empty.
func ParseSingleParameter(header string, data []byte) (Parameter, error) {
	var (
		param   Parameter
		isArray = false
	)

	// skip empty reads. minimum possible size of header is 3 (a:b)
	if header == "" || len(header) < 2 {
		return Parameter{}, nil
	}
	ptype := header[:1]
	parts := strings.Split(header[1:], "/")
	// non-array types should have only one string in head (no "/")
	if len(parts) > 1 {
		if len(parts[1]) == 0 {
			return Parameter{}, fmt.Errorf("invalid header: size (%s)", header)
		}
		param.Name = parts[0]
		size, err := strconv.Atoi(parts[1])
		if err != nil {
			return Parameter{}, fmt.Errorf("failed to read size into integer: %v (%s)", err, header)
		}
		// add array defined bytes into the parameter buffer
		param.Buffer = append([]byte(nil), data[:size]...)
		isArray = true
	} else {
		param.Name = header[1:]
	}

	param.Type = uint8(GetParameterType(ptype))

	// remove possible null-terminator from first byte
	if len(data) > 0 && data[0] == '\000' {
		data = data[1:]
	}
	if !isArray { // now get data buffer if it wasnt array
		switch int(param.Type) {
		case PARAMETER_ANSISTRING:
			str := GetAnsiValue(data)
			param.Buffer = append([]byte(nil), data[:len(str)+1]...)
		case PARAMETER_BOOLEAN, PARAMETER_UINT32:
			param.Buffer = append([]byte(nil), data[:4]...)
		case PARAMETER_UINT64, PARAMETER_POINTER:
			param.Buffer = append([]byte(nil), data[:8]...)
		}
	}
	return param, nil
}

// Print the provided parameters
func PrintParameters(params map[string]Parameter) {
	for _, param := range params {
		fmt.Printf("\t%s: %v", param.Name, param.GetValue())
	}
}

// Return the dynamic parameter as a regular Go value.
// This method is generally used to print a value with "%v"
// Array values are returned as a list of bullet points (i.e. string)
func (p Parameter) GetValue() any {
	switch p.Type {
	case PARAMETER_ANSISTRING:
		return GetAnsiValue(p.Buffer)
	case PARAMETER_ASTR_ARRAY:
		return GetStringArrayFromBuffer(p.Buffer)
	case PARAMETER_BOOLEAN:
		return binary.LittleEndian.Uint32(p.Buffer) == 1
	case PARAMETER_BOOLEAN_ARRAY:
		return GetBooleanArrayFromBuffer(p.Buffer)
	case PARAMETER_UINT32:
		val := binary.LittleEndian.Uint32(p.Buffer)
		if p.Domain == 0 {
			return val
		}
		//return InterpretBitmaskValue((Bitmask)(val), p.Domain)
	case PARAMETER_UINT32_ARRAY:
		return GetUint32ArrayFromBuffer(p.Buffer)
	case PARAMETER_UINT64:
		return binary.LittleEndian.Uint64(p.Buffer)
	case PARAMETER_UINT64_ARRAY:
		return GetUint64ArrayFromBuffer(p.Buffer)
	case PARAMETER_POINTER:
		return fmt.Sprintf("%p", binary.LittleEndian.Uint64(p.Buffer))
	case PARAMETER_POINTER_ARRAY:
		return GetPointerArrayFromBuffer(p.Buffer)
	case PARAMETER_BYTES:
		return p.Buffer
	}
	return "(invalid parameter)"
}
