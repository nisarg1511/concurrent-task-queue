# Persistent Concurrent Task Queue

A crash-safe concurrent task queue written in Go.

The queue stores tasks in SQLite, lets multiple workers execute tasks concurrently,
persists every state transition, retries failed tasks, and recovers tasks that
were in progress when the process stopped.

## Features

- Persistent task storage using SQLite
- Configurable worker pool
- Atomic task claiming across concurrent workers
- Durable task states: `pending`, `running`, `completed`, `failed`
- Retry support with configurable max attempts
- Crash recovery for tasks left in `running`
- Task handler registry keyed by persisted task type
- Context-aware task execution and graceful shutdown
- CLI flags for workers, task count, database path, task type, and retries
- Tests for persistence, concurrent claiming, retry behavior, and recovery

## Architecture

```text
Producer -> SQLite tasks table -> Worker pool -> Task handlers
```

The database is the source of truth. Workers do not take work from an in-memory
queue in the persistent path. Instead, each worker atomically claims one pending
task by updating it to `running`.

Task state transitions:

```text
pending -> running -> completed
pending -> running -> pending  (failed attempt, retries remaining)
pending -> running -> failed   (failed final attempt)
running -> pending             (startup recovery after crash)
```

## Project Structure

```text
.
├── main.go                 # CLI, startup recovery, enqueueing, worker startup
├── db/
│   ├── db.go               # SQLite schema, task persistence, claiming, recovery
│   └── db_test.go          # DB lifecycle and concurrency tests
├── worker/
│   ├── worker.go           # Worker pool and task handler registry
│   └── worker_test.go      # Retry and handler tests
├── task/
│   └── task.go             # Task model and task statuses
├── testtasks/
│   └── testtasks.go        # Sample task handler
└── queue/
    └── queue.go            # Earlier in-memory queue implementation
```

## Requirements

- Go 1.21+
- CGO-enabled build environment for `github.com/mattn/go-sqlite3`
- SQLite CLI is optional, useful only for inspecting `tasks.db`

## Running

Build:

```bash
go build
```

Run workers against the default database:

```bash
./concurrent-task-queue
```

Enqueue and process sample tasks:

```bash
./concurrent-task-queue --workers 5 --tasks 100
```

Use a temporary database:

```bash
TASK_DB_PATH=/tmp/tasks.db ./concurrent-task-queue --workers 2 --tasks 10
```

The `TASK_DB_PATH` environment variable overrides `--db`.

## CLI Flags

```text
--workers      number of concurrent workers, default 10
--tasks        number of sample tasks to enqueue before processing, default 0
--task-type    sample task handler type, default print
--max-retries  maximum attempts per task, default 3
--db           sqlite database path, default tasks.db
```

By default, `--tasks` is `0`. This is intentional: restarting the process should
recover and process existing durable work without accidentally enqueueing
duplicate sample tasks.

## Testing

Run the full test suite:

```bash
go test ./...
```

The tests cover:

- task insertion, claiming, and completion
- concurrent claims without duplicate execution
- recovery of `running` tasks after a simulated crash
- retry and eventual completion
- failure for unknown task types

## Crash Safety

On startup, the queue resets tasks in `running` state back to `pending`.

This handles the case where the process crashed after a worker claimed a task
but before the worker persisted the final result. The task will be executed
again on restart.

This gives **at-least-once execution**. That is the normal tradeoff for a durable
task queue. Real handlers should be idempotent, or they should use external
operation IDs so retrying the same task is safe.

## Adding a Task Type

Create a context-aware handler:

```go
func SendEmail(ctx context.Context, payload string) error {
	// Parse payload and perform work.
	return nil
}
```

Register it in `main.go`:

```go
handlers := worker.Registry{
	"print": testtasks.PrintPayload,
	"email": SendEmail,
}
```

Then enqueue tasks of that type:

```bash
./concurrent-task-queue --tasks 10 --task-type email
```

## Inspecting Tasks

If you have the SQLite CLI installed:

```bash
sqlite3 tasks.db "SELECT id, task_type, payload, status, retry_count, max_retries FROM tasks;"
```

Status values:

```text
0 = pending
1 = running
2 = completed
3 = failed
```

## Notes

The `queue/` package remains in the repository as the earlier in-memory channel
queue implementation. The main executable now uses the SQLite-backed persistent
queue path.

Good next steps for a larger version would be metrics, delayed jobs, scheduled
tasks, priority queues, and a distributed backend such as Redis, PostgreSQL, or
Kafka.
