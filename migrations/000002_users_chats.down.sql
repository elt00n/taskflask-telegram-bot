ALTER TABLE chat_members
    DROP CONSTRAINT IF EXISTS chat_members_user_fk,
    DROP CONSTRAINT IF EXISTS chat_members_chat_fk;

ALTER TABLE task_participants
    DROP CONSTRAINT IF EXISTS task_participants_user_fk;

ALTER TABLE tasks
    DROP CONSTRAINT IF EXISTS tasks_creator_fk,
    DROP CONSTRAINT IF EXISTS tasks_chat_fk;

DROP TABLE IF EXISTS chats;
DROP TABLE IF EXISTS users;
