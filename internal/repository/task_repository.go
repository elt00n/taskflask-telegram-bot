// Package repository описывает контракты хранения данных приложения.
package repository

import (
	"context"
	"errors"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskAlreadyExists = errors.New("task already exists")
)

// TaskFilter задаёт условия получения списка задач.
// ChatID обязателен: задачи разных чатов никогда не смешиваются.
type TaskFilter struct {
	ChatID domain.ChatID
	UserID *domain.UserID
}

// TaskRepository скрывает конкретный способ хранения задач.
type TaskRepository interface {
	Create(
		ctx context.Context,
		task domain.Task,
		participants []domain.TaskParticipant,
	) error
	Get(
		ctx context.Context,
		taskID domain.TaskID,
	) (domain.Task, []domain.TaskParticipant, error)
	Update(ctx context.Context, task domain.Task) error
	List(ctx context.Context, filter TaskFilter) ([]domain.Task, error)
}
