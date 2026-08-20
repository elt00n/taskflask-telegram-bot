package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrTaskDeleted            = errors.New("task is deleted")
	ErrInvalidTaskPriority    = errors.New("invalid task priority")
	ErrTaskUpdateTimeRequired = errors.New("task update time is required")
)

// IsValid сообщает, является ли важность одним из поддерживаемых значений.
func (priority TaskPriority) IsValid() bool {
	return priority >= TaskPriorityLow && priority <= TaskPriorityCritical
}

// IsDeleted сообщает, была ли задача мягко удалена.
func (task Task) IsDeleted() bool {
	return task.DeletedAt != nil
}

// Rename изменяет название задачи.
func (task *Task) Rename(title string, now time.Time) error {
	if err := task.prepareUpdate(now); err != nil {
		return err
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return ErrTaskTitleRequired
	}

	task.Title = title
	task.UpdatedAt = now.UTC()
	return nil
}

// SetDescription изменяет или очищает описание задачи.
func (task *Task) SetDescription(description string, now time.Time) error {
	if err := task.prepareUpdate(now); err != nil {
		return err
	}

	task.Description = strings.TrimSpace(description)
	task.UpdatedAt = now.UTC()
	return nil
}

// SetPriority изменяет важность задачи.
func (task *Task) SetPriority(priority TaskPriority, now time.Time) error {
	if err := task.prepareUpdate(now); err != nil {
		return err
	}
	if !priority.IsValid() {
		return ErrInvalidTaskPriority
	}

	task.Priority = priority
	task.UpdatedAt = now.UTC()
	return nil
}

// SetSchedule устанавливает или очищает время начала и окончания.
// Событие без явно заданного конца по умолчанию длится один час.
func (task *Task) SetSchedule(startAt *time.Time, endAt *time.Time, now time.Time) error {
	if err := task.prepareUpdate(now); err != nil {
		return err
	}
	if startAt == nil && endAt != nil {
		return ErrStartTimeRequired
	}

	normalizedStart := timeInUTC(startAt)
	normalizedEnd := timeInUTC(endAt)
	kind := TaskKindTask

	if normalizedStart != nil {
		kind = TaskKindEvent
		if normalizedEnd == nil {
			defaultEnd := normalizedStart.Add(DefaultEventDuration)
			normalizedEnd = &defaultEnd
		}
		if !normalizedEnd.After(*normalizedStart) {
			return ErrInvalidTimeRange
		}
	}

	task.StartAt = normalizedStart
	task.EndAt = normalizedEnd
	task.Kind = kind
	task.UpdatedAt = now.UTC()
	return nil
}

// SetDeadline устанавливает или очищает дедлайн задачи.
func (task *Task) SetDeadline(deadline *time.Time, now time.Time) error {
	if err := task.prepareUpdate(now); err != nil {
		return err
	}

	normalizedDeadline := timeInUTC(deadline)
	if normalizedDeadline != nil && normalizedDeadline.Before(now.UTC()) {
		return ErrDeadlineInPast
	}

	task.Deadline = normalizedDeadline
	task.UpdatedAt = now.UTC()
	return nil
}

// Complete переводит задачу в завершённое состояние.
func (task *Task) Complete(now time.Time) error {
	if err := task.prepareUpdate(now); err != nil {
		return err
	}

	task.Status = TaskStatusDone
	task.UpdatedAt = now.UTC()
	return nil
}

// Delete мягко удаляет задачу. Повторный вызов безопасен и ничего не меняет.
func (task *Task) Delete(now time.Time) error {
	if task.IsDeleted() {
		return nil
	}
	if now.IsZero() {
		return ErrTaskUpdateTimeRequired
	}

	deletedAt := now.UTC()
	task.DeletedAt = &deletedAt
	task.UpdatedAt = deletedAt
	return nil
}

func (task Task) prepareUpdate(now time.Time) error {
	if task.IsDeleted() {
		return ErrTaskDeleted
	}
	if now.IsZero() {
		return ErrTaskUpdateTimeRequired
	}

	return nil
}
