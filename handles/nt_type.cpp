#include <unordered_map>
#include <mutex>
#include <ntstatus.h>
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

static NTSTATUS ObjectTypeLookupInitStatus;
static std::once_flag ObjectTypeLookupOnceFlag;
static std::unordered_map<DWORD, DWORD> ObjectTypeLookupTable;

extern "C" {
    // Convert the unstable Windows internal
    // object type index into own stable type id.
    //
    // If the type lookup table fails to initialize,
    // this will return 0. In that case, you should
    // use NtQueryObject for each handle as fallback.
    //
    // :param typeWindex:  Windows internal object type index
    // :return:            Own stable object type id
    DWORD GetObjectTypeIdFromWindex(DWORD typeWindex) {
        //* make sure lookup is initialized (thread-safe)
        std::call_once(ObjectTypeLookupOnceFlag, []() {
            NTSTATUS status = FillObjectTypeLookupTable();
            ObjectTypeLookupInitStatus = status;
            if (status != STATUS_SUCCESS) {
                printf("[FATAL] Failed to initialize type lookup, NTSTATUS 0x%X\n", status);
            }
        });
        
        auto it = ObjectTypeLookupTable.find(typeWindex);
        if (it != ObjectTypeLookupTable.end()) {
            return it->second;
        }
        return 0;
    }
    
    // Initialize the cached object type lookup.
    NTSTATUS FillObjectTypeLookupTable() {
        POBJECT_TYPES_INFORMATION typesInfo = NULL;
        NTSTATUS status = GetHandleTypesInformation(&typesInfo);
        if (status != STATUS_SUCCESS) {
            return status;
        }

        POBJECT_TYPE_INFORMATION2 type = FIRST_OBJECT_TYPE(typesInfo);
        for (ULONG i = 0; i < typesInfo->NumberOfTypes; i++) {
            DWORD id = GetObjectTypeIdFromName(type->TypeName);
            if (id != OBJ_TYPE_UNKNOWN) {
                ObjectTypeLookupTable.insert({type->TypeIndex, id});
            }

            type = NEXT_OBJECT_TYPE(type);
        }

        HeapFree(GetProcessHeap(), 0, typesInfo);
        return status;
    }

    // Wrapper to call NtQueryObject with ObjectTypesInformation.
    // Caller must free the result with HeapFree if status is STATUS_SUCCESS.
    NTSTATUS GetHandleTypesInformation(POBJECT_TYPES_INFORMATION* out) {
        if (NtQueryObject == NULL) {
            NtQueryObject = (NQO)GetProcAddress(GetModuleHandle("ntdll.dll"), "NtQueryObject");
        }
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
            if (bufferSize > MAX_BUFFER_SIZE) {
                return STATUS_BUFFER_OVERFLOW;
            }

            buffer = HeapAlloc(
                GetProcessHeap(), HEAP_ZERO_MEMORY, bufferSize);
        }

        if (status != STATUS_SUCCESS) {
            HeapFree(GetProcessHeap(), 0, buffer);
            return status;
        }

        *out = (POBJECT_TYPES_INFORMATION)buffer;
        return status;
    }
}