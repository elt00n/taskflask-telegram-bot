package repository

import (
	"context"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
)

// ChatRepository хранит известные приложению Telegram-чаты.
type ChatRepository interface {
	Upsert(ctx context.Context, chat domain.Chat) error
}
