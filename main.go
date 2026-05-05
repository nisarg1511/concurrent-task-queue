package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	taskdb "github.com/nisarg1511/concurrent-task-queue/db"
	"github.com/nisarg1511/concurrent-task-queue/task"
	"github.com/nisarg1511/concurrent-task-queue/testtasks"
	"github.com/nisarg1511/concurrent-task-queue/worker"
)

func main() {
	var wg sync.WaitGroup
	numOfWorkers := flag.Int("workers", 10, "number of concurrent workers")
	numTasks := flag.Int("tasks", 0, "number of sample tasks to enqueue before processing")
	maxRetries := flag.Int("max-retries", 3, "maximum attempts per task")
	taskType := flag.String("task-type", "print", "task handler type to enqueue for sample tasks")
	dbPath := flag.String("db", "tasks.db", "sqlite database path")
	flag.Parse()

	if value := os.Getenv("TASK_DB_PATH"); value != "" {
		*dbPath = value
	}
	if *numOfWorkers < 1 {
		log.Fatal("workers must be at least 1")
	}
	if *numTasks < 0 {
		log.Fatal("tasks cannot be negative")
	}
	if *maxRetries < 1 {
		log.Fatal("max-retries must be at least 1")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := taskdb.InitDB("sqlite3", *dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	recovered, err := taskdb.RecoverRunningTasks(store)
	if err != nil {
		log.Fatal(err)
	}
	if recovered > 0 {
		log.Printf("recovered %d task(s) that were running during the previous shutdown", recovered)
	}

	handlers := worker.Registry{
		"print": testtasks.PrintPayload,
	}
	if *numTasks > 0 {
		if _, ok := handlers[*taskType]; !ok {
			log.Fatalf("unknown task type %q", *taskType)
		}
	}

	producer := func(done <-chan struct{}, count int, taskType string) <-chan task.Task {
		tasks := make(chan task.Task)
		go func() {
			defer close(tasks)
			for i := 0; i < count; i++ {
				select {
				case tasks <- task.Task{
					Type:       taskType,
					Status:     task.Pending,
					Payload:    strconv.Itoa(i),
					MaxRetries: *maxRetries,
				}:
				case <-done:
					return
				}
			}
		}()
		return tasks
	}

	done := make(chan struct{})
	defer close(done)

	for t := range producer(done, *numTasks, *taskType) {
		id, err := taskdb.InsertTask(store, t.Type, t.Payload, t.MaxRetries)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("queued task %d type=%s payload=%q", id, t.Type, t.Payload)
	}

	worker.StartPersistentWorkers(ctx, *numOfWorkers, store, handlers, &wg)
	wg.Wait()

	printStatusCounts(store)
}

func printStatusCounts(store *sql.DB) {
	statuses := []task.TaskStatus{
		task.Pending,
		task.Running,
		task.Completed,
		task.Failed,
	}
	for _, status := range statuses {
		count, err := taskdb.CountByStatus(store, status)
		if err != nil {
			log.Printf("could not count %s tasks: %v", status, err)
			continue
		}
		log.Printf("%s tasks: %d", status, count)
	}
}
