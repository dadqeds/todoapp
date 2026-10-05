package tasks_postgres_repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

func (r *TasksRepository) GetTasks(
	ctx context.Context,
	filter domain.TaskFilter,
	limit int,
	offset int,
) ([]domain.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	args := []any{limit, offset}
	conditions := []string{}

	if filter.AuthorUserID != nil {
		args = append(args, *filter.AuthorUserID)
		conditions = append(conditions, fmt.Sprintf("author_user_id=$%d", len(args)))
	}

	if filter.AccessibleToUserID != nil {
		args = append(args, *filter.AccessibleToUserID)
		n := len(args)
		conditions = append(conditions, fmt.Sprintf(
			"list_id IN (SELECT id FROM todoapp.lists WHERE owner_user_id=$%d UNION SELECT list_id FROM todoapp.list_members WHERE user_id=$%d)",
			n, n,
		))
	}

	if filter.ListID != nil {
		args = append(args, *filter.ListID)
		conditions = append(conditions, fmt.Sprintf("list_id=$%d", len(args)))
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Сначала невыполненные, среди них — по ближайшему сроку, без срока в конце.
	query := `
	SELECT ` + taskColumns + `
	FROM todoapp.tasks
	` + where + `
	ORDER BY completed ASC, due_at ASC NULLS LAST, id DESC
	LIMIT $1
	OFFSET $2;
	`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select tasks: %w", err)
	}
	defer rows.Close()

	var taskModels []TaskModel

	for rows.Next() {
		taskModel, err := scanTaskModel(rows)
		if err != nil {
			return nil, fmt.Errorf("scan tasks: %w", err)
		}

		taskModels = append(taskModels, taskModel)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}

	return taskDomainsFromModels(taskModels), nil
}
