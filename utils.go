package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	red    = color.New(color.FgRed)
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgHiYellow, color.Bold)
)

func GetInput(reader *bufio.Reader, msg ...string) string {
	if len(msg) > 0 {
		fmt.Printf("%s: ", msg)
	}
	if reader == nil {
		reader = bufio.NewReader(os.Stdin)
	}

	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func PrintError(format string, v ...any) {
	PrintWithRedLabel("[*]", format, v)
}

func PrintWithRedLabel(label string, format string, v ...any) {
	red.Printf("%s ", label)
	fmt.Printf(format, v)
}

func PrintBanner() {
	grey := color.New(color.FgWhite)

	fmt.Println("\t   __ _____   _  _____  __   ____   ")
	fmt.Println("\t  / // / _ | / |/ / _ \\/ /  / __/   ")
	fmt.Println("\t / _  / __ |/    / // / /__/ _/     ")
	fmt.Println("\t/_//_/_/ |_/_/|_/____/____/___/     ")
	yellow.Printf("\t  / __/_ __ ___  / /__  _______ ____\n")
	yellow.Printf("\t / _/ \\ \\ // _ \\/ / _ \\/ __/ -_) __/\n")
	yellow.Printf("\t/___//_\\_\\/ .__/_/\\___/_/  \\__/_/   \n")
	yellow.Printf("\t         /_/                        \n")
	grey.Printf("\t\t\tv%d.%d by emryll\n\n", MAJOR_VERSION, MINOR_VERSION)

	PrintDescription()
	fmt.Println()
}

func PrintDescription() {
	fmt.Println("\tThis is a commandline-tool for searching")
	fmt.Println("\t& analyzing object access through handles.\n")
	fmt.Println("\tTo view available commands, run \"help\"")
}

// Get the access mask as a single printable value. No arrays
func (e *AccessEntry) GetAccessAsString() any {
	domain := GetDomainFromObject(e.Object)
	return InterpretBitmaskValue((Bitmask)(e.Access), domain)
}

// Get the access mask as a list of flags in human readable form.
func (e *AccessEntry) GetAccessFlagsAsString() []string {
	domain := GetDomainFromObject(e.Object)
	// last parameter as true guarantees []string return value
	return InterpretBitmaskValue((Bitmask)(e.Access), domain, true).([]string)
}

// Get the access mask as a single printable value. No arrays
func (h *HandleEntry) GetAccessAsString() any {
	domain := GetDomainFromObject(h.Type)
	return InterpretBitmaskValue((Bitmask)(h.Access), domain)
}

// Get the access mask as a list of flags in human readable form.
func (h *HandleEntry) GetAccessFlagsAsString() []string {
	domain := GetDomainFromObject(h.Type)
	// last parameter as true guarantees []string return value
	return InterpretBitmaskValue((Bitmask)(h.Access), domain, true).([]string)
}

// Returns string interpretation of all contained flags,
// or if it couldn't find corresponding enums, it returns raw value
func InterpretBitmaskValue(mask Bitmask, domain uint8, array ...bool) any {
	var flags []string
	for _, entry := range valToEnum[domain] {
		if mask&entry.Value == entry.Value {
			//* strip flag and save it
			mask &^= entry.Value
			flags = append(flags, entry.Name)
		}
	}
	//* check the generic domain
	for _, entry := range valToEnum[DOMAIN_GLOBAL] {
		if mask&entry.Value != 0 {
			//* strip flag and save it
			mask &^= entry.Value
			flags = append(flags, entry.Name)
		}
	}

	if len(flags) == 0 {
		return mask
	}

	if len(array) == 0 || !array[0] {
		var result string
		for i := range flags {
			result += flags[i]
			if i+1 < len(flags) {
				result += " | "
			}
		}
		return result
	}
	return flags
}

func parseAccessString(accessList string) Bitmask {
	var mask Bitmask
	flags := strings.Split(accessList, "|")
	for _, flag := range flags {
		if enum, exists := enumToVal[flag]; exists {
			mask |= enum.Value
		}
	}
	return mask
}

// interpret raw c ansi string as a go string
func GetAnsiValue(data []byte) string {
	n := 0
	for ; n < len(data); n++ {
		if data[n] == 0 {
			break // null terminator
		}
	}
	return string(data[:n])
}

// Get all pids that accessed a named object.
func GetObjectAccessPids(objType uint32, name string) []uint32 {
	g_ObjectAccessRegistry.mu.Lock()
	defer g_ObjectAccessRegistry.mu.Unlock()

	if len(g_ObjectAccessRegistry.ObjectLookup[objType]) == 0 {
		return nil
	}
	var (
		pids []uint32
		seen = make(map[uint32]bool)
	)
	for key := range g_ObjectAccessRegistry.ObjectLookup[objType] {
		if key.Name != name {
			continue
		}
		if seen[key.Pid] {
			continue
		}
		pids = append(pids, key.Pid)
		seen[key.Pid] = true
	}
	return pids
}

// Get all object types that a process has accessed
// This will read lock the object access registry.
func GetObjectTypesAccessed(pid uint32) map[uint32]bool {
	g_ObjectAccessRegistry.mu.RLock()
	defer g_ObjectAccessRegistry.mu.RUnlock()

	if len(g_ObjectAccessRegistry.ProcessLookup[pid]) == 0 {
		return nil
	}

	accessed := make(map[uint32]bool)
	for key := range g_ObjectAccessRegistry.ProcessLookup[pid] {
		accessed[key.ObjType] = true
	}
	return accessed
}

// Get the domain id for an object type.
// This is used to translate access mask
// values into human readable string enums.
func GetDomainFromObject(objType uint32) uint8 {
	var domain uint8
	switch objType {
	case OBJ_TYPE_PROCESS:
		domain = DOMAIN_PROCESS
	case OBJ_TYPE_THREAD:
		domain = DOMAIN_THREAD
	case OBJ_TYPE_EVENT:
		domain = DOMAIN_EVENT
	case OBJ_TYPE_MUTANT:
		domain = DOMAIN_MUTEX
	case OBJ_TYPE_TIMER, OBJ_TYPE_IRTIMER:
		domain = DOMAIN_TIMER
	case OBJ_TYPE_SEMAPHORE:
		domain = DOMAIN_SEMAPHORE
	case OBJ_TYPE_SECTION:
		domain = DOMAIN_SECTION
	case OBJ_TYPE_FILE:
		domain = DOMAIN_FILE
	case OBJ_TYPE_PIPE:
		domain = DOMAIN_PIPE
	case OBJ_TYPE_JOB:
		domain = DOMAIN_JOB
	case OBJ_TYPE_KEY:
		domain = DOMAIN_KEY
	case OBJ_TYPE_TOKEN:
		domain = DOMAIN_TOKEN
	case OBJ_TYPE_DESKTOP:
		domain = DOMAIN_DESKTOP
	default:
		domain = DOMAIN_GLOBAL
	}
	return domain
}

// Print the string, or a dash if empty
func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
