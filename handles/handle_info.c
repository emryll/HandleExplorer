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
            // NtQueryObject, no timeout
            return GetObjectNameGeneric(hObject);
    }

    return NULL;
}