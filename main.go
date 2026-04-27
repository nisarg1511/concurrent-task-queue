package main

import (
	"fmt"
	"sync"

	ctq "github.com/nisarg1511/concurrent-task-queue/queue"
	"github.com/nisarg1511/concurrent-task-queue/task"
)

func main() {
	var wg sync.WaitGroup
	noWorkers := 10
	worker := func(done <-chan interface{}, queue ctq.TaskQueue, wg *sync.WaitGroup, workerId int) {
		go func() {
			defer wg.Done()
			for {
				select {
				case t := <-queue.Get():
					fmt.Println("Worker with ID:", workerId, ". Executing task with ID", t.Id)
				case <-done:
					return
				}
			}
		}()
	}
	done := make(chan interface{})
	defer close(done)
	wg.Add(noWorkers)
	queue := ctq.GetTaskQueue(100)
	for i := 0; i < 100; i++ {
		queue.Add(task.Task{
			Id:      i + 1,
			Status:  task.Pending,
			Execute: func() error { return nil },
			Payload: struct{}{},
		})
	}
	for i := 0; i < noWorkers; i++ {
		go worker(done, queue, &wg, i+1)
	}
	wg.Wait()
}
