#include <windows.h>
#include <ntstatus.h>
#include <stdio.h>
#include "handles.h"

static NQO NtQueryObject = NULL;

// Get the global handle table via NtQuerySystemInformation. It also gets object information,
// which calls NtQueryObject. Note that this call is quite heavy, currently typically taking 1000ms.
// Caller must free returned handle table with FreeHandleTable. NULL is returned upon failure.
HANDLE_ENTRY* GetGlobalHandleTable(size_t* handleCount) {
    HANDLE_ENTRY* handleTable = NULL;
    (*handleCount) = 0;
    ULONG hiLenght = 0;
    ULONG infoSize = HANDLE_INFO_MEM_BLOCK;

    NQSI NtQuerySystemInformation = (NQSI)GetProcAddress(GetModuleHandle("ntdll"), "NtQuerySystemInformation");
    PSYSTEM_HANDLE_INFORMATION handleTableInformation = (PSYSTEM_HANDLE_INFORMATION)HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, infoSize);
    NTSTATUS status = NtQuerySystemInformation(SystemHandleInformation, handleTableInformation, infoSize, &hiLenght);
    if (status == STATUS_INFO_LENGTH_MISMATCH) {
        while (status == STATUS_INFO_LENGTH_MISMATCH) {
            HeapFree(GetProcessHeap(), 0, handleTableInformation);
            infoSize += HANDLE_INFO_MEM_BLOCK;
            if (infoSize > 10000000) return NULL; // avoid infinite loop with 10MB limit
            handleTableInformation = (PSYSTEM_HANDLE_INFORMATION)HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, infoSize);
            status = NtQuerySystemInformation(SystemHandleInformation, handleTableInformation, infoSize, &hiLenght);
        }
    } else if (status != STATUS_SUCCESS) {
        printf("failed to query system information, status: %X\n", status);
        HeapFree(GetProcessHeap(), 0, handleTableInformation);
        return NULL;
    }

    for (int i = 0; i < handleTableInformation->NumberOfHandles; i++) {
        SYSTEM_HANDLE_TABLE_ENTRY_INFO handleInfo = handleTableInformation->Handles[i];

        HANDLE hProcess = OpenProcess(PROCESS_DUP_HANDLE | PROCESS_QUERY_INFORMATION | PROCESS_VM_READ,
            FALSE, handleInfo.UniqueProcessId);
        if (hProcess == NULL) {
            continue;
        }

        HANDLE hObject = NULL;
        //TODO: what are the minimum required access rights?
        //* Duplicate handle to query information about the object
        if (!DuplicateHandle(hProcess, (HANDLE)(DWORD_PTR)handleInfo.HandleValue, GetCurrentProcess(),
                &hObject, STANDARD_RIGHTS_REQUIRED | GENERIC_READ, FALSE, 0)) {
            DWORD err = GetLastError();
            if (err != ERROR_ACCESS_DENIED && err != ERROR_NOT_SUPPORTED && err != ERROR_INVALID_HANDLE) {
                printf("Failed to duplicate handle, error: %d\n", err);
            }
            CloseHandle(hProcess);
            continue;
        }
        CloseHandle(hProcess);

        //* create HANDLE_ENTRY
        handleTable = (HANDLE_ENTRY*)realloc(handleTable, ((*handleCount) + 1) * sizeof(HANDLE_ENTRY));
        if (handleTable == NULL) {
            printf("[CRITICAL] Failed to realloc (%dB)\n", ((*handleCount) + 1) * sizeof(HANDLE_ENTRY));
        }

        handleTable[*handleCount].Type   = GetHandleObjectType(hObject);
        handleTable[*handleCount].Pid    = handleInfo.UniqueProcessId;
        handleTable[*handleCount].Access = handleInfo.GrantedAccess;
        handleTable[*handleCount].Handle = (DWORD)handleInfo.HandleValue;
        handleTable[*handleCount].Params = GetHandleParameters(hObject, handleTable[*handleCount].Type, &handleTable[*handleCount].paramsSize);
        CloseHandle(hObject);
        (*handleCount)++;
    }
    HeapFree(GetProcessHeap(), 0, handleTableInformation);
    return handleTable;
}

// Get packet parameters for a handle event. Remember to free buffer after use.
BYTE* GetHandleParameters(HANDLE hObject, DWORD objectType, size_t* paramsSize) {
    BYTE* parameters = NULL;
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
                printf("[dbg] failed to get process %d path (%d)\n", pid, GetLastError());
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
        
            parameters = (BYTE*)malloc(tidParamSize + pidParamSize);
            memcpy(parameters, tidParam, tidParamSize);
            memcpy(parameters + pidParamSize, pidParam, pidParamSize);
            free(tidParam);
            free(pidParam);
            *paramsSize = pidParamSize + tidParamSize;
            break;
        }
        case OBJ_TYPE_FILE: {
            char path[1026];
            DWORD retLen = GetFinalPathNameByHandleA(hObject, path, 1026, 0);
            if (retLen == 0) {
                printf("[ERROR] Failed to get path of file, error code: %d\n", GetLastError());
                return parameters;
            }
            parameters = BuildParameter(paramsSize, PARAMETER_ANSISTRING, "Name", path);
            break;
        }
        case OBJ_TYPE_PIPE: {
            char path[1026];
            DWORD retLen = GetFinalPathNameByHandleA(hObject, path, 1026, VOLUME_NAME_NT | FILE_NAME_NORMALIZED);
            if (retLen == 0) {
                printf("[ERROR] Failed to get name of pipe, error code: %d\n", GetLastError());
                return parameters;
            }
            parameters = BuildParameter(paramsSize, PARAMETER_ANSISTRING, "Name", path);
            break;
        }
        case OBJ_TYPE_EVENT:
        case OBJ_TYPE_MUTEX:
        case OBJ_TYPE_SEMAPHORE:
            char* name = GetObjectName(hObject);
            if (name == NULL) break;

            parameters = BuildParameter(paramsSize, PARAMETER_ANSISTRING, "Name", name);
            free(name);
            break;
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
    }
    return parameters;
}
char* GetObjectName(HANDLE hObject) {
    if (NtQueryObject == NULL) {
        NtQueryObject = (NQO)GetProcAddress(GetModuleHandle("ntdll.dll"), "NtQueryObject");
    }
    
    // First try with a stack buffer; grow if needed.
    BYTE stackBuf[1024];
    ULONG returnLen = 0;
    NTSTATUS status = NtQueryObject(hObject,
        ObjectNameInformation, stackBuf, sizeof(stackBuf), &returnLen);
    //if (status == STATUS_BUFFER_TOO_SMALL || status == STATUS_INFO_LENGTH_MISMATCH) {
        //TODO:
    //}
    if (status != STATUS_SUCCESS) {
        printf("[dbg] NtQueryObject failed with NTSTATUS %X\n", status);
        return NULL;
    }
    POBJECT_NAME_INFORMATION info = (POBJECT_NAME_INFORMATION)stackBuf;
    if (info->Name.Length > 0 && info->Name.Buffer != NULL) {
        int wlen = info->Name.Length / sizeof(WCHAR); // no null terminator guaranteed
        return UnicodeToAnsi(info->Name);
    }
    return NULL;
}
