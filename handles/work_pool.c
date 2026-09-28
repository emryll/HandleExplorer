#include <windows.h>
#include <stdio.h>
#include "handles.h"

//?================================================================+
//?    When querying object parameters, there may be a delay.      |
//?   This is the case especially with file objects, where the     |
//?   name query may hang indefinitely, and the timeouts add up.   |
//?     Because of this, a simple thread worker pool is used.      |
//?================================================================+

//*==========================[ HP Worker ]========================

// Complete a handle parameter task.
// Once the task is done, the handle associated
// with the task is closed, and the task freed.
void ExecuteHParamTask(HPARAM_TASK* task) {
    if (task == NULL) return;

    if (task->entry != NULL) {
        task->entry->Params = GetHandleParameters(
            task->hObject,
            task->entry->Type,
            &task->entry->paramsSize
        );
    }

    if (task->hObject != NULL && task->hObject != INVALID_HANDLE_VALUE) {
        CloseHandle(task->hObject);
    }
    free(task);
}

// HP_WORK_POOL worker routine
DWORD WINAPI HParamWorker(HP_TASK_QUEUE* queue) {
    HPARAM_TASK* task = NULL;

    // This will return FALSE once all work is done.
    while (GetTaskFromQueue(queue, &task)) {
        ExecuteHParamTask(task);
    }

    return 0;
}

//*=========================[ Task Queue ]=========================

// Create a handle parameter query task and add it to the queue.
// If task creation and queueing succeeds, the return value is TRUE.
BOOL CreateHpTask(HP_TASK_QUEUE* q, HANDLE hObject, HANDLE_ENTRY* entry) {
    HPARAM_TASK* task = (HPARAM_TASK*)malloc(sizeof(HPARAM_TASK));
    if (!task) return FALSE;

    task->hObject    = hObject;
    task->entry      = entry;

    return AddTaskToQueue(q, task);
}

// Add a task to the task queue. Should always succeed,
// aside from OOM. The queue never rejects tasks itself.
// If the operation fails, the return value is FALSE.
BOOL AddTaskToQueue(HP_TASK_QUEUE* q, HPARAM_TASK* task) {
    HP_QUEUE_NODE* node = (HP_QUEUE_NODE*)malloc(sizeof(HP_QUEUE_NODE));
    if (node == NULL) return FALSE;

    node->Task = task;
    node->Next = NULL;

    EnterCriticalSection(&q->Lock);
    if (q->Tail) {
        q->Tail->Next = node;
    } else {
        q->Head = node;
    }
    q->Tail = node;
    InterlockedIncrement(&q->Count);
    LeaveCriticalSection(&q->Lock);

    // wake exactly one idle worker
    WakeConditionVariable(&q->NotEmpty);
    return TRUE;
}

// Get a task from the work queue. Thread-safe.
// Blocks until a task is available or shutdown is signaled.
// If the operation fails, the return value is FALSE.
BOOL GetTaskFromQueue(HP_TASK_QUEUE* q, HPARAM_TASK** out) {
    EnterCriticalSection(&q->Lock);
    while (q->Head == NULL && !q->ShuttingDown) {
        SleepConditionVariableCS(&q->NotEmpty, &q->Lock, INFINITE);
    }

    if (q->Head == NULL) { // shutting down and drained
        LeaveCriticalSection(&q->Lock);
        return FALSE;
    }

    HP_QUEUE_NODE* node = q->Head;
    q->Head = node->Next;
    if (!q->Head) q->Tail = NULL;

    InterlockedDecrement(&q->Count);
    LeaveCriticalSection(&q->Lock);

    *out = node->Task;
    free(node);
    return TRUE;
}

void InitializeHpQueue(HP_TASK_QUEUE* q) {
    InitializeCriticalSection(&q->Lock);
    InitializeConditionVariable(&q->NotEmpty);

    q->ShuttingDown = FALSE;
    q->Head = q->Tail = NULL;
    q->Count = 0;
}

void QueueShutdown(HP_TASK_QUEUE* q) {
    EnterCriticalSection(&q->Lock);
    q->ShuttingDown = TRUE;
    LeaveCriticalSection(&q->Lock);
    WakeAllConditionVariable(&q->NotEmpty);
}
