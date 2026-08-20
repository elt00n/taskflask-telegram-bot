package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
)

func TestNewUserAppliesDefaultsAndNormalizesProfile(t *testing.T) {
	now := time.Date(2026, time.August, 20, 15, 0, 0, 0, time.FixedZone("MSK", 3*60*60))

	user, err := domain.NewUser(domain.NewUserParams{
		ID:        42,
		Username:  "  @user1  ",
		FirstName: "  Семён  ",
	}, now)
	if err != nil {
		t.Fatalf("NewUser() returned an unexpected error: %v", err)
	}

	if user.Username != "user1" {
		t.Errorf("Username = %q, want %q", user.Username, "user1")
	}
	if user.FirstName != "Семён" {
		t.Errorf("FirstName = %q, want %q", user.FirstName, "Семён")
	}
	if user.Timezone != domain.DefaultUserTimezone {
		t.Errorf("Timezone = %q, want %q", user.Timezone, domain.DefaultUserTimezone)
	}
	if !user.CreatedAt.Equal(now.UTC()) || !user.UpdatedAt.Equal(now.UTC()) {
		t.Error("CreatedAt and UpdatedAt must equal the creation time in UTC")
	}
}

func TestNewUserRejectsInvalidTimezone(t *testing.T) {
	_, err := domain.NewUser(domain.NewUserParams{
		ID:        42,
		FirstName: "Семён",
		Timezone:  "Mars/Olympus",
	}, time.Now())

	if !errors.Is(err, domain.ErrInvalidUserTimezone) {
		t.Fatalf("NewUser() error = %v, want %v", err, domain.ErrInvalidUserTimezone)
	}
}

func TestNewChatAcceptsNegativeTelegramGroupID(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)

	chat, err := domain.NewChat(-100123, "  Рабочая группа  ", domain.ChatTypeSupergroup, now)
	if err != nil {
		t.Fatalf("NewChat() returned an unexpected error: %v", err)
	}

	if chat.ID != -100123 {
		t.Errorf("ID = %d, want %d", chat.ID, -100123)
	}
	if chat.Title != "Рабочая группа" {
		t.Errorf("Title = %q, want %q", chat.Title, "Рабочая группа")
	}
}

func TestChatMemberIsActive(t *testing.T) {
	tests := []struct {
		status domain.ChatMemberStatus
		want   bool
	}{
		{status: domain.ChatMemberStatusMember, want: true},
		{status: domain.ChatMemberStatusAdministrator, want: true},
		{status: domain.ChatMemberStatusOwner, want: true},
		{status: domain.ChatMemberStatusLeft, want: false},
		{status: domain.ChatMemberStatusBanned, want: false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			member := domain.ChatMember{Status: tt.status}
			if got := member.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %t, want %t", got, tt.want)
			}
		})
	}
}
