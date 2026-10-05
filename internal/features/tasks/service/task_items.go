package tasks_service

import (
	"context"
	"fmt"

	"github.com/dadqeds/todoapp/internal/core/domain"
)

// Пункты чеклиста доступны тем же, кому доступна их задача.

func (s *TasksService) GetTaskItems(ctx context.Context, taskID int) ([]domain.TaskItem, error) {
	if _, err := s.getAccessibleTask(ctx, taskID); err != nil {
		return nil, err
	}

	items, err := s.tasksRepository.GetTaskItems(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task items from repository: %w", err)
	}

	return items, nil
}

func (s *TasksService) CreateTaskItem(ctx context.Context, taskID int, title string) (domain.TaskItem, error) {
	if _, err := s.getAccessibleTask(ctx, taskID); err != nil {
		return domain.TaskItem{}, err
	}

	item := domain.NewTaskItemUninitialized(taskID, title)
	if err := item.Validate(); err != nil {
		return domain.TaskItem{}, fmt.Errorf("validate task item: %w", err)
	}

	item, err := s.tasksRepository.CreateTaskItem(ctx, item)
	if err != nil {
		return domain.TaskItem{}, fmt.Errorf("create task item: %w", err)
	}

	return item, nil
}

func (s *TasksService) PatchTaskItem(ctx context.Context, taskID int, itemID int, patch domain.TaskItemPatch) (domain.TaskItem, error) {
	if _, err := s.getAccessibleTask(ctx, taskID); err != nil {
		return domain.TaskItem{}, err
	}

	item, err := s.tasksRepository.GetTaskItem(ctx, taskID, itemID)
	if err != nil {
		return domain.TaskItem{}, fmt.Errorf("get task item: %w", err)
	}

	if err := item.ApplyPatch(patch); err != nil {
		return domain.TaskItem{}, fmt.Errorf("apply task item patch: %w", err)
	}

	item, err = s.tasksRepository.PatchTaskItem(ctx, item)
	if err != nil {
		return domain.TaskItem{}, fmt.Errorf("patch task item: %w", err)
	}

	return item, nil
}

func (s *TasksService) DeleteTaskItem(ctx context.Context, taskID int, itemID int) error {
	if _, err := s.getAccessibleTask(ctx, taskID); err != nil {
		return err
	}

	if err := s.tasksRepository.DeleteTaskItem(ctx, taskID, itemID); err != nil {
		return fmt.Errorf("delete task item: %w", err)
	}

	return nil
}
