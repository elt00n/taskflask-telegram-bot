package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"

	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/elt00n/taskflask-telegram-bot/internal/domain"
)

const taskCallbackPrefix = "task:"

type taskCallbackResult struct {
	text       string
	notice     string
	markup     models.ReplyMarkup
	markupOnly bool
	showAlert  bool
}

func taskActionsKeyboard(task domain.Task) *models.InlineKeyboardMarkup {
	firstRow := make([]models.InlineKeyboardButton, 0, 2)
	if task.Status != domain.TaskStatusDone {
		firstRow = append(firstRow, models.InlineKeyboardButton{
			Text:         "✅ Завершить",
			CallbackData: taskCallbackData("done", task.ID),
		})
	}
	firstRow = append(firstRow, models.InlineKeyboardButton{
		Text:         "⚡ " + priorityIcon(task.Priority),
		CallbackData: taskCallbackData("priority", task.ID),
	})

	return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
		firstRow,
		{
			{Text: "✏️ Изменить", CallbackData: taskCallbackData("edit", task.ID)},
			{Text: "🗑 Удалить", CallbackData: taskCallbackData("delete", task.ID)},
		},
	}}
}

func deleteConfirmationKeyboard(taskID domain.TaskID) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
		{Text: "Да, удалить", CallbackData: taskCallbackData("delete-confirm", taskID)},
		{Text: "Отмена", CallbackData: taskCallbackData("cancel", taskID)},
	}}}
}

func taskCallbackData(action string, taskID domain.TaskID) string {
	return taskCallbackPrefix + action + ":" + string(taskID)
}

// HandleTaskCallback обрабатывает нажатия inline-кнопок задачи.
func (handler *Handler) HandleTaskCallback(
	ctx context.Context,
	bot *telegrambot.Bot,
	update *models.Update,
) {
	query := update.CallbackQuery
	if query == nil {
		return
	}
	if query.Message.Message == nil {
		handler.answerCallback(ctx, bot, query, "Сообщение с задачей больше недоступно.", true)
		return
	}

	message := query.Message.Message
	senderMessage := *message
	senderMessage.From = &query.From
	if err := handler.observeSender(ctx, &senderMessage); err != nil {
		handler.answerCallback(ctx, bot, query, "Не удалось зарегистрировать участника.", true)
		return
	}

	result, err := handler.executeTaskCallback(ctx, query)
	if err != nil {
		log.Printf("handle task callback: %v", err)
		handler.answerCallback(ctx, bot, query, userErrorMessage(err), true)
		return
	}

	handler.answerCallback(ctx, bot, query, result.notice, result.showAlert)
	if result.markupOnly {
		_, err = bot.EditMessageReplyMarkup(ctx, &telegrambot.EditMessageReplyMarkupParams{
			ChatID:      message.Chat.ID,
			MessageID:   message.ID,
			ReplyMarkup: result.markup,
		})
	} else if result.text != "" {
		_, err = bot.EditMessageText(ctx, &telegrambot.EditMessageTextParams{
			ChatID:      message.Chat.ID,
			MessageID:   message.ID,
			Text:        result.text,
			ReplyMarkup: result.markup,
		})
	}
	if err != nil {
		log.Print("edit Telegram task message failed")
	}
}

func (handler *Handler) executeTaskCallback(
	ctx context.Context,
	query *models.CallbackQuery,
) (taskCallbackResult, error) {
	action, taskID, valid := parseTaskCallbackData(query.Data)
	if !valid || query.Message.Message == nil {
		return taskCallbackResult{}, errTaskCallbackInvalid
	}

	message := query.Message.Message
	chatID := domain.ChatID(message.Chat.ID)
	userID := domain.UserID(query.From.ID)
	resolvedID, err := handler.tasks.ResolveTaskID(ctx, userID, chatID, string(taskID))
	if err != nil {
		return taskCallbackResult{}, err
	}

	switch action {
	case "done":
		task, err := handler.tasks.Complete(ctx, userID, resolvedID)
		if err != nil {
			return taskCallbackResult{}, err
		}
		return taskCallbackResult{
			text:   formatUpdatedTask("✅ Задача завершена", task, handler.location),
			notice: "Задача завершена",
			markup: taskActionsKeyboard(task),
		}, nil
	case "priority":
		task, err := handler.tasks.GetTask(ctx, userID, chatID, resolvedID)
		if err != nil {
			return taskCallbackResult{}, err
		}
		task, err = handler.tasks.ChangePriority(ctx, userID, resolvedID, nextPriority(task.Priority))
		if err != nil {
			return taskCallbackResult{}, err
		}
		return taskCallbackResult{
			text:   formatUpdatedTask("⚡ Важность изменена", task, handler.location),
			notice: "Важность изменена",
			markup: taskActionsKeyboard(task),
		}, nil
	case "edit":
		task, err := handler.tasks.GetTask(ctx, userID, chatID, resolvedID)
		if err != nil {
			return taskCallbackResult{}, err
		}
		id := shortTaskID(task.ID)
		return taskCallbackResult{
			notice: fmt.Sprintf(
				"Название: /rename %s Новое название\nДедлайн: /deadline %s 25 августа 18:00",
				id,
				id,
			),
			showAlert: true,
		}, nil
	case "delete":
		if _, err := handler.tasks.GetTask(ctx, userID, chatID, resolvedID); err != nil {
			return taskCallbackResult{}, err
		}
		return taskCallbackResult{
			notice:     "Подтвердите удаление",
			markup:     deleteConfirmationKeyboard(resolvedID),
			markupOnly: true,
		}, nil
	case "delete-confirm":
		task, err := handler.tasks.Delete(ctx, userID, resolvedID)
		if err != nil {
			return taskCallbackResult{}, err
		}
		return taskCallbackResult{
			text:   fmt.Sprintf("🗑 Задача «%s» удалена.", task.Title),
			notice: "Задача удалена",
		}, nil
	case "cancel":
		task, err := handler.tasks.GetTask(ctx, userID, chatID, resolvedID)
		if err != nil {
			return taskCallbackResult{}, err
		}
		return taskCallbackResult{
			notice:     "Удаление отменено",
			markup:     taskActionsKeyboard(task),
			markupOnly: true,
		}, nil
	default:
		return taskCallbackResult{}, errTaskCallbackInvalid
	}
}

func parseTaskCallbackData(data string) (string, domain.TaskID, bool) {
	parts := strings.SplitN(data, ":", 3)
	if len(parts) != 3 || parts[0]+":" != taskCallbackPrefix || parts[1] == "" || parts[2] == "" {
		return "", "", false
	}
	return parts[1], domain.TaskID(parts[2]), true
}

func nextPriority(current domain.TaskPriority) domain.TaskPriority {
	if current >= domain.TaskPriorityCritical {
		return domain.TaskPriorityLow
	}
	return current + 1
}

func (handler *Handler) answerCallback(
	ctx context.Context,
	bot *telegrambot.Bot,
	query *models.CallbackQuery,
	text string,
	showAlert bool,
) {
	_, err := bot.AnswerCallbackQuery(ctx, &telegrambot.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            text,
		ShowAlert:       showAlert,
	})
	if err != nil {
		log.Print("answer Telegram callback failed")
	}
}
