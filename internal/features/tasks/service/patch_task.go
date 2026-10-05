package tasks_service

import (
	"context"
	"fmt"
	"time"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
)

func (s *TasksService) PatchTask(
	ctx context.Context,
	id int,
	patch domain.TaskPatch,
) (domain.Task, error) {
	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		return domain.Task{}, err
	}

	task, err := s.getAccessibleTask(ctx, id)
	if err != nil {
		return domain.Task{}, err
	}

	if patch.ListID.Set && patch.ListID.Value != nil {
		listID, err := s.resolveTaskList(ctx, *patch.ListID.Value, task.AuthorUserID)
		if err != nil {
			return domain.Task{}, err
		}
		patch.ListID.Value = &listID
	}

	wasCompleted := task.Completed

	if err := task.ApplyPatch(patch); err != nil {
		return domain.Task{}, fmt.Errorf("apply task patch: %w", err)
	}

	loc := actor.User.Location()
	anchorRepeat(&task, loc)

	// Выполнили повторяющуюся задачу: она остаётся в истории без повтора,
	// а повтор переходит к следующему экземпляру с новым сроком.
	var next *domain.Task
	if !wasCompleted && task.Completed {
		if n, ok := task.NextOccurrence(time.Now(), loc); ok {
			next = &n
			task.Repeat = nil
		}
	}

	patchedTask, err := s.tasksRepository.PatchTask(ctx, id, task)
	if err != nil {
		return domain.Task{}, fmt.Errorf("patch task: %w", err)
	}

	if !wasCompleted && patchedTask.Completed {
		s.enqueueListChange(ctx, actor, patchedTask, domain.ListChangeTaskCompleted)
	}

	// Без транзакции: если создать следующую не удалось, выполнение уже
	// сохранено, а ошибка вернётся клиенту.
	if next != nil {
		created, err := s.tasksRepository.CreateTask(ctx, *next)
		if err != nil {
			return domain.Task{}, fmt.Errorf("create next occurrence: %w", err)
		}
		// Чеклист переходит в следующий повтор без отметок.
		if task.ItemsTotal > 0 {
			if err := s.tasksRepository.CopyTaskItems(ctx, id, created.ID); err != nil {
				return domain.Task{}, fmt.Errorf("copy checklist to next occurrence: %w", err)
			}
		}
	}

	return patchedTask, nil
}

// anchorRepeat привязывает месячный и годовой повтор к дню срока по местному
// времени пользователя.
func anchorRepeat(task *domain.Task, loc *time.Location) {
	if task.Repeat == nil || task.DueAt == nil {
		return
	}
	anchored := task.Repeat.Anchored(*task.DueAt, loc)
	task.Repeat = &anchored
}
