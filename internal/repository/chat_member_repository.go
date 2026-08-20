package repository

import (
	"context"
	"errors"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
)

var ErrChatMemberNotFound = errors.New("chat member not found")

// ChatMemberRepository хранит известное приложению членство пользователей в чатах.
type ChatMemberRepository interface {
	Upsert(ctx context.Context, member domain.ChatMember) error
	Get(
		ctx context.Context,
		chatID domain.ChatID,
		userID domain.UserID,
	) (domain.ChatMember, error)
}
