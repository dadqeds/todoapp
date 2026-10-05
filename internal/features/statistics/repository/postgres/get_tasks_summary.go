package statistics_postgres_repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
	statistics_service "github.com/dadqeds/todoapp/internal/features/statistics/service"
)

// GetTasksSummary считает агрегаты на стороне БД, не загружая задачи в память.
func (r *StatisticsRepository) GetTasksSummary(
	ctx context.Context,
	userID *int,
	from *time.Time,
	to *time.Time,
) (statistics_service.TasksSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	var queryBuilder strings.Builder

	queryBuilder.WriteString(`
	SELECT
		COUNT(*),
		COUNT(*) FILTER (WHERE completed),
		EXTRACT(EPOCH FROM AVG(completed_at - created_at) FILTER (WHERE completed))::float8,
		COUNT(*) FILTER (WHERE completed AND due_at IS NOT NULL),
		COUNT(*) FILTER (WHERE completed AND due_at IS NOT NULL AND completed_at <= due_at)
	FROM todoapp.tasks t
	`)

	args := []any{}
	conditions := []string{}

	if userID != nil {
		args = append(args, *userID)
		conditions = append(conditions, fmt.Sprintf("t.author_user_id=$%d", len(args)))
	}

	if from != nil {
		args = append(args, *from)
		conditions = append(conditions, fmt.Sprintf("t.created_at>=$%d", len(args)))
	}

	if to != nil {
		args = append(args, *to)
		conditions = append(conditions, fmt.Sprintf("t.created_at<$%d", len(args)))
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	queryBuilder.WriteString(where)

	var (
		summary            statistics_service.TasksSummary
		avgCompletionInSec *float64
	)

	err := r.pool.QueryRow(ctx, queryBuilder.String(), args...).Scan(
		&summary.Created,
		&summary.Completed,
		&avgCompletionInSec,
		&summary.CompletedWithDue,
		&summary.CompletedOnTime,
	)
	if err != nil {
		return statistics_service.TasksSummary{}, fmt.Errorf("select tasks summary: %w", err)
	}

	if avgCompletionInSec != nil {
		avg := time.Duration(*avgCompletionInSec * float64(time.Second))
		summary.AverageCompletionTime = &avg
	}

	lists, err := r.getListsStatistics(ctx, where, args)
	if err != nil {
		return statistics_service.TasksSummary{}, err
	}
	summary.Lists = lists

	return summary, nil
}

func (r *StatisticsRepository) getListsStatistics(ctx context.Context, where string, args []any) ([]domain.ListStatistics, error) {
	query := `
	SELECT l.id, l.title, l.color, COUNT(t.id), COUNT(t.id) FILTER (WHERE t.completed)
	FROM todoapp.tasks t
	JOIN todoapp.lists l ON l.id = t.list_id
	` + where + `
	GROUP BY l.id
	ORDER BY COUNT(t.id) DESC, l.id ASC;
	`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select lists statistics: %w", err)
	}
	defer rows.Close()

	var lists []domain.ListStatistics
	for rows.Next() {
		var l domain.ListStatistics
		if err := rows.Scan(&l.ListID, &l.Title, &l.Color, &l.TasksCreated, &l.TasksCompleted); err != nil {
			return nil, fmt.Errorf("scan lists statistics: %w", err)
		}
		lists = append(lists, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}

	return lists, nil
}
