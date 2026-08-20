// Package memory содержит временные реализации репозиториев в оперативной памяти.
package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
)

var _ repository.TaskRepository = (*TaskRepository)(nil)

type taskRecord struct {
	task         domain.Task
	participants []domain.TaskParticipant
}

// TaskRepository хранит задачи до завершения процесса.
// mutex защищает map от одновременного чтения и записи разными goroutine.
type TaskRepository struct {
	mutex sync.RWMutex
	tasks map[domain.TaskID]taskRecord
}

// NewTaskRepository возвращает пустое хранилище, готовое к использованию.
func NewTaskRepository() *TaskRepository {
	return &TaskRepository{
		tasks: make(map[domain.TaskID]taskRecord),
	}
}

// Create сохраняет новую задачу и её назначенных участников.
func (repo *TaskRepository) Create(
	ctx context.Context,
	task domain.Task,
	participants []domain.TaskParticipant,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	if _, exists := repo.tasks[task.ID]; exists {
		return repository.ErrTaskAlreadyExists
	}

	repo.tasks[task.ID] = taskRecord{
		task:         task,
		participants: cloneParticipants(participants),
	}

	return nil
}

// Get возвращает задачу и независимую копию списка участников.
func (repo *TaskRepository) Get(
	ctx context.Context,
	taskID domain.TaskID,
) (domain.Task, []domain.TaskParticipant, error) {
	if err := ctx.Err(); err != nil {
		return domain.Task{}, nil, err
	}

	repo.mutex.RLock()
	defer repo.mutex.RUnlock()

	record, exists := repo.tasks[taskID]
	if !exists {
		return domain.Task{}, nil, repository.ErrTaskNotFound
	}

	return record.task, cloneParticipants(record.participants), nil
}

// Update заменяет данные существующей задачи, сохраняя её участников.
func (repo *TaskRepository) Update(ctx context.Context, task domain.Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	record, exists := repo.tasks[task.ID]
	if !exists {
		return repository.ErrTaskNotFound
	}

	record.task = task
	repo.tasks[task.ID] = record
	return nil
}

// List возвращает задачи чата, при необходимости связанные с выбранным человеком.
// Связь с человеком означает: он создатель или назначенный участник задачи.
func (repo *TaskRepository) List(
	ctx context.Context,
	filter repository.TaskFilter,
) ([]domain.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo.mutex.RLock()
	defer repo.mutex.RUnlock()

	tasks := make([]domain.Task, 0)
	for _, record := range repo.tasks {
		if record.task.ChatID != filter.ChatID {
			continue
		}
		if record.task.IsDeleted() {
			continue
		}
		if filter.UserID != nil && !recordBelongsToUser(record, *filter.UserID) {
			continue
		}

		tasks = append(tasks, record.task)
	}

	sort.Slice(tasks, func(i int, j int) bool {
		if tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})

	return tasks, nil
}

func recordBelongsToUser(record taskRecord, userID domain.UserID) bool {
	if record.task.CreatorID == userID {
		return true
	}

	for _, participant := range record.participants {
		if participant.UserID == userID {
			return true
		}
	}

	return false
}

func cloneParticipants(participants []domain.TaskParticipant) []domain.TaskParticipant {
	return append([]domain.TaskParticipant(nil), participants...)
}
