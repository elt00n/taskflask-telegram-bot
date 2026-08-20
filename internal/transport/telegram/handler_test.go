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

const testTaskID domain.TaskID = "11111111-1111-4111-8111-111111111111"

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
	if !strings.Contains(listed, "📋 Задачи") ||
		!strings.Contains(listed, "Позвонить врачу") ||
		!strings.Contains(listed, "11111111") {
		t.Errorf("listTasks() response = %q", listed)
	}
}

func TestUserErrorMessageExplainsPastDeadline(t *testing.T) {
	got := userErrorMessage(domain.ErrDeadlineInPast)
	want := "Указанное время уже прошло. Выберите будущее время."
	if got != want {
		t.Errorf("userErrorMessage() = %q, want %q", got, want)
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

func TestHandlerEditsTaskWithCommands(t *testing.T) {
	handler := newTestHandler(t)
	message := testMessage(1, "user1", "/task Исходная задача")
	if err := handler.observeSender(context.Background(), message); err != nil {
		t.Fatalf("observeSender() returned an unexpected error: %v", err)
	}
	if _, err := handler.createTask(context.Background(), message); err != nil {
		t.Fatalf("createTask() returned an unexpected error: %v", err)
	}

	message.Text = "/rename 11111111 Новое название"
	if response, err := handler.renameTask(context.Background(), message); err != nil ||
		!strings.Contains(response, "Новое название") {
		t.Fatalf("renameTask() = %q, %v", response, err)
	}

	message.Text = "/priority 11111111 критический"
	if response, err := handler.changeTaskPriority(context.Background(), message); err != nil ||
		!strings.Contains(response, "🔴") {
		t.Fatalf("changeTaskPriority() = %q, %v", response, err)
	}

	message.Text = "/deadline 11111111 25 августа 2026 18:00"
	if response, err := handler.changeTaskDeadline(context.Background(), message); err != nil ||
		!strings.Contains(response, "25.08.2026 18:00") {
		t.Fatalf("changeTaskDeadline() = %q, %v", response, err)
	}

	message.Text = "/deadline 11111111 -"
	if response, err := handler.changeTaskDeadline(context.Background(), message); err != nil ||
		!strings.Contains(response, "Без срока") {
		t.Fatalf("clear deadline = %q, %v", response, err)
	}

	message.Text = "/done 11111111"
	if response, err := handler.completeTask(context.Background(), message); err != nil ||
		!strings.Contains(response, "Задача завершена") {
		t.Fatalf("completeTask() = %q, %v", response, err)
	}

	message.Text = "/delete 11111111"
	if response, err := handler.deleteTask(context.Background(), message); err != nil ||
		!strings.Contains(response, "удалена") {
		t.Fatalf("deleteTask() = %q, %v", response, err)
	}
}

func TestHandlerExecutesTaskCallbacks(t *testing.T) {
	handler := newTestHandler(t)
	message := testMessage(1, "user1", "/task Задача с кнопками")
	if err := handler.observeSender(context.Background(), message); err != nil {
		t.Fatalf("observeSender() returned an unexpected error: %v", err)
	}
	if _, err := handler.createTask(context.Background(), message); err != nil {
		t.Fatalf("createTask() returned an unexpected error: %v", err)
	}

	query := &models.CallbackQuery{
		From: *message.From,
		Message: models.MaybeInaccessibleMessage{
			Type:    models.MaybeInaccessibleMessageTypeMessage,
			Message: message,
		},
		Data: taskCallbackData("priority", testTaskID),
	}
	result, err := handler.executeTaskCallback(context.Background(), query)
	if err != nil || !strings.Contains(result.text, "Важность изменена") {
		t.Fatalf("priority callback = %#v, %v", result, err)
	}

	query.Data = taskCallbackData("done", testTaskID)
	result, err = handler.executeTaskCallback(context.Background(), query)
	if err != nil || !strings.Contains(result.text, "Задача завершена") {
		t.Fatalf("done callback = %#v, %v", result, err)
	}

	query.Data = taskCallbackData("delete", testTaskID)
	result, err = handler.executeTaskCallback(context.Background(), query)
	if err != nil || !result.markupOnly {
		t.Fatalf("delete callback = %#v, %v", result, err)
	}

	query.Data = taskCallbackData("delete-confirm", testTaskID)
	result, err = handler.executeTaskCallback(context.Background(), query)
	if err != nil || !strings.Contains(result.text, "удалена") {
		t.Fatalf("delete confirmation = %#v, %v", result, err)
	}
}

func TestTaskCallbackDataFitsTelegramLimit(t *testing.T) {
	for _, action := range []string{"done", "priority", "edit", "delete", "delete-confirm", "cancel"} {
		data := taskCallbackData(action, testTaskID)
		if len(data) > 64 {
			t.Errorf("callback %q has %d bytes, Telegram limit is 64", data, len(data))
		}
	}
}

func TestParseTaskCallbackDataRejectsMalformedValue(t *testing.T) {
	for _, data := range []string{"", "task:", "task:done", "other:done:task-id"} {
		if _, _, valid := parseTaskCallbackData(data); valid {
			t.Errorf("parseTaskCallbackData(%q) unexpectedly succeeded", data)
		}
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
	userRepo := memory.NewUserRepository(memberRepo)
	chatRepo := memory.NewChatRepository()
	fixedNow := time.Date(2026, time.August, 20, 12, 0, 0, 0, location)
	taskService := service.NewTaskService(
		taskRepo,
		memberRepo,
		func() (domain.TaskID, error) { return testTaskID, nil },
		func() time.Time { return fixedNow },
	)

	return NewHandler(
		taskService,
		memberRepo,
		userRepo,
		chatRepo,
		location,
		func() time.Time { return fixedNow },
	)
}

func testMessage(userID int64, username string, text string) *models.Message {
	return &models.Message{
		Chat: models.Chat{
			ID:    -100123,
			Type:  models.ChatTypeGroup,
			Title: "Тестовый чат",
		},
		From: &models.User{
			ID:        userID,
			Username:  username,
			FirstName: username,
		},
		Text: text,
	}
}
