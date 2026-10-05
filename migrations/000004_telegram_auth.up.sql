ALTER TABLE todoapp.users ADD COLUMN telegram_id BIGINT UNIQUE;

-- В Telegram бывают имена из одной-двух букв.
ALTER TABLE todoapp.users DROP CONSTRAINT users_full_name_check;
ALTER TABLE todoapp.users ADD CONSTRAINT users_full_name_check
    CHECK (char_length(full_name) BETWEEN 1 AND 100);
