package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository/memory"
)

func TestTaskRepositoryCreateAndGet(t *testing.T) {
	repo := memory.NewTaskRepository()
	task := newTestTask(t, "task-1", -100123, 1, time.Now())
	participants := []domain.TaskParticipant{
		{TaskID: task.ID, UserID: 2, Role: domain.TaskParticipantRoleAssignee},
	}

	if err := repo.Create(context.Background(), task, participants); err != nil {
		t.Fatalf("Create() returned an unexpected error: %v", err)
	}

	participants[0].UserID = 999
	gotTask, gotParticipants, err := repo.Get(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("Get() returned an unexpected error: %v", err)
	}
	if gotTask.ID != task.ID {
		t.Errorf("Get() task ID = %q, want %q", gotTask.ID, task.ID)
	}
	if len(gotParticipants) != 1 || gotParticipants[0].UserID != 2 {
		t.Fatalf("Get() participants = %#v, want the original participant", gotParticipants)
	}

	gotParticipants[0].UserID = 888
	_, secondParticipants, err := repo.Get(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("second Get() returned an unexpected error: %v", err)
	}
	if secondParticipants[0].UserID != 2 {
		t.Error("a caller must not be able to mutate repository participants")
	}
}

func TestTaskRepositoryRejectsDuplicateID(t *testing.T) {
	repo := memory.NewTaskRepository()
	task := newTestTask(t, "task-1", -100123, 1, time.Now())

	if err := repo.Create(context.Background(), task, nil); err != nil {
		t.Fatalf("first Create() returned an unexpected error: %v", err)
	}
	if err := repo.Create(context.Background(), task, nil); !errors.Is(err, repository.ErrTaskAlreadyExists) {
		t.Fatalf("second Create() error = %v, want %v", err, repository.ErrTaskAlreadyExists)
	}
}

func TestTaskRepositoryListByChatAndUser(t *testing.T) {
	repo := memory.NewTaskRepository()
	baseTime := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	task1 := newTestTask(t, "task-1", -100123, 1, baseTime)
	task2 := newTestTask(t, "task-2", -100123, 2, baseTime.Add(time.Minute))
	task3 := newTestTask(t, "task-3", -200456, 1, baseTime.Add(2*time.Minute))

	mustCreateTask(t, repo, task1, nil)
	mustCreateTask(t, repo, task2, []domain.TaskParticipant{
		{TaskID: task2.ID, UserID: 1, Role: domain.TaskParticipantRoleAssignee},
	})
	mustCreateTask(t, repo, task3, nil)

	chatTasks, err := repo.List(context.Background(), repository.TaskFilter{ChatID: -100123})
	if err != nil {
		t.Fatalf("List() by chat returned an unexpected error: %v", err)
	}
	if len(chatTasks) != 2 || chatTasks[0].ID != task1.ID || chatTasks[1].ID != task2.ID {
		t.Errorf("List() by chat = %#v, want task-1 followed by task-2", chatTasks)
	}

	userID := domain.UserID(1)
	userTasks, err := repo.List(context.Background(), repository.TaskFilter{
		ChatID: -100123,
		UserID: &userID,
	})
	if err != nil {
		t.Fatalf("List() by user returned an unexpected error: %v", err)
	}
	if len(userTasks) != 2 {
		t.Errorf("List() by user returned %d tasks, want 2", len(userTasks))
	}
}

func TestTaskRepositoryHonorsCancelledContext(t *testing.T) {
	repo := memory.NewTaskRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.List(ctx, repository.TaskFilter{ChatID: -100123})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("List() error = %v, want %v", err, context.Canceled)
	}
}

func TestChatMemberRepositoryUpsertAndGet(t *testing.T) {
	repo := memory.NewChatMemberRepository()
	member := domain.ChatMember{
		ChatID: -100123,
		UserID: 1,
		Status: domain.ChatMemberStatusMember,
	}

	if err := repo.Upsert(context.Background(), member); err != nil {
		t.Fatalf("Upsert() returned an unexpected error: %v", err)
	}
	got, err := repo.Get(context.Background(), member.ChatID, member.UserID)
	if err != nil {
		t.Fatalf("Get() returned an unexpected error: %v", err)
	}
	if got != member {
		t.Errorf("Get() = %#v, want %#v", got, member)
	}

	member.Status = domain.ChatMemberStatusLeft
	if err := repo.Upsert(context.Background(), member); err != nil {
		t.Fatalf("second Upsert() returned an unexpected error: %v", err)
	}
	got, err = repo.Get(context.Background(), member.ChatID, member.UserID)
	if err != nil {
		t.Fatalf("second Get() returned an unexpected error: %v", err)
	}
	if got.Status != domain.ChatMemberStatusLeft {
		t.Errorf("updated status = %q, want %q", got.Status, domain.ChatMemberStatusLeft)
	}
}

func TestChatMemberRepositoryReturnsNotFound(t *testing.T) {
	repo := memory.NewChatMemberRepository()

	_, err := repo.Get(context.Background(), -100123, 1)
	if !errors.Is(err, repository.ErrChatMemberNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, repository.ErrChatMemberNotFound)
	}
}

func newTestTask(
	t *testing.T,
	id domain.TaskID,
	chatID domain.ChatID,
	creatorID domain.UserID,
	createdAt time.Time,
) domain.Task {
	t.Helper()

	task, err := domain.NewTask(domain.NewTaskParams{
		ID:        id,
		ChatID:    chatID,
		CreatorID: creatorID,
		Title:     "Задача " + string(id),
	}, createdAt)
	if err != nil {
		t.Fatalf("NewTask() returned an unexpected error: %v", err)
	}

	return task
}

func mustCreateTask(
	t *testing.T,
	repo *memory.TaskRepository,
	task domain.Task,
	participants []domain.TaskParticipant,
) {
	t.Helper()

	if err := repo.Create(context.Background(), task, participants); err != nil {
		t.Fatalf("Create() returned an unexpected error: %v", err)
	}
}
