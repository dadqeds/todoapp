-- Часовой пояс пользователя (IANA, например Europe/Moscow). Мини-апп
-- присылает его с телефона; нужен для повторов и времени уведомлений.
ALTER TABLE todoapp.users ADD COLUMN timezone VARCHAR(64) NOT NULL DEFAULT 'UTC';

-- Правило повтора: daily | weekly:1,4 (дни недели, 1 = пн) | monthly:31 | yearly:02-29.
ALTER TABLE todoapp.tasks ADD COLUMN repeat_rule VARCHAR(32);
ALTER TABLE todoapp.tasks ADD CONSTRAINT tasks_repeat_requires_due_check
    CHECK (repeat_rule IS NULL OR due_at IS NOT NULL);
