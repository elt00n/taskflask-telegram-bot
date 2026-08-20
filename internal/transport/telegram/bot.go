package telegram

import (
	"context"
	"errors"
	"log"
	"time"

	telegrambot "github.com/go-telegram/bot"

	"github.com/elt00n/taskflask-telegram-bot/internal/repository"
	"github.com/elt00n/taskflask-telegram-bot/internal/service"
)

var ErrTelegramInitialization = errors.New("Telegram client initialization failed")

// Bot связывает Telegram framework с обработчиками приложения.
type Bot struct {
	client *telegrambot.Bot
}

func New(
	token string,
	tasks *service.TaskService,
	members repository.ChatMemberRepository,
	location *time.Location,
) (*Bot, error) {
	handler := NewHandler(tasks, members, NewMemberDirectory(), location, time.Now)
	client, err := telegrambot.New(
		token,
		telegrambot.WithDefaultHandler(handler.Handle),
		telegrambot.WithErrorsHandler(func(error) {
			log.Print("Telegram polling request failed")
		}),
	)
	if err != nil {
		return nil, ErrTelegramInitialization
	}

	return &Bot{client: client}, nil
}

// Start запускает long polling и работает до отмены context.
func (bot *Bot) Start(ctx context.Context) {
	bot.client.Start(ctx)
}
