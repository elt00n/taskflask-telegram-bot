package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
)

func TestTaskEditingMethods(t *testing.T) {
	createdAt := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	task := newEditableTask(t, createdAt)

	if err := task.Rename("  Новое название  ", updatedAt); err != nil {
		t.Fatalf("Rename() returned an unexpected error: %v", err)
	}
	if task.Title != "Новое название" {
		t.Errorf("Title = %q, want %q", task.Title, "Новое название")
	}

	if err := task.SetDescription("  Новое описание  ", updatedAt); err != nil {
		t.Fatalf("SetDescription() returned an unexpected error: %v", err)
	}
	if task.Description != "Новое описание" {
		t.Errorf("Description = %q, want %q", task.Description, "Новое описание")
	}

	if err := task.SetPriority(domain.TaskPriorityHigh, updatedAt); err != nil {
		t.Fatalf("SetPriority() returned an unexpected error: %v", err)
	}
	if task.Priority != domain.TaskPriorityHigh {
		t.Errorf("Priority = %d, want %d", task.Priority, domain.TaskPriorityHigh)
	}
	if !task.UpdatedAt.Equal(updatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", task.UpdatedAt, updatedAt)
	}
}

func TestTaskSetScheduleUsesDefaultDurationAndCanClearSchedule(t *testing.T) {
	createdAt := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
	moscow := time.FixedZone("MSK", 3*60*60)
	start := time.Date(2026, time.August, 21, 18, 0, 0, 0, moscow)
	task := newEditableTask(t, createdAt)

	if err := task.SetSchedule(&start, nil, createdAt.Add(time.Hour)); err != nil {
		t.Fatalf("SetSchedule() returned an unexpected error: %v", err)
	}
	if task.Kind != domain.TaskKindEvent {
		t.Errorf("Kind = %q, want %q", task.Kind, domain.TaskKindEvent)
	}
	if task.StartAt == nil || !task.StartAt.Equal(start.UTC()) {
		t.Errorf("StartAt = %v, want %v", task.StartAt, start.UTC())
	}
	wantEnd := start.UTC().Add(domain.DefaultEventDuration)
	if task.EndAt == nil || !task.EndAt.Equal(wantEnd) {
		t.Errorf("EndAt = %v, want %v", task.EndAt, wantEnd)
	}

	if err := task.SetSchedule(nil, nil, createdAt.Add(2*time.Hour)); err != nil {
		t.Fatalf("clearing SetSchedule() returned an unexpected error: %v", err)
	}
	if task.Kind != domain.TaskKindTask || task.StartAt != nil || task.EndAt != nil {
		t.Error("clearing schedule must turn the event back into an unscheduled task")
	}
}

func TestTaskSetDeadlineNormalizesTimeAndCanClearDeadline(t *testing.T) {
	createdAt := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
	deadline := time.Date(2026, time.August, 22, 18, 0, 0, 0, time.FixedZone("MSK", 3*60*60))
	task := newEditableTask(t, createdAt)

	if err := task.SetDeadline(&deadline, createdAt.Add(time.Hour)); err != nil {
		t.Fatalf("SetDeadline() returned an unexpected error: %v", err)
	}
	if task.Deadline == nil || !task.Deadline.Equal(deadline.UTC()) {
		t.Errorf("Deadline = %v, want %v", task.Deadline, deadline.UTC())
	}

	if err := task.SetDeadline(nil, createdAt.Add(2*time.Hour)); err != nil {
		t.Fatalf("clearing SetDeadline() returned an unexpected error: %v", err)
	}
	if task.Deadline != nil {
		t.Errorf("Deadline = %v, want nil", task.Deadline)
	}
}

func TestTaskSetDeadlineRejectsPastValue(t *testing.T) {
	createdAt := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
	updateTime := createdAt.Add(time.Hour)
	pastDeadline := updateTime.Add(-time.Nanosecond)
	task := newEditableTask(t, createdAt)

	err := task.SetDeadline(&pastDeadline, updateTime)
	if !errors.Is(err, domain.ErrDeadlineInPast) {
		t.Fatalf("SetDeadline() error = %v, want %v", err, domain.ErrDeadlineInPast)
	}
	if task.Deadline != nil {
		t.Errorf("Deadline changed after failed update: %v", task.Deadline)
	}
	if !task.UpdatedAt.Equal(createdAt) {
		t.Errorf("UpdatedAt = %v, want unchanged %v", task.UpdatedAt, createdAt)
	}
}

func TestTaskCompleteAndDelete(t *testing.T) {
	createdAt := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
	completedAt := createdAt.Add(time.Hour)
	deletedAt := createdAt.Add(2 * time.Hour)
	task := newEditableTask(t, createdAt)

	if err := task.Complete(completedAt); err != nil {
		t.Fatalf("Complete() returned an unexpected error: %v", err)
	}
	if task.Status != domain.TaskStatusDone {
		t.Errorf("Status = %q, want %q", task.Status, domain.TaskStatusDone)
	}

	if err := task.Delete(deletedAt); err != nil {
		t.Fatalf("Delete() returned an unexpected error: %v", err)
	}
	if !task.IsDeleted() || task.DeletedAt == nil || !task.DeletedAt.Equal(deletedAt) {
		t.Errorf("DeletedAt = %v, want %v", task.DeletedAt, deletedAt)
	}
	if err := task.Rename("Нельзя изменить", deletedAt.Add(time.Hour)); !errors.Is(err, domain.ErrTaskDeleted) {
		t.Errorf("Rename() after deletion error = %v, want %v", err, domain.ErrTaskDeleted)
	}
}

func TestTaskRejectsInvalidChanges(t *testing.T) {
	createdAt := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
	task := newEditableTask(t, createdAt)
	start := createdAt.Add(2 * time.Hour)
	endBeforeStart := start.Add(-time.Minute)

	if err := task.Rename("   ", createdAt.Add(time.Hour)); !errors.Is(err, domain.ErrTaskTitleRequired) {
		t.Errorf("Rename() error = %v, want %v", err, domain.ErrTaskTitleRequired)
	}
	if err := task.SetPriority(99, createdAt.Add(time.Hour)); !errors.Is(err, domain.ErrInvalidTaskPriority) {
		t.Errorf("SetPriority() error = %v, want %v", err, domain.ErrInvalidTaskPriority)
	}
	if err := task.SetSchedule(&start, &endBeforeStart, createdAt.Add(time.Hour)); !errors.Is(err, domain.ErrInvalidTimeRange) {
		t.Errorf("SetSchedule() error = %v, want %v", err, domain.ErrInvalidTimeRange)
	}
	if err := task.Complete(time.Time{}); !errors.Is(err, domain.ErrTaskUpdateTimeRequired) {
		t.Errorf("Complete() error = %v, want %v", err, domain.ErrTaskUpdateTimeRequired)
	}
}

func newEditableTask(t *testing.T, createdAt time.Time) domain.Task {
	t.Helper()

	task, err := domain.NewTask(domain.NewTaskParams{
		ID:        "task-1",
		ChatID:    -100123,
		CreatorID: 1,
		Title:     "Исходная задача",
	}, createdAt)
	if err != nil {
		t.Fatalf("NewTask() returned an unexpected error: %v", err)
	}

	return task
}
