package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
	"github.com/elt00n/taskflask-telegram-bot/internal/parser"
	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
	"github.com/elt00n/taskflask-telegram-bot/internal/service"
)

const helpText = `Taskflask — менеджер задач для Telegram.

Команды:
/task Купить молоко
/task завтра 10:00 Позвонить врачу
/task до пятницы 18:00 Закончить макет важно
/tasks — все задачи текущего чата
/tasks @username — задачи выбранного участника
/help — эта справка

Важность: добавьте слово «важно» или «срочно».`

// Handler обрабатывает Telegram updates и вызывает бизнес-сервисы.
type Handler struct {
	tasks    *service.TaskService
	members  repository.ChatMemberRepository
	users    repository.UserRepository
	chats    repository.ChatRepository
	location *time.Location
	clock    service.Clock
}

func NewHandler(
	tasks *service.TaskService,
	members repository.ChatMemberRepository,
	users repository.UserRepository,
	chats repository.ChatRepository,
	location *time.Location,
	clock service.Clock,
) *Handler {
	return &Handler{
		tasks:    tasks,
		members:  members,
		users:    users,
		chats:    chats,
		location: location,
		clock:    clock,
	}
}

// Handle является общей точкой входа для текстовых Telegram updates.
func (handler *Handler) Handle(
	ctx context.Context,
	bot *telegrambot.Bot,
	update *models.Update,
) {
	if update.Message == nil || update.Message.From == nil || update.Message.Text == "" {
		return
	}

	message := update.Message
	if err := handler.observeSender(ctx, message); err != nil {
		handler.send(ctx, bot, message, "Не удалось зарегистрировать участника. Попробуйте ещё раз.")
		log.Printf("observe Telegram sender: %v", err)
		return
	}

	command := commandName(message.Text)
	var response string
	var err error

	switch command {
	case "/start":
		response = "Привет! Я помогу вести задачи этого чата.\n\n" + helpText
	case "/help":
		response = helpText
	case "/task":
		response, err = handler.createTask(ctx, message)
	case "/tasks":
		response, err = handler.listTasks(ctx, message)
	default:
		return
	}

	if err != nil {
		log.Printf("handle %s: %v", command, err)
		response = userErrorMessage(err)
	}
	handler.send(ctx, bot, message, response)
}

func (handler *Handler) createTask(
	ctx context.Context,
	message *models.Message,
) (string, error) {
	parsed, err := parser.ParseTask(message.Text, handler.clock(), handler.location)
	if err != nil {
		return "", err
	}

	chatID := domain.ChatID(message.Chat.ID)
	participantIDs, unknown, err := handler.resolveUsernames(
		ctx,
		chatID,
		parsed.ParticipantUsernames,
	)
	if err != nil {
		return "", err
	}
	if len(unknown) > 0 {
		return "", fmt.Errorf("%w: @%s", errUnknownParticipants, strings.Join(unknown, ", @"))
	}

	task, err := handler.tasks.Create(ctx, service.CreateTaskCommand{
		ChatID:         chatID,
		CreatorID:      domain.UserID(message.From.ID),
		Title:          parsed.Title,
		Priority:       parsed.Priority,
		StartAt:        parsed.StartAt,
		Deadline:       parsed.Deadline,
		ParticipantIDs: participantIDs,
	})
	if err != nil {
		return "", err
	}

	return formatCreatedTask(task, handler.location), nil
}

func (handler *Handler) listTasks(
	ctx context.Context,
	message *models.Message,
) (string, error) {
	chatID := domain.ChatID(message.Chat.ID)
	requesterID := domain.UserID(message.From.ID)
	arguments := commandArguments(message.Text)

	var tasks []domain.Task
	var err error
	if arguments == "" {
		tasks, err = handler.tasks.ListChatTasks(ctx, requesterID, chatID)
	} else {
		fields := strings.Fields(arguments)
		if len(fields) != 1 || !strings.HasPrefix(fields[0], "@") {
			return "", errTasksUsage
		}
		username := normalizeUsername(fields[0])
		userIDs, unknown, resolveErr := handler.resolveUsernames(
			ctx,
			chatID,
			[]string{username},
		)
		if resolveErr != nil {
			return "", resolveErr
		}
		if len(unknown) > 0 {
			return "", fmt.Errorf("%w: @%s", errUnknownParticipants, username)
		}
		tasks, err = handler.tasks.ListUserTasks(ctx, requesterID, userIDs[0], chatID)
	}
	if err != nil {
		return "", err
	}

	return formatTaskList(tasks, handler.location), nil
}

func (handler *Handler) observeSender(ctx context.Context, message *models.Message) error {
	now := handler.clock()
	user, err := domain.NewUser(domain.NewUserParams{
		ID:        domain.UserID(message.From.ID),
		Username:  message.From.Username,
		FirstName: message.From.FirstName,
	}, now)
	if err != nil {
		return fmt.Errorf("create user profile: %w", err)
	}

	chatTitle := message.Chat.Title
	if message.Chat.Type == models.ChatTypePrivate {
		chatTitle = strings.TrimSpace(message.Chat.FirstName + " " + message.Chat.LastName)
	}
	chat, err := domain.NewChat(
		domain.ChatID(message.Chat.ID),
		chatTitle,
		domain.ChatType(message.Chat.Type),
		now,
	)
	if err != nil {
		return fmt.Errorf("create chat: %w", err)
	}

	member, err := domain.NewChatMember(
		chat.ID,
		user.ID,
		domain.ChatMemberStatusMember,
		now,
	)
	if err != nil {
		return fmt.Errorf("create chat member: %w", err)
	}

	if err := handler.users.Upsert(ctx, user); err != nil {
		return err
	}
	if err := handler.chats.Upsert(ctx, chat); err != nil {
		return err
	}
	if err := handler.members.Upsert(ctx, member); err != nil {
		return err
	}
	return nil
}

func (handler *Handler) resolveUsernames(
	ctx context.Context,
	chatID domain.ChatID,
	usernames []string,
) ([]domain.UserID, []string, error) {
	userIDs := make([]domain.UserID, 0, len(usernames))
	unknown := make([]string, 0)
	for _, username := range usernames {
		username = normalizeUsername(username)
		user, err := handler.users.FindByUsernameInChat(ctx, chatID, username)
		if errors.Is(err, repository.ErrUserNotFound) {
			unknown = append(unknown, username)
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		userIDs = append(userIDs, user.ID)
	}
	return userIDs, unknown, nil
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(username), "@"))
}

func (handler *Handler) send(
	ctx context.Context,
	bot *telegrambot.Bot,
	message *models.Message,
	text string,
) {
	_, err := bot.SendMessage(ctx, &telegrambot.SendMessageParams{
		ChatID:          message.Chat.ID,
		MessageThreadID: message.MessageThreadID,
		Text:            text,
	})
	if err != nil {
		log.Print("send Telegram message failed")
	}
}

func commandName(text string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(text), " ")
	name, _, _ := strings.Cut(first, "@")
	return strings.ToLower(name)
}

func commandArguments(text string) string {
	_, arguments, found := strings.Cut(strings.TrimSpace(text), " ")
	if !found {
		return ""
	}
	return strings.TrimSpace(arguments)
}

var (
	errUnknownParticipants = errors.New("unknown task participants")
	errTasksUsage          = errors.New("invalid tasks command")
)

func userErrorMessage(err error) string {
	switch {
	case errors.Is(err, parser.ErrTaskTitleRequired):
		return "Укажите название после /task. Например: /task Купить молоко"
	case errors.Is(err, parser.ErrInvalidTaskTime), errors.Is(err, parser.ErrInvalidTaskDate):
		return "Не удалось разобрать дату или время. Используйте, например: /task завтра 18:00 Встреча"
	case errors.Is(err, parser.ErrTaskTimeInPast), errors.Is(err, domain.ErrDeadlineInPast):
		return "Указанное время уже прошло. Выберите будущее время."
	case errors.Is(err, errUnknownParticipants):
		return "Я ещё не знаю одного из указанных участников. Попросите его сначала выполнить /start или /help в этом чате."
	case errors.Is(err, errTasksUsage):
		return "Используйте /tasks или /tasks @username."
	case errors.Is(err, service.ErrTaskAccessDenied):
		return "У вас нет доступа к этой операции."
	default:
		return "Не удалось выполнить команду. Попробуйте ещё раз."
	}
}
