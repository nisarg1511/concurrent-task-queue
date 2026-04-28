# Design Document (V1) — Concurrent Task Queue

## Problem Statement

Build a simple concurrent task queue in Go that:

- accepts tasks
- processes them concurrently
- shuts down gracefully

---

## High-Level Design

The system follows a **producer → queue → worker** model:

Producer → TaskQueue → Workers

- Producer generates tasks
- TaskQueue buffers tasks using a channel
- Workers consume tasks concurrently

---

## Core Components

### 1. TaskQueue

- Wraps a buffered channel
- Responsible for:
  - accepting tasks (`Submit`)
  - exposing read-only access (`Tasks()`)
  - closing the channel (`Close`)

---

### 2. Workers

- Run as goroutines
- Continuously read from queue
- Exit when:
  - queue is closed
  - cancellation signal is received

---

### 3. Producer

- Generates tasks
- Sends them to main via channel
- Main submits them to queue

---

## Channel Ownership Model

- The queue **owns the channel**
- Only the queue is responsible for closing it
- External components:
  - can submit tasks via API
  - can read via `<-chan`
  - cannot close or write directly

---

## Lifecycle

1. Workers are started
2. Producer generates tasks
3. Main submits tasks to queue
4. Main closes the queue after submission
5. Workers drain remaining tasks
6. Workers exit gracefully

---

## Problems Encountered

### 1. Deadlock

- Caused by improper goroutine coordination
- Fixed by restructuring worker lifecycle and channel usage

---

### 2. Channel Ownership Confusion

- Initially unclear who should close the channel
- Resolved by defining:
  → “last sender closes the channel”

---

### 3. Over-engineering with Channel Wrappers

- Early design added unnecessary abstraction
- Simplified by using channels directly

---

### 4. Synchronization Confusion

- Misunderstanding between channel sync vs mutex
- Learned:
  - channels handle communication
  - mutex protects shared state

---

## Key Design Decisions

- Use channels as primary communication mechanism
- Use worker pool for controlled concurrency
- Expose read-only channel to enforce safety
- Keep design minimal and idiomatic

---

## Future Improvements

- Retry mechanism
- Task prioritization
- Backpressure strategies
- Context-based cancellation
- Distributed task queue

---

## Key Learnings

- Importance of channel ownership
- Goroutine lifecycle management
- Difference between synchronization and communication
- Designing for clarity over cleverness
