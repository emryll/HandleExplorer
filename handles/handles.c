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

    size_t nameParamSize;
    BYTE* nameParam = GetObjectNameParameter(
            hObject, objectType, &nameParamSize);
    *paramsSize += nameParamSize;

    if (nameParam != NULL) {
        parameters = nameParam;
    }

    size_t extraParamsSize;
    BYTE* extraParams = NULL;

    switch (objectType) {
        case OBJ_TYPE_PROCESS:
            extraParams = GetProcessObjectExtraParams(hObject, &extraParamsSize);
        case OBJ_TYPE_THREAD:
            extraParams = GetThreadObjectExtraParams(hObject, &extraParamsSize);
    }

    if (extraParams != NULL) {
        memcpy(parameters + nameParamSize,
                extraParams, extraParamsSize);
        free(extraParams);
    }

    return parameters;
}