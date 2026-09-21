#include <windows.h>
#include <handles.h>


BYTE* GetObjectNameParameter(HANDLE hObject, DWORD objectType, size_t* paramSize) {
    char* name = GetObjectName(hObject, objectType);
    if (name == NULL) return NULL;

    BYTE* parameter = BuildParameter(paramSize, PARAMETER_ANSISTRING, "Name", name);
    free(name);

    return parameter;
}

char* GetObjectName(HANDLE hObject, DWORD objectType) {
    switch (objectType) {
        case OBJ_TYPE_PROCESS:
            return GetProcessImagePath(hObject);

        case OBJ_TYPE_SYMLINK:
            return GetSymlinkTarget(hObject);

        case OBJ_TYPE_FILE:
            //TODO: try what diversenok suggested
            return GetFileObjectName(hObject);

        case OBJ_TYPE_ALPC_PORT:
            return GetAlpcPortName(hObject);

        case OBJ_TYPE_DESKTOP:
        case OBJ_TYPE_WINDOW_STATION:
            return GetWinstaOrDesktopName(hObject);

        case OBJ_TYPE_JOB:
        case OBJ_TYPE_TIMER:
        case OBJ_TYPE_IRTIMER:
        case OBJ_TYPE_EVENT:
        case OBJ_TYPE_MUTANT:
        case OBJ_TYPE_SEMAPHORE:
        case OBJ_TYPE_SECTION:
        case OBJ_TYPE_DIRECTORY:
        case OBJ_TYPE_IO_COMPLETION:
        case OBJ_TYPE_PIPE:
            OBJECT_NAME_QUERY query = {0};
            query.hObject = hObject;

            // NtQueryObject, no timeout
            GetObjectNameGeneric(query);
            return query.out;
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
        printf("[dbg] NtQueryObject failed with NTSTATUS %X\n", status);
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

char* GetObjectNameWithTimeout(HANDLE hObject, DWORD dwMilliseconds) {
    OBJECT_NAME_QUERY query = {0};
    query.hObject = hObject;
    HANDLE hThread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)GetObjectNameGeneric, &query, 0, NULL);
    WaitForSingleObject(hThread, dwMilliseconds);
    return query.out;
}

void GetObjectName2(OBJECT_NAME_QUERY* query) {
    query.out = GetObjectNameGeneric(query.hObject)
}
