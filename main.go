package main

import (
	"fmt"
	"sync"
	"time"

	ctq "github.com/nisarg1511/concurrent-task-queue/queue"
	"github.com/nisarg1511/concurrent-task-queue/task"
)

func main() {
	var wg sync.WaitGroup
	numOfWorkers := 10
	producer := func(done chan interface{}, numTasks int) <-chan task.Task {
		tasks := make(chan task.Task)
		go func() {
			defer close(tasks)
			for i := 0; i < numTasks; i++ {
				select {
				case tasks <- task.Task{
					Id:      i,
					Status:  task.Pending,
					Payload: nil,
					Execute: nil,
				}:
				case <-done:
					return
				}

			}
		}()
		return tasks
	}
	worker := func(done <-chan interface{}, queue *ctq.TaskQueue, wg *sync.WaitGroup, workerId int) {
		defer wg.Done()
		defer fmt.Printf("Closing worker with ID:%d!\n", workerId)
		for {
			select {
			case <-done:
				return
			case task, ok := <-queue.Tasks():
				if !ok {
					return
				}
				fmt.Printf("Executing task!%d\n", task.Id)
				time.Sleep(1 * time.Second)
				fmt.Printf("Done with the task!\n")
			}
		}
	}

	queue := ctq.New(100)
	done := make(chan interface{})
	defer close(done)

	wg.Add(numOfWorkers)

	for i := 1; i <= numOfWorkers; i++ {
		go worker(done, queue, &wg, i)
	}
	for task := range producer(done, 100) {
		queue.Submit(task)
	}
	queue.Close()
	wg.Wait()
}
