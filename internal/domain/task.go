// Package domain содержит основные сущности и бизнес-правила приложения.
// Он не зависит от Telegram, PostgreSQL или других внешних систем.
package domain

import (
	"errors"
	"strings"
	"time"
)

const DefaultEventDuration = time.Hour

// TaskID — отдельный тип для идентификатора задачи.
// Благодаря этому обычную строку сложнее случайно перепутать с ID другой сущности.
type TaskID string

// TaskStatus описывает текущее состояние задачи.
type TaskStatus string

const (
	TaskStatusNew        TaskStatus = "new"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// TaskPriority задаёт относительную важность задачи.
type TaskPriority uint8

const (
	TaskPriorityLow TaskPriority = iota + 1
	TaskPriorityNormal
	TaskPriorityHigh
	TaskPriorityCritical
)

// TaskKind позволяет отличить задачу со сроком от события с временем начала.
type TaskKind string

const (
	TaskKindTask  TaskKind = "task"
	TaskKindEvent TaskKind = "event"
)

var (
	ErrTaskIDRequired       = errors.New("task ID is required")
	ErrChatIDRequired       = errors.New("chat ID is required")
	ErrCreatorRequired      = errors.New("creator ID is required")
	ErrTaskTitleRequired    = errors.New("task title is required")
	ErrStartTimeRequired    = errors.New("start time is required when end time is set")
	ErrInvalidTimeRange     = errors.New("end time must be after start time")
	ErrDeadlineInPast       = errors.New("deadline must not be in the past")
	ErrCreationTimeRequired = errors.New("creation time is required")
)

// Task представляет задачу или календарное событие.
// Указатели *time.Time означают, что соответствующее время может отсутствовать.
type Task struct {
	ID          TaskID
	ChatID      ChatID
	CreatorID   UserID
	Title       string
	Description string
	Kind        TaskKind
	Status      TaskStatus
	Priority    TaskPriority
	StartAt     *time.Time
	EndAt       *time.Time
	Deadline    *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// NewTaskParams группирует данные, из которых создаётся новая задача.
// Необязательные значения имеют нулевое значение или равны nil.
type NewTaskParams struct {
	ID          TaskID
	ChatID      ChatID
	CreatorID   UserID
	Title       string
	Description string
	Priority    TaskPriority
	StartAt     *time.Time
	EndAt       *time.Time
	Deadline    *time.Time
}

// NewTask проверяет входные данные и применяет значения по умолчанию.
// Параметр now передаётся снаружи, чтобы тесты не зависели от часов компьютера.
func NewTask(params NewTaskParams, now time.Time) (Task, error) {
	if strings.TrimSpace(string(params.ID)) == "" {
		return Task{}, ErrTaskIDRequired
	}
	if !params.ChatID.IsValid() {
		return Task{}, ErrChatIDRequired
	}
	if !params.CreatorID.IsValid() {
		return Task{}, ErrCreatorRequired
	}

	title := strings.TrimSpace(params.Title)
	if title == "" {
		return Task{}, ErrTaskTitleRequired
	}
	if now.IsZero() {
		return Task{}, ErrCreationTimeRequired
	}
	if params.StartAt == nil && params.EndAt != nil {
		return Task{}, ErrStartTimeRequired
	}

	startAt := timeInUTC(params.StartAt)
	endAt := timeInUTC(params.EndAt)
	deadline := timeInUTC(params.Deadline)
	now = now.UTC()
	if deadline != nil && deadline.Before(now) {
		return Task{}, ErrDeadlineInPast
	}
	kind := TaskKindTask
	priority := params.Priority
	if priority == 0 {
		priority = TaskPriorityNormal
	}
	if !priority.IsValid() {
		return Task{}, ErrInvalidTaskPriority
	}

	if startAt != nil {
		kind = TaskKindEvent
		if endAt == nil {
			defaultEnd := startAt.Add(DefaultEventDuration)
			endAt = &defaultEnd
		}
		if !endAt.After(*startAt) {
			return Task{}, ErrInvalidTimeRange
		}
	}

	return Task{
		ID:          params.ID,
		ChatID:      params.ChatID,
		CreatorID:   params.CreatorID,
		Title:       title,
		Description: strings.TrimSpace(params.Description),
		Kind:        kind,
		Status:      TaskStatusNew,
		Priority:    priority,
		StartAt:     startAt,
		EndAt:       endAt,
		Deadline:    deadline,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func timeInUTC(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}
