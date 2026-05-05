package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"

	"github.com/nisarg1511/concurrent-task-queue/task"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	store, err := InitDB("sqlite3", filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	})

	return store
}

func TestInsertClaimAndCompleteTask(t *testing.T) {
	store := openTestDB(t)

	id, err := InsertTask(store, "print", "hello", 3)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	claimed, err := ClaimTask(context.Background(), store)
	if err != nil {
		t.Fatalf("claim task: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected a claimed task")
	}
	if claimed.Id != int(id) || claimed.Type != "print" || claimed.Payload != "hello" {
		t.Fatalf("claimed wrong task: %+v", claimed)
	}
	if claimed.Status != task.Running || claimed.RetryCount != 1 {
		t.Fatalf("expected running task on first attempt, got %+v", claimed)
	}

	if err := MarkCompleted(store, claimed.Id); err != nil {
		t.Fatalf("mark completed: %v", err)
	}

	completed, err := CountByStatus(store, task.Completed)
	if err != nil {
		t.Fatalf("count completed: %v", err)
	}
	if completed != 1 {
		t.Fatalf("expected 1 completed task, got %d", completed)
	}
}

func TestInsertTaskValidatesRetryCount(t *testing.T) {
	store := openTestDB(t)

	if _, err := InsertTask(store, "print", "hello", 0); err == nil {
		t.Fatal("expected max retries validation error")
	}
}

func TestConcurrentClaimDoesNotDuplicateTasks(t *testing.T) {
	store := openTestDB(t)

	const taskCount = 25
	for i := 0; i < taskCount; i++ {
		if _, err := InsertTask(store, "print", "payload", 3); err != nil {
			t.Fatalf("insert task %d: %v", i, err)
		}
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		claimed = make(map[int]bool)
	)

	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			claimedTask, err := ClaimTask(context.Background(), store)
			if err != nil {
				t.Errorf("claim task: %v", err)
				return
			}
			if claimedTask == nil {
				return
			}

			mu.Lock()
			defer mu.Unlock()
			if claimed[claimedTask.Id] {
				t.Errorf("task %d was claimed more than once", claimedTask.Id)
			}
			claimed[claimedTask.Id] = true
		}()
	}
	wg.Wait()

	if len(claimed) != taskCount {
		t.Fatalf("expected %d unique claimed tasks, got %d", taskCount, len(claimed))
	}
}

func TestRecoverRunningTasks(t *testing.T) {
	store := openTestDB(t)

	id, err := InsertTask(store, "print", "recover-me", 3)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}
	if _, err := store.Exec(`UPDATE tasks SET status = ? WHERE id = ?`, task.Running, id); err != nil {
		t.Fatalf("mark task running: %v", err)
	}

	recovered, err := RecoverRunningTasks(store)
	if err != nil {
		t.Fatalf("recover running tasks: %v", err)
	}
	if recovered != 1 {
		t.Fatalf("expected 1 recovered task, got %d", recovered)
	}

	claimed, err := ClaimTask(context.Background(), store)
	if err != nil {
		t.Fatalf("claim recovered task: %v", err)
	}
	if claimed == nil || claimed.Id != int(id) {
		t.Fatalf("expected recovered task %d to be claimable, got %+v", id, claimed)
	}
}
