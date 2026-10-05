package notifications_postgres_repository

import (
	"context"
	"fmt"
	"time"

	core_postgres_pool "github.com/dadqeds/todoapp/internal/core/repository/postgres/pool"
	notifications_service "github.com/dadqeds/todoapp/internal/features/notifications/service"
)

type NotificationsRepository struct {
	pool core_postgres_pool.Pool
}

func NewNotificationsRepository(pool core_postgres_pool.Pool) *NotificationsRepository {
	return &NotificationsRepository{pool: pool}
}

// GetPendingReminders — неотправленные напоминания по незакрытым задачам со
// сроком до horizon у авторов, которые включили напоминания и вошли через Telegram.
func (r *NotificationsRepository) GetPendingReminders(ctx context.Context, horizon time.Time) ([]notifications_service.PendingReminder, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	SELECT t.id, t.title, l.title, t.due_at, t.due_all_day, t.remind_before_minutes,
		u.telegram_id, u.timezone, u.digest_minute
	FROM todoapp.tasks t
	JOIN todoapp.users u ON u.id = t.author_user_id
	JOIN todoapp.lists l ON l.id = t.list_id
	WHERE t.remind_before_minutes IS NOT NULL
		AND t.reminded_at IS NULL
		AND NOT t.completed
		AND t.due_at <= $1
		AND u.remind_enabled
		AND u.telegram_id IS NOT NULL
	ORDER BY t.due_at;
	`

	rows, err := r.pool.Query(ctx, query, horizon)
	if err != nil {
		return nil, fmt.Errorf("select pending reminders: %w", err)
	}
	defer rows.Close()

	var out []notifications_service.PendingReminder
	for rows.Next() {
		var p notifications_service.PendingReminder
		if err := rows.Scan(&p.TaskID, &p.Title, &p.ListTitle, &p.DueAt, &p.DueAllDay, &p.RemindBeforeMinutes,
			&p.ChatID, &p.Timezone, &p.DigestMinute); err != nil {
			return nil, fmt.Errorf("scan pending reminder: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}

	return out, nil
}

// ClaimReminder помечает напоминание отправленным. false — его уже кто-то забрал
// или срок поменяли: отправлять не нужно.
func (r *NotificationsRepository) ClaimReminder(ctx context.Context, taskID int) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	tag, err := r.pool.Exec(ctx, `UPDATE todoapp.tasks SET reminded_at = now() WHERE id=$1 AND reminded_at IS NULL;`, taskID)
	if err != nil {
		return false, fmt.Errorf("claim reminder: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// ReleaseReminder возвращает напоминание в очередь после временной ошибки отправки.
func (r *NotificationsRepository) ReleaseReminder(ctx context.Context, taskID int) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if _, err := r.pool.Exec(ctx, `UPDATE todoapp.tasks SET reminded_at = NULL WHERE id=$1;`, taskID); err != nil {
		return fmt.Errorf("release reminder: %w", err)
	}
	return nil
}

// GetDigestRecipients — пользователи с включённой сводкой, вошедшие через Telegram.
func (r *NotificationsRepository) GetDigestRecipients(ctx context.Context) ([]notifications_service.DigestRecipient, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	SELECT id, telegram_id, timezone, digest_minute, digest_sent_on
	FROM todoapp.users
	WHERE digest_enabled AND telegram_id IS NOT NULL;
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("select digest recipients: %w", err)
	}
	defer rows.Close()

	var out []notifications_service.DigestRecipient
	for rows.Next() {
		var d notifications_service.DigestRecipient
		if err := rows.Scan(&d.UserID, &d.ChatID, &d.Timezone, &d.DigestMinute, &d.SentOn); err != nil {
			return nil, fmt.Errorf("scan digest recipient: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}

	return out, nil
}

// ClaimDigest отмечает, что сводка за localDate отправлена. false — уже была.
func (r *NotificationsRepository) ClaimDigest(ctx context.Context, userID int, localDate time.Time) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	UPDATE todoapp.users SET digest_sent_on = $2::date
	WHERE id=$1 AND (digest_sent_on IS NULL OR digest_sent_on < $2::date);
	`
	tag, err := r.pool.Exec(ctx, query, userID, localDate.Format("2006-01-02"))
	if err != nil {
		return false, fmt.Errorf("claim digest: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// GetDigestTasks — незакрытые задачи из своих и общих списков пользователя со
// сроком до until (сегодняшние и просроченные).
func (r *NotificationsRepository) GetDigestTasks(ctx context.Context, userID int, until time.Time) ([]notifications_service.DigestTask, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	SELECT t.title, t.due_at, t.due_all_day
	FROM todoapp.tasks t
	WHERE NOT t.completed
		AND t.due_at <= $2
		AND t.list_id IN (
			SELECT id FROM todoapp.lists WHERE owner_user_id = $1
			UNION
			SELECT list_id FROM todoapp.list_members WHERE user_id = $1
		)
	ORDER BY t.due_at
	LIMIT 30;
	`

	rows, err := r.pool.Query(ctx, query, userID, until)
	if err != nil {
		return nil, fmt.Errorf("select digest tasks: %w", err)
	}
	defer rows.Close()

	var out []notifications_service.DigestTask
	for rows.Next() {
		var d notifications_service.DigestTask
		if err := rows.Scan(&d.Title, &d.DueAt, &d.DueAllDay); err != nil {
			return nil, fmt.Errorf("scan digest task: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}

	return out, nil
}
