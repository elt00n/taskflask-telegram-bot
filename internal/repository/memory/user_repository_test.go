package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository/memory"
)

func TestUserRepositoryFindsActiveUserOnlyInsideChat(t *testing.T) {
	ctx := context.Background()
	members := memory.NewChatMemberRepository()
	users := memory.NewUserRepository(members)
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)

	user, err := domain.NewUser(domain.NewUserParams{
		ID:        42,
		Username:  "User42",
		FirstName: "Пользователь",
	}, now)
	if err != nil {
		t.Fatalf("NewUser() returned an unexpected error: %v", err)
	}
	if err := users.Upsert(ctx, user); err != nil {
		t.Fatalf("Upsert() returned an unexpected error: %v", err)
	}

	member, err := domain.NewChatMember(-100123, user.ID, domain.ChatMemberStatusMember, now)
	if err != nil {
		t.Fatalf("NewChatMember() returned an unexpected error: %v", err)
	}
	if err := members.Upsert(ctx, member); err != nil {
		t.Fatalf("member Upsert() returned an unexpected error: %v", err)
	}

	got, err := users.FindByUsernameInChat(ctx, -100123, "@user42")
	if err != nil || got.ID != user.ID {
		t.Fatalf("FindByUsernameInChat() = %#v, %v; want user 42", got, err)
	}
	if _, err := users.FindByUsernameInChat(ctx, -200456, "user42"); !errors.Is(err, repository.ErrUserNotFound) {
		t.Fatalf("another chat error = %v, want %v", err, repository.ErrUserNotFound)
	}
}

func TestUserRepositoryUpdatesUsername(t *testing.T) {
	ctx := context.Background()
	members := memory.NewChatMemberRepository()
	users := memory.NewUserRepository(members)
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)

	user, _ := domain.NewUser(domain.NewUserParams{
		ID: 42, Username: "old_name", FirstName: "Имя",
	}, now)
	if err := users.Upsert(ctx, user); err != nil {
		t.Fatalf("first Upsert() returned an unexpected error: %v", err)
	}
	member, _ := domain.NewChatMember(-100123, user.ID, domain.ChatMemberStatusMember, now)
	if err := members.Upsert(ctx, member); err != nil {
		t.Fatalf("member Upsert() returned an unexpected error: %v", err)
	}

	updated, _ := domain.NewUser(domain.NewUserParams{
		ID: 42, Username: "new_name", FirstName: "Новое имя",
	}, now.Add(time.Minute))
	if err := users.Upsert(ctx, updated); err != nil {
		t.Fatalf("second Upsert() returned an unexpected error: %v", err)
	}

	if _, err := users.FindByUsernameInChat(ctx, -100123, "old_name"); !errors.Is(err, repository.ErrUserNotFound) {
		t.Fatalf("old username error = %v, want %v", err, repository.ErrUserNotFound)
	}
	got, err := users.FindByUsernameInChat(ctx, -100123, "new_name")
	if err != nil || got.FirstName != "Новое имя" {
		t.Fatalf("new username result = %#v, %v", got, err)
	}
}
