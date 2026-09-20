#include <map>
#include <windows.h>
#include <stdio.h>
#include "handles.h"

// 1MB, type info should be nowhere near this big
#define MAX_BUFFER_SIZE 1000000

//?===============================================================+
//?       I hate C++ so I'm only using it for maps...             |
//?                                                               |
//?    Handle enumeration returns windows internal type index,    |
//?    which is not stable. That's why it needs to be converted.  |
//?                                                               |
//?    This object type resolution is with the "better way",      |
//?      as is done in the System Informer project, via cache.    |
//?===============================================================+

static BOOLEAN ObjectTypeLookupInitialized;
static std::map<DWORD, DWORD> ObjectTypeLookupTable;

extern "C" {
    NTSTATUS FillObjectTypeLookupTable() {
        POBJECT_TYPES_INFORMATION typesInfo = NULL;
        NTSTATUS status = GetHandleTypesInformation(&typesInfo);
        if (status != STATUS_SUCCESS) {
            return status;
        }

        POBJECT_TYPE_INFORMATION type = FIRST_OBJECT_TYPE(typesInfo);
        for (ULONG i = 0; i < typesInfo->NumberOfTypes; i++) {
            DWORD id = GetObjectTypeIdFromName(type->TypeName);
            if (id != OBJ_TYPE_UNKNOWN) {
                ObjectTypeLookupTable.insert({type->TypeIndex, id})
            }

            type = NEXT_OBJECT_TYPE(type);
        }

        ObjectTypeLookupInitialized = TRUE;
        HeapFree(GetProcessHeap(), 0, typesInfo);
        return status;
    }

    NTSTATUS GetHandleTypesInformation(POBJECT_TYPES_INFORMATION* out) {
        ULONG bufferSize = 0x1000;
        ULONG returnLength;
        NTSTATUS status;

        PVOID buffer = HeapAlloc(
            GetProcessHeap(), HEAP_ZERO_MEMORY, bufferSize);
        
        while ((status = NtQueryObject(
            NULL,
            ObjectTypesInformation,
            buffer,
            bufferSize,
            &returnLength
        )) == STATUS_INFO_LENGTH_MISMATCH) {

            HeapFree(GetProcessHeap(), 0, buffer);
            bufferSize *= 2;
            buffer = HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, bufferSize);

            if (bufferSize > MAX_BUFFER_SIZE) {
                return STATUS_BUFFER_OVERFLOW;
            }
        }

        if (status != STATUS_SUCCESS) {
            HeapFree(GetProcessHeap(), 0, buffer);
            return status;
        }

        *out = (POBJECT_TYPES_INFORMATION)buffer;
        return status;
    }
}