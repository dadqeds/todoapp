-- Участники общих списков (владелец хранится в lists.owner_user_id).
CREATE TABLE todoapp.list_members(
    list_id     INTEGER         NOT NULL REFERENCES todoapp.lists(id) ON DELETE CASCADE,
    user_id     INTEGER         NOT NULL REFERENCES todoapp.users(id) ON DELETE CASCADE,
    joined_at   TIMESTAMPTZ     NOT NULL DEFAULT now(),
    PRIMARY KEY (list_id, user_id)
);

CREATE INDEX list_members_user_id_idx ON todoapp.list_members (user_id);

-- Код приглашения. NULL — приглашение выключено; новый код делает старую ссылку недействительной.
ALTER TABLE todoapp.lists ADD COLUMN invite_code VARCHAR(32) UNIQUE;
