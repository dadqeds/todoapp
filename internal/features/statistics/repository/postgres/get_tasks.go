package statistics_postgres_repository

import (
	"context"
	"fmt"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

func (r *StatisticsRepository) GetTasks(
	ctx context.Context,
	userID *int,
	from *time.Time,
	to *time.Time,
)([]domain.Task, error){
	ctx, cancel := context.WithTimeout(ctx, r.poll.OpTimeout())
	defer cancel()

	query := `
	SELECT id, version, title, description, completed, created_at, completed_at, authot_user_id
	FROM todoapp.tasks
	`

	args := []any{}
	conditions := []string{}

	if userID != nil{
		conditions = append(conditions,fmt.Sprintf("authot_user_id=$%d", len(args)+1))
		args = append(args, userID)
	}

	if from != nil{
		conditions = append(conditions, fmt.Sprintf("created_at>=$%d", len(args)+1))
		args = append(args, from)
	}

	if to != nil{
		conditions = append(conditions, fmt.Sprintf("created_at<$%d", len(args)+1))
		args = append(args, to)
	}
}