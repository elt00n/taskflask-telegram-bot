package database

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/elt00n/taskflask-telegram-bot/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationLockID int64 = 7_524_011_925

// Migrate последовательно применяет ещё не выполненные .up.sql-файлы.
// Advisory lock не позволяет двум запущенным экземплярам менять схему одновременно.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	connection, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer connection.Release()

	if _, err := connection.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	defer func() {
		_, _ = connection.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", migrationLockID)
	}()

	if _, err := connection.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, filename := range files {
		if err := applyMigration(ctx, connection, filename); err != nil {
			return err
		}
	}

	return nil
}

func applyMigration(ctx context.Context, connection *pgxpool.Conn, filename string) error {
	var applied bool
	if err := connection.QueryRow(
		ctx,
		"SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)",
		filename,
	).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %s: %w", filename, err)
	}
	if applied {
		return nil
	}

	contents, err := migrations.Files.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", filename, err)
	}

	transaction, err := connection.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", filename, err)
	}
	defer func() { _ = transaction.Rollback(context.Background()) }()

	if _, err := transaction.Exec(ctx, string(contents)); err != nil {
		return fmt.Errorf("execute migration %s: %w", filename, err)
	}
	if _, err := transaction.Exec(
		ctx,
		"INSERT INTO schema_migrations (version) VALUES ($1)",
		filename,
	); err != nil {
		return fmt.Errorf("record migration %s: %w", filename, err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", filename, err)
	}

	return nil
}
