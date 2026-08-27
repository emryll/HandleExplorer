package main

//?========================================================================+
//?   This file contains the NT object type to internal enum conversions.  |
//?    The object type enums used internally in Windows are not stable.    |
//?      This is why NtQueryObject returns a string, the NT name           |
//?========================================================================+

// Internal enums for object types.
// The type enums used by windows are not stable.
const (
	OBJ_TYPE_UNKNOWN                 = 0
	OBJ_TYPE_PROCESS                 = 1
	OBJ_TYPE_THREAD                  = 2
	OBJ_TYPE_EVENT                   = 3
	OBJ_TYPE_SEMAPHORE               = 4
	OBJ_TYPE_MUTANT                  = 5
	OBJ_TYPE_SECTION                 = 6
	OBJ_TYPE_SESSION                 = 7
	OBJ_TYPE_FILE                    = 8
	OBJ_TYPE_KEY                     = 9
	OBJ_TYPE_DIRECTORY               = 10
	OBJ_TYPE_SYMLINK                 = 11
	OBJ_TYPE_TOKEN                   = 12
	OBJ_TYPE_JOB                     = 13
	OBJ_TYPE_DESKTOP                 = 14
	OBJ_TYPE_PIPE                    = 15
	OBJ_TYPE_DEBUG_OBJECT            = 16
	OBJ_TYPE_CALLBACK                = 17
	OBJ_TYPE_ADAPTER                 = 18
	OBJ_TYPE_CONTROLLER              = 19
	OBJ_TYPE_DEVICE                  = 20
	OBJ_TYPE_DRIVER                  = 21
	OBJ_TYPE_IO_RING                 = 22
	OBJ_TYPE_TM_TM                   = 23
	OBJ_TYPE_TM_TX                   = 24
	OBJ_TYPE_TM_RM                   = 25
	OBJ_TYPE_TM_EN                   = 26
	OBJ_TYPE_TIMER                   = 27
	OBJ_TYPE_IRTIMER                 = 28
	OBJ_TYPE_PROFILE                 = 29
	OBJ_TYPE_KEYED_EVENT             = 30
	OBJ_TYPE_WINDOW_STATION          = 31
	OBJ_TYPE_COMPOSITION             = 32
	OBJ_TYPE_RAW_INPUT_MANAGER       = 33
	OBJ_TYPE_CORE_MESSAGING          = 34
	OBJ_TYPE_ACTIVATION_OBJECT       = 35
	OBJ_TYPE_TP_WORKER_FACTORY       = 36
	OBJ_TYPE_IO_COMPLETION           = 37
	OBJ_TYPE_WAIT_COMPLETION_PACKET  = 38
	OBJ_TYPE_USER_APC_RESERVE        = 39
	OBJ_TYPE_IO_COMP_RESERVE         = 40
	OBJ_TYPE_ACTIVITY_REFERENCE      = 41
	OBJ_TYPE_PS_STATE_CHANGE         = 42
	OBJ_TYPE_THREAD_STATE_CHANGE     = 43
	OBJ_TYPE_CPU_PARTITION           = 44
	OBJ_TYPE_PS_SILO_CTX_PAGED       = 45
	OBJ_TYPE_PS_SILO_CTX_NON_PAGED   = 46
	OBJ_TYPE_REGISTRY_TRANSACTION    = 47
	OBJ_TYPE_DMA_ADAPTER             = 48
	OBJ_TYPE_ALPC_PORT               = 49
	OBJ_TYPE_ENERGY_TRACKER          = 50
	OBJ_TYPE_POWER_REQUEST           = 51
	OBJ_TYPE_WMI_GUID                = 52
	OBJ_TYPE_ETW_REGISTRATION        = 53
	OBJ_TYPE_ETW_SESSION_DEMUX_ENTRY = 54
	OBJ_TYPE_ETW_CONSUMER            = 55
	OBJ_TYPE_PCW_OBJECT              = 56
	OBJ_TYPE_COVERAGE_SAMPLER        = 57
	OBJ_TYPE_FILTER_CONNECTION_PORT  = 58
	OBJ_TYPE_FILTER_COMM_PORT        = 59
	OBJ_TYPE_NDIS_CM_STATE           = 60
	OBJ_TYPE_DXGK_SHARED_RSRC        = 61
	OBJ_TYPE_DXGK_SHARED_MUTEX       = 62
	OBJ_TYPE_DXGK_SHARED_SYNC        = 63
	OBJ_TYPE_DXGK_SHARED_SWAP        = 64
	OBJ_TYPE_DXGK_DISPLAY_MGR        = 65
	OBJ_TYPE_DXGK_SHARED_BUNDLE      = 66
	OBJ_TYPE_DXGK_SHARED_SESSION     = 67
	OBJ_TYPE_DXGK_COMPOSITION        = 68
	OBJ_TYPE_V_REG_CONFIG_CONTEXT    = 69
	OBJ_TYPE_PARTITION               = 70
	OBJ_TYPE_DXGK_CURRENT_DXG_THREAD = 71
)

// Translate internal object type enum into name.
func GetTypeName(object uint32) string {
	switch object {
	case OBJ_TYPE_PROCESS:
		return "Process"
	case OBJ_TYPE_THREAD:
		return "Thread"
	case OBJ_TYPE_EVENT:
		return "Event"
	case OBJ_TYPE_SEMAPHORE:
		return "Semaphore"
	case OBJ_TYPE_MUTANT:
		return "Mutant"
	case OBJ_TYPE_SECTION:
		return "Section"
	case OBJ_TYPE_SESSION:
		return "Session"
	case OBJ_TYPE_FILE:
		return "File"
	case OBJ_TYPE_KEY:
		return "Key"
	case OBJ_TYPE_DIRECTORY:
		return "Directory"
	case OBJ_TYPE_SYMLINK:
		return "SymbolicLink"
	case OBJ_TYPE_TOKEN:
		return "Token"
	case OBJ_TYPE_JOB:
		return "Job"
	case OBJ_TYPE_DESKTOP:
		return "Desktop"
	case OBJ_TYPE_PARTITION:
		return "Partition"
	case OBJ_TYPE_DEBUG_OBJECT:
		return "DebugObject"
	case OBJ_TYPE_CALLBACK:
		return "Callback"
	case OBJ_TYPE_ADAPTER:
		return "Adapter"
	case OBJ_TYPE_CONTROLLER:
		return "Controller"
	case OBJ_TYPE_DEVICE:
		return "Device"
	case OBJ_TYPE_DRIVER:
		return "Driver"
	case OBJ_TYPE_IO_RING:
		return "IoRing"
	case OBJ_TYPE_TM_TM:
		return "TmTm"
	case OBJ_TYPE_TM_TX:
		return "TmTx"
	case OBJ_TYPE_TM_RM:
		return "TmRm"
	case OBJ_TYPE_TM_EN:
		return "TmEn"
	case OBJ_TYPE_TIMER:
		return "Timer"
	case OBJ_TYPE_IRTIMER:
		return "IRTimer"
	case OBJ_TYPE_PROFILE:
		return "Profile"
	case OBJ_TYPE_KEYED_EVENT:
		return "KeyedEvent"
	case OBJ_TYPE_WINDOW_STATION:
		return "WindowStation"
	case OBJ_TYPE_COMPOSITION:
		return "Composition"
	case OBJ_TYPE_RAW_INPUT_MANAGER:
		return "RawInputManager"
	case OBJ_TYPE_CORE_MESSAGING:
		return "CoreMessaging"
	case OBJ_TYPE_ACTIVATION_OBJECT:
		return "ActivationObject"
	case OBJ_TYPE_TP_WORKER_FACTORY:
		return "TpWorkerFactory"
	case OBJ_TYPE_IO_COMPLETION:
		return "IoCompletion"
	case OBJ_TYPE_WAIT_COMPLETION_PACKET:
		return "WaitCompletionPacket"
	case OBJ_TYPE_USER_APC_RESERVE:
		return "UserApcReserve"
	case OBJ_TYPE_IO_COMP_RESERVE:
		return "IoCompletionReserve"
	case OBJ_TYPE_ACTIVITY_REFERENCE:
		return "ActivityReference"
	case OBJ_TYPE_PS_STATE_CHANGE:
		return "ProcessStateChange"
	case OBJ_TYPE_THREAD_STATE_CHANGE:
		return "ThreadStateChange"
	case OBJ_TYPE_CPU_PARTITION:
		return "CpuPartition"
	case OBJ_TYPE_PS_SILO_CTX_PAGED:
		return "PsSiloContextPaged"
	case OBJ_TYPE_PS_SILO_CTX_NON_PAGED:
		return "PsSiloContextNonPaged"
	case OBJ_TYPE_REGISTRY_TRANSACTION:
		return "RegistryTransaction"
	case OBJ_TYPE_DMA_ADAPTER:
		return "DmaAdapter"
	case OBJ_TYPE_ALPC_PORT:
		return "ALPC Port"
	case OBJ_TYPE_ENERGY_TRACKER:
		return "EnergyTracker"
	case OBJ_TYPE_POWER_REQUEST:
		return "PowerRequest"
	case OBJ_TYPE_WMI_GUID:
		return "WmiGuid"
	case OBJ_TYPE_ETW_REGISTRATION:
		return "EtwRegistration"
	case OBJ_TYPE_ETW_SESSION_DEMUX_ENTRY:
		return "EtwSessionDemuxEntry"
	case OBJ_TYPE_ETW_CONSUMER:
		return "EtwConsumer"
	case OBJ_TYPE_PCW_OBJECT:
		return "PcwObject"
	case OBJ_TYPE_COVERAGE_SAMPLER:
		return "CoverageSampler"
	case OBJ_TYPE_FILTER_CONNECTION_PORT:
		return "FilterConnectionPort"
	case OBJ_TYPE_FILTER_COMM_PORT:
		return "FilterCommunicationPort"
	case OBJ_TYPE_NDIS_CM_STATE:
		return "NdisCmState"
	case OBJ_TYPE_DXGK_SHARED_RSRC:
		return "DxgkSharedResource"
	case OBJ_TYPE_DXGK_SHARED_MUTEX:
		return "DxgkSharedKeyedMutexObject"
	case OBJ_TYPE_DXGK_SHARED_SYNC:
		return "DxgkSharedSyncObject"
	case OBJ_TYPE_DXGK_SHARED_SWAP:
		return "DxgkSharedSwapChainObject"
	case OBJ_TYPE_DXGK_SHARED_SESSION:
		return "DxgkSharedProtectedSessionObject"
	case OBJ_TYPE_DXGK_SHARED_BUNDLE:
		return "DxgkSharedBundleObject"
	case OBJ_TYPE_DXGK_DISPLAY_MGR:
		return "DxgkDisplayManagerObject"
	case OBJ_TYPE_DXGK_COMPOSITION:
		return "DxgkCompositionObject"
	case OBJ_TYPE_DXGK_CURRENT_DXG_THREAD:
		return "DxgkCurrentDxgThreadObject"
	case OBJ_TYPE_V_REG_CONFIG_CONTEXT:
		return "VRegConfigurationContext"
	}
	return "(unknown)"
}

// Translate internal object type enum into name.
func GetTypeIdentifier(object string) uint32 {
	switch object {
	case "Process":
		return OBJ_TYPE_PROCESS
	case "Thread":
		return OBJ_TYPE_THREAD
	case "Event":
		return OBJ_TYPE_EVENT
	case "Semaphore":
		return OBJ_TYPE_SEMAPHORE
	case "Mutant":
		return OBJ_TYPE_MUTANT
	case "Section":
		return OBJ_TYPE_SECTION
	case "Session":
		return OBJ_TYPE_SESSION
	case "File":
		return OBJ_TYPE_FILE
	case "Key":
		return OBJ_TYPE_KEY
	case "Directory":
		return OBJ_TYPE_DIRECTORY
	case "SymbolicLink":
		return OBJ_TYPE_SYMLINK
	case "Token":
		return OBJ_TYPE_TOKEN
	case "Job":
		return OBJ_TYPE_JOB
	case "Desktop":
		return OBJ_TYPE_DESKTOP
	case "Partition":
		return OBJ_TYPE_PARTITION
	case "DebugObject":
		return OBJ_TYPE_DEBUG_OBJECT
	case "Callback":
		return OBJ_TYPE_CALLBACK
	case "Adapter":
		return OBJ_TYPE_ADAPTER
	case "Controller":
		return OBJ_TYPE_CONTROLLER
	case "Device":
		return OBJ_TYPE_DEVICE
	case "Driver":
		return OBJ_TYPE_DRIVER
	case "IoRing":
		return OBJ_TYPE_IO_RING
	case "TmTm":
		return OBJ_TYPE_TM_TM
	case "TmTx":
		return OBJ_TYPE_TM_TX
	case "TmRm":
		return OBJ_TYPE_TM_RM
	case "TmEn":
		return OBJ_TYPE_TM_EN
	case "Timer":
		return OBJ_TYPE_TIMER
	case "IRTimer":
		return OBJ_TYPE_IRTIMER
	case "Profile":
		return OBJ_TYPE_PROFILE
	case "KeyedEvent":
		return OBJ_TYPE_KEYED_EVENT
	case "WindowStation":
		return OBJ_TYPE_WINDOW_STATION
	case "Composition":
		return OBJ_TYPE_COMPOSITION
	case "RawInputManager":
		return OBJ_TYPE_RAW_INPUT_MANAGER
	case "CoreMessaging":
		return OBJ_TYPE_CORE_MESSAGING
	case "ActivationObject":
		return OBJ_TYPE_ACTIVATION_OBJECT
	case "TpWorkerFactory":
		return OBJ_TYPE_TP_WORKER_FACTORY
	case "IoCompletion":
		return OBJ_TYPE_IO_COMPLETION
	case "WaitCompletionPacket":
		return OBJ_TYPE_WAIT_COMPLETION_PACKET
	case "UserApcReserve":
		return OBJ_TYPE_USER_APC_RESERVE
	case "IoCompletionReserve":
		return OBJ_TYPE_IO_COMP_RESERVE
	case "ActivityReference":
		return OBJ_TYPE_ACTIVITY_REFERENCE
	case "ProcessStateChange":
		return OBJ_TYPE_PS_STATE_CHANGE
	case "ThreadStateChange":
		return OBJ_TYPE_THREAD_STATE_CHANGE
	case "CpuPartition":
		return OBJ_TYPE_CPU_PARTITION
	case "PsSiloContextPaged":
		return OBJ_TYPE_PS_SILO_CTX_PAGED
	case "PsSiloContextNonPaged":
		return OBJ_TYPE_PS_SILO_CTX_NON_PAGED
	case "RegistryTransaction":
		return OBJ_TYPE_REGISTRY_TRANSACTION
	case "DmaAdapter":
		return OBJ_TYPE_DMA_ADAPTER
	case "ALPCPort":
		return OBJ_TYPE_ALPC_PORT
	case "EnergyTracker":
		return OBJ_TYPE_ENERGY_TRACKER
	case "PowerRequest":
		return OBJ_TYPE_POWER_REQUEST
	case "WmiGuid":
		return OBJ_TYPE_WMI_GUID
	case "EtwRegistration":
		return OBJ_TYPE_ETW_REGISTRATION
	case "EtwSessionDemuxEntry":
		return OBJ_TYPE_ETW_SESSION_DEMUX_ENTRY
	case "EtwConsumer":
		return OBJ_TYPE_ETW_CONSUMER
	case "PcwObject":
		return OBJ_TYPE_PCW_OBJECT
	case "CoverageSampler":
		return OBJ_TYPE_COVERAGE_SAMPLER
	case "FilterConnectionPort":
		return OBJ_TYPE_FILTER_CONNECTION_PORT
	case "FilterCommunicationPort":
		return OBJ_TYPE_FILTER_COMM_PORT
	case "NdisCmState":
		return OBJ_TYPE_NDIS_CM_STATE
	case "DxgkSharedResource":
		return OBJ_TYPE_DXGK_SHARED_RSRC
	case "DxgkSharedKeyedMutexObject":
		return OBJ_TYPE_DXGK_SHARED_MUTEX
	case "DxgkSharedSyncObject":
		return OBJ_TYPE_DXGK_SHARED_SYNC
	case "DxgkSharedSwapChainObject":
		return OBJ_TYPE_DXGK_SHARED_SWAP
	case "DxgkSharedProtectedSessionObject":
		return OBJ_TYPE_DXGK_SHARED_SESSION
	case "DxgkSharedBundleObject":
		return OBJ_TYPE_DXGK_SHARED_BUNDLE
	case "DxgkDisplayManagerObject":
		return OBJ_TYPE_DXGK_DISPLAY_MGR
	case "DxgkCompositionObject":
		return OBJ_TYPE_DXGK_COMPOSITION
	case "DxgkCurrentDxgThreadObject":
		return OBJ_TYPE_DXGK_CURRENT_DXG_THREAD
	case "VRegConfigurationContext":
		return OBJ_TYPE_V_REG_CONFIG_CONTEXT
	}
	return OBJ_TYPE_UNKNOWN
}
