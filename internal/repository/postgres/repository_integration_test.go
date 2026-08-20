//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/database"
	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
	postgresrepository "github.com/elt00n/taskflask-telegram-bot/internal/repository/postgres"
	"github.com/elt00n/taskflask-telegram-bot/internal/service"
)

func TestRepositoriesPersistTaskAndMembership(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	taskID, err := service.GenerateTaskID()
	if err != nil {
		t.Fatalf("generate task ID: %v", err)
	}

	chatID := domain.ChatID(-time.Now().UnixNano())
	creatorID := domain.UserID(7_001)
	assigneeID := domain.UserID(7_002)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM tasks WHERE id = $1", taskID)
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM chat_members WHERE chat_id = $1",
			chatID,
		)
		_, _ = pool.Exec(context.Background(), "DELETE FROM chats WHERE id = $1", chatID)
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM users WHERE id IN ($1, $2)",
			creatorID,
			assigneeID,
		)
	})

	now := time.Now().UTC().Truncate(time.Microsecond)
	userRepository := postgresrepository.NewUserRepository(pool)
	chatRepository := postgresrepository.NewChatRepository(pool)
	memberRepository := postgresrepository.NewChatMemberRepository(pool)

	creator, err := domain.NewUser(domain.NewUserParams{
		ID:        creatorID,
		Username:  "database_creator",
		FirstName: "Создатель",
	}, now)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	assignee, err := domain.NewUser(domain.NewUserParams{
		ID:        assigneeID,
		Username:  "database_assignee",
		FirstName: "Исполнитель",
	}, now)
	if err != nil {
		t.Fatalf("create assignee: %v", err)
	}
	chat, err := domain.NewChat(chatID, "Проверка БД", domain.ChatTypeGroup, now)
	if err != nil {
		t.Fatalf("create chat: %v", err)
	}
	if err := userRepository.Upsert(ctx, creator); err != nil {
		t.Fatalf("save creator: %v", err)
	}
	if err := userRepository.Upsert(ctx, assignee); err != nil {
		t.Fatalf("save assignee: %v", err)
	}
	if err := chatRepository.Upsert(ctx, chat); err != nil {
		t.Fatalf("save chat: %v", err)
	}

	for _, userID := range []domain.UserID{creatorID, assigneeID} {
		member, err := domain.NewChatMember(
			chatID,
			userID,
			domain.ChatMemberStatusMember,
			now,
		)
		if err != nil {
			t.Fatalf("create member: %v", err)
		}
		if err := memberRepository.Upsert(ctx, member); err != nil {
			t.Fatalf("save member: %v", err)
		}
	}

	storedMember, err := memberRepository.Get(ctx, chatID, creatorID)
	if err != nil {
		t.Fatalf("get member: %v", err)
	}
	if storedMember.UserID != creatorID || storedMember.ChatID != chatID {
		t.Fatalf("stored member = %#v", storedMember)
	}

	resolved, err := userRepository.FindByUsernameInChat(ctx, chatID, "@DATABASE_ASSIGNEE")
	if err != nil || resolved.ID != assigneeID {
		t.Fatalf("resolve assignee = %#v, error %v", resolved, err)
	}

	task, err := domain.NewTask(domain.NewTaskParams{
		ID:        taskID,
		ChatID:    chatID,
		CreatorID: creatorID,
		Title:     "Проверить PostgreSQL",
	}, now)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	participants := []domain.TaskParticipant{{
		TaskID: task.ID,
		UserID: assigneeID,
		Role:   domain.TaskParticipantRoleAssignee,
	}}

	taskRepository := postgresrepository.NewTaskRepository(pool)
	if err := taskRepository.Create(ctx, task, participants); err != nil {
		t.Fatalf("save task: %v", err)
	}
	resolvedTaskID, err := taskRepository.ResolveID(ctx, chatID, string(task.ID)[:8])
	if err != nil || resolvedTaskID != task.ID {
		t.Fatalf("resolve task ID = %q, error %v; want %q", resolvedTaskID, err, task.ID)
	}
	if _, err := taskRepository.ResolveID(ctx, chatID, strings.Repeat("f", 8)); !errors.Is(err, repository.ErrTaskNotFound) {
		t.Fatalf("resolve missing task error = %v, want %v", err, repository.ErrTaskNotFound)
	}

	storedTask, storedParticipants, err := taskRepository.Get(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if storedTask != task {
		t.Fatalf("stored task = %#v, want %#v", storedTask, task)
	}
	if len(storedParticipants) != 1 || storedParticipants[0] != participants[0] {
		t.Fatalf("stored participants = %#v, want %#v", storedParticipants, participants)
	}

	if err := storedTask.SetPriority(domain.TaskPriorityCritical, now.Add(time.Minute)); err != nil {
		t.Fatalf("change priority: %v", err)
	}
	if err := taskRepository.Update(ctx, storedTask); err != nil {
		t.Fatalf("update task: %v", err)
	}

	listedTasks, err := taskRepository.List(ctx, repository.TaskFilter{
		ChatID: chatID,
		UserID: &assigneeID,
	})
	if err != nil {
		t.Fatalf("list assignee tasks: %v", err)
	}
	if len(listedTasks) != 1 || listedTasks[0].Priority != domain.TaskPriorityCritical {
		t.Fatalf("listed tasks = %#v, want one critical task", listedTasks)
	}

	renamedAssignee, err := domain.NewUser(domain.NewUserParams{
		ID:        assigneeID,
		Username:  "renamed_assignee",
		FirstName: "Новое имя",
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("create renamed assignee: %v", err)
	}
	if err := userRepository.Upsert(ctx, renamedAssignee); err != nil {
		t.Fatalf("save renamed assignee: %v", err)
	}
	if _, err := userRepository.FindByUsernameInChat(ctx, chatID, "database_assignee"); !errors.Is(err, repository.ErrUserNotFound) {
		t.Fatalf("old username error = %v, want %v", err, repository.ErrUserNotFound)
	}
	resolved, err = userRepository.FindByUsernameInChat(ctx, chatID, "renamed_assignee")
	if err != nil || resolved.ID != assigneeID {
		t.Fatalf("resolve renamed assignee = %#v, error %v", resolved, err)
	}
}
