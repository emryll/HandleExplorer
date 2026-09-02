package main

import (
	"math/bits"
	"sort"

	"golang.org/x/sys/windows"
)

const (
	CERT_VALID          = 1
	CERT_MISSING        = 1
	CERT_HASH_MISMATCH  = 3
	CERT_EXP_DISTRUST   = 4
	CERT_UNTRUSTED_CA   = 5
	CERT_UNTRUSTED_ROOT = 6
	CERT_REVOKED        = 7
	CERT_EXPIRED        = 8
)

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

var (
	PS_REFRESH_INTERVAL     = 10
	HANDLE_REFRESH_INTERVAL = 30
	HANDLE_CACHE_EXPIRATION = 10
	// This indicates a desired maximum,
	// not an actual hard limit on the capacity.
	// After this count, cache gets cleanup.
	HANDLE_CACHE_MAX_COUNT = 1000000 // around 50MB
)

var ( //* Handle Cache Cleanup modifiers
	HCC_MULTIPLIER_CONST  = 1
	HCC_OBJECT_MULTIPLIER = 2
	HCC_TIME_MULTIPLIER   = 2
	HCC_TIME_POWER        = 1.5

	HCC_DEFAULT_QUOTA        = 50
	HCC_TOP_PRIORITY_BONUS   = 5
	HCC_OBJECT_TIER_1_SCORE  = 2
	HCC_OBJECT_TIER_2_SCORE  = 15
	HCC_OBJECT_TIER_3_SCORE  = 40
	HCC_OBJECT_TIER_4_SCORE  = 60
	HCC_OBJECT_UNKNOWN_SCORE = 70
)

// lower score indicates its more important
var ObjectTypeTier = map[uint32]int{
	OBJ_TYPE_TP_WORKER_FACTORY:       1,
	OBJ_TYPE_PROCESS:                 1,
	OBJ_TYPE_THREAD:                  1,
	OBJ_TYPE_TOKEN:                   1,
	OBJ_TYPE_SECTION:                 1,
	OBJ_TYPE_ALPC_PORT:               1,
	OBJ_TYPE_DRIVER:                  1,
	OBJ_TYPE_DESKTOP:                 2,
	OBJ_TYPE_DEBUG_OBJECT:            2,
	OBJ_TYPE_SESSION:                 2,
	OBJ_TYPE_JOB:                     3,
	OBJ_TYPE_ETW_CONSUMER:            3,
	OBJ_TYPE_ETW_REGISTRATION:        3,
	OBJ_TYPE_ETW_SESSION_DEMUX_ENTRY: 3,
	OBJ_TYPE_DIRECTORY:               3,
	OBJ_TYPE_FILE:                    3,
	OBJ_TYPE_EVENT:                   3,
	OBJ_TYPE_SEMAPHORE:               3,
	OBJ_TYPE_CALLBACK:                3,
	OBJ_TYPE_WMI_GUID:                3,
	OBJ_TYPE_TIMER:                   4,
	OBJ_TYPE_IRTIMER:                 4,
	OBJ_TYPE_SYMLINK:                 4,
}

const ( //* bitmask enum domains
	DOMAIN_GLOBAL uint8 = iota
	DOMAIN_PROCESS
	DOMAIN_THREAD
	DOMAIN_EVENT
	DOMAIN_MUTEX
	DOMAIN_TIMER
	DOMAIN_SEMAPHORE
	DOMAIN_SECTION
	DOMAIN_FILE
	DOMAIN_PIPE
	DOMAIN_JOB
	DOMAIN_KEY
	DOMAIN_TOKEN
	DOMAIN_DESKTOP
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

// in this one there are no duplicates
var valToEnum map[uint8][]BitFlag // domain key

func fillReverseEnumLookup() {
	if len(valToEnum) > 0 {
		return
	}
	valToEnum = make(map[uint8][]BitFlag)
	entries := make(map[uint8]map[Bitmask]string)
	//* initially add only to entries for deduplication
	for enum, entry := range enumToVal {
		if entries[entry.Domain] == nil {
			entries[entry.Domain] = make(map[Bitmask]string)
		}
		if existing, exists := entries[entry.Domain][entry.Value]; !exists || len(enum) > len(existing) {
			entries[entry.Domain][entry.Value] = enum
		}
	}
	//* convert map to slice
	for domain, enums := range entries {
		for val, enum := range enums {
			valToEnum[domain] = append(valToEnum[domain], BitFlag{Name: enum, Value: val})
		}
		//* sort slice from most bits set to least
		sort.Slice(valToEnum[domain], func(i, j int) bool {
			return bits.OnesCount32(uint32(valToEnum[domain][i].Value)) > bits.OnesCount32(uint32(valToEnum[domain][j].Value))
		})
	}
}

const (
	THREAD_ALL_ACCESS = 0x1FFFFF

	JOB_OBJECT_ALL_ACCESS              = 0x1F003F
	JOB_OBJECT_ASSIGN_PROCESS          = 0x1
	JOB_OBJECT_SET_ATTRIBUTES          = 0x2
	JOB_OBJECT_SET_SECURITY_ATTRIBUTES = 0x10
	JOB_OBJECT_TERMINATE               = 0x8
	JOB_OBJECT_QUERY                   = 0x4

	DESKTOP_CREATEMENU      = 0x4
	DESKTOP_CREATEWINDOW    = 0x2
	DESKTOP_ENUMERATE       = 0x40
	DESKTOP_HOOKCONTROL     = 0x8
	DESKTOP_JOURNALPLAYBACK = 0x20
	DESKTOP_JOURNALRECORD   = 0x10
	DESKTOP_READOBJECTS     = 0x1
	DESKTOP_WRITEOBJECTS    = 0x80
	DESKTOP_SWITCHDESKTOP   = 0x100

	SECTION_ALL_ACCESS  = 0xF001F
	SECTION_EXTEND_SIZE = 0x10
	SECTION_MAP_EXECUTE = 0x8
	SECTION_MAP_WRITE   = 0x2
	SECTION_MAP_READ    = 0x4
	SECTION_QUERY       = 0x1
)

// dictionary to allow for using string enums for bitflags
var enumToVal = map[string]Enum{
	"DELETE":       Enum{Domain: DOMAIN_GLOBAL, Value: windows.DELETE},
	"READ_CONTROL": Enum{Domain: DOMAIN_GLOBAL, Value: windows.READ_CONTROL},
	"SYNCHRONIZE":  Enum{Domain: DOMAIN_GLOBAL, Value: windows.SYNCHRONIZE},
	"WRITE_DAC":    Enum{Domain: DOMAIN_GLOBAL, Value: windows.WRITE_DAC},
	"WRITE_OWNER":  Enum{Domain: DOMAIN_GLOBAL, Value: windows.WRITE_OWNER},

	"GENERIC_ALL":     Enum{Domain: DOMAIN_GLOBAL, Value: windows.GENERIC_ALL},
	"GENERIC_EXECUTE": Enum{Domain: DOMAIN_GLOBAL, Value: windows.GENERIC_EXECUTE},
	"GENERIC_WRITE":   Enum{Domain: DOMAIN_GLOBAL, Value: windows.GENERIC_WRITE},
	"GENERIC_READ":    Enum{Domain: DOMAIN_GLOBAL, Value: windows.GENERIC_READ},

	"STANDARD_RIGHTS_ALL":      Enum{Domain: DOMAIN_GLOBAL, Value: windows.STANDARD_RIGHTS_ALL},
	"STANDARD_RIGHTS_EXECUTE":  Enum{Domain: DOMAIN_GLOBAL, Value: windows.STANDARD_RIGHTS_EXECUTE},
	"STANDARD_RIGHTS_READ":     Enum{Domain: DOMAIN_GLOBAL, Value: windows.STANDARD_RIGHTS_READ},
	"STANDARD_RIGHTS_REQUIRED": Enum{Domain: DOMAIN_GLOBAL, Value: windows.STANDARD_RIGHTS_REQUIRED},
	"STANDARD_RIGHTS_WRITE":    Enum{Domain: DOMAIN_GLOBAL, Value: windows.STANDARD_RIGHTS_WRITE},

	"PROCESS_ALL_ACCESS":                Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_ALL_ACCESS},
	"PROCESS_CREATE_PROCESS":            Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_CREATE_PROCESS},
	"PROCESS_CREATE_THREAD":             Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_CREATE_THREAD},
	"PROCESS_DUP_HANDLE":                Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_DUP_HANDLE},
	"PROCESS_QUERY_INFORMATION":         Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_QUERY_INFORMATION},
	"PROCESS_QUERY_LIMITED_INFORMATION": Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_QUERY_LIMITED_INFORMATION},
	"PROCESS_SET_INFORMATION":           Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_SET_INFORMATION},
	"PROCESS_SET_QUOTA":                 Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_SET_QUOTA},
	"PROCESS_SUSPEND_RESUME":            Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_SUSPEND_RESUME},
	"PROCESS_TERMINATE":                 Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_TERMINATE},
	"PROCESS_VM_OPERATION":              Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_VM_OPERATION},
	"PROCESS_VM_READ":                   Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_VM_READ},
	"PROCESS_VM_WRITE":                  Enum{Domain: DOMAIN_PROCESS, Value: windows.PROCESS_VM_WRITE},

	"THREAD_ALL_ACCESS":                Enum{Domain: DOMAIN_THREAD, Value: THREAD_ALL_ACCESS},
	"THREAD_DIRECT_IMPERSONATION":      Enum{Domain: DOMAIN_THREAD, Value: windows.THREAD_DIRECT_IMPERSONATION},
	"THREAD_GET_CONTEXT":               Enum{Domain: DOMAIN_THREAD, Value: windows.THREAD_GET_CONTEXT},
	"THREAD_IMPERSONATE":               Enum{Domain: DOMAIN_THREAD, Value: windows.THREAD_IMPERSONATE},
	"THREAD_QUERY_INFORMATION":         Enum{Domain: DOMAIN_THREAD, Value: windows.THREAD_QUERY_INFORMATION},
	"THREAD_QUERY_LIMITED_INFORMATION": Enum{Domain: DOMAIN_THREAD, Value: windows.THREAD_QUERY_LIMITED_INFORMATION},
	"THREAD_SET_CONTEXT":               Enum{Domain: DOMAIN_THREAD, Value: windows.THREAD_SET_CONTEXT},
	"THREAD_SET_INFORMATION":           Enum{Domain: DOMAIN_THREAD, Value: windows.THREAD_SET_INFORMATION},
	"THREAD_SET_LIMITED_INFORMATION":   Enum{Domain: DOMAIN_THREAD, Value: windows.THREAD_SET_LIMITED_INFORMATION},
	"THREAD_SET_THREAD_TOKEN":          Enum{Domain: DOMAIN_THREAD, Value: windows.THREAD_SET_THREAD_TOKEN},
	"THREAD_SUSPEND_RESUME":            Enum{Domain: DOMAIN_THREAD, Value: windows.THREAD_SUSPEND_RESUME},
	"THREAD_TERMINATE":                 Enum{Domain: DOMAIN_THREAD, Value: windows.THREAD_TERMINATE},

	"EVENT_ALL_ACCESS":   Enum{Domain: DOMAIN_EVENT, Value: windows.EVENT_ALL_ACCESS},
	"EVENT_MODIFY_STATE": Enum{Domain: DOMAIN_EVENT, Value: windows.EVENT_MODIFY_STATE},

	"SEMAPHORE_ALL_ACCESS":   Enum{Domain: DOMAIN_SEMAPHORE, Value: windows.SEMAPHORE_ALL_ACCESS},
	"SEMAPHORE_MODIFY_STATE": Enum{Domain: DOMAIN_SEMAPHORE, Value: windows.SEMAPHORE_MODIFY_STATE},

	"MUTEX_ALL_ACCESS":   Enum{Domain: DOMAIN_MUTEX, Value: windows.MUTEX_ALL_ACCESS},
	"MUTEX_MODIFY_STATE": Enum{Domain: DOMAIN_MUTEX, Value: windows.MUTEX_ALL_ACCESS},

	"TIMER_ALL_ACCESS":   Enum{Domain: DOMAIN_TIMER, Value: windows.TIMER_ALL_ACCESS},
	"TIMER_MODIFY_STATE": Enum{Domain: DOMAIN_TIMER, Value: windows.TIMER_MODIFY_STATE},
	"TIMER_QUERY_STATE":  Enum{Domain: DOMAIN_TIMER, Value: windows.TIMER_QUERY_STATE},

	"SECTION_ALL_ACCESS":  Enum{Domain: DOMAIN_SECTION, Value: SECTION_ALL_ACCESS},
	"SECTION_EXTEND_SIZE": Enum{Domain: DOMAIN_SECTION, Value: SECTION_EXTEND_SIZE},
	"SECTION_MAP_EXECUTE": Enum{Domain: DOMAIN_SECTION, Value: SECTION_MAP_EXECUTE},
	"SECTION_MAP_WRITE":   Enum{Domain: DOMAIN_SECTION, Value: SECTION_MAP_WRITE},
	"SECTION_MAP_READ":    Enum{Domain: DOMAIN_SECTION, Value: SECTION_MAP_READ},
	"SECTION_QUERY":       Enum{Domain: DOMAIN_SECTION, Value: SECTION_QUERY},

	//TODO: Session

	"FILE_GENERIC_EXECUTE": Enum{Domain: DOMAIN_FILE, Value: windows.FILE_GENERIC_EXECUTE},
	"FILE_GENERIC_READ":    Enum{Domain: DOMAIN_FILE, Value: windows.FILE_GENERIC_READ},
	"FILE_GENERIC_WRITE":   Enum{Domain: DOMAIN_FILE, Value: windows.FILE_GENERIC_WRITE},

	"FILE_EXECUTE":          Enum{Domain: DOMAIN_FILE, Value: windows.FILE_EXECUTE},
	"FILE_READ_EA":          Enum{Domain: DOMAIN_FILE, Value: windows.FILE_READ_EA},
	"FILE_READ_DATA":        Enum{Domain: DOMAIN_FILE, Value: windows.FILE_READ_DATA},
	"FILE_READ_ATTRIBUTES":  Enum{Domain: DOMAIN_FILE, Value: windows.FILE_READ_ATTRIBUTES},
	"FILE_WRITE_ATTRIBUTES": Enum{Domain: DOMAIN_FILE, Value: windows.FILE_WRITE_ATTRIBUTES},
	"FILE_WRITE_DATA":       Enum{Domain: DOMAIN_FILE, Value: windows.FILE_WRITE_DATA},
	"FILE_WRITE_EA":         Enum{Domain: DOMAIN_FILE, Value: windows.FILE_WRITE_EA},

	"KEY_ALL_ACCESS":         Enum{Domain: DOMAIN_KEY, Value: windows.KEY_ALL_ACCESS},
	"KEY_CREATE_LINK":        Enum{Domain: DOMAIN_KEY, Value: windows.KEY_CREATE_LINK},
	"KEY_CREATE_SUB_KEY":     Enum{Domain: DOMAIN_KEY, Value: windows.KEY_CREATE_SUB_KEY},
	"KEY_ENUMERATE_SUB_KEYS": Enum{Domain: DOMAIN_KEY, Value: windows.KEY_ENUMERATE_SUB_KEYS},
	"KEY_NOTIFY":             Enum{Domain: DOMAIN_KEY, Value: windows.KEY_NOTIFY},
	"KEY_QUERY_VALUE":        Enum{Domain: DOMAIN_KEY, Value: windows.KEY_QUERY_VALUE},
	"KEY_READ":               Enum{Domain: DOMAIN_KEY, Value: windows.KEY_READ},
	"KEY_EXECUTE":            Enum{Domain: DOMAIN_KEY, Value: windows.KEY_EXECUTE},
	"KEY_SET_VALUE":          Enum{Domain: DOMAIN_KEY, Value: windows.KEY_SET_VALUE},
	"KEY_WOW64_32KEY":        Enum{Domain: DOMAIN_KEY, Value: windows.KEY_WOW64_32KEY},
	"KEY_WOW64_64KEY":        Enum{Domain: DOMAIN_KEY, Value: windows.KEY_WOW64_64KEY},
	"KEY_WRITE":              Enum{Domain: DOMAIN_KEY, Value: windows.KEY_WRITE},

	"PIPE_ACCESS_DUPLEX":   Enum{Domain: DOMAIN_PIPE, Value: windows.PIPE_ACCESS_DUPLEX},
	"PIPE_ACCESS_INBOUND":  Enum{Domain: DOMAIN_PIPE, Value: windows.PIPE_ACCESS_INBOUND},
	"PIPE_ACCESS_OUTBOUND": Enum{Domain: DOMAIN_PIPE, Value: windows.PIPE_ACCESS_OUTBOUND},

	//TODO: Symlink

	"TOKEN_ALL_ACCESS":        Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_ALL_ACCESS},
	"TOKEN_ADJUST_DEFAULT":    Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_ADJUST_DEFAULT},
	"TOKEN_ADJUST_GROUPS":     Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_ADJUST_GROUPS},
	"TOKEN_ADJUST_PRIVILEGES": Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_ADJUST_PRIVILEGES},
	"TOKEN_ADJUST_SESSIONID":  Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_ADJUST_SESSIONID},
	"TOKEN_ASSIGN_PRIMARY":    Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_ASSIGN_PRIMARY},
	"TOKEN_DUPLICATE":         Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_DUPLICATE},
	"TOKEN_EXECUTE":           Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_EXECUTE},
	"TOKEN_IMPERSONATE":       Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_IMPERSONATE},
	"TOKEN_QUERY":             Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_QUERY},
	"TOKEN_QUERY_SOURCE":      Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_QUERY_SOURCE},
	"TOKEN_READ":              Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_READ},
	"TOKEN_WRITE":             Enum{Domain: DOMAIN_TOKEN, Value: windows.TOKEN_WRITE},

	"JOB_OBJECT_ALL_ACCESS":              Enum{Domain: DOMAIN_JOB, Value: JOB_OBJECT_ALL_ACCESS},
	"JOB_OBJECT_ASSIGN_PROCESS":          Enum{Domain: DOMAIN_JOB, Value: JOB_OBJECT_ASSIGN_PROCESS},
	"JOB_OBJECT_QUERY":                   Enum{Domain: DOMAIN_JOB, Value: JOB_OBJECT_QUERY},
	"JOB_OBJECT_SET_ATTRIBUTES":          Enum{Domain: DOMAIN_JOB, Value: JOB_OBJECT_SET_ATTRIBUTES},
	"JOB_OBJECT_SET_SECURITY_ATTRIBUTES": Enum{Domain: DOMAIN_JOB, Value: JOB_OBJECT_SET_SECURITY_ATTRIBUTES},
	"JOB_OBJECT_TERMINATE":               Enum{Domain: DOMAIN_JOB, Value: JOB_OBJECT_TERMINATE},

	"DESKTOP_CREATEMENU":      Enum{Domain: DOMAIN_DESKTOP, Value: DESKTOP_CREATEMENU},
	"DESKTOP_CREATEWINDOW":    Enum{Domain: DOMAIN_DESKTOP, Value: DESKTOP_CREATEWINDOW},
	"DESKTOP_ENUMERATE":       Enum{Domain: DOMAIN_DESKTOP, Value: DESKTOP_ENUMERATE},
	"DESKTOP_HOOKCONTROL":     Enum{Domain: DOMAIN_DESKTOP, Value: DESKTOP_HOOKCONTROL},
	"DESKTOP_JOURNALPLAYBACK": Enum{Domain: DOMAIN_DESKTOP, Value: DESKTOP_JOURNALPLAYBACK},
	"DESKTOP_JOURNALRECORD":   Enum{Domain: DOMAIN_DESKTOP, Value: DESKTOP_JOURNALRECORD},
	"DESKTOP_READOBJECTS":     Enum{Domain: DOMAIN_DESKTOP, Value: DESKTOP_READOBJECTS},
	"DESKTOP_SWITCHDESKTOP":   Enum{Domain: DOMAIN_DESKTOP, Value: DESKTOP_SWITCHDESKTOP},
	"DESKTOP_WRITEOBJECTS":    Enum{Domain: DOMAIN_DESKTOP, Value: DESKTOP_WRITEOBJECTS},

	//TODO: Partition
	//TODO: DebugObject
	//TODO: Adapter
	//TODO: Controller
	//TODO: Device
}
