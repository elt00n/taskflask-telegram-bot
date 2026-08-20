package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ repository.UserRepository = (*UserRepository)(nil)

// UserRepository постоянно хранит Telegram-профили пользователей.
type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (repo *UserRepository) Upsert(ctx context.Context, user domain.User) error {
	transaction, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin upsert user: %w", err)
	}
	defer func() { _ = transaction.Rollback(context.Background()) }()

	var updateIsCurrent bool
	err = transaction.QueryRow(ctx, `
		SELECT updated_at <= $2
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, user.ID, user.UpdatedAt).Scan(&updateIsCurrent)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("check current user profile: %w", err)
	}
	if err == nil && !updateIsCurrent {
		return nil
	}

	username := normalizeUsername(user.Username)
	if username != "" {
		// Telegram username уникален. Блокировка упорядочивает редкий случай,
		// когда старый username одновременно переходит к другому пользователю.
		if _, err := transaction.Exec(
			ctx,
			"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
			username,
		); err != nil {
			return fmt.Errorf("lock username: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			UPDATE users
			SET username = '', updated_at = $3
			WHERE id <> $1 AND lower(username) = $2
		`, user.ID, username, user.UpdatedAt); err != nil {
			return fmt.Errorf("release previous username: %w", err)
		}
	}

	if _, err := transaction.Exec(ctx, `
		INSERT INTO users (
			id, username, first_name, timezone, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			updated_at = EXCLUDED.updated_at
		WHERE users.updated_at <= EXCLUDED.updated_at
	`, user.ID, username, user.FirstName, user.Timezone, user.CreatedAt, user.UpdatedAt); err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit upsert user: %w", err)
	}
	return nil
}

func (repo *UserRepository) FindByUsernameInChat(
	ctx context.Context,
	chatID domain.ChatID,
	username string,
) (domain.User, error) {
	username = normalizeUsername(username)
	if username == "" {
		return domain.User{}, repository.ErrUserNotFound
	}

	var user domain.User
	err := repo.pool.QueryRow(ctx, `
		SELECT u.id, u.username, u.first_name, u.timezone, u.created_at, u.updated_at
		FROM users u
		JOIN chat_members member ON member.user_id = u.id
		WHERE member.chat_id = $1
			AND member.status IN ('member', 'administrator', 'owner')
			AND lower(u.username) = $2
	`, chatID, username).Scan(
		&user.ID,
		&user.Username,
		&user.FirstName,
		&user.Timezone,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, repository.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("find user by username in chat: %w", err)
	}
	return user, nil
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(username), "@"))
}
