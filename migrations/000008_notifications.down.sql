DROP INDEX todoapp.tasks_pending_reminders_idx;
ALTER TABLE todoapp.tasks DROP COLUMN reminded_at;
ALTER TABLE todoapp.tasks DROP CONSTRAINT tasks_remind_requires_due_check;
ALTER TABLE todoapp.tasks DROP COLUMN remind_before_minutes;
ALTER TABLE todoapp.users DROP COLUMN digest_sent_on;
ALTER TABLE todoapp.users DROP COLUMN digest_minute;
ALTER TABLE todoapp.users DROP COLUMN digest_enabled;
ALTER TABLE todoapp.users DROP COLUMN remind_enabled;
