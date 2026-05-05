package worker

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	taskdb "github.com/nisarg1511/concurrent-task-queue/db"
	"github.com/nisarg1511/concurrent-task-queue/task"
)

func TestPersistentWorkerRetriesAndCompletesTask(t *testing.T) {
	store, err := taskdb.InitDB("sqlite3", filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer store.Close()

	if _, err := taskdb.InsertTask(store, "flaky", "payload", 2); err != nil {
		t.Fatalf("insert task: %v", err)
	}

	var attempts atomic.Int32
	handlers := Registry{
		"flaky": func(context.Context, string) error {
			if attempts.Add(1) == 1 {
				return errors.New("temporary failure")
			}
			return nil
		},
	}

	var wg sync.WaitGroup
	StartPersistentWorkers(context.Background(), 1, store, handlers, &wg)
	wg.Wait()

	if attempts.Load() != 2 {
		t.Fatalf("expected 2 handler attempts, got %d", attempts.Load())
	}

	completed, err := taskdb.CountByStatus(store, task.Completed)
	if err != nil {
		t.Fatalf("count completed: %v", err)
	}
	if completed != 1 {
		t.Fatalf("expected task to complete after retry, got %d completed tasks", completed)
	}
}

func TestPersistentWorkerMarksUnknownTaskTypeFailed(t *testing.T) {
	store, err := taskdb.InitDB("sqlite3", filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer store.Close()

	if _, err := taskdb.InsertTask(store, "missing", "payload", 1); err != nil {
		t.Fatalf("insert task: %v", err)
	}

	var wg sync.WaitGroup
	StartPersistentWorkers(context.Background(), 1, store, Registry{}, &wg)
	wg.Wait()

	failed, err := taskdb.CountByStatus(store, task.Failed)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if failed != 1 {
		t.Fatalf("expected unknown task type to fail, got %d failed tasks", failed)
	}
}
