package main

import (
	"log"
	"os"
	"strconv"
	"sync"

	ctq "github.com/nisarg1511/concurrent-task-queue/queue"
	"github.com/nisarg1511/concurrent-task-queue/task"
	"github.com/nisarg1511/concurrent-task-queue/testtasks"
	"github.com/nisarg1511/concurrent-task-queue/worker"
)

func main() {
	var wg sync.WaitGroup
	//Default number of workers.
	numOfWorkers := 10

	if len(os.Args) > 1 {
		n, err := strconv.Atoi(os.Args[1])
		if err != nil {
			log.Fatal("Invalid argument! Number of workers should be a valid integer.")
		}
		numOfWorkers = n
	}

	producer := func(done chan interface{}, numTasks int) <-chan task.Task {
		tasks := make(chan task.Task)
		go func() {
			defer close(tasks)
			for i := 0; i < numTasks; i++ {
				select {
				case tasks <- task.Task{
					Id:      i,
					Status:  task.Pending,
					Payload: i,
					Execute: testtasks.PrintPayload,
				}:
				case <-done:
					return
				}

			}
		}()
		return tasks
	}

	queue := ctq.New(100)
	done := make(chan interface{})
	defer close(done)

	wg.Add(numOfWorkers)
	worker.StartWorkers(numOfWorkers, done, queue, &wg)

	for task := range producer(done, 150) {
		queue.Submit(task)
	}
	queue.Close()
	wg.Wait()
}
