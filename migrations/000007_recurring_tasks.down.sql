ALTER TABLE todoapp.tasks DROP CONSTRAINT tasks_repeat_requires_due_check;
ALTER TABLE todoapp.tasks DROP COLUMN repeat_rule;
ALTER TABLE todoapp.users DROP COLUMN timezone;
