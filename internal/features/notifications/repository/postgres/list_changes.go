package notifications_postgres_repository

import (
	"context"
	"fmt"
	"time"

	"github.com/dadqeds/todoapp/internal/core/domain"
	notifications_service "github.com/dadqeds/todoapp/internal/features/notifications/service"
)

// EnqueueListChange ставит событие в очередь каждому, кто должен о нём узнать:
// владельцу и участникам списка с включённым переключателем, кроме автора
// изменения и тех, кто не входил через Telegram. В личном списке получателей нет.
func (r *NotificationsRepository) EnqueueListChange(ctx context.Context, change domain.ListChange) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	INSERT INTO todoapp.list_change_notifications (list_id, recipient_user_id, actor_user_id, kind, task_title)
	SELECT $1, r.user_id, $2, $3, $4
	FROM (
		SELECT owner_user_id AS user_id, owner_notify_changes AS notify FROM todoapp.lists WHERE id = $1
		UNION ALL
		SELECT user_id, notify_changes FROM todoapp.list_members WHERE list_id = $1
	) r
	JOIN todoapp.users u ON u.id = r.user_id
	WHERE r.notify AND r.user_id <> $2 AND u.telegram_id IS NOT NULL;
	`
	if _, err := r.pool.Exec(ctx, query, change.ListID, change.ActorUserID, string(change.Kind), change.TaskTitle); err != nil {
		return fmt.Errorf("insert list change: %w", err)
	}

	return nil
}

// GetPendingListChanges — неотправленные события, сгруппированные по списку и
// получателю. Группа попадает в выборку целиком, когда её самому раннему
// событию не меньше readyBefore. Получатель должен по-прежнему быть в списке
// и не выключить уведомления.
func (r *NotificationsRepository) GetPendingListChanges(ctx context.Context, readyBefore time.Time) ([]notifications_service.ListChangeBatch, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	SELECT n.id, n.list_id, l.title, n.recipient_user_id, u.telegram_id, a.full_name, n.kind, n.task_title, n.created_at
	FROM todoapp.list_change_notifications n
	JOIN todoapp.lists l ON l.id = n.list_id
	JOIN todoapp.users u ON u.id = n.recipient_user_id
	JOIN todoapp.users a ON a.id = n.actor_user_id
	LEFT JOIN todoapp.list_members m ON m.list_id = n.list_id AND m.user_id = n.recipient_user_id
	WHERE n.sent_at IS NULL
		AND u.telegram_id IS NOT NULL
		AND (
			(l.owner_user_id = n.recipient_user_id AND l.owner_notify_changes)
			OR m.notify_changes
		)
		AND (n.list_id, n.recipient_user_id) IN (
			SELECT list_id, recipient_user_id
			FROM todoapp.list_change_notifications
			WHERE sent_at IS NULL
			GROUP BY list_id, recipient_user_id
			HAVING MIN(created_at) <= $1
		)
	ORDER BY n.list_id, n.recipient_user_id, n.created_at, n.id;
	`

	rows, err := r.pool.Query(ctx, query, readyBefore)
	if err != nil {
		return nil, fmt.Errorf("select pending list changes: %w", err)
	}
	defer rows.Close()

	var out []notifications_service.ListChangeBatch
	for rows.Next() {
		var (
			b notifications_service.ListChangeBatch
			c notifications_service.ListChange
		)
		if err := rows.Scan(&c.ID, &b.ListID, &b.ListTitle, &b.RecipientUserID, &b.ChatID, &c.ActorName, &c.Kind, &c.TaskTitle, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan pending list change: %w", err)
		}

		if n := len(out); n > 0 && out[n-1].ListID == b.ListID && out[n-1].RecipientUserID == b.RecipientUserID {
			out[n-1].Changes = append(out[n-1].Changes, c)
			continue
		}
		b.Changes = []notifications_service.ListChange{c}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("next rows: %w", err)
	}

	return out, nil
}

// ClaimListChanges помечает события отправленными. false — их уже забрали.
func (r *NotificationsRepository) ClaimListChanges(ctx context.Context, ids []int64) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	tag, err := r.pool.Exec(ctx, `UPDATE todoapp.list_change_notifications SET sent_at = now() WHERE id = ANY($1) AND sent_at IS NULL;`, ids)
	if err != nil {
		return false, fmt.Errorf("claim list changes: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// ReleaseListChanges возвращает события в очередь после временной ошибки отправки.
func (r *NotificationsRepository) ReleaseListChanges(ctx context.Context, ids []int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if _, err := r.pool.Exec(ctx, `UPDATE todoapp.list_change_notifications SET sent_at = NULL WHERE id = ANY($1);`, ids); err != nil {
		return fmt.Errorf("release list changes: %w", err)
	}
	return nil
}

// DeleteListChangesBefore чистит очередь: отправленное и то, что так и не
// удалось отправить (получатель вышел из списка, выключил уведомления).
func (r *NotificationsRepository) DeleteListChangesBefore(ctx context.Context, before time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	if _, err := r.pool.Exec(ctx, `DELETE FROM todoapp.list_change_notifications WHERE created_at < $1;`, before); err != nil {
		return fmt.Errorf("delete old list changes: %w", err)
	}
	return nil
}
