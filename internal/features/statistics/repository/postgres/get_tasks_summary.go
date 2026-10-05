package statistics_postgres_repository

import (
	"context"
	"fmt"
	"strings"
	"time"

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
		EXTRACT(EPOCH FROM AVG(completed_at - created_at) FILTER (WHERE completed))::float8
	FROM todoapp.tasks
	`)

	args := []any{}
	conditions := []string{}

	if userID != nil {
		args = append(args, *userID)
		conditions = append(conditions, fmt.Sprintf("author_user_id=$%d", len(args)))
	}

	if from != nil {
		args = append(args, *from)
		conditions = append(conditions, fmt.Sprintf("created_at>=$%d", len(args)))
	}

	if to != nil {
		args = append(args, *to)
		conditions = append(conditions, fmt.Sprintf("created_at<$%d", len(args)))
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}

	var (
		summary            statistics_service.TasksSummary
		avgCompletionInSec *float64
	)

	err := r.pool.QueryRow(ctx, queryBuilder.String(), args...).Scan(
		&summary.Created,
		&summary.Completed,
		&avgCompletionInSec,
	)
	if err != nil {
		return statistics_service.TasksSummary{}, fmt.Errorf("select tasks summary: %w", err)
	}

	if avgCompletionInSec != nil {
		avg := time.Duration(*avgCompletionInSec * float64(time.Second))
		summary.AverageCompletionTime = &avg
	}

	return summary, nil
}
