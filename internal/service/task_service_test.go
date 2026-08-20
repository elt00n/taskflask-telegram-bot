package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository/memory"
	"github.com/elt00n/taskflask-telegram-bot/internal/service"
)

func TestTaskServiceCreate(t *testing.T) {
	ctx := context.Background()
	fixedNow := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	taskRepo := memory.NewTaskRepository()
	memberRepo := memory.NewChatMemberRepository()
	upsertActiveMember(t, memberRepo, -100123, 1)
	upsertActiveMember(t, memberRepo, -100123, 2)

	taskService := service.NewTaskService(
		taskRepo,
		memberRepo,
		func() (domain.TaskID, error) { return "task-1", nil },
		func() time.Time { return fixedNow },
	)

	task, err := taskService.Create(ctx, service.CreateTaskCommand{
		ChatID:         -100123,
		CreatorID:      1,
		Title:          "  Подготовить отчёт  ",
		ParticipantIDs: []domain.UserID{1, 2, 2},
	})
	if err != nil {
		t.Fatalf("Create() returned an unexpected error: %v", err)
	}

	if task.ID != "task-1" {
		t.Errorf("task ID = %q, want %q", task.ID, "task-1")
	}
	if task.Title != "Подготовить отчёт" {
		t.Errorf("task title = %q, want %q", task.Title, "Подготовить отчёт")
	}
	if !task.CreatedAt.Equal(fixedNow) {
		t.Errorf("CreatedAt = %v, want %v", task.CreatedAt, fixedNow)
	}

	_, participants, err := taskRepo.Get(ctx, task.ID)
	if err != nil {
		t.Fatalf("Get() returned an unexpected error: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("saved %d participants, want 2", len(participants))
	}
	if participants[0].UserID != 1 || participants[0].Role != domain.TaskParticipantRoleOwner {
		t.Errorf("creator participant = %#v, want owner user 1", participants[0])
	}
	if participants[1].UserID != 2 || participants[1].Role != domain.TaskParticipantRoleAssignee {
		t.Errorf("assigned participant = %#v, want assignee user 2", participants[1])
	}
	for _, participant := range participants {
		if participant.TaskID != task.ID {
			t.Errorf("participant task ID = %q, want %q", participant.TaskID, task.ID)
		}
	}
}

func TestTaskServiceCreateDeniesUnknownCreator(t *testing.T) {
	taskService := service.NewTaskService(
		memory.NewTaskRepository(),
		memory.NewChatMemberRepository(),
		func() (domain.TaskID, error) { return "task-1", nil },
		time.Now,
	)

	_, err := taskService.Create(context.Background(), service.CreateTaskCommand{
		ChatID:    -100123,
		CreatorID: 1,
		Title:     "Чужая задача",
	})
	if !errors.Is(err, service.ErrTaskAccessDenied) {
		t.Fatalf("Create() error = %v, want %v", err, service.ErrTaskAccessDenied)
	}
}

func TestTaskServiceCreateRejectsInactiveParticipant(t *testing.T) {
	ctx := context.Background()
	taskRepo := memory.NewTaskRepository()
	memberRepo := memory.NewChatMemberRepository()
	upsertActiveMember(t, memberRepo, -100123, 1)

	leftMember := domain.ChatMember{
		ChatID: -100123,
		UserID: 2,
		Status: domain.ChatMemberStatusLeft,
	}
	if err := memberRepo.Upsert(ctx, leftMember); err != nil {
		t.Fatalf("Upsert() returned an unexpected error: %v", err)
	}

	taskService := service.NewTaskService(
		taskRepo,
		memberRepo,
		func() (domain.TaskID, error) { return "task-1", nil },
		time.Now,
	)

	_, err := taskService.Create(ctx, service.CreateTaskCommand{
		ChatID:         -100123,
		CreatorID:      1,
		Title:          "Общая задача",
		ParticipantIDs: []domain.UserID{2},
	})
	if !errors.Is(err, service.ErrParticipantNotInChat) {
		t.Fatalf("Create() error = %v, want %v", err, service.ErrParticipantNotInChat)
	}
}

func TestTaskServiceListsChatAndSelectedUserTasks(t *testing.T) {
	ctx := context.Background()
	taskRepo := memory.NewTaskRepository()
	memberRepo := memory.NewChatMemberRepository()
	for _, userID := range []domain.UserID{1, 2, 3} {
		upsertActiveMember(t, memberRepo, -100123, userID)
	}

	ids := []domain.TaskID{"task-1", "task-2", "task-3"}
	nextID := 0
	taskService := service.NewTaskService(
		taskRepo,
		memberRepo,
		func() (domain.TaskID, error) {
			id := ids[nextID]
			nextID++
			return id, nil
		},
		time.Now,
	)

	createTaskThroughService(t, taskService, service.CreateTaskCommand{
		ChatID:         -100123,
		CreatorID:      1,
		Title:          "Задача от user1 для user2",
		ParticipantIDs: []domain.UserID{2},
	})
	createTaskThroughService(t, taskService, service.CreateTaskCommand{
		ChatID:    -100123,
		CreatorID: 2,
		Title:     "Задача от user2",
	})
	createTaskThroughService(t, taskService, service.CreateTaskCommand{
		ChatID:    -100123,
		CreatorID: 1,
		Title:     "Только user1",
	})

	chatTasks, err := taskService.ListChatTasks(ctx, 3, -100123)
	if err != nil {
		t.Fatalf("ListChatTasks() returned an unexpected error: %v", err)
	}
	if len(chatTasks) != 3 {
		t.Errorf("ListChatTasks() returned %d tasks, want 3", len(chatTasks))
	}

	userTasks, err := taskService.ListUserTasks(ctx, 3, 2, -100123)
	if err != nil {
		t.Fatalf("ListUserTasks() returned an unexpected error: %v", err)
	}
	if len(userTasks) != 2 {
		t.Fatalf("ListUserTasks() returned %d tasks, want 2", len(userTasks))
	}
	if userTasks[0].ID != "task-1" || userTasks[1].ID != "task-2" {
		t.Errorf("ListUserTasks() IDs = %q, %q; want task-1, task-2", userTasks[0].ID, userTasks[1].ID)
	}
}

func TestTaskServiceListDeniesNonMember(t *testing.T) {
	taskService := service.NewTaskService(
		memory.NewTaskRepository(),
		memory.NewChatMemberRepository(),
		nil,
		nil,
	)

	_, err := taskService.ListChatTasks(context.Background(), 99, -100123)
	if !errors.Is(err, service.ErrTaskAccessDenied) {
		t.Fatalf("ListChatTasks() error = %v, want %v", err, service.ErrTaskAccessDenied)
	}
}

func TestGenerateTaskIDReturnsUUIDVersion4(t *testing.T) {
	first, err := service.GenerateTaskID()
	if err != nil {
		t.Fatalf("first GenerateTaskID() returned an unexpected error: %v", err)
	}
	second, err := service.GenerateTaskID()
	if err != nil {
		t.Fatalf("second GenerateTaskID() returned an unexpected error: %v", err)
	}

	if first == second {
		t.Fatal("two generated task IDs must be different")
	}
	id := string(first)
	if len(id) != 36 || id[14] != '4' || !strings.ContainsRune("89ab", rune(id[19])) {
		t.Errorf("GenerateTaskID() = %q, want a UUID version 4", id)
	}
}

func upsertActiveMember(
	t *testing.T,
	repo *memory.ChatMemberRepository,
	chatID domain.ChatID,
	userID domain.UserID,
) {
	t.Helper()

	if err := repo.Upsert(context.Background(), activeMember(chatID, userID)); err != nil {
		t.Fatalf("Upsert() returned an unexpected error: %v", err)
	}
}

func createTaskThroughService(
	t *testing.T,
	taskService *service.TaskService,
	command service.CreateTaskCommand,
) domain.Task {
	t.Helper()

	task, err := taskService.Create(context.Background(), command)
	if err != nil {
		t.Fatalf("Create() returned an unexpected error: %v", err)
	}

	return task
}
