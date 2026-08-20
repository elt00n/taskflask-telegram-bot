package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
)

var (
	ErrTaskAccessDenied     = errors.New("task access denied")
	ErrParticipantNotInChat = errors.New("task participant is not an active chat member")
)

// TaskIDGenerator создаёт публичный идентификатор новой задачи.
type TaskIDGenerator func() (domain.TaskID, error)

// Clock возвращает текущее время. Отдельный тип позволяет заменять часы в тестах.
type Clock func() time.Time

// CreateTaskCommand содержит данные пользовательского сценария создания задачи.
type CreateTaskCommand struct {
	ChatID         domain.ChatID
	CreatorID      domain.UserID
	Title          string
	Description    string
	StartAt        *time.Time
	EndAt          *time.Time
	Deadline       *time.Time
	ParticipantIDs []domain.UserID
}

// TaskService выполняет пользовательские сценарии работы с задачами.
type TaskService struct {
	tasks      repository.TaskRepository
	members    repository.ChatMemberRepository
	generateID TaskIDGenerator
	clock      Clock
}

// NewTaskService собирает сервис из независимых компонентов.
func NewTaskService(
	tasks repository.TaskRepository,
	members repository.ChatMemberRepository,
	generateID TaskIDGenerator,
	clock Clock,
) *TaskService {
	if generateID == nil {
		generateID = GenerateTaskID
	}
	if clock == nil {
		clock = time.Now
	}

	return &TaskService{
		tasks:      tasks,
		members:    members,
		generateID: generateID,
		clock:      clock,
	}
}

// Create проверяет членство, создаёт задачу и сохраняет назначения.
func (service *TaskService) Create(
	ctx context.Context,
	command CreateTaskCommand,
) (domain.Task, error) {
	if _, err := service.activeMember(ctx, command.ChatID, command.CreatorID); err != nil {
		return domain.Task{}, err
	}

	participants, err := service.prepareParticipants(ctx, command)
	if err != nil {
		return domain.Task{}, err
	}

	taskID, err := service.generateID()
	if err != nil {
		return domain.Task{}, fmt.Errorf("generate task ID: %w", err)
	}

	task, err := domain.NewTask(domain.NewTaskParams{
		ID:          taskID,
		ChatID:      command.ChatID,
		CreatorID:   command.CreatorID,
		Title:       command.Title,
		Description: command.Description,
		StartAt:     command.StartAt,
		EndAt:       command.EndAt,
		Deadline:    command.Deadline,
	}, service.clock())
	if err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
	}

	for index := range participants {
		participants[index].TaskID = task.ID
	}
	if err := service.tasks.Create(ctx, task, participants); err != nil {
		return domain.Task{}, fmt.Errorf("save task: %w", err)
	}

	return task, nil
}

// ListChatTasks возвращает все задачи чата доступному участнику.
func (service *TaskService) ListChatTasks(
	ctx context.Context,
	requesterID domain.UserID,
	chatID domain.ChatID,
) ([]domain.Task, error) {
	if _, err := service.activeMember(ctx, chatID, requesterID); err != nil {
		return nil, err
	}

	return service.tasks.List(ctx, repository.TaskFilter{ChatID: chatID})
}

// ListUserTasks возвращает созданные пользователем или назначенные ему задачи
// только внутри выбранного общего чата.
func (service *TaskService) ListUserTasks(
	ctx context.Context,
	requesterID domain.UserID,
	targetUserID domain.UserID,
	chatID domain.ChatID,
) ([]domain.Task, error) {
	if _, err := service.activeMember(ctx, chatID, requesterID); err != nil {
		return nil, err
	}
	if _, err := service.activeMember(ctx, chatID, targetUserID); err != nil {
		if errors.Is(err, ErrTaskAccessDenied) {
			return nil, ErrParticipantNotInChat
		}
		return nil, err
	}

	return service.tasks.List(ctx, repository.TaskFilter{
		ChatID: chatID,
		UserID: &targetUserID,
	})
}

func (service *TaskService) prepareParticipants(
	ctx context.Context,
	command CreateTaskCommand,
) ([]domain.TaskParticipant, error) {
	participants := []domain.TaskParticipant{
		{UserID: command.CreatorID, Role: domain.TaskParticipantRoleOwner},
	}
	seen := map[domain.UserID]struct{}{command.CreatorID: {}}

	for _, userID := range command.ParticipantIDs {
		if _, duplicate := seen[userID]; duplicate {
			continue
		}
		if _, err := service.activeMember(ctx, command.ChatID, userID); err != nil {
			if errors.Is(err, ErrTaskAccessDenied) {
				return nil, fmt.Errorf("%w: user %d", ErrParticipantNotInChat, userID)
			}
			return nil, err
		}

		participants = append(participants, domain.TaskParticipant{
			UserID: userID,
			Role:   domain.TaskParticipantRoleAssignee,
		})
		seen[userID] = struct{}{}
	}

	return participants, nil
}

func (service *TaskService) activeMember(
	ctx context.Context,
	chatID domain.ChatID,
	userID domain.UserID,
) (domain.ChatMember, error) {
	member, err := service.members.Get(ctx, chatID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrChatMemberNotFound) {
			return domain.ChatMember{}, ErrTaskAccessDenied
		}
		return domain.ChatMember{}, fmt.Errorf("get chat member: %w", err)
	}
	if !member.IsActive() {
		return domain.ChatMember{}, ErrTaskAccessDenied
	}

	return member, nil
}

// GenerateTaskID создаёт случайный UUID версии 4 средствами стандартной библиотеки.
func GenerateTaskID() (domain.TaskID, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80

	return domain.TaskID(fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		bytes[0:4],
		bytes[4:6],
		bytes[6:8],
		bytes[8:10],
		bytes[10:16],
	)), nil
}
