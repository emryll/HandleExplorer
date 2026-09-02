package tmenu

//*====================[ Object Type Picker ]=======================

//? This is a shared component for picking object types
type objectTypePicker struct {
	objectTypes []string

	cursor   int
	selected map[int]bool
	filter   string

	cols        int
	cellWidth   int
	nameMaxLen  int
	visibleRows int

	boxWidth  int
	gridWidth int
}

var ntObjectTypes = []string{
	"Event",
	"Mutant",
	"Semaphore",
	"Thread",
	"Process",
	"File",
	"Pipe",
	"Key",
	"Token",
	"WmiGuid",
	"Job",
	"TpWorkerFactory",
	"DebugObject",
	"UserApcReserve",
	"Section",
	"Timer",
	"IRTimer",
	"Desktop",
	"Driver",
	"Directory",
	"SymbolicLink",
	"Session",
	"Callback",
	"Adapter",
	"Controller",
	"Device",
	"IoRing",
	"Profile",
	"KeyedEvent",
	"WindowStation",
	"EtwRegistration",
	"EtwSessionDemuxEntry",
	"EtwConsumer",
	"Partition",
	"IoCompletion",
	"IoCompletionReserve",
	"EnergyTracker",
	"ALPC Port",
	"PowerRequest",
	"CoreMessaging",
	"RawInputManager",
	"ActivationObject",
	"Composition",
	"ActivityReference",
	"ProcessStateChange",
	"ThreadStateChange",
	"CpuPartition",
	"DmaAdapter",
	"PcwObject",
	"CoverageSampler",
	"NdisCmState",
	"TmTm",
	"TmTx",
	"TmRm",
	"TmEn",
	"VRegConfigurationContext",
	"PsSiloContextPaged",
	"PsSiloContextNonPaged",
	"RegistryTransaction",
	"WaitCompletionPacket",
	"FilterConnectionPort",
	"FilterCommunicationPort",
	"DxgkSharedResource",
	"DxgkSharedKeyedMutexObject",
	"DxgkSharedSyncObject",
	"DxgkSharedSwapChainObject",
	"DxgkSharedProtectedSessionObject",
	"DxgkSharedBundleObject",
	"DxgkCompositionObject",
	"DxgkDisplayManagerObject",
	"DxgkCurrentDxgThreadObject",
}
