DROP TABLE todoapp.list_change_notifications;
ALTER TABLE todoapp.lists DROP COLUMN owner_notify_changes;
ALTER TABLE todoapp.list_members DROP COLUMN notify_changes;
