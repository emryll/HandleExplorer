package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
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

// Translate internal object type enum into name.
func GetTypeName(object int) string {
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
	case OBJ_TYPE_V_REG_CONFIG_CONTEXT:
		return "VRegConfigurationContext"
	}
	return "(unknown)"
}

func PrintBanner() {
	yellow := color.NewColor(color.FgHiYellow, color.Bold)
	fmt.Println("\t   __ _____   _  _____  __   ____   ")
	fmt.Println("\t  / // / _ | / |/ / _ \\/ /  / __/   ")
	fmt.Println("\t / _  / __ |/    / // / /__/ _/     ")
	fmt.Println("\t/_//_/_/ |_/_/|_/____/____/___/     ")
	yellow.Printf("\t  / __/_ __ ___  / /__  _______ ____\n")
	yellow.Printf("\t / _/ \\ \\ // _ \\/ / _ \\/ __/ -_) __/\n")
	yellow.Printf("\t/___//_\\_\\/ .__/_/\\___/_/  \\__/_/   \n")
	yellow.Printf("\t         /_/                        \n")
	fmt.Printf("\t\t\tv%d.%d by emryll\n\n", MAJOR_VERSION, MINOR_VERSION)

	PrintDescription()
}

func PrintDescription() {
	fmt.Println("\tThis is a commandline-tool for searching")
	fmt.Println("\t& analyzing object access through handles.\n")
	fmt.Println("\tTo view available commands, run \"help\"")
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
