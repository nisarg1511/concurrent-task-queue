package concurrenttaskqueue

import (
	"fmt"
	"sync"

	"github.com/nisarg1511/concurrent-task-queue/task"
)

type TaskQueue struct {
	tasks  chan task.Task
	closed bool
	mu     sync.Mutex
}

func (q *TaskQueue) Submit(task task.Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return fmt.Errorf("queue closed")
	}
	q.tasks <- task
	return nil
}

func New(capacity int) *TaskQueue {
	tasks := make(chan task.Task, capacity)
	var queue = TaskQueue{
		tasks:  tasks,
		closed: false,
	}
	return &queue
}

func (q *TaskQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		close(q.tasks)
		q.closed = true
	}
}

func (q *TaskQueue) Tasks() <-chan task.Task {

	return q.tasks
}
