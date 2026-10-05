package notifications_service

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (n *Notifier) sendDigests(ctx context.Context, now time.Time) {
	recipients, err := n.repo.GetDigestRecipients(ctx)
	if err != nil {
		n.log.Error("get digest recipients", zap.Error(err))
		return
	}

	for _, r := range recipients {
		n.processDigest(ctx, now, r)
	}
}

func (n *Notifier) processDigest(ctx context.Context, now time.Time, r DigestRecipient) {
	loc := loadLocation(r.Timezone)
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)

	if r.SentOn != nil && !r.SentOn.Before(time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)) {
		return
	}

	minute := local.Hour()*60 + local.Minute()
	if minute < r.DigestMinute || minute > r.DigestMinute+digestWindow {
		return
	}

	claimed, err := n.repo.ClaimDigest(ctx, r.UserID, today)
	if err != nil {
		n.log.Error("claim digest", zap.Int("user_id", r.UserID), zap.Error(err))
		return
	}
	if !claimed {
		return
	}

	tasks, err := n.repo.GetDigestTasks(ctx, r.UserID, today.AddDate(0, 0, 1))
	if err != nil {
		n.log.Error("get digest tasks", zap.Int("user_id", r.UserID), zap.Error(err))
		return
	}

	// Пустую сводку не шлём: «задач нет» каждое утро быстро надоест.
	if len(tasks) == 0 {
		return
	}

	if err := n.sender.SendMessage(ctx, r.ChatID, digestText(tasks, now, loc), n.openButton("Открыть задачи", "")); err != nil {
		n.log.Warn("send digest", zap.Int("user_id", r.UserID), zap.Error(err))
		return
	}
	n.log.Info("digest sent", zap.Int("user_id", r.UserID), zap.Int("tasks", len(tasks)))
}

func digestText(tasks []DigestTask, now time.Time, loc *time.Location) string {
	var today, overdue []string
	for _, t := range tasks {
		line := "• " + html.EscapeString(t.Title)
		isOverdue := now.After(t.DueAt)
		if !t.DueAllDay && !isOverdue {
			line += " — " + t.DueAt.In(loc).Format("15:04")
		}
		if isOverdue {
			line += " (" + dueText(t.DueAt, t.DueAllDay, now, loc) + ")"
			overdue = append(overdue, line)
		} else {
			today = append(today, line)
		}
	}

	var b strings.Builder
	hello := greeting(now.In(loc).Hour())
	if len(today) > 0 {
		fmt.Fprintf(&b, "<b>%s. На сегодня %d %s</b>\n%s", hello, len(today), plural(len(today), "задача", "задачи", "задач"), strings.Join(today, "\n"))
	} else {
		fmt.Fprintf(&b, "<b>%s. На сегодня новых задач нет</b>", hello)
	}
	if len(overdue) > 0 {
		fmt.Fprintf(&b, "\n\n<b>Просрочено</b>\n%s", strings.Join(overdue, "\n"))
	}
	return b.String()
}

// greeting — приветствие по местному времени: сводку можно поставить и на вечер.
func greeting(hour int) string {
	switch {
	case hour >= 5 && hour < 12:
		return "Доброе утро"
	case hour >= 12 && hour < 18:
		return "Добрый день"
	case hour >= 18:
		return "Добрый вечер"
	default:
		return "Доброй ночи"
	}
}
