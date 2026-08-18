#include <windows.h>
#include <stdarg.h>
#include <stdio.h>
#include "handles.h"

BYTE* BuildParameter(size_t* totalSize, DWORD type, const char* name, ...) {
    va_list args;
    va_start(args, name);

    // Determine value pointer and size based on type
    const void* value = NULL;
    DWORD valueSize = 0;

    switch (type) {
        case PARAMETER_UINT32: {
            // va_arg promotes to int, so we capture then take address
            static DWORD tmp; // static so pointer remains valid briefly
            tmp = (DWORD)va_arg(args, unsigned int);
            value = &tmp;
            valueSize = sizeof(DWORD);
            break;
        }
        case PARAMETER_UINT64: {
            static UINT64 tmp;
            tmp = va_arg(args, UINT64);
            value = &tmp;
            valueSize = sizeof(UINT64);
            break;
        }
        case PARAMETER_POINTER: {
            static void* tmp;
            tmp = va_arg(args, void*);
            value = &tmp;
            valueSize = sizeof(void*);
            break;
        }
        case PARAMETER_BOOLEAN: {
            static BYTE tmp;
            tmp = (BYTE)va_arg(args, int);
            value = &tmp;
            valueSize = sizeof(BYTE);
            break;
        }
        case PARAMETER_ANSISTRING: {
            value = va_arg(args, const char*);
            valueSize = (DWORD)(strlen((const char*)value) + 1); // include null
            break;
        }
        case PARAMETER_BYTES: {
            value = va_arg(args, const void*);
            valueSize = va_arg(args, DWORD); // bytes type needs explicit size
            break;
        }
        default:
            va_end(args);
            return NULL;
    }
    va_end(args);

    // For fixed-size types, size is inferrable so pass 0; for bytes pass valueSize
    DWORD headerSize = (type == PARAMETER_BYTES) ? valueSize : 0;

    size_t headerLen = 0;
    BYTE* header = CreateParameterHeader((char*)name, headerSize, type, &headerLen);
    if (!header) return NULL;

    // Layout: [header bytes (includes null terminator)] [raw value]
    *totalSize = headerLen + valueSize;
    BYTE* buf = (BYTE*)malloc(*totalSize);
    if (!buf) { free(header); return NULL; }

    memcpy(buf, header, headerLen);               // copy header (null terminator included)
    memcpy(buf + headerLen, value, valueSize);     // append raw value

    free(header);
    return buf;
}

// Create string header for parameter
BYTE* CreateParameterHeader(char* name, DWORD size, DWORD type, size_t* dataSize) {
    if (size > 50000) return NULL;
    // data size will also work as a counter for how much memory to allocate
    (*dataSize) = strlen(name) + 2; // +2 is for the symbol and the null-terminator at the end.

    size_t sizeStrLen;
    if (size > 0) {    
        // get the amount of characters it takes to represent size
        sizeStrLen = snprintf(NULL, 0, "%d", size);
        (*dataSize) += 1; // for the "/"
    } else {
        sizeStrLen = 0;
    }
    (*dataSize) += sizeStrLen;

    char symbol;
    switch (type) {
        case PARAMETER_ANSISTRING:
            symbol = 's'; break;
        case PARAMETER_UINT32:
            symbol = 'd'; break;
        case PARAMETER_UINT64:
            symbol = 'q'; break;
        case PARAMETER_POINTER:
            symbol = 'p'; break;
        case PARAMETER_BOOLEAN:
            symbol = 'b'; break;
        case PARAMETER_BYTES: 
            symbol = 'x'; break;
        default: return NULL;
    }
    
    BYTE* packet = (BYTE*)malloc((*dataSize));
    if (packet == NULL) return NULL;

    if (sizeStrLen == 0) {
        snprintf((char*)packet, (*dataSize), "%c%s", symbol, name);
    } else {
        snprintf((char*)packet, (*dataSize), "%c%s/%d", symbol, name, size);
    }
    //printf("\n[debug] parameter packet: %s\n", (char*)packet);
    return packet;
}
