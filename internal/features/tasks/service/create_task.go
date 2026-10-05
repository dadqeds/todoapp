package tasks_service

import (
	"context"
	"fmt"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
)

func (s *TasksService) CreateTask(
	ctx context.Context,
	task domain.Task,
) (domain.Task, error) {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return domain.Task{}, err
	}

	// Обычный пользователь создаёт задачи только себе. Администратор может
	// указать автора, иначе автором становится он сам.
	if !actor.IsAdmin || task.AuthorUserID == 0 {
		task.AuthorUserID = actor.User.ID
	}

	task.ListID, err = s.resolveTaskList(ctx, task.ListID, task.AuthorUserID)
	if err != nil {
		return domain.Task{}, err
	}

	anchorRepeat(&task, actor.User.Location())

	if err := task.Validate(); err != nil {
		return domain.Task{}, fmt.Errorf(
			"validate task domain: %w",
			err,
		)
	}

	task, err = s.tasksRepository.CreateTask(ctx, task)
	if err != nil {
		return domain.Task{}, fmt.Errorf(
			"create task: %w",
			err,
		)
	}

	return task, nil
}
