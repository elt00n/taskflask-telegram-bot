package postgres

import (
	"context"
	"fmt"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ repository.ChatRepository = (*ChatRepository)(nil)

// ChatRepository постоянно хранит Telegram-чаты.
type ChatRepository struct {
	pool *pgxpool.Pool
}

func NewChatRepository(pool *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{pool: pool}
}

func (repo *ChatRepository) Upsert(ctx context.Context, chat domain.Chat) error {
	_, err := repo.pool.Exec(ctx, `
		INSERT INTO chats (id, title, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			type = EXCLUDED.type,
			updated_at = EXCLUDED.updated_at
		WHERE chats.updated_at <= EXCLUDED.updated_at
	`, chat.ID, chat.Title, chat.Type, chat.CreatedAt, chat.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert chat: %w", err)
	}
	return nil
}
