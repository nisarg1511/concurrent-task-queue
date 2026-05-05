package worker

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	taskdb "github.com/nisarg1511/concurrent-task-queue/db"
	"github.com/nisarg1511/concurrent-task-queue/task"
)

type Handler func(context.Context, string) error

type Registry map[string]Handler

func (r Registry) Execute(ctx context.Context, t task.Task) error {
	handler, ok := r[t.Type]
	if !ok {
		return fmt.Errorf("no handler registered for task type %q", t.Type)
	}

	return handler(ctx, t.Payload)
}

func StartPersistentWorkers(ctx context.Context, n int, store *sql.DB, handlers Registry, wg *sync.WaitGroup) {
	worker := func(workerID int) {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			claimedTask, err := taskdb.ClaimTask(ctx, store)
			if err != nil {
				log.Printf("worker %d could not claim task: %v", workerID, err)
				time.Sleep(250 * time.Millisecond)
				continue
			}
			if claimedTask == nil {
				return
			}

			log.Printf(
				"worker %d executing task %d type=%s (attempt %d/%d)",
				workerID,
				claimedTask.Id,
				claimedTask.Type,
				claimedTask.RetryCount,
				claimedTask.MaxRetries,
			)
			if err := handlers.Execute(ctx, *claimedTask); err != nil {
				log.Printf("worker %d failed task %d: %v", workerID, claimedTask.Id, err)
				if markErr := taskdb.MarkFailed(store, claimedTask.Id, err); markErr != nil {
					log.Printf("worker %d could not persist failure for task %d: %v", workerID, claimedTask.Id, markErr)
				}
				continue
			}

			if err := taskdb.MarkCompleted(store, claimedTask.Id); err != nil {
				log.Printf("worker %d could not persist completion for task %d: %v", workerID, claimedTask.Id, err)
				continue
			}
			log.Printf("worker %d completed task %d", workerID, claimedTask.Id)
		}
	}

	for i := 1; i <= n; i++ {
		wg.Add(1)
		go worker(i)
	}
}

func StartWorkers(n int, done <-chan interface{}, tasks <-chan task.Task, wg *sync.WaitGroup) {
	worker := func(workerID int) {
		defer wg.Done()

		for {
			select {
			case <-done:
				return
			case t, ok := <-tasks:
				if !ok {
					return
				}
				log.Printf("worker %d executing task %d", workerID, t.Id)
				if err := t.Execute(t.Payload); err != nil {
					log.Printf("worker %d failed task %d: %v", workerID, t.Id, err)
					//Later rescheduling the task
				}
			}
		}
	}

	for i := 1; i <= n; i++ {
		wg.Add(1)
		go worker(i)
	}
}
