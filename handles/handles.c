#include <windows.h>
#include <ntstatus.h>
#include <stdio.h>
#include "handles.h"

//?==========================================================================+
//?   This file has the core of handle table enumeration using the NT API.   |
//?   This part is written in C instead of Go, because these APIs and the    |
//?     required NT structures are much more of a pain to write in Go...     |
//?==========================================================================+

// this is actually defined in winternl.h,
// but im too lazy to refactor now...
static NQO NtQueryObject = NULL;

// Get the global handle table via NtQuerySystemInformation. It also gets object information,
// which calls NtQueryObject. Note that this call is quite heavy, currently typically taking 1000ms.
// Caller must free returned handle table with FreeHandleTable. NULL is returned upon failure.
HANDLE_ENTRY* GetGlobalHandleTable(size_t* handleCount) {
    PSYSTEM_HANDLE_INFORMATION_EX handleTableInformation;
    HANDLE_ENTRY* handleTable = NULL;
    (*handleCount) = 0;
    NTSTATUS status;

    ULONG initialSize = 0x10000;
    ULONG returnLenght = 0;
    ULONG bufferSize = initialSize;

    NQSI NtQuerySystemInformation = (NQSI)GetProcAddress(GetModuleHandle("ntdll"), "NtQuerySystemInformation");
    handleTableInformation = (PSYSTEM_HANDLE_INFORMATION_EX)HeapAlloc(
        GetProcessHeap(), HEAP_ZERO_MEMORY, bufferSize);
    
    //* Query the global handle table
    while (status = NtQuerySystemInformation(
        SystemExtendedHandleInformation,
        handleTableInformation,
        bufferSize,
        &returnLenght
        ) == STATUS_INFO_LENGTH_MISMATCH) {
            HeapFree(GetProcessHeap(), 0, handleTableInformation);
            bufferSize *= 2;

            // avoid infinite loop and high memory usage
            if (bufferSize > MAX_HANDLE_TABLE_BUFFER) return NULL;
                
            handleTableInformation = (PSYSTEM_HANDLE_INFORMATION_EX)HeapAlloc(
                GetProcessHeap(), HEAP_ZERO_MEMORY, bufferSize);
        }

    if (status != STATUS_SUCCESS) {
        printf("failed to query system information, status: %X\n", status);
        HeapFree(GetProcessHeap(), 0, handleTableInformation);
        return NULL;
    }
    
    DbgResetTracker();
    //* main handle table enumeration loop
    for (int i = 0; i < handleTableInformation->NumberOfHandles; i++) {
        DbgIncrementSeenHandles();

        SYSTEM_HANDLE_TABLE_ENTRY_INFO_EX handleInfo = handleTableInformation->Handles[i];
        /*if (handleInfo.ObjectTypeIndex == 37) {
            printf("\n[1] new handle entry (wintype %d, pid %d)\n",
                handleInfo.ObjectTypeIndex, handleInfo.UniqueProcessId);
        }*/

        //TODO: what is process_query_limited_information needed for??
        HANDLE hProcess = OpenProcess(PROCESS_DUP_HANDLE,
            FALSE, handleInfo.UniqueProcessId);
            
        if (hProcess == NULL) {
            DbgAddFail(FAIL_OPEN_PROCESS, handleInfo.UniqueProcessId, handleInfo.ObjectTypeIndex, GetLastError());
            continue;
        }

        HANDLE hObject = NULL;
        //TODO: what are the minimum required access rights?
        //TODO: try setting it as 0 default, but query_limited_information for thread or process
        //* Duplicate handle to query information about the object
        //DWORD access = STANDARD_RIGHTS_REQUIRED | GENERIC_READ;
        DWORD access = DUPLICATE_SAME_ACCESS;
        if (!DuplicateHandle(hProcess, (HANDLE)(DWORD_PTR)handleInfo.HandleValue,
                GetCurrentProcess(), &hObject, access, FALSE, 0)) {
            DWORD err = GetLastError();
            if (err != ERROR_ACCESS_DENIED && err != ERROR_NOT_SUPPORTED && err != ERROR_INVALID_HANDLE) {
                printf("Failed to duplicate handle, error: %d\n", err);
            }/* else if (handleInfo.ObjectTypeIndex != 50) {
                printf("unexpected error on DuplicateHandle: %d (windows type index %d)\n",
                    err, handleInfo.ObjectTypeIndex);
            }*/
            DbgAddFail(FAIL_DUP_HANDLE, handleInfo.UniqueProcessId, handleInfo.ObjectTypeIndex, err);
            CloseHandle(hProcess);
            continue;
        }
        CloseHandle(hProcess);

        //* create HANDLE_ENTRY
        handleTable = (HANDLE_ENTRY*)realloc(handleTable, ((*handleCount) + 1) * sizeof(HANDLE_ENTRY));
        if (handleTable == NULL) {
            printf("[CRITICAL] Failed to realloc (%dB)\n", ((*handleCount) + 1) * sizeof(HANDLE_ENTRY));
        }

        handleTable[*handleCount].Type    = GetHandleObjectTypeEx(hObject, handleInfo.ObjectTypeIndex);
        handleTable[*handleCount].Pid     = handleInfo.UniqueProcessId;
        handleTable[*handleCount].Access  = handleInfo.GrantedAccess;
        handleTable[*handleCount].Handle  = (DWORD)handleInfo.HandleValue;
        handleTable[*handleCount].Address = handleInfo.Object;
        handleTable[*handleCount].Params  = GetHandleParameters(hObject, handleTable[*handleCount].Type, &handleTable[*handleCount].paramsSize);

        if (handleInfo.ObjectTypeIndex == 37 &&
        handleTable[*handleCount].Type != OBJ_TYPE_FILE &&
        handleTable[*handleCount].Type != OBJ_TYPE_PIPE) {
            printf("\n[dbg] id mismatch!!!!! (file as %d)", handleTable[*handleCount].Type);
        }

        CloseHandle(hObject);
        (*handleCount)++;
    }
    //printf("\n");
    HeapFree(GetProcessHeap(), 0, handleTableInformation);

    DbgPrintFails();
    return handleTable;
}

// Get packet parameters for a handle event. Remember to free buffer after use.
BYTE* GetHandleParameters(HANDLE hObject, DWORD objectType, size_t* paramsSize) {
    BYTE* parameters = NULL;

    //TODO: refactor cleaner, do a generic GetObjectName param, then additional params only if needed

    switch (objectType) {
        case OBJ_TYPE_PROCESS: {
        // process id
            DWORD pid = GetProcessId(hObject);
            size_t pidParamSize;
            BYTE* pidParam = BuildParameter(&pidParamSize, PARAMETER_UINT32, "Pid", pid);
        //  process path
            char path[1026];
            DWORD pathLen = 1026;
            size_t pathParamSize;
            BYTE* pathParam = NULL;
            BOOL ok = QueryFullProcessImageNameA(hObject, 0, path, &pathLen);
            if (!ok) {
                //printf("[dbg] failed to get process %d path (%d)\n", pid, GetLastError());
                pathParamSize = 0;
            } else {
                pathParam = BuildParameter(&pathParamSize, PARAMETER_ANSISTRING, "ImagePath", path);
            }

            // construct parameter buffer
            parameters = (BYTE*)malloc(pidParamSize + pathParamSize);
            memcpy(parameters, pidParam, pidParamSize);
            if (pathParamSize > 0) {
                memcpy(parameters + pidParamSize, pathParam, pathParamSize);
                free(pathParam);
            }
            free(pidParam);
            *paramsSize = pidParamSize + pathParamSize;
            break;
        }
        case OBJ_TYPE_THREAD: {
        // thread id
            DWORD tid = GetThreadId(hObject);
            size_t tidParamSize;
            BYTE* tidParam = BuildParameter(&tidParamSize, PARAMETER_UINT32, "Tid", tid);
        // owning process pid
            DWORD pid = GetProcessIdOfThread(hObject);
            size_t pidParamSize;
            BYTE* pidParam = BuildParameter(&pidParamSize, PARAMETER_UINT32, "Pid", pid);
        // owning process path
            HANDLE hProcess = OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, FALSE, pid);
            char path[1026];
            DWORD pathLen = 1026;
            size_t pathParamSize = 0;
            BYTE* pathParam = NULL;
            if (hProcess != NULL) {
                BOOL ok = QueryFullProcessImageNameA(hProcess, 0, path, &pathLen);
                if (ok) {
                    pathParam = BuildParameter(&pathParamSize, PARAMETER_ANSISTRING, "Path", path);
                }
                CloseHandle(hProcess);
            }

            parameters = (BYTE*)malloc(tidParamSize + pidParamSize + pathParamSize);
            memcpy(parameters, tidParam, tidParamSize);
            memcpy(parameters + tidParamSize, pidParam, pidParamSize);
            if (pathParamSize > 0) {
                memcpy(parameters + tidParamSize + pidParamSize, pathParam, pathParamSize);
                free(pathParam);
            }
            free(tidParam);
            free(pidParam);
            *paramsSize = pidParamSize + tidParamSize + pathParamSize;
            break;
        }
        case OBJ_TYPE_FILE: {
            char path[1026];
            DWORD retLen = GetFinalPathNameByHandleA(hObject, path, 1026, 0);
            if (retLen == 0) {
                printf("[ERROR] Failed to get path of file, error code: %d\n", GetLastError());
                return parameters;
            }
            printf("[dbg] filepath: %s\n", path);
            parameters = BuildParameter(paramsSize, PARAMETER_ANSISTRING, "Name", path);
            break;
        }
        /*case OBJ_TYPE_PIPE: {
            char path[1026];
            DWORD retLen = GetFinalPathNameByHandleA(hObject, path, 1026, VOLUME_NAME_NT | FILE_NAME_NORMALIZED);
            if (retLen == 0) {
                printf("[ERROR] Failed to get name of pipe, error code: %d\n", GetLastError());
                return parameters;
            }
            parameters = BuildParameter(paramsSize, PARAMETER_ANSISTRING, "Name", path);
            break;
        }*/
        case OBJ_TYPE_SYMLINK:
            ULONG returnLength = 0;
            UNICODE_STRING ucTarget;
            ucTarget.Buffer = malloc(0x1000);
            ucTarget.MaximumLength = 0x1000;

            if (NtQuerySymbolicLinkObject == NULL) {
                NtQuerySymbolicLinkObject = (NQSLO)GetProcAddress(GetModuleHandle("ntdll.dll"), "NtQuerySymbolicLinkObject");
            }

            NTSTATUS status = NtQuerySymbolicLinkObject(hObject, &ucTarget, &returnLength);
            if (status == STATUS_BUFFER_TOO_SMALL) {
                ucTarget.Buffer = malloc(returnLength);
                ucTarget.MaximumLength = returnLength;
                status = NtQuerySymbolicLinkObject(hObject, &ucTarget, &returnLength);
            }
            if (status != STATUS_SUCCESS) {
                printf("[ERROR] Failed to query symlink info, NTSTATUS: 0x%x\n", status);
                break;
            }
            if (returnLength == 0 || ucTarget.Length == 0) break;
            char* target = UnicodeToAnsi(ucTarget);
            free(ucTarget.Buffer);
            if (target == NULL) {
                printf("[ERROR] Failed to convert UNICODE_STRING to ansi\n");
                break;
            }
            
            parameters = BuildParameter(paramsSize, PARAMETER_ANSISTRING, "Name", target);
            free(target);
            break;
        /*
        case TYPE_TOKEN:
        // owning process
        // access rights or something like that
            break;*/
        case OBJ_TYPE_DESKTOP:
        case OBJ_TYPE_WINDOW_STATION: {
            char* name = GetWinstaOrDesktopName(hObject);
            if (name == NULL) break;

            parameters = BuildParameter(paramsSize, PARAMETER_ANSISTRING, "Name", name);
            free(name);
        }
        case OBJ_TYPE_JOB:
        case OBJ_TYPE_TIMER:
        case OBJ_TYPE_IRTIMER:
        case OBJ_TYPE_EVENT:
        case OBJ_TYPE_MUTANT:
        case OBJ_TYPE_SEMAPHORE:
        case OBJ_TYPE_SECTION:
        case OBJ_TYPE_DIRECTORY:
        case OBJ_TYPE_IO_COMPLETION:
        case OBJ_TYPE_ALPC_PORT:
        case OBJ_TYPE_PIPE:
            char* name = GetObjectNameWithTimeout(hObject, 1000);
            if (name == NULL) break;

            parameters = BuildParameter(paramsSize, PARAMETER_ANSISTRING, "Name", name);
            free(name);
            break;
    }
    return parameters;
}

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

DWORD GetHandleObjectType2(HANDLE hObject) {
    if (NtQueryObject == NULL) {
        NtQueryObject = (NQO)GetProcAddress(GetModuleHandle("ntdll"), "NtQueryObject");
    }
    DWORD bufSize = sizeof(PUBLIC_OBJECT_TYPE_INFORMATION);
    PUBLIC_OBJECT_TYPE_INFORMATION* typeInfo = (PUBLIC_OBJECT_TYPE_INFORMATION*)malloc(bufSize);
    NTSTATUS status = NtQueryObject(hObject, ObjectTypeInformation, (PVOID)typeInfo, bufSize, &bufSize);
    if ((status == STATUS_BUFFER_OVERFLOW) || (status == STATUS_INFO_LENGTH_MISMATCH)) {
        typeInfo = (PUBLIC_OBJECT_TYPE_INFORMATION*)realloc(typeInfo, bufSize);
        if (typeInfo == NULL) {
            printf("Failed to realloc (%dB)\n", bufSize);
            free(typeInfo);
            return OBJ_TYPE_UNKNOWN;
        }
        status = NtQueryObject(hObject, ObjectTypeInformation, (PVOID)typeInfo, bufSize, &bufSize);
    }
    if ((status != STATUS_SUCCESS) {
        free(typeInfo);
        return OBJ_TYPE_UNKNOWN;
    }

    DWORD id = GetObjectTypeIdFromName(typeInfo->TypeName);
    free(typeInfo);
    return id;
}

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
        wprintf(L"[dbg] unknown object type: %ls\n", typeInfo->TypeName.Buffer);
    }
    free(typeInfo);
    return type;
}

char* GetObjectNameWithTimeout(HANDLE hObject, DWORD dwMilliseconds) {
    OBJECT_NAME_QUERY query = {0};
    query.hObject = hObject;
    HANDLE hThread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)GetObjectName, &hObject, 0, NULL);
    WaitForSingleObject(hThread, dwMilliseconds);
    return query.out;
}

BOOL GetObjectName(OBJECT_NAME_QUERY* query) {
    if (query == NULL) return FALSE;

    if (NtQueryObject == NULL) {
        NtQueryObject = (NQO)GetProcAddress(GetModuleHandle("ntdll.dll"), "NtQueryObject");
    }

    size_t initalSize = 1024;
    size_t bufSize = initalSize;
    BYTE* buffer = (BYTE*)malloc(bufSize);
    if (buffer == NULL) return FALSE;

    NTSTATUS status;
    ULONG returnLen;

    while ((status = NtQueryObject(
        query->hObject,
        ObjectNameInformation,
        buffer,
        bufSize,
        &returnLen
    )) == STATUS_BUFFER_TOO_SMALL ||
    status == STATUS_INFO_LENGTH_MISMATCH) {
        bufSize *= 2;
        buffer = realloc(buffer, bufSize);
        if (buffer == NULL) return FALSE;
    }

    if (status != STATUS_SUCCESS) {
        printf("[dbg] NtQueryObject failed with NTSTATUS %X\n", status);
        return FALSE;
    }
    POBJECT_NAME_INFORMATION info = (POBJECT_NAME_INFORMATION)buffer;
    if (info->Name.Length > 0 && info->Name.Buffer != NULL) {
        int wlen = info->Name.Length / sizeof(WCHAR); // no null terminator guaranteed
        query->out = UnicodeToAnsi(info->Name);
        free(buffer);
        return TRUE;
    }
    free(buffer);
    return FALSE;
}

char* GetObjectName2(HANDLE hObject) {
    OBJECT_NAME_QUERY query = {0};
    query.hObject = hObject;

    if (GetObjectName(&query)) {
        return NULL;
    }
    return query.out;
}

char* GetWinstaOrDesktopName(HANDLE hObject) {
    DWORD lenNeeded;
    char* nameBuf = malloc(sizeof(char) * 1026);
    //TODO: check for too short name and realloc
    if (!GetUserObjectInformation(hObject, UOI_NAME, nameBuf, sizeof(nameBuf), &lenNeeded)) {
        return NULL;
    }
    return nameBuf;
}