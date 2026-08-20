package repository

import (
	"context"
	"errors"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
)

var ErrUserNotFound = errors.New("user not found")

// UserRepository хранит профили и ищет активного участника по username в чате.
type UserRepository interface {
	Upsert(ctx context.Context, user domain.User) error
	FindByUsernameInChat(
		ctx context.Context,
		chatID domain.ChatID,
		username string,
	) (domain.User, error)
}
