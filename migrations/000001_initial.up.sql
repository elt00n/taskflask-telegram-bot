CREATE TABLE tasks (
    id uuid PRIMARY KEY,
    chat_id bigint NOT NULL,
    creator_id bigint NOT NULL CHECK (creator_id > 0),
    title text NOT NULL CHECK (btrim(title) <> ''),
    description text NOT NULL DEFAULT '',
    kind text NOT NULL CHECK (kind IN ('task', 'event')),
    status text NOT NULL CHECK (status IN ('new', 'in_progress', 'done', 'cancelled')),
    priority smallint NOT NULL CHECK (priority BETWEEN 1 AND 4),
    start_at timestamptz,
    end_at timestamptz,
    deadline timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    deleted_at timestamptz,
    CHECK (chat_id <> 0),
    CHECK (end_at IS NULL OR start_at IS NOT NULL),
    CHECK (end_at IS NULL OR end_at > start_at)
);

CREATE TABLE task_participants (
    task_id uuid NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id bigint NOT NULL CHECK (user_id > 0),
    role text NOT NULL CHECK (role IN ('owner', 'assignee', 'editor')),
    PRIMARY KEY (task_id, user_id)
);

CREATE TABLE chat_members (
    chat_id bigint NOT NULL CHECK (chat_id <> 0),
    user_id bigint NOT NULL CHECK (user_id > 0),
    status text NOT NULL CHECK (
        status IN ('member', 'administrator', 'owner', 'left', 'banned')
    ),
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (chat_id, user_id)
);

CREATE INDEX tasks_chat_created_idx
    ON tasks (chat_id, created_at, id)
    WHERE deleted_at IS NULL;

CREATE INDEX tasks_chat_creator_idx
    ON tasks (chat_id, creator_id)
    WHERE deleted_at IS NULL;

CREATE INDEX tasks_chat_priority_deadline_idx
    ON tasks (chat_id, priority DESC, deadline)
    WHERE deleted_at IS NULL;

CREATE INDEX task_participants_user_idx
    ON task_participants (user_id, task_id);
