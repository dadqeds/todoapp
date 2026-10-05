CREATE TABLE todoapp.lists(
    id              SERIAL          PRIMARY KEY,
    version         BIGINT          NOT NULL DEFAULT 1,
    title           VARCHAR(50)     NOT NULL CHECK (char_length(title) BETWEEN 1 AND 50),
    color           VARCHAR(16)     NOT NULL CHECK (color IN ('green', 'violet', 'coral', 'blue', 'pink', 'amber')),
    owner_user_id   INTEGER         NOT NULL REFERENCES todoapp.users(id) ON DELETE CASCADE,
    is_default      BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX lists_owner_user_id_idx ON todoapp.lists (owner_user_id);
-- У каждого пользователя ровно один список по умолчанию.
CREATE UNIQUE INDEX lists_one_default_per_owner ON todoapp.lists (owner_user_id) WHERE is_default;

-- Список «Личное» по умолчанию для всех существующих пользователей.
INSERT INTO todoapp.lists (title, color, owner_user_id, is_default)
SELECT 'Личное', 'coral', id, TRUE FROM todoapp.users;

ALTER TABLE todoapp.tasks ADD COLUMN list_id INTEGER REFERENCES todoapp.lists(id) ON DELETE CASCADE;

UPDATE todoapp.tasks t
SET list_id = l.id
FROM todoapp.lists l
WHERE l.owner_user_id = t.author_user_id AND l.is_default;

ALTER TABLE todoapp.tasks ALTER COLUMN list_id SET NOT NULL;
CREATE INDEX tasks_list_id_idx ON todoapp.tasks (list_id);

-- Срок: момент времени с часовым поясом. due_all_day = срок «на весь день»,
-- тогда due_at указывает на конец этого дня по времени пользователя.
ALTER TABLE todoapp.tasks ADD COLUMN due_at TIMESTAMPTZ;
ALTER TABLE todoapp.tasks ADD COLUMN due_all_day BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE todoapp.tasks ADD CONSTRAINT tasks_due_all_day_check
    CHECK (due_at IS NOT NULL OR NOT due_all_day);
CREATE INDEX tasks_due_at_idx ON todoapp.tasks (due_at);
