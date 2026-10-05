-- Настройки уведомлений пользователя.
ALTER TABLE todoapp.users ADD COLUMN remind_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE todoapp.users ADD COLUMN digest_enabled BOOLEAN NOT NULL DEFAULT FALSE;
-- Время утренней сводки по местному времени, минуты от полуночи (9:00 = 540).
ALTER TABLE todoapp.users ADD COLUMN digest_minute SMALLINT NOT NULL DEFAULT 540
    CHECK (digest_minute BETWEEN 0 AND 1439);
-- Местная дата последней отправленной сводки: не больше одной в день.
ALTER TABLE todoapp.users ADD COLUMN digest_sent_on DATE;

-- За сколько минут до срока напомнить: 0 — в срок, 15, 60, 1440 — за день.
ALTER TABLE todoapp.tasks ADD COLUMN remind_before_minutes INTEGER
    CHECK (remind_before_minutes IN (0, 15, 60, 1440));
ALTER TABLE todoapp.tasks ADD CONSTRAINT tasks_remind_requires_due_check
    CHECK (remind_before_minutes IS NULL OR due_at IS NOT NULL);
-- Когда напоминание отправлено; сбрасывается при смене срока.
ALTER TABLE todoapp.tasks ADD COLUMN reminded_at TIMESTAMPTZ;

CREATE INDEX tasks_pending_reminders_idx ON todoapp.tasks (due_at)
    WHERE remind_before_minutes IS NOT NULL AND reminded_at IS NULL AND NOT completed;
