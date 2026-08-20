package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ repository.ChatMemberRepository = (*ChatMemberRepository)(nil)

// ChatMemberRepository хранит последнее известное членство пользователя в чате.
type ChatMemberRepository struct {
	pool *pgxpool.Pool
}

func NewChatMemberRepository(pool *pgxpool.Pool) *ChatMemberRepository {
	return &ChatMemberRepository{pool: pool}
}

func (repo *ChatMemberRepository) Upsert(
	ctx context.Context,
	member domain.ChatMember,
) error {
	_, err := repo.pool.Exec(ctx, `
		INSERT INTO chat_members (chat_id, user_id, status, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (chat_id, user_id) DO UPDATE SET
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
		WHERE chat_members.updated_at <= EXCLUDED.updated_at
	`, member.ChatID, member.UserID, member.Status, member.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert chat member: %w", err)
	}
	return nil
}

func (repo *ChatMemberRepository) Get(
	ctx context.Context,
	chatID domain.ChatID,
	userID domain.UserID,
) (domain.ChatMember, error) {
	var member domain.ChatMember
	err := repo.pool.QueryRow(ctx, `
		SELECT chat_id, user_id, status, updated_at
		FROM chat_members
		WHERE chat_id = $1 AND user_id = $2
	`, chatID, userID).Scan(
		&member.ChatID,
		&member.UserID,
		&member.Status,
		&member.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ChatMember{}, repository.ErrChatMemberNotFound
	}
	if err != nil {
		return domain.ChatMember{}, fmt.Errorf("select chat member: %w", err)
	}
	return member, nil
}
