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
    while ((status = NtQuerySystemInformation(
        SystemExtendedHandleInformation,
        handleTableInformation,
        bufferSize,
        &returnLenght
        )) == STATUS_INFO_LENGTH_MISMATCH) {
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
    
    *handleCount = handleTableInformation->NumberOfHandles;
    size_t tableSize = (*handleCount) * sizeof(HANDLE_ENTRY);
    handleTable = HeapAlloc((HANDLE_ENTRY*)GetProcessHeap(), HEAP_ZERO_MEMORY, tableSize);

    HP_WORK_POOL HpWorkPool = {0};
    if (!InitHpWorkPoolEx(&HpWorkPool, HP_WORKER_COUNT)) {
        printf("[FATAL] Failed to initialize work pool\n");
        //TODO: handle this properly with fallback
    }

    //* main handle table enumeration loop
    for (int i = 0; i < handleTableInformation->NumberOfHandles; i++) {

        SYSTEM_HANDLE_TABLE_ENTRY_INFO_EX handleInfo = handleTableInformation->Handles[i];

        DWORD pid = handleInfo.UniqueProcessId;
        HANDLE hProcess = OpenProcess(PROCESS_DUP_HANDLE, FALSE, pid);
            
        HANDLE hObject = NULL;
        //TODO: what are the minimum required access rights?
        //TODO: try setting it as 0 default, but query_limited_information for thread or process
        //* Duplicate handle to query information about the object
        //DWORD access = STANDARD_RIGHTS_REQUIRED | GENERIC_READ;
        DWORD access = DUPLICATE_SAME_ACCESS;
        if (hProcess != NULL) {
            DuplicateHandle(hProcess, (HANDLE)(DWORD_PTR)handleInfo.HandleValue,
                    GetCurrentProcess(), &hObject, access, FALSE, 0);
                
            CloseHandle(hProcess);
        }

        handleTable[i].Type    = GetHandleObjectTypeEx(hObject, handleInfo.ObjectTypeIndex);
        handleTable[i].Pid     = handleInfo.UniqueProcessId;
        handleTable[i].Access  = handleInfo.GrantedAccess;
        handleTable[i].Handle  = (DWORD)handleInfo.HandleValue;
        handleTable[i].Address = handleInfo.Object;

        //? This will fill the parameters for the handle entry
        //? and close the handle once finished (see work_pool.c)
        if (handleTable[i].Type == OBJ_TYPE_FILE || handleTable[i].Type == OBJ_TYPE_PIPE) {
            CreateHpTask(HpWorkPool.Queue, hObject, &handleTable[i]);
        } else {
            handleTable[i].Params = GetHandleParameters(
                hObject, handleTable[i].Type, &handleTable[i].paramsSize);
            CloseHandle(hObject);
        }
    }
    //printf("\n");
    HeapFree(GetProcessHeap(), 0, handleTableInformation);
    HpWorkPoolShutdownAndWait(&HpWorkPool);

    return handleTable;
}

// Get packet parameters for a handle event. Remember to free buffer after use.
BYTE* GetHandleParameters(HANDLE hObject, DWORD objectType, size_t* paramsSize) {
    BYTE* parameters = NULL;
    *paramsSize = 0;

    size_t nameParamSize = 0;
    BYTE* nameParam = GetObjectNameParameter(
            hObject, objectType, &nameParamSize);

    if (nameParam != NULL) {
        *paramsSize += nameParamSize;
    }

    size_t extraParamsSize = 0;
    BYTE* extraParams = NULL;

    switch (objectType) {
        case OBJ_TYPE_PROCESS:
            extraParams = GetProcessObjectExtraParams(hObject, &extraParamsSize);
            break;
        case OBJ_TYPE_THREAD:
            extraParams = GetThreadObjectExtraParams(hObject, &extraParamsSize);
            break;
    }

    if (extraParams != NULL) {
        *paramsSize += extraParamsSize;

        if (nameParam == NULL) {
            return extraParams;
        }

        parameters = malloc(*paramsSize);
        memcpy(parameters, nameParam, nameParamSize);
        memcpy(parameters + nameParamSize, extraParams, extraParamsSize);
        
        free(nameParam);
        free(extraParams);
        return parameters;

    } else {
        return nameParam;
    }
}

void FreeHandleTable(HANDLE_ENTRY* handleTable) {
    HeapFree(GetProcessHeap(), 0, handleTable);
}
