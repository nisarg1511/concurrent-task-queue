package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nisarg1511/concurrent-task-queue/task"
)

func InitDB(provider, name string) (*sql.DB, error) {
	store, err := sql.Open(provider, name)
	if err != nil {
		return nil, err
	}

	if _, err := store.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA busy_timeout=5000;
		PRAGMA foreign_keys=ON;
	`); err != nil {
		store.Close()
		return nil, err
	}

	if err := createTable(store); err != nil {
		store.Close()
		return nil, err
	}

	return store, nil
}

func createTable(store *sql.DB) error {
	if _, err := store.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_type TEXT NOT NULL DEFAULT 'print',
			payload TEXT NOT NULL,
			status INTEGER NOT NULL,
			retry_count INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL DEFAULT 3,
			last_error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}

	return ensureTaskTypeColumn(store)
}

func ensureTaskTypeColumn(store *sql.DB) error {
	rows, err := store.Query(`PRAGMA table_info(tasks)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return err
		}
		if name == "task_type" {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = store.Exec(`ALTER TABLE tasks ADD COLUMN task_type TEXT NOT NULL DEFAULT 'print'`)
	return err
}

func InsertTask(store *sql.DB, taskType, payload string, maxRetries int) (int64, error) {
	if strings.TrimSpace(taskType) == "" {
		return 0, fmt.Errorf("task type is required")
	}
	if maxRetries < 1 {
		return 0, fmt.Errorf("max retries must be at least 1")
	}

	result, err := store.Exec(`
		INSERT INTO tasks (task_type, payload, status, retry_count, max_retries)
		VALUES (?, ?, ?, 0, ?)
	`, taskType, payload, task.Pending, maxRetries)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func ClaimTask(ctx context.Context, store *sql.DB) (*task.Task, error) {
	var t task.Task
	err := store.QueryRowContext(ctx, `
		UPDATE tasks
		SET status = ?, retry_count = retry_count + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = (
			SELECT id
			FROM tasks
			WHERE status = ? AND retry_count < max_retries
			ORDER BY created_at, id
			LIMIT 1
		)
		RETURNING id, task_type, payload, status, retry_count, max_retries
	`, task.Running, task.Pending).Scan(
		&t.Id,
		&t.Type,
		&t.Payload,
		&t.Status,
		&t.RetryCount,
		&t.MaxRetries,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func MarkCompleted(store *sql.DB, id int) error {
	_, err := store.Exec(`
		UPDATE tasks
		SET status = ?, last_error = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, task.Completed, id)
	return err
}

func MarkFailed(store *sql.DB, id int, executeErr error) error {
	status := task.Failed
	errText := ""
	if executeErr != nil {
		errText = executeErr.Error()
	}

	_, err := store.Exec(`
		UPDATE tasks
		SET status = CASE
				WHEN retry_count < max_retries THEN ?
				ELSE ?
			END,
			last_error = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, task.Pending, status, errText, id)
	return err
}

func RecoverRunningTasks(store *sql.DB) (int64, error) {
	result, err := store.Exec(`
		UPDATE tasks
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE status = ?
	`, task.Pending, task.Running)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func CountByStatus(store *sql.DB, status task.TaskStatus) (int, error) {
	var count int
	if err := store.QueryRow(`
		SELECT COUNT(*)
		FROM tasks
		WHERE status = ?
	`, status).Scan(&count); err != nil {
		return 0, fmt.Errorf("count tasks by status: %w", err)
	}

	return count, nil
}
