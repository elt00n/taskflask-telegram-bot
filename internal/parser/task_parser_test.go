package parser_test

import (
	"errors"
	"testing"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/parser"
)

func TestParseTaskSupportedCommands(t *testing.T) {
	location := mustLocation(t, "Europe/Moscow")
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, location)

	tests := []struct {
		name          string
		input         string
		wantTitle     string
		wantStart     *time.Time
		wantDeadline  *time.Time
		wantPriority  domain.TaskPriority
		wantUsernames []string
	}{
		{
			name:         "plain task",
			input:        "/task Купить молоко",
			wantTitle:    "Купить молоко",
			wantPriority: domain.TaskPriorityNormal,
		},
		{
			name:         "today at a specific time",
			input:        "/task сегодня 18:00 Позвонить врачу",
			wantTitle:    "Позвонить врачу",
			wantStart:    timePointer(time.Date(2026, time.August, 20, 18, 0, 0, 0, location)),
			wantPriority: domain.TaskPriorityNormal,
		},
		{
			name:         "tomorrow at a specific time",
			input:        "/task завтра в 10:00 Позвонить врачу",
			wantTitle:    "Позвонить врачу",
			wantStart:    timePointer(time.Date(2026, time.August, 21, 10, 0, 0, 0, location)),
			wantPriority: domain.TaskPriorityNormal,
		},
		{
			name:          "weekday event with participants",
			input:         "/task в пятницу 19:00 Ужин @Anna @bob @ANNA",
			wantTitle:     "Ужин",
			wantStart:     timePointer(time.Date(2026, time.August, 21, 19, 0, 0, 0, location)),
			wantPriority:  domain.TaskPriorityNormal,
			wantUsernames: []string{"anna", "bob"},
		},
		{
			name:         "calendar date deadline and high priority",
			input:        "/task до 25 августа Подготовить презентацию важно",
			wantTitle:    "Подготовить презентацию",
			wantDeadline: timePointer(time.Date(2026, time.August, 25, 23, 59, 0, 0, location)),
			wantPriority: domain.TaskPriorityHigh,
		},
		{
			name:         "weekday deadline with time and critical priority",
			input:        "/task до пятницы 18:00 Закончить макет срочно",
			wantTitle:    "Закончить макет",
			wantDeadline: timePointer(time.Date(2026, time.August, 21, 18, 0, 0, 0, location)),
			wantPriority: domain.TaskPriorityCritical,
		},
		{
			name:         "command addressed to a bot",
			input:        "/task@taskflask_bot Завершить документацию",
			wantTitle:    "Завершить документацию",
			wantPriority: domain.TaskPriorityNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.ParseTask(tt.input, now, location)
			if err != nil {
				t.Fatalf("ParseTask() returned an unexpected error: %v", err)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
			if got.Priority != tt.wantPriority {
				t.Errorf("Priority = %d, want %d", got.Priority, tt.wantPriority)
			}
			assertOptionalTime(t, "StartAt", got.StartAt, tt.wantStart)
			assertOptionalTime(t, "Deadline", got.Deadline, tt.wantDeadline)
			assertStrings(t, got.ParticipantUsernames, tt.wantUsernames)
		})
	}
}

func TestParseTaskMovesDateWithoutYearToNextYear(t *testing.T) {
	location := mustLocation(t, "Europe/Moscow")
	now := time.Date(2026, time.December, 30, 12, 0, 0, 0, location)

	got, err := parser.ParseTask("/task до 5 января Купить подарки", now, location)
	if err != nil {
		t.Fatalf("ParseTask() returned an unexpected error: %v", err)
	}
	want := time.Date(2027, time.January, 5, 23, 59, 0, 0, location)
	assertOptionalTime(t, "Deadline", got.Deadline, &want)
}

func TestParseTaskRejectsInvalidCommands(t *testing.T) {
	location := mustLocation(t, "Europe/Moscow")
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, location)

	tests := []struct {
		name    string
		input   string
		now     time.Time
		wantErr error
	}{
		{name: "missing command", input: "Купить молоко", now: now, wantErr: parser.ErrTaskCommandRequired},
		{name: "empty title", input: "/task", now: now, wantErr: parser.ErrTaskTitleRequired},
		{name: "past time today", input: "/task сегодня 10:00 Опоздавшая встреча", now: now, wantErr: parser.ErrTaskTimeInPast},
		{name: "invalid clock", input: "/task завтра 25:00 Встреча", now: now, wantErr: parser.ErrInvalidTaskTime},
		{name: "invalid date", input: "/task до 30 февраля Отчёт", now: now, wantErr: parser.ErrInvalidTaskDate},
		{name: "explicit past year", input: "/task до 1 января 2025 Старый отчёт", now: now, wantErr: parser.ErrTaskTimeInPast},
		{name: "missing reference time", input: "/task Задача", wantErr: parser.ErrReferenceTimeNeeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseTask(tt.input, tt.now, location)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ParseTask() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func mustLocation(t *testing.T, name string) *time.Location {
	t.Helper()

	location, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("LoadLocation() returned an unexpected error: %v", err)
	}
	return location
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func assertOptionalTime(t *testing.T, name string, got *time.Time, want *time.Time) {
	t.Helper()

	if got == nil && want == nil {
		return
	}
	if got == nil || want == nil || !got.Equal(*want) {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func assertStrings(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("usernames = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("usernames[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}
