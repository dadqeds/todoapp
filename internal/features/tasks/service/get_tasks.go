package tasks_service

import (
	"context"
	"fmt"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
)

func (s *TasksService) GetTasks(
	ctx context.Context,
	userID *int,
	limit *int,
	offset *int,
) ([]domain.Task, error) {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Обычный пользователь видит только свои задачи, фильтр user_id игнорируется.
	if !actor.IsAdmin {
		userID = &actor.User.ID
	}

	l, o, err := domain.NormalizePagination(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("normalize pagination: %w", err)
	}

	tasks, err := s.tasksRepository.GetTasks(ctx, userID, l, o)
	if err != nil {
		return nil, fmt.Errorf(
			"get tasks from repository: %w",
			err,
		)
	}
	return tasks, nil
}
