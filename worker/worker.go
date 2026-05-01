package worker

import (
	"fmt"
	"log"
	"sync"

	ctq "github.com/nisarg1511/concurrent-task-queue/queue"
)

func StartWorkers(n int, done <-chan interface{}, queue *ctq.TaskQueue, wg *sync.WaitGroup) {
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
				err := task.Execute(task.Payload)
				if err != nil {
					log.Printf("%v", err)
					//Later rescheduling the task
				}
				fmt.Printf("Done with the task!\n")
			}
		}
	}

	for i := 1; i <= n; i++ {
		go worker(done, queue, wg, i)
	}
}
