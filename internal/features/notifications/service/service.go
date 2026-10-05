// Package notifications_service отправляет через бота напоминания о сроках,
// утреннюю сводку и изменения в общих списках. Работает фоном: раз в минуту
// проверяет, что пора отправить.
package notifications_service

import (
	"context"
	"time"

	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_telegram "github.com/dadqeds/todoapp/internal/core/telegram"
)

type PendingReminder struct {
	TaskID              int
	Title               string
	ListTitle           string
	DueAt               time.Time
	DueAllDay           bool
	RemindBeforeMinutes int
	ChatID              int64
	Timezone            string
	DigestMinute        int
}

type DigestRecipient struct {
	UserID       int
	ChatID       int64
	Timezone     string
	DigestMinute int
	SentOn       *time.Time
}

type DigestTask struct {
	Title     string
	DueAt     time.Time
	DueAllDay bool
}

// ListChange — событие из очереди: кто что сделал с задачей.
type ListChange struct {
	ID        int64
	ActorName string
	Kind      string
	TaskTitle string
	CreatedAt time.Time
}

// ListChangeBatch — накопившиеся изменения одного списка для одного получателя.
type ListChangeBatch struct {
	ListID          int
	ListTitle       string
	RecipientUserID int
	ChatID          int64
	Changes         []ListChange
}

type Repository interface {
	GetPendingReminders(ctx context.Context, horizon time.Time) ([]PendingReminder, error)
	ClaimReminder(ctx context.Context, taskID int) (bool, error)
	ReleaseReminder(ctx context.Context, taskID int) error
	GetDigestRecipients(ctx context.Context) ([]DigestRecipient, error)
	ClaimDigest(ctx context.Context, userID int, localDate time.Time) (bool, error)
	GetDigestTasks(ctx context.Context, userID int, until time.Time) ([]DigestTask, error)

	GetPendingListChanges(ctx context.Context, readyBefore time.Time) ([]ListChangeBatch, error)
	ClaimListChanges(ctx context.Context, ids []int64) (bool, error)
	ReleaseListChanges(ctx context.Context, ids []int64) error
	DeleteListChangesBefore(ctx context.Context, before time.Time) error
}

type Sender interface {
	SendMessage(ctx context.Context, chatID int64, html string, button *core_telegram.Button) error
}

type Notifier struct {
	repo        Repository
	sender      Sender
	botUsername string
	log         *core_logger.Logger
	now         func() time.Time
}

func NewNotifier(repo Repository, sender Sender, botUsername string, log *core_logger.Logger) *Notifier {
	return &Notifier{
		repo:        repo,
		sender:      sender,
		botUsername: botUsername,
		log:         log,
		now:         time.Now,
	}
}

const (
	tickInterval = time.Minute
	// Напоминание «за день» по задаче на весь день — самое раннее: до двух суток.
	reminderHorizon = 50 * time.Hour
	// Сводку, которую не успели отправить вовремя (сервер был выключен), шлём
	// не позже трёх часов после назначенного времени.
	digestWindow = 3 * 60
)

// Run проверяет уведомления сразу и затем раз в минуту, пока жив ctx.
func (n *Notifier) Run(ctx context.Context) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		n.Tick(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Tick отправляет всё, что пора отправить к текущему моменту.
func (n *Notifier) Tick(ctx context.Context) {
	now := n.now()
	n.sendReminders(ctx, now)
	n.sendDigests(ctx, now)
	n.sendListChanges(ctx, now)
}

func loadLocation(name string) *time.Location {
	if loc, err := time.LoadLocation(name); err == nil && name != "" {
		return loc
	}
	return time.UTC
}

func (n *Notifier) openButton(text, startParam string) *core_telegram.Button {
	if n.botUsername == "" {
		return nil
	}
	url := "https://t.me/" + n.botUsername + "?startapp"
	if startParam != "" {
		url += "=" + startParam
	}
	return &core_telegram.Button{Text: text, URL: url}
}
