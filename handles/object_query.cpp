#include <unordered_map>
#include <mutex>
#include "handles.h"

//?=====================================================================+
//?   This is a helper for querying object info only once per object.   |
//?=====================================================================+

static std::unordered_map<LPVOID, ObjectInfo*> ObjectInfoTracker;
static std::mutex ObjectInfoTrackerLock;

extern "C" {
    // Reset the object info tracker state and free all entries.
    // This should be called every time you refresh handle data.
    void ResetObjectInfoTracker() {
        std::lock_guard<std::mutex> lock(ObjectInfoTrackerLock);
        for (auto& [address, entry] : ObjectInfoTracker) {
            FreeObjectInfoEntry(entry);
        }
        ObjectInfoTracker = {};
    }

    // Add an object info entry to the tracker.
    // THE ENTRY MUST BE MALLOC'D!! DO NOT FREE IT!
    // The info tracker owns the entry and will free it.
    // If an entry already exists, this will replace it.
    ObjectInfo* AddObjectInfoEntry(ObjectInfo* entry) {
        if (entry == NULL || entry->Address == NULL) {
            return NULL;
        }

        std::lock_guard<std::mutex> lock(ObjectInfoTrackerLock);

        auto [it, inserted] = ObjectInfoTracker.emplace(entry->Address, entry);
        if (inserted) {
            return entry;
        }

        // another worker won the data race
        FreeObjectInfoEntry(entry);
        return it->second;
    }

    // Lookup an object in the tracker.
    // Return value is NULL if there is no entry.
    ObjectInfo* LookupObjectInfo(LPVOID address) {
        std::lock_guard<std::mutex> lock(ObjectInfoTrackerLock);
        return LookupObjectInfoNoLock(address);
    }

    // Lookup an object in the tracker.
    // Return value is NULL if there is no entry.
    ObjectInfo* LookupObjectInfoNoLock(LPVOID address) {
        auto it = ObjectInfoTracker.find(address);
        if (it == ObjectInfoTracker.end()) return NULL;

        ObjectInfo* entry = it->second;
        if (entry != NULL && entry->Parameters == NULL) {
            return NULL;
        }
        return entry;
    }
    
    void FreeObjectInfoEntry(ObjectInfo* entry) {
        if (entry == NULL) return;

        if (entry->Parameters != NULL) {
            free(entry->Parameters);
        }
        free(entry);
    }
}