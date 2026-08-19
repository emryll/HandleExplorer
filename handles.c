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
