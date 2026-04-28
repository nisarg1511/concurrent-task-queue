# Concurrent Task Queue (Go)

A simple concurrent task queue implemented in Go using goroutines and channels.
This project demonstrates core concurrency patterns like worker pools, channel-based communication, and graceful shutdown.

---

## Features

- Bounded task queue using channels
- Worker pool with configurable concurrency
- Graceful shutdown via channel closure
- Cancellation support using `done` channel
- Clean separation of producer, queue, and workers

---

## Concepts Demonstrated

- Goroutines & scheduling
- Channel ownership and lifecycle
- Fan-out worker pattern
- Select-based cancellation
- Graceful shutdown

---

## Architecture

Producer → TaskQueue → Workers

- **Producer** generates tasks
- **TaskQueue** acts as a bounded buffer
- **Workers** consume tasks concurrently

---

## Running the Project

```bash
go build

./concurrent-task-queue
```

---

## Example Flow

1. Workers are started
2. Tasks are produced
3. Tasks are submitted to queue
4. Queue is closed after submission
5. Workers finish remaining tasks and exit

---

## Future Improvements

- Retry mechanism for failed tasks
- Backpressure handling when queue is full
- Context-based cancellation
- Metrics and observability
- Distributed queue (Redis / Kafka)

---

## Learning Goal

This project is part of a hands-on exploration of Go concurrency, focusing on building intuition through real problems like deadlocks, synchronization, and system design.
