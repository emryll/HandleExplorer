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
