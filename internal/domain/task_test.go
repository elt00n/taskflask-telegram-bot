package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
)

func TestNewTaskAppliesDefaults(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.FixedZone("MSK", 3*60*60))

	task, err := domain.NewTask(domain.NewTaskParams{
		ID:          "task-1",
		ChatID:      -100123,
		CreatorID:   42,
		Title:       "  Подготовить отчёт  ",
		Description: "  Данные за август  ",
	}, now)
	if err != nil {
		t.Fatalf("NewTask() returned an unexpected error: %v", err)
	}

	if task.Title != "Подготовить отчёт" {
		t.Errorf("Title = %q, want %q", task.Title, "Подготовить отчёт")
	}
	if task.Description != "Данные за август" {
		t.Errorf("Description = %q, want %q", task.Description, "Данные за август")
	}
	if task.Status != domain.TaskStatusNew {
		t.Errorf("Status = %q, want %q", task.Status, domain.TaskStatusNew)
	}
	if task.Priority != domain.TaskPriorityNormal {
		t.Errorf("Priority = %d, want %d", task.Priority, domain.TaskPriorityNormal)
	}
	if task.Kind != domain.TaskKindTask {
		t.Errorf("Kind = %q, want %q", task.Kind, domain.TaskKindTask)
	}
	if task.StartAt != nil || task.EndAt != nil || task.Deadline != nil {
		t.Error("a task without a schedule must not receive time values")
	}
	if !task.CreatedAt.Equal(now.UTC()) || !task.UpdatedAt.Equal(now.UTC()) {
		t.Error("CreatedAt and UpdatedAt must equal the creation time in UTC")
	}
}

func TestNewTaskCreatesOneHourEventByDefault(t *testing.T) {
	moscow := time.FixedZone("MSK", 3*60*60)
	start := time.Date(2026, time.August, 21, 18, 0, 0, 0, moscow)
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, moscow)

	task, err := domain.NewTask(domain.NewTaskParams{
		ID:        "event-1",
		ChatID:    -100123,
		CreatorID: 42,
		Title:     "Встреча",
		StartAt:   &start,
	}, now)
	if err != nil {
		t.Fatalf("NewTask() returned an unexpected error: %v", err)
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
}

func TestNewTaskRejectsInvalidData(t *testing.T) {
	now := time.Date(2026, time.August, 20, 9, 0, 0, 0, time.UTC)
	start := now.Add(time.Hour)
	endBeforeStart := start.Add(-time.Minute)

	tests := []struct {
		name    string
		params  domain.NewTaskParams
		now     time.Time
		wantErr error
	}{
		{
			name:    "missing task ID",
			params:  validTaskParams(),
			now:     now,
			wantErr: domain.ErrTaskIDRequired,
		},
		{
			name: "missing chat ID",
			params: domain.NewTaskParams{
				ID:        "task-1",
				CreatorID: 42,
				Title:     "Задача",
			},
			now:     now,
			wantErr: domain.ErrChatIDRequired,
		},
		{
			name: "missing creator ID",
			params: domain.NewTaskParams{
				ID:     "task-1",
				ChatID: -100123,
				Title:  "Задача",
			},
			now:     now,
			wantErr: domain.ErrCreatorRequired,
		},
		{
			name: "blank title",
			params: domain.NewTaskParams{
				ID:        "task-1",
				ChatID:    -100123,
				CreatorID: 42,
				Title:     "   ",
			},
			now:     now,
			wantErr: domain.ErrTaskTitleRequired,
		},
		{
			name:    "missing creation time",
			params:  validTaskParamsWithID(),
			wantErr: domain.ErrCreationTimeRequired,
		},
		{
			name: "end without start",
			params: domain.NewTaskParams{
				ID:        "task-1",
				ChatID:    -100123,
				CreatorID: 42,
				Title:     "Задача",
				EndAt:     &start,
			},
			now:     now,
			wantErr: domain.ErrStartTimeRequired,
		},
		{
			name: "end before start",
			params: domain.NewTaskParams{
				ID:        "task-1",
				ChatID:    -100123,
				CreatorID: 42,
				Title:     "Задача",
				StartAt:   &start,
				EndAt:     &endBeforeStart,
			},
			now:     now,
			wantErr: domain.ErrInvalidTimeRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewTask(tt.params, tt.now)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewTask() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func validTaskParams() domain.NewTaskParams {
	return domain.NewTaskParams{
		ChatID:    -100123,
		CreatorID: 42,
		Title:     "Задача",
	}
}

func validTaskParamsWithID() domain.NewTaskParams {
	params := validTaskParams()
	params.ID = "task-1"
	return params
}
