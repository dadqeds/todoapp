package tasks_service

import (
	"context"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

func (s *TasksService) GetTasks(
	ctx context.Context,
	userID *int,
	limit *int,
	offset *int,
) ([]domain.Task, error) {
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
