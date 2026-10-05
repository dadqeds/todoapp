package tasks_service

import (
	"context"
	"fmt"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
)

// canUseList: пользователь владеет списком или участвует в нём.
func (s *TasksService) canUseList(ctx context.Context, list domain.List, userID int) (bool, error) {
	if list.OwnerUserID == userID {
		return true, nil
	}

	member, err := s.listsRepository.IsListMember(ctx, list.ID, userID)
	if err != nil {
		return false, fmt.Errorf("check list member: %w", err)
	}

	return member, nil
}

// getAccessibleTask возвращает задачу, если текущий пользователь владеет её
// списком или участвует в нём, или он администратор. Авторство само по себе
// доступа не даёт: вышедший из общего списка теряет доступ и к своим задачам
// в нём. Чужие задачи выглядят как несуществующие.
func (s *TasksService) getAccessibleTask(ctx context.Context, id int) (domain.Task, error) {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return domain.Task{}, err
	}

	task, err := s.tasksRepository.GetTask(ctx, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("get task from repository: %w", err)
	}

	if actor.IsAdmin {
		return task, nil
	}

	list, err := s.listsRepository.GetList(ctx, task.ListID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("get task list: %w", err)
	}

	ok, err := s.canUseList(ctx, list, actor.User.ID)
	if err != nil {
		return domain.Task{}, err
	}
	if !ok {
		return domain.Task{}, fmt.Errorf("task with id='%d': %w", id, core_errors.ErrNotFound)
	}

	return task, nil
}

// resolveTaskList возвращает id списка для задачи автора authorUserID:
// 0 — список по умолчанию автора, иначе автор должен владеть списком или
// участвовать в нём.
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

	ok, err := s.canUseList(ctx, list, authorUserID)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("list with id='%d': %w", listID, core_errors.ErrNotFound)
	}

	return list.ID, nil
}
