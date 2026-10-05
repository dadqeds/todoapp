CREATE INDEX IF NOT EXISTS tasks_author_user_id_idx ON todoapp.tasks (author_user_id);
CREATE INDEX IF NOT EXISTS tasks_created_at_idx ON todoapp.tasks (created_at);
