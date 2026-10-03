#include <stdio.h>
#include <windows.h>
#include <ntstatus.h>
#include "handles.h"

//?===================================================================+
//?    This file defines object information queries for handles.      |
//?===================================================================+

//? The outer functions to use for name query are:
//?   GetObjectName            * just get the name
//?   GetObjectNameParameter   * get name as parameter ready for packet


BYTE* GetProcessObjectExtraParams(HANDLE hProcess, size_t* paramsSize) {
    //? currently just pid is taken at this point
    //? other details are retrieved in Go later in the code

    DWORD pid = GetProcessId(hProcess);
    return BuildParameter(paramsSize, PARAMETER_UINT32, "Pid", pid);
}

BYTE* GetThreadObjectExtraParams(HANDLE hThread, size_t* paramsSize) {
    size_t pathParamSize = 0;
    size_t pidParamSize  = 0;
    size_t tidParamSize  = 0;
    *paramsSize = 0;

    DWORD tid = GetThreadId(hThread);
    BYTE* tidParam = BuildParameter(&tidParamSize, PARAMETER_UINT32, "Tid", tid);
    if (tidParam != NULL) *paramsSize += tidParamSize;

    DWORD pid = GetProcessIdOfThread(hThread);
    BYTE* pidParam = BuildParameter(&pidParamSize, PARAMETER_UINT32, "Pid", pid);
    if (pidParam != NULL) *paramsSize += pidParamSize;

    BYTE* pathParam = NULL;
    if (pid != 0) {
        HANDLE hProcess = OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, FALSE, pid);
        if (hProcess != NULL) {
            char* path = GetProcessImagePath(hProcess);
            if (path != NULL) {
                pathParam = BuildParameter(&pathParamSize, PARAMETER_ANSISTRING, "Path", path);
                free(path);
            }
            CloseHandle(hProcess);
        }
    }

    *paramsSize = tidParamSize + pidParamSize + pathParamSize;
    if (*paramsSize == 0) return NULL;

    BYTE* parameters = (BYTE*)malloc(*paramsSize); 
    if (parameters == NULL) {
        if (tidParam != NULL) free(tidParam);
        if (pidParam != NULL) free(pidParam);
        if (pathParam != NULL) free(pathParam);
        return NULL;
    }

    size_t cursor = 0;
    if (tidParam != NULL) {
        memcpy(parameters + cursor, tidParam, tidParamSize);
        cursor += tidParamSize;
        free(tidParam);
    }
    if (pidParam != NULL) {
        memcpy(parameters + cursor, pidParam, pidParamSize);
        cursor += pidParamSize;
        free(pidParam);
    }
    if (pathParam != NULL) {
        memcpy(parameters + cursor, pathParam, pathParamSize);
        cursor += pathParamSize;
        free(pathParam);
    }
    return parameters;
}

//*=========================[ Object Name Information ]==========================

BYTE* GetObjectNameParameter(HANDLE hObject, DWORD objectType, size_t* paramSize) {
    char* name = GetObjectName(hObject, objectType);
    if (name == NULL) return NULL;

    BYTE* parameter = BuildParameter(paramSize, PARAMETER_ANSISTRING, "Name", name);
    free(name);

    return parameter;
}

char* GetObjectName(HANDLE hObject, DWORD objectType) {
    if (hObject == NULL || hObject == INVALID_HANDLE_VALUE) return NULL;
    
    switch (objectType) {
        case OBJ_TYPE_PROCESS:
            return GetProcessImagePath(hObject);
            break;

        case OBJ_TYPE_SYMLINK:
            return GetSymlinkTarget(hObject);
            break;

        case OBJ_TYPE_PIPE:
        case OBJ_TYPE_FILE:
            return GetFileObjectName(hObject);
            break;

        /*case OBJ_TYPE_ALPC_PORT:
            return GetAlpcPortName(hObject);
            break;*/

        case OBJ_TYPE_DESKTOP:
        case OBJ_TYPE_WINDOW_STATION:
            return GetWinstaOrDesktopName(hObject);
            break;

        case OBJ_TYPE_JOB:
        case OBJ_TYPE_TIMER:
        case OBJ_TYPE_IRTIMER:
        case OBJ_TYPE_EVENT:
        case OBJ_TYPE_MUTANT:
        case OBJ_TYPE_SEMAPHORE:
        case OBJ_TYPE_SECTION:
        case OBJ_TYPE_DIRECTORY:
        case OBJ_TYPE_IO_COMPLETION:
            // NtQueryObject, no timeout
            return GetObjectNameGeneric(hObject);
    }

    return NULL;
}

char* GetObjectNameGeneric(HANDLE hObject) {
    if (NtQueryObject == NULL) {
        NtQueryObject = (NQO)GetProcAddress(GetModuleHandle("ntdll.dll"), "NtQueryObject");
    }

    size_t initialSize = 1024;
    size_t bufSize = initialSize;
    BYTE* buffer = (BYTE*)malloc(bufSize);
    if (buffer == NULL) return FALSE;

    NTSTATUS status;
    ULONG returnLen;

    while ((status = NtQueryObject(
        hObject,
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
        //printf("[dbg] NtQueryObject failed with NTSTATUS %X\n", status);
        return FALSE;
    }

    char* name = NULL;
    POBJECT_NAME_INFORMATION info = (POBJECT_NAME_INFORMATION)buffer;

    if (info->Name.Length > 0 && info->Name.Buffer != NULL) {
        name = UnicodeToAnsi(info->Name);
    }

    free(info);
    return name;
}
