-- Сообщать ли участнику об изменениях в общем списке. Переключатель у каждого
-- свой; у владельца — в lists, у участников — в list_members.
ALTER TABLE todoapp.list_members ADD COLUMN notify_changes BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE todoapp.lists ADD COLUMN owner_notify_changes BOOLEAN NOT NULL DEFAULT TRUE;

-- Очередь уведомлений об изменениях: строка на событие и получателя.
-- Бот собирает события за несколько минут в одно сообщение на список и получателя.
CREATE TABLE todoapp.list_change_notifications(
    id                  BIGSERIAL       PRIMARY KEY,
    list_id             INTEGER         NOT NULL REFERENCES todoapp.lists(id) ON DELETE CASCADE,
    recipient_user_id   INTEGER         NOT NULL REFERENCES todoapp.users(id) ON DELETE CASCADE,
    actor_user_id       INTEGER         NOT NULL REFERENCES todoapp.users(id) ON DELETE CASCADE,
    kind                VARCHAR(16)     NOT NULL CHECK (kind IN ('task_added', 'task_completed')),
    task_title          VARCHAR(100)    NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT now(),
    -- Когда отправлено; NULL — ещё в очереди.
    sent_at             TIMESTAMPTZ
);

CREATE INDEX list_change_notifications_pending_idx ON todoapp.list_change_notifications (list_id, recipient_user_id, created_at)
    WHERE sent_at IS NULL;
CREATE INDEX list_change_notifications_created_at_idx ON todoapp.list_change_notifications (created_at);
