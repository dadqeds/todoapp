package tasks_postgres_repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
	core_postgres_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool"
)

func (r *TasksRepository) PatchTask(
	ctx context.Context,
	id int,
	task domain.Task,
) (domain.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	UPDATE todoapp.tasks
	SET
		title=$1,
		description=$2,
		completed=$3,
		completed_at=$4,
		list_id=$5,
		due_at=$6,
		due_all_day=$7,
		repeat_rule=$8,
		version=version + 1
	WHERE id=$9 AND version=$10 

	RETURNING ` + taskColumns + `;
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		task.Title,
		task.Description,
		task.Completed,
		task.CompletedAt,
		task.ListID,
		task.DueAt,
		task.DueAllDay,
		repeatRuleToModel(task.Repeat),
		id,
		task.Version,
	)

	taskModel, err := scanTaskModel(row)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain.Task{}, fmt.Errorf(
				"task with id='%d' concurrently accessed: %w",
				id,
				core_errors.ErrConflict,
			)
		}
		return domain.Task{}, fmt.Errorf("scan error: %w", err)
	}

	taskDomain := taskDomainFromModel(taskModel)

	return taskDomain, nil
}
