#include <stdio.h>
#include <windows.h>
#include <ntstatus.h>
#include "handles.h"

//?========================================================================+
//?    This file has the C code for resolving object types from handles.   |
//?      The rest of the code for this can be found in nt_type.cpp         |
//?========================================================================+


// Get the object type of a handle as own stable type id.
// This will try to use cached object type lookup table, using
// GetHandleObjectType2 as fallback (NtQueryObject ObjectTypeInformation)
DWORD GetHandleObjectTypeEx(HANDLE hObject, UCHAR typeWindex) {
    if (typeWindex == 0) { // the type indexes start at 2 (for some reason)
        return GetHandleObjectType2(hObject);
    }

    // cached object type id lookup
    DWORD id = GetObjectTypeIdFromWindex(typeWindex);
    if (id == OBJ_TYPE_UNKNOWN) {
        // Use NtQueryObject ObjectTypeInformation as fallback
        return GetHandleObjectType2(hObject);
    }

    if (id == OBJ_TYPE_FILE &&
        hObject != INVALID_HANDLE_VALUE &&
        hObject != NULL) {
        
        if (GetFileType(hObject) == FILE_TYPE_PIPE) {
            id = OBJ_TYPE_PIPE;
        }
    }
    return id;
}

// Get object type by handle, with a NtQueryObject call.
// Note that it's preferred to use GetObjectTypeIdFromWindex.
DWORD GetHandleObjectType2(HANDLE hObject) {
    if (NtQueryObject == NULL) {
        NtQueryObject = (NQO)GetProcAddress(GetModuleHandle("ntdll"), "NtQueryObject");
    }
    DWORD bufSize = sizeof(PUBLIC_OBJECT_TYPE_INFORMATION);
    PUBLIC_OBJECT_TYPE_INFORMATION* typeInfo = (PUBLIC_OBJECT_TYPE_INFORMATION*)malloc(bufSize);

    NTSTATUS status = NtQueryObject(
        hObject,
        ObjectTypeInformation,
        (PVOID)typeInfo,
        bufSize,
        &bufSize
    );

    if ((status != STATUS_SUCCESS)) {
        free(typeInfo);
        return OBJ_TYPE_UNKNOWN;
    }

    DWORD id = GetObjectTypeIdFromName(typeInfo->TypeName);
    free(typeInfo);
    return id;
}

// Get stable internal object type id from NT type name.
DWORD GetObjectTypeIdFromName(UNICODE_STRING name) {
    if (name.Buffer == NULL || name.Length == 0) {
        return OBJ_TYPE_UNKNOWN;
    }

    //TODO: convert this to hash checks
    //TODO: also iterate a list instead of this

    DWORD type = OBJ_TYPE_UNKNOWN;
    if (wcscmp(name.Buffer, L"Process") == 0) {
        type = OBJ_TYPE_PROCESS;
    } else if (wcscmp(name.Buffer, L"Thread") == 0) {
        type = OBJ_TYPE_THREAD;
    } else if (wcscmp(name.Buffer, L"File") == 0) {
        type = OBJ_TYPE_FILE;
    } else if (wcscmp(name.Buffer, L"Event") == 0) {
        type = OBJ_TYPE_EVENT;
    } else if (wcscmp(name.Buffer, L"Mutant") == 0) {
        type = OBJ_TYPE_MUTANT;
    } else if (wcscmp(name.Buffer, L"Semaphore") == 0) {
        type = OBJ_TYPE_SEMAPHORE;
    } else if (wcscmp(name.Buffer, L"Section") == 0) {
        type = OBJ_TYPE_SECTION;
    } else if (wcscmp(name.Buffer, L"Session") == 0) {
        type = OBJ_TYPE_SESSION;
    } else if (wcscmp(name.Buffer, L"Key") == 0) {
        type = OBJ_TYPE_KEY;
    } else if (wcscmp(name.Buffer, L"Directory") == 0) {
        type = OBJ_TYPE_DIRECTORY;
    } else if (wcscmp(name.Buffer, L"SymbolicLink") == 0) {
        type = OBJ_TYPE_SYMLINK;
    } else if (wcscmp(name.Buffer, L"Token") == 0) {
        type = OBJ_TYPE_TOKEN;
    } else if (wcscmp(name.Buffer, L"Job") == 0) {
        type = OBJ_TYPE_JOB;
    } else if (wcscmp(name.Buffer, L"Device") == 0) {
        type = OBJ_TYPE_DEVICE;
    } else if (wcscmp(name.Buffer, L"Desktop") == 0) {
        type = OBJ_TYPE_DESKTOP;
    } else if (wcscmp(name.Buffer, L"Partition") == 0) {
        type = OBJ_TYPE_PARTITION;
    } else if (wcscmp(name.Buffer, L"DebugObject") == 0) {
        type = OBJ_TYPE_DEBUG_OBJECT;
    } else if (wcscmp(name.Buffer, L"Callback") == 0) {
        type = OBJ_TYPE_CALLBACK;
    } else if (wcscmp(name.Buffer, L"Adapter") == 0) {
        type = OBJ_TYPE_ADAPTER;
    } else if (wcscmp(name.Buffer, L"Controller") == 0) {
        type = OBJ_TYPE_CONTROLLER;
    } else if (wcscmp(name.Buffer, L"Device") == 0) {
        type = OBJ_TYPE_DEVICE;
    } else if (wcscmp(name.Buffer, L"Driver") == 0) {
        type = OBJ_TYPE_DRIVER;
    } else if (wcscmp(name.Buffer, L"IoRing") == 0) {
        type = OBJ_TYPE_IO_RING;
    } else if (wcscmp(name.Buffer, L"TmTm") == 0) {
        type = OBJ_TYPE_TM_TM;
    } else if (wcscmp(name.Buffer, L"TmTx") == 0) {
        type = OBJ_TYPE_TM_TX;
    } else if (wcscmp(name.Buffer, L"TmRm") == 0) {
        type = OBJ_TYPE_TM_RM;
    } else if (wcscmp(name.Buffer, L"TmEn") == 0) {
        type = OBJ_TYPE_TM_EN;
    } else if (wcscmp(name.Buffer, L"Timer") == 0) {
        type = OBJ_TYPE_TIMER;
    } else if (wcscmp(name.Buffer, L"IRTimer") == 0) {
        type = OBJ_TYPE_IRTIMER;
    } else if (wcscmp(name.Buffer, L"Profile") == 0) {
        type = OBJ_TYPE_PROFILE;
    } else if (wcscmp(name.Buffer, L"KeyedEvent") == 0) {
        type = OBJ_TYPE_KEYED_EVENT;
    } else if (wcscmp(name.Buffer, L"WindowStation") == 0) {
        type = OBJ_TYPE_WINDOW_STATION;
    } else if (wcscmp(name.Buffer, L"Composition") == 0) {
        type = OBJ_TYPE_COMPOSITION;
    } else if (wcscmp(name.Buffer, L"RawInputManager") == 0) {
        type = OBJ_TYPE_RAW_INPUT_MANAGER;
    } else if (wcscmp(name.Buffer, L"CoreMessaging") == 0) {
        type = OBJ_TYPE_CORE_MESSAGING;
    } else if (wcscmp(name.Buffer, L"ActivationObject") == 0) {
        type = OBJ_TYPE_ACTIVATION_OBJECT;
    } else if (wcscmp(name.Buffer, L"TpWorkerFactory") == 0) {
        type = OBJ_TYPE_TP_WORKER_FACTORY;
    } else if (wcscmp(name.Buffer, L"IoCompletion") == 0) {
        type = OBJ_TYPE_IO_COMPLETION;
    } else if (wcscmp(name.Buffer, L"WaitCompletionPacket") == 0) {
        type = OBJ_TYPE_WAIT_COMPLETION_PACKET;
    } else if (wcscmp(name.Buffer, L"UserApcReserve") == 0) {
        type = OBJ_TYPE_USER_APC_RESERVE;
    } else if (wcscmp(name.Buffer, L"IoCompletionReserve") == 0) {
        type = OBJ_TYPE_IO_COMP_RESERVE;
    } else if (wcscmp(name.Buffer, L"ActivityReference") == 0) {
        type = OBJ_TYPE_ACTIVITY_REFERENCE;
    } else if (wcscmp(name.Buffer, L"ProcessStateChange") == 0) {
        type = OBJ_TYPE_PS_STATE_CHANGE;
    } else if (wcscmp(name.Buffer, L"ThreadStateChange") == 0) {
        type = OBJ_TYPE_THREAD_STATE_CHANGE;
    } else if (wcscmp(name.Buffer, L"CpuPartition") == 0) {
        type = OBJ_TYPE_CPU_PARTITION;
    } else if (wcscmp(name.Buffer, L"PsSiloContextPaged") == 0) {
        type = OBJ_TYPE_PS_SILO_CTX_PAGED;
    } else if (wcscmp(name.Buffer, L"PsSiloContextNonPaged") == 0) {
        type = OBJ_TYPE_PS_SILO_CTX_NON_PAGED;
    } else if (wcscmp(name.Buffer, L"RegistryTransaction") == 0) {
        type = OBJ_TYPE_REGISTRY_TRANSACTION;
    } else if (wcscmp(name.Buffer, L"DmaAdapter") == 0) {
        type = OBJ_TYPE_DMA_ADAPTER;
    } else if (wcscmp(name.Buffer, L"ALPC Port") == 0) {
        type = OBJ_TYPE_ALPC_PORT;
    } else if (wcscmp(name.Buffer, L"EnergyTracker") == 0) {
        type = OBJ_TYPE_ENERGY_TRACKER;
    } else if (wcscmp(name.Buffer, L"PowerRequest") == 0) {
        type = OBJ_TYPE_POWER_REQUEST;
    } else if (wcscmp(name.Buffer, L"WmiGuid") == 0) {
        type = OBJ_TYPE_WMI_GUID;
    } else if (wcscmp(name.Buffer, L"EtwRegistration") == 0) {
        type = OBJ_TYPE_ETW_REGISTRATION;
    } else if (wcscmp(name.Buffer, L"EtwSessionDemuxEntry") == 0) {
        type = OBJ_TYPE_ETW_SESSION_DEMUX_ENTRY;
    } else if (wcscmp(name.Buffer, L"EtwConsumer") == 0) {
        type = OBJ_TYPE_ETW_CONSUMER;
    } else if (wcscmp(name.Buffer, L"PcwObject") == 0) {
        type = OBJ_TYPE_PCW_OBJECT;
    } else if (wcscmp(name.Buffer, L"CoverageSampler") == 0) {
        type = OBJ_TYPE_COVERAGE_SAMPLER;
    } else if (wcscmp(name.Buffer, L"FilterConnectionPort") == 0) {
        type = OBJ_TYPE_FILTER_CONNECTION_PORT;
    } else if (wcscmp(name.Buffer, L"FilterCommunicationPort") == 0) {
        type = OBJ_TYPE_FILTER_COMM_PORT;
    } else if (wcscmp(name.Buffer, L"NdisCmState") == 0) {
        type = OBJ_TYPE_NDIS_CM_STATE;
    } else if (wcscmp(name.Buffer, L"DxgkSharedResource") == 0) {
        type = OBJ_TYPE_DXGK_SHARED_RSRC;
    } else if (wcscmp(name.Buffer, L"DxgkSharedKeyedMutexObject") == 0) {
        type = OBJ_TYPE_DXGK_SHARED_MUTEX;
    } else if (wcscmp(name.Buffer, L"DxgkSharedSyncObject") == 0) {
        type = OBJ_TYPE_DXGK_SHARED_SYNC;
    } else if (wcscmp(name.Buffer, L"DxgkSharedSwapChainObject") == 0) {
        type = OBJ_TYPE_DXGK_SHARED_SWAP;
    } else if (wcscmp(name.Buffer, L"DxgkDisplayManagerObject") == 0) {
        type = OBJ_TYPE_DXGK_DISPLAY_MGR;
    } else if (wcscmp(name.Buffer, L"DxgkSharedProtectedSessionObject") == 0) {
        type = OBJ_TYPE_DXGK_SHARED_SESSION;
    } else if (wcscmp(name.Buffer, L"DxgkSharedBundleObject") == 0) {
        type = OBJ_TYPE_DXGK_SHARED_BUNDLE;
    } else if (wcscmp(name.Buffer, L"DxgkCompositionObject") == 0) {
        type = OBJ_TYPE_DXGK_COMPOSITION;
    } else if (wcscmp(name.Buffer, L"DxgkCurrentDxgThreadObject") == 0) {
        type = OBJ_TYPE_DXGK_CURRENT_DXG_THREAD;
    } else if (wcscmp(name.Buffer, L"VRegConfigurationContext") == 0) {
        type = OBJ_TYPE_V_REG_CONFIG_CONTEXT;
    }

    if (type == OBJ_TYPE_UNKNOWN) {
        wprintf(L"[dbg] unknown object type: %ls\n", name.Buffer);
    }
    return type;
}
