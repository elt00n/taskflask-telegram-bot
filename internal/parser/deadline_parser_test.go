package parser_test

import (
	"errors"
	"testing"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/parser"
)

func TestParseDeadline(t *testing.T) {
	location := mustLocation(t, "Europe/Moscow")
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, location)

	tests := []struct {
		name    string
		value   string
		want    time.Time
		wantErr error
	}{
		{
			name:  "calendar date",
			value: "25 августа 2026 18:00",
			want:  time.Date(2026, time.August, 25, 18, 0, 0, 0, location),
		},
		{
			name:  "weekday with optional prefix",
			value: "до пятницы 18:00",
			want:  time.Date(2026, time.August, 21, 18, 0, 0, 0, location),
		},
		{
			name:    "invalid date",
			value:   "31 апреля 2027",
			wantErr: parser.ErrInvalidTaskDate,
		},
		{
			name:    "past date",
			value:   "1 января 2025",
			wantErr: parser.ErrTaskTimeInPast,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.ParseDeadline(tt.value, now, location)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseDeadline() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && (got == nil || !got.Equal(tt.want)) {
				t.Errorf("ParseDeadline() = %v, want %v", got, tt.want)
			}
		})
	}
}
