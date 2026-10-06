-- Пункты чеклиста внутри задачи: только текст и галочка.
CREATE TABLE todoapp.task_items(
    id          SERIAL          PRIMARY KEY,
    version     BIGINT          NOT NULL DEFAULT 1,
    task_id     INTEGER         NOT NULL REFERENCES todoapp.tasks(id) ON DELETE CASCADE,
    title       VARCHAR(200)    NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
    done        BOOLEAN         NOT NULL DEFAULT FALSE,
    -- Порядок внутри задачи: новый пункт встаёт в конец.
    position    INTEGER         NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX task_items_task_id_idx ON todoapp.task_items (task_id, position);
