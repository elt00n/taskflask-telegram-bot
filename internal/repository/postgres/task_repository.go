// Package postgres содержит реализации репозиториев для PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ repository.TaskRepository = (*TaskRepository)(nil)

const taskColumns = `
	id::text, chat_id, creator_id, title, description, kind, status, priority,
	start_at, end_at, deadline, created_at, updated_at, deleted_at
`

// TaskRepository сохраняет задачи и назначения в PostgreSQL.
type TaskRepository struct {
	pool *pgxpool.Pool
}

func (repo *TaskRepository) ResolveID(
	ctx context.Context,
	chatID domain.ChatID,
	reference string,
) (domain.TaskID, error) {
	reference = strings.ToLower(strings.TrimSpace(reference))
	if reference == "" {
		return "", repository.ErrTaskNotFound
	}
	rows, err := repo.pool.Query(ctx, `
		SELECT id::text
		FROM tasks
		WHERE chat_id = $1
			AND deleted_at IS NULL
			AND id::text LIKE $2 || '%'
		ORDER BY id
		LIMIT 2
	`, chatID, reference)
	if err != nil {
		return "", fmt.Errorf("resolve task ID: %w", err)
	}
	defer rows.Close()

	ids := make([]domain.TaskID, 0, 2)
	for rows.Next() {
		var taskID domain.TaskID
		if err := rows.Scan(&taskID); err != nil {
			return "", fmt.Errorf("scan resolved task ID: %w", err)
		}
		ids = append(ids, taskID)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate resolved task IDs: %w", err)
	}
	if len(ids) == 0 {
		return "", repository.ErrTaskNotFound
	}
	if len(ids) > 1 {
		return "", repository.ErrTaskIDAmbiguous
	}
	return ids[0], nil
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

func (repo *TaskRepository) Create(
	ctx context.Context,
	task domain.Task,
	participants []domain.TaskParticipant,
) error {
	transaction, err := repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create task: %w", err)
	}
	defer func() { _ = transaction.Rollback(context.Background()) }()

	_, err = transaction.Exec(ctx, `
		INSERT INTO tasks (
			id, chat_id, creator_id, title, description, kind, status, priority,
			start_at, end_at, deadline, created_at, updated_at, deleted_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14
		)
	`, task.ID, task.ChatID, task.CreatorID, task.Title, task.Description,
		task.Kind, task.Status, task.Priority, task.StartAt, task.EndAt,
		task.Deadline, task.CreatedAt, task.UpdatedAt, task.DeletedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return repository.ErrTaskAlreadyExists
		}
		return fmt.Errorf("insert task: %w", err)
	}

	for _, participant := range participants {
		_, err := transaction.Exec(ctx, `
			INSERT INTO task_participants (task_id, user_id, role)
			VALUES ($1, $2, $3)
		`, task.ID, participant.UserID, participant.Role)
		if err != nil {
			return fmt.Errorf("insert task participant: %w", err)
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit create task: %w", err)
	}
	return nil
}

func (repo *TaskRepository) Get(
	ctx context.Context,
	taskID domain.TaskID,
) (domain.Task, []domain.TaskParticipant, error) {
	row := repo.pool.QueryRow(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = $1`, taskID)
	task, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Task{}, nil, repository.ErrTaskNotFound
	}
	if err != nil {
		return domain.Task{}, nil, fmt.Errorf("select task: %w", err)
	}

	rows, err := repo.pool.Query(ctx, `
		SELECT task_id::text, user_id, role
		FROM task_participants
		WHERE task_id = $1
		ORDER BY user_id
	`, taskID)
	if err != nil {
		return domain.Task{}, nil, fmt.Errorf("select task participants: %w", err)
	}
	defer rows.Close()

	participants := make([]domain.TaskParticipant, 0)
	for rows.Next() {
		var participant domain.TaskParticipant
		if err := rows.Scan(&participant.TaskID, &participant.UserID, &participant.Role); err != nil {
			return domain.Task{}, nil, fmt.Errorf("scan task participant: %w", err)
		}
		participants = append(participants, participant)
	}
	if err := rows.Err(); err != nil {
		return domain.Task{}, nil, fmt.Errorf("iterate task participants: %w", err)
	}

	return task, participants, nil
}

func (repo *TaskRepository) Update(ctx context.Context, task domain.Task) error {
	result, err := repo.pool.Exec(ctx, `
		UPDATE tasks SET
			chat_id = $2,
			creator_id = $3,
			title = $4,
			description = $5,
			kind = $6,
			status = $7,
			priority = $8,
			start_at = $9,
			end_at = $10,
			deadline = $11,
			created_at = $12,
			updated_at = $13,
			deleted_at = $14
		WHERE id = $1
	`, task.ID, task.ChatID, task.CreatorID, task.Title, task.Description,
		task.Kind, task.Status, task.Priority, task.StartAt, task.EndAt,
		task.Deadline, task.CreatedAt, task.UpdatedAt, task.DeletedAt)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return repository.ErrTaskNotFound
	}
	return nil
}

func (repo *TaskRepository) List(
	ctx context.Context,
	filter repository.TaskFilter,
) ([]domain.Task, error) {
	query := `SELECT ` + taskColumns + `
		FROM tasks
		WHERE chat_id = $1 AND deleted_at IS NULL`
	arguments := []any{filter.ChatID}

	if filter.UserID != nil {
		query += ` AND (
			creator_id = $2 OR EXISTS (
				SELECT 1 FROM task_participants
				WHERE task_participants.task_id = tasks.id
				AND task_participants.user_id = $2
			)
		)`
		arguments = append(arguments, *filter.UserID)
	}
	query += ` ORDER BY created_at, id`

	rows, err := repo.pool.Query(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan listed task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

type taskScanner interface {
	Scan(destinations ...any) error
}

func scanTask(scanner taskScanner) (domain.Task, error) {
	var task domain.Task
	err := scanner.Scan(
		&task.ID,
		&task.ChatID,
		&task.CreatorID,
		&task.Title,
		&task.Description,
		&task.Kind,
		&task.Status,
		&task.Priority,
		&task.StartAt,
		&task.EndAt,
		&task.Deadline,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.DeletedAt,
	)
	return task, err
}

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}
