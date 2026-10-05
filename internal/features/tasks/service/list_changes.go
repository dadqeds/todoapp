package tasks_service

import (
	"context"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	"github.com/dadqeds/todoapp/internal/core/domain"
	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	"go.uber.org/zap"
)

// enqueueListChange сообщает остальным участникам списка об изменении.
// Задача уже сохранена, поэтому ошибка очереди только пишется в лог: из-за
// уведомления запрос пользователя не должен падать.
func (s *TasksService) enqueueListChange(ctx context.Context, actor core_auth.Actor, task domain.Task, kind domain.ListChangeKind) {
	change := domain.ListChange{
		ListID:      task.ListID,
		ActorUserID: actor.User.ID,
		Kind:        kind,
		TaskTitle:   task.Title,
	}

	if err := s.changesRepository.EnqueueListChange(ctx, change); err != nil {
		core_logger.FromContext(ctx).Warn("enqueue list change",
			zap.Int("list_id", task.ListID),
			zap.String("kind", string(kind)),
			zap.Error(err),
		)
	}
}
