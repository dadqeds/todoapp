package tasks_service

import (
	"context"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

func (s *TasksService) GetTask(
	ctx context.Context,
	id int,
) (domain.Task, error) {
	return s.getAccessibleTask(ctx, id)
}
