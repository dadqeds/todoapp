ALTER TABLE todoapp.tasks DROP CONSTRAINT tasks_due_all_day_check;
ALTER TABLE todoapp.tasks DROP COLUMN due_all_day;
ALTER TABLE todoapp.tasks DROP COLUMN due_at;
ALTER TABLE todoapp.tasks DROP COLUMN list_id;
DROP TABLE todoapp.lists;
