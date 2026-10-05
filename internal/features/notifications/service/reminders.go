package notifications_service

import (
	"context"
	"errors"
	"fmt"
	"html"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_telegram "github.com/dadqeds/todoapp/internal/core/telegram"
	"go.uber.org/zap"
)

func (n *Notifier) sendReminders(ctx context.Context, now time.Time) {
	pending, err := n.repo.GetPendingReminders(ctx, now.Add(reminderHorizon))
	if err != nil {
		n.log.Error("get pending reminders", zap.Error(err))
		return
	}

	for _, p := range pending {
		n.processReminder(ctx, now, p)
	}
}

func (n *Notifier) processReminder(ctx context.Context, now time.Time, p PendingReminder) {
	loc := loadLocation(p.Timezone)
	remind := p.RemindBeforeMinutes
	task := domain.Task{DueAt: &p.DueAt, DueAllDay: p.DueAllDay, RemindBeforeMinutes: &remind}

	at, _ := task.ReminderAt(loc, p.DigestMinute)
	if now.Before(at) {
		return
	}

	// Срок уже прошёл (например, сервер был выключен): напоминать поздно,
	// просто снимаем из очереди.
	missed := now.After(p.DueAt)

	claimed, err := n.repo.ClaimReminder(ctx, p.TaskID)
	if err != nil {
		n.log.Error("claim reminder", zap.Int("task_id", p.TaskID), zap.Error(err))
		return
	}
	if !claimed || missed {
		return
	}

	err = n.sender.SendMessage(ctx, p.ChatID, reminderText(p, now, loc), n.openButton("Открыть задачу", fmt.Sprintf("task_%d", p.TaskID)))
	switch {
	case err == nil:
		n.log.Info("reminder sent", zap.Int("task_id", p.TaskID))
	case errors.Is(err, core_telegram.ErrChatUnavailable):
		// Человек не разрешил боту писать — повторять бессмысленно.
		n.log.Warn("reminder not delivered", zap.Int("task_id", p.TaskID), zap.Error(err))
	default:
		n.log.Warn("send reminder, will retry", zap.Int("task_id", p.TaskID), zap.Error(err))
		if err := n.repo.ReleaseReminder(ctx, p.TaskID); err != nil {
			n.log.Error("release reminder", zap.Int("task_id", p.TaskID), zap.Error(err))
		}
	}
}

// reminderText — заголовок по тому, сколько осталось, название и где/когда.
func reminderText(p PendingReminder, now time.Time, loc *time.Location) string {
	var head string
	switch {
	case p.DueAllDay && p.RemindBeforeMinutes >= 1440:
		head = "Завтра срок"
	case p.DueAllDay:
		head = "Сегодня срок"
	case p.RemindBeforeMinutes == 0:
		head = "Пора"
	case p.RemindBeforeMinutes == 15:
		head = "Через 15 минут"
	case p.RemindBeforeMinutes == 60:
		head = "Через час"
	default:
		head = "Завтра"
	}

	return fmt.Sprintf("<b>%s</b>\n%s\n<i>%s, %s</i>",
		head,
		html.EscapeString(p.Title),
		html.EscapeString(p.ListTitle),
		dueText(p.DueAt, p.DueAllDay, now, loc),
	)
}
