package tasks_postgres_repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
	core_postgres_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool"
)

const taskItemColumns = `id, version, task_id, title, done, position, created_at`

type TaskItemModel struct {
	ID        int
	Version   int
	TaskID    int
	Title     string
	Done      bool
	Position  int
	CreatedAt time.Time
}

func scanTaskItemModel(s scanner) (TaskItemModel, error) {
	var m TaskItemModel
	err := s.Scan(&m.ID, &m.Version, &m.TaskID, &m.Title, &m.Done, &m.Position, &m.CreatedAt)
	return m, err
}

func taskItemDomainFromModel(m TaskItemModel) domain.TaskItem {
	return domain.TaskItem{
		ID:        m.ID,
		Version:   m.Version,
		TaskID:    m.TaskID,
		Title:     m.Title,
		Done:      m.Done,
		Position:  m.Position,
		CreatedAt: m.CreatedAt,
	}
}

// GetTaskItems возвращает пункты задачи по порядку.
func (r *TasksRepository) GetTaskItems(ctx context.Context, taskID int) ([]domain.TaskItem, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	SELECT ` + taskItemColumns + `
	FROM todoapp.task_items
	WHERE task_id=$1
	ORDER BY position, id;
	`

	rows, err := r.pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("select task items: %w", err)
	}
	defer rows.Close()

	items := []domain.TaskItem{}
	for rows.Next() {
		m, err := scanTaskItemModel(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task items: %w", err)
		}
		items = append(items, taskItemDomainFromModel(m))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}

	return items, nil
}

// GetTaskItem возвращает пункт, если он принадлежит задаче taskID.
func (r *TasksRepository) GetTaskItem(ctx context.Context, taskID int, itemID int) (domain.TaskItem, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `SELECT ` + taskItemColumns + ` FROM todoapp.task_items WHERE id=$1 AND task_id=$2;`

	m, err := scanTaskItemModel(r.pool.QueryRow(ctx, query, itemID, taskID))
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain.TaskItem{}, fmt.Errorf("item with id='%d' in task '%d': %w", itemID, taskID, core_errors.ErrNotFound)
		}
		return domain.TaskItem{}, fmt.Errorf("scan error: %w", err)
	}

	return taskItemDomainFromModel(m), nil
}

// CreateTaskItem добавляет пункт в конец чеклиста. Не больше domain.MaxTaskItems
// пунктов: проверка и вставка в одном запросе.
func (r *TasksRepository) CreateTaskItem(ctx context.Context, item domain.TaskItem) (domain.TaskItem, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	INSERT INTO todoapp.task_items (task_id, title, done, position, created_at)
	SELECT $1, $2, $3, COALESCE(MAX(position), 0) + 1, $4
	FROM todoapp.task_items
	WHERE task_id=$1
	HAVING COUNT(*) < $5
	RETURNING ` + taskItemColumns + `;
	`

	m, err := scanTaskItemModel(r.pool.QueryRow(ctx, query, item.TaskID, item.Title, item.Done, item.CreatedAt, domain.MaxTaskItems))
	if err != nil {
		switch {
		case errors.Is(err, core_postgres_pool.ErrNoRows):
			return domain.TaskItem{}, fmt.Errorf("task already has %d items: %w", domain.MaxTaskItems, core_errors.ErrConflict)
		case errors.Is(err, core_postgres_pool.ErrViolatesForeignKey):
			return domain.TaskItem{}, fmt.Errorf("task with id='%d': %w", item.TaskID, core_errors.ErrNotFound)
		}
		return domain.TaskItem{}, fmt.Errorf("scan error: %w", err)
	}

	return taskItemDomainFromModel(m), nil
}

func (r *TasksRepository) PatchTaskItem(ctx context.Context, item domain.TaskItem) (domain.TaskItem, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	UPDATE todoapp.task_items
	SET title=$1, done=$2, version=version + 1
	WHERE id=$3 AND task_id=$4 AND version=$5
	RETURNING ` + taskItemColumns + `;
	`

	m, err := scanTaskItemModel(r.pool.QueryRow(ctx, query, item.Title, item.Done, item.ID, item.TaskID, item.Version))
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain.TaskItem{}, fmt.Errorf("item with id='%d' concurrently accessed: %w", item.ID, core_errors.ErrConflict)
		}
		return domain.TaskItem{}, fmt.Errorf("scan error: %w", err)
	}

	return taskItemDomainFromModel(m), nil
}

func (r *TasksRepository) DeleteTaskItem(ctx context.Context, taskID int, itemID int) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	tag, err := r.pool.Exec(ctx, `DELETE FROM todoapp.task_items WHERE id=$1 AND task_id=$2;`, itemID, taskID)
	if err != nil {
		return fmt.Errorf("delete task item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("item with id='%d' in task '%d': %w", itemID, taskID, core_errors.ErrNotFound)
	}

	return nil
}

// CopyTaskItems переносит пункты в следующий повтор задачи: тексты и порядок
// те же, галочки сняты.
func (r *TasksRepository) CopyTaskItems(ctx context.Context, fromTaskID int, toTaskID int) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	INSERT INTO todoapp.task_items (task_id, title, done, position)
	SELECT $2, title, FALSE, position
	FROM todoapp.task_items
	WHERE task_id=$1;
	`
	if _, err := r.pool.Exec(ctx, query, fromTaskID, toTaskID); err != nil {
		return fmt.Errorf("copy task items: %w", err)
	}

	return nil
}
