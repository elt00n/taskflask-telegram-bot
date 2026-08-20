CREATE TABLE users (
    id bigint PRIMARY KEY CHECK (id > 0),
    username text NOT NULL DEFAULT '',
    first_name text NOT NULL CHECK (btrim(first_name) <> ''),
    timezone text NOT NULL DEFAULT 'UTC',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE UNIQUE INDEX users_username_unique_idx
    ON users (lower(username))
    WHERE username <> '';

CREATE TABLE chats (
    id bigint PRIMARY KEY CHECK (id <> 0),
    title text NOT NULL DEFAULT '',
    type text NOT NULL CHECK (type IN ('private', 'group', 'supergroup')),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

INSERT INTO users (id, first_name, created_at, updated_at)
SELECT user_id, 'Telegram user ' || user_id, now(), now()
FROM (
    SELECT creator_id AS user_id FROM tasks
    UNION
    SELECT user_id FROM task_participants
    UNION
    SELECT user_id FROM chat_members
) known_users
ON CONFLICT (id) DO NOTHING;

INSERT INTO chats (id, type, created_at, updated_at)
SELECT
    chat_id,
    CASE WHEN chat_id > 0 THEN 'private' ELSE 'supergroup' END,
    now(),
    now()
FROM (
    SELECT chat_id FROM tasks
    UNION
    SELECT chat_id FROM chat_members
) known_chats
ON CONFLICT (id) DO NOTHING;

ALTER TABLE tasks
    ADD CONSTRAINT tasks_chat_fk
        FOREIGN KEY (chat_id) REFERENCES chats(id),
    ADD CONSTRAINT tasks_creator_fk
        FOREIGN KEY (creator_id) REFERENCES users(id);

ALTER TABLE task_participants
    ADD CONSTRAINT task_participants_user_fk
        FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE chat_members
    ADD CONSTRAINT chat_members_chat_fk
        FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
    ADD CONSTRAINT chat_members_user_fk
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
