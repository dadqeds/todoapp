package tasks_service

import (
	"context"
	"fmt"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

// getAccessibleTask возвращает задачу, только если текущий пользователь
// её автор или администратор. Чужие задачи выглядят как несуществующие.
func (s *TasksService) getAccessibleTask(ctx context.Context, id int) (domain.Task, error) {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return domain.Task{}, err
	}

	task, err := s.tasksRepository.GetTask(ctx, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("get task from repository: %w", err)
	}

	if !actor.CanAccessUser(task.AuthorUserID) {
		return domain.Task{}, fmt.Errorf("task with id='%d': %w", id, core_errors.ErrNotFound)
	}

	return task, nil
}

// resolveTaskList возвращает id списка для задачи автора authorUserID:
// 0 — список по умолчанию автора, иначе список должен принадлежать автору.
func (s *TasksService) resolveTaskList(ctx context.Context, listID int, authorUserID int) (int, error) {
	if listID == 0 {
		list, err := s.listsRepository.GetOrCreateDefaultList(ctx, authorUserID)
		if err != nil {
			return 0, fmt.Errorf("get default list: %w", err)
		}
		return list.ID, nil
	}

	list, err := s.listsRepository.GetList(ctx, listID)
	if err != nil {
		return 0, fmt.Errorf("get list: %w", err)
	}

	if list.OwnerUserID != authorUserID {
		return 0, fmt.Errorf("list with id='%d': %w", listID, core_errors.ErrNotFound)
	}

	return list.ID, nil
}
