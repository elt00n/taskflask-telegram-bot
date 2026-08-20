package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository/memory"
	"github.com/elt00n/taskflask-telegram-bot/internal/service"
)

func TestHandlerCreatesAndListsTask(t *testing.T) {
	handler := newTestHandler(t)
	message := testMessage(1, "user1", "/task завтра 10:00 Позвонить врачу важно")
	if err := handler.observeSender(context.Background(), message); err != nil {
		t.Fatalf("observeSender() returned an unexpected error: %v", err)
	}

	created, err := handler.createTask(context.Background(), message)
	if err != nil {
		t.Fatalf("createTask() returned an unexpected error: %v", err)
	}
	if !strings.Contains(created, "✅ Дело создано") ||
		!strings.Contains(created, "Позвонить врачу") ||
		!strings.Contains(created, "🟠") {
		t.Errorf("createTask() response = %q", created)
	}

	message.Text = "/tasks"
	listed, err := handler.listTasks(context.Background(), message)
	if err != nil {
		t.Fatalf("listTasks() returned an unexpected error: %v", err)
	}
	if !strings.Contains(listed, "📋 Задачи") || !strings.Contains(listed, "Позвонить врачу") {
		t.Errorf("listTasks() response = %q", listed)
	}
}

func TestHandlerResolvesKnownParticipantAndListsTheirTasks(t *testing.T) {
	handler := newTestHandler(t)
	user1 := testMessage(1, "user1", "/start")
	user2 := testMessage(2, "user2", "/help")
	if err := handler.observeSender(context.Background(), user1); err != nil {
		t.Fatalf("observeSender(user1) returned an unexpected error: %v", err)
	}
	if err := handler.observeSender(context.Background(), user2); err != nil {
		t.Fatalf("observeSender(user2) returned an unexpected error: %v", err)
	}

	user1.Text = "/task Общая задача @user2"
	if _, err := handler.createTask(context.Background(), user1); err != nil {
		t.Fatalf("createTask() returned an unexpected error: %v", err)
	}

	user1.Text = "/tasks @user2"
	listed, err := handler.listTasks(context.Background(), user1)
	if err != nil {
		t.Fatalf("listTasks() returned an unexpected error: %v", err)
	}
	if !strings.Contains(listed, "Общая задача") {
		t.Errorf("listTasks() response = %q, want the shared task", listed)
	}
}

func TestHandlerRejectsUnknownParticipant(t *testing.T) {
	handler := newTestHandler(t)
	message := testMessage(1, "user1", "/task Общая задача @unknown")
	if err := handler.observeSender(context.Background(), message); err != nil {
		t.Fatalf("observeSender() returned an unexpected error: %v", err)
	}

	_, err := handler.createTask(context.Background(), message)
	if !errors.Is(err, errUnknownParticipants) {
		t.Fatalf("createTask() error = %v, want %v", err, errUnknownParticipants)
	}
}

func TestMemberDirectoryUsesChatScope(t *testing.T) {
	directory := NewMemberDirectory()
	directory.Observe(-100123, 1, "@User1")

	ids, unknown := directory.Resolve(-100123, []string{"user1"})
	if len(ids) != 1 || ids[0] != 1 || len(unknown) != 0 {
		t.Fatalf("Resolve() = ids %#v, unknown %#v; want user 1", ids, unknown)
	}
	ids, unknown = directory.Resolve(-200456, []string{"user1"})
	if len(ids) != 0 || len(unknown) != 1 {
		t.Fatalf("Resolve() in another chat = ids %#v, unknown %#v", ids, unknown)
	}
}

func newTestHandler(t *testing.T) *Handler {
	t.Helper()

	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatalf("LoadLocation() returned an unexpected error: %v", err)
	}
	taskRepo := memory.NewTaskRepository()
	memberRepo := memory.NewChatMemberRepository()
	fixedNow := time.Date(2026, time.August, 20, 12, 0, 0, 0, location)
	taskService := service.NewTaskService(
		taskRepo,
		memberRepo,
		func() (domain.TaskID, error) { return "task-1", nil },
		func() time.Time { return fixedNow },
	)

	return NewHandler(
		taskService,
		memberRepo,
		NewMemberDirectory(),
		location,
		func() time.Time { return fixedNow },
	)
}

func testMessage(userID int64, username string, text string) *models.Message {
	return &models.Message{
		Chat: models.Chat{ID: -100123},
		From: &models.User{ID: userID, Username: username},
		Text: text,
	}
}
