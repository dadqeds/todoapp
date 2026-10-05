package tasks_service

import (
	"context"
	"fmt"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
)

func (s *TasksService) GetTasks(
	ctx context.Context,
	filter domain.TaskFilter,
	limit *int,
	offset *int,
) ([]domain.Task, error) {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	// По умолчанию — задачи из своих и общих списков. Обычному пользователю
	// фильтр user_id недоступен; администратор с user_id видит задачи автора.
	if !actor.IsAdmin || filter.AuthorUserID == nil {
		filter.AuthorUserID = nil
		filter.AccessibleToUserID = &actor.User.ID
	}

	l, o, err := domain.NormalizePagination(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("normalize pagination: %w", err)
	}

	tasks, err := s.tasksRepository.GetTasks(ctx, filter, l, o)
	if err != nil {
		return nil, fmt.Errorf(
			"get tasks from repository: %w",
			err,
		)
	}
	return tasks, nil
}
