package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/parser"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
)

var (
	errRenameUsage   = errors.New("invalid rename command")
	errPriorityUsage = errors.New("invalid priority command")
	errDeadlineUsage = errors.New("invalid deadline command")
	errDoneUsage     = errors.New("invalid done command")
	errDeleteUsage   = errors.New("invalid delete command")
)

func (handler *Handler) renameTask(
	ctx context.Context,
	message *models.Message,
) (string, error) {
	reference, title, found := strings.Cut(commandArguments(message.Text), " ")
	if !found || strings.TrimSpace(title) == "" {
		return "", errRenameUsage
	}
	taskID, err := handler.resolveTaskID(ctx, message, reference)
	if err != nil {
		return "", err
	}
	task, err := handler.tasks.Rename(ctx, domain.UserID(message.From.ID), taskID, title)
	if err != nil {
		return "", err
	}
	return formatUpdatedTask("✏️ Название изменено", task, handler.location), nil
}

func (handler *Handler) changeTaskPriority(
	ctx context.Context,
	message *models.Message,
) (string, error) {
	reference, value, found := strings.Cut(commandArguments(message.Text), " ")
	if !found || len(strings.Fields(value)) != 1 {
		return "", errPriorityUsage
	}
	priority, valid := parsePriority(value)
	if !valid {
		return "", errPriorityUsage
	}
	taskID, err := handler.resolveTaskID(ctx, message, reference)
	if err != nil {
		return "", err
	}
	task, err := handler.tasks.ChangePriority(
		ctx,
		domain.UserID(message.From.ID),
		taskID,
		priority,
	)
	if err != nil {
		return "", err
	}
	return formatUpdatedTask("⚡ Важность изменена", task, handler.location), nil
}

func (handler *Handler) changeTaskDeadline(
	ctx context.Context,
	message *models.Message,
) (string, error) {
	reference, value, found := strings.Cut(commandArguments(message.Text), " ")
	if !found || strings.TrimSpace(value) == "" {
		return "", errDeadlineUsage
	}

	var deadline *time.Time
	if !isClearValue(value) {
		parsed, err := parser.ParseDeadline(value, handler.clock(), handler.location)
		if err != nil {
			return "", err
		}
		deadline = parsed
	}
	taskID, err := handler.resolveTaskID(ctx, message, reference)
	if err != nil {
		return "", err
	}
	task, err := handler.tasks.ChangeDeadline(
		ctx,
		domain.UserID(message.From.ID),
		taskID,
		deadline,
	)
	if err != nil {
		return "", err
	}
	return formatUpdatedTask("📅 Дедлайн изменён", task, handler.location), nil
}

func (handler *Handler) completeTask(
	ctx context.Context,
	message *models.Message,
) (string, error) {
	reference, valid := singleArgument(message.Text)
	if !valid {
		return "", errDoneUsage
	}
	taskID, err := handler.resolveTaskID(ctx, message, reference)
	if err != nil {
		return "", err
	}
	task, err := handler.tasks.Complete(ctx, domain.UserID(message.From.ID), taskID)
	if err != nil {
		return "", err
	}
	return formatUpdatedTask("✅ Задача завершена", task, handler.location), nil
}

func (handler *Handler) deleteTask(
	ctx context.Context,
	message *models.Message,
) (string, error) {
	reference, valid := singleArgument(message.Text)
	if !valid {
		return "", errDeleteUsage
	}
	taskID, err := handler.resolveTaskID(ctx, message, reference)
	if err != nil {
		return "", err
	}
	task, err := handler.tasks.Delete(ctx, domain.UserID(message.From.ID), taskID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("🗑 Задача «%s» удалена.", task.Title), nil
}

func (handler *Handler) resolveTaskID(
	ctx context.Context,
	message *models.Message,
	reference string,
) (domain.TaskID, error) {
	reference = strings.TrimSpace(reference)
	if len(reference) < 8 {
		return "", repository.ErrTaskNotFound
	}
	return handler.tasks.ResolveTaskID(
		ctx,
		domain.UserID(message.From.ID),
		domain.ChatID(message.Chat.ID),
		reference,
	)
}

func parsePriority(value string) (domain.TaskPriority, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "низкий", "low", "1":
		return domain.TaskPriorityLow, true
	case "обычный", "normal", "2":
		return domain.TaskPriorityNormal, true
	case "важный", "высокий", "high", "3":
		return domain.TaskPriorityHigh, true
	case "критический", "critical", "4":
		return domain.TaskPriorityCritical, true
	default:
		return 0, false
	}
}

func singleArgument(text string) (string, bool) {
	fields := strings.Fields(commandArguments(text))
	return firstField(fields), len(fields) == 1
}

func firstField(fields []string) string {
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func isClearValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "-", "нет", "очистить", "clear":
		return true
	default:
		return false
	}
}
