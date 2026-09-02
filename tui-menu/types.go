package tmenu

import "charm.land/bubbles/v2/textinput"

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

//*=====================[ Object Search Filter ]=======================

type HandleFilter struct {
	ObjectTypes []string
	ObjectName  string
	Process     string
	AccessLevel string
}

type objectFocusField int

const (
	objectFocusTypes objectFocusField = iota
	objectFocusName
	objectFocusProcess
	objectFocusAccess

	objectFocusCount
)

type objectFilterModel struct {
	focus objectFocusField

	types objectTypePicker

	objectName  textinput.Model
	process     textinput.Model
	accessLevel textinput.Model

	width  int
	height int

	quitting  bool
	submitted bool

	result *HandleFilter
}

//*=====================[ Process Search Filter ]=======================

type ProcessFilter struct {
	Path               string
	DirectoryAllowlist []string
	ParentProcess      []string
	SignatureStatus    []string
	Elevation          []string
	ObjectTypes        []string
}

type processModel struct {
	focus processFocusField

	path           *textinput.Model
	allowlistInput *textinput.Model
	parentInput    *textinput.Model

	allowlist []string
	parent    []string

	signatureStatus [4]bool
	elevation       [2]bool

	objectTypes   []string
	typeCursor    int
	selectedTypes map[int]bool
	typeFilter    string

	typeCols        int
	typeCellWidth   int
	typeVisibleRows int
	typeBoxWidth    int
	typeGridWidth   int

	width  int
	height int

	submitted bool
	quitting  bool
	result    *ProcessFilter
}

type processFocusField int

const (
	processFocusPath processFocusField = iota
	processFocusAllowlist
	processFocusParent
	processFocusProperties
	processFocusObjectTypes
	processFocusFieldCount
)

const (
	signatureSigned = iota
	signatureNotSigned
	signatureHashMismatch
	signatureOther
)

const (
	elevationElevated = iota
	elevationNotElevated
)

const (
	processTypeMinCellWidth = 24

	// The object type section is the flexible part of the UI.
	processMinVisibleRows = 1
	processMaxVisibleRows = 7

	processFormSideMargin = 4
)

//*==================[ NT object type names ]=====================

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
