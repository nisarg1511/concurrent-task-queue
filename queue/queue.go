package concurrenttaskqueue

import (
	"github.com/nisarg1511/concurrent-task-queue/task"
)

type TaskQueue struct {
	Tasks chan task.Task
}

func (q *TaskQueue) Add(task task.Task) {
	q.Tasks <- task
}

func (q *TaskQueue) Get() <-chan task.Task {
	return q.Tasks
}

func GetTaskQueue(capacity int) TaskQueue {
	tasks := make(chan task.Task, capacity)
	var queue = TaskQueue{
		Tasks: tasks,
	}
	return queue
}
