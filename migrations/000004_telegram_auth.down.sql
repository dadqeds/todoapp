ALTER TABLE todoapp.users DROP CONSTRAINT users_full_name_check;
ALTER TABLE todoapp.users ADD CONSTRAINT users_full_name_check
    CHECK (char_length(full_name) BETWEEN 3 AND 100);

ALTER TABLE todoapp.users DROP COLUMN telegram_id;
